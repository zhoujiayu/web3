package main // 声明当前文件属于 main 包，方便直接运行本周整理演示。

import "fmt" // 引入 fmt，用于打印项目结构、命令和测试示例。

func main() { // main 函数负责输出 Week 1 的测试与项目整理演示说明。
	layout := `src/week1/
├── cache/
│   ├── client.go
│   └── client_test.go
├── di/
│   ├── container.go
│   └── container_test.go
├── worker/
│   ├── pool.go
│   ├── bus.go
│   └── bus_test.go
├── pool/
│   ├── conn_pool.go
│   └── conn_pool_test.go
├── retry/
│   ├── retry.go
│   └── retry_test.go
└── cmd/
    └── demo/
        └── main.go` // 定义 Markdown 推荐的标准 Go 项目结构字符串。
	tableDrivenExample := `func TestCacheClient_Get_CacheHit(t *testing.T) {
	tests := []struct {
		name     string
		ttl      time.Duration
		requests int
		wantHits int
	}{
		{name: "single hit", ttl: time.Second, requests: 2, wantHits: 1},
		{name: "expired", ttl: time.Millisecond, requests: 2, wantHits: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 在这里创建 httptest 服务并断言缓存命中次数。
		})
	}
}` // 定义 table-driven test 示例字符串。
	benchmarkExample := `func BenchmarkCacheClient_Get(b *testing.B) {
	client := NewCacheClient(time.Second)
	for i := 0; i < b.N; i++ {
		_, _ = client.Get("http://example.test")
	}
}` // 定义 benchmark 示例字符串。
	fmt.Println("Week 1 测试与项目整理演示")                                         // 打印演示标题，提示这是本周总结输出。
	fmt.Println()                                                           // 打印空行，让输出分段更清晰。
	fmt.Println("推荐的标准 Go 项目结构：")                                           // 打印结构说明标题。
	fmt.Println(layout)                                                     // 打印题目建议的 src/week1 项目结构。
	fmt.Println()                                                           // 打印空行，分隔下一个说明部分。
	fmt.Println("关键路径示例：")                                                  // 打印关键路径说明标题。
	fmt.Println("- src/week1/cache/client.go")                              // 打印 Day 1 对应的标准路径示例。
	fmt.Println("- src/week1/di/container.go")                              // 打印 Day 2 对应的标准路径示例。
	fmt.Println("- src/week1/worker/pool.go")                               // 打印 Day 3/4 对应的标准路径示例。
	fmt.Println("- src/week1/pool/conn_pool.go")                            // 打印 Day 5 对应的标准路径示例。
	fmt.Println("- src/week1/retry/retry.go")                               // 打印 Day 6 对应的标准路径示例。
	fmt.Println("- src/week1/cmd/demo/main.go")                             // 打印 demo 入口的标准路径示例。
	fmt.Println()                                                           // 打印空行，分隔命令说明。
	fmt.Println("推荐命令：")                                                    // 打印常用命令说明标题。
	fmt.Println("- go test ./...")                                          // 打印运行全部单元测试的命令。
	fmt.Println("- go test -bench=.")                                       // 打印运行 benchmark 的命令。
	fmt.Println("- go vet ./...")                                           // 打印静态检查命令。
	fmt.Println("- go mod tidy")                                            // 打印整理模块依赖命令。
	fmt.Println("- go build ./cmd/demo")                                    // 打印构建演示程序命令。
	fmt.Println()                                                           // 打印空行，分隔测试示例说明。
	fmt.Println("table-driven test 示例：")                                    // 打印表驱动测试标题。
	fmt.Println(tableDrivenExample)                                         // 打印包含 func Test 和 tests := []struct 的示例。
	fmt.Println()                                                           // 打印空行，分隔 benchmark 示例。
	fmt.Println("benchmark 示例：")                                            // 打印 benchmark 标题。
	fmt.Println(benchmarkExample)                                           // 打印包含 func Benchmark 的示例。
	fmt.Println()                                                           // 打印空行，分隔项目实际约定说明。
	fmt.Println("当前项目的实际约定：")                                               // 打印当前仓库规则说明标题。
	fmt.Println("- 为了不覆盖你的手写练习代码，Claude 生成的每日参考实现保留在 week/dayN_claude。")    // 打印 daily Claude solutions 的目录约定说明。
	fmt.Println("- 例如：week/day1_claude、week/day2_claude、week/day3_claude。") // 打印具体示例目录，帮助用户理解约定。
	fmt.Println("- 这种方式便于你保留自己的练习版本，同时对照参考实现学习。")                           // 打印该约定的目的说明。
} // 结束 main 函数。
