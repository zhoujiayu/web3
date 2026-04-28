package main

import (
	"fmt"
	"sync"
	"time"
)

// 定义事件实体
type Event struct {
	Type string
	Data any
}

// 定义事件总线
type EventBus struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
	closed      bool
}

// 构造方法，初始化EventBus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

// 关闭清理事件总线所有订阅

func (eventBus *EventBus) Close() {

	eventBus.mu.Lock()
	defer eventBus.mu.Unlock()
	if eventBus.closed {
		return
	}
	eventBus.closed = true
	for _, list := range eventBus.subscribers {
		for _, ch := range list {
			close(ch)
		}
	}
	eventBus.subscribers = make(map[string][]chan Event)
}

// 订阅
func (eventBus *EventBus) Subscribe(eventType string) (<-chan Event, func()) {

	// 初始化队列
	ch := make(chan Event, 10)
	eventBus.mu.Lock()
	if eventBus.closed {
		eventBus.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	eventBus.subscribers[eventType] = append(eventBus.subscribers[eventType], ch)
	eventBus.mu.Unlock()
	var once sync.Once
	cancle := func() {
		once.Do(func() {
			eventBus.mu.Lock()
			defer eventBus.mu.Unlock()
			if eventBus.closed {
				return
			}
			list := eventBus.subscribers[eventType]
			for i, subscriber := range list {
				if subscriber == ch {
					eventBus.subscribers[eventType] = append(list[:i], list[i+1:]...)
					if len(eventBus.subscribers[eventType]) == 0 {
						delete(eventBus.subscribers, eventType)
					}
					close(ch)
					return
				}
			}
		})
	}
	return ch, cancle
}

func (bus *EventBus) Publish(event Event) {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	if bus.closed { // 如果事件总线已经关闭，则忽略发布请求
		return // 直接返回，不向任何 channel 发送事件
	}
	for _, ch := range bus.subscribers[event.Type] {
		select {
		case ch <- event:
		default:
			fmt.Printf("[publish] skip slow subscriber for %s: %v\n", event.Type, event.Data) // 输出慢消费者被跳过的信息，证明发布不会阻塞
		}
	}
}

type Handler func(Event)

func (eb *EventBus) SubscribeAsync(eventType string, handler Handler) func() { // SubscribeAsync 用回调函数异步订阅指定事件类型
	ch, cancel := eb.Subscribe(eventType)
	go func() { // 启动 goroutine 异步消费 channel 中的事件
		for event := range ch { // 持续读取事件，直到 channel 被关闭
			handler(event) // 调用调用方传入的事件处理函数
		} // 结束 channel range 循环
	}() // 立即启动异步 goroutine
	return cancel // 返回取消函数
}

func main() {

	// 初始化事件总线对象
	bus := NewEventBus()
	// 关闭
	defer bus.Close()
	// ch1, cancel1 := bus.Subscribe("user:login") // 创建第一个 channel 订阅者，订阅 user:login 事件
	// defer cancel1()                             // main 退出前取消第一个订阅者，重复 Close 场景也不会 panic
	// go func() {                                 // 启动第一个 goroutine 消费 ch1 事件
	// 	for event := range ch1 { // 持续读取第一个订阅者 channel 中的事件
	// 		fmt.Printf("[handler-1] %s: %v\n", event.Type, event.Data) // 输出第一个订阅者收到的事件
	// 	} // 结束第一个订阅者 channel range 循环
	// }() // 立即启动第一个消费者 goroutine
	asyncDone := make(chan struct{}, 1)
	asyncCancel := bus.SubscribeAsync("user:login", func(event Event) {
		fmt.Printf("[async-handler] %s: %v\n", event.Type, event.Data) // 输出异步 handler 收到的事件
		select {

		case asyncDone <- struct{}{}:
		default:

		}
	})
	defer asyncCancel()
	bus.Publish(Event{Type: "user:login", Data: "alice"}) // 发布 alice 登录事件，此时两个 channel 订阅者和异步 handler 都应收到
	<-asyncDone                                           // 等待异步 handler 至少收到一个事件，确保异步订阅演示成立
	time.Sleep(100 * time.Millisecond)

}
