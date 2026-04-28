package main // 声明当前文件属于 main 包，方便直接用 go run 执行。

import ( // 导入本示例只需要的 Go 标准库。
	"context"           // 导入 context，用于传递取消信号和超时控制。
	"errors"            // 导入 errors，用于创建错误并检查错误链。
	"fmt"               // 导入 fmt，用于格式化错误和打印演示信息。
	"io"                // 导入 io，用于读取 HTTP 响应体。
	"math/rand"         // 导入 math/rand，用于给退避时间增加随机抖动。
	"net/http"          // 导入 net/http，用于实现 HTTP 客户端请求。
	"net/http/httptest" // 导入 httptest，用于启动本地测试 HTTP 服务。
	"time"              // 导入 time，用于表达延迟、超时和计时。
) // 结束 import 声明。

// RetryableError 表示一个可以安全重试的错误。
type RetryableError struct { // 定义可重试错误结构体，用来给普通错误打上“可重试”标记。
	Cause error // 保存原始错误原因，方便调用方通过 errors.As 或 errors.Unwrap 继续分析。
} // 结束 RetryableError 结构体定义。

func (e *RetryableError) Error() string { // 实现 error 接口，返回带有可重试标记的错误文本。
	return fmt.Sprintf("retryable: %v", e.Cause) // 把底层错误包装进更清晰的错误消息中。
} // 结束 RetryableError 的 Error 方法。

func (e *RetryableError) Unwrap() error { // 实现 Unwrap，支持 Go 1.13+ 的错误链能力。
	return e.Cause // 返回底层错误，让 errors.Is 和 errors.As 能继续向下检查。
} // 结束 RetryableError 的 Unwrap 方法。

func IsRetryable(err error) bool { // 判断一个错误是否属于可重试错误。
	var retryable *RetryableError     // 声明目标类型变量，供 errors.As 在错误链中查找。
	return errors.As(err, &retryable) // 如果错误链中存在 *RetryableError，就认为该错误可重试。
} // 结束 IsRetryable 函数。

// RetryConfig 保存重试行为的所有可配置参数。
type RetryConfig struct { // 定义重试配置结构体，避免把重试策略硬编码在函数里。
	MaxRetries   int              // MaxRetries 表示失败后最多额外重试多少次。
	BaseDelay    time.Duration    // BaseDelay 表示指数退避的初始等待时间。
	MaxDelay     time.Duration    // MaxDelay 表示单次退避等待时间的上限。
	MaxTotalTime time.Duration    // MaxTotalTime 表示整个重试流程允许消耗的最长时间。
	ShouldRetry  func(error) bool // ShouldRetry 根据错误决定是否继续重试。
} // 结束 RetryConfig 结构体定义。

func DefaultRetryConfig() RetryConfig { // 返回适合本地 demo 的默认重试配置。
	return RetryConfig{ // 构造并返回 RetryConfig 字面量。
		MaxRetries:   3,                      // 最多重试 3 次，所以总尝试次数最多是 4 次。
		BaseDelay:    50 * time.Millisecond,  // 初始退避 50ms，方便 demo 快速完成。
		MaxDelay:     300 * time.Millisecond, // 单次等待最多 300ms，避免示例运行太久。
		MaxTotalTime: 2 * time.Second,        // 整体最多运行 2s，防止无限等待。
		ShouldRetry: func(err error) bool { // 定义默认的可重试判断逻辑。
			if IsRetryable(err) { // 如果错误链包含 RetryableError，就允许重试。
				return true // 返回 true 表示当前错误可以重试。
			} // 结束 RetryableError 判断。
			if errors.Is(err, context.DeadlineExceeded) { // 如果错误是 context 超时，也认为可重试。
				return true // 返回 true 表示超时错误可以重试。
			} // 结束 context 超时判断。
			return false // 其他错误默认不可重试，例如 4xx 客户端错误。
		}, // 结束 ShouldRetry 函数字段赋值。
	} // 结束 RetryConfig 字面量。
} // 结束 DefaultRetryConfig 函数。

func DoWithRetry(ctx context.Context, cfg RetryConfig, operation func(context.Context) error) error { // 按配置执行带重试的操作，并把总超时 context 传给操作。
	if cfg.ShouldRetry == nil { // 如果调用方没有提供 ShouldRetry 函数，就避免空指针调用。
		cfg.ShouldRetry = IsRetryable // 使用只识别 RetryableError 的默认判断逻辑。
	} // 结束 ShouldRetry 默认值处理。
	if cfg.MaxRetries < 0 { // 如果最大重试次数被误设为负数，就修正为不重试。
		cfg.MaxRetries = 0 // 把负数重试次数归零，保证循环边界合理。
	} // 结束 MaxRetries 修正。
	if cfg.BaseDelay <= 0 { // 如果基础延迟没有设置，就给一个安全默认值。
		cfg.BaseDelay = 50 * time.Millisecond // 使用 50ms 作为默认初始退避时间。
	} // 结束 BaseDelay 修正。
	if cfg.MaxDelay <= 0 { // 如果最大延迟没有设置，就给一个安全默认值。
		cfg.MaxDelay = cfg.BaseDelay // 让最大延迟至少等于基础延迟。
	} // 结束 MaxDelay 修正。
	totalCtx := ctx           // 默认使用外部传入的 context 作为总控制 context。
	cancel := func() {}       // 准备一个空 cancel，方便后面统一 defer 调用。
	if cfg.MaxTotalTime > 0 { // 如果配置了总超时时间，就创建派生 context。
		totalCtx, cancel = context.WithTimeout(ctx, cfg.MaxTotalTime) // 用 context 限制整个重试流程的最长时间。
	} // 结束总超时 context 创建。
	defer cancel()                                           // 函数退出时释放 context 相关资源。
	var lastErr error                                        // 记录最后一次 operation 返回的错误。
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ { // 从第 0 次尝试开始，最多执行 MaxRetries 次重试。
		if err := totalCtx.Err(); err != nil { // 每次尝试前先检查 context 是否已经取消或超时。
			return err // 如果 context 已结束，就立即返回对应错误。
		} // 结束尝试前 context 检查。
		lastErr = operation(totalCtx) // 使用总控制 context 执行业务操作，并记录本次错误。
		if lastErr == nil {           // 如果本次操作成功，就不再重试。
			return nil // 返回 nil 表示最终成功。
		} // 结束成功判断。
		if !cfg.ShouldRetry(lastErr) { // 如果错误不可重试，就立即返回给调用方。
			return lastErr // 返回原始错误，避免隐藏不可重试原因。
		} // 结束不可重试判断。
		if attempt == cfg.MaxRetries { // 如果已经到达最后一次允许尝试，就不能继续等待重试。
			return lastErr // 返回最后一次失败的错误。
		} // 结束重试次数用尽判断。
		delay := cfg.BaseDelay * time.Duration(1<<attempt) // 按 attempt 计算指数退避时间。
		if delay > cfg.MaxDelay {                          // 如果指数退避超过最大延迟，就截断到上限。
			delay = cfg.MaxDelay // 使用配置的最大延迟作为本次基础等待时间。
		} // 结束最大延迟截断。
		jitterLimit := int64(delay / 2)            // 用基础等待时间的一半作为 jitter 上限。
		if jitterLimit < int64(time.Millisecond) { // 如果 jitter 上限太小，就保证至少有 1ms。
			jitterLimit = int64(time.Millisecond) // 设置最小 jitter 上限为 1ms。
		} // 结束 jitter 上限修正。
		jitter := time.Duration(rand.Int63n(jitterLimit))                                  // 生成 0 到 jitterLimit 之间的随机抖动。
		wait := delay + jitter                                                             // 把基础退避和 jitter 相加，得到最终等待时间。
		fmt.Printf("attempt %d failed: %v; retrying after %v\n", attempt+1, lastErr, wait) // 打印本次失败原因和下一次重试等待时间。
		select {                                                                           // 等待重试时间，期间同时监听 context 取消或超时。
		case <-totalCtx.Done(): // 如果 context 在等待期间结束，就优先响应取消。
			return totalCtx.Err() // 返回 context 的取消或超时错误。
		case <-time.After(wait): // 如果等待时间正常结束，就进入下一轮尝试。
		} // 结束 select 等待。
	} // 结束重试循环。
	return lastErr // 理论上不会走到这里，保留返回最后错误作为兜底。
} // 结束 DoWithRetry 函数。

func HTTPGet(ctx context.Context, url string) ([]byte, error) { // 执行带重试语义的 HTTP GET 请求。
	client := &http.Client{ // 创建 HTTP 客户端，超时由传入的 context 控制，并自定义重定向策略。
		CheckRedirect: func(req *http.Request, via []*http.Request) error { // 禁用自动重定向，让调用方直接看到原始 3xx 响应。
			return http.ErrUseLastResponse // 要求 net/http 返回最后一个 3xx 响应，而不是继续跟随 Location。
		}, // 结束 CheckRedirect 函数字段赋值。
	} // 结束 HTTP 客户端字面量。
	var body []byte                                                                   // 保存最终成功读取到的响应体。
	err := DoWithRetry(ctx, DefaultRetryConfig(), func(opCtx context.Context) error { // 用通用重试函数包裹一次 HTTP GET 操作。
		req, err := http.NewRequestWithContext(opCtx, http.MethodGet, url, nil) // 使用本次操作 context 创建 GET 请求。
		if err != nil {                                                         // 如果请求构造失败，通常是 URL 非法，不能重试。
			return err // 直接返回构造错误。
		} // 结束请求构造错误处理。
		resp, err := client.Do(req) // 发送 HTTP 请求并等待响应。
		if err != nil {             // 如果网络调用失败，就把错误标记为可重试。
			return &RetryableError{Cause: err} // 返回可重试错误，交给 DoWithRetry 决定是否继续。
		} // 结束网络错误处理。
		defer resp.Body.Close()            // 确保本次响应体最终关闭，避免连接泄漏。
		data, err := io.ReadAll(resp.Body) // 读取完整响应体，供成功或错误消息使用。
		if err != nil {                    // 如果读取响应体失败，通常可以尝试重试。
			return &RetryableError{Cause: err} // 把读取失败包装成可重试错误。
		} // 结束响应体读取错误处理。
		if resp.StatusCode >= http.StatusInternalServerError { // 如果服务端返回 5xx，就认为是临时服务端问题。
			return &RetryableError{Cause: fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))} // 返回可重试的服务端错误。
		} // 结束 5xx 处理。
		if resp.StatusCode >= http.StatusMultipleChoices { // 如果服务端返回 3xx 或 4xx，就认为不是本函数要自动跟进的成功响应。
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data)) // 返回不可重试错误，避免把重定向或客户端错误当作成功。
		} // 结束 3xx 和 4xx 处理。
		body = data // 只有 2xx 响应才把响应体保存为最终结果。
		return nil  // 返回 nil 表示 HTTP 请求成功。
	}) // 结束 DoWithRetry 调用。
	if err != nil { // 如果重试后仍然失败，就返回错误。
		return nil, err // 返回 nil 响应体和最终错误。
	} // 结束最终错误处理。
	return body, nil // 返回成功读取到的响应体。
} // 结束 HTTPGet 函数。

func main() { // main 函数演示普通操作、HTTP 重试、HTTP 400 和 context 超时四种场景。
	rand.Seed(time.Now().UnixNano())                                                  // 设置随机种子，让 jitter 在每次运行时略有不同。
	cfg := DefaultRetryConfig()                                                       // 获取适合本地演示的默认重试配置。
	attempt := 0                                                                      // 记录普通 operation 已经执行了几次。
	err := DoWithRetry(context.Background(), cfg, func(opCtx context.Context) error { // 演示一个前两次失败、第三次成功的普通操作。
		attempt++                                     // 每次执行 operation 时增加尝试次数。
		fmt.Printf("operation attempt %d\n", attempt) // 打印当前普通操作尝试次数。
		if attempt < 3 {                              // 前两次故意返回可重试错误。
			return &RetryableError{Cause: errors.New("temporary operation failure")} // 返回一个可重试的临时错误。
		} // 结束前两次失败模拟。
		fmt.Printf("operation succeeds on attempt %d\n", attempt) // 第三次成功时打印验收需要的成功语义。
		return nil                                                // 返回 nil 表示普通操作成功。
	}) // 结束普通操作重试演示。
	fmt.Printf("operation final error: %v\n", err)                                                    // 打印普通操作最终错误，成功时为 nil。
	httpAttempts := 0                                                                                 // 记录 HTTP 5xx 演示服务被调用次数。
	retryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 创建前两次 5xx、第三次成功的本地 HTTP 服务。
		httpAttempts++                                             // 每次请求到达时增加 HTTP 尝试次数。
		fmt.Printf("HTTP retry server attempt %d\n", httpAttempts) // 打印服务端收到的 HTTP 尝试次数。
		if httpAttempts < 3 {                                      // 前两次故意返回 5xx。
			http.Error(w, "temporary server error", http.StatusServiceUnavailable) // 返回 503，触发客户端重试。
			return                                                                 // 结束本次 5xx 响应。
		} // 结束前两次 5xx 模拟。
		fmt.Fprint(w, "HTTP retry success") // 第三次返回成功响应体。
	})) // 结束重试 HTTP 服务创建。
	defer retryServer.Close()                                                                       // main 退出时关闭重试演示服务。
	body, err := HTTPGet(context.Background(), retryServer.URL)                                     // 对本地 5xx 服务执行带重试的 HTTP GET。
	fmt.Printf("HTTP retry result: %s, error: %v\n", string(body), err)                             // 打印包含 HTTP retry success 的最终结果。
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 创建始终返回 400 的本地 HTTP 服务。
		http.Error(w, "bad request demo", http.StatusBadRequest) // 返回 400，用于演示不可重试错误。
	})) // 结束 400 HTTP 服务创建。
	defer badServer.Close()                                                              // main 退出时关闭 400 演示服务。
	_, err = HTTPGet(context.Background(), badServer.URL)                                // 对 400 服务执行 HTTP GET，应立即失败且不重试。
	fmt.Printf("HTTP 400 result: %v\n", err)                                             // 打印包含 HTTP 400 的不可重试错误。
	timeoutCfg := DefaultRetryConfig()                                                   // 获取一份默认配置用于 context 超时演示。
	timeoutCfg.BaseDelay = 100 * time.Millisecond                                        // 设置较长一点的基础等待，方便 timeout 在等待中触发。
	timeoutCfg.MaxDelay = 100 * time.Millisecond                                         // 固定最大等待时间，保证演示稳定。
	timeoutCfg.MaxTotalTime = time.Second                                                // 设置总超时大于外层 context，突出外层 context 提前中断。
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond) // 创建很短的外层超时 context。
	defer cancel()                                                                       // main 退出时释放 timeout context。
	timeoutAttempt := 0                                                                  // 记录 timeout 演示 operation 的尝试次数。
	err = DoWithRetry(timeoutCtx, timeoutCfg, func(opCtx context.Context) error {        // 执行一个一直返回可重试错误的操作，用于等待期间触发 context 超时。
		timeoutAttempt++                                                          // 增加 timeout 演示尝试次数。
		fmt.Printf("timeout operation attempt %d\n", timeoutAttempt)              // 打印 timeout 演示尝试次数。
		return &RetryableError{Cause: errors.New("still failing before timeout")} // 返回可重试错误，让 DoWithRetry 进入等待。
	}) // 结束 context 超时演示。
	fmt.Printf("timeout result: %v\n", err) // 打印 context deadline exceeded 或类似语义的最终错误。
} // 结束 main 函数。
