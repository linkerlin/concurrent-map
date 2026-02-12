package cmap

import (
	"encoding/json"
	"hash/fnv"
	"sort"
	"strconv"
	"sync"
	"testing"
)

type Animal struct {
	name string
}

func TestMapCreation(t *testing.T) {
	m := New[string]()
	if m.shards == nil {
		t.Error("map is null.")
	}

	if m.Count() != 0 {
		t.Error("new map should be empty.")
	}
}

func TestMapCreationWithShardCount(t *testing.T) {
	m := NewWithShardCount[int](64)
	if m.shardCount != 64 {
		t.Errorf("expected shardCount to be 64, got %d", m.shardCount)
	}

	// Test with invalid shard count
	m2 := NewWithShardCount[int](0)
	if m2.shardCount != DefaultShardCount {
		t.Errorf("expected shardCount to be DefaultShardCount, got %d", m2.shardCount)
	}
}

func TestInsert(t *testing.T) {
	m := New[Animal]()
	elephant := Animal{"elephant"}
	monkey := Animal{"monkey"}

	m.Set("elephant", elephant)
	m.Set("monkey", monkey)

	if m.Count() != 2 {
		t.Error("map should contain exactly two elements.")
	}
}

func TestInsertAbsent(t *testing.T) {
	m := New[Animal]()
	elephant := Animal{"elephant"}
	monkey := Animal{"monkey"}

	m.SetIfAbsent("elephant", elephant)
	if ok := m.SetIfAbsent("elephant", monkey); ok {
		t.Error("map set a new value even the entry is already present")
	}
}

func TestGet(t *testing.T) {
	m := New[Animal]()

	// Get a missing element.
	val, ok := m.Get("Money")

	if ok == true {
		t.Error("ok should be false when item is missing from map.")
	}

	if (val != Animal{}) {
		t.Error("Missing values should return as null.")
	}

	elephant := Animal{"elephant"}
	m.Set("elephant", elephant)

	// Retrieve inserted element.
	elephant, ok = m.Get("elephant")
	if ok == false {
		t.Error("ok should be true for item stored within the map.")
	}

	if elephant.name != "elephant" {
		t.Error("item was modified.")
	}
}

func TestGetOrSet(t *testing.T) {
	m := New[string]()

	// First call should set the value
	val, loaded := m.GetOrSet("key", "first")
	if loaded {
		t.Error("loaded should be false on first call")
	}
	if val != "first" {
		t.Error("value should be 'first'")
	}

	// Second call should return existing value
	val, loaded = m.GetOrSet("key", "second")
	if !loaded {
		t.Error("loaded should be true on second call")
	}
	if val != "first" {
		t.Error("value should still be 'first'")
	}

	if m.Count() != 1 {
		t.Error("map should contain exactly one element")
	}
}

func TestGetAndSet(t *testing.T) {
	m := New[string]()

	// First call - no previous value
	prev, loaded := m.GetAndSet("key", "first")
	if loaded {
		t.Error("loaded should be false when key doesn't exist")
	}
	var emptyStr string
	if prev != emptyStr {
		t.Error("previous value should be zero value")
	}

	// Second call - has previous value
	prev, loaded = m.GetAndSet("key", "second")
	if !loaded {
		t.Error("loaded should be true when key exists")
	}
	if prev != "first" {
		t.Error("previous value should be 'first'")
	}

	// Verify current value
	val, _ := m.Get("key")
	if val != "second" {
		t.Error("current value should be 'second'")
	}
}

func TestHas(t *testing.T) {
	m := New[Animal]()

	// Get a missing element.
	if m.Has("Money") == true {
		t.Error("element shouldn't exists")
	}

	elephant := Animal{"elephant"}
	m.Set("elephant", elephant)

	if m.Has("elephant") == false {
		t.Error("element exists, expecting Has to return True.")
	}
}

func TestRemove(t *testing.T) {
	m := New[Animal]()

	monkey := Animal{"monkey"}
	m.Set("monkey", monkey)

	m.Remove("monkey")

	if m.Count() != 0 {
		t.Error("Expecting count to be zero once item was removed.")
	}

	temp, ok := m.Get("monkey")

	if ok != false {
		t.Error("Expecting ok to be false for missing items.")
	}

	if (temp != Animal{}) {
		t.Error("Expecting item to be nil after its removal.")
	}

	// Remove a none existing element.
	m.Remove("noone")
}

func TestRemoveCb(t *testing.T) {
	m := New[Animal]()

	monkey := Animal{"monkey"}
	m.Set("monkey", monkey)
	elephant := Animal{"elephant"}
	m.Set("elephant", elephant)

	var (
		mapKey   string
		mapVal   Animal
		wasFound bool
	)
	cb := func(key string, val Animal, exists bool) bool {
		mapKey = key
		mapVal = val
		wasFound = exists

		return val.name == "monkey"
	}

	// Monkey should be removed
	result := m.RemoveCb("monkey", cb)
	if !result {
		t.Errorf("Result was not true")
	}

	if mapKey != "monkey" {
		t.Error("Wrong key was provided to the callback")
	}

	if mapVal != monkey {
		t.Errorf("Wrong value was provided to the value")
	}

	if !wasFound {
		t.Errorf("Key was not found")
	}

	if m.Has("monkey") {
		t.Errorf("Key was not removed")
	}

	// Elephant should not be removed
	result = m.RemoveCb("elephant", cb)
	if result {
		t.Errorf("Result was true")
	}

	if mapKey != "elephant" {
		t.Error("Wrong key was provided to the callback")
	}

	if mapVal != elephant {
		t.Errorf("Wrong value was provided to the value")
	}

	if !wasFound {
		t.Errorf("Key was not found")
	}

	if !m.Has("elephant") {
		t.Errorf("Key was removed")
	}

	// Unset key should remain unset
	result = m.RemoveCb("horse", cb)
	if result {
		t.Errorf("Result was true")
	}

	if mapKey != "horse" {
		t.Error("Wrong key was provided to the callback")
	}

	if (mapVal != Animal{}) {
		t.Errorf("Wrong value was provided to the value")
	}

	if wasFound {
		t.Errorf("Key was found")
	}

	if m.Has("horse") {
		t.Errorf("Key was created")
	}
}

func TestPop(t *testing.T) {
	m := New[Animal]()

	monkey := Animal{"monkey"}
	m.Set("monkey", monkey)

	v, exists := m.Pop("monkey")

	if !exists || v != monkey {
		t.Error("Pop didn't find a monkey.")
	}

	v2, exists2 := m.Pop("monkey")

	if exists2 || v2 == monkey {
		t.Error("Pop keeps finding monkey")
	}

	if m.Count() != 0 {
		t.Error("Expecting count to be zero once item was Pop'ed.")
	}

	temp, ok := m.Get("monkey")

	if ok != false {
		t.Error("Expecting ok to be false for missing items.")
	}

	if (temp != Animal{}) {
		t.Error("Expecting item to be nil after its removal.")
	}
}

func TestCount(t *testing.T) {
	m := New[Animal]()
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	if m.Count() != 100 {
		t.Error("Expecting 100 element within map.")
	}
}

func TestSize(t *testing.T) {
	m := New[string]()
	if m.Size() != 0 {
		t.Error("Size should be 0 for empty map")
	}
	m.Set("key", "value")
	if m.Size() != 1 {
		t.Error("Size should be 1 after adding one element")
	}
}

func TestIsEmpty(t *testing.T) {
	m := New[Animal]()

	if m.IsEmpty() == false {
		t.Error("new map should be empty")
	}

	m.Set("elephant", Animal{"elephant"})

	if m.IsEmpty() != false {
		t.Error("map shouldn't be empty.")
	}
}

func TestIterator(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	counter := 0
	// Iterate over elements.
	for item := range m.Iter() {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 100 {
		t.Error("We should have counted 100 elements.")
	}
}

func TestBufferedIterator(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	counter := 0
	// Iterate over elements.
	for item := range m.IterBuffered() {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 100 {
		t.Error("We should have counted 100 elements.")
	}
}

func TestClear(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	m.Clear()

	if m.Count() != 0 {
		t.Error("We should have 0 elements.")
	}

	// Verify we can still use the map after Clear
	m.Set("new", Animal{"new"})
	if m.Count() != 1 {
		t.Error("Map should be usable after Clear")
	}
}

func TestIterCb(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	counter := 0
	// Iterate over elements.
	m.IterCb(func(key string, v Animal) {
		counter++
	})
	if counter != 100 {
		t.Error("We should have counted 100 elements.")
	}
}

func TestRange(t *testing.T) {
	m := New[int]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), i)
	}

	counter := 0
	sum := 0
	m.Range(func(key string, value int) bool {
		counter++
		sum += value
		return true
	})

	if counter != 100 {
		t.Errorf("We should have counted 100 elements, got %d", counter)
	}
	if sum != 4950 { // 0+1+2+...+99 = 4950
		t.Errorf("Sum should be 4950, got %d", sum)
	}

	// Test early termination
	counter = 0
	m.Range(func(key string, value int) bool {
		counter++
		return counter < 10
	})
	if counter != 10 {
		t.Errorf("Counter should be 10 after early termination, got %d", counter)
	}
}

func TestItems(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	items := m.Items()

	if len(items) != 100 {
		t.Error("We should have counted 100 elements.")
	}

	// Verify all items are present
	for i := 0; i < 100; i++ {
		key := strconv.Itoa(i)
		if val, ok := items[key]; !ok {
			t.Errorf("Key %s not found in items", key)
		} else if val.name != key {
			t.Errorf("Value mismatch for key %s", key)
		}
	}
}

func TestKeys(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	keys := m.Keys()
	if len(keys) != 100 {
		t.Error("We should have counted 100 elements.")
	}

	// Verify all keys are unique and valid
	keyMap := make(map[string]bool)
	for _, key := range keys {
		if key == "" {
			t.Error("Empty key returned")
		}
		if keyMap[key] {
			t.Errorf("Duplicate key: %s", key)
		}
		keyMap[key] = true
	}
}

func TestValues(t *testing.T) {
	m := New[int]()

	// Insert 100 elements.
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), i*2)
	}

	values := m.Values()
	if len(values) != 100 {
		t.Errorf("We should have 100 values, got %d", len(values))
	}

	// Calculate sum
	sum := 0
	for _, v := range values {
		sum += v
	}
	expectedSum := 9900 // 0+2+4+...+198 = 2*(0+1+...+99) = 2*4950 = 9900
	if sum != expectedSum {
		t.Errorf("Sum should be %d, got %d", expectedSum, sum)
	}
}

func TestClone(t *testing.T) {
	m := New[int]()
	for i := 0; i < 100; i++ {
		m.Set(strconv.Itoa(i), i)
	}

	cloned := m.Clone()

	if cloned.Count() != m.Count() {
		t.Error("Cloned map should have same count")
	}

	// Verify cloned map has same values
	for i := 0; i < 100; i++ {
		key := strconv.Itoa(i)
		val, ok := cloned.Get(key)
		if !ok {
			t.Errorf("Key %s not found in cloned map", key)
		}
		if val != i {
			t.Errorf("Value mismatch for key %s", key)
		}
	}

	// Verify modifying clone doesn't affect original
	cloned.Set("new", 999)
	if m.Has("new") {
		t.Error("Modifying clone should not affect original")
	}
}

func TestMerge(t *testing.T) {
	m1 := New[int]()
	m1.Set("a", 1)
	m1.Set("b", 2)

	m2 := New[int]()
	m2.Set("b", 20) // Should overwrite m1's "b"
	m2.Set("c", 30)

	m1.Merge(m2)

	if m1.Count() != 3 {
		t.Errorf("Merged map should have 3 elements, got %d", m1.Count())
	}

	val, _ := m1.Get("a")
	if val != 1 {
		t.Errorf("Value for 'a' should be 1, got %d", val)
	}

	val, _ = m1.Get("b")
	if val != 20 {
		t.Errorf("Value for 'b' should be 20 (overwritten), got %d", val)
	}

	val, _ = m1.Get("c")
	if val != 30 {
		t.Errorf("Value for 'c' should be 30, got %d", val)
	}
}

func TestMSet(t *testing.T) {
	animals := map[string]Animal{
		"elephant": {"elephant"},
		"monkey":   {"monkey"},
	}
	m := New[Animal]()
	m.MSet(animals)

	if m.Count() != 2 {
		t.Error("map should contain exactly two elements.")
	}

	// Verify values
	for name, expected := range animals {
		actual, ok := m.Get(name)
		if !ok {
			t.Errorf("Key %s not found", name)
		}
		if actual != expected {
			t.Errorf("Value mismatch for key %s", name)
		}
	}
}

func TestMSetEmpty(t *testing.T) {
	m := New[string]()
	m.Set("existing", "value")

	emptyMap := make(map[string]string)
	m.MSet(emptyMap)

	if m.Count() != 1 {
		t.Error("MSet with empty map should not change anything")
	}
}

func TestConcurrent(t *testing.T) {
	m := New[int]()
	ch := make(chan int)
	const iterations = 1000
	var a [iterations]int

	// Using go routines insert 1000 ints into our map.
	go func() {
		for i := 0; i < iterations/2; i++ {
			// Add item to map.
			m.Set(strconv.Itoa(i), i)

			// Retrieve item from map.
			val, _ := m.Get(strconv.Itoa(i))

			// Write to channel inserted value.
			ch <- val
		} // Call go routine with current index.
	}()

	go func() {
		for i := iterations / 2; i < iterations; i++ {
			// Add item to map.
			m.Set(strconv.Itoa(i), i)

			// Retrieve item from map.
			val, _ := m.Get(strconv.Itoa(i))

			// Write to channel inserted value.
			ch <- val
		} // Call go routine with current index.
	}()

	// Wait for all go routines to finish.
	counter := 0
	for elem := range ch {
		a[counter] = elem
		counter++
		if counter == iterations {
			break
		}
	}

	// Sorts array, will make is simpler to verify all inserted values we're returned.
	sort.Ints(a[0:iterations])

	// Make sure map contains 1000 elements.
	if m.Count() != iterations {
		t.Error("Expecting 1000 elements.")
	}

	// Make sure all inserted values we're fetched from map.
	for i := 0; i < iterations; i++ {
		if i != a[i] {
			t.Error("missing value", i)
		}
	}
}

func TestConcurrentGetOrSet(t *testing.T) {
	m := New[int]()
	var wg sync.WaitGroup
	const numGoroutines = 100
	const numIterations = 100

	counter := 0
	var counterMu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				key := strconv.Itoa(j)
				_, loaded := m.GetOrSet(key, j)
				if !loaded {
					counterMu.Lock()
					counter++
					counterMu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// Only numIterations unique keys, so GetOrSet should have set each only once
	if counter != numIterations {
		t.Errorf("Expected %d new sets, got %d", numIterations, counter)
	}

	if m.Count() != numIterations {
		t.Errorf("Expected %d elements in map, got %d", numIterations, m.Count())
	}
}

func TestJsonMarshal(t *testing.T) {
	oldShardCount := SHARD_COUNT
	SHARD_COUNT = 2
	defer func() { SHARD_COUNT = oldShardCount }()

	expected := `{"a":1,"b":2}`
	m := New[int]()
	m.Set("a", 1)
	m.Set("b", 2)
	j, err := json.Marshal(m)
	if err != nil {
		t.Error(err)
	}

	if string(j) != expected {
		t.Error("json", string(j), "differ from expected", expected)
		return
	}
}

func TestNewFromJSON(t *testing.T) {
	jsonData := `{"a":1,"b":2,"c":3}`

	m, err := NewFromJSON[int]([]byte(jsonData))
	if err != nil {
		t.Errorf("NewFromJSON failed: %v", err)
	}

	if m.Count() != 3 {
		t.Errorf("Expected 3 elements, got %d", m.Count())
	}

	val, _ := m.Get("a")
	if val != 1 {
		t.Errorf("Expected a=1, got %d", val)
	}
}

func TestNewFromJSONInvalid(t *testing.T) {
	_, err := NewFromJSON[int]([]byte(`invalid json`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestNewFromJSONEmpty(t *testing.T) {
	m, err := NewFromJSON[int]([]byte(`{}`))
	if err != nil {
		t.Errorf("NewFromJSON of empty object failed: %v", err)
	}
	if m.Count() != 0 {
		t.Errorf("Expected 0 elements, got %d", m.Count())
	}
}

func TestFnv32(t *testing.T) {
	key := []byte("ABC")

	hasher := fnv.New32()
	_, err := hasher.Write(key)
	if err != nil {
		t.Errorf("hasher.Write failed: %v", err)
	}
	if fnv32(string(key)) != hasher.Sum32() {
		t.Errorf("Bundled fnv32 produced %d, expected result from hash/fnv32 is %d", fnv32(string(key)), hasher.Sum32())
	}
}

func TestUpsert(t *testing.T) {
	dolphin := Animal{"dolphin"}
	whale := Animal{"whale"}
	tiger := Animal{"tiger"}
	lion := Animal{"lion"}

	cb := func(exists bool, valueInMap Animal, newValue Animal) Animal {
		if !exists {
			return newValue
		}
		valueInMap.name += newValue.name
		return valueInMap
	}

	m := New[Animal]()
	m.Set("marine", dolphin)
	m.Upsert("marine", whale, cb)
	m.Upsert("predator", tiger, cb)
	m.Upsert("predator", lion, cb)

	if m.Count() != 2 {
		t.Error("map should contain exactly two elements.")
	}

	marineAnimals, ok := m.Get("marine")
	if marineAnimals.name != "dolphinwhale" || !ok {
		t.Error("Set, then Upsert failed")
	}

	predators, ok := m.Get("predator")
	if !ok || predators.name != "tigerlion" {
		t.Error("Upsert, then Upsert failed")
	}
}

func TestKeysWhenRemoving(t *testing.T) {
	m := New[Animal]()

	// Insert 100 elements.
	Total := 100
	for i := 0; i < Total; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}

	// Remove 10 elements concurrently.
	Num := 10
	for i := 0; i < Num; i++ {
		go func(c *ConcurrentMap[string, Animal], n int) {
			c.Remove(strconv.Itoa(n))
		}(&m, i)
	}
	keys := m.Keys()
	for _, k := range keys {
		if k == "" {
			t.Error("Empty keys returned")
		}
	}
}

func TestUnDrainedIter(t *testing.T) {
	m := New[Animal]()
	// Insert 100 elements.
	Total := 100
	for i := 0; i < Total; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	counter := 0
	// Iterate over elements.
	ch := m.Iter()
	for item := range ch {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
		if counter == 42 {
			break
		}
	}
	for i := Total; i < 2*Total; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	for item := range ch {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 100 {
		t.Error("We should have been right where we stopped")
	}

	counter = 0
	for item := range m.IterBuffered() {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 200 {
		t.Error("We should have counted 200 elements.")
	}
}

func TestUnDrainedIterBuffered(t *testing.T) {
	m := New[Animal]()
	// Insert 100 elements.
	Total := 100
	for i := 0; i < Total; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	counter := 0
	// Iterate over elements.
	ch := m.IterBuffered()
	for item := range ch {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
		if counter == 42 {
			break
		}
	}
	for i := Total; i < 2*Total; i++ {
		m.Set(strconv.Itoa(i), Animal{strconv.Itoa(i)})
	}
	for item := range ch {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 100 {
		t.Error("We should have been right where we stopped")
	}

	counter = 0
	for item := range m.IterBuffered() {
		val := item.Val

		if (val == Animal{}) {
			t.Error("Expecting an object.")
		}
		counter++
	}

	if counter != 200 {
		t.Error("We should have counted 200 elements.")
	}
}

func TestNilValues(t *testing.T) {
	m := New[*Animal]()
	m.Set("nil", nil)
	val, ok := m.Get("nil")
	if !ok {
		t.Error("nil value should be found")
	}
	if val != nil {
		t.Error("nil value should be nil")
	}
}

func TestEmptyStringKey(t *testing.T) {
	m := New[int]()
	m.Set("", 42)

	val, ok := m.Get("")
	if !ok {
		t.Error("Empty string key should be found")
	}
	if val != 42 {
		t.Errorf("Value should be 42, got %d", val)
	}
}

func TestConcurrentRemoveDuringIteration(t *testing.T) {
	m := New[int]()
	for i := 0; i < 1000; i++ {
		m.Set(strconv.Itoa(i), i)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for item := range m.IterBuffered() {
			_ = item.Val
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.Remove(strconv.Itoa(i))
		}
	}()

	wg.Wait()
}

func TestConcurrentClearAndSet(t *testing.T) {
	m := New[int]()

	var wg sync.WaitGroup
	wg.Add(3)

	// Writer
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.Set(strconv.Itoa(i), i)
		}
	}()

	// Clearer
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.Clear()
		}
	}()

	// Reader
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			m.Get(strconv.Itoa(i))
		}
	}()

	wg.Wait()
}

func TestCustomShardingFunction(t *testing.T) {
	// Use a simple modulo sharding for integers
	sharding := func(key uint32) uint32 {
		return key
	}

	m := NewWithCustomShardingFunction[uint32, string](sharding)
	for i := uint32(0); i < 100; i++ {
		m.Set(i, strconv.Itoa(int(i)))
	}

	if m.Count() != 100 {
		t.Errorf("Expected 100 elements, got %d", m.Count())
	}

	for i := uint32(0); i < 100; i++ {
		val, ok := m.Get(i)
		if !ok {
			t.Errorf("Key %d not found", i)
		}
		if val != strconv.Itoa(int(i)) {
			t.Errorf("Value mismatch for key %d", i)
		}
	}
}

func TestNewWithCustomShardingFunctionAndShardCount(t *testing.T) {
	sharding := func(key int) uint32 {
		return uint32(key)
	}

	m := NewWithCustomShardingFunctionAndShardCount[int, string](sharding, 64)
	if m.shardCount != 64 {
		t.Errorf("Expected shardCount 64, got %d", m.shardCount)
	}

	// Test with invalid shard count
	m2 := NewWithCustomShardingFunctionAndShardCount[int, string](sharding, 0)
	if m2.shardCount != DefaultShardCount {
		t.Errorf("Expected DefaultShardCount, got %d", m2.shardCount)
	}
}

// TestStringerKeyType tests using a custom key type that implements Stringer
type TestKey struct {
	id   int
	name string
}

func (k TestKey) String() string {
	return strconv.Itoa(k.id) + "-" + k.name
}

func TestStringerKeyType(t *testing.T) {
	m := NewStringer[TestKey, int]()

	key1 := TestKey{1, "first"}
	key2 := TestKey{2, "second"}

	m.Set(key1, 100)
	m.Set(key2, 200)

	if m.Count() != 2 {
		t.Errorf("Expected 2 elements, got %d", m.Count())
	}

	val1, ok := m.Get(key1)
	if !ok || val1 != 100 {
		t.Error("Failed to retrieve value for key1")
	}

	val2, ok := m.Get(key2)
	if !ok || val2 != 200 {
		t.Error("Failed to retrieve value for key2")
	}
}

// TestRaceCondition runs various operations concurrently to detect race conditions
func TestRaceCondition(t *testing.T) {
	m := New[int]()
	var wg sync.WaitGroup

	// Multiple goroutines performing different operations
	operations := []func(){
		func() { m.Set("key", 1) },
		func() { m.Get("key") },
		func() { m.Has("key") },
		func() { m.Remove("key") },
		func() { m.Count() },
		func() { m.Keys() },
		func() { m.Values() },
		func() { m.Items() },
		func() { m.Clear() },
		func() { m.IterBuffered() },
		func() { m.Range(func(k string, v int) bool { return true }) },
	}

	for i := 0; i < 100; i++ {
		for _, op := range operations {
			wg.Add(1)
			go func(operation func()) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					operation()
				}
			}(op)
		}
	}

	wg.Wait()
}
