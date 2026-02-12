# concurrent map [![CI](https://github.com/orcaman/concurrent-map/actions/workflows/ci.yml/badge.svg)](https://github.com/orcaman/concurrent-map/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/orcaman/concurrent-map/v2)](https://goreportcard.com/report/github.com/orcaman/concurrent-map/v2) [![codecov](https://codecov.io/gh/orcaman/concurrent-map/branch/master/graph/badge.svg)](https://codecov.io/gh/orcaman/concurrent-map)

正如 [这里](http://golang.org/doc/faq#atomic_maps) 和 [这里](http://blog.golang.org/go-maps-in-action)所描述的, Go语言原生的`map`类型并不支持并发读写。`concurrent-map`提供了一种高性能的解决方案:通过对内部`map`进行分片，降低锁粒度，从而达到最少的锁等待时间(锁冲突)。

在Go 1.9之前，go语言标准库中并没有实现并发`map`。在Go 1.9中，引入了`sync.Map`。新的`sync.Map`与此`concurrent-map`有几个关键区别。标准库中的`sync.Map`是专为`append-only`场景设计的。因此，如果您想将`Map`用于一个类似内存数据库，那么使用我们的版本可能会受益。你可以在golang repo上读到更多，[这里](https://github.com/golang/go/issues/21035) 和 [这里](https://stackoverflow.com/questions/11063473/map-with-concurrent-access)。

***译注:`sync.Map`在读多写少性能比较好，否则并发性能很差***

## 特性

- **高性能**: 分片策略减少锁竞争
- **类型安全**: 完整支持 Go 泛型 (Go 1.18+)
- **丰富的 API**: 包含 `GetOrSet`、`Range`、`Clone`、`Merge` 等全面方法
- **充分测试**: 99.5%+ 测试覆盖率，包含竞态检测
- **JSON 支持**: 支持字符串键 map 的序列化/反序列化
- **可定制**: 可配置分片数量和自定义分片函数

## 安装

```bash
go get github.com/orcaman/concurrent-map/v2
```

## 用法

导入包:

```go
import (
    "github.com/orcaman/concurrent-map/v2"
)
```

现在包被导入到了`cmap`命名空间下。

### 基础示例

```go
// 创建一个新的 map.
m := cmap.New[string]()

// 设置变量m一个键为"foo"值为"bar"键值对
m.Set("foo", "bar")

// 从m中获取指定键值.
bar, ok := m.Get("foo")

// 删除键为"foo"的项
m.Remove("foo")
```

### 高级特性

```go
// GetOrSet - 原子性地获取现有值或设置新值
val, loaded := m.GetOrSet("key", "default")

// Range - 高效迭代，支持提前终止
m.Range(func(key string, value string) bool {
    fmt.Printf("%s: %s\n", key, value)
    return true // 继续迭代
})

// MSet - 批量插入以获得更好性能
m.MSet(map[string]string{
    "key1": "value1",
    "key2": "value2",
})

// Clone - 创建 map 的副本
cloned := m.Clone()

// Merge - 将另一个 map 合并到当前 map
m.Merge(otherMap)

// 针对特定工作负载自定义分片数量
m := cmap.NewWithShardCount[int](64)
```

### JSON 序列化

```go
// 序列化为 JSON
m := cmap.New[int]()
m.Set("a", 1)
m.Set("b", 2)
data, _ := json.Marshal(m)
// 输出: {"a":1,"b":2}

// 从 JSON 反序列化
m2, _ := cmap.NewFromJSON[int](data)
```

### 自定义键类型

对于实现了 `fmt.Stringer` 的自定义键类型:

```go
type MyKey struct {
    ID   int
    Name string
}

func (k MyKey) String() string {
    return fmt.Sprintf("%d-%s", k.ID, k.Name)
}

m := cmap.NewStringer[MyKey, string]()
m.Set(MyKey{1, "test"}, "value")
```

### 自定义分片函数

```go
// 对于整数键，使用恒等函数作为分片函数
sharding := func(key uint32) uint32 {
    return key
}

m := cmap.NewWithCustomShardingFunction[uint32, string](sharding)
```

## API 参考

### 创建

- `New[V]()` - 创建字符串键的新 map
- `NewWithShardCount[V](shardCount int)` - 使用自定义分片数量创建
- `NewStringer[K, V]()` - 使用实现了 Stringer 的自定义键类型创建
- `NewWithCustomShardingFunction[K, V](sharding func(K) uint32)` - 使用自定义分片函数创建
- `NewFromJSON[V](data []byte)` - 从 JSON 数据创建

### 基本操作

- `Set(key K, value V)` - 设置键值对
- `Get(key K) (V, bool)` - 根据键获取值
- `GetOrSet(key K, value V) (V, bool)` - 获取现有值或设置新值
- `GetAndSet(key K, value V) (V, bool)` - 设置并返回之前的值
- `SetIfAbsent(key K, value V) bool` - 仅在键不存在时设置
- `Has(key K) bool` - 检查键是否存在
- `Remove(key K)` - 删除键
- `RemoveCb(key K, cb RemoveCb) bool` - 使用回调删除
- `Pop(key K) (V, bool)` - 删除并返回值

### 批量操作

- `MSet(data map[K]V)` - 设置多个键值对
- `Clear()` - 删除所有项
- `Clone() ConcurrentMap[K, V]` - 创建副本
- `Merge(other ConcurrentMap[K, V])` - 合并另一个 map

### 迭代

- `Iter() <-chan Tuple[K, V]` - 基于通道的迭代器（已弃用）
- `IterBuffered() <-chan Tuple[K, V]` - 带缓冲的通道迭代器
- `IterCb(fn IterCb[K, V])` - 基于回调的迭代
- `Range(f func(key K, value V) bool)` - 高效的 range 迭代

### 信息

- `Count() int` / `Size() int` - 元素数量
- `IsEmpty() bool` - 检查是否为空
- `Keys() []K` - 获取所有键
- `Values() []V` - 获取所有值
- `Items() map[K]V` - 获取所有项作为 map

### 序列化

- `MarshalJSON() ([]byte, error)` - 序列化为 JSON

## 性能

并发 map 使用分片策略，将键分布到多个内部 map（分片）中，每个分片有自己的锁。与使用单个锁保护整个 map 相比，这显著减少了锁竞争。

运行基准测试:

```bash
go test -bench=. -benchmem
```

## 测试

```bash
# 运行所有测试
go test -v ./...

# 使用竞态检测运行
go test -race ./...

# 运行覆盖率测试
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 贡献说明

我们非常欢迎大家的贡献。如欲合并贡献，请遵循以下指引:
- 新建一个issue,并且叙述为什么这么做(解决一个bug，增加一个功能，等等)
- 根据核心团队对上述问题的反馈，提交一个PR，描述变更并链接到该问题。
- 新代码必须具有测试覆盖率。
- 如果代码是关于性能问题的，则必须在流程中包括基准测试(无论是在问题中还是在PR中)。
- 一般来说，我们希望`concurrent-map`尽可能简单，且与原生的`map`有相似的操作。当你新建issue时请注意这一点。

## 许可证

MIT (see [LICENSE](https://github.com/orcaman/concurrent-map/blob/master/LICENSE) file)
