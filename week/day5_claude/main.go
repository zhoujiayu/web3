package main // 声明当前文件属于 main 包，便于直接运行示例程序。

import ( // 引入本示例需要的 Go 标准库。
	"context" // 引入 context，用于让 Get 支持超时和取消。
	"errors"  // 引入 errors，用于创建清晰的错误值。
	"fmt"     // 引入 fmt，用于格式化输入输出。
	"io"      // 引入 io，用于在 TCP 服务端回显数据。
	"net"     // 引入 net，用于 TCP 监听、拨号和连接抽象。
	"sync"    // 引入 sync，用于互斥锁和等待 goroutine 完成。
	"time"    // 引入 time，用于超时、暂停和清理 deadline。
) // 结束 import 声明。

type Pool struct { // 定义 Pool 结构体，用于管理可复用的 TCP 连接。
	factory func() (net.Conn, error) // 保存创建新连接的工厂函数。
	pool    chan net.Conn            // 使用带缓冲 channel 保存空闲连接。
	done    chan struct{}            // 使用关闭信号 channel 唤醒等待 Get 的 goroutine。
	maxSize int                      // 保存连接池允许的最大连接总数。
	mu      sync.Mutex               // 使用互斥锁保护 current 和 closed 等共享状态。
	current int                      // 记录当前总连接数，包含已借出连接和空闲连接。
	closed  bool                     // 记录连接池是否已经关闭。
} // 结束 Pool 结构体定义。

func NewPool(maxSize int, factory func() (net.Conn, error)) (*Pool, error) { // 定义 NewPool，用于创建并预热连接池。
	if maxSize <= 0 { // 校验最大连接数必须大于 0。
		return nil, errors.New("maxSize must be greater than 0") // 返回参数错误，避免创建无效连接池。
	} // 结束 maxSize 校验。
	if factory == nil { // 校验工厂函数不能为空。
		return nil, errors.New("factory must not be nil") // 返回参数错误，避免后续调用空函数 panic。
	} // 结束 factory 校验。
	p := &Pool{ // 初始化连接池实例。
		factory: factory,                      // 保存连接工厂函数。
		pool:    make(chan net.Conn, maxSize), // 创建容量为 maxSize 的空闲连接 channel。
		done:    make(chan struct{}),          // 创建关闭信号 channel，用于广播连接池关闭事件。
		maxSize: maxSize,                      // 保存最大连接数。
	} // 结束连接池实例初始化。
	initialSize := minInt(maxSize, 2)  // 计算预创建连接数量，最多预创建 2 个。
	for i := 0; i < initialSize; i++ { // 循环预创建初始连接。
		conn, err := factory() // 调用工厂函数创建 TCP 连接。
		if err != nil {        // 判断创建连接是否失败。
			p.Close()                                                    // 关闭已经创建的空闲连接，避免资源泄漏。
			return nil, fmt.Errorf("create initial connection: %w", err) // 包装并返回初始化失败错误。
		} // 结束错误判断。
		p.pool <- conn // 将预创建连接放入空闲连接池。
		p.current++    // 增加当前总连接数计数。
	} // 结束预创建循环。
	return p, nil // 返回创建成功的连接池。
} // 结束 NewPool 函数。

func (p *Pool) Get(ctx context.Context) (net.Conn, error) { // 定义 Get 方法，用于获取一个可用连接并支持 context 超时。
	if ctx == nil { // 判断调用方是否传入 nil context。
		ctx = context.Background() // 使用 Background 作为默认 context，避免 select 监听 nil。
	} // 结束 nil context 处理。
	p.mu.Lock()   // 加锁同步 closed 状态，避免关闭后快速路径借出空闲连接。
	if p.closed { // 判断连接池是否已经关闭。
		p.mu.Unlock()                            // 解锁后返回错误。
		return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
	} // 结束关闭状态判断。
	p.mu.Unlock() // 解锁后尝试快速获取空闲连接。
	select {      // 先尝试无阻塞获取空闲连接或关闭信号。
	case <-p.done: // 如果连接池已经关闭。
		return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
	case conn := <-p.pool: // 如果存在空闲连接则立即取出。
		p.mu.Lock()   // 取到空闲连接后再次同步 closed 状态。
		if p.closed { // 如果连接池在取连接期间被关闭。
			if p.current > 0 { // 防御性判断，避免 current 递减到负数。
				p.current-- // 关闭这个已取出的空闲连接后减少当前总连接数。
			} // 结束防御性判断。
			p.mu.Unlock()                            // 解锁后关闭连接并返回错误。
			_ = conn.Close()                         // 关闭关闭期间取出的空闲连接。
			return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
		} // 结束关闭状态复查。
		p.mu.Unlock()    // 连接池仍然打开，解锁后返回连接。
		return conn, nil // 返回取到的空闲连接。
	default: // 如果当前没有空闲连接则继续后续逻辑。
	} // 结束快速获取空闲连接。
	p.mu.Lock()   // 加锁检查和更新连接池状态。
	if p.closed { // 判断连接池是否已经关闭。
		p.mu.Unlock()                            // 解锁后返回错误。
		return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
	} // 结束关闭状态判断。
	if p.current < p.maxSize { // 如果当前总连接数还没有达到最大值。
		p.current++   // 先占用一个连接名额，避免并发创建超过 maxSize。
		p.mu.Unlock() // 解锁后执行可能较慢的网络拨号。
		select {      // 动态创建前快速感知关闭信号。
		case <-p.done: // 如果连接池在解锁后已经关闭。
			p.mu.Lock()        // 加锁修正刚才占用的连接名额。
			if p.current > 0 { // 防御性判断，避免 current 递减到负数。
				p.current-- // 释放未使用的连接名额。
			} // 结束防御性判断。
			p.mu.Unlock()                            // 解锁完成计数修正。
			return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
		default: // 如果连接池尚未关闭。
		} // 结束动态创建前关闭信号检查。
		conn, err := p.factory() // 调用工厂函数动态创建新连接。
		if err != nil {          // 判断动态创建是否失败。
			p.mu.Lock()                                          // 加锁修正 current 计数。
			p.current--                                          // 创建失败时释放刚才占用的连接名额。
			p.mu.Unlock()                                        // 解锁完成计数修正。
			return nil, fmt.Errorf("create connection: %w", err) // 返回创建连接失败错误。
		} // 结束创建失败处理。
		p.mu.Lock()   // 动态创建完成后重新同步 closed 状态。
		if p.closed { // 如果连接池在创建连接期间被关闭。
			if p.current > 0 { // 防御性判断，避免 current 递减到负数。
				p.current-- // 关闭这个新建连接后减少当前总连接数。
			} // 结束防御性判断。
			p.mu.Unlock()                            // 解锁后关闭连接并返回错误。
			_ = conn.Close()                         // 关闭关闭期间创建出的连接。
			return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
		} // 结束关闭状态复查。
		p.mu.Unlock()    // 连接池仍然打开，解锁后返回连接。
		return conn, nil // 返回动态创建的新连接。
	} // 结束动态创建分支。
	p.mu.Unlock() // 达到 maxSize 时解锁并进入等待。
	select {      // 等待其他 goroutine 归还连接、context 结束或连接池关闭。
	case <-p.done: // 如果连接池关闭信号到达。
		return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
	case conn := <-p.pool: // 如果有连接归还到空闲池。
		p.mu.Lock()   // 取到归还连接后同步 closed 状态。
		if p.closed { // 如果连接池在等待期间被关闭。
			if p.current > 0 { // 防御性判断，避免 current 递减到负数。
				p.current-- // 关闭这个等待到的连接后减少当前总连接数。
			} // 结束防御性判断。
			p.mu.Unlock()                            // 解锁后关闭连接并返回错误。
			_ = conn.Close()                         // 关闭关闭期间等待到的连接。
			return nil, errors.New("pool is closed") // 返回连接池已关闭错误。
		} // 结束关闭状态复查。
		p.mu.Unlock()    // 连接池仍然打开，解锁后返回连接。
		return conn, nil // 返回等待到的连接。
	case <-ctx.Done(): // 如果 context 超时或取消。
		return nil, ctx.Err() // 返回 context 的具体错误。
	} // 结束等待逻辑。
} // 结束 Get 方法。

func (p *Pool) Put(conn net.Conn) { // 定义 Put 方法，仅用于归还从本池 Get 得到且尚未归还过的连接。
	if conn == nil { // 忽略 nil 连接，提升调用容错性。
		return // 直接返回，不修改连接池状态。
	} // 结束 nil 判断。
	if err := conn.SetDeadline(time.Time{}); err != nil { // 清理连接 deadline，并检查连接是否还能复用。
		_ = conn.Close()   // 清理 deadline 失败时关闭连接，避免坏连接回到池中。
		p.mu.Lock()        // 加锁修正连接总数。
		if p.current > 0 { // 防御性判断，避免 current 递减到负数。
			p.current-- // 关闭坏连接后减少当前总连接数。
		} // 结束防御性判断。
		p.mu.Unlock() // 解锁完成计数修正。
		return        // 结束归还逻辑，不把坏连接放回池。
	} // 结束 deadline 清理错误处理。
	p.mu.Lock()   // 加锁保护 closed 检查、归还发送和 current 修正，避免 Put 与 Close 的 TOCTOU 竞态。
	if p.closed { // 如果连接池已经关闭。
		if p.current > 0 { // 防御性判断，避免 current 递减到负数。
			p.current-- // 关闭归还连接后减少当前总连接数。
		} // 结束防御性判断。
		p.mu.Unlock()    // 解锁完成关闭状态处理，不允许连接重新进入已关闭连接池。
		_ = conn.Close() // 关闭归还的连接，避免关闭后继续复用。
		return           // 结束归还逻辑。
	} // 结束关闭状态判断。
	select { // 在持有同一把锁时非阻塞尝试归还到空闲池，避免 Close 插入到检查和发送之间。
	case p.pool <- conn: // 如果空闲池还有容量。
		p.mu.Unlock() // 归还成功后解锁。
		return        // 归还成功后直接返回。
	default: // 如果空闲池已满。
		if p.current > 0 { // 防御性判断，避免 current 递减到负数。
			p.current-- // 池满丢弃连接时减少当前总连接数。
		} // 结束防御性判断。
		p.mu.Unlock()    // 解锁完成池满状态处理。
		_ = conn.Close() // 关闭多余连接，避免连接泄漏。
	} // 结束非阻塞归还逻辑。
} // 结束 Put 方法。

func (p *Pool) Close() { // 定义 Close 方法，用于幂等关闭连接池。
	p.mu.Lock()   // 加锁保护 closed 状态。
	if p.closed { // 判断连接池是否已经关闭过。
		p.mu.Unlock() // 已关闭则解锁。
		return        // 幂等返回，避免重复关闭。
	} // 结束重复关闭判断。
	p.closed = true // 标记连接池已经关闭，阻止后续复用。
	close(p.done)   // 首次关闭时广播关闭信号，唤醒所有正在 Get 中等待的 goroutine。
	p.mu.Unlock()   // 解锁后开始清理空闲连接。
	for {           // 循环 drain 当前空闲连接，不关闭 channel 以避免 Put panic。
		select { // 非阻塞读取空闲连接。
		case conn := <-p.pool: // 如果还能取到空闲连接。
			_ = conn.Close()   // 关闭空闲连接。
			p.mu.Lock()        // 加锁修正连接总数。
			if p.current > 0 { // 防御性判断，避免 current 变成负数。
				p.current-- // 关闭一个空闲连接后减少当前总连接数。
			} // 结束防御性判断。
			p.mu.Unlock() // 解锁完成计数修正。
		default: // 如果空闲池已经没有连接。
			return // 清理完成后返回。
		} // 结束 select。
	} // 结束 drain 循环。
} // 结束 Close 方法。

func (p *Pool) Stats() (idle, active, total int) { // 定义 Stats 方法，用于返回连接池状态。
	p.mu.Lock()           // 加锁读取 current，避免数据竞争。
	defer p.mu.Unlock()   // 函数返回前自动解锁。
	idle = len(p.pool)    // 读取空闲连接数量。
	total = p.current     // 读取当前连接总数。
	active = total - idle // 计算已借出连接数量。
	if active < 0 {       // 防御性处理误用导致的异常统计值。
		active = 0 // 将活跃连接数下限限制为 0。
	} // 结束活跃连接数下限处理。
	return idle, active, total // 返回空闲、活跃和总连接数。
} // 结束 Stats 方法。

func minInt(a int, b int) int { // 定义 minInt 辅助函数，用于兼容不同 Go 版本。
	if a < b { // 判断 a 是否更小。
		return a // 返回较小的 a。
	} // 结束判断。
	return b // 返回较小或相等情况下的 b。
} // 结束 minInt 函数。

func startEchoServer() (net.Listener, error) { // 定义 startEchoServer，用于启动本地 TCP 回显服务。
	listener, err := net.Listen("tcp", "127.0.0.1:0") // 监听本地随机端口，避免依赖外网。
	if err != nil {                                   // 判断监听是否失败。
		return nil, err // 返回监听错误。
	} // 结束监听错误判断。
	go func() { // 启动后台 goroutine 接受客户端连接。
		for { // 循环接受多个 TCP 连接。
			conn, err := listener.Accept() // 阻塞等待新连接。
			if err != nil {                // 判断 Accept 是否失败。
				return // listener 关闭后正常退出服务 goroutine。
			} // 结束 Accept 错误判断。
			go handleEchoConn(conn) // 为每个连接启动独立 goroutine 处理读写。
		} // 结束 Accept 循环。
	}() // 立即启动服务端 goroutine。
	return listener, nil // 返回监听器给 main 关闭。
} // 结束 startEchoServer 函数。

func handleEchoConn(conn net.Conn) { // 定义 handleEchoConn，用于处理单个客户端连接。
	defer conn.Close()        // 函数退出时关闭服务端连接。
	buf := make([]byte, 1024) // 创建读缓冲区。
	for {                     // 循环读取并回显客户端数据。
		n, err := conn.Read(buf) // 从连接读取数据。
		if err != nil {          // 判断读取是否失败。
			if !errors.Is(err, io.EOF) { // 区分普通 EOF 和其他错误。
				return // 非 EOF 错误时结束连接处理。
			} // 结束非 EOF 判断。
			return // EOF 时结束连接处理。
		} // 结束读取错误判断。
		_, err = fmt.Fprintf(conn, "echo: %s", string(buf[:n])) // 将读取到的数据带前缀写回客户端。
		if err != nil {                                         // 判断写回是否失败。
			return // 写失败时结束连接处理。
		} // 结束写回错误判断。
	} // 结束读写循环。
} // 结束 handleEchoConn 函数。

func main() { // 定义 main 函数，运行连接池并发示例。
	listener, err := startEchoServer() // 启动本地 TCP 回显服务。
	if err != nil {                    // 判断服务启动是否失败。
		panic(err) // 示例程序中直接 panic 暴露错误。
	} // 结束服务启动错误判断。
	defer listener.Close()                // main 退出时关闭监听器。
	addr := listener.Addr().String()      // 获取本地服务监听地址。
	factory := func() (net.Conn, error) { // 定义连接工厂函数。
		return net.DialTimeout("tcp", addr, 2*time.Second) // 使用超时拨号连接本地 TCP 服务。
	} // 结束连接工厂函数定义。
	pool, err := NewPool(3, factory) // 创建最大连接数为 3 的连接池。
	if err != nil {                  // 判断连接池创建是否失败。
		panic(err) // 示例程序中直接 panic 暴露错误。
	} // 结束连接池创建错误判断。
	defer pool.Close()        // main 退出时关闭连接池。
	var wg sync.WaitGroup     // 定义 WaitGroup 等待 10 个 goroutine 完成。
	for i := 0; i < 10; i++ { // 启动 10 个并发任务。
		wg.Add(1)         // 为当前任务增加等待计数。
		go func(id int) { // 启动 goroutine 执行一次借用连接、写入、读取和归还。
			defer wg.Done()                                                         // goroutine 结束时减少等待计数。
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // 创建 3 秒超时 context。
			defer cancel()                                                          // goroutine 结束时释放 context 资源。
			conn, err := pool.Get(ctx)                                              // 从连接池获取连接。
			if err != nil {                                                         // 判断获取连接是否失败。
				fmt.Printf("goroutine %02d get failed: %v\n", id, err) // 打印获取失败信息。
				return                                                 // 获取失败时结束当前 goroutine。
			} // 结束获取错误判断。
			defer pool.Put(conn)                          // goroutine 结束时将连接归还连接池。
			message := fmt.Sprintf("hello from %02d", id) // 构造发送给 TCP 服务的消息。
			_, err = fmt.Fprintln(conn, message)          // 将消息写入 TCP 连接。
			if err != nil {                               // 判断写入是否失败。
				fmt.Printf("goroutine %02d write failed: %v\n", id, err) // 打印写入失败信息。
				return                                                   // 写入失败时结束当前 goroutine。
			} // 结束写入错误判断。
			buf := make([]byte, 1024) // 创建读取响应的缓冲区。
			n, err := conn.Read(buf)  // 从 TCP 连接读取服务端回显。
			if err != nil {           // 判断读取是否失败。
				fmt.Printf("goroutine %02d read failed: %v\n", id, err) // 打印读取失败信息。
				return                                                  // 读取失败时结束当前 goroutine。
			} // 结束读取错误判断。
			idle, active, total := pool.Stats()                                                                           // 获取当前连接池状态。
			fmt.Printf("goroutine %02d read %q | idle=%d active=%d total=%d\n", id, string(buf[:n]), idle, active, total) // 打印读取结果和连接池统计，观察 total 不超过 3。
		}(i) // 传入循环变量副本，避免闭包捕获问题。
	} // 结束 goroutine 启动循环。
	wg.Wait()                                                                                  // 等待所有 goroutine 完成。
	idle, active, total := pool.Stats()                                                        // 获取最终连接池状态。
	fmt.Printf("final stats: idle=%d active=%d total=%d maxSize=%d\n", idle, active, total, 3) // 打印最终统计，证明 total 不超过 maxSize。
} // 结束 main 函数。
