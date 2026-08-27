package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"

	"github.com/puzpuzpuz/xsync/v3"
)

// concurrentHashMap is a thread-safe hash map backed by xsync.MapOf[K,V].
// Single-key operations are atomic. Bulk/scan operations are not atomic as a whole.
type concurrentHashMap[K comparable, V any] struct {
	m *xsync.MapOf[K, V]
}

// NewConcurrentHashMap creates an empty concurrent map.
func NewConcurrentHashMap[K comparable, V any]() ConcurrentMap[K, V] {
	return &concurrentHashMap[K, V]{m: xsync.NewMapOf[K, V]()}
}

// NewConcurrentHashMapFrom creates a concurrent map copying entries from a standard map.
func NewConcurrentHashMapFrom[K comparable, V any](src map[K]V) ConcurrentMap[K, V] {
	m := &concurrentHashMap[K, V]{m: xsync.NewMapOf[K, V]()}
	for k, v := range src {
		m.m.Store(k, v)
	}
	return m
}

// computeAbort is deferred inside every callback handed to xsync's Compute
// that runs a user function: it records a panic from that function and
// rewrites the callback result so the map is left unchanged (write the old
// value back when the key exists, delete the absent key otherwise — both
// no-ops). The internal bucket lock is then released normally and the panic
// is re-raised by the caller after Compute returns.
func computeAbort[V any](panicked *bool, panicVal *any, newValue *V, del *bool, prev V, loaded bool) {
	if p := recover(); p != nil {
		*panicked, *panicVal = true, p
		*newValue, *del = prev, !loaded
	}
}

// Size returns an approximate number of entries.
func (m *concurrentHashMap[K, V]) Size() int { return m.m.Size() }

// IsEmpty reports whether the map is empty (approximate under concurrency).
func (m *concurrentHashMap[K, V]) IsEmpty() bool    { return m.Size() == 0 }
func (m *concurrentHashMap[K, V]) IsNotEmpty() bool { return !m.IsEmpty() }

// Clear removes all entries.
func (m *concurrentHashMap[K, V]) Clear() { m.m.Clear() }

// String returns a concise representation (unordered).
func (m *concurrentHashMap[K, V]) String() string {
	return formatMap("concurrentHashMap", m.Seq())
}

// Put associates value with key. Returns (oldValue, true) if key existed.
func (m *concurrentHashMap[K, V]) Put(key K, value V) (V, bool) {
	var (
		old     V
		existed bool
	)
	m.m.Compute(key, func(prev V, loaded bool) (V, bool) {
		if loaded {
			old = prev
			existed = true
		}
		return value, false // set new value
	})
	return old, existed
}

// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
func (m *concurrentHashMap[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	v, loaded := m.m.LoadOrStore(key, value)
	// loaded true: existed (not inserted)
	return v, !loaded
}

// PutAll copies all entries from other into this map.
func (m *concurrentHashMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq() {
		m.m.Store(k, v)
	}
}

// PutSeq copies entries from a Seq2. Returns number of unique keys touched.
func (m *concurrentHashMap[K, V]) PutSeq(seq iter.Seq2[K, V]) int {
	seen := make(map[K]struct{})
	changed := 0
	seq(func(k K, v V) bool {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			changed++
		}
		m.m.Store(k, v)
		return true
	})
	return changed
}

// Get returns (value, true) if key present; otherwise (zero, false).
func (m *concurrentHashMap[K, V]) Get(key K) (V, bool) {
	return m.m.Load(key)
}

// GetOrDefault returns value for key or defaultValue if absent.
func (m *concurrentHashMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if v, ok := m.m.Load(key); ok {
		return v
	}
	return defaultValue
}

// Remove deletes key. Returns (oldValue, true) if key existed.
func (m *concurrentHashMap[K, V]) Remove(key K) (V, bool) {
	return m.m.LoadAndDelete(key)
}

// RemoveIf deletes only if (key, value) matches. Returns true if removed.
// The eq function runs while an internal bucket lock is held: it must not
// call back into the same map. A panic in eq is propagated after the lock
// is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) RemoveIf(key K, value V, eq Equaler[V]) bool {
	var (
		removed  bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (newValue V, del bool) {
		defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
		// Deleting an absent key is a no-op; returning del=false would
		// insert prev (a zero value) for the key.
		if !loaded {
			return prev, true
		}
		if eq(prev, value) {
			removed = true
			var zero V
			return zero, true // delete
		}
		return prev, false // keep
	})
	if panicked {
		panic(panicVal)
	}
	return removed
}

// ContainsKey reports whether key exists.
func (m *concurrentHashMap[K, V]) ContainsKey(key K) bool {
	_, ok := m.m.Load(key)
	return ok
}

// ContainsValue reports whether value exists (O(n)).
func (m *concurrentHashMap[K, V]) ContainsValue(value V, eq Equaler[V]) bool {
	found := false
	m.m.Range(func(_ K, v V) bool {
		if eq(v, value) {
			found = true
			return false
		}
		return true
	})
	return found
}

// RemoveAll removes all specified keys. Returns count removed.
func (m *concurrentHashMap[K, V]) RemoveAll(keys ...K) int {
	removed := 0
	for _, k := range keys {
		if _, ok := m.m.LoadAndDelete(k); ok {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes keys from the sequence. Returns count removed.
func (m *concurrentHashMap[K, V]) RemoveSeq(seq iter.Seq[K]) int {
	removed := 0
	for k := range seq {
		if _, ok := m.m.LoadAndDelete(k); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes entries where predicate returns true. Returns count removed.
func (m *concurrentHashMap[K, V]) RemoveFunc(predicate func(key K, value V) bool) int {
	dels := make([]K, 0, m.Size())
	m.m.Range(func(k K, v V) bool {
		if predicate(k, v) {
			dels = append(dels, k)
		}
		return true
	})
	count := 0
	for _, k := range dels {
		if _, ok := m.m.LoadAndDelete(k); ok {
			count++
		}
	}
	return count
}

// Compute recomputes mapping for key. If keep==false, the key is removed.
// The remapping function runs while an internal bucket lock is held: it must
// not call back into the same map, and should be short. A panic in remapping
// is propagated after the lock is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool) {
	var (
		result   V
		ok       bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (newValue V, del bool) {
		defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
		newVal, keep := remapping(key, prev, loaded)
		if !keep {
			ok = false
			var zero V
			result = zero
			return zero, true // delete
		}
		ok = true
		result = newVal
		return newVal, false
	})
	if panicked {
		panic(panicVal)
	}
	return result, ok
}

// ComputeIfAbsent computes and stores value if key is absent.
// The mapping function runs while an internal bucket lock is held: it must
// not call back into the same map, and should be short. A panic in mapping
// is propagated after the lock is released and stores nothing.
func (m *concurrentHashMap[K, V]) ComputeIfAbsent(key K, mapping func(key K) V) V {
	v, _ := m.loadOrTryCompute(key, func() V { return mapping(key) })
	return v
}

// loadOrTryCompute wraps xsync's LoadOrTryCompute so a panicking compute
// cancels the store, is re-raised only after the internal bucket lock has
// been released, and can never leave that lock held.
func (m *concurrentHashMap[K, V]) loadOrTryCompute(key K, compute func() V) (V, bool) {
	var (
		panicked bool
		panicVal any
	)
	v, loaded := m.m.LoadOrTryCompute(key, func() (newValue V, cancel bool) {
		defer func() {
			if p := recover(); p != nil {
				panicked, panicVal = true, p
				cancel = true
			}
		}()
		return compute(), false
	})
	if panicked {
		panic(panicVal)
	}
	return v, loaded
}

// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
// The remapping function runs while an internal bucket lock is held: it must
// not call back into the same map, and should be short. A panic in remapping
// is propagated after the lock is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool) {
	var (
		out      V
		ok       bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (newValue V, del bool) {
		if !loaded {
			ok = false
			// Deleting an absent key is a no-op; returning del=false would
			// insert prev (a zero value) for the key.
			return prev, true
		}
		defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
		newVal, keep := remapping(key, prev)
		if !keep {
			ok = false
			var zero V
			out = zero
			return zero, true // delete
		}
		ok = true
		out = newVal
		return newVal, false
	})
	if panicked {
		panic(panicVal)
	}
	return out, ok
}

// Merge merges value with existing. If keep==false, removes key.
// The remapping function runs while an internal bucket lock is held: it must
// not call back into the same map, and should be short. A panic in remapping
// is propagated after the lock is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool) {
	var (
		out      V
		ok       bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (newValue V, del bool) {
		if !loaded {
			ok = true
			out = value
			return value, false
		}
		defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
		merged, keep := remapping(prev, value)
		if !keep {
			ok = false
			var zero V
			out = zero
			return zero, true // delete
		}
		ok = true
		out = merged
		return out, false
	})
	if panicked {
		panic(panicVal)
	}
	return out, ok
}

// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
func (m *concurrentHashMap[K, V]) Replace(key K, value V) (V, bool) {
	var (
		old      V
		replaced bool
	)
	m.m.Compute(key, func(prev V, loaded bool) (V, bool) {
		// Deleting an absent key is a no-op; returning del=false would
		// insert prev (a zero value) for the key.
		if !loaded {
			return prev, true
		}
		old = prev
		replaced = true
		return value, false
	})
	return old, replaced
}

// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced.
// The eq function runs while an internal bucket lock is held: it must not
// call back into the same map. A panic in eq is propagated after the lock
// is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool {
	var (
		ok       bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (nv V, del bool) {
		defer computeAbort(&panicked, &panicVal, &nv, &del, prev, loaded)
		// Deleting an absent key is a no-op; returning del=false would
		// insert prev (a zero value) for the key.
		if !loaded {
			return prev, true
		}
		if eq(prev, oldValue) {
			ok = true
			return newValue, false
		}
		return prev, false
	})
	if panicked {
		panic(panicVal)
	}
	return ok
}

// ReplaceAll replaces each value with function(key, value).
// The function runs while an internal bucket lock is held: it must not
// call back into the same map. A panic in it is propagated after the lock
// is released and leaves the entry being replaced unchanged.
func (m *concurrentHashMap[K, V]) ReplaceAll(function func(key K, value V) V) {
	var (
		panicked bool
		panicVal any
	)
	m.m.Range(func(k K, _ V) bool {
		m.m.Compute(k, func(prev V, loaded bool) (newValue V, del bool) {
			defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
			// The key can vanish between Range and Compute; deleting an
			// absent key is a no-op, del=false would insert a zero value.
			if !loaded {
				return prev, true
			}
			return function(k, prev), false
		})
		return !panicked
	})
	if panicked {
		panic(panicVal)
	}
}

// Keys returns all keys as a slice.
func (m *concurrentHashMap[K, V]) Keys() []K {
	out := make([]K, 0, m.Size())
	m.m.Range(func(k K, _ V) bool {
		out = append(out, k)
		return true
	})
	return out
}

// Values returns all values as a slice.
func (m *concurrentHashMap[K, V]) Values() []V {
	out := make([]V, 0, m.Size())
	m.m.Range(func(_ K, v V) bool {
		out = append(out, v)
		return true
	})
	return out
}

// Entries returns all entries as a slice.
func (m *concurrentHashMap[K, V]) Entries() []Entry[K, V] {
	out := make([]Entry[K, V], 0, m.Size())
	m.m.Range(func(k K, v V) bool {
		out = append(out, Entry[K, V]{Key: k, Value: v})
		return true
	})
	return out
}

// ForEach iterates over entries; stops early if action returns false.
func (m *concurrentHashMap[K, V]) ForEach(action func(key K, value V) bool) {
	m.m.Range(func(k K, v V) bool {
		return action(k, v)
	})
}

// Seq returns a sequence of (key, value) pairs (unordered).
func (m *concurrentHashMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		m.m.Range(func(k K, v V) bool {
			return yield(k, v)
		})
	}
}

// SeqKeys returns a sequence of keys (unordered).
func (m *concurrentHashMap[K, V]) SeqKeys() iter.Seq[K] {
	return func(yield func(K) bool) {
		m.m.Range(func(k K, _ V) bool {
			return yield(k)
		})
	}
}

// SeqValues returns a sequence of values (unordered).
func (m *concurrentHashMap[K, V]) SeqValues() iter.Seq[V] {
	return func(yield func(V) bool) {
		m.m.Range(func(_ K, v V) bool {
			return yield(v)
		})
	}
}

// Clone returns a shallow copy (as a non-concurrent HashMap snapshot).
func (m *concurrentHashMap[K, V]) Clone() Map[K, V] {
	cp := NewHashMapWithCapacity[K, V](m.Size())
	m.m.Range(func(k K, v V) bool {
		cp.Put(k, v)
		return true
	})
	return cp
}

// Filter returns a new map with entries satisfying predicate (non-concurrent snapshot).
func (m *concurrentHashMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	out := NewHashMap[K, V]()
	m.m.Range(func(k K, v V) bool {
		if predicate(k, v) {
			out.Put(k, v)
		}
		return true
	})
	return out
}

// Equals reports whether both maps contain the same entries (snapshot-based).
func (m *concurrentHashMap[K, V]) Equals(other Map[K, V], valueEq Equaler[V]) bool {
	// Snapshot this map and compare to other.
	snap := NewHashMap[K, V]()
	m.m.Range(func(k K, v V) bool {
		snap.Put(k, v)
		return true
	})
	return snap.Equals(other, valueEq)
}

// GetOrCompute atomically returns existing value or computes and stores a new one.
// Returns (value, true) if computed (i.e., absent before).
// The compute function runs while an internal bucket lock is held: it must
// not call back into the same map, and should be short. A panic in compute
// is propagated after the lock is released and stores nothing.
func (m *concurrentHashMap[K, V]) GetOrCompute(key K, compute func() V) (V, bool) {
	v, loaded := m.loadOrTryCompute(key, compute)
	return v, !loaded
}

// RemoveAndGet atomically removes and returns the value for key.
func (m *concurrentHashMap[K, V]) RemoveAndGet(key K) (V, bool) {
	return m.m.LoadAndDelete(key)
}

// CompareAndSwap atomically replaces value if current equals old.
// The eq function runs while an internal bucket lock is held: it must not
// call back into the same map. A panic in eq is propagated after the lock
// is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) CompareAndSwap(key K, oldValue, newValue V, eq Equaler[V]) bool {
	var (
		swapped  bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (nv V, del bool) {
		defer computeAbort(&panicked, &panicVal, &nv, &del, prev, loaded)
		// Deleting an absent key is a no-op; returning del=false would
		// insert prev (a zero value) for the key.
		if !loaded {
			return prev, true
		}
		if eq(prev, oldValue) {
			swapped = true
			return newValue, false
		}
		return prev, false
	})
	if panicked {
		panic(panicVal)
	}
	return swapped
}

// CompareAndDelete atomically deletes entry if current value equals provided.
// The eq function runs while an internal bucket lock is held: it must not
// call back into the same map. A panic in eq is propagated after the lock
// is released and leaves the map unchanged.
func (m *concurrentHashMap[K, V]) CompareAndDelete(key K, value V, eq Equaler[V]) bool {
	var (
		deleted  bool
		panicked bool
		panicVal any
	)
	m.m.Compute(key, func(prev V, loaded bool) (newValue V, del bool) {
		defer computeAbort(&panicked, &panicVal, &newValue, &del, prev, loaded)
		// Deleting an absent key is a no-op; returning del=false would
		// insert prev (a zero value) for the key.
		if !loaded {
			return prev, true
		}
		if eq(prev, value) {
			deleted = true
			var zero V
			return zero, true
		}
		return prev, false
	})
	if panicked {
		panic(panicVal)
	}
	return deleted
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes a snapshot of the map as a JSON object.
// NOTE: Provides snapshot consistency - concurrent modifications
// during serialization may not be reflected.
func (m *concurrentHashMap[K, V]) MarshalJSON() ([]byte, error) {
	// Build a standard Go map from the concurrent map
	snapshot := make(map[K]V)
	m.m.Range(func(key K, value V) bool {
		snapshot[key] = value
		return true
	})
	return json.Marshal(snapshot)
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON object.
func (m *concurrentHashMap[K, V]) UnmarshalJSON(data []byte) error {
	var snapshot map[K]V
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return err
	}
	// Refill the existing backing map: replacing the m.m pointer would race
	// with concurrent readers of the receiver.
	m.m.Clear()
	for key, value := range snapshot {
		m.m.Store(key, value)
	}
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the map.
func (m *concurrentHashMap[K, V]) GobEncode() ([]byte, error) {
	snapshot := make(map[K]V)
	m.m.Range(func(key K, value V) bool {
		snapshot[key] = value
		return true
	})

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(snapshot); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (m *concurrentHashMap[K, V]) GobDecode(data []byte) error {
	var snapshot map[K]V
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&snapshot); err != nil {
		return err
	}
	// Refill the existing backing map: replacing the m.m pointer would race
	// with concurrent readers of the receiver.
	m.m.Clear()
	for key, value := range snapshot {
		m.m.Store(key, value)
	}
	return nil
}

// Conformance.
var (
	_ ConcurrentMap[int, string] = (*concurrentHashMap[int, string])(nil)
)
