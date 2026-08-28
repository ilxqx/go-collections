package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"iter"
	"slices"

	"github.com/zhangyunhao116/skipmap"
)

// concurrentSkipMap is a concurrent-safe sorted map backed by a lock-free skip list.
//   - Single-key operations like Load/Store/LoadAndDelete are atomic.
//   - Scans (Range/Seq) and bulk operations are not atomic as a whole; they are
//     best-effort snapshots under concurrency.
//   - Navigation (Floor/Ceiling/etc.) and rank operations are implemented via scans (O(n)).
type concurrentSkipMap[K Ordered, V any] struct {
	m *skipmap.OrderedMap[K, V]
}

// NewConcurrentSkipMap creates an empty concurrent sorted map.
func NewConcurrentSkipMap[K Ordered, V any]() ConcurrentSortedMap[K, V] {
	return &concurrentSkipMap[K, V]{m: skipmap.New[K, V]()}
}

// NewConcurrentSkipMapFrom creates a map populated from a standard Go map.
// Note: population is not atomic as a whole.
func NewConcurrentSkipMapFrom[K Ordered, V any](src map[K]V) ConcurrentSortedMap[K, V] {
	cm := &concurrentSkipMap[K, V]{m: skipmap.New[K, V]()}
	for k, v := range src {
		cm.m.Store(k, v)
	}
	return cm
}

// Size returns an approximate number of entries.
func (c *concurrentSkipMap[K, V]) Size() int { return c.m.Len() }

// IsEmpty reports whether the map is empty.
func (c *concurrentSkipMap[K, V]) IsEmpty() bool    { return c.Size() == 0 }
func (c *concurrentSkipMap[K, V]) IsNotEmpty() bool { return !c.IsEmpty() }

// Clear removes all entries one by one, so concurrent readers and writers
// stay safe. Best-effort under concurrency: entries inserted while Clear
// runs may survive it.
func (c *concurrentSkipMap[K, V]) Clear() {
	c.m.Range(func(k K, _ V) bool {
		c.m.Delete(k)
		return true
	})
}

// String returns a concise representation (ascending by key).
func (c *concurrentSkipMap[K, V]) String() string {
	return formatMap("concurrentSkipMap", c.Seq())
}

// Put associates value with key. Returns (oldValue, true) if key existed.
// Note: Not atomic as a read-modify-write with respect to the previous value.
// The implementation first loads the prior value (if any) and then stores the new value.
// Under heavy concurrency, the returned "old" value can be stale if another writer
// updates the same key between the Load and Store. Prefer PutIfAbsent
// or application-level Compute patterns when strict atomicity is required.
func (c *concurrentSkipMap[K, V]) Put(key K, value V) (V, bool) {
	old, ok := c.m.Load(key)
	c.m.Store(key, value)
	return old, ok
}

// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
func (c *concurrentSkipMap[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	v, loaded := c.m.LoadOrStore(key, value)
	return v, !loaded
}

// PutAll copies all entries from other into this map.
func (c *concurrentSkipMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq() {
		c.m.Store(k, v)
	}
}

// PutSeq copies entries from a sequence. Returns number of unique keys touched.
func (c *concurrentSkipMap[K, V]) PutSeq(seq iter.Seq2[K, V]) int {
	seen := make(map[K]struct{})
	changed := 0
	seq(func(k K, v V) bool {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			changed++
		}
		c.m.Store(k, v)
		return true
	})
	return changed
}

// Get returns (value, true) if key present; otherwise (zero, false).
func (c *concurrentSkipMap[K, V]) Get(key K) (V, bool) { return c.m.Load(key) }

// GetOrDefault returns value for key or defaultValue if absent.
func (c *concurrentSkipMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if v, ok := c.m.Load(key); ok {
		return v
	}
	return defaultValue
}

// Remove deletes key. Returns (oldValue, true) if key existed.
func (c *concurrentSkipMap[K, V]) Remove(key K) (V, bool) { return c.m.LoadAndDelete(key) }

// RemoveIf deletes only if (key, value) matches. Returns true if removed (best-effort).
func (c *concurrentSkipMap[K, V]) RemoveIf(key K, value V, eq Equaler[V]) bool {
	if cur, ok := c.m.Load(key); ok && eq(cur, value) {
		return c.m.Delete(key)
	}
	return false
}

// ContainsKey reports whether key exists.
func (c *concurrentSkipMap[K, V]) ContainsKey(key K) bool {
	_, ok := c.m.Load(key)
	return ok
}

// ContainsValue reports whether value exists (O(n)).
func (c *concurrentSkipMap[K, V]) ContainsValue(value V, eq Equaler[V]) bool {
	found := false
	c.m.Range(func(_ K, v V) bool {
		if eq(v, value) {
			found = true
			return false
		}
		return true
	})
	return found
}

// RemoveAll removes all specified keys. Returns count removed.
func (c *concurrentSkipMap[K, V]) RemoveAll(keys ...K) int {
	removed := 0
	for _, k := range keys {
		if _, ok := c.m.LoadAndDelete(k); ok {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes keys from the sequence. Returns count removed.
func (c *concurrentSkipMap[K, V]) RemoveSeq(seq iter.Seq[K]) int {
	removed := 0
	for k := range seq {
		if _, ok := c.m.LoadAndDelete(k); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes entries where predicate returns true. Returns count removed.
func (c *concurrentSkipMap[K, V]) RemoveFunc(predicate func(key K, value V) bool) int {
	size := c.Size()
	if size == 0 {
		return 0
	}
	dels := make([]K, 0, size/2)
	c.m.Range(func(k K, v V) bool {
		if predicate(k, v) {
			dels = append(dels, k)
		}
		return true
	})
	count := 0
	for _, k := range dels {
		if _, ok := c.m.LoadAndDelete(k); ok {
			count++
		}
	}
	return count
}

// Compute recomputes mapping for key. If keep==false, the key is removed.
// Note: Not atomic as a whole; races may occur with concurrent writers.
func (c *concurrentSkipMap[K, V]) Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool) {
	old, exists := c.m.Load(key)
	newVal, keep := remapping(key, old, exists)
	if !keep {
		if exists {
			c.m.Delete(key)
		}
		var zero V
		return zero, false
	}
	c.m.Store(key, newVal)
	return newVal, true
}

// ComputeIfAbsent computes and stores value if key is absent.
// The mapping function runs without any lock held, so it may call back into
// the map and a panic in it stores nothing and leaves the map usable. When
// two writers race on the same absent key, both may invoke mapping and the
// loser's result is discarded — the store itself is atomic.
func (c *concurrentSkipMap[K, V]) ComputeIfAbsent(key K, mapping func(key K) V) V {
	if v, ok := c.m.Load(key); ok {
		return v
	}
	v, _ := c.m.LoadOrStore(key, mapping(key))
	return v
}

// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
// Note: Not atomic as a whole.
func (c *concurrentSkipMap[K, V]) ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool) {
	old, ok := c.m.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	newVal, keep := remapping(key, old)
	if !keep {
		c.m.Delete(key)
		var zero V
		return zero, false
	}
	c.m.Store(key, newVal)
	return newVal, true
}

// Merge merges value with existing. If keep==false, removes key.
// Note: Not atomic as a whole.
func (c *concurrentSkipMap[K, V]) Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool) {
	old, ok := c.m.Load(key)
	if ok {
		merged, keep := remapping(old, value)
		if !keep {
			c.m.Delete(key)
			var zero V
			return zero, false
		}
		c.m.Store(key, merged)
		return merged, true
	}
	c.m.Store(key, value)
	return value, true
}

// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
// Note: Not atomic as a whole.
func (c *concurrentSkipMap[K, V]) Replace(key K, value V) (V, bool) {
	old, ok := c.m.Load(key)
	if ok {
		c.m.Store(key, value)
	}
	return old, ok
}

// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced (best-effort).
func (c *concurrentSkipMap[K, V]) ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool {
	if cur, ok := c.m.Load(key); ok && eq(cur, oldValue) {
		c.m.Store(key, newValue)
		return true
	}
	return false
}

// ReplaceAll replaces each value with function(key, value).
func (c *concurrentSkipMap[K, V]) ReplaceAll(function func(key K, value V) V) {
	c.m.Range(func(k K, v V) bool {
		c.m.Store(k, function(k, v))
		return true
	})
}

// Keys returns all keys as a slice (ascending).
func (c *concurrentSkipMap[K, V]) Keys() []K {
	out := make([]K, 0, c.Size())
	c.m.Range(func(k K, _ V) bool {
		out = append(out, k)
		return true
	})
	return out
}

// Values returns all values as a slice (ascending by key).
func (c *concurrentSkipMap[K, V]) Values() []V {
	out := make([]V, 0, c.Size())
	c.m.Range(func(_ K, v V) bool {
		out = append(out, v)
		return true
	})
	return out
}

// Entries returns all entries as a slice (ascending by key).
func (c *concurrentSkipMap[K, V]) Entries() []Entry[K, V] {
	out := make([]Entry[K, V], 0, c.Size())
	c.m.Range(func(k K, v V) bool {
		out = append(out, Entry[K, V]{Key: k, Value: v})
		return true
	})
	return out
}

// ForEach iterates over entries; stops early if action returns false.
func (c *concurrentSkipMap[K, V]) ForEach(action func(key K, value V) bool) {
	c.m.Range(func(k K, v V) bool { return action(k, v) })
}

// Seq returns a sequence of (key, value) pairs in ascending key order.
func (c *concurrentSkipMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.m.Range(func(k K, v V) bool { return yield(k, v) })
	}
}

// SeqKeys returns a sequence of keys in ascending order.
func (c *concurrentSkipMap[K, V]) SeqKeys() iter.Seq[K] {
	return func(yield func(K) bool) {
		c.m.Range(func(k K, _ V) bool { return yield(k) })
	}
}

// SeqValues returns a sequence of values in ascending key order.
func (c *concurrentSkipMap[K, V]) SeqValues() iter.Seq[V] {
	return func(yield func(V) bool) {
		c.m.Range(func(_ K, v V) bool { return yield(v) })
	}
}

// Clone returns a shallow snapshot as a non-concurrent HashMap.
func (c *concurrentSkipMap[K, V]) Clone() Map[K, V] {
	cp := NewHashMapWithCapacity[K, V](c.Size())
	c.m.Range(func(k K, v V) bool { cp.Put(k, v); return true })
	return cp
}

// Filter returns a new map with entries satisfying predicate (non-concurrent snapshot).
func (c *concurrentSkipMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	out := NewHashMap[K, V]()
	c.m.Range(func(k K, v V) bool {
		if predicate(k, v) {
			out.Put(k, v)
		}
		return true
	})
	return out
}

// Equals reports whether both maps contain the same entries (snapshot-based).
func (c *concurrentSkipMap[K, V]) Equals(other Map[K, V], valueEq Equaler[V]) bool {
	snap := NewHashMap[K, V]()
	c.m.Range(func(k K, v V) bool { snap.Put(k, v); return true })
	return snap.Equals(other, valueEq)
}

// GetOrCompute returns the value for key, computing and storing one if absent.
// The bool reports whether this call's computed value was stored: false means
// an already-present value was returned — compute may still have run and lost
// the race (or itself stored the key), its result then being discarded.
// The compute function runs without any lock held, so it may call back into
// the map and a panic in it stores nothing and leaves the map usable; the
// store itself is atomic.
func (c *concurrentSkipMap[K, V]) GetOrCompute(key K, compute func() V) (V, bool) {
	if v, ok := c.m.Load(key); ok {
		return v, false
	}
	v, loaded := c.m.LoadOrStore(key, compute())
	return v, !loaded
}

// RemoveAndGet atomically removes and returns the value for key.
func (c *concurrentSkipMap[K, V]) RemoveAndGet(key K) (V, bool) { return c.m.LoadAndDelete(key) }

// CompareAndSwap replaces value if current equals old (best-effort, not atomic).
func (c *concurrentSkipMap[K, V]) CompareAndSwap(key K, oldValue, newValue V, eq Equaler[V]) bool {
	if cur, ok := c.m.Load(key); ok && eq(cur, oldValue) {
		c.m.Store(key, newValue)
		return true
	}
	return false
}

// CompareAndDelete deletes entry if current value equals provided (best-effort, not atomic).
func (c *concurrentSkipMap[K, V]) CompareAndDelete(key K, value V, eq Equaler[V]) bool {
	if cur, ok := c.m.Load(key); ok && eq(cur, value) {
		return c.m.Delete(key)
	}
	return false
}

// --- SortedMap extras ---

// FirstKey returns the smallest key, or (zero, false) if empty.
func (c *concurrentSkipMap[K, V]) FirstKey() (K, bool) {
	var res K
	found := false
	c.m.Range(func(k K, _ V) bool {
		res = k
		found = true
		return false
	})
	return res, found
}

// LastKey returns the largest key, or (zero, false) if empty.
func (c *concurrentSkipMap[K, V]) LastKey() (K, bool) {
	var res K
	found := false
	c.m.Range(func(k K, _ V) bool {
		res = k
		found = true
		return true
	})
	return res, found
}

// FirstEntry returns the entry with the smallest key.
func (c *concurrentSkipMap[K, V]) FirstEntry() (Entry[K, V], bool) {
	var e Entry[K, V]
	found := false
	c.m.Range(func(k K, v V) bool {
		e = Entry[K, V]{Key: k, Value: v}
		found = true
		return false
	})
	return e, found
}

// LastEntry returns the entry with the largest key.
func (c *concurrentSkipMap[K, V]) LastEntry() (Entry[K, V], bool) {
	var e Entry[K, V]
	found := false
	c.m.Range(func(k K, v V) bool {
		e = Entry[K, V]{Key: k, Value: v}
		found = true
		return true
	})
	return e, found
}

// PopFirst removes and returns the smallest-key entry (best-effort).
func (c *concurrentSkipMap[K, V]) PopFirst() (Entry[K, V], bool) {
	for {
		e, ok := c.FirstEntry()
		if !ok {
			return Entry[K, V]{}, false
		}
		if v, deleted := c.m.LoadAndDelete(e.Key); deleted {
			return Entry[K, V]{Key: e.Key, Value: v}, true
		}
	}
}

// PopLast removes and returns the largest-key entry (best-effort).
// O(n) per call: the skip list cannot iterate backwards, so finding the
// largest key scans the whole map; draining a map this way is O(n²).
func (c *concurrentSkipMap[K, V]) PopLast() (Entry[K, V], bool) {
	for {
		e, ok := c.LastEntry()
		if !ok {
			return Entry[K, V]{}, false
		}
		if v, deleted := c.m.LoadAndDelete(e.Key); deleted {
			return Entry[K, V]{Key: e.Key, Value: v}, true
		}
	}
}

// FloorKey returns the greatest key <= k.
func (c *concurrentSkipMap[K, V]) FloorKey(k K) (K, bool) {
	var res K
	found := false
	c.m.Range(func(x K, _ V) bool {
		if x <= k {
			res = x
			found = true
			return true
		}
		return false
	})
	return res, found
}

// CeilingKey returns the smallest key >= k.
func (c *concurrentSkipMap[K, V]) CeilingKey(k K) (K, bool) {
	var res K
	found := false
	c.m.Range(func(x K, _ V) bool {
		if x >= k {
			res = x
			found = true
			return false
		}
		return true
	})
	return res, found
}

// LowerKey returns the greatest key < k.
func (c *concurrentSkipMap[K, V]) LowerKey(k K) (K, bool) {
	var res K
	found := false
	c.m.Range(func(x K, _ V) bool {
		if x < k {
			res = x
			found = true
			return true
		}
		return false
	})
	return res, found
}

// HigherKey returns the smallest key > k.
func (c *concurrentSkipMap[K, V]) HigherKey(k K) (K, bool) {
	var res K
	found := false
	c.m.Range(func(x K, _ V) bool {
		if x > k {
			res = x
			found = true
			return false
		}
		return true
	})
	return res, found
}

// FloorEntry returns entry with greatest key <= k.
func (c *concurrentSkipMap[K, V]) FloorEntry(k K) (Entry[K, V], bool) {
	if key, ok := c.FloorKey(k); ok {
		if v, ok2 := c.m.Load(key); ok2 {
			return Entry[K, V]{Key: key, Value: v}, true
		}
	}
	return Entry[K, V]{}, false
}

// CeilingEntry returns entry with smallest key >= k.
func (c *concurrentSkipMap[K, V]) CeilingEntry(k K) (Entry[K, V], bool) {
	if key, ok := c.CeilingKey(k); ok {
		if v, ok2 := c.m.Load(key); ok2 {
			return Entry[K, V]{Key: key, Value: v}, true
		}
	}
	return Entry[K, V]{}, false
}

// LowerEntry returns entry with greatest key < k.
func (c *concurrentSkipMap[K, V]) LowerEntry(k K) (Entry[K, V], bool) {
	if key, ok := c.LowerKey(k); ok {
		if v, ok2 := c.m.Load(key); ok2 {
			return Entry[K, V]{Key: key, Value: v}, true
		}
	}
	return Entry[K, V]{}, false
}

// HigherEntry returns entry with smallest key > k.
func (c *concurrentSkipMap[K, V]) HigherEntry(k K) (Entry[K, V], bool) {
	if key, ok := c.HigherKey(k); ok {
		if v, ok2 := c.m.Load(key); ok2 {
			return Entry[K, V]{Key: key, Value: v}, true
		}
	}
	return Entry[K, V]{}, false
}

// Range iterates entries with keys in [from, to] ascending.
// O(n) in the keys below from: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) Range(from, to K, action func(key K, value V) bool) {
	if from > to {
		return
	}
	c.m.Range(func(k K, v V) bool {
		if k < from {
			return true
		}
		if k > to {
			return false
		}
		return action(k, v)
	})
}

// RangeSeq returns a sequence for entries with keys in [from, to] ascending.
// O(n) in the keys below from: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) RangeSeq(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if from > to {
			return
		}
		c.m.Range(func(k K, v V) bool {
			if k < from {
				return true
			}
			if k > to {
				return false
			}
			return yield(k, v)
		})
	}
}

// RangeFrom iterates entries with keys >= from.
// O(n) regardless of the result size: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) RangeFrom(from K, action func(key K, value V) bool) {
	c.m.Range(func(k K, v V) bool {
		if k >= from {
			return action(k, v)
		}
		return true
	})
}

// RangeTo iterates entries with keys <= to.
func (c *concurrentSkipMap[K, V]) RangeTo(to K, action func(key K, value V) bool) {
	c.m.Range(func(k K, v V) bool {
		if k > to {
			return false
		}
		return action(k, v)
	})
}

// Ascend iterates all entries in ascending key order.
func (c *concurrentSkipMap[K, V]) Ascend(action func(key K, value V) bool) {
	c.m.Range(func(k K, v V) bool { return action(k, v) })
}

// Descend iterates all entries in descending key order (snapshot-based).
func (c *concurrentSkipMap[K, V]) Descend(action func(key K, value V) bool) {
	ents := c.Entries()
	for _, e := range slices.Backward(ents) {
		if !action(e.Key, e.Value) {
			return
		}
	}
}

// AscendFrom iterates entries with keys >= pivot ascending.
// O(n) regardless of the result size: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) AscendFrom(pivot K, action func(key K, value V) bool) {
	c.m.Range(func(k K, v V) bool {
		if k >= pivot {
			return action(k, v)
		}
		return true
	})
}

// DescendFrom iterates entries with keys <= pivot descending (snapshot-based).
func (c *concurrentSkipMap[K, V]) DescendFrom(pivot K, action func(key K, value V) bool) {
	ents := c.Entries()
	for _, e := range slices.Backward(ents) {
		if e.Key <= pivot {
			if !action(e.Key, e.Value) {
				return
			}
		}
	}
}

// Reversed returns a sequence iterating in descending key order (snapshot-based).
func (c *concurrentSkipMap[K, V]) Reversed() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ents := c.Entries()
		for _, e := range slices.Backward(ents) {
			if !yield(e.Key, e.Value) {
				return
			}
		}
	}
}

// SubMap returns entries with keys in [from, to] (snapshot).
// O(n) in the keys below from: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) SubMap(from, to K) SortedMap[K, V] {
	out := NewTreeMapOrdered[K, V]()
	if from > to {
		return out
	}
	c.Range(from, to, func(k K, v V) bool { out.Put(k, v); return true })
	return out
}

// HeadMap returns entries with keys < to (or <= if inclusive) (snapshot).
func (c *concurrentSkipMap[K, V]) HeadMap(to K, inclusive bool) SortedMap[K, V] {
	out := NewTreeMapOrdered[K, V]()
	c.m.Range(func(k K, v V) bool {
		if k < to || (inclusive && k == to) {
			out.Put(k, v)
			return true
		}
		return k < to
	})
	return out
}

// TailMap returns entries with keys > from (or >= if inclusive) (snapshot).
// O(n) regardless of the result size: the skip list's public API offers no pivot seek, so this scans from the smallest key.
func (c *concurrentSkipMap[K, V]) TailMap(from K, inclusive bool) SortedMap[K, V] {
	out := NewTreeMapOrdered[K, V]()
	c.m.Range(func(k K, v V) bool {
		if k > from || (inclusive && k == from) {
			out.Put(k, v)
		}
		return true
	})
	return out
}

// RankOfKey returns the 0-based rank of key, or -1 if not present (O(n)).
func (c *concurrentSkipMap[K, V]) RankOfKey(key K) int {
	idx := 0
	found := -1
	c.m.Range(func(k K, _ V) bool {
		if k == key {
			found = idx
			return false
		}
		idx++
		return true
	})
	return found
}

// GetByRank returns the entry at rank, or (Entry{}, false).
func (c *concurrentSkipMap[K, V]) GetByRank(rank int) (Entry[K, V], bool) {
	if rank < 0 {
		return Entry[K, V]{}, false
	}
	idx := 0
	var e Entry[K, V]
	ok := false
	c.m.Range(func(k K, v V) bool {
		if idx == rank {
			e = Entry[K, V]{Key: k, Value: v}
			ok = true
			return false
		}
		idx++
		return true
	})
	if !ok {
		return Entry[K, V]{}, false
	}
	return e, true
}

// CloneSorted returns a shallow copy as SortedMap snapshot.
func (c *concurrentSkipMap[K, V]) CloneSorted() SortedMap[K, V] {
	out := NewTreeMapOrdered[K, V]()
	c.m.Range(func(k K, v V) bool { out.Put(k, v); return true })
	return out
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes entries in ascending key order.
// NOTE: ConcurrentSkipMap only supports Ordered key types, so deserialization
// can be done directly without providing a comparator.
func (c *concurrentSkipMap[K, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializableMap[K, V]{Entries: c.snapshotEntries()})
}

// refill replaces the map's contents with entries. A zero-value receiver —
// gob allocates one when decoding an interface field — gets a fresh backing
// skip list, which nothing else can reference yet. Otherwise the existing
// one is refilled in place: replacing the c.m pointer would race with
// concurrent readers of the receiver.
func (c *concurrentSkipMap[K, V]) refill(entries []serializableEntry[K, V]) {
	if c.m == nil {
		c.m = skipmap.New[K, V]()
	} else {
		c.Clear()
	}
	for _, entry := range entries {
		c.m.Store(entry.Key, entry.Value)
	}
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON object.
// Since ConcurrentSkipMap only supports Ordered key types, no comparator is needed.
func (c *concurrentSkipMap[K, V]) UnmarshalJSON(data []byte) error {
	var wrapped serializableMap[K, V]
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	c.refill(wrapped.Entries)
	return nil
}

// snapshotEntries copies the entries in ascending key order (best-effort
// under concurrency).
func (c *concurrentSkipMap[K, V]) snapshotEntries() []serializableEntry[K, V] {
	entries := make([]serializableEntry[K, V], 0, c.Size())
	c.m.Range(func(k K, v V) bool {
		entries = append(entries, serializableEntry[K, V]{Key: k, Value: v})
		return true
	})
	return entries
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (c *concurrentSkipMap[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, serializableMap[K, V]{Entries: c.snapshotEntries()})
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
func (c *concurrentSkipMap[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var wrapped serializableMap[K, V]
	if err := jsonv2.UnmarshalDecode(dec, &wrapped); err != nil {
		return err
	}
	c.refill(wrapped.Entries)
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes entries in ascending key order.
func (c *concurrentSkipMap[K, V]) GobEncode() ([]byte, error) {
	entries := c.snapshotEntries()
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entries); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (c *concurrentSkipMap[K, V]) GobDecode(data []byte) error {
	var entries []serializableEntry[K, V]
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&entries); err != nil {
		return err
	}
	c.refill(entries)
	return nil
}

// Conformance.
var (
	_ ConcurrentSortedMap[int, string] = (*concurrentSkipMap[int, string])(nil)
)
