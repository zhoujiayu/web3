# Week 1 Day 3：生产者-消费者

## 目标

实现一个并发图片下载器：
- 一个生产者 goroutine 生成 URL 列表
- 多个消费者 goroutine 并发下载
- 用 buffered channel 做任务队列
- 全部完成后主程序退出

## Java > Go 映射

| Java | Go |
|------|-----|
| `BlockingQueue<String> queue` | `urls := make(chan string, 100)` |
| `queue.put(url)` / `queue.take()` | `urls <- url` / `url := <-urls` |
| `ExecutorService` + `shutdown()` | 直接 `go func()`，用 `sync.WaitGroup` 等结束 |
| `AtomicInteger` 计数 | `sync/atomic` 或直接用 channel 传结果 |

## 需要掌握的知识点

1. **channel**：`make(chan T, size)`，无缓冲（同步）vs 有缓冲（异步）
2. **goroutine**：`go func()` 开新协程，不是线程，是更轻量的调度单位
3. **WaitGroup**：`Add(n)` / `Done()` / `Wait()`，等待一组 goroutine 完成
4. **range over channel**：`for url := range urls { ... }`，channel 关闭后自动退出循环
5. **单向 channel**：`chan<- T`（只写），`<-chan T`（只读），用于函数参数限制方向

## 代码框架

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Job 下载任务
type Job struct {
	URL      string
	Filename string
}

// Result 下载结果
type Result struct {
	Job   Job
	Size  int
	Error error
}

func main() {
	jobs := make(chan Job, 10)
	results := make(chan Result, 10)

	// 生产者：生成 20 个 URL
	go func() {
		defer close(jobs) // TODO: 为什么 defer？如果不 close 会怎样？
		for i := 0; i < 20; i++ {
			jobs <- Job{
				URL:      fmt.Sprintf("https://picsum.photos/200/300?random=%d", i),
				Filename: fmt.Sprintf("img_%d.jpg", i),
			}
		}
	}()

	var wg sync.WaitGroup
	numWorkers := 3

	// TODO: 启动 numWorkers 个消费者 goroutine
	// 每个消费者：从 jobs channel 取任务，下载，把结果发到 results
	// 记得 wg.Add 和 wg.Done

	// TODO: 启动一个 goroutine，等所有消费者完成后 close(results)
	// 提示：go func() { wg.Wait(); close(results) }()

	// 主 goroutine 收集结果
	successCount := 0
	failCount := 0
	for r := range results {
		if r.Error != nil {
			failCount++
			fmt.Printf("FAIL %s: %v\n", r.Job.Filename, r.Error)
		} else {
			successCount++
			fmt.Printf("OK %s: %d bytes\n", r.Job.Filename, r.Size)
		}
	}

	fmt.Printf("Total: %d success, %d fail\n", successCount, failCount)
}

// download 下载并返回字节数
func download(job Job) (int, error) {
	// TODO:
	// 1. http.Get(job.URL)
	// 2. resp.Body.Close()
	// 3. io.ReadAll(resp.Body) 或 io.Discard
	// 4. 返回长度和 nil
	// 提示：注意 resp.StatusCode 检查
	return 0, nil
}
```

## 验收标准

- [ ] 3 个 worker 并发下载，主程序最后打印成功/失败数
- [ ] 所有 goroutine 正确退出，没有 goroutine leak（程序能正常结束）
- [ ] `jobs` channel 被 producer close，`results` channel 等所有 worker 结束后 close
- [ ] 扩展：给每个 worker 一个 id，打印日志时带上 `[worker-1]` `[worker-2]`
- [ ] 扩展：用 `context.WithTimeout` 给整个下载流程加 30 秒超时

## 参考

- [Go by Example: Goroutines](https://gobyexample.com/goroutines)
- [Go by Example: Channels](https://gobyexample.com/channels)
- [Go by Example: Channel Buffering](https://gobyexample.com/channel-buffering)
- [Go by Example: Range over Channels](https://gobyexample.com/range-over-channels)
- [Go by Example: WaitGroups](https://gobyexample.com/waitgroups)

---

> 关键理解：channel 是 Go 并发的核心。Java 用共享内存 + 锁（`synchronized`、`Lock`），Go 提倡"通过通信来共享内存，而非通过共享内存来通信"。
