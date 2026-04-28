package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

// cacheEntry 表示缓存中的一个条目。
type cacheEntry struct {
	body      string    // body 保存 HTTP 响应体内容。
	expiresAt time.Time // expiresAt 表示这个缓存条目的过期时间。
}

// CacheClient 表示一个带内存缓存的 HTTP 客户端。
type CacheClient struct {
	client *http.Client          // client 负责发起真实的 HTTP 请求。
	cache  map[string]cacheEntry // cache 使用 URL 作为 key 保存响应内容和过期时间。
	mu     sync.RWMutex          // mu 用来保护 cache，确保并发读写安全。
	ttl    time.Duration         // ttl 表示每个缓存条目的存活时间。
}

// NewCacheClient 创建一个新的带缓存的 HTTP 客户端。
func NewCacheClient(ttl time.Duration) *CacheClient {
	return &CacheClient{ // 返回初始化完成的 CacheClient 指针。
		client: &http.Client{Timeout: 5 * time.Second}, // client 使用一个带超时的默认 HTTP 客户端。
		cache:  make(map[string]cacheEntry),            // cache 必须用 make 初始化，否则写入时会 panic。
		ttl:    ttl,                                    // ttl 保存调用方传入的缓存过期时长。
	}
}

// Get 发送 GET 请求，并优先从缓存中读取结果。
func (c *CacheClient) Get(url string) (string, error) {
	now := time.Now() // now 记录当前时间，方便后续做过期判断。

	c.mu.RLock()                           // 先加读锁，适合读多写少的缓存读取场景。
	entry, ok := c.cache[url]              // 从缓存里读取当前 URL 对应的条目。
	if ok && now.Before(entry.expiresAt) { // 如果缓存命中并且还没有过期。
		c.mu.RUnlock()         // 返回前必须先释放读锁。
		return entry.body, nil // 直接返回缓存内容，避免再次发起 HTTP 请求。
	}
	c.mu.RUnlock() // 如果未命中或已过期，也要先释放读锁。

	c.mu.Lock()         // 再加写锁，准备处理过期删除、双重检查和真实请求。
	defer c.mu.Unlock() // 函数结束前释放写锁，避免忘记解锁。

	now = time.Now()                       // 重新获取当前时间，保证后续判断更准确。
	entry, ok = c.cache[url]               // 在写锁内再次检查缓存，防止并发场景下重复请求。
	if ok && now.Before(entry.expiresAt) { // 如果别的 goroutine 已经把缓存写好了。
		return entry.body, nil // 直接复用最新缓存结果。
	}
	if ok && !now.Before(entry.expiresAt) { // 如果缓存存在但已经过期。
		delete(c.cache, url) // 顺手删除过期缓存，避免旧数据残留。
	}

	resp, err := c.client.Get(url) // 在写锁保护下发起真实 HTTP 请求，确保并发下不重复请求。
	if err != nil {                // 如果请求失败。
		return "", err // 直接返回错误给调用方。
	}
	defer resp.Body.Close() // 读取完响应体后关闭 body，避免资源泄漏。

	bodyBytes, err := io.ReadAll(resp.Body) // 读取完整响应体内容。
	if err != nil {                         // 如果读取响应体失败。
		return "", err // 直接返回错误。
	}

	body := string(bodyBytes)  // 把字节切片转换为字符串，便于缓存和返回。
	c.cache[url] = cacheEntry{ // 把新的响应内容写入缓存。
		body:      body,                  // body 保存本次请求拿到的响应体。
		expiresAt: time.Now().Add(c.ttl), // expiresAt 设置为当前时间加上 TTL。
	}

	return body, nil // 返回最新响应结果。
}

// main 用来演示缓存命中、并发安全和 TTL 过期后的重新拉取。
func main() {
	var requestCount int32 // requestCount 用来统计底层真实 HTTP 请求次数。

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { // 启动一个本地测试 HTTP 服务。
		current := atomic.AddInt32(&requestCount, 1) // 每次进入处理函数都说明发生了一次真实请求。
		fmt.Fprintf(w, "response-%d", current)       // 返回带计数的响应内容，便于观察是否重复请求。
	})) // 这里结束测试服务的处理逻辑定义。
	defer server.Close() // main 结束前关闭测试服务。

	client := NewCacheClient(2 * time.Second) // 创建一个 TTL 为 2 秒的缓存客户端。
	url := server.URL                         // 使用本地测试服务地址作为请求目标。

	start1 := time.Now()          // 记录第一次请求的开始时间。
	body1, err := client.Get(url) // 第一次请求应该走真实 HTTP。
	if err != nil {               // 如果第一次请求失败。
		panic(err) // 直接 panic，方便在学习阶段快速发现问题。
	}
	fmt.Printf("第一次请求: body=%s, cost=%v, requestCount=%d\n", body1, time.Since(start1), atomic.LoadInt32(&requestCount)) // 打印第一次请求结果。

	start2 := time.Now()          // 记录第二次请求的开始时间。
	body2, err := client.Get(url) // 第二次请求应该命中缓存。
	if err != nil {               // 如果第二次请求失败。
		panic(err) // 直接 panic。
	}
	fmt.Printf("第二次请求: body=%s, cost=%v, requestCount=%d\n", body2, time.Since(start2), atomic.LoadInt32(&requestCount)) // 打印第二次请求结果。

	var wg sync.WaitGroup     // wg 用来等待所有 goroutine 执行完成。
	for i := 0; i < 10; i++ { // 启动 10 个 goroutine 并发请求同一个 URL。
		wg.Add(1)            // 每启动一个 goroutine 前先把计数加一。
		go func(index int) { // 启动一个匿名 goroutine。
			defer wg.Done()              // goroutine 结束时通知 WaitGroup。
			body, err := client.Get(url) // 并发请求同一个 URL。
			if err != nil {              // 如果 goroutine 内请求失败。
				fmt.Printf("goroutine-%d error: %v\n", index, err) // 打印错误信息。
				return                                             // 出错后直接返回。
			}
			fmt.Printf("goroutine-%d body=%s\n", index, body) // 打印每个 goroutine 拿到的结果。
		}(i) // 把当前循环下标传入 goroutine，避免闭包变量陷阱。
	}
	wg.Wait()                                                              // 等待全部 10 个 goroutine 都执行完成。
	fmt.Printf("并发请求后 requestCount=%d\n", atomic.LoadInt32(&requestCount)) // 打印并发请求后的真实请求次数。

	time.Sleep(3 * time.Second) // 休眠超过 TTL，确保缓存过期。

	start3 := time.Now()          // 记录过期后再次请求的开始时间。
	body3, err := client.Get(url) // 缓存过期后这次请求应该重新走真实 HTTP。
	if err != nil {               // 如果第三次请求失败。
		panic(err) // 直接 panic。
	}
	fmt.Printf("过期后请求: body=%s, cost=%v, requestCount=%d\n", body3, time.Since(start3), atomic.LoadInt32(&requestCount)) // 打印过期后的请求结果。
}
