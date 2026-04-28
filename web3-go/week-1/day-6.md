# Week 1 Day 6：错误处理 + 重试 + Context

## 目标

实现一个带重试机制的 HTTP Client：
- 请求失败时自动重试（指数退避）
- 支持总超时控制（Context）
- 可配置重试次数、退避策略
- 区分"可重试错误"和"不可重试错误"

## Java > Go 映射

| Java | Go |
|------|-----|
| `try { ... } catch (IOException e) { retry() }` | `if err != nil { retry() }` |
| `RetryTemplate` (Spring Retry) | 手写 for 循环 + time.Sleep |
| `ExecutorService.submit().get(timeout)` | `ctx, cancel := context.WithTimeout(...)` |
| `Callable<T>` | 函数闭包 `func() error` |

## 需要掌握的知识点

1. **`error` 接口**：`type error interface { Error() string }`，不是异常，是返回值
2. **`errors.Is` / `errors.As`**：Go 1.13+ 错误链处理，类似 Java 异常 cause
3. **`context.Context`**：超时、取消信号传递，必须显式传递（不是 ThreadLocal！）
4. **`defer`**：函数退出时执行，类似 `finally`，但写在开头，更直观
5. **指数退避**：`base * (2 ^ attempt)` + 随机抖动（jitter）

## 代码框架

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// RetryableError 可重试的错误标记
type RetryableError struct {
	Cause error
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable: %v", e.Cause)
}

// IsRetryable 判断错误是否可重试
func IsRetryable(err error) bool {
	// TODO: 用 errors.As 判断 err 是否是 *RetryableError
	return false
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries  int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	ShouldRetry func(error) bool
}

// DefaultRetryConfig 默认配置
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		ShouldRetry: func(err error) bool {
			// TODO:
			// 1. 如果是 RetryableError，返回 true
			// 2. 如果是网络超时（可以用 errors.Is(err, context.DeadlineExceeded)），返回 true
			// 3. 其他（如 4xx 客户端错误）返回 false
			return false
		},
	}
}

// DoWithRetry 执行带重试的操作
// operation: 返回 error 的函数
func DoWithRetry(ctx context.Context, cfg RetryConfig, operation func() error) error {
	// TODO:
	// 1. for i := 0; i <= cfg.MaxRetries; i++
	// 2. 调用 operation()
	// 3. 如果 nil，直接返回
	// 4. 判断 ShouldRetry，如果 false，直接返回 err
	// 5. 计算退避时间：min(base*2^i, maxDelay) + jitter
	// 6. 用 select { case <-ctx.Done(): return ctx.Err(); case <-time.After(delay): } 等待
	// 7. 重试次数用完，返回最后一个 err
	return nil
}

func main() {
	cfg := DefaultRetryConfig()

	attempt := 0
	operation := func() error {
		attempt++
		fmt.Printf("Attempt %d\n", attempt)

		// 模拟：前两次失败，第三次成功
		if attempt < 3 {
			return &RetryableError{Cause: errors.New("network timeout")}
		}
		return nil
	}

	ctx := context.Background()
	err := DoWithRetry(ctx, cfg, operation)
	fmt.Printf("Final: %v\n", err)

	// TODO: 第二个练习：封装一个 HTTP GET 函数，内部用 DoWithRetry
	// body, err := HTTPGet(ctx, "https://httpbin.org/delay/1")
}

// HTTPGet 带重试的 HTTP GET
func HTTPGet(ctx context.Context, url string) ([]byte, error) {
	// TODO:
	// 1. 用 DoWithRetry 包装 http.Get
	// 2. 注意：http.NewRequestWithContext(ctx, ...) + client.Do(req) 才能支持 context 取消
	// 3. 如果 resp.StatusCode >= 500，返回 RetryableError
	// 4. 如果 resp.StatusCode >= 400 && < 500，返回不可重试错误
	return nil, nil
}
```

## 验收标准

- [ ] 可重试错误会自动重试，最多 MaxRetries 次
- [ ] 不可重试错误（如 400 Bad Request）立即返回，不重试
- [ ] context 取消或超时，立即中断，不再等待重试
- [ ] 退避时间符合指数增长 + 抖动（打印出来肉眼可见）
- [ ] 扩展：实现 `RetryConfig.MaxTotalTime`，限制整个重试流程的总耗时

## 参考

- [Go by Example: Errors](https://gobyexample.com/errors)
- [Go by Example: Context](https://gobyexample.com/context)
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)

---

> 关键理解：Go 的错误处理显式、啰嗦，但极其清晰。Java 的异常堆栈在微服务里经常跨几十个帧找不到根因，Go 的 `if err != nil` 强迫你在每一层做出决策：处理、包装、还是继续传。
