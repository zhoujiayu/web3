# Week 1 Day 7：单元测试 + 项目整理

## 目标

给本周代码补测试，整理成标准 Go 项目结构：
- 写单元测试（`testing` 包）
- 写 HTTP Mock 测试（`httptest`）
- 跑通 `go test ./...`
- 理解 Go 项目布局惯例

## 需要掌握的知识点

1. **`testing.T`**：`func TestXxx(t *testing.T)`，失败用 `t.Errorf` / `t.Fatalf`
2. **table-driven tests**：用一个切片装测试用例，for 循环跑，是 Go 的惯用写法
3. **`httptest.NewServer`**：Mock HTTP 服务端，不用真的发网络请求
4. **`testing.B`**：基准测试，`go test -bench=.`
5. **项目结构**：`cmd/`（可执行文件入口）、`pkg/`（可导入包）、`internal/`（私有包）

## 任务 1：给 Day 1 的 CacheClient 写测试

创建 `src/week1/day1/cache_test.go`：

```go
package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestCacheClient_Get_CacheHit(t *testing.T) {
	// TODO: 用 httptest.NewServer 创建一个返回固定字符串的服务
	// 创建 CacheClient，先 Get 一次，再 Get 第二次
	// 验证：第二次应该命中缓存（可以通过请求计数器来判断）
}

func TestCacheClient_Get_Concurrent(t *testing.T) {
	// TODO: 启动 10 个 goroutine 同时请求同一个 URL
	// 服务端只应该收到 1 次真实请求
}

func TestCacheClient_Get_Expired(t *testing.T) {
	// TODO: 设置 TTL = 100ms，先 Get，等 200ms，再 Get
	// 第二次应该重新发请求
}
```

## 任务 2：给 Day 3 的生产者-消费者写测试

```go
func TestDownload(t *testing.T) {
	// TODO: 用 httptest 模拟下载，验证 download() 函数返回正确的字节数
}

func TestProducerConsumer(t *testing.T) {
	// TODO: 生产 5 个 job，3 个 worker 消费，验证 5 个结果都被收集
	// 用 sync.WaitGroup 确保测试等所有 goroutine 完成
}
```

## 任务 3：项目结构整理

把本周代码整理成如下结构：

```
src/week1/
├── cache/              # Day 1
│   ├── client.go
│   └── client_test.go
├── di/                 # Day 2
│   ├── container.go
│   └── container_test.go
├── worker/             # Day 3 + Day 4（Event Bus 也可以用 worker 模式）
│   ├── pool.go
│   ├── bus.go
│   └── bus_test.go
├── pool/               # Day 5
│   ├── conn_pool.go
│   └── conn_pool_test.go
├── retry/              # Day 6
│   ├── retry.go
│   └── retry_test.go
└── cmd/
    └── demo/
        └── main.go     # 调用上面所有包的演示入口
```

## 验收标准

- [ ] `go test ./...` 全绿（至少每个包有 2 个测试用例）
- [ ] 使用 table-driven test 风格（至少一个文件）
- [ ] 有一个 benchmark 测试（如测试 CacheClient 并发性能）
- [ ] `go vet ./...` 没有警告
- [ ] `go mod tidy` 后 `go build ./cmd/demo` 能编译通过

## 参考

- [Go by Example: Testing](https://gobyexample.com/testing)
- [Go by Example: HTTP Testing](https://gobyexample.com/http-servers)（找 httptest 用法）
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

---

## Week 1 结束检查清单

- [ ] GitHub 有 7 天绿色 commit
- [ ] 能熟练手写 `struct` + `method`，不再想写 `class`
- [ ] 看到并发问题先想到 channel，不是锁
- [ ] 习惯 `if err != nil` 的显式处理

下周一进入 Week 2：连接 Ethereum 节点。
