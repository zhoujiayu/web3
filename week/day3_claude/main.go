package main // 声明当前文件属于 main 包，表示这是一个可执行程序。

import ( // 导入本程序需要使用的 Go 标准库包。
	"context"           // 导入 context 包，用于控制请求和 worker 的取消与超时。
	"fmt"               // 导入 fmt 包，用于格式化字符串和打印结果。
	"io"                // 导入 io 包，用于读取 HTTP 响应体内容。
	"net/http"          // 导入 net/http 包，用于创建 HTTP 客户端和请求。
	"net/http/httptest" // 导入 httptest 包，用于创建本地稳定的测试 HTTP 服务。
	"sync"              // 导入 sync 包，用于 WaitGroup 等待多个 worker 完成。
	"time"              // 导入 time 包，用于设置整体下载流程的超时时间。
)

type Job struct { // 定义 Job 结构体，表示一个下载任务。
	URL      string // URL 保存需要下载的资源地址。
	Filename string // Filename 保存模拟保存文件时使用的文件名。
}

type Result struct { // 定义 Result 结构体，表示一个下载结果。
	Job      Job   // Job 保存本次结果对应的原始下载任务。
	Size     int   // Size 保存下载到的响应体字节数。
	Error    error // Error 保存下载过程中遇到的错误，成功时为 nil。
	WorkerID int   // WorkerID 保存处理该任务的 worker 编号。
}

func download(ctx context.Context, client *http.Client, job Job) (int, error) { // 定义 download 函数，使用带上下文的请求下载任务并返回字节数或错误。
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.URL, nil) // 创建绑定 ctx 的 GET 请求，使超时或取消能中断请求。
	if err != nil {                                                           // 判断创建请求时是否发生错误。
		return 0, err // 如果请求创建失败，返回 0 字节和对应错误。
	}
	resp, err := client.Do(req) // 使用传入的 HTTP 客户端执行请求。
	if err != nil {             // 判断执行请求时是否发生错误。
		return 0, err // 如果请求执行失败，返回 0 字节和对应错误。
	}
	defer resp.Body.Close()                                                               // 延迟关闭响应体，避免资源泄漏。
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices { // 检查状态码是否不在 2xx 成功范围内。
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode) // 如果状态码不是 2xx，返回包含状态码的错误。
	}
	body, err := io.ReadAll(resp.Body) // 读取完整响应体内容。
	if err != nil {                    // 判断读取响应体时是否发生错误。
		return 0, err // 如果读取失败，返回 0 字节和对应错误。
	}
	return len(body), nil // 返回响应体字节长度和 nil 错误表示下载成功。
}

func worker(ctx context.Context, id int, client *http.Client, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) { // 定义 worker 函数，从任务队列读取任务并把结果写入结果队列。
	defer wg.Done() // worker 退出前通知 WaitGroup 当前 goroutine 已完成。
	for {           // 使用循环持续等待任务或取消信号。
		select { // 使用 select 同时监听 ctx 取消和 jobs channel。
		case <-ctx.Done(): // 当上下文被取消或超时时进入该分支。
			return // 直接退出 worker，避免继续阻塞或处理新任务。
		case job, ok := <-jobs: // 从 jobs channel 中读取一个任务。
			if !ok { // 如果 jobs channel 已关闭且任务已读完。
				return // 退出 worker，表示没有更多任务需要处理。
			}
			size, err := download(ctx, client, job)                          // 调用 download 执行实际下载逻辑。
			result := Result{Job: job, Size: size, Error: err, WorkerID: id} // 构造包含任务、大小、错误和 worker 编号的结果。
			select {                                                         // 使用 select 发送结果，同时支持 ctx 取消时退出。
			case <-ctx.Done(): // 如果发送结果前上下文已经取消。
				return // 退出 worker，避免在结果 channel 上继续等待。
			case results <- result: // 将本次任务的处理结果发送给主 goroutine。
			} // 结束发送结果的 select 语句。
		} // 结束读取任务的 select 语句。
	} // 结束 worker 主循环。
} // 结束 worker 函数。

func main() { // 定义程序入口函数。
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)                     // 创建 30 秒超时的根上下文，限制整个下载流程的最长运行时间。
	defer cancel()                                                                               // main 结束时释放上下文相关资源。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 创建本地测试 HTTP 服务，提供稳定响应体。
		_, _ = w.Write([]byte("stable image bytes")) // 写入固定响应体，忽略测试服务写入错误。
	})) // 结束 httptest 服务处理函数定义。
	defer server.Close()             // main 结束时关闭本地测试 HTTP 服务。
	client := server.Client()        // 使用 httptest 提供的客户端访问本地测试服务。
	jobs := make(chan Job, 10)       // 创建容量为 10 的 Job 缓冲 channel，作为生产者到消费者的任务队列。
	results := make(chan Result, 10) // 创建容量为 10 的 Result 缓冲 channel，作为 worker 到主 goroutine 的结果队列。
	go func() {                      // 启动生产者 goroutine，负责生成下载任务。
		defer close(jobs)         // 生产者结束时关闭 jobs channel，通知 worker 不再有新任务。
		for i := 0; i < 20; i++ { // 循环生成 20 个下载任务。
			job := Job{URL: fmt.Sprintf("%s/image/%d", server.URL, i), Filename: fmt.Sprintf("img_%d.jpg", i)} // 构造当前序号对应的下载任务。
			select {                                                                                           // 使用 select 支持在发送任务时响应 ctx 取消。
			case <-ctx.Done(): // 如果上下文已经取消。
				return // 结束生产者 goroutine。
			case jobs <- job: // 将任务发送到 jobs channel。
			} // 结束发送任务的 select 语句。
		} // 结束任务生成循环。
	}() // 立即执行生产者 goroutine。
	var wg sync.WaitGroup              // 声明 WaitGroup，用于等待所有 worker 完成。
	numWorkers := 3                    // 设置 worker 数量为 3。
	for i := 1; i <= numWorkers; i++ { // 循环启动指定数量的 worker。
		wg.Add(1)                                     // 每启动一个 worker，就把 WaitGroup 计数加一。
		go worker(ctx, i, client, jobs, results, &wg) // 启动 worker goroutine 处理下载任务。
	} // 结束 worker 启动循环。
	go func() { // 启动关闭结果队列的协调 goroutine。
		wg.Wait()      // 等待所有 worker 完成。
		close(results) // 所有 worker 完成后关闭 results channel，通知主 goroutine 停止读取。
	}() // 立即执行协调 goroutine。
	successCount := 0             // 初始化成功任务计数。
	failCount := 0                // 初始化失败任务计数。
	for result := range results { // 主 goroutine 持续读取结果，直到 results channel 被关闭。
		if result.Error != nil { // 判断当前结果是否包含错误。
			failCount++                                                                                 // 如果失败，失败计数加一。
			fmt.Printf("[worker-%d] FAIL %s: %v\n", result.WorkerID, result.Job.Filename, result.Error) // 打印带 worker 标签的失败日志。
			continue                                                                                    // 当前失败结果已经处理完，继续读取下一个结果。
		} // 结束失败分支判断。
		successCount++                                                                                 // 如果成功，成功计数加一。
		fmt.Printf("[worker-%d] OK %s: %d bytes\n", result.WorkerID, result.Job.Filename, result.Size) // 打印带 worker 标签的成功日志。
	} // 结束结果读取循环。
	fmt.Printf("Total: %d success, %d fail\n", successCount, failCount) // 打印最终成功和失败总数。
} // 结束 main 函数。
