package cmap

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

type Integer int

func (i Integer) String() string {
	return strconv.Itoa(int(i))
}

func BenchmarkItems(b *testing.B) {
	m := New[Animal]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Items()
	}
}

func BenchmarkItemsInteger(b *testing.B) {
	m := NewStringer[Integer, Animal]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set((Integer)(i), Animal{strconv.Itoa(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Items()
	}
}

func directSharding(key uint32) uint32 {
	return key
}

func BenchmarkItemsInt(b *testing.B) {
	m := NewWithCustomShardingFunction[uint32, Animal](directSharding)

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set((uint32)(i), Animal{strconv.Itoa(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Items()
	}
}

func BenchmarkMarshalJson(b *testing.B) {
	m := New[Animal]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.MarshalJSON()
		if err != nil {
			b.FailNow()
		}
	}
}

func BenchmarkStrconv(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strconv.Itoa(i)
	}
}

func BenchmarkSingleInsertAbsent(b *testing.B) {
	m := New[string]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set(strconv.Itoa(i), "value")
	}
}

func BenchmarkSingleInsertAbsentSyncMap(b *testing.B) {
	var m sync.Map
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store(strconv.Itoa(i), "value")
	}
}

func BenchmarkSingleInsertPresent(b *testing.B) {
	m := New[string]()
	m.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Set("key", "value")
	}
}

func BenchmarkSingleInsertPresentSyncMap(b *testing.B) {
	var m sync.Map
	m.Store("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Store("key", "value")
	}
}

func benchmarkMultiInsertDifferent(b *testing.B) {
	m := New[string]()
	finished := make(chan struct{}, b.N)
	_, set := GetSet(m, finished)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i), "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiInsertDifferentSyncMap(b *testing.B) {
	var m sync.Map
	finished := make(chan struct{}, b.N)
	_, set := GetSetSyncMap[string, string](&m, finished)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i), "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiInsertDifferent_1_Shard(b *testing.B) {
	runWithShards(benchmarkMultiInsertDifferent, b, 1)
}
func BenchmarkMultiInsertDifferent_16_Shard(b *testing.B) {
	runWithShards(benchmarkMultiInsertDifferent, b, 16)
}
func BenchmarkMultiInsertDifferent_32_Shard(b *testing.B) {
	runWithShards(benchmarkMultiInsertDifferent, b, 32)
}
func BenchmarkMultiInsertDifferent_256_Shard(b *testing.B) {
	runWithShards(benchmarkMultiInsertDifferent, b, 256)
}

func BenchmarkMultiInsertSame(b *testing.B) {
	m := New[string]()
	finished := make(chan struct{}, b.N)
	_, set := GetSet(m, finished)
	m.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set("key", "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiInsertSameSyncMap(b *testing.B) {
	var m sync.Map
	finished := make(chan struct{}, b.N)
	_, set := GetSetSyncMap[string, string](&m, finished)
	m.Store("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set("key", "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSame(b *testing.B) {
	m := New[string]()
	finished := make(chan struct{}, b.N)
	get, _ := GetSet(m, finished)
	m.Set("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go get("key", "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSameSyncMap(b *testing.B) {
	var m sync.Map
	finished := make(chan struct{}, b.N)
	get, _ := GetSetSyncMap[string, string](&m, finished)
	m.Store("key", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go get("key", "value")
	}
	for i := 0; i < b.N; i++ {
		<-finished
	}
}

func benchmarkMultiGetSetDifferent(b *testing.B) {
	m := New[string]()
	finished := make(chan struct{}, 2*b.N)
	get, set := GetSet(m, finished)
	m.Set("-1", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i-1), "value")
		go get(strconv.Itoa(i), "value")
	}
	for i := 0; i < 2*b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSetDifferentSyncMap(b *testing.B) {
	var m sync.Map
	finished := make(chan struct{}, 2*b.N)
	get, set := GetSetSyncMap[string, string](&m, finished)
	m.Store("-1", "value")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i-1), "value")
		go get(strconv.Itoa(i), "value")
	}
	for i := 0; i < 2*b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSetDifferent_1_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetDifferent, b, 1)
}
func BenchmarkMultiGetSetDifferent_16_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetDifferent, b, 16)
}
func BenchmarkMultiGetSetDifferent_32_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetDifferent, b, 32)
}
func BenchmarkMultiGetSetDifferent_256_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetDifferent, b, 256)
}

func benchmarkMultiGetSetBlock(b *testing.B) {
	m := New[string]()
	finished := make(chan struct{}, 2*b.N)
	get, set := GetSet(m, finished)
	for i := 0; i < b.N; i++ {
		m.Set(strconv.Itoa(i%100), "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i%100), "value")
		go get(strconv.Itoa(i%100), "value")
	}
	for i := 0; i < 2*b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSetBlockSyncMap(b *testing.B) {
	var m sync.Map
	finished := make(chan struct{}, 2*b.N)
	get, set := GetSetSyncMap[string, string](&m, finished)
	for i := 0; i < b.N; i++ {
		m.Store(strconv.Itoa(i%100), "value")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go set(strconv.Itoa(i%100), "value")
		go get(strconv.Itoa(i%100), "value")
	}
	for i := 0; i < 2*b.N; i++ {
		<-finished
	}
}

func BenchmarkMultiGetSetBlock_1_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetBlock, b, 1)
}
func BenchmarkMultiGetSetBlock_16_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetBlock, b, 16)
}
func BenchmarkMultiGetSetBlock_32_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetBlock, b, 32)
}
func BenchmarkMultiGetSetBlock_256_Shard(b *testing.B) {
	runWithShards(benchmarkMultiGetSetBlock, b, 256)
}

func GetSet[K comparable, V any](m ConcurrentMap[K, V], finished chan struct{}) (set func(key K, value V), get func(key K, value V)) {
	return func(key K, value V) {
			for i := 0; i < 10; i++ {
				m.Get(key)
			}
			finished <- struct{}{}
		}, func(key K, value V) {
			for i := 0; i < 10; i++ {
				m.Set(key, value)
			}
			finished <- struct{}{}
		}
}

func GetSetSyncMap[K comparable, V any](m *sync.Map, finished chan struct{}) (get func(key K, value V), set func(key K, value V)) {
	get = func(key K, value V) {
		for i := 0; i < 10; i++ {
			m.Load(key)
		}
		finished <- struct{}{}
	}
	set = func(key K, value V) {
		for i := 0; i < 10; i++ {
			m.Store(key, value)
		}
		finished <- struct{}{}
	}
	return
}

func runWithShards(bench func(b *testing.B), b *testing.B, shardsCount int) {
	oldShardsCount := SHARD_COUNT
	SHARD_COUNT = shardsCount
	bench(b)
	SHARD_COUNT = oldShardsCount
}

func BenchmarkKeys(b *testing.B) {
	m := New[Animal]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Keys()
	}
}

func BenchmarkValues(b *testing.B) {
	m := New[int]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Values()
	}
}

func BenchmarkRange(b *testing.B) {
	m := New[int]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := 0
		m.Range(func(key string, value int) bool {
			counter++
			return true
		})
	}
}

func BenchmarkIterCb(b *testing.B) {
	m := New[int]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := 0
		m.IterCb(func(key string, value int) {
			counter++
		})
	}
}

func BenchmarkIterBuffered(b *testing.B) {
	m := New[int]()

	// Insert 10000 elements.
	for i := 0; i < 10000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter := 0
		for item := range m.IterBuffered() {
			_ = item
			counter++
		}
	}
}

func BenchmarkGetOrSet(b *testing.B) {
	m := New[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetOrSet(strconv.Itoa(i), i)
	}
}

func BenchmarkGetAndSet(b *testing.B) {
	m := New[int]()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetAndSet(strconv.Itoa(i), i)
	}
}

func BenchmarkMSet(b *testing.B) {
	data := make(map[string]int)
	for i := 0; i < 1000; i++ {
		data[strconv.Itoa(i)] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New[int]()
		m.MSet(data)
	}
}

func BenchmarkClone(b *testing.B) {
	m := New[int]()
	for i := 0; i < 1000; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Clone()
	}
}

func BenchmarkClear(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		m := New[int]()
		for j := 0; j < 1000; j++ {
			m.Set(strconv.Itoa(j), j)
		}
		b.StartTimer()
		m.Clear()
	}
}

// BenchmarkShardCountComparison compares performance with different shard counts
func BenchmarkShardCountComparison(b *testing.B) {
	for _, shards := range []int{1, 16, 32, 64, 128, 256} {
		b.Run(fmt.Sprintf("shards-%d", shards), func(b *testing.B) {
			oldShardCount := SHARD_COUNT
			SHARD_COUNT = shards
			defer func() { SHARD_COUNT = oldShardCount }()

			m := New[int]()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					m.Set(strconv.Itoa(i), i)
					i++
				}
			})
		})
	}
}

// BenchmarkReadWriteRatio tests different read/write ratios
func BenchmarkReadWriteRatio(b *testing.B) {
	ratios := []struct {
		reads  int
		writes int
	}{
		{100, 0}, // Read-only
		{90, 10}, // Read-heavy
		{50, 50}, // Balanced
		{10, 90}, // Write-heavy
		{0, 100}, // Write-only
	}

	for _, ratio := range ratios {
		name := fmt.Sprintf("reads%d-writes%d", ratio.reads, ratio.writes)
		b.Run(name, func(b *testing.B) {
			m := New[int]()
			// Pre-populate
			for i := 0; i < 1000; i++ {
				m.Set(strconv.Itoa(i), i)
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					// Cycle through operations based on ratio
					cycle := i % (ratio.reads + ratio.writes)
					if cycle < ratio.reads {
						m.Get(strconv.Itoa(i % 1000))
					} else {
						m.Set(strconv.Itoa(i%1000), i)
					}
					i++
				}
			})
		})
	}
}
