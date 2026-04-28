# Week 1 Day 5：连接池

## 目标

实现一个 TCP 连接池：
- 初始化时建立固定数量的连接
- `Get()` 从池中取连接，池空时阻塞等待
- `Put()` 归还连接，坏连接丢弃并重建
- 支持池满时拒绝或等待

Web3 场景中：RPC 节点连接池、数据库连接池、Redis 连接池都是同一模式。

## 需要掌握的知识点

1. **buffered channel 做信号量**：`pool := make(chan net.Conn, size)`
2. **channel 的阻塞特性**：满了发送阻塞，空了接收阻塞，天然适合做池
3. **资源健康检查**：`Put` 时检查连接是否可用（如 `conn.SetDeadline` 探测）
4. **context**：`Get(ctx)` 支持超时取消

## 代码框架

```go
package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Pool 连接池
type Pool struct {
	// TODO:
	// factory: func() (net.Conn, error)  // 创建连接的工厂函数
	// pool: chan net.Conn                 // 用 buffered channel 存连接
	// maxSize int
	// mu sync.Mutex
	// current int  // 当前总连接数（已借出 + 池中）
}

// NewPool 创建连接池
// maxSize: 最大连接数
// factory: 创建新连接的函数
func NewPool(maxSize int, factory func() (net.Conn, error)) (*Pool, error) {
	// TODO:
	// 1. 初始化 Pool
	// 2. 预创建 min(maxSize, 2) 个连接放入 pool
	// 3. 返回 Pool
	return nil, nil
}

// Get 获取连接，支持超时
func (p *Pool) Get(ctx context.Context) (net.Conn, error) {
	// TODO:
	// 方案 A：直接从 pool channel 取（阻塞到有人 Put）
	// 方案 B：如果池空且当前总数 < maxSize，动态创建（需要 current 计数 + 锁）
	// 方案 C：结合 context，超时返回 error
	return nil, nil
}

// Put 归还连接
func (p *Pool) Put(conn net.Conn) {
	// TODO:
	// 1. 检查连接是否健康（如 conn.RemoteAddr() 是否 nil，或简单 ping）
	// 2. 如果健康，放回 pool channel
	// 3. 如果不健康，关闭连接，current--（如果用了动态创建）
}

// Close 关闭整个池
func (p *Pool) Close() {
	// TODO: 关闭 pool channel，遍历关闭所有 conn
}

func main() {
	// 用 TCP 连接到 example.com:80 作为测试
	factory := func() (net.Conn, error) {
		return net.DialTimeout("tcp", "example.com:80", 3*time.Second)
	}

	pool, err := NewPool(3, factory)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	// 并发取连接，发送 HTTP 请求
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			conn, err := pool.Get(ctx)
			if err != nil {
				fmt.Printf("[%d] Get failed: %v\n", id, err)
				return
			}
			defer pool.Put(conn)

			// 发送一个极简 HTTP 请求
			fmt.Fprintf(conn, "GET / HTTP/1.0\r\nHost: example.com\r\n\r\n")
			buf := make([]byte, 1024)
			n, _ := conn.Read(buf)
			fmt.Printf("[%d] Read %d bytes\n", id, n)
		}(i)
	}

	wg.Wait()
}
```

## 验收标准

- [ ] 连接池大小固定，并发获取不超过 maxSize
- [ ] `Get` 支持 context 超时
- [ ] `Put` 归还的连接能复用
- [ ] 扩展：实现 `Stats() (idle, active, total int)` 返回当前池状态
- [ ] 扩展：空闲连接超时时自动关闭（后台 goroutine 定时清理）

## 参考

- [Go by Example: Worker Pools](https://gobyexample.com/worker-pools)
- [database/sql 连接池源码](https://cs.opensource.google/go/go/+/master:src/database/sql/sql.go)（高级，选读）

---

> 对比 Java：HikariCP 是 Java 世界最优秀的连接池，核心也是 `BlockingQueue` + CAS。Go 里用 channel 比 `BlockingQueue` 更自然，代码量更少。
