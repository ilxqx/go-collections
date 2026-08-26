package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"

	"github.com/zhangyunhao116/skipset"
)

// concurrentSkipSet is a concurrent-safe sorted set backed by a lock-free skip list.
//   - Single-key operations (Add/Remove/Contains/AddIfAbsent/RemoveAndGet) are atomic.
//   - Iteration and bulk/algebra operations are not atomic as a whole; they provide
//     best-effort snapshots under concurrency.
//   - Many navigation/rank operations are O(n) because skipset does not expose
//     direct predecessor/successor APIs.
type concurrentSkipSet[T Ordered] struct {
	s *skipset.OrderedSet[T]
}

// NewConcurrentSkipSet creates an empty concurrent sorted set for Ordered types.
func NewConcurrentSkipSet[T Ordered]() ConcurrentSortedSet[T] {
	return &concurrentSkipSet[T]{s: skipset.New[T]()}
}

// NewConcurrentSkipSetFrom creates a set and inserts the given elements.
func NewConcurrentSkipSetFrom[T Ordered](elements ...T) ConcurrentSortedSet[T] {
	cs := &concurrentSkipSet[T]{s: skipset.New[T]()}
	for _, e := range elements {
		cs.s.Add(e)
	}
	return cs
}

// Size returns the number of elements (approximate under heavy concurrency).
func (c *concurrentSkipSet[T]) Size() int { return c.s.Len() }

// IsEmpty reports whether the set is empty.
func (c *concurrentSkipSet[T]) IsEmpty() bool    { return c.Size() == 0 }
func (c *concurrentSkipSet[T]) IsNotEmpty() bool { return !c.IsEmpty() }

// Clear removes all elements by replacing the underlying structure.
// Note: This may cause a temporary memory peak while the old structure is
// garbage-collected, depending on GC timing.
func (c *concurrentSkipSet[T]) Clear() { c.s = skipset.New[T]() }

// ToSlice returns a snapshot slice of all elements in ascending order.
func (c *concurrentSkipSet[T]) ToSlice() []T {
	out := make([]T, 0, c.Size())
	c.s.Range(func(v T) bool {
		out = append(out, v)
		return true
	})
	return out
}

// String returns a string representation (ascending order).
func (c *concurrentSkipSet[T]) String() string {
	return formatCollection("concurrentSkipSet", c.Seq())
}

// Seq returns a sequence of elements in ascending order.
func (c *concurrentSkipSet[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		c.s.Range(func(v T) bool {
			return yield(v)
		})
	}
}

// ForEach iterates elements ascending; stops early if action returns false.
func (c *concurrentSkipSet[T]) ForEach(action func(element T) bool) {
	c.s.Range(func(v T) bool { return action(v) })
}

// Add inserts element if absent. Returns true if the set changed (atomic).
func (c *concurrentSkipSet[T]) Add(element T) bool { return c.s.Add(element) }

// AddAll inserts all given elements. Returns number added.
func (c *concurrentSkipSet[T]) AddAll(elements ...T) int {
	added := 0
	for _, e := range elements {
		if c.s.Add(e) {
			added++
		}
	}
	return added
}

// AddSeq inserts all elements from the sequence. Returns number added.
func (c *concurrentSkipSet[T]) AddSeq(seq iter.Seq[T]) int {
	added := 0
	for v := range seq {
		if c.s.Add(v) {
			added++
		}
	}
	return added
}

// Remove deletes the element if present. Returns true if removed (atomic).
func (c *concurrentSkipSet[T]) Remove(element T) bool { return c.s.Remove(element) }

// RemoveAll deletes all given elements. Returns the number removed.
func (c *concurrentSkipSet[T]) RemoveAll(elements ...T) int {
	removed := 0
	for _, e := range elements {
		if c.s.Remove(e) {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes all elements from the sequence. Returns the number removed.
func (c *concurrentSkipSet[T]) RemoveSeq(seq iter.Seq[T]) int {
	removed := 0
	for v := range seq {
		if c.s.Remove(v) {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes elements satisfying predicate. Returns count removed.
func (c *concurrentSkipSet[T]) RemoveFunc(predicate func(element T) bool) int {
	size := c.Size()
	if size == 0 {
		return 0
	}
	dels := make([]T, 0, size/2)
	c.s.Range(func(v T) bool {
		if predicate(v) {
			dels = append(dels, v)
		}
		return true
	})
	count := 0
	for _, v := range dels {
		if c.s.Remove(v) {
			count++
		}
	}
	return count
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (c *concurrentSkipSet[T]) RetainFunc(predicate func(element T) bool) int {
	size := c.Size()
	if size == 0 {
		return 0
	}
	dels := make([]T, 0, size/2)
	c.s.Range(func(v T) bool {
		if !predicate(v) {
			dels = append(dels, v)
		}
		return true
	})
	count := 0
	for _, v := range dels {
		if c.s.Remove(v) {
			count++
		}
	}
	return count
}

// Pop removes and returns the smallest element.
func (c *concurrentSkipSet[T]) Pop() (T, bool) { return c.PopFirst() }

// Contains reports whether element exists (atomic).
func (c *concurrentSkipSet[T]) Contains(element T) bool { return c.s.Contains(element) }

// ContainsAll reports whether all elements exist in the set (best-effort).
func (c *concurrentSkipSet[T]) ContainsAll(elements ...T) bool {
	for _, e := range elements {
		if !c.s.Contains(e) {
			return false
		}
	}
	return true
}

// ContainsAny reports whether any of the elements exist in the set (best-effort).
func (c *concurrentSkipSet[T]) ContainsAny(elements ...T) bool {
	return slices.ContainsFunc(elements, c.s.Contains)
}

// Union returns a new HashSet snapshot of c ∪ other (unordered result).
func (c *concurrentSkipSet[T]) Union(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	c.s.Range(func(v T) bool { out.Add(v); return true })
	for v := range other.Seq() {
		out.Add(v)
	}
	return out
}

// Intersection returns a new HashSet snapshot of c ∩ other.
func (c *concurrentSkipSet[T]) Intersection(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	c.s.Range(func(v T) bool {
		if other.Contains(v) {
			out.Add(v)
		}
		return true
	})
	return out
}

// Difference returns a new HashSet snapshot of c - other.
func (c *concurrentSkipSet[T]) Difference(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	c.s.Range(func(v T) bool {
		if !other.Contains(v) {
			out.Add(v)
		}
		return true
	})
	return out
}

// SymmetricDifference returns a new HashSet snapshot of (c - other) ∪ (other - c).
func (c *concurrentSkipSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	out := NewHashSet[T]()
	c.s.Range(func(v T) bool {
		if !other.Contains(v) {
			out.Add(v)
		}
		return true
	})
	for v := range other.Seq() {
		if !c.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// IsSubsetOf reports whether all elements of c are in other (best-effort).
func (c *concurrentSkipSet[T]) IsSubsetOf(other Set[T]) bool {
	ok := true
	c.s.Range(func(v T) bool {
		if !other.Contains(v) {
			ok = false
			return false
		}
		return true
	})
	return ok
}

// IsSupersetOf reports whether c contains all elements of other.
func (c *concurrentSkipSet[T]) IsSupersetOf(other Set[T]) bool { return other.IsSubsetOf(c) }

// IsProperSubsetOf reports whether c is a strict subset of other (best-effort).
func (c *concurrentSkipSet[T]) IsProperSubsetOf(other Set[T]) bool {
	return c.Size() < other.Size() && c.IsSubsetOf(other)
}

// IsProperSupersetOf reports whether c is a strict superset of other (best-effort).
func (c *concurrentSkipSet[T]) IsProperSupersetOf(other Set[T]) bool {
	return c.Size() > other.Size() && c.IsSupersetOf(other)
}

// IsDisjoint reports whether c and other share no common elements (best-effort).
func (c *concurrentSkipSet[T]) IsDisjoint(other Set[T]) bool {
	disjoint := true
	c.s.Range(func(v T) bool {
		if other.Contains(v) {
			disjoint = false
			return false
		}
		return true
	})
	return disjoint
}

// Equals reports whether c and other contain exactly the same elements.
func (c *concurrentSkipSet[T]) Equals(other Set[T]) bool {
	// Snapshot this into a HashSet and compare.
	snap := NewHashSet[T]()
	c.s.Range(func(v T) bool { snap.Add(v); return true })
	return snap.Equals(other)
}

// Clone returns a shallow copy (as Set).
func (c *concurrentSkipSet[T]) Clone() Set[T] { return c.CloneSorted() }

// Filter returns a new set with elements satisfying predicate.
func (c *concurrentSkipSet[T]) Filter(predicate func(element T) bool) Set[T] {
	out := NewTreeSetOrdered[T]() // keep sorted snapshot
	c.s.Range(func(v T) bool {
		if predicate(v) {
			out.Add(v)
		}
		return true
	})
	return out
}

// Find returns the first element satisfying predicate (ascending order).
func (c *concurrentSkipSet[T]) Find(predicate func(element T) bool) (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		if predicate(v) {
			res = v
			found = true
			return false
		}
		return true
	})
	return res, found
}

// Any returns true if at least one element satisfies predicate.
func (c *concurrentSkipSet[T]) Any(predicate func(element T) bool) bool {
	ans := false
	c.s.Range(func(v T) bool {
		if predicate(v) {
			ans = true
			return false
		}
		return true
	})
	return ans
}

// Every returns true if all elements satisfy predicate (true for empty set).
func (c *concurrentSkipSet[T]) Every(predicate func(element T) bool) bool {
	all := true
	c.s.Range(func(v T) bool {
		if !predicate(v) {
			all = false
			return false
		}
		return true
	})
	return all
}

// AddIfAbsent atomically inserts element only if currently absent.
func (c *concurrentSkipSet[T]) AddIfAbsent(element T) bool { return c.s.Add(element) }

// RemoveAndGet atomically removes and returns the element if present.
func (c *concurrentSkipSet[T]) RemoveAndGet(element T) (T, bool) {
	if c.s.Remove(element) {
		return element, true
	}
	var zero T
	return zero, false
}

// First returns the smallest element, or (zero, false) if empty.
func (c *concurrentSkipSet[T]) First() (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		res = v
		found = true
		return false
	})
	return res, found
}

// Last returns the largest element, or (zero, false) if empty.
func (c *concurrentSkipSet[T]) Last() (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		res = v
		found = true
		return true
	})
	return res, found
}

// Min is an alias for First.
func (c *concurrentSkipSet[T]) Min() (T, bool) { return c.First() }

// Max is an alias for Last.
func (c *concurrentSkipSet[T]) Max() (T, bool) { return c.Last() }

// PopFirst removes and returns the smallest element.
func (c *concurrentSkipSet[T]) PopFirst() (T, bool) {
	for {
		v, ok := c.First()
		if !ok {
			var zero T
			return zero, false
		}
		if c.s.Remove(v) {
			return v, true
		}
		// retry if a race removed it
	}
}

// PopLast removes and returns the largest element.
func (c *concurrentSkipSet[T]) PopLast() (T, bool) {
	for {
		v, ok := c.Last()
		if !ok {
			var zero T
			return zero, false
		}
		if c.s.Remove(v) {
			return v, true
		}
	}
}

// Floor returns the greatest element <= x.
func (c *concurrentSkipSet[T]) Floor(x T) (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		if v <= x {
			res = v
			found = true
			return true
		}
		// v > x, past the floor
		return false
	})
	return res, found
}

// Ceiling returns the smallest element >= x.
func (c *concurrentSkipSet[T]) Ceiling(x T) (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		if v >= x {
			res = v
			found = true
			return false
		}
		return true
	})
	return res, found
}

// Lower returns the greatest element strictly < x.
func (c *concurrentSkipSet[T]) Lower(x T) (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		if v < x {
			res = v
			found = true
			return true
		}
		// v >= x, we've passed lower
		return false
	})
	return res, found
}

// Higher returns the smallest element strictly > x.
func (c *concurrentSkipSet[T]) Higher(x T) (T, bool) {
	var res T
	found := false
	c.s.Range(func(v T) bool {
		if v > x {
			res = v
			found = true
			return false
		}
		return true
	})
	return res, found
}

// Range iterates elements in [from, to] ascending.
func (c *concurrentSkipSet[T]) Range(from, to T, action func(element T) bool) {
	if from > to {
		return
	}
	c.s.Range(func(v T) bool {
		if v < from {
			return true
		}
		if v > to {
			return false
		}
		return action(v)
	})
}

// RangeSeq returns a sequence for elements in [from, to] ascending.
func (c *concurrentSkipSet[T]) RangeSeq(from, to T) iter.Seq[T] {
	return func(yield func(T) bool) {
		if from > to {
			return
		}
		c.s.Range(func(v T) bool {
			if v < from {
				return true
			}
			if v > to {
				return false
			}
			return yield(v)
		})
	}
}

// Ascend iterates all elements in ascending order.
func (c *concurrentSkipSet[T]) Ascend(action func(element T) bool) {
	c.s.Range(func(v T) bool { return action(v) })
}

// Descend iterates all elements in descending order (O(n) snapshot).
func (c *concurrentSkipSet[T]) Descend(action func(element T) bool) {
	buf := c.ToSlice()
	for _, v := range slices.Backward(buf) {
		if !action(v) {
			return
		}
	}
}

// AscendFrom iterates elements >= pivot in ascending order.
func (c *concurrentSkipSet[T]) AscendFrom(pivot T, action func(element T) bool) {
	c.s.Range(func(v T) bool {
		if v >= pivot {
			return action(v)
		}
		return true
	})
}

// DescendFrom iterates elements <= pivot in descending order (O(n) snapshot).
func (c *concurrentSkipSet[T]) DescendFrom(pivot T, action func(element T) bool) {
	buf := c.ToSlice()
	for _, v := range slices.Backward(buf) {
		if v <= pivot {
			if !action(v) {
				return
			}
		}
	}
}

// Reversed returns a descending sequence (snapshot).
func (c *concurrentSkipSet[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		buf := c.ToSlice()
		for _, v := range slices.Backward(buf) {
			if !yield(v) {
				return
			}
		}
	}
}

// SubSet returns a new set containing elements in [from, to] (snapshot).
func (c *concurrentSkipSet[T]) SubSet(from, to T) SortedSet[T] {
	out := NewTreeSetOrdered[T]()
	if from > to {
		return out
	}
	c.Range(from, to, func(e T) bool { out.Add(e); return true })
	return out
}

// HeadSet returns elements < to (or <= to if inclusive) (snapshot).
func (c *concurrentSkipSet[T]) HeadSet(to T, inclusive bool) SortedSet[T] {
	out := NewTreeSetOrdered[T]()
	c.s.Range(func(v T) bool {
		if v < to || (inclusive && v == to) {
			out.Add(v)
			return true
		}
		return v < to
	})
	return out
}

// TailSet returns elements > from (or >= from if inclusive) (snapshot).
func (c *concurrentSkipSet[T]) TailSet(from T, inclusive bool) SortedSet[T] {
	out := NewTreeSetOrdered[T]()
	c.s.Range(func(v T) bool {
		if v > from || (inclusive && v == from) {
			out.Add(v)
		}
		return true
	})
	return out
}

// Rank returns the 0-based rank (index) of x; -1 if not present.
func (c *concurrentSkipSet[T]) Rank(x T) int {
	i := 0
	found := -1
	c.s.Range(func(v T) bool {
		if v == x {
			found = i
			return false
		}
		i++
		return true
	})
	return found
}

// GetByRank returns the element at rank (0-based).
func (c *concurrentSkipSet[T]) GetByRank(rank int) (T, bool) {
	var (
		zero T
		i    int
	)
	if rank < 0 {
		return zero, false
	}
	var res T
	var ok bool
	c.s.Range(func(v T) bool {
		if i == rank {
			res = v
			ok = true
			return false
		}
		i++
		return true
	})
	if ok {
		return res, true
	}
	return zero, false
}

// CloneSorted returns a shallow copy as a SortedSet snapshot.
func (c *concurrentSkipSet[T]) CloneSorted() SortedSet[T] {
	out := NewTreeSetOrdered[T]()
	c.s.Range(func(v T) bool { out.Add(v); return true })
	return out
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes elements in ascending order as a JSON array.
// NOTE: ConcurrentSkipSet only supports Ordered types, so deserialization
// can be done directly without providing a comparator.
func (c *concurrentSkipSet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
// Since ConcurrentSkipSet only supports Ordered types, no comparator is needed.
func (c *concurrentSkipSet[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	c.s = skipset.New[T]()
	for _, elem := range slice {
		c.s.Add(elem)
	}
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes elements in ascending order.
func (c *concurrentSkipSet[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(c.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (c *concurrentSkipSet[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	c.s = skipset.New[T]()
	for _, elem := range slice {
		c.s.Add(elem)
	}
	return nil
}

// Conformance.
var (
	_ ConcurrentSortedSet[int] = (*concurrentSkipSet[int])(nil)
)
