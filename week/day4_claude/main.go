package main // 声明当前文件属于可执行程序入口包

import ( // 引入程序需要使用的标准库包
	"fmt"  // 引入 fmt 用于格式化输出演示结果
	"sync" // 引入 sync 用于互斥锁和一次性执行控制
	"time" // 引入 time 用于在演示中等待 goroutine 输出
) // 结束 import 声明

type Event struct { // 定义事件结构体，用来承载事件类型和数据
	Type string      // Type 表示事件类型，例如 user:login
	Data interface{} // Data 表示事件携带的任意数据
} // 结束 Event 结构体定义

type Handler func(Event) // 定义 Handler 函数类型，用于异步订阅的回调处理

type EventBus struct { // 定义 EventBus 事件总线结构体
	subscribers map[string][]chan Event // subscribers 保存每种事件类型对应的订阅者 channel 列表
	mu          sync.RWMutex            // mu 保护 subscribers 和 closed 字段的并发访问
	closed      bool                    // closed 表示事件总线是否已经关闭
} // 结束 EventBus 结构体定义

func NewEventBus() *EventBus { // NewEventBus 创建并初始化一个新的事件总线
	return &EventBus{subscribers: make(map[string][]chan Event)} // 返回带有空订阅者 map 的 EventBus 指针
} // 结束 NewEventBus 函数

func (eb *EventBus) Subscribe(eventType string) (<-chan Event, func()) { // Subscribe 订阅指定事件类型并返回只读 channel 和取消函数
	ch := make(chan Event, 10) // 创建带缓冲的订阅者 channel，避免短时间内发布者被消费者阻塞
	eb.mu.Lock()               // 加写锁，准备修改订阅者列表
	if eb.closed {             // 如果事件总线已经关闭，则不能再加入订阅者
		eb.mu.Unlock()       // 释放写锁，避免死锁
		close(ch)            // 关闭新建 channel，让调用方能够正常 range 退出
		return ch, func() {} // 返回空取消函数，保证调用 cancel 不会 panic
	} // 结束关闭状态判断
	eb.subscribers[eventType] = append(eb.subscribers[eventType], ch) // 将订阅者 channel 加入对应事件类型的列表
	eb.mu.Unlock()                                                    // 修改完成后释放写锁
	var once sync.Once                                                // 使用 sync.Once 保证 cancel 重复调用时只执行一次关闭逻辑
	cancel := func() {                                                // 定义取消订阅函数
		once.Do(func() { // 确保删除和关闭 channel 的逻辑最多执行一次
			eb.mu.Lock()         // 加写锁，准备从订阅者列表移除 channel
			defer eb.mu.Unlock() // 函数返回前释放写锁
			if eb.closed {       // 如果事件总线已经关闭，Close 已经负责关闭所有 channel
				return // 直接返回，避免重复 close channel 导致 panic
			} // 结束关闭状态判断
			list := eb.subscribers[eventType] // 取出当前事件类型的订阅者列表
			for i, subscriber := range list { // 遍历订阅者列表，寻找当前 channel
				if subscriber == ch { // 判断是否找到了需要取消的订阅者 channel
					eb.subscribers[eventType] = append(list[:i], list[i+1:]...) // 从切片中移除当前订阅者 channel
					if len(eb.subscribers[eventType]) == 0 {                    // 如果该事件类型已经没有订阅者
						delete(eb.subscribers, eventType) // 从 map 中删除空列表，保持数据结构整洁
					} // 结束空列表判断
					close(ch) // 关闭订阅者 channel，通知消费者退出
					return    // 移除并关闭后结束 cancel 逻辑
				} // 结束 channel 匹配判断
			} // 结束订阅者列表遍历
		}) // 结束 sync.Once 包裹的函数
	} // 结束 cancel 函数定义
	return ch, cancel // 返回只读订阅 channel 和取消函数
} // 结束 Subscribe 方法

func (eb *EventBus) Publish(event Event) { // Publish 将事件发布给所有订阅了该类型的订阅者
	eb.mu.RLock()         // 加读锁，阻止取消订阅和关闭在发送期间并发关闭 channel
	defer eb.mu.RUnlock() // 函数返回前释放读锁，确保所有非阻塞发送完成后再允许写锁操作
	if eb.closed {        // 如果事件总线已经关闭，则忽略发布请求
		return // 直接返回，不向任何 channel 发送事件
	} // 结束关闭状态判断
	for _, ch := range eb.subscribers[event.Type] { // 在读锁保护下遍历当前事件类型的订阅者 channel 列表
		select { // 使用 select 实现非阻塞发送，避免慢消费者卡住发布者
		case ch <- event: // 如果订阅者 channel 有缓冲空间，则发送事件
		default: // 如果订阅者 channel 已满，则走默认分支
			fmt.Printf("[publish] skip slow subscriber for %s: %v\n", event.Type, event.Data) // 输出慢消费者被跳过的信息，证明发布不会阻塞
		} // 结束 select 语句
	} // 结束订阅者遍历
} // 结束 Publish 方法

func (eb *EventBus) Close() { // Close 关闭事件总线并清理所有订阅者
	eb.mu.Lock()         // 加写锁，准备关闭和清理共享状态
	defer eb.mu.Unlock() // 函数返回前释放写锁
	if eb.closed {       // 如果已经关闭过事件总线
		return // 直接返回，保证重复 Close 不会 panic
	} // 结束重复关闭判断
	eb.closed = true                      // 标记事件总线已关闭，阻止后续订阅和发布
	for _, list := range eb.subscribers { // 遍历所有事件类型的订阅者列表
		for _, ch := range list { // 遍历当前事件类型下的所有订阅者 channel
			close(ch) // 关闭订阅者 channel，通知消费者退出
		} // 结束当前订阅者列表遍历
	} // 结束所有订阅者遍历
	eb.subscribers = make(map[string][]chan Event) // 清空订阅者 map，释放引用并保持对象可重复安全使用
} // 结束 Close 方法

func (eb *EventBus) SubscribeAsync(eventType string, handler Handler) func() { // SubscribeAsync 用回调函数异步订阅指定事件类型
	ch, cancel := eb.Subscribe(eventType) // 复用 Subscribe 创建 channel 订阅并获得取消函数
	go func() {                           // 启动 goroutine 异步消费 channel 中的事件
		for event := range ch { // 持续读取事件，直到 channel 被关闭
			handler(event) // 调用调用方传入的事件处理函数
		} // 结束 channel range 循环
	}() // 立即启动异步 goroutine
	return cancel // 返回取消函数，让调用方可以停止异步订阅
} // 结束 SubscribeAsync 方法

func main() { // main 函数演示 EventBus 的订阅、发布、取消和异步回调
	bus := NewEventBus()                                                // 创建一个新的事件总线实例
	defer bus.Close()                                                   // 程序结束时关闭事件总线，确保所有 channel 被清理
	ch1, cancel1 := bus.Subscribe("user:login")                         // 创建第一个 channel 订阅者，订阅 user:login 事件
	defer cancel1()                                                     // main 退出前取消第一个订阅者，重复 Close 场景也不会 panic
	ch2, cancel2 := bus.Subscribe("user:login")                         // 创建第二个 channel 订阅者，订阅同一个事件类型
	asyncDone := make(chan struct{}, 1)                                 // 创建完成信号 channel，用于证明异步 handler 至少收到一个事件
	asyncCancel := bus.SubscribeAsync("user:login", func(event Event) { // 创建异步回调订阅者并定义处理逻辑
		fmt.Printf("[async-handler] %s: %v\n", event.Type, event.Data) // 输出异步 handler 收到的事件
		select {                                                       // 使用非阻塞发送完成信号，避免多次事件导致阻塞
		case asyncDone <- struct{}{}: // 第一次收到事件时发送完成信号
		default: // 如果完成信号已经发送过，则不再发送
		} // 结束 select 语句
	}) // 结束异步订阅调用
	defer asyncCancel() // main 退出前取消异步订阅，确保 goroutine 可以退出
	go func() {         // 启动第一个 goroutine 消费 ch1 事件
		for event := range ch1 { // 持续读取第一个订阅者 channel 中的事件
			fmt.Printf("[handler-1] %s: %v\n", event.Type, event.Data) // 输出第一个订阅者收到的事件
		} // 结束第一个订阅者 channel range 循环
	}() // 立即启动第一个消费者 goroutine
	go func() { // 启动第二个 goroutine 消费 ch2 事件
		for event := range ch2 { // 持续读取第二个订阅者 channel 中的事件
			fmt.Printf("[handler-2] %s: %v\n", event.Type, event.Data) // 输出第二个订阅者收到的事件
		} // 结束第二个订阅者 channel range 循环
	}() // 立即启动第二个消费者 goroutine
	bus.Publish(Event{Type: "user:login", Data: "alice"}) // 发布 alice 登录事件，此时两个 channel 订阅者和异步 handler 都应收到
	time.Sleep(100 * time.Millisecond)                    // 短暂等待 goroutine 输出 alice 事件，便于演示顺序清晰
	cancel2()                                             // 取消第二个 channel 订阅者，后续 bob 不应再被 handler-2 收到
	fmt.Println("[main] canceled handler-2")              // 输出取消第二个订阅者的提示信息
	bus.Publish(Event{Type: "user:login", Data: "bob"})   // 发布 bob 登录事件，此时只有 handler-1 和 async-handler 应收到
	<-asyncDone                                           // 等待异步 handler 至少收到一个事件，确保异步订阅演示成立
	time.Sleep(100 * time.Millisecond)                    // 短暂等待剩余 goroutine 输出完成，保证程序正常退出前日志完整
} // 结束 main 函数
