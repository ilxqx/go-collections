package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"iter"

	"github.com/puzpuzpuz/xsync/v3"
)

// concurrentHashSet is a thread-safe hash set backed by xsync.MapOf[T,struct{}].
// Single-key operations (Add/Remove/Contains/AddIfAbsent/RemoveAndGet) are atomic.
// Bulk and algebra operations iterate over the map and are not atomic as a whole;
// results reflect a best-effort snapshot under concurrency.
type concurrentHashSet[T comparable] struct {
	m *xsync.MapOf[T, struct{}]
}

// NewConcurrentHashSet creates an empty concurrent set.
func NewConcurrentHashSet[T comparable]() ConcurrentSet[T] {
	return &concurrentHashSet[T]{m: newBackingMapOf[T, struct{}]()}
}

// NewConcurrentHashSetFrom creates a concurrent set and inserts elements.
func NewConcurrentHashSetFrom[T comparable](elements ...T) ConcurrentSet[T] {
	s := &concurrentHashSet[T]{m: newBackingMapOf[T, struct{}]()}
	for _, e := range elements {
		s.m.Store(e, struct{}{})
	}
	return s
}

// Size returns an approximate element count.
// Note: xsync.MapOf.Size() is approximate under concurrency.
func (s *concurrentHashSet[T]) Size() int { return s.m.Size() }

// IsEmpty reports whether the set is empty (approximate under concurrency).
func (s *concurrentHashSet[T]) IsEmpty() bool    { return s.Size() == 0 }
func (s *concurrentHashSet[T]) IsNotEmpty() bool { return !s.IsEmpty() }

// Clear removes all elements.
func (s *concurrentHashSet[T]) Clear() { s.m.Clear() }

// ToSlice returns a snapshot slice of elements (order not guaranteed).
func (s *concurrentHashSet[T]) ToSlice() []T {
	out := make([]T, 0, s.Size())
	s.m.Range(func(k T, _ struct{}) bool {
		out = append(out, k)
		return true
	})
	return out
}

// String returns a concise representation (unordered).
func (s *concurrentHashSet[T]) String() string {
	return formatCollection("concurrentHashSet", s.Seq())
}

// Seq returns a sequence of elements (unordered).
func (s *concurrentHashSet[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		s.m.Range(func(k T, _ struct{}) bool {
			return yield(k)
		})
	}
}

// ForEach iterates elements; stops early if action returns false (unordered).
func (s *concurrentHashSet[T]) ForEach(action func(element T) bool) {
	s.m.Range(func(k T, _ struct{}) bool {
		return action(k)
	})
}

// Add inserts element if absent. Returns true if the set changed (atomic).
func (s *concurrentHashSet[T]) Add(element T) bool {
	_, loaded := s.m.LoadOrStore(element, struct{}{})
	return !loaded
}

// AddAll inserts all given elements. Returns the number of elements added.
func (s *concurrentHashSet[T]) AddAll(elements ...T) int {
	added := 0
	for _, e := range elements {
		if _, loaded := s.m.LoadOrStore(e, struct{}{}); !loaded {
			added++
		}
	}
	return added
}

// AddSeq inserts all elements from the sequence. Returns the number added.
func (s *concurrentHashSet[T]) AddSeq(seq iter.Seq[T]) int {
	added := 0
	for v := range seq {
		if _, loaded := s.m.LoadOrStore(v, struct{}{}); !loaded {
			added++
		}
	}
	return added
}

// Remove deletes the element if present. Returns true if removed (atomic).
func (s *concurrentHashSet[T]) Remove(element T) bool {
	_, ok := s.m.LoadAndDelete(element)
	return ok
}

// RemoveAll deletes all given elements. Returns the number removed.
func (s *concurrentHashSet[T]) RemoveAll(elements ...T) int {
	removed := 0
	for _, e := range elements {
		if _, ok := s.m.LoadAndDelete(e); ok {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes all elements from the sequence. Returns the number removed.
func (s *concurrentHashSet[T]) RemoveSeq(seq iter.Seq[T]) int {
	removed := 0
	for v := range seq {
		if _, ok := s.m.LoadAndDelete(v); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes elements satisfying predicate. Returns count removed.
func (s *concurrentHashSet[T]) RemoveFunc(predicate func(element T) bool) int {
	size := s.Size()
	if size == 0 {
		return 0
	}
	dels := make([]T, 0, size/2)
	s.m.Range(func(k T, _ struct{}) bool {
		if predicate(k) {
			dels = append(dels, k)
		}
		return true
	})
	count := 0
	for _, k := range dels {
		if _, ok := s.m.LoadAndDelete(k); ok {
			count++
		}
	}
	return count
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (s *concurrentHashSet[T]) RetainFunc(predicate func(element T) bool) int {
	return s.RemoveFunc(func(element T) bool { return !predicate(element) })
}

// Pop removes and returns an arbitrary element. Returns (zero, false) if empty.
func (s *concurrentHashSet[T]) Pop() (T, bool) {
	var zero T
	for {
		var picked T
		found := false
		s.m.Range(func(k T, _ struct{}) bool {
			picked = k
			found = true
			return false // stop early
		})
		if !found {
			return zero, false
		}
		if _, ok := s.m.LoadAndDelete(picked); ok {
			return picked, true
		}
		// retry if race removed it
	}
}

// Contains reports whether element exists (atomic).
func (s *concurrentHashSet[T]) Contains(element T) bool {
	_, ok := s.m.Load(element)
	return ok
}

// ContainsAll reports whether all elements exist in the set (best-effort).
func (s *concurrentHashSet[T]) ContainsAll(elements ...T) bool {
	for _, e := range elements {
		if _, ok := s.m.Load(e); !ok {
			return false
		}
	}
	return true
}

// ContainsAny reports whether any of the elements exist in the set (best-effort).
func (s *concurrentHashSet[T]) ContainsAny(elements ...T) bool {
	for _, e := range elements {
		if _, ok := s.m.Load(e); ok {
			return true
		}
	}
	return false
}

// Union returns a new HashSet as snapshot of s ∪ other.
func (s *concurrentHashSet[T]) Union(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		out.Add(k)
		return true
	})
	for v := range other.Seq() {
		out.Add(v)
	}
	return out
}

// Intersection returns a new HashSet snapshot of s ∩ other.
func (s *concurrentHashSet[T]) Intersection(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		if other.Contains(k) {
			out.Add(k)
		}
		return true
	})
	return out
}

// Difference returns a new HashSet snapshot of s - other.
func (s *concurrentHashSet[T]) Difference(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		if !other.Contains(k) {
			out.Add(k)
		}
		return true
	})
	return out
}

// SymmetricDifference returns a new HashSet snapshot of (s - other) ∪ (other - s).
func (s *concurrentHashSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		if !other.Contains(k) {
			out.Add(k)
		}
		return true
	})
	for v := range other.Seq() {
		if !s.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// IsSubsetOf reports whether all elements of s are in other (best-effort).
func (s *concurrentHashSet[T]) IsSubsetOf(other Set[T]) bool {
	ok := true
	s.m.Range(func(k T, _ struct{}) bool {
		if !other.Contains(k) {
			ok = false
			return false
		}
		return true
	})
	return ok
}

// IsSupersetOf reports whether s contains all elements of other.
func (s *concurrentHashSet[T]) IsSupersetOf(other Set[T]) bool {
	return other.IsSubsetOf(s)
}

// IsProperSubsetOf reports whether s is a strict subset of other (best-effort).
func (s *concurrentHashSet[T]) IsProperSubsetOf(other Set[T]) bool {
	return s.Size() < other.Size() && s.IsSubsetOf(other)
}

// IsProperSupersetOf reports whether s is a strict superset of other (best-effort).
func (s *concurrentHashSet[T]) IsProperSupersetOf(other Set[T]) bool {
	return s.Size() > other.Size() && s.IsSupersetOf(other)
}

// IsDisjoint reports whether s and other share no common elements (best-effort).
func (s *concurrentHashSet[T]) IsDisjoint(other Set[T]) bool {
	disjoint := true
	s.m.Range(func(k T, _ struct{}) bool {
		if other.Contains(k) {
			disjoint = false
			return false
		}
		return true
	})
	return disjoint
}

// Equals reports whether s and other contain exactly the same elements.
func (s *concurrentHashSet[T]) Equals(other Set[T]) bool {
	// Snapshot s into HashSet and compare with other to avoid approximate size.
	snap := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		snap.Add(k)
		return true
	})
	return snap.Equals(other)
}

// Clone returns a shallow copy as HashSet snapshot.
func (s *concurrentHashSet[T]) Clone() Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		out.Add(k)
		return true
	})
	return out
}

// Filter returns a new HashSet containing elements that satisfy predicate.
func (s *concurrentHashSet[T]) Filter(predicate func(element T) bool) Set[T] {
	out := NewHashSet[T]()
	s.m.Range(func(k T, _ struct{}) bool {
		if predicate(k) {
			out.Add(k)
		}
		return true
	})
	return out
}

// Find returns the first element that satisfies predicate (best-effort).
func (s *concurrentHashSet[T]) Find(predicate func(element T) bool) (T, bool) {
	var res T
	found := false
	s.m.Range(func(k T, _ struct{}) bool {
		if predicate(k) {
			res = k
			found = true
			return false
		}
		return true
	})
	return res, found
}

// Any returns true if at least one element satisfies predicate.
func (s *concurrentHashSet[T]) Any(predicate func(element T) bool) bool {
	ans := false
	s.m.Range(func(k T, _ struct{}) bool {
		if predicate(k) {
			ans = true
			return false
		}
		return true
	})
	return ans
}

// Every returns true if all elements satisfy predicate (true for empty set).
func (s *concurrentHashSet[T]) Every(predicate func(element T) bool) bool {
	all := true
	s.m.Range(func(k T, _ struct{}) bool {
		if !predicate(k) {
			all = false
			return false
		}
		return true
	})
	return all
}

// AddIfAbsent atomically adds the element only if not present.
func (s *concurrentHashSet[T]) AddIfAbsent(element T) bool { return s.Add(element) }

// RemoveAndGet atomically removes and returns the element if present.
func (s *concurrentHashSet[T]) RemoveAndGet(element T) (T, bool) {
	_, ok := s.m.LoadAndDelete(element)
	if ok {
		return element, true
	}
	var zero T
	return zero, false
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes a snapshot of the set as a JSON array.
// NOTE: Provides snapshot consistency - concurrent modifications
// during serialization may not be reflected.
func (s *concurrentHashSet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.ToSlice())
}

// refill replaces the set's contents with elements. A zero-value receiver —
// gob allocates one when decoding an interface field — gets a fresh backing
// map, which nothing else can reference yet. Otherwise the existing backing
// map is refilled in place: replacing the s.m pointer would race with
// concurrent readers of the receiver.
func (s *concurrentHashSet[T]) refill(elements []T) {
	if s.m == nil {
		s.m = newBackingMapOf[T, struct{}]()
	} else {
		s.m.Clear()
	}
	for _, elem := range elements {
		s.m.Store(elem, struct{}{})
	}
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array. A payload holding a dynamically unhashable
// element is rejected with an error and leaves the set untouched.
func (s *concurrentHashSet[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	if err := validateHashable(slice); err != nil {
		return fmt.Errorf("unmarshal concurrent hashset: %w", err)
	}
	s.refill(slice)
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (s *concurrentHashSet[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, s.ToSlice())
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
// A payload holding a dynamically unhashable element is rejected with an\n// error and leaves the set untouched.
func (s *concurrentHashSet[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	if err := validateHashable(slice); err != nil {
		return fmt.Errorf("unmarshal concurrent hashset: %w", err)
	}
	s.refill(slice)
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the set.
func (s *concurrentHashSet[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data. A payload holding a dynamically unhashable
// element is rejected with an error and leaves the set untouched.
func (s *concurrentHashSet[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	if err := validateHashable(slice); err != nil {
		return fmt.Errorf("unmarshal concurrent hashset: %w", err)
	}
	s.refill(slice)
	return nil
}

// Conformance.
var (
	_ ConcurrentSet[int] = (*concurrentHashSet[int])(nil)
)
