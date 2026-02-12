// Package cmap provides a high-performance concurrent map implementation.
// It uses sharding to reduce lock contention and supports Go generics.
package cmap

import (
	"encoding/json"
	"fmt"
	"sync"
)

// DefaultShardCount is the default number of shards for the concurrent map.
const DefaultShardCount = 32

// SHARD_COUNT is the number of shards used for sharding (kept for backward compatibility).
// Deprecated: Use DefaultShardCount instead. This variable is kept for backward compatibility
// but modifying it concurrently is not safe.
var SHARD_COUNT = DefaultShardCount

// Stringer is a constraint that matches types that implement fmt.Stringer and are comparable.
type Stringer interface {
	fmt.Stringer
	comparable
}

// ConcurrentMap is a "thread" safe map of type K:V.
// To avoid lock bottlenecks this map is divided into several (SHARD_COUNT) map shards.
type ConcurrentMap[K comparable, V any] struct {
	shards     []*ConcurrentMapShared[K, V]
	sharding   func(key K) uint32
	shardCount int
}

// ConcurrentMapShared is a thread-safe shard containing a map and a RWMutex.
type ConcurrentMapShared[K comparable, V any] struct {
	items map[K]V
	sync.RWMutex
}

// create creates a new ConcurrentMap with the given sharding function.
func create[K comparable, V any](sharding func(key K) uint32, shardCount int) ConcurrentMap[K, V] {
	m := ConcurrentMap[K, V]{
		sharding:   sharding,
		shards:     make([]*ConcurrentMapShared[K, V], shardCount),
		shardCount: shardCount,
	}
	for i := 0; i < shardCount; i++ {
		m.shards[i] = &ConcurrentMapShared[K, V]{items: make(map[K]V)}
	}
	return m
}

// New creates a new concurrent map with string keys.
func New[V any]() ConcurrentMap[string, V] {
	return create[string, V](fnv32, DefaultShardCount)
}

// NewWithShardCount creates a new concurrent map with a custom number of shards.
func NewWithShardCount[V any](shardCount int) ConcurrentMap[string, V] {
	if shardCount <= 0 {
		shardCount = DefaultShardCount
	}
	return create[string, V](fnv32, shardCount)
}

// NewStringer creates a new concurrent map with keys that implement Stringer.
func NewStringer[K Stringer, V any]() ConcurrentMap[K, V] {
	return create[K, V](strfnv32[K], DefaultShardCount)
}

// NewWithCustomShardingFunction creates a new concurrent map with a custom sharding function.
func NewWithCustomShardingFunction[K comparable, V any](sharding func(key K) uint32) ConcurrentMap[K, V] {
	return create[K, V](sharding, DefaultShardCount)
}

// NewWithCustomShardingFunctionAndShardCount creates a new concurrent map with custom sharding and shard count.
func NewWithCustomShardingFunctionAndShardCount[K comparable, V any](sharding func(key K) uint32, shardCount int) ConcurrentMap[K, V] {
	if shardCount <= 0 {
		shardCount = DefaultShardCount
	}
	return create[K, V](sharding, shardCount)
}

// GetShard returns the shard for the given key.
func (m ConcurrentMap[K, V]) GetShard(key K) *ConcurrentMapShared[K, V] {
	return m.shards[uint(m.sharding(key))%uint(m.shardCount)]
}

// MSet sets multiple key-value pairs in the map.
// Optimized to group by shard to minimize lock/unlock operations.
func (m ConcurrentMap[K, V]) MSet(data map[K]V) {
	if len(data) == 0 {
		return
	}

	// Group items by shard
	grouped := make([][]Tuple[K, V], m.shardCount)
	for key, value := range data {
		shardIdx := uint(m.sharding(key)) % uint(m.shardCount)
		grouped[shardIdx] = append(grouped[shardIdx], Tuple[K, V]{Key: key, Val: value})
	}

	// Process each shard
	for i, items := range grouped {
		if len(items) == 0 {
			continue
		}
		shard := m.shards[i]
		shard.Lock()
		for _, item := range items {
			shard.items[item.Key] = item.Val
		}
		shard.Unlock()
	}
}

// Set sets the given value under the specified key.
func (m ConcurrentMap[K, V]) Set(key K, value V) {
	shard := m.GetShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// UpsertCb is a callback to return a new element to be inserted into the map.
// It is called while the lock is held, therefore it MUST NOT
// try to access other keys in the same map, as it can lead to deadlock since
// Go sync.RWLock is not reentrant.
type UpsertCb[V any] func(exist bool, valueInMap V, newValue V) V

// Upsert inserts or updates an element using the provided callback.
func (m ConcurrentMap[K, V]) Upsert(key K, value V, cb UpsertCb[V]) (res V) {
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	res = cb(ok, v, value)
	shard.items[key] = res
	shard.Unlock()
	return res
}

// SetIfAbsent sets the given value under the specified key only if no value is associated with it.
// Returns true if the value was set, false otherwise.
func (m ConcurrentMap[K, V]) SetIfAbsent(key K, value V) bool {
	shard := m.GetShard(key)
	shard.Lock()
	_, ok := shard.items[key]
	if !ok {
		shard.items[key] = value
	}
	shard.Unlock()
	return !ok
}

// GetOrSet returns the existing value for the key if present.
// Otherwise, it sets and returns the given value.
// The loaded result is true if the value was loaded, false if set.
func (m ConcurrentMap[K, V]) GetOrSet(key K, value V) (actual V, loaded bool) {
	shard := m.GetShard(key)
	shard.Lock()
	if v, ok := shard.items[key]; ok {
		shard.Unlock()
		return v, true
	}
	shard.items[key] = value
	shard.Unlock()
	return value, false
}

// GetAndSet sets the given value under the specified key and returns the previous value (if any).
// The loaded result is true if a previous value existed.
func (m ConcurrentMap[K, V]) GetAndSet(key K, value V) (previous V, loaded bool) {
	shard := m.GetShard(key)
	shard.Lock()
	prev, ok := shard.items[key]
	shard.items[key] = value
	shard.Unlock()
	return prev, ok
}

// Get retrieves an element from the map under the given key.
func (m ConcurrentMap[K, V]) Get(key K) (V, bool) {
	shard := m.GetShard(key)
	shard.RLock()
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// Count returns the number of elements within the map.
func (m ConcurrentMap[K, V]) Count() int {
	count := 0
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Size is an alias for Count.
func (m ConcurrentMap[K, V]) Size() int {
	return m.Count()
}

// Has checks if an item exists under the specified key.
func (m ConcurrentMap[K, V]) Has(key K) bool {
	shard := m.GetShard(key)
	shard.RLock()
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// Remove removes an element from the map.
func (m ConcurrentMap[K, V]) Remove(key K) {
	shard := m.GetShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// RemoveCb is a callback executed in a map.RemoveCb() call, while the lock is held.
// If it returns true, the element will be removed from the map.
type RemoveCb[K comparable, V any] func(key K, v V, exists bool) bool

// RemoveCb locks the shard containing the key, retrieves its current value, and calls the callback.
// If the callback returns true and the element exists, it will be removed from the map.
// Returns the value returned by the callback (even if the element was not present).
func (m ConcurrentMap[K, V]) RemoveCb(key K, cb RemoveCb[K, V]) bool {
	shard := m.GetShard(key)
	shard.Lock()
	v, ok := shard.items[key]
	remove := cb(key, v, ok)
	if remove && ok {
		delete(shard.items, key)
	}
	shard.Unlock()
	return remove
}

// Pop removes an element from the map and returns it.
func (m ConcurrentMap[K, V]) Pop(key K) (v V, exists bool) {
	shard := m.GetShard(key)
	shard.Lock()
	v, exists = shard.items[key]
	if exists {
		delete(shard.items, key)
	}
	shard.Unlock()
	return v, exists
}

// IsEmpty checks if the map is empty.
func (m ConcurrentMap[K, V]) IsEmpty() bool {
	return m.Count() == 0
}

// Tuple is used by the Iter and IterBuffered functions to wrap key-value pairs.
type Tuple[K comparable, V any] struct {
	Key K
	Val V
}

// Iter returns an iterator which can be used in a for range loop.
//
// Deprecated: Use IterBuffered or Range for better performance.
func (m ConcurrentMap[K, V]) Iter() <-chan Tuple[K, V] {
	chans := snapshot(m)
	ch := make(chan Tuple[K, V])
	go fanIn(chans, ch)
	return ch
}

// IterBuffered returns a buffered iterator which can be used in a for range loop.
func (m ConcurrentMap[K, V]) IterBuffered() <-chan Tuple[K, V] {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple[K, V], total)
	go fanIn(chans, ch)
	return ch
}

// Clear removes all items from the map.
// Optimized to recreate each shard's map instead of deleting items one by one.
func (m ConcurrentMap[K, V]) Clear() {
	for i := 0; i < m.shardCount; i++ {
		shard := m.shards[i]
		shard.Lock()
		shard.items = make(map[K]V)
		shard.Unlock()
	}
}

// snapshot returns an array of channels that contain elements in each shard,
// which effectively takes a snapshot of m.
func snapshot[K comparable, V any](m ConcurrentMap[K, V]) (chans []chan Tuple[K, V]) {
	if len(m.shards) == 0 {
		panic("cmap.ConcurrentMap is not initialized. Should run New() before usage.")
	}
	chans = make([]chan Tuple[K, V], m.shardCount)
	wg := sync.WaitGroup{}
	wg.Add(m.shardCount)

	for index, shard := range m.shards {
		go func(index int, shard *ConcurrentMapShared[K, V]) {
			shard.RLock()
			chans[index] = make(chan Tuple[K, V], len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- Tuple[K, V]{Key: key, Val: val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels chans into channel out.
func fanIn[K comparable, V any](chans []chan Tuple[K, V], out chan Tuple[K, V]) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple[K, V]) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Items returns all items as a map[K]V.
func (m ConcurrentMap[K, V]) Items() map[K]V {
	tmp := make(map[K]V, m.shardCount*16)
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return tmp
}

// IterCb is a callback function called for every key-value pair in the map.
// RLock is held for all calls within a given shard, ensuring a consistent
// view of that shard, but not across different shards.
type IterCb[K comparable, V any] func(key K, v V)

// IterCb executes the callback for every element in the map.
// This is the cheapest way to read all elements in the map.
func (m ConcurrentMap[K, V]) IterCb(fn IterCb[K, V]) {
	for _, shard := range m.shards {
		shard.RLock()
		for key, value := range shard.items {
			fn(key, value)
		}
		shard.RUnlock()
	}
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, Range stops the iteration.
// This is more efficient than using channel-based iterators.
func (m ConcurrentMap[K, V]) Range(f func(key K, value V) bool) {
	for _, shard := range m.shards {
		shard.RLock()
		for key, value := range shard.items {
			if !f(key, value) {
				shard.RUnlock()
				return
			}
		}
		shard.RUnlock()
	}
}

// Keys returns all keys as a []K.
func (m ConcurrentMap[K, V]) Keys() []K {
	count := m.Count()
	ch := make(chan K, count)
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(m.shardCount)
		for _, shard := range m.shards {
			go func(shard *ConcurrentMapShared[K, V]) {
				shard.RLock()
				for key := range shard.items {
					ch <- key
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	keys := make([]K, 0, count)
	for k := range ch {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values as a []V.
func (m ConcurrentMap[K, V]) Values() []V {
	count := m.Count()
	ch := make(chan V, count)
	go func() {
		wg := sync.WaitGroup{}
		wg.Add(m.shardCount)
		for _, shard := range m.shards {
			go func(shard *ConcurrentMapShared[K, V]) {
				shard.RLock()
				for _, value := range shard.items {
					ch <- value
				}
				shard.RUnlock()
				wg.Done()
			}(shard)
		}
		wg.Wait()
		close(ch)
	}()

	values := make([]V, 0, count)
	for v := range ch {
		values = append(values, v)
	}
	return values
}

// Clone returns a copy of the concurrent map.
func (m ConcurrentMap[K, V]) Clone() ConcurrentMap[K, V] {
	cloned := create[K, V](m.sharding, m.shardCount)
	for item := range m.IterBuffered() {
		cloned.Set(item.Key, item.Val)
	}
	return cloned
}

// Merge merges another concurrent map into this one.
// If the same key exists in both maps, the value from the other map wins.
func (m ConcurrentMap[K, V]) Merge(other ConcurrentMap[K, V]) {
	other.IterCb(func(key K, value V) {
		m.Set(key, value)
	})
}

// MarshalJSON exposes ConcurrentMap's items for JSON marshaling.
func (m ConcurrentMap[K, V]) MarshalJSON() ([]byte, error) {
	tmp := make(map[K]V)
	for item := range m.IterBuffered() {
		tmp[item.Key] = item.Val
	}
	return json.Marshal(tmp)
}

// NewFromJSON creates a new ConcurrentMap from JSON data.
// Returns an error if the JSON data is invalid.
func NewFromJSON[V any](data []byte) (ConcurrentMap[string, V], error) {
	m := New[V]()
	tmp := make(map[string]V)
	if err := json.Unmarshal(data, &tmp); err != nil {
		return m, err
	}
	for key, val := range tmp {
		m.Set(key, val)
	}
	return m, nil
}

func strfnv32[K fmt.Stringer](key K) uint32 {
	return fnv32(key.String())
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
