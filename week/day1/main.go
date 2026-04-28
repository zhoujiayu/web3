package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"
)

// 缓存条目
type cacheEntry struct {
	body       string
	expireTime time.Time
}

// 缓存client客户端
type CacheClient struct {
	client   *http.Client
	cacheMap map[string]cacheEntry
	mu       sync.RWMutex
	ttl      time.Duration
}

// 创建client客户端
func CreateCacheClient(ttl time.Duration) *CacheClient {
	return &CacheClient{
		client:   &http.Client{Timeout: 5 * time.Second},
		cacheMap: make(map[string]cacheEntry),
		ttl:      ttl,
	}
}

func (c *CacheClient) Get(url string) (string, error) {

	now := time.Now()
	// 枷锁
	c.mu.RLock()
	// 根据url读取缓存
	entry, ok := c.cacheMap[url]
	if ok && now.Before(entry.expireTime) {
		c.mu.RUnlock()
		return entry.body, nil
	}

	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	now = time.Now()            // 重新获取当前时间，保证后续判断更准确。
	entry, ok = c.cacheMap[url] // 在写锁内再次检查缓存，防止并发场景下重复请求。
	if ok && now.Before(entry.expireTime) {
		return entry.body, nil
	}

	// 过期删除缓存内容
	if ok && now.After(entry.expireTime) {
		delete(c.cacheMap, url)
	}

	// 发起请求
	reponse, err := c.client.Get(url)
	if err != nil {
		return "", err
	}

	defer reponse.Body.Close()
	bodyBytes, err := io.ReadAll(reponse.Body)
	if err != nil {

		return "", err
	}

	body := string(bodyBytes)
	c.cacheMap[url] = cacheEntry{
		body:       body,
		expireTime: time.Now().Add(c.ttl),
	}

	return body, nil
}

func main() {
	log.Printf("start")

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&requestCount, 1)
		fmt.Fprintf(w, "respones-%d", current)
	}))
	defer server.Close()

	client := CreateCacheClient(2 * time.Second)
	url := server.URL

	start1 := time.Now()          // 记录第一次请求的开始时间。
	body1, err := client.Get(url) // 第一次请求应该走真实 HTTP。
	if err != nil {               // 如果第一次请求失败。
		panic(err) // 直接 panic，方便在学习阶段快速发现问题。
	}
	fmt.Printf("第一次请求: body=%s, cost=%v, requestCount=%d\n", body1, time.Since(start1), atomic.LoadInt32(&requestCount)) // 打印第一次请求结果。

	start2 := time.Now()
	body2, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	fmt.Printf("第二次请求: body=%s, cost=%v, requestCount=%d\n", body2, time.Since(start2), atomic.LoadInt32(&requestCount)) // 打印第一次请求结果。

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			body, err := client.Get(url)
			if err != nil {
				fmt.Printf("goroutine-%d error: %v\n", index, err) // 打印错误信息。
				return
			}
			fmt.Printf("goroutine-%d body=%s\n", index, body) // 打印每个 goroutine 拿到的结果。
		}(i)
	}
	wg.Wait()

	start3 := time.Now()
	body3, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	fmt.Printf("第三次请求: body3=%s, cost=%v, requestCount=%d\n", body3, time.Since(start3), atomic.LoadInt32(&requestCount)) // 打印第一次请求结果。

}
