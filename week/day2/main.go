package main

import (
	"fmt"
)

// 定义scope 类型
type Scope int

// 定义常量
const (
	Singleton Scope = iota
	Prototype
)

// 定义 Provider 函数类型，用于创建任意类型实例。
type Provider func() any

type registration struct {
	provider Provider
	scope    Scope
}

type Container struct {
	providers  map[string]registration
	singletons map[string]any
}

// Container 构造方法
func NewContainer() *Container {
	return &Container{
		providers:  make(map[string]registration),
		singletons: make(map[string]any),
	}
}

func (c *Container) Register(name string, provider Provider, scope Scope) {
	c.providers[name] = registration{provider: provider, scope: scope}
}

func (c *Container) Get(name string) any {

	reg, ok := c.providers[name]
	if !ok {
		panic(fmt.Sprintf("provider %q is not registered", name)) // 未注册时抛出包含名称的明确错误。

	}

	if reg.scope == Prototype {
		return reg.provider()
	}

	if reg.scope != Singleton {
		panic(fmt.Sprintf("provider %q has unknown scope %d", name, reg.scope)) // 未知作用域时抛出明确错误。
	}

	instance, ok := c.singletons[name]
	if ok {
		return instance
	}
	instance = reg.provider()
	c.singletons[name] = instance
	return instance
}

func MustGet[T any](c *Container, name string) T {
	value := c.Get(name)
	typed, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("provider %q has type %T, not requested type", name, value)) // 类型不匹配时抛出包含名称和实际类型的明确错误。

	}
	return typed
}

type Logger struct {
	name string
}

type UserService struct {
	Logger *Logger
}

func main() {

	c := NewContainer()
	counter := 0
	c.Register("singletonCounter", func() any {
		counter++
		return counter
	}, Singleton)
	singletonCounter1 := c.Get("singletonCounter")
	singletonCounter2 := c.Get("singletonCounter")
	fmt.Println("singleton counter:", singletonCounter1, singletonCounter2) // 打印单例计数器结果，应为同一值。
	c.Register("prototypeCounter", func() any {
		counter++      // 每次提供者被调用时递增计数器。
		return counter // 返回当前计数器值。
	}, Prototype)
	prototypeCounter1 := c.Get("prototypeCounter")
	prototypeCounter2 := c.Get("prototypeCounter")
	fmt.Println("prototype counter:", prototypeCounter1, prototypeCounter2) // 打印单例计数器结果，应为同一值。

	c.Register("logger", func() any {
		return &Logger{name: "app"}
	}, Singleton)

	c.Register("userService", func() any {
		logger := MustGet[*Logger](c, "logger")
		return &UserService{Logger: logger}
	}, Prototype)
	logger1 := MustGet[*Logger](c, "logger")            // 第一次获取日志服务。
	logger2 := MustGet[*Logger](c, "logger")            // 第二次获取日志服务。
	service1 := MustGet[*UserService](c, "userService") // 第一次获取用户服务。
	service2 := MustGet[*UserService](c, "userService") // 第二次获取用户服务。
	fmt.Println("same logger:", logger1 == logger2)     // 打印两次日志服务是否为同一实例，应为 true。
	fmt.Println("same service:", service1 == service2)  // 打印两次用户服务是否为同一实例，应为 false。
}
