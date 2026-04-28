# Week 1 Day 2：依赖注入容器

## 目标

手写一个极简的 DI 容器：
- 支持按名字注册和获取实例
- 支持单例（Singleton）和每次新建（Prototype）
- 理解 `interface{}`、类型断言、闭包

## Java > Go 映射

| Java | Go |
|------|-----|
| `Map<String, Object>` | `map[string]interface{}` |
| `T obj = (T) map.get("key")` | `obj := container.Get("key").(SomeType)` |
| `Supplier<T>` | `func() interface{}`（函数是一等公民） |
| Spring `@Singleton` / `@Prototype` | 自己用 enum/const 区分作用域 |

## 需要掌握的知识点

1. **`interface{}`**：空接口，类似 Java 的 `Object`，所有类型都实现它
2. **类型断言**：`v := obj.(Type)`，转换失败会 panic；`v, ok := obj.(Type)` 安全转换
3. **函数类型**：`type Provider func() interface{}`，函数可以作为参数和返回值
4. **闭包**：工厂函数返回一个闭包，闭包捕获外部变量

## 代码框架

```go
package main

import "fmt"

// Scope 作用域类型
type Scope int

const (
	Singleton Scope = iota // 单例
	Prototype              // 每次新建
)

// Container DI 容器
type Container struct {
	// TODO: 两个 map
	// 1. providers: name > Provider 函数
	// 2. singletons: name > 单例实例（缓存）
}

// Provider 实例提供者
type Provider func() interface{}

// NewContainer 创建容器
func NewContainer() *Container {
	// TODO
}

// Register 注册提供者
func (c *Container) Register(name string, provider Provider, scope Scope) {
	// TODO
}

// Get 获取实例
func (c *Container) Get(name string) interface{} {
	// TODO:
	// Singleton: 先查缓存，没有再创建并缓存
	// Prototype: 直接调用 provider
}

// MustGet 类型安全获取（返回指定类型，失败 panic）
func MustGet[T any](c *Container, name string) T {
	// TODO: 使用类型断言 v.(T)
	// 注意：Go 1.18+ 才支持泛型，这里练习类型断言
	var zero T
	return zero
}

func main() {
	c := NewContainer()

	// 注册一个单例计数器
	counter := 0
	c.Register("counter", func() interface{} {
		counter++
		return counter
	}, Singleton)

	// 获取两次，应该是同一个值
	v1 := c.Get("counter")
	v2 := c.Get("counter")
	fmt.Println(v1, v2) // 期望: 1 1

	// TODO: 注册一个 Prototype，验证每次获取都是新实例
}
```

## 验收标准

- [ ] Singleton 作用域：多次获取返回同一个实例
- [ ] Prototype 作用域：每次获取都调用 provider 创建新实例
- [ ] `MustGet` 能正确做类型断言，类型不匹配时给出明确 panic 信息
- [ ] 扩展练习：支持构造函数注入（`Register("a", func() interface{} { return &ServiceA{} }, Singleton)`，然后在另一个 provider 里 `c.Get("a")`）

## 参考

- [Go by Example: Interfaces](https://gobyexample.com/interfaces)
- [Go by Example: Type Assertions](https://gobyexample.com/type-assertions)
- [Go by Example: Closures](https://gobyexample.com/closures)

---

> 陷阱提醒：不要滥用 `interface{}`。Go 社区推崇"显式优于隐式"，生产代码中除非必要（如 JSON 解析、DI 框架），否则不用 `interface{}`。
