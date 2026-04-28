# Week 1 Day 1：带缓存的 HTTP Client

## 目标

实现一个带内存缓存的 HTTP Client：
- 缓存 GET 请求的响应（URL > Response Body）
- 支持 TTL（过期自动清理）
- 并发安全（多个 goroutine 同时读写）

## Java > Go 映射

| Java | Go |
|------|-----|
| `class CacheClient { ... }` | `type CacheClient struct { ... }` |
| `private final Map<String, String> cache` | `cache map[string]cacheEntry`（小写开头 = private） |
| `synchronized` / `ReentrantReadWriteLock` | `sync.RWMutex` |
| `new CacheClient()` | `NewCacheClient(ttl time.Duration) *CacheClient`（工厂函数） |
| `null` | `nil`，但 Go 有零值，struct 初始化时字段自动有默认值 |

## 需要掌握的知识点

1. **struct**：Go 没有 class，`type T struct { ... }`
2. **方法**：`func (c *CacheClient) Get(...) (...)`，值接收者 vs 指针接收者
3. **可见性**：首字母大写 = public，小写 = package private
4. **map**：引用类型，未初始化时是 `nil`，必须用 `make` 创建
5. **sync.RWMutex**：读多写少场景用 `RLock()` / `RUnlock()`

## 代码框架

创建 `src/week1/day1/main.go`，自己补全实现：

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// cacheEntry 缓存条目
type cacheEntry struct {
	body      string
	expiresAt time.Time
}

// CacheClient 带缓存的 HTTP 客户端
type CacheClient struct {
	client *http.Client
	// TODO: 补全字段
	// 1. 缓存 map
	// 2. 互斥锁
	// 3. TTL 时间
}

// NewCacheClient 创建客户端（工厂函数）
func NewCacheClient(ttl time.Duration) *CacheClient {
	// TODO: 初始化，注意 map 要用 make
}

// Get 发送 GET 请求，优先走缓存
func (c *CacheClient) Get(url string) (string, error) {
	// TODO:
	// 1. 加读锁，查缓存
	// 2. 如果命中且未过期，直接返回
	// 3. 释放读锁，加写锁（注意：不能从读锁直接升级写锁！）
	// 4. 再次检查缓存（双重检查，类似 Java DCL）
	// 5. 发送真实 HTTP 请求
	// 6. 写入缓存，释放写锁
	// 7. 返回结果
}

func main() {
	client := NewCacheClient(5 * time.Second)

	// 第一次请求，应该走 HTTP
	body1, err := client.Get("https://api.github.com")
	if err != nil {
		panic(err)
	}
	fmt.Printf("First request, len=%d\n", len(body1))

	// 第二次请求，应该走缓存
	body2, err := client.Get("https://api.github.com")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Second request, len=%d\n", len(body2))

	// TODO: 加一段并发测试，用 sync.WaitGroup 开 10 个 goroutine 同时请求同一个 URL
}
```

## 验收标准

- [ ] `go run main.go` 能跑通，第二次请求明显比第一次快
- [ ] 开 10 个 goroutine 并发请求同一个 URL，不会 panic，不会重复发 HTTP（可用计数器验证）
- [ ] 缓存过期后，再次请求能重新拉取
- [ ] 代码里不出现 `class`、`extends`、`synchronized`

## 参考

- [Go by Example: Structs](https://gobyexample.com/structs)
- [Go by Example: Methods](https://gobyexample.com/methods)
- [Go by Example: Mutexes](https://gobyexample.com/mutexes)

---

完成后在 `progress.md` 打卡，然后进入 `day-2.md`。
