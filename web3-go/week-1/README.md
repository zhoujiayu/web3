# Week 1：Go 语法脱敏 + 标准库实战

## 本周目标

把 Java 的肌肉记忆替换成 Go。本周结束后，你应该：
- 不写 `class`，不写继承，用组合思维解决问题
- 习惯 `if err != nil`，不再想念 `try-catch`
- 能手写 goroutine + channel 的并发模式
- 会写单元测试，会用 `go test`

## 每日安排

| 天数 | 主题 | 核心概念 |
|------|------|----------|
| Day 1 | 带缓存的 HTTP Client | `struct`, `method`, `sync.RWMutex`, `map` |
| Day 2 | 依赖注入容器 | `map[string]interface{}`, `interface{}`, 类型断言 |
| Day 3 | 生产者-消费者 | `chan`, `goroutine`, `sync.WaitGroup` |
| Day 4 | Event Bus | `select`, `chan` 关闭，广播模式 |
| Day 5 | 连接池 | buffered channel 做信号量，资源管理 |
| Day 6 | 错误处理 + 重试 | `error` 接口，`defer`，`context.Context` |
| Day 7 | 单元测试 + 整理 | `testing`, `httptest`, `go test` |

## 提交规范

每天代码放在 `src/week1/dayX/` 下，必须：
1. `go mod init dayX-xxx`
2. 至少一个可运行的 `main.go`
3. 至少一个 `_test.go` 文件（Day 7 统一补也行）
4. `git commit -m "week1/dayX: xxx"`

---

开始 Day 1：`day-1.md`
