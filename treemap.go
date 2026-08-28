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

	"github.com/tidwall/btree"
)

// mapEntry is an internal entry used by TreeMap's btree.
type mapEntry[K any, V any] struct {
	key   K
	value V
}

// treeMap is a sorted map backed by a B-Tree on keys.
// Ordering is defined by the provided Comparator[K].
// Not concurrent-safe.
type treeMap[K any, V any] struct {
	bt     *btree.BTreeG[mapEntry[K, V]]
	keyCmp Comparator[K]
}

func newTreeMap[K any, V any](cmpK Comparator[K]) *treeMap[K, V] {
	if cmpK == nil {
		panic("NewTreeMap: key comparator must not be nil")
	}
	lt := func(a, b mapEntry[K, V]) bool { return cmpK(a.key, b.key) < 0 }
	return &treeMap[K, V]{bt: btree.NewBTreeG(lt), keyCmp: cmpK}
}

// NewTreeMap creates an empty SortedMap with a custom key comparator.
func NewTreeMap[K any, V any](cmpK Comparator[K]) SortedMap[K, V] {
	return newTreeMap[K, V](cmpK)
}

// NewTreeMapOrdered creates an empty TreeMap for Ordered keys.
func NewTreeMapOrdered[K Ordered, V any]() SortedMap[K, V] {
	return newTreeMap[K, V](cmp.Compare)
}

// NewTreeMapFrom creates a TreeMap from a standard Go map (copying entries).
func NewTreeMapFrom[K comparable, V any](cmpK Comparator[K], m map[K]V) SortedMap[K, V] {
	tm := newTreeMap[K, V](cmpK)
	for k, v := range m {
		tm.bt.Set(mapEntry[K, V]{key: k, value: v})
	}
	return tm
}

// Size returns the number of entries.
func (t *treeMap[K, V]) Size() int { return t.bt.Len() }

// IsEmpty reports whether the map is empty.
func (t *treeMap[K, V]) IsEmpty() bool    { return t.bt.Len() == 0 }
func (t *treeMap[K, V]) IsNotEmpty() bool { return !t.IsEmpty() }

// Clear removes all entries.
func (t *treeMap[K, V]) Clear() { t.bt.Clear() }

// String returns a concise representation.
func (t *treeMap[K, V]) String() string {
	return formatMap("treeMap", t.Seq())
}

// Put associates value with key. Returns (oldValue, true) if key existed.
func (t *treeMap[K, V]) Put(key K, value V) (V, bool) {
	entry := mapEntry[K, V]{key: key, value: value}
	prev, replaced := t.bt.Set(entry)
	if replaced {
		return prev.value, true
	}
	var zero V
	return zero, false
}

// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
func (t *treeMap[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		return me.value, false
	}
	t.bt.Set(mapEntry[K, V]{key: key, value: value})
	return value, true
}

// PutAll copies all entries from other into this map.
func (t *treeMap[K, V]) PutAll(other Map[K, V]) {
	for k, v := range other.Seq() {
		t.bt.Set(mapEntry[K, V]{key: k, value: v})
	}
}

// PutSeq copies entries from a sequence. Returns number of keys changed (unique keys touched).
func (t *treeMap[K, V]) PutSeq(seq iter.Seq2[K, V]) int {
	changed := 0
	seen := newTreeSet(t.keyCmp)
	seq(func(k K, v V) bool {
		if !seen.Contains(k) {
			seen.Add(k)
			changed++
		}
		t.bt.Set(mapEntry[K, V]{key: k, value: v})
		return true
	})
	return changed
}

// Get returns (value, true) if key present; otherwise (zero, false).
func (t *treeMap[K, V]) Get(key K) (V, bool) {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		return me.value, true
	}
	var zero V
	return zero, false
}

// GetOrDefault returns value for key or defaultValue if absent.
func (t *treeMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		return me.value
	}
	return defaultValue
}

// Remove deletes key. Returns (oldValue, true) if key existed.
func (t *treeMap[K, V]) Remove(key K) (V, bool) {
	if prev, ok := t.bt.Delete(mapEntry[K, V]{key: key}); ok {
		return prev.value, true
	}
	var zero V
	return zero, false
}

// RemoveIf deletes only if (key, value) matches. Returns true if removed.
func (t *treeMap[K, V]) RemoveIf(key K, value V, eq Equaler[V]) bool {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok && eq(me.value, value) {
		t.bt.Delete(me)
		return true
	}
	return false
}

// ContainsKey reports whether key exists.
func (t *treeMap[K, V]) ContainsKey(key K) bool {
	_, ok := t.bt.Get(mapEntry[K, V]{key: key})
	return ok
}

// ContainsValue reports whether value exists (O(n)).
func (t *treeMap[K, V]) ContainsValue(value V, eq Equaler[V]) bool {
	found := false
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		if eq(me.value, value) {
			found = true
			return false
		}
		return true
	})
	return found
}

// RemoveAll removes all specified keys. Returns count removed.
func (t *treeMap[K, V]) RemoveAll(keys ...K) int {
	removed := 0
	for _, k := range keys {
		if _, ok := t.bt.Delete(mapEntry[K, V]{key: k}); ok {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes keys from the sequence. Returns count removed.
func (t *treeMap[K, V]) RemoveSeq(seq iter.Seq[K]) int {
	removed := 0
	for k := range seq {
		if _, ok := t.bt.Delete(mapEntry[K, V]{key: k}); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes entries where predicate returns true. Returns count removed.
func (t *treeMap[K, V]) RemoveFunc(predicate func(key K, value V) bool) int {
	dels := make([]K, 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		if predicate(me.key, me.value) {
			dels = append(dels, me.key)
		}
		return true
	})
	for _, k := range dels {
		t.bt.Delete(mapEntry[K, V]{key: k})
	}
	return len(dels)
}

// Compute recomputes mapping for key. If keep==false, the key is removed.
func (t *treeMap[K, V]) Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool) {
	me, exists := t.bt.Get(mapEntry[K, V]{key: key})
	old := me.value
	newVal, keep := remapping(key, old, exists)
	if !keep {
		if exists {
			t.bt.Delete(me)
		}
		var zero V
		return zero, false
	}
	t.bt.Set(mapEntry[K, V]{key: key, value: newVal})
	return newVal, true
}

// ComputeIfAbsent computes and stores value if key is absent.
func (t *treeMap[K, V]) ComputeIfAbsent(key K, mapping func(key K) V) V {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		return me.value
	}
	v := mapping(key)
	t.bt.Set(mapEntry[K, V]{key: key, value: v})
	return v
}

// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
func (t *treeMap[K, V]) ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool) {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		newVal, keep := remapping(key, me.value)
		if !keep {
			t.bt.Delete(me)
			var zero V
			return zero, false
		}
		t.bt.Set(mapEntry[K, V]{key: key, value: newVal})
		return newVal, true
	}
	var zero V
	return zero, false
}

// Merge merges value with existing. If keep==false, removes key.
func (t *treeMap[K, V]) Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool) {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		merged, keep := remapping(me.value, value)
		if !keep {
			t.bt.Delete(me)
			var zero V
			return zero, false
		}
		t.bt.Set(mapEntry[K, V]{key: key, value: merged})
		return merged, true
	}
	t.bt.Set(mapEntry[K, V]{key: key, value: value})
	return value, true
}

// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
func (t *treeMap[K, V]) Replace(key K, value V) (V, bool) {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok {
		t.bt.Set(mapEntry[K, V]{key: key, value: value})
		return me.value, true
	}
	var zero V
	return zero, false
}

// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced.
func (t *treeMap[K, V]) ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool {
	if me, ok := t.bt.Get(mapEntry[K, V]{key: key}); ok && eq(me.value, oldValue) {
		t.bt.Set(mapEntry[K, V]{key: key, value: newValue})
		return true
	}
	return false
}

// ReplaceAll replaces each value with function(key, value).
func (t *treeMap[K, V]) ReplaceAll(function func(key K, value V) V) {
	changes := make([]mapEntry[K, V], 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		changes = append(changes, mapEntry[K, V]{key: me.key, value: function(me.key, me.value)})
		return true
	})
	for _, ch := range changes {
		t.bt.Set(ch)
	}
}

// Keys returns all keys as a slice.
func (t *treeMap[K, V]) Keys() []K {
	out := make([]K, 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		out = append(out, me.key)
		return true
	})
	return out
}

// Values returns all values as a slice.
func (t *treeMap[K, V]) Values() []V {
	out := make([]V, 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		out = append(out, me.value)
		return true
	})
	return out
}

// Entries returns all entries as a slice.
func (t *treeMap[K, V]) Entries() []Entry[K, V] {
	out := make([]Entry[K, V], 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		out = append(out, Entry[K, V]{Key: me.key, Value: me.value})
		return true
	})
	return out
}

// ForEach iterates over entries ascending; stops early if action returns false.
func (t *treeMap[K, V]) ForEach(action func(key K, value V) bool) {
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		return action(me.key, me.value)
	})
}

// Seq returns a sequence of (key, value) pairs ascending.
func (t *treeMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		t.bt.Scan(func(me mapEntry[K, V]) bool {
			return yield(me.key, me.value)
		})
	}
}

// SeqKeys returns a sequence of keys ascending.
func (t *treeMap[K, V]) SeqKeys() iter.Seq[K] {
	return func(yield func(K) bool) {
		t.bt.Scan(func(me mapEntry[K, V]) bool {
			return yield(me.key)
		})
	}
}

// SeqValues returns a sequence of values ascending.
func (t *treeMap[K, V]) SeqValues() iter.Seq[V] {
	return func(yield func(V) bool) {
		t.bt.Scan(func(me mapEntry[K, V]) bool {
			return yield(me.value)
		})
	}
}

// Clone returns a shallow copy (as Map).
func (t *treeMap[K, V]) Clone() Map[K, V] {
	return t.CloneSorted()
}

// Filter returns a new map with entries satisfying predicate.
func (t *treeMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	out := newTreeMap[K, V](t.keyCmp)
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		if predicate(me.key, me.value) {
			out.bt.Set(me)
		}
		return true
	})
	return out
}

// Equals reports whether both maps contain the same entries.
func (t *treeMap[K, V]) Equals(other Map[K, V], valueEq Equaler[V]) bool {
	if t.Size() != other.Size() {
		return false
	}
	eq := true
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		ov, ok := other.Get(me.key)
		if !ok || !valueEq(me.value, ov) {
			eq = false
			return false
		}
		return true
	})
	return eq
}

// FirstKey returns the smallest key.
func (t *treeMap[K, V]) FirstKey() (K, bool) {
	me, ok := t.bt.Min()
	return me.key, ok
}

// LastKey returns the largest key.
func (t *treeMap[K, V]) LastKey() (K, bool) {
	me, ok := t.bt.Max()
	return me.key, ok
}

// FirstEntry returns the entry with the smallest key.
func (t *treeMap[K, V]) FirstEntry() (Entry[K, V], bool) {
	me, ok := t.bt.Min()
	if !ok {
		return Entry[K, V]{}, false
	}
	return Entry[K, V]{Key: me.key, Value: me.value}, true
}

// LastEntry returns the entry with the largest key.
func (t *treeMap[K, V]) LastEntry() (Entry[K, V], bool) {
	me, ok := t.bt.Max()
	if !ok {
		return Entry[K, V]{}, false
	}
	return Entry[K, V]{Key: me.key, Value: me.value}, true
}

// PopFirst removes and returns the smallest-key entry.
func (t *treeMap[K, V]) PopFirst() (Entry[K, V], bool) {
	me, ok := t.bt.Min()
	if !ok {
		return Entry[K, V]{}, false
	}
	t.bt.Delete(me)
	return Entry[K, V]{Key: me.key, Value: me.value}, true
}

// PopLast removes and returns the largest-key entry.
func (t *treeMap[K, V]) PopLast() (Entry[K, V], bool) {
	me, ok := t.bt.Max()
	if !ok {
		return Entry[K, V]{}, false
	}
	t.bt.Delete(me)
	return Entry[K, V]{Key: me.key, Value: me.value}, true
}

// FloorKey returns the greatest key <= k.
func (t *treeMap[K, V]) FloorKey(k K) (K, bool) {
	var res mapEntry[K, V]
	found := false
	t.bt.Descend(mapEntry[K, V]{key: k}, func(me mapEntry[K, V]) bool {
		res = me
		found = true
		return false
	})
	return res.key, found
}

// CeilingKey returns the smallest key >= k.
func (t *treeMap[K, V]) CeilingKey(k K) (K, bool) {
	var res mapEntry[K, V]
	found := false
	t.bt.Ascend(mapEntry[K, V]{key: k}, func(me mapEntry[K, V]) bool {
		res = me
		found = true
		return false
	})
	return res.key, found
}

// LowerKey returns the greatest key < k.
func (t *treeMap[K, V]) LowerKey(k K) (K, bool) {
	var res mapEntry[K, V]
	found := false
	t.bt.Descend(mapEntry[K, V]{key: k}, func(me mapEntry[K, V]) bool {
		if t.keyCmp(me.key, k) < 0 {
			res = me
			found = true
			return false
		}
		return true
	})
	return res.key, found
}

// HigherKey returns the smallest key > k.
func (t *treeMap[K, V]) HigherKey(k K) (K, bool) {
	var res mapEntry[K, V]
	found := false
	t.bt.Ascend(mapEntry[K, V]{key: k}, func(me mapEntry[K, V]) bool {
		if t.keyCmp(me.key, k) > 0 {
			res = me
			found = true
			return false
		}
		return true
	})
	return res.key, found
}

// FloorEntry returns the entry with greatest key <= k.
func (t *treeMap[K, V]) FloorEntry(k K) (Entry[K, V], bool) {
	if key, ok := t.FloorKey(k); ok {
		v, _ := t.Get(key)
		return Entry[K, V]{Key: key, Value: v}, true
	}
	return Entry[K, V]{}, false
}

// CeilingEntry returns the entry with smallest key >= k.
func (t *treeMap[K, V]) CeilingEntry(k K) (Entry[K, V], bool) {
	if key, ok := t.CeilingKey(k); ok {
		v, _ := t.Get(key)
		return Entry[K, V]{Key: key, Value: v}, true
	}
	return Entry[K, V]{}, false
}

// LowerEntry returns the entry with greatest key < k.
func (t *treeMap[K, V]) LowerEntry(k K) (Entry[K, V], bool) {
	if key, ok := t.LowerKey(k); ok {
		v, _ := t.Get(key)
		return Entry[K, V]{Key: key, Value: v}, true
	}
	return Entry[K, V]{}, false
}

// HigherEntry returns the entry with smallest key > k.
func (t *treeMap[K, V]) HigherEntry(k K) (Entry[K, V], bool) {
	if key, ok := t.HigherKey(k); ok {
		v, _ := t.Get(key)
		return Entry[K, V]{Key: key, Value: v}, true
	}
	return Entry[K, V]{}, false
}

// Range iterates entries with keys in [from, to] ascending.
func (t *treeMap[K, V]) Range(from, to K, action func(key K, value V) bool) {
	if t.keyCmp(from, to) > 0 {
		return
	}
	t.bt.Ascend(mapEntry[K, V]{key: from}, func(me mapEntry[K, V]) bool {
		if t.keyCmp(me.key, to) > 0 {
			return false
		}
		return action(me.key, me.value)
	})
}

// RangeFrom iterates entries with keys >= from.
func (t *treeMap[K, V]) RangeFrom(from K, action func(key K, value V) bool) {
	t.bt.Ascend(mapEntry[K, V]{key: from}, func(me mapEntry[K, V]) bool {
		return action(me.key, me.value)
	})
}

// RangeTo iterates entries with keys <= to.
func (t *treeMap[K, V]) RangeTo(to K, action func(key K, value V) bool) {
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		if t.keyCmp(me.key, to) > 0 {
			return false
		}
		return action(me.key, me.value)
	})
}

// RangeSeq returns a sequence for entries with keys in [from, to] ascending.
func (t *treeMap[K, V]) RangeSeq(from, to K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if t.keyCmp(from, to) > 0 {
			return
		}
		t.bt.Ascend(mapEntry[K, V]{key: from}, func(me mapEntry[K, V]) bool {
			if t.keyCmp(me.key, to) > 0 {
				return false
			}
			return yield(me.key, me.value)
		})
	}
}

// Ascend iterates all entries in ascending key order.
func (t *treeMap[K, V]) Ascend(action func(key K, value V) bool) {
	t.bt.Scan(func(me mapEntry[K, V]) bool { return action(me.key, me.value) })
}

// Descend iterates all entries in descending key order.
func (t *treeMap[K, V]) Descend(action func(key K, value V) bool) {
	t.bt.Reverse(func(me mapEntry[K, V]) bool { return action(me.key, me.value) })
}

// AscendFrom iterates entries with keys >= pivot ascending.
func (t *treeMap[K, V]) AscendFrom(pivot K, action func(key K, value V) bool) {
	t.bt.Ascend(mapEntry[K, V]{key: pivot}, func(me mapEntry[K, V]) bool { return action(me.key, me.value) })
}

// DescendFrom iterates entries with keys <= pivot descending.
func (t *treeMap[K, V]) DescendFrom(pivot K, action func(key K, value V) bool) {
	t.bt.Descend(mapEntry[K, V]{key: pivot}, func(me mapEntry[K, V]) bool { return action(me.key, me.value) })
}

// Reversed returns a sequence iterating in descending key order.
func (t *treeMap[K, V]) Reversed() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		t.bt.Reverse(func(me mapEntry[K, V]) bool { return yield(me.key, me.value) })
	}
}

// SubMap returns entries with keys in [from, to].
func (t *treeMap[K, V]) SubMap(from, to K) SortedMap[K, V] {
	out := newTreeMap[K, V](t.keyCmp)
	if t.keyCmp(from, to) > 0 {
		return out
	}
	t.bt.Ascend(mapEntry[K, V]{key: from}, func(me mapEntry[K, V]) bool {
		if t.keyCmp(me.key, to) > 0 {
			return false
		}
		out.bt.Set(me)
		return true
	})
	return out
}

// HeadMap returns entries with keys < to (or <= if inclusive).
func (t *treeMap[K, V]) HeadMap(to K, inclusive bool) SortedMap[K, V] {
	out := newTreeMap[K, V](t.keyCmp)
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		c := t.keyCmp(me.key, to)
		if c < 0 || (inclusive && c == 0) {
			out.bt.Set(me)
			return true
		}
		return false // past the bound: every later key is larger
	})
	return out
}

// TailMap returns entries with keys > from (or >= if inclusive).
func (t *treeMap[K, V]) TailMap(from K, inclusive bool) SortedMap[K, V] {
	out := newTreeMap[K, V](t.keyCmp)
	t.bt.Ascend(mapEntry[K, V]{key: from}, func(me mapEntry[K, V]) bool {
		c := t.keyCmp(me.key, from)
		if c > 0 || (inclusive && c == 0) {
			out.bt.Set(me)
		}
		return true
	})
	return out
}

// RankOfKey returns the 0-based rank of key, or -1 if not present.
// O(log² n): binary search over the B-tree's index access.
func (t *treeMap[K, V]) RankOfKey(key K) int {
	lo, hi := 0, t.bt.Len()-1
	for lo <= hi {
		mid := int(uint(lo+hi) >> 1)
		me, _ := t.bt.GetAt(mid)
		switch c := t.keyCmp(me.key, key); {
		case c == 0:
			return mid
		case c < 0:
			lo = mid + 1
		default:
			hi = mid - 1
		}
	}
	return -1
}

// GetByRank returns the entry at rank.
func (t *treeMap[K, V]) GetByRank(rank int) (Entry[K, V], bool) {
	me, ok := t.bt.GetAt(rank)
	if !ok {
		return Entry[K, V]{}, false
	}
	return Entry[K, V]{Key: me.key, Value: me.value}, true
}

// CloneSorted returns a shallow copy as SortedMap.
// The backing B-tree is copied copy-on-write, so this is O(1); both trees
// share nodes until either side writes.
func (t *treeMap[K, V]) CloneSorted() SortedMap[K, V] {
	return &treeMap[K, V]{bt: t.bt.Copy(), keyCmp: t.keyCmp}
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes entries in ascending key order as a JSON object with "entries" array.
//
// NOTE: The comparator is NOT serialized. Decode into a map constructed with
// NewTreeMap, or use UnmarshalTreeMapJSON / UnmarshalTreeMapOrderedJSON.
func (t *treeMap[K, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializableMap[K, V]{Entries: t.snapshotEntries()})
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes into the receiver using its existing comparator, so construct
// the map with NewTreeMap (or a helper) before decoding.
func (t *treeMap[K, V]) UnmarshalJSON(data []byte) error {
	if t.keyCmp == nil {
		return errors.New("unmarshal treemap: no comparator; construct the map with NewTreeMap before decoding, or use UnmarshalTreeMapJSON/UnmarshalTreeMapOrderedJSON")
	}
	var wrapped serializableMap[K, V]
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	t.replaceEntries(wrapped.Entries)
	return nil
}

// snapshotEntries copies the entries in ascending key order.
func (t *treeMap[K, V]) snapshotEntries() []serializableEntry[K, V] {
	entries := make([]serializableEntry[K, V], 0, t.bt.Len())
	t.bt.Scan(func(me mapEntry[K, V]) bool {
		entries = append(entries, serializableEntry[K, V]{Key: me.key, Value: me.value})
		return true
	})
	return entries
}

// replaceEntries replaces the map's contents with entries.
func (t *treeMap[K, V]) replaceEntries(entries []serializableEntry[K, V]) {
	t.Clear()
	for _, entry := range entries {
		t.Put(entry.Key, entry.Value)
	}
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (t *treeMap[K, V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, serializableMap[K, V]{Entries: t.snapshotEntries()})
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
// Same comparator contract as UnmarshalJSON.
func (t *treeMap[K, V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if t.keyCmp == nil {
		return errors.New("unmarshal treemap: no comparator; construct the map with NewTreeMap before decoding, or use UnmarshalTreeMapJSON/UnmarshalTreeMapOrderedJSON")
	}
	var wrapped serializableMap[K, V]
	if err := jsonv2.UnmarshalDecode(dec, &wrapped); err != nil {
		return err
	}
	t.replaceEntries(wrapped.Entries)
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes entries in ascending key order.
//
// NOTE: The comparator is NOT serialized. Decode into a map constructed with
// NewTreeMap, or use UnmarshalTreeMapGob / UnmarshalTreeMapOrderedGob.
func (t *treeMap[K, V]) GobEncode() ([]byte, error) {
	entries := t.snapshotEntries()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(entries); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes into the receiver using its existing comparator, so construct
// the map with NewTreeMap (or a helper) before decoding.
func (t *treeMap[K, V]) GobDecode(data []byte) error {
	if t.keyCmp == nil {
		return errors.New("unmarshal treemap: no comparator; construct the map with NewTreeMap before decoding, or use UnmarshalTreeMapGob/UnmarshalTreeMapOrderedGob")
	}
	var entries []serializableEntry[K, V]
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&entries); err != nil {
		return err
	}
	t.replaceEntries(entries)
	return nil
}

// Compile-time conformance check.
var (
	_ SortedMap[int, string] = (*treeMap[int, string])(nil)
)
