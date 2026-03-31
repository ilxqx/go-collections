package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"maps"
	"slices"
)

// hashMap is a hash-based implementation of Map[K,V] backed by map[K]V.
// - Unordered: iteration order is not guaranteed and may vary between runs.
// - Not concurrent-safe: external synchronization is required when used from multiple goroutines.
// - Zero values: the zero value of hashMap is not ready for use; use the constructors below.
type hashMap[K comparable, V any] struct {
	m map[K]V
}

// NewHashMap creates an empty Map.
func NewHashMap[K comparable, V any]() Map[K, V] {
	return &hashMap[K, V]{m: make(map[K]V)}
}

// NewHashMapWithCapacity creates an empty Map with an initial capacity hint.
func NewHashMapWithCapacity[K comparable, V any](capacity int) Map[K, V] {
	if capacity < 0 {
		capacity = 0
	}
	return &hashMap[K, V]{m: make(map[K]V, capacity)}
}

// NewHashMapFrom creates a Map from a standard Go map (copying entries).
func NewHashMapFrom[K comparable, V any](src map[K]V) Map[K, V] {
	h := &hashMap[K, V]{m: make(map[K]V, len(src))}
	maps.Copy(h.m, src)
	return h
}

// Size returns the number of entries.
func (h *hashMap[K, V]) Size() int { return len(h.m) }

// IsEmpty reports whether the map has no entries.
func (h *hashMap[K, V]) IsEmpty() bool    { return len(h.m) == 0 }
func (h *hashMap[K, V]) IsNotEmpty() bool { return !h.IsEmpty() }

// Clear removes all entries.
func (h *hashMap[K, V]) Clear() { clear(h.m) }

// String returns a concise string representation.
func (h *hashMap[K, V]) String() string {
	return formatMap("hashMap", h.Seq())
}

// Put associates value with key. Returns (oldValue, true) if key existed.
func (h *hashMap[K, V]) Put(key K, value V) (V, bool) {
	old, ok := h.m[key]
	h.m[key] = value
	return old, ok
}

// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
func (h *hashMap[K, V]) PutIfAbsent(key K, value V) (V, bool) {
	if old, ok := h.m[key]; ok {
		return old, false
	}
	h.m[key] = value
	return value, true
}

// PutAll copies all entries from other into this map.
func (h *hashMap[K, V]) PutAll(other Map[K, V]) {
	maps.Insert(h.m, other.Seq())
}

// PutSeq copies entries from a sequence. Returns number of keys changed (unique keys touched).
func (h *hashMap[K, V]) PutSeq(seq iter.Seq2[K, V]) int {
	changed := 0
	seen := make(map[K]struct{})
	for k, v := range seq {
		if _, ok := seen[k]; !ok {
			changed++
			seen[k] = struct{}{}
		}
		h.m[k] = v
	}
	return changed
}

// Get returns (value, true) if key present; otherwise (zero, false).
func (h *hashMap[K, V]) Get(key K) (V, bool) {
	v, ok := h.m[key]
	return v, ok
}

// GetOrDefault returns value for key or defaultValue if absent.
func (h *hashMap[K, V]) GetOrDefault(key K, defaultValue V) V {
	if v, ok := h.m[key]; ok {
		return v
	}
	return defaultValue
}

// Remove deletes key. Returns (oldValue, true) if key existed.
func (h *hashMap[K, V]) Remove(key K) (V, bool) {
	old, ok := h.m[key]
	if ok {
		delete(h.m, key)
	}
	return old, ok
}

// RemoveIf deletes only if (key, value) matches. Returns true if removed.
func (h *hashMap[K, V]) RemoveIf(key K, value V, eq Equaler[V]) bool {
	old, ok := h.m[key]
	if ok && eq(old, value) {
		delete(h.m, key)
		return true
	}
	return false
}

// ContainsKey reports whether key exists.
func (h *hashMap[K, V]) ContainsKey(key K) bool {
	_, ok := h.m[key]
	return ok
}

// ContainsValue reports whether value exists (O(n)).
func (h *hashMap[K, V]) ContainsValue(value V, eq Equaler[V]) bool {
	for _, v := range h.m {
		if eq(v, value) {
			return true
		}
	}
	return false
}

// RemoveAll removes all specified keys. Returns count removed.
func (h *hashMap[K, V]) RemoveAll(keys ...K) int {
	removed := 0
	for _, k := range keys {
		if _, ok := h.m[k]; ok {
			delete(h.m, k)
			removed++
		}
	}
	return removed
}

// RemoveSeq removes keys from the sequence. Returns count removed.
func (h *hashMap[K, V]) RemoveSeq(seq iter.Seq[K]) int {
	removed := 0
	for k := range seq {
		if _, ok := h.m[k]; ok {
			delete(h.m, k)
			removed++
		}
	}
	return removed
}

// RemoveFunc removes entries where predicate returns true. Returns count removed.
func (h *hashMap[K, V]) RemoveFunc(predicate func(key K, value V) bool) int {
	removed := 0
	for k, v := range h.m {
		if predicate(k, v) {
			delete(h.m, k)
			removed++
		}
	}
	return removed
}

// Compute recomputes mapping for key. If keep==false, the key is removed.
func (h *hashMap[K, V]) Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool) {
	old, exists := h.m[key]
	newVal, keep := remapping(key, old, exists)
	if !keep {
		if exists {
			delete(h.m, key)
		}
		var zero V
		return zero, false
	}
	h.m[key] = newVal
	return newVal, true
}

// ComputeIfAbsent computes and stores value if key is absent.
func (h *hashMap[K, V]) ComputeIfAbsent(key K, mapping func(key K) V) V {
	if v, ok := h.m[key]; ok {
		return v
	}
	v := mapping(key)
	h.m[key] = v
	return v
}

// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
func (h *hashMap[K, V]) ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool) {
	old, ok := h.m[key]
	if !ok {
		var zero V
		return zero, false
	}
	newVal, keep := remapping(key, old)
	if !keep {
		delete(h.m, key)
		var zero V
		return zero, false
	}
	h.m[key] = newVal
	return newVal, true
}

// Merge merges value with existing. If keep==false, removes key.
func (h *hashMap[K, V]) Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool) {
	if old, ok := h.m[key]; ok {
		merged, keep := remapping(old, value)
		if !keep {
			delete(h.m, key)
			var zero V
			return zero, false
		}
		h.m[key] = merged
		return merged, true
	}
	h.m[key] = value
	return value, true
}

// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
func (h *hashMap[K, V]) Replace(key K, value V) (V, bool) {
	if old, ok := h.m[key]; ok {
		h.m[key] = value
		return old, true
	}
	var zero V
	return zero, false
}

// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced.
func (h *hashMap[K, V]) ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool {
	if old, ok := h.m[key]; ok && eq(old, oldValue) {
		h.m[key] = newValue
		return true
	}
	return false
}

// ReplaceAll replaces each value with function(key, value).
func (h *hashMap[K, V]) ReplaceAll(function func(key K, value V) V) {
	for k, v := range h.m {
		h.m[k] = function(k, v)
	}
}

// Keys returns all keys as a slice.
func (h *hashMap[K, V]) Keys() []K {
	return slices.Collect(maps.Keys(h.m))
}

// Values returns all values as a slice.
func (h *hashMap[K, V]) Values() []V {
	return slices.Collect(maps.Values(h.m))
}

// Entries returns all entries as a slice.
func (h *hashMap[K, V]) Entries() []Entry[K, V] {
	out := make([]Entry[K, V], 0, len(h.m))
	for k, v := range h.m {
		out = append(out, Entry[K, V]{Key: k, Value: v})
	}
	return out
}

// ForEach iterates over entries. Stops early if action returns false.
func (h *hashMap[K, V]) ForEach(action func(key K, value V) bool) {
	for k, v := range h.m {
		if !action(k, v) {
			return
		}
	}
}

// Seq returns a sequence of (key, value) pairs.
func (h *hashMap[K, V]) Seq() iter.Seq2[K, V] {
	return maps.All(h.m)
}

// SeqKeys returns a sequence of keys.
func (h *hashMap[K, V]) SeqKeys() iter.Seq[K] {
	return maps.Keys(h.m)
}

// SeqValues returns a sequence of values.
func (h *hashMap[K, V]) SeqValues() iter.Seq[V] {
	return maps.Values(h.m)
}

// Clone returns a shallow copy.
func (h *hashMap[K, V]) Clone() Map[K, V] {
	return &hashMap[K, V]{m: maps.Clone(h.m)}
}

// ToGoMap returns a snapshot copy as a standard Go map[K]V.
func (h *hashMap[K, V]) ToGoMap() map[K]V {
	return maps.Clone(h.m)
}

// Filter returns a new map with entries satisfying predicate.
func (h *hashMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	out := &hashMap[K, V]{m: make(map[K]V)}
	for k, v := range h.m {
		if predicate(k, v) {
			out.m[k] = v
		}
	}
	return out
}

// Equals reports whether both maps contain the same entries.
func (h *hashMap[K, V]) Equals(other Map[K, V], valueEq Equaler[V]) bool {
	if h.Size() != other.Size() {
		return false
	}
	for k, v := range h.m {
		ov, ok := other.Get(k)
		if !ok || !valueEq(v, ov) {
			return false
		}
	}
	return true
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes the map as a JSON object (keys must be string-convertible) or array of entries.
func (h *hashMap[K, V]) MarshalJSON() ([]byte, error) {
	// Serialize as a standard Go map since K is comparable
	return json.Marshal(h.m)
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON object.
func (h *hashMap[K, V]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &h.m)
}

// GobEncode implements gob.GobEncoder.
func (h *hashMap[K, V]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(h.m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (h *hashMap[K, V]) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(&h.m)
}

// Compile-time conformance checks with concrete instantiation.
var (
	_ Map[int, string] = (*hashMap[int, string])(nil)
)
