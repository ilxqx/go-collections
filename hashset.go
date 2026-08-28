package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"iter"
	"maps"
	"slices"
)

// hashSet is a hash-based implementation of Set[T] backed by map[T]struct{}.
// - Unordered: iteration order is not guaranteed and may vary between runs.
// - Not concurrent-safe: external synchronization is required when used from multiple goroutines.
// - Zero values: the zero value of hashSet is not ready for use; use the constructors below.
type hashSet[T comparable] struct {
	m map[T]struct{}
}

// NewHashSet creates an empty Set.
func NewHashSet[T comparable]() Set[T] {
	return &hashSet[T]{m: make(map[T]struct{})}
}

// NewHashSetWithCapacity creates an empty Set with an initial capacity hint.
func NewHashSetWithCapacity[T comparable](capacity int) Set[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &hashSet[T]{m: make(map[T]struct{}, capacity)}
}

// NewHashSetFrom creates a Set and inserts all provided elements.
func NewHashSetFrom[T comparable](elements ...T) Set[T] {
	s := &hashSet[T]{m: make(map[T]struct{}, len(elements))}
	for _, e := range elements {
		s.m[e] = struct{}{}
	}
	return s
}

// Size returns the number of elements in the set.
func (s *hashSet[T]) Size() int { return len(s.m) }

// IsEmpty reports whether the set contains no elements.
func (s *hashSet[T]) IsEmpty() bool    { return len(s.m) == 0 }
func (s *hashSet[T]) IsNotEmpty() bool { return !s.IsEmpty() }

// Clear removes all elements from the set.
func (s *hashSet[T]) Clear() { clear(s.m) }

// ToSlice returns a snapshot slice containing all elements.
// Order is not guaranteed.
func (s *hashSet[T]) ToSlice() []T {
	return slices.Collect(maps.Keys(s.m))
}

// String returns a concise string representation.
func (s *hashSet[T]) String() string {
	return formatCollection("hashSet", s.Seq())
}

// Seq returns an iter.Seq to iterate over elements with for-range.
func (s *hashSet[T]) Seq() iter.Seq[T] {
	return maps.Keys(s.m)
}

// ForEach applies action to each element, stopping early if action returns false.
func (s *hashSet[T]) ForEach(action func(element T) bool) {
	for v := range s.m {
		if !action(v) {
			return
		}
	}
}

// Add inserts element if absent. Returns true if the set changed.
func (s *hashSet[T]) Add(element T) bool {
	if _, ok := s.m[element]; ok {
		return false
	}
	s.m[element] = struct{}{}
	return true
}

// AddAll inserts all given elements. Returns the number of elements added.
func (s *hashSet[T]) AddAll(elements ...T) int {
	added := 0
	for _, e := range elements {
		if _, ok := s.m[e]; !ok {
			s.m[e] = struct{}{}
			added++
		}
	}
	return added
}

// AddSeq inserts all elements from the sequence. Returns the number added.
func (s *hashSet[T]) AddSeq(seq iter.Seq[T]) int {
	added := 0
	for v := range seq {
		if _, ok := s.m[v]; !ok {
			s.m[v] = struct{}{}
			added++
		}
	}
	return added
}

// Remove deletes the element if present. Returns true if removed.
func (s *hashSet[T]) Remove(element T) bool {
	if _, ok := s.m[element]; ok {
		delete(s.m, element)
		return true
	}
	return false
}

// RemoveAll deletes all given elements. Returns the number removed.
func (s *hashSet[T]) RemoveAll(elements ...T) int {
	removed := 0
	for _, e := range elements {
		if _, ok := s.m[e]; ok {
			delete(s.m, e)
			removed++
		}
	}
	return removed
}

// RemoveSeq removes all elements from the sequence. Returns the number removed.
func (s *hashSet[T]) RemoveSeq(seq iter.Seq[T]) int {
	removed := 0
	for v := range seq {
		if _, ok := s.m[v]; ok {
			delete(s.m, v)
			removed++
		}
	}
	return removed
}

// RemoveFunc removes elements satisfying predicate. Returns count removed.
func (s *hashSet[T]) RemoveFunc(predicate func(element T) bool) int {
	removed := 0
	for v := range s.m {
		if predicate(v) {
			delete(s.m, v)
			removed++
		}
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (s *hashSet[T]) RetainFunc(predicate func(element T) bool) int {
	removed := 0
	for v := range s.m {
		if !predicate(v) {
			delete(s.m, v)
			removed++
		}
	}
	return removed
}

// Pop removes and returns an arbitrary element. Returns (zero, false) if empty.
func (s *hashSet[T]) Pop() (T, bool) {
	for v := range s.m {
		delete(s.m, v)
		return v, true
	}
	var zero T
	return zero, false
}

// Contains reports whether element exists in the set.
func (s *hashSet[T]) Contains(element T) bool {
	_, ok := s.m[element]
	return ok
}

// ContainsAll reports whether all elements exist in the set.
func (s *hashSet[T]) ContainsAll(elements ...T) bool {
	for _, e := range elements {
		if _, ok := s.m[e]; !ok {
			return false
		}
	}
	return true
}

// ContainsAny reports whether any of the elements exist in the set.
func (s *hashSet[T]) ContainsAny(elements ...T) bool {
	for _, e := range elements {
		if _, ok := s.m[e]; ok {
			return true
		}
	}
	return false
}

// Union returns a new set: s ∪ other.
func (s *hashSet[T]) Union(other Set[T]) Set[T] {
	out := NewHashSetWithCapacity[T](len(s.m) + other.Size())
	for v := range s.m {
		out.Add(v)
	}
	for v := range other.Seq() {
		out.Add(v)
	}
	return out
}

// Intersection returns a new set: s ∩ other.
// The receiver's elements are tested with other.Contains, so the result
// holds the receiver's representatives and does not depend on set sizes.
func (s *hashSet[T]) Intersection(other Set[T]) Set[T] {
	// Preallocate up to the size of the smaller set.
	capacity := min(len(s.m), other.Size())
	out := NewHashSetWithCapacity[T](capacity)
	for v := range s.m {
		if other.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// Difference returns a new set: s - other.
func (s *hashSet[T]) Difference(other Set[T]) Set[T] {
	out := NewHashSetWithCapacity[T](len(s.m))
	for v := range s.m {
		if !other.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// SymmetricDifference returns a new set: (s - other) ∪ (other - s).
func (s *hashSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	// Preallocate up to sum of sizes as an upper bound.
	out := NewHashSetWithCapacity[T](len(s.m) + other.Size())
	for v := range s.m {
		if !other.Contains(v) {
			out.Add(v)
		}
	}
	for v := range other.Seq() {
		if _, ok := s.m[v]; !ok {
			out.Add(v)
		}
	}
	return out
}

// IsSubsetOf reports whether all elements of s are in other.
func (s *hashSet[T]) IsSubsetOf(other Set[T]) bool {
	for v := range s.m {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// IsSupersetOf reports whether s contains all elements of other.
func (s *hashSet[T]) IsSupersetOf(other Set[T]) bool {
	return other.IsSubsetOf(s)
}

// IsProperSubsetOf reports whether s ⊂ other (subset but not equal).
func (s *hashSet[T]) IsProperSubsetOf(other Set[T]) bool {
	return s.Size() < other.Size() && s.IsSubsetOf(other)
}

// IsProperSupersetOf reports whether s ⊃ other (superset but not equal).
func (s *hashSet[T]) IsProperSupersetOf(other Set[T]) bool {
	return s.Size() > other.Size() && s.IsSupersetOf(other)
}

// IsDisjoint reports whether s and other have no elements in common.
// The receiver's elements are tested with other.Contains, so the answer
// does not depend on set sizes.
func (s *hashSet[T]) IsDisjoint(other Set[T]) bool {
	for v := range s.m {
		if other.Contains(v) {
			return false
		}
	}
	return true
}

// Equals reports whether s and other contain exactly the same elements.
func (s *hashSet[T]) Equals(other Set[T]) bool {
	if s.Size() != other.Size() {
		return false
	}
	for v := range s.m {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// Clone returns a shallow copy of the set.
func (s *hashSet[T]) Clone() Set[T] {
	return &hashSet[T]{m: maps.Clone(s.m)}
}

// Filter returns a new set containing elements that satisfy predicate.
func (s *hashSet[T]) Filter(predicate func(element T) bool) Set[T] {
	out := NewHashSetWithCapacity[T](len(s.m) / 2)
	for v := range s.m {
		if predicate(v) {
			out.Add(v)
		}
	}
	return out
}

// Find returns the first element satisfying predicate, or (zero, false).
func (s *hashSet[T]) Find(predicate func(element T) bool) (T, bool) {
	for v := range s.m {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Any returns true if at least one element satisfies predicate.
func (s *hashSet[T]) Any(predicate func(element T) bool) bool {
	for v := range s.m {
		if predicate(v) {
			return true
		}
	}
	return false
}

// Every returns true if all elements satisfy predicate (true for empty set).
func (s *hashSet[T]) Every(predicate func(element T) bool) bool {
	for v := range s.m {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes the set as a JSON array.
func (s *hashSet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array. A payload holding a dynamically unhashable
// element is rejected with an error and leaves the set untouched.
func (s *hashSet[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	return s.replaceFromSlice(slice)
}

// replaceFromSlice validates the decoded elements and replaces the set's
// contents wholesale, so a rejected payload leaves the set untouched.
func (s *hashSet[T]) replaceFromSlice(slice []T) error {
	if err := validateHashable(slice); err != nil {
		return fmt.Errorf("unmarshal hashset: %w", err)
	}
	m := make(map[T]struct{}, len(slice))
	for _, elem := range slice {
		m[elem] = struct{}{}
	}
	s.m = m
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (s *hashSet[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, s.ToSlice())
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
// A payload holding a dynamically unhashable element is rejected with an\n// error and leaves the set untouched.
func (s *hashSet[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	return s.replaceFromSlice(slice)
}

// GobEncode implements gob.GobEncoder.
func (s *hashSet[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder. A payload holding a dynamically
// unhashable element is rejected with an error and leaves the set untouched.
func (s *hashSet[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	return s.replaceFromSlice(slice)
}

// Compile-time conformance checks (spot-check with concrete instantiation).
var (
	_ Set[int] = (*hashSet[int])(nil)
)
