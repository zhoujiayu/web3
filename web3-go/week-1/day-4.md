# Week 1 Day 4：Event Bus（事件总线）

## 目标

实现一个内存中的 Event Bus：
- 支持订阅某个事件类型，回调函数收到事件
- 支持广播：一个事件发给所有订阅者
- 支持取消订阅
- 并发安全

这是 Web3 中的核心模式：监听链上事件 > 广播给多个处理器。

## 需要掌握的知识点

1. **channel 广播**：一个 sender > 多个 receiver，每个 receiver 有自己的 channel
2. **`select`**：多路复用，类似 Java NIO Selector，但用于 channel
3. **channel 关闭**：广播关闭信号，所有接收方都能收到 `ok == false` 或 range 退出
4. **`sync.Map`**：如果普通 map 的并发操作太麻烦，可以用 `sync.Map`（但性能不如 RWMutex + map）
5. **函数类型作为回调**：`type Handler func(Event)`

## 代码框架

```go
package main

import (
	"fmt"
	"sync"
	"time"
)

// Event 事件结构
type Event struct {
	Type string
	Data interface{}
}

// EventBus 事件总线
type EventBus struct {
	// TODO:
	// subscribers: map[eventType][]chan Event
	// 需要一个锁保护 subscribers
	// 每个 subscriber 分配一个带缓冲的 channel
}

// Subscribe 订阅事件，返回一个只读 channel 和取消函数
func (eb *EventBus) Subscribe(eventType string) (<-chan Event, func()) {
	// TODO:
	// 1. 创建 ch := make(chan Event, 10)
	// 2. 加锁，把 ch 加入 subscribers[eventType]
	// 3. 返回 ch 和一个取消函数（cancel func() 从 subscribers 中移除并 close ch）
}

// Publish 发布事件
func (eb *EventBus) Publish(event Event) {
	// TODO:
	// 1. 加读锁，获取该 eventType 的所有 subscriber channel
	// 2. 遍历发送（注意：不要持有锁时发送！为什么？）
	// 3. 发送时加 select + default 防止某个 subscriber 阻塞导致 Publish 卡住
}

// Close 关闭总线，清理所有 subscriber
func (eb *EventBus) Close() {
	// TODO: 关闭所有 channel，清空 map
}

func main() {
	bus := NewEventBus()
	defer bus.Close()

	// 订阅 "user:login"
	ch1, cancel1 := bus.Subscribe("user:login")
	defer cancel1()

	// 另一个订阅者
	ch2, cancel2 := bus.Subscribe("user:login")
	defer cancel2()

	// 启动两个 goroutine 消费
	go func() {
		for e := range ch1 {
			fmt.Printf("[handler-1] %s: %v\n", e.Type, e.Data)
		}
	}()

	go func() {
		for e := range ch2 {
			fmt.Printf("[handler-2] %s: %v\n", e.Type, e.Data)
		}
	}()

	// 发布事件
	bus.Publish(Event{Type: "user:login", Data: "alice"})
	bus.Publish(Event{Type: "user:login", Data: "bob"})

	time.Sleep(1 * time.Second)
}

func NewEventBus() *EventBus {
	// TODO
	return nil
}
```

## 验收标准

- [ ] 多个订阅者能收到同一个事件
- [ ] `cancel()` 后，对应的 subscriber 不再收到事件，channel 被 close
- [ ] `Publish` 是非阻塞的，即使某个 subscriber 消费慢也不影响其他 subscriber
- [ ] 扩展：实现 `SubscribeAsync`，事件在 goroutine 里异步调用 handler（而不是通过 channel）

## 参考

- [Go by Example: Select](https://gobyexample.com/select)
- [Go by Example: Timeouts](https://gobyexample.com/timeouts)
- [Go by Example: Non-Blocking Channel Operations](https://gobyexample.com/non-blocking-channel-operations)

---

> 与 Java 对比：Java 的 EventBus（如 Guava）是方法回调模式。Go 里既可以用 channel（解耦、背压自然），也可以用函数回调（简单直接）。两种都实现一遍，感受差异。
