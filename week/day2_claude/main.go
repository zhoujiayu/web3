package main // 声明当前文件属于 main 包。

import "fmt" // 导入 fmt 标准库用于格式化输出。

type Scope int // 定义 Scope 整型类型表示对象作用域。

const ( // 定义作用域常量分组。
	Singleton Scope = iota // Singleton 表示单例作用域，同名服务只创建一次。
	Prototype              // Prototype 表示原型作用域，每次获取都重新创建。
) // 结束作用域常量分组。

type Provider func() interface{} // 定义 Provider 函数类型，用于创建任意类型实例。

type registration struct { // 定义 registration 结构体保存注册信息。
	provider Provider // 保存实例提供函数。
	scope    Scope    // 保存实例作用域。
} // 结束 registration 结构体定义。

type Container struct { // 定义 Container 结构体作为 DI 容器。
	providers  map[string]registration // providers 按名称保存服务注册信息。
	singletons map[string]interface{}  // singletons 按名称缓存单例实例。
} // 结束 Container 结构体定义。

func NewContainer() *Container { // 定义 NewContainer 函数创建并初始化容器。
	return &Container{ // 返回容器指针并初始化内部 map。
		providers:  make(map[string]registration), // 初始化注册信息 map。
		singletons: make(map[string]interface{}),  // 初始化单例缓存 map。
	} // 结束容器字面量。
} // 结束 NewContainer 函数。

func (c *Container) Register(name string, provider Provider, scope Scope) { // 定义 Register 方法按名称注册提供者和作用域。
	c.providers[name] = registration{provider: provider, scope: scope} // 将提供者和作用域保存到注册信息 map。
} // 结束 Register 方法。

func (c *Container) Get(name string) interface{} { // 定义 Get 方法按名称获取实例。
	reg, ok := c.providers[name] // 从注册信息 map 中查找指定名称。
	if !ok {                     // 判断指定名称是否未注册。
		panic(fmt.Sprintf("provider %q is not registered", name)) // 未注册时抛出包含名称的明确错误。
	} // 结束未注册判断。
	if reg.scope == Prototype { // 判断注册作用域是否为 Prototype。
		return reg.provider() // Prototype 每次直接调用提供者创建新实例。
	} // 结束 Prototype 判断。
	if reg.scope != Singleton { // 判断注册作用域是否不是已支持的 Singleton。
		panic(fmt.Sprintf("provider %q has unknown scope %d", name, reg.scope)) // 未知作用域时抛出明确错误。
	} // 结束未知作用域判断。
	instance, ok := c.singletons[name] // 从单例缓存中查找已有实例。
	if ok {                            // 判断单例实例是否已经存在。
		return instance // 已存在则直接返回缓存实例。
	} // 结束单例缓存命中判断。
	instance = reg.provider()     // 缓存未命中时调用提供者创建实例。
	c.singletons[name] = instance // 将新创建的实例保存到单例缓存。
	return instance               // 返回新创建的单例实例。
} // 结束 Get 方法。

func MustGet[T any](c *Container, name string) T { // 定义 MustGet 泛型函数获取指定类型实例。
	value := c.Get(name)   // 先从容器中按名称获取原始实例。
	typed, ok := value.(T) // 使用类型断言将原始实例转换为目标类型。
	if !ok {               // 判断类型断言是否失败。
		panic(fmt.Sprintf("provider %q has type %T, not requested type", name, value)) // 类型不匹配时抛出包含名称和实际类型的明确错误。
	} // 结束类型断言失败判断。
	return typed // 返回类型断言成功后的实例。
} // 结束 MustGet 函数。

type Logger struct { // 定义 Logger 结构体表示日志服务。
	Name string // Name 字段保存日志服务名称。
} // 结束 Logger 结构体定义。

type UserService struct { // 定义 UserService 结构体表示用户服务。
	Logger *Logger // Logger 字段保存注入的日志服务依赖。
} // 结束 UserService 结构体定义。

func main() { // 定义程序入口函数。
	c := NewContainer()                                 // 创建一个新的 DI 容器。
	counter := 0                                        // 定义计数器变量用于演示作用域行为。
	c.Register("singletonCounter", func() interface{} { // 注册单例计数器提供者。
		counter++      // 每次提供者被调用时递增计数器。
		return counter // 返回当前计数器值。
	}, Singleton) // 指定计数器为 Singleton 作用域。
	singletonCounter1 := c.Get("singletonCounter")                          // 第一次获取单例计数器。
	singletonCounter2 := c.Get("singletonCounter")                          // 第二次获取单例计数器。
	fmt.Println("singleton counter:", singletonCounter1, singletonCounter2) // 打印单例计数器结果，应为同一值。
	c.Register("prototypeCounter", func() interface{} {                     // 注册原型计数器提供者。
		counter++      // 每次提供者被调用时递增计数器。
		return counter // 返回当前计数器值。
	}, Prototype) // 指定计数器为 Prototype 作用域。
	prototypeCounter1 := c.Get("prototypeCounter")                          // 第一次获取原型计数器。
	prototypeCounter2 := c.Get("prototypeCounter")                          // 第二次获取原型计数器。
	fmt.Println("prototype counter:", prototypeCounter1, prototypeCounter2) // 打印原型计数器结果，应为不同值。
	c.Register("logger", func() interface{} {                               // 注册日志服务提供者。
		return &Logger{Name: "app"} // 创建并返回日志服务指针。
	}, Singleton) // 指定日志服务为 Singleton 作用域。
	c.Register("userService", func() interface{} { // 注册用户服务提供者。
		logger := MustGet[*Logger](c, "logger") // 通过 MustGet 获取并注入日志服务依赖。
		return &UserService{Logger: logger}     // 创建并返回带依赖的用户服务指针。
	}, Prototype) // 指定用户服务为 Prototype 作用域。
	logger1 := MustGet[*Logger](c, "logger")            // 第一次获取日志服务。
	logger2 := MustGet[*Logger](c, "logger")            // 第二次获取日志服务。
	service1 := MustGet[*UserService](c, "userService") // 第一次获取用户服务。
	service2 := MustGet[*UserService](c, "userService") // 第二次获取用户服务。
	fmt.Println("same logger:", logger1 == logger2)     // 打印两次日志服务是否为同一实例，应为 true。
	fmt.Println("same service:", service1 == service2)  // 打印两次用户服务是否为同一实例，应为 false。
} // 结束 main 函数。
