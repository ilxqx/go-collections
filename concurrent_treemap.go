package collections

import (
	"bytes"
	"cmp"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"iter"
	"sync"
)

// concurrentTreeMap is a concurrent-safe sorted map implemented by guarding treeMap with RWMutex.
//   - Single-key operations are atomic under the mutex.
//   - Iteration methods (Seq/ForEach/Range*/Ascend*/Reversed) operate on snapshots to avoid holding locks
//     while invoking user callbacks.
//   - For atomic compound operations (Compute*/Merge/GetOrCompute/PutIfAbsent/CompareAnd*), user callbacks
//     are executed while holding the lock. Avoid calling back into the same map from those callbacks.
type concurrentTreeMap[K any, V any] struct {
	mu sync.RWMutex
	tm *treeMap[K, V]
}

// NewConcurrentTreeMap creates an empty concurrent sorted map with a custom key comparator.
func NewConcurrentTreeMap[K any, V any](cmpK Comparator[K]) ConcurrentSortedMap[K, V] {
	return &concurrentTreeMap[K, V]{tm: newTreeMap[K, V](cmpK)}
}

// NewConcurrentTreeMapOrdered creates an empty map for Ordered keys.
func NewConcurrentTreeMapOrdered[K Ordered, V any]() ConcurrentSortedMap[K, V] {
	return NewConcurrentTreeMap[K, V](cmp.Compare)
}

// NewConcurrentTreeMapFrom creates a map populated from a standard Go map.
func NewConcurrentTreeMapFrom[K comparable, V any](cmpK Comparator[K], m map[K]V) ConcurrentSortedMap[K, V] {
	ct := &concurrentTreeMap[K, V]{tm: newTreeMap[K, V](cmpK)}
	ct.mu.Lock()
	defer ct.mu.Unlock()
	for k, v := range m {
		ct.tm.bt.Set(mapEntry[K, V]{key: k, value: v})
	}
	return ct
}

// Size returns the number of entries.
func (c *concurrentTreeMap[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.Size()
}

// IsEmpty reports whether the map is empty.
func (c *concurrentTreeMap[K, V]) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.IsEmpty()
}

// IsNotEmpty reports whether the map contains at least one entry.
func (c *concurrentTreeMap[K, V]) IsNotEmpty() bool { return !c.IsEmpty() }

// Clear removes all entries.
func (c *concurrentTreeMap[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tm.Clear()
}

// String returns a concise representation (ascending by key).
func (c *concurrentTreeMap[K, V]) String() string {
	return formatMap("concurrentTreeMap", c.Seq())
}

// Put associates value with key. Returns (oldValue, true) if key existed.
func (c *concurrentTreeMap[K, V]) Put(key K, value V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Put(key, value)
}

// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
func (c *concurrentTreeMap[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.PutIfAbsent(key, value)
}

// PutAll copies all entries from other into this map.
func (c *concurrentTreeMap[K, V]) PutAll(other Map[K, V]) {
	// Avoid holding lock while consuming external sequence.
	buf := make([]Entry[K, V], 0, 16)
	for k, v := range other.Seq() {
		buf = append(buf, Entry[K, V]{Key: k, Value: v})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range buf {
		c.tm.bt.Set(mapEntry[K, V]{key: e.Key, value: e.Value})
	}
}

// PutSeq copies entries from a sequence. Returns number of unique keys touched.
func (c *concurrentTreeMap[K, V]) PutSeq(seq iter.Seq2[K, V]) int {
	buf := make([]Entry[K, V], 0, 16)
	seq(func(k K, v V) bool {
		buf = append(buf, Entry[K, V]{Key: k, Value: v})
		return true
	})
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := newTreeSet(c.tm.keyCmp)
	changed := 0
	for _, e := range buf {
		if !seen.Contains(e.Key) {
			seen.Add(e.Key)
			changed++
		}
		c.tm.bt.Set(mapEntry[K, V]{key: e.Key, value: e.Value})
	}
	return changed
}

// Get returns (value, true) if key present; otherwise (zero, false).
func (c *concurrentTreeMap[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.Get(key)
}

// GetOrDefault returns value for key or defaultValue if absent.
func (c *concurrentTreeMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.GetOrDefault(key, defaultValue)
}

// Remove deletes key. Returns (oldValue, true) if key existed.
func (c *concurrentTreeMap[K, V]) Remove(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Remove(key)
}

// RemoveIf deletes only if (key, value) matches. Returns true if removed.
func (c *concurrentTreeMap[K, V]) RemoveIf(key K, value V, eq Equaler[V]) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.RemoveIf(key, value, eq)
}

// ContainsKey reports whether key exists.
func (c *concurrentTreeMap[K, V]) ContainsKey(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.ContainsKey(key)
}

// ContainsValue reports whether value exists (O(n)) using snapshot to avoid calling eq under lock.
func (c *concurrentTreeMap[K, V]) ContainsValue(value V, eq Equaler[V]) bool {
	vals := c.Values()
	for _, v := range vals {
		if eq(v, value) {
			return true
		}
	}
	return false
}

// RemoveAll removes all specified keys. Returns count removed.
func (c *concurrentTreeMap[K, V]) RemoveAll(keys ...K) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.RemoveAll(keys...)
}

// RemoveSeq removes keys from the sequence. Returns count removed.
func (c *concurrentTreeMap[K, V]) RemoveSeq(seq iter.Seq[K]) int {
	var (
		keys []K
		size int
	)
	c.mu.RLock()
	size = c.tm.Size()
	c.mu.RUnlock()
	if size == 0 {
		return 0
	}
	keys = make([]K, 0, min(size, 16))
	for k := range seq {
		keys = append(keys, k)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for _, k := range keys {
		if _, ok := c.tm.Remove(k); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes entries where predicate returns true. Returns count removed.
func (c *concurrentTreeMap[K, V]) RemoveFunc(predicate func(key K, value V) bool) int {
	ents := c.Entries()
	size := len(ents)
	if size == 0 {
		return 0
	}
	dels := make([]K, 0, size/2)
	for _, e := range ents {
		if predicate(e.Key, e.Value) {
			dels = append(dels, e.Key)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for _, k := range dels {
		if _, ok := c.tm.Remove(k); ok {
			removed++
		}
	}
	return removed
}

// Compute recomputes mapping for key. If keep==false, the key is removed.
func (c *concurrentTreeMap[K, V]) Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Compute(key, remapping)
}

// ComputeIfAbsent computes and stores value if key is absent.
func (c *concurrentTreeMap[K, V]) ComputeIfAbsent(key K, mapping func(key K) V) V {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.ComputeIfAbsent(key, mapping)
}

// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
func (c *concurrentTreeMap[K, V]) ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.ComputeIfPresent(key, remapping)
}

// Merge merges value with existing. If keep==false, removes key.
func (c *concurrentTreeMap[K, V]) Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Merge(key, value, remapping)
}

// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
func (c *concurrentTreeMap[K, V]) Replace(key K, value V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Replace(key, value)
}

// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced.
func (c *concurrentTreeMap[K, V]) ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.ReplaceIf(key, oldValue, newValue, eq)
}

// ReplaceAll replaces each entry's value with the result of the function (executed under lock).
func (c *concurrentTreeMap[K, V]) ReplaceAll(function func(key K, value V) V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tm.ReplaceAll(function)
}

// Keys returns all keys as a slice (ascending).
func (c *concurrentTreeMap[K, V]) Keys() []K {
	c.mu.RLock()
	keys := c.tm.Keys()
	c.mu.RUnlock()
	return keys
}

// Values returns all values as a slice (ascending by key).
func (c *concurrentTreeMap[K, V]) Values() []V {
	c.mu.RLock()
	vals := c.tm.Values()
	c.mu.RUnlock()
	return vals
}

// Entries returns all entries as a slice (ascending by key).
func (c *concurrentTreeMap[K, V]) Entries() []Entry[K, V] {
	c.mu.RLock()
	ents := c.tm.Entries()
	c.mu.RUnlock()
	return ents
}

// ForEach iterates over entries; stops early if action returns false (over snapshot).
func (c *concurrentTreeMap[K, V]) ForEach(action func(key K, value V) bool) {
	for k, v := range c.Seq() {
		if !action(k, v) {
			return
		}
	}
}

// Seq returns a sequence of (key, value) pairs in ascending key order (snapshot).
func (c *concurrentTreeMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		ents := c.Entries()
		for _, e := range ents {
			if !yield(e.Key, e.Value) {
				return
			}
		}
	}
}

// SeqKeys returns a sequence of keys in ascending order (snapshot).
func (c *concurrentTreeMap[K, V]) SeqKeys() iter.Seq[K] {
	return func(yield func(K) bool) {
		ks := c.Keys()
		for _, k := range ks {
			if !yield(k) {
				return
			}
		}
	}
}

// SeqValues returns a sequence of values in ascending key order (snapshot).
func (c *concurrentTreeMap[K, V]) SeqValues() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range c.Values() {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a shallow snapshot as a non-concurrent HashMap.
func (c *concurrentTreeMap[K, V]) Clone() Map[K, V] {
	// Use TreeMap to avoid requiring comparable keys.
	return c.CloneSorted()
}

// Filter returns a new map with entries satisfying predicate (snapshot).
func (c *concurrentTreeMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	out := NewTreeMap[K, V](c.tm.keyCmp)
	for k, v := range c.Seq() {
		if predicate(k, v) {
			out.Put(k, v)
		}
	}
	return out
}

// Equals reports whether both maps contain the same entries (snapshot-based).
func (c *concurrentTreeMap[K, V]) Equals(other Map[K, V], valueEq Equaler[V]) bool {
	snap := c.Clone()
	return snap.Equals(other, valueEq)
}

// GetOrCompute atomically returns existing value or computes and stores a new one.
// Returns (value, true) if computed.
func (c *concurrentTreeMap[K, V]) GetOrCompute(key K, compute func() V) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.tm.Get(key); ok {
		return v, false
	}
	v := compute()
	c.tm.Put(key, v)
	return v, true
}

// RemoveAndGet atomically removes and returns the value for key.
func (c *concurrentTreeMap[K, V]) RemoveAndGet(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.Remove(key)
}

// CompareAndSwap atomically replaces value if current equals old.
func (c *concurrentTreeMap[K, V]) CompareAndSwap(key K, oldValue, newValue V, eq Equaler[V]) bool {
	return c.ReplaceIf(key, oldValue, newValue, eq)
}

// CompareAndDelete atomically deletes entry if current value equals provided.
func (c *concurrentTreeMap[K, V]) CompareAndDelete(key K, value V, eq Equaler[V]) bool {
	return c.RemoveIf(key, value, eq)
}

// --- SortedMap extras ---

// FirstKey returns the smallest key.
func (c *concurrentTreeMap[K, V]) FirstKey() (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.FirstKey()
}

// LastKey returns the largest key.
func (c *concurrentTreeMap[K, V]) LastKey() (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.LastKey()
}

// FirstEntry returns the entry with the smallest key.
func (c *concurrentTreeMap[K, V]) FirstEntry() (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.FirstEntry()
}

// LastEntry returns the entry with the largest key.
func (c *concurrentTreeMap[K, V]) LastEntry() (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.LastEntry()
}

// PopFirst removes and returns the smallest-key entry.
func (c *concurrentTreeMap[K, V]) PopFirst() (Entry[K, V], bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.PopFirst()
}

// PopLast removes and returns the largest-key entry.
func (c *concurrentTreeMap[K, V]) PopLast() (Entry[K, V], bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tm.PopLast()
}

// FloorKey returns the greatest key <= k.
func (c *concurrentTreeMap[K, V]) FloorKey(k K) (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.FloorKey(k)
}

// CeilingKey returns the smallest key >= k.
func (c *concurrentTreeMap[K, V]) CeilingKey(k K) (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.CeilingKey(k)
}

// LowerKey returns the greatest key < k.
func (c *concurrentTreeMap[K, V]) LowerKey(k K) (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.LowerKey(k)
}

// HigherKey returns the smallest key > k.
func (c *concurrentTreeMap[K, V]) HigherKey(k K) (K, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.HigherKey(k)
}

// FloorEntry returns entry with greatest key <= k.
func (c *concurrentTreeMap[K, V]) FloorEntry(k K) (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.FloorEntry(k)
}

// CeilingEntry returns entry with smallest key >= k.
func (c *concurrentTreeMap[K, V]) CeilingEntry(k K) (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.CeilingEntry(k)
}

// LowerEntry returns entry with greatest key < k.
func (c *concurrentTreeMap[K, V]) LowerEntry(k K) (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.LowerEntry(k)
}

// HigherEntry returns entry with smallest key > k.
func (c *concurrentTreeMap[K, V]) HigherEntry(k K) (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.HigherEntry(k)
}

// collectEntries snapshots the entries produced by iterate under the read
// lock, so the caller can run user callbacks without holding it. Only the
// entries iterate visits are collected — a narrow range costs a narrow
// snapshot, not a full copy.
func (c *concurrentTreeMap[K, V]) collectEntries(iterate func(*treeMap[K, V], func(K, V) bool)) []Entry[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var buf []Entry[K, V]
	iterate(c.tm, func(k K, v V) bool {
		buf = append(buf, Entry[K, V]{Key: k, Value: v})
		return true
	})
	return buf
}

// replayEntries invokes action over ents, honoring early exit.
func replayEntries[K, V any](ents []Entry[K, V], action func(key K, value V) bool) {
	for _, e := range ents {
		if !action(e.Key, e.Value) {
			return
		}
	}
}

// Range iterates entries with keys in [from, to] ascending over snapshot.
func (c *concurrentTreeMap[K, V]) Range(from, to K, action func(key K, value V) bool) {
	replayEntries(c.collectEntries(func(tm *treeMap[K, V], yield func(K, V) bool) {
		tm.Range(from, to, yield)
	}), action)
}

// RangeSeq returns a sequence for entries with keys in [from, to] ascending (snapshot).
func (c *concurrentTreeMap[K, V]) RangeSeq(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.Range(from, to, func(k K, v V) bool { return yield(k, v) })
	}
}

// RangeFrom iterates entries with keys >= from (snapshot).
func (c *concurrentTreeMap[K, V]) RangeFrom(from K, action func(key K, value V) bool) {
	replayEntries(c.collectEntries(func(tm *treeMap[K, V], yield func(K, V) bool) {
		tm.RangeFrom(from, yield)
	}), action)
}

// RangeTo iterates entries with keys <= to (snapshot).
func (c *concurrentTreeMap[K, V]) RangeTo(to K, action func(key K, value V) bool) {
	replayEntries(c.collectEntries(func(tm *treeMap[K, V], yield func(K, V) bool) {
		tm.RangeTo(to, yield)
	}), action)
}

// Ascend iterates all entries in ascending key order over snapshot.
func (c *concurrentTreeMap[K, V]) Ascend(action func(key K, value V) bool) {
	replayEntries(c.collectEntries((*treeMap[K, V]).Ascend), action)
}

// Descend iterates all entries in descending key order over snapshot.
func (c *concurrentTreeMap[K, V]) Descend(action func(key K, value V) bool) {
	replayEntries(c.collectEntries((*treeMap[K, V]).Descend), action)
}

// AscendFrom iterates entries with keys >= pivot ascending (snapshot).
func (c *concurrentTreeMap[K, V]) AscendFrom(pivot K, action func(key K, value V) bool) {
	replayEntries(c.collectEntries(func(tm *treeMap[K, V], yield func(K, V) bool) {
		tm.AscendFrom(pivot, yield)
	}), action)
}

// DescendFrom iterates entries with keys <= pivot descending (snapshot).
func (c *concurrentTreeMap[K, V]) DescendFrom(pivot K, action func(key K, value V) bool) {
	replayEntries(c.collectEntries(func(tm *treeMap[K, V], yield func(K, V) bool) {
		tm.DescendFrom(pivot, yield)
	}), action)
}

// Reversed returns a sequence iterating in descending key order (snapshot).
func (c *concurrentTreeMap[K, V]) Reversed() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		c.Descend(func(k K, v V) bool { return yield(k, v) })
	}
}

// SubMap returns entries with keys in [from, to] (snapshot).
func (c *concurrentTreeMap[K, V]) SubMap(from, to K) SortedMap[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.SubMap(from, to)
}

// HeadMap returns entries with keys < to (or <= if inclusive) (snapshot).
func (c *concurrentTreeMap[K, V]) HeadMap(to K, inclusive bool) SortedMap[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.HeadMap(to, inclusive)
}

// TailMap returns entries with keys > from (or >= if inclusive) (snapshot).
func (c *concurrentTreeMap[K, V]) TailMap(from K, inclusive bool) SortedMap[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.TailMap(from, inclusive)
}

// RankOfKey returns the 0-based rank of key, or -1 if not present.
func (c *concurrentTreeMap[K, V]) RankOfKey(key K) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.RankOfKey(key)
}

// GetByRank returns the entry at rank, or (Entry{}, false).
func (c *concurrentTreeMap[K, V]) GetByRank(rank int) (Entry[K, V], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.GetByRank(rank)
}

// CloneSorted returns a shallow copy as SortedMap snapshot.
func (c *concurrentTreeMap[K, V]) CloneSorted() SortedMap[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tm.CloneSorted()
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes entries in ascending key order.
//
// NOTE: The comparator is NOT serialized. Decode into a map constructed with
// NewConcurrentTreeMap.
func (c *concurrentTreeMap[K, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializableMap[K, V]{Entries: c.snapshotEntries()})
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes into the receiver using its existing comparator, so construct
// the map with NewConcurrentTreeMap before decoding.
func (c *concurrentTreeMap[K, V]) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tm == nil {
		return errors.New("unmarshal concurrent treemap: no comparator; construct the map with NewConcurrentTreeMap before decoding")
	}
	return c.tm.UnmarshalJSON(data)
}

// snapshotEntries copies the entries in ascending key order under the read
// lock, so encoding can run without holding it.
func (c *concurrentTreeMap[K, V]) snapshotEntries() []serializableEntry[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entries := make([]serializableEntry[K, V], 0, c.tm.Size())
	for k, v := range c.tm.Seq() {
		entries = append(entries, serializableEntry[K, V]{Key: k, Value: v})
	}
	return entries
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (c *concurrentTreeMap[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, serializableMap[K, V]{Entries: c.snapshotEntries()})
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
// Same comparator contract as UnmarshalJSON.
func (c *concurrentTreeMap[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tm == nil {
		return errors.New("unmarshal concurrent treemap: no comparator; construct the map with NewConcurrentTreeMap before decoding")
	}
	return c.tm.UnmarshalJSONFrom(dec)
}

// GobEncode implements gob.GobEncoder.
// Serializes entries in ascending key order.
//
// NOTE: The comparator is NOT serialized. Decode into a map constructed with
// NewConcurrentTreeMap.
func (c *concurrentTreeMap[K, V]) GobEncode() ([]byte, error) {
	entries := c.snapshotEntries()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entries); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes into the receiver using its existing comparator, so construct
// the map with NewConcurrentTreeMap before decoding.
func (c *concurrentTreeMap[K, V]) GobDecode(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tm == nil {
		return errors.New("unmarshal concurrent treemap: no comparator; construct the map with NewConcurrentTreeMap before decoding")
	}
	return c.tm.GobDecode(data)
}

// Conformance.
var (
	_ ConcurrentSortedMap[int, string] = (*concurrentTreeMap[int, string])(nil)
)
