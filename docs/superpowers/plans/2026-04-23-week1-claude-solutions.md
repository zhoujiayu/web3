# Week 1 Claude Solutions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 一次性生成 Week 1 Day 2 到 Day 7 的 Claude 参考实现，并保持用户手写代码与 Claude 生成代码分离。

**Architecture:** 每天一个独立 `week/dayN_claude/` 目录，每个目录优先使用单文件 `main.go` 演示该天 Markdown 的核心知识点。示例统一使用本地 `httptest` 或本地 TCP 服务，避免外网依赖导致验证不稳定。

**Tech Stack:** Go 标准库，包括 `fmt`、`sync`、`context`、`net/http/httptest`、`net`、`errors`、`time`、`math/rand`。

---

## File Structure

- Create: `week/day2_claude/main.go`
  - 实现极简 DI 容器，覆盖 Singleton、Prototype、泛型 `MustGet[T]`、构造函数注入演示。
- Create: `week/day3_claude/main.go`
  - 实现生产者-消费者下载器，使用 3 个 worker、buffered channel、`sync.WaitGroup`、本地 `httptest` 服务。
- Create: `week/day4_claude/main.go`
  - 实现并发安全 Event Bus，支持订阅、发布、取消订阅、关闭、非阻塞发布和 `SubscribeAsync`。
- Create: `week/day5_claude/main.go`
  - 实现 TCP 连接池，支持固定最大连接数、`Get(ctx)` 超时、`Put` 复用、`Close` 清理、`Stats` 演示。
- Create: `week/day6_claude/main.go`
  - 实现带重试的 HTTP Client，支持 `RetryableError`、指数退避 + jitter、context 中断、4xx 不重试、5xx 重试。
- Create: `week/day7_claude/main.go`
  - 生成 Week 1 测试与项目整理演示说明，不强行重构 Day 1-6。
- Do not modify: `week/day1/main.go`
- Do not overwrite: `week/day1_claude/main.go`
- Reference only: `web3-go/week-1/day-2.md` through `web3-go/week-1/day-7.md`

---

### Task 1: Day 2 DI 容器参考实现

**Files:**
- Create: `week/day2_claude/main.go`
- Reference: `web3-go/week-1/day-2.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day2_claude/main.go` with a complete single-file implementation.

- [ ] **Step 2: 写入 Day 2 实现代码**

Use this implementation content:

```go
package main

import "fmt"

type Scope int

const (
	Singleton Scope = iota
	Prototype
)

type Provider func() interface{}

type registration struct {
	provider Provider
	scope    Scope
}

type Container struct {
	providers  map[string]registration
	singletons map[string]interface{}
}

func NewContainer() *Container {
	return &Container{
		providers:  make(map[string]registration),
		singletons: make(map[string]interface{}),
	}
}

func (c *Container) Register(name string, provider Provider, scope Scope) {
	c.providers[name] = registration{provider: provider, scope: scope}
}

func (c *Container) Get(name string) interface{} {
	reg, ok := c.providers[name]
	if !ok {
		panic(fmt.Sprintf("provider %q not registered", name))
	}
	if reg.scope == Prototype {
		return reg.provider()
	}
	if instance, ok := c.singletons[name]; ok {
		return instance
	}
	instance := reg.provider()
	c.singletons[name] = instance
	return instance
}

func MustGet[T any](c *Container, name string) T {
	value := c.Get(name)
	typed, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("provider %q has type %T, not requested type", name, value))
	}
	return typed
}

type Logger struct {
	Prefix string
}

type UserService struct {
	Logger *Logger
}

func main() {
	c := NewContainer()
	counter := 0
	c.Register("counter", func() interface{} {
		counter++
		return counter
	}, Singleton)
	fmt.Printf("singleton counter: %v %v\n", c.Get("counter"), c.Get("counter"))
	c.Register("prototype-counter", func() interface{} {
		counter++
		return counter
	}, Prototype)
	fmt.Printf("prototype counter: %v %v\n", c.Get("prototype-counter"), c.Get("prototype-counter"))
	c.Register("logger", func() interface{} {
		return &Logger{Prefix: "app"}
	}, Singleton)
	c.Register("user-service", func() interface{} {
		return &UserService{Logger: MustGet[*Logger](c, "logger")}
	}, Prototype)
	service1 := MustGet[*UserService](c, "user-service")
	service2 := MustGet[*UserService](c, "user-service")
	fmt.Printf("same logger: %v\n", service1.Logger == service2.Logger)
	fmt.Printf("same service: %v\n", service1 == service2)
}
```

Then add Chinese comments to every code line before saving, preserving Go syntax.

- [ ] **Step 3: 运行验证**

Run: `go run week/day2_claude/main.go`

Expected output includes:
- `singleton counter: 1 1`
- `prototype counter: 2 3`
- `same logger: true`
- `same service: false`

---

### Task 2: Day 3 生产者-消费者参考实现

**Files:**
- Create: `week/day3_claude/main.go`
- Reference: `web3-go/week-1/day-3.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day3_claude/main.go` with a complete single-file implementation.

- [ ] **Step 2: 实现结构与函数**

Implementation must include:
- `Job` with `URL` and `Filename`
- `Result` with `Job`, `Size`, `Error`, and `WorkerID`
- `download(ctx context.Context, client *http.Client, job Job) (int, error)`
- `worker(ctx context.Context, id int, client *http.Client, jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup)`
- `main()` using `context.WithTimeout`, `httptest.NewServer`, `jobs := make(chan Job, 10)`, `results := make(chan Result, 10)`, and 3 workers

Use local `httptest.NewServer` to return a deterministic byte body. Produce 20 jobs. Close `jobs` in the producer. Close `results` after `wg.Wait()`.

- [ ] **Step 3: 运行验证**

Run: `go run week/day3_claude/main.go`

Expected output includes:
- worker-tagged logs like `[worker-1] OK img_0.jpg: ... bytes`
- final summary `Total: 20 success, 0 fail`
- command exits without hanging

---

### Task 3: Day 4 Event Bus 参考实现

**Files:**
- Create: `week/day4_claude/main.go`
- Reference: `web3-go/week-1/day-4.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day4_claude/main.go` with a complete single-file implementation.

- [ ] **Step 2: 实现 Event Bus**

Implementation must include:
- `Event` with `Type string` and `Data interface{}`
- `EventBus` with `subscribers map[string][]chan Event`, `mu sync.RWMutex`, and `closed bool`
- `NewEventBus() *EventBus`
- `Subscribe(eventType string) (<-chan Event, func())`
- `Publish(event Event)` using copied subscriber slice and non-blocking `select { case ch <- event: default: }`
- `Close()` closing all subscriber channels and clearing the map
- `SubscribeAsync(eventType string, handler Handler) func()` using goroutine consumption

- [ ] **Step 3: 运行验证**

Run: `go run week/day4_claude/main.go`

Expected output includes:
- both channel subscribers receive `alice`
- after `cancel2()`, only remaining subscriber receives later event
- async handler prints at least one event
- command exits cleanly

---

### Task 4: Day 5 TCP 连接池参考实现

**Files:**
- Create: `week/day5_claude/main.go`
- Reference: `web3-go/week-1/day-5.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day5_claude/main.go` with a complete single-file implementation.

- [ ] **Step 2: 实现连接池**

Implementation must include:
- `Pool` with `factory func() (net.Conn, error)`, `pool chan net.Conn`, `maxSize int`, `mu sync.Mutex`, `current int`, `closed bool`
- `NewPool(maxSize int, factory func() (net.Conn, error)) (*Pool, error)` pre-creating `min(maxSize, 2)` connections
- `Get(ctx context.Context) (net.Conn, error)` supporting context timeout
- `Put(conn net.Conn)` returning reusable connections or closing when pool is full/closed
- `Close()` closing idle connections
- `Stats() (idle, active, total int)`
- local TCP echo-style server using `net.Listen("tcp", "127.0.0.1:0")`

- [ ] **Step 3: 运行验证**

Run: `go run week/day5_claude/main.go`

Expected output includes:
- 10 goroutines complete reads from pooled connections
- `total` never exceeds `maxSize`
- final stats show no panic and clean close

---

### Task 5: Day 6 重试 HTTP Client 参考实现

**Files:**
- Create: `week/day6_claude/main.go`
- Reference: `web3-go/week-1/day-6.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day6_claude/main.go` with a complete single-file implementation.

- [ ] **Step 2: 实现重试逻辑**

Implementation must include:
- `RetryableError` and `Error() string`
- `IsRetryable(err error) bool` using `errors.As`
- `RetryConfig` with `MaxRetries`, `BaseDelay`, `MaxDelay`, `MaxTotalTime`, `ShouldRetry`
- `DefaultRetryConfig()`
- `DoWithRetry(ctx context.Context, cfg RetryConfig, operation func() error) error`
- exponential backoff capped by `MaxDelay` and deterministic jitter small enough for quick demos
- `HTTPGet(ctx context.Context, url string) ([]byte, error)` using `http.NewRequestWithContext`
- local `httptest` cases for retry-then-success, 400 no retry, and context timeout

- [ ] **Step 3: 运行验证**

Run: `go run week/day6_claude/main.go`

Expected output includes:
- operation succeeds on third attempt
- HTTP 5xx scenario retries then succeeds
- HTTP 400 scenario returns immediately without retrying
- context timeout scenario stops early

---

### Task 6: Day 7 Week 1 测试与整理演示

**Files:**
- Create: `week/day7_claude/main.go`
- Reference: `web3-go/week-1/day-7.md`

- [ ] **Step 1: 创建目录和文件**

Create `week/day7_claude/main.go` with a single-file educational summary.

- [ ] **Step 2: 实现整理演示**

Implementation must:
- print the recommended `src/week1/` project layout from the Markdown
- print example commands: `go test ./...`, `go test -bench=.`, `go vet ./...`, `go mod tidy`, `go build ./cmd/demo`
- include a table-driven test example as a string literal
- include a benchmark example as a string literal
- explain that this project currently keeps daily Claude solutions in `week/dayN_claude/` to avoid overwriting hand-written practice code

- [ ] **Step 3: 运行验证**

Run: `go run week/day7_claude/main.go`

Expected output includes:
- `src/week1/cache/client.go`
- `go test ./...`
- `Benchmark`
- `week/dayN_claude`

---

### Task 7: 全量格式化与验证

**Files:**
- Verify: `week/day2_claude/main.go`
- Verify: `week/day3_claude/main.go`
- Verify: `week/day4_claude/main.go`
- Verify: `week/day5_claude/main.go`
- Verify: `week/day6_claude/main.go`
- Verify: `week/day7_claude/main.go`
- Verify unchanged: `week/day1/main.go`
- Verify preserved: `week/day1_claude/main.go`

- [ ] **Step 1: 格式化生成代码**

Run: `gofmt -w week/day2_claude/main.go week/day3_claude/main.go week/day4_claude/main.go week/day5_claude/main.go week/day6_claude/main.go week/day7_claude/main.go`

Expected: command exits with status 0.

- [ ] **Step 2: 逐个运行示例**

Run each command:

```bash
go run week/day2_claude/main.go
go run week/day3_claude/main.go
go run week/day4_claude/main.go
go run week/day5_claude/main.go
go run week/day6_claude/main.go
go run week/day7_claude/main.go
```

Expected: all commands exit with status 0.

- [ ] **Step 3: 检查手写代码未被覆盖**

Run: `git status --short week/day1 week/day1_claude week/day2_claude week/day3_claude week/day4_claude week/day5_claude week/day6_claude week/day7_claude`

Expected:
- `week/day1/main.go` remains unmodified by this task.
- New Claude directories are visible as untracked or added files.
- Existing `week/day1_claude/main.go` is not overwritten.

---

## Self-Review

- Spec coverage: Day 2 through Day 7 Markdown requirements are each mapped to one task and one `week/dayN_claude/main.go` output.
- Placeholder scan: No TBD/TODO placeholders remain in executable requirements; Task 1 contains minimal code skeleton but explicitly requires full Chinese-commented final code.
- Type consistency: Function names and structs are consistent with the Markdown: `Container`, `MustGet`, `Job`, `Result`, `EventBus`, `Pool`, `RetryConfig`, `HTTPGet`.
- Scope check: This plan intentionally keeps each day independent and does not refactor into `src/week1/`, because the project rule prioritizes `*_claude` directories and non-overwrite behavior.
