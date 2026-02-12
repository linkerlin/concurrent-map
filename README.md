# concurrent map [![CI](https://github.com/orcaman/concurrent-map/actions/workflows/ci.yml/badge.svg)](https://github.com/orcaman/concurrent-map/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/orcaman/concurrent-map/v2)](https://goreportcard.com/report/github.com/orcaman/concurrent-map/v2) [![codecov](https://codecov.io/gh/orcaman/concurrent-map/branch/master/graph/badge.svg)](https://codecov.io/gh/orcaman/concurrent-map)

As explained [here](http://golang.org/doc/faq#atomic_maps) and [here](http://blog.golang.org/go-maps-in-action), the `map` type in Go doesn't support concurrent reads and writes. `concurrent-map` provides a high-performance solution to this by sharding the map with minimal time spent waiting for locks.

Prior to Go 1.9, there was no concurrent map implementation in the stdlib. In Go 1.9, `sync.Map` was introduced. The new `sync.Map` has a few key differences from this map. The stdlib `sync.Map` is designed for append-only scenarios. So if you want to use the map for something more like in-memory db, you might benefit from using our version. You can read more about it in the golang repo, for example [here](https://github.com/golang/go/issues/21035) and [here](https://stackoverflow.com/questions/11063473/map-with-concurrent-access)

## Features

- **High Performance**: Sharding strategy reduces lock contention
- **Type Safe**: Full support for Go generics (Go 1.18+)
- **Rich API**: Comprehensive set of methods including `GetOrSet`, `Range`, `Clone`, `Merge`, etc.
- **Well Tested**: 99.5%+ test coverage with race detection
- **JSON Support**: Marshal/Unmarshal support for string-keyed maps
- **Customizable**: Configurable shard count and custom sharding functions

## Installation

```bash
go get github.com/orcaman/concurrent-map/v2
```

## Usage

Import the package:

```go
import (
    "github.com/orcaman/concurrent-map/v2"
)
```

The package is imported under the "cmap" namespace.

### Basic Example

```go
// Create a new map.
m := cmap.New[string]()

// Sets item within map, sets "bar" under key "foo"
m.Set("foo", "bar")

// Retrieve item from map.
bar, ok := m.Get("foo")

// Removes item under key "foo"
m.Remove("foo")
```

### Advanced Features

```go
// GetOrSet - atomically get existing or set new value
val, loaded := m.GetOrSet("key", "default")

// Range - efficient iteration with early termination
m.Range(func(key string, value string) bool {
    fmt.Printf("%s: %s\n", key, value)
    return true // continue iteration
})

// MSet - batch insert for better performance
m.MSet(map[string]string{
    "key1": "value1",
    "key2": "value2",
})

// Clone - create a copy of the map
cloned := m.Clone()

// Merge - merge another map into this one
m.Merge(otherMap)

// Custom shard count for specific workloads
m := cmap.NewWithShardCount[int](64)
```

### JSON Serialization

```go
// Marshal to JSON
m := cmap.New[int]()
m.Set("a", 1)
m.Set("b", 2)
data, _ := json.Marshal(m)
// Output: {"a":1,"b":2}

// Unmarshal from JSON
m2, _ := cmap.NewFromJSON[int](data)
```

### Custom Key Types

For custom key types that implement `fmt.Stringer`:

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

### Custom Sharding Function

```go
// For integer keys, use identity function as sharding function
sharding := func(key uint32) uint32 {
    return key
}

m := cmap.NewWithCustomShardingFunction[uint32, string](sharding)
```

## API Reference

### Creation

- `New[V]()` - Create a new map with string keys
- `NewWithShardCount[V](shardCount int)` - Create with custom shard count
- `NewStringer[K, V]()` - Create with custom key type implementing Stringer
- `NewWithCustomShardingFunction[K, V](sharding func(K) uint32)` - Create with custom sharding function
- `NewFromJSON[V](data []byte)` - Create from JSON data

### Basic Operations

- `Set(key K, value V)` - Set a key-value pair
- `Get(key K) (V, bool)` - Get value by key
- `GetOrSet(key K, value V) (V, bool)` - Get existing or set new value
- `GetAndSet(key K, value V) (V, bool)` - Set and return previous value
- `SetIfAbsent(key K, value V) bool` - Set only if key doesn't exist
- `Has(key K) bool` - Check if key exists
- `Remove(key K)` - Remove a key
- `RemoveCb(key K, cb RemoveCb) bool` - Remove with callback
- `Pop(key K) (V, bool)` - Remove and return value

### Batch Operations

- `MSet(data map[K]V)` - Set multiple key-value pairs
- `Clear()` - Remove all items
- `Clone() ConcurrentMap[K, V]` - Create a copy
- `Merge(other ConcurrentMap[K, V])` - Merge another map

### Iteration

- `Iter() <-chan Tuple[K, V]` - Channel-based iterator (deprecated)
- `IterBuffered() <-chan Tuple[K, V]` - Buffered channel iterator
- `IterCb(fn IterCb[K, V])` - Callback-based iteration
- `Range(f func(key K, value V) bool)` - Efficient range iteration

### Information

- `Count() int` / `Size() int` - Number of elements
- `IsEmpty() bool` - Check if empty
- `Keys() []K` - Get all keys
- `Values() []V` - Get all values
- `Items() map[K]V` - Get all items as a map

### Serialization

- `MarshalJSON() ([]byte, error)` - Marshal to JSON

## Performance

The concurrent map uses a sharding strategy where keys are distributed across multiple internal maps (shards), each with its own lock. This significantly reduces lock contention compared to a single lock protecting the entire map.

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Testing

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Contributing

Contributions are highly welcome. In order for a contribution to be merged, please follow these guidelines:

- Open an issue and describe what you are after (fixing a bug, adding an enhancement, etc.).
- According to the core team's feedback on the above mentioned issue, submit a pull request, describing the changes and linking to the issue.
- New code must have test coverage.
- If the code is about performance issues, you must include benchmarks in the process (either in the issue or in the PR).
- In general, we would like to keep `concurrent-map` as simple as possible and as similar to the native `map`. Please keep this in mind when opening issues.

## Language

- [中文说明](./README-zh.md)

## License

MIT (see [LICENSE](https://github.com/orcaman/concurrent-map/blob/master/LICENSE) file)
