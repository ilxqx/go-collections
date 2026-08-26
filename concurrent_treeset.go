package collections

import (
	"bytes"
	"cmp"
	"encoding/gob"
	"encoding/json"
	"errors"
	"iter"
	"slices"
	"sync"
)

// concurrentTreeSet is a concurrent-safe sorted set implemented by guarding treeSet with RWMutex.
//   - Single-key operations are atomic under the mutex.
//   - Iteration methods (Seq/ForEach/Range*/Ascend*/Reversed) operate on snapshots to avoid holding locks
//     while invoking user callbacks.
//   - For operations that accept user-provided functions and need atomicity (e.g., AddIfAbsent/RemoveAndGet),
//     the functions are executed while holding the lock. Do not call back into the same set from those
//     callbacks to avoid deadlocks.
type concurrentTreeSet[T any] struct {
	mu   sync.RWMutex
	tree *treeSet[T]
}

// NewConcurrentTreeSet creates an empty concurrent sorted set with a custom comparator.
func NewConcurrentTreeSet[T any](cmpT Comparator[T]) ConcurrentSortedSet[T] {
	return &concurrentTreeSet[T]{tree: newTreeSet(cmpT)}
}

// NewConcurrentTreeSetOrdered creates an empty set for Ordered types.
func NewConcurrentTreeSetOrdered[T Ordered]() ConcurrentSortedSet[T] {
	return NewConcurrentTreeSet(cmp.Compare[T])
}

// NewConcurrentTreeSetFrom creates a set and inserts the given elements.
func NewConcurrentTreeSetFrom[T any](cmpT Comparator[T], elements ...T) ConcurrentSortedSet[T] {
	cs := &concurrentTreeSet[T]{tree: newTreeSet(cmpT)}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.tree.AddAll(elements...)
	return cs
}

// Size returns the number of elements.
func (c *concurrentTreeSet[T]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Size()
}

// IsEmpty reports whether the set is empty.
func (c *concurrentTreeSet[T]) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.IsEmpty()
}

// IsNotEmpty reports whether the set contains at least one element.
func (c *concurrentTreeSet[T]) IsNotEmpty() bool { return !c.IsEmpty() }

// Clear removes all elements.
func (c *concurrentTreeSet[T]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tree.Clear()
}

// ToSlice returns a snapshot of elements in comparator order.
func (c *concurrentTreeSet[T]) ToSlice() []T {
	c.mu.RLock()
	snap := c.tree.ToSlice()
	c.mu.RUnlock()
	return snap
}

// String returns a concise representation (order according to comparator).
func (c *concurrentTreeSet[T]) String() string {
	return formatCollection("concurrentTreeSet", c.Seq())
}

// Seq returns a sequence of elements in comparator order (snapshot).
func (c *concurrentTreeSet[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		snap := c.ToSlice()
		for _, v := range snap {
			if !yield(v) {
				return
			}
		}
	}
}

// ForEach iterates elements in comparator order over a snapshot.
func (c *concurrentTreeSet[T]) ForEach(action func(element T) bool) {
	for v := range c.Seq() {
		if !action(v) {
			return
		}
	}
}

// Add inserts element if absent. Returns true if the set changed.
func (c *concurrentTreeSet[T]) Add(element T) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.Add(element)
}

// AddAll inserts all given elements. Returns number added.
func (c *concurrentTreeSet[T]) AddAll(elements ...T) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.AddAll(elements...)
}

// AddSeq inserts all elements from the sequence. Returns number added.
func (c *concurrentTreeSet[T]) AddSeq(seq iter.Seq[T]) int {
	// Avoid holding lock while consuming external sequence.
	buf := make([]T, 0, 16)
	for v := range seq {
		buf = append(buf, v)
	}
	if len(buf) == 0 {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	added := 0
	for _, v := range buf {
		if c.tree.Add(v) {
			added++
		}
	}
	return added
}

// Remove deletes the element if present. Returns true if removed.
func (c *concurrentTreeSet[T]) Remove(element T) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.Remove(element)
}

// RemoveAll deletes all given elements. Returns the number removed.
func (c *concurrentTreeSet[T]) RemoveAll(elements ...T) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.RemoveAll(elements...)
}

// RemoveSeq removes all elements from the sequence. Returns the number removed.
func (c *concurrentTreeSet[T]) RemoveSeq(seq iter.Seq[T]) int {
	buf := make([]T, 0, 16)
	for v := range seq {
		buf = append(buf, v)
	}
	if len(buf) == 0 {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for _, v := range buf {
		if c.tree.Remove(v) {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes elements satisfying predicate. Returns count removed.
func (c *concurrentTreeSet[T]) RemoveFunc(predicate func(element T) bool) int {
	// Snapshot under RLock, evaluate predicates without lock, then remove under Lock.
	c.mu.RLock()
	snap := c.tree.ToSlice()
	c.mu.RUnlock()
	size := len(snap)
	if size == 0 {
		return 0
	}
	dels := make([]T, 0, size/2)
	for _, v := range snap {
		if predicate(v) {
			dels = append(dels, v)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for _, v := range dels {
		if c.tree.Remove(v) {
			removed++
		}
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (c *concurrentTreeSet[T]) RetainFunc(predicate func(element T) bool) int {
	c.mu.RLock()
	snap := c.tree.ToSlice()
	c.mu.RUnlock()
	size := len(snap)
	if size == 0 {
		return 0
	}
	dels := make([]T, 0, size/2)
	for _, v := range snap {
		if !predicate(v) {
			dels = append(dels, v)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	removed := 0
	for _, v := range dels {
		if c.tree.Remove(v) {
			removed++
		}
	}
	return removed
}

// Pop removes and returns the smallest element.
func (c *concurrentTreeSet[T]) Pop() (T, bool) { return c.PopFirst() }

// Contains reports whether element exists in the set.
func (c *concurrentTreeSet[T]) Contains(element T) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Contains(element)
}

// ContainsAll reports whether all elements exist in the set.
func (c *concurrentTreeSet[T]) ContainsAll(elements ...T) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.ContainsAll(elements...)
}

// ContainsAny reports whether any of the elements exist in the set.
func (c *concurrentTreeSet[T]) ContainsAny(elements ...T) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.ContainsAny(elements...)
}

// Union returns a new set containing elements from both sets (sorted).
func (c *concurrentTreeSet[T]) Union(other Set[T]) Set[T] {
	c.mu.RLock()
	out := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool { out.bt.Set(item); return true })
	c.mu.RUnlock()
	for v := range other.Seq() {
		out.Add(v)
	}
	return out
}

// Intersection returns a new set with common elements (sorted).
func (c *concurrentTreeSet[T]) Intersection(other Set[T]) Set[T] {
	c.mu.RLock()
	out := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool {
		if other.Contains(item) {
			out.bt.Set(item)
		}
		return true
	})
	c.mu.RUnlock()
	return out
}

// Difference returns a new set with elements in this set but not in other (sorted).
func (c *concurrentTreeSet[T]) Difference(other Set[T]) Set[T] {
	c.mu.RLock()
	out := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			out.bt.Set(item)
		}
		return true
	})
	c.mu.RUnlock()
	return out
}

// SymmetricDifference returns a new set with elements in either set but not both (sorted).
func (c *concurrentTreeSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	// First half from c
	c.mu.RLock()
	out := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			out.bt.Set(item)
		}
		return true
	})
	c.mu.RUnlock()
	// Second half from other
	for v := range other.Seq() {
		if !c.Contains(v) {
			out.Add(v)
		}
	}
	return out
}

// IsSubsetOf reports whether all elements of this set are in other.
func (c *concurrentTreeSet[T]) IsSubsetOf(other Set[T]) bool {
	c.mu.RLock()
	ok := true
	c.tree.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			ok = false
			return false
		}
		return true
	})
	c.mu.RUnlock()
	return ok
}

// IsSupersetOf reports whether this set contains all elements of other.
func (c *concurrentTreeSet[T]) IsSupersetOf(other Set[T]) bool { return other.IsSubsetOf(c) }

// IsProperSubsetOf reports whether this set is a strict subset of other.
func (c *concurrentTreeSet[T]) IsProperSubsetOf(other Set[T]) bool {
	return c.Size() < other.Size() && c.IsSubsetOf(other)
}

// IsProperSupersetOf reports whether this set is a strict superset of other.
func (c *concurrentTreeSet[T]) IsProperSupersetOf(other Set[T]) bool {
	return c.Size() > other.Size() && c.IsSupersetOf(other)
}

// IsDisjoint reports whether this set and other share no common elements.
func (c *concurrentTreeSet[T]) IsDisjoint(other Set[T]) bool {
	disjoint := true
	c.mu.RLock()
	c.tree.bt.Scan(func(item T) bool {
		if other.Contains(item) {
			disjoint = false
			return false
		}
		return true
	})
	c.mu.RUnlock()
	return disjoint
}

// Equals reports whether this set and other contain exactly the same elements.
func (c *concurrentTreeSet[T]) Equals(other Set[T]) bool {
	// Snapshot into a TreeSet and compare to avoid holding locks across other.Seq().
	c.mu.RLock()
	snap := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool { snap.bt.Set(item); return true })
	c.mu.RUnlock()
	return snap.Equals(other)
}

// Clone returns a shallow copy (as Set).
func (c *concurrentTreeSet[T]) Clone() Set[T] { return c.CloneSorted() }

// Filter returns a new set with elements satisfying predicate (sorted).
func (c *concurrentTreeSet[T]) Filter(predicate func(element T) bool) Set[T] {
	c.mu.RLock()
	out := newTreeSet(c.tree.cmp)
	c.tree.bt.Scan(func(item T) bool {
		if predicate(item) {
			out.bt.Set(item)
		}
		return true
	})
	c.mu.RUnlock()
	return out
}

// Find returns the first element satisfying predicate (comparator order) over snapshot.
func (c *concurrentTreeSet[T]) Find(predicate func(element T) bool) (T, bool) {
	snap := c.ToSlice()
	for _, v := range snap {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Any returns true if at least one element satisfies predicate over snapshot.
func (c *concurrentTreeSet[T]) Any(predicate func(element T) bool) bool {
	for v := range c.Seq() {
		if predicate(v) {
			return true
		}
	}
	return false
}

// Every returns true if all elements satisfy predicate over snapshot.
func (c *concurrentTreeSet[T]) Every(predicate func(element T) bool) bool {
	for v := range c.Seq() {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// AddIfAbsent atomically inserts element only if currently absent.
func (c *concurrentTreeSet[T]) AddIfAbsent(element T) bool { return c.Add(element) }

// RemoveAndGet atomically removes and returns the element if present.
func (c *concurrentTreeSet[T]) RemoveAndGet(element T) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tree.Remove(element) {
		return element, true
	}
	var zero T
	return zero, false
}

// First returns the smallest element.
func (c *concurrentTreeSet[T]) First() (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.First()
}

// Last returns the largest element.
func (c *concurrentTreeSet[T]) Last() (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Last()
}

// Min alias of First.
func (c *concurrentTreeSet[T]) Min() (T, bool) { return c.First() }

// Max alias of Last.
func (c *concurrentTreeSet[T]) Max() (T, bool) { return c.Last() }

// PopFirst removes and returns the smallest element.
func (c *concurrentTreeSet[T]) PopFirst() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.PopFirst()
}

// PopLast removes and returns the largest element.
func (c *concurrentTreeSet[T]) PopLast() (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.tree.PopLast()
}

// Floor returns the greatest element <= x.
func (c *concurrentTreeSet[T]) Floor(x T) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Floor(x)
}

// Ceiling returns the smallest element >= x.
func (c *concurrentTreeSet[T]) Ceiling(x T) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Ceiling(x)
}

// Lower returns the greatest element < x.
func (c *concurrentTreeSet[T]) Lower(x T) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Lower(x)
}

// Higher returns the smallest element > x.
func (c *concurrentTreeSet[T]) Higher(x T) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Higher(x)
}

// Range iterates elements in [from, to] ascending over a snapshot.
func (c *concurrentTreeSet[T]) Range(from, to T, action func(element T) bool) {
	snap := func() []T {
		c.mu.RLock()
		defer c.mu.RUnlock()
		buf := make([]T, 0, c.tree.Size())
		c.tree.Range(from, to, func(e T) bool {
			buf = append(buf, e)
			return true
		})
		return buf
	}()
	for _, v := range snap {
		if !action(v) {
			return
		}
	}
}

// RangeSeq returns a sequence for elements in [from, to] ascending (snapshot).
func (c *concurrentTreeSet[T]) RangeSeq(from, to T) iter.Seq[T] {
	return func(yield func(T) bool) {
		c.Range(from, to, func(e T) bool { return yield(e) })
	}
}

// Ascend iterates all elements in ascending order over snapshot.
func (c *concurrentTreeSet[T]) Ascend(action func(element T) bool) {
	for v := range c.Seq() {
		if !action(v) {
			return
		}
	}
}

// Descend iterates all elements in descending order over snapshot.
func (c *concurrentTreeSet[T]) Descend(action func(element T) bool) {
	snap := c.ToSlice()
	for _, v := range slices.Backward(snap) {
		if !action(v) {
			return
		}
	}
}

// AscendFrom iterates elements >= pivot ascending over snapshot.
func (c *concurrentTreeSet[T]) AscendFrom(pivot T, action func(element T) bool) {
	snap := c.ToSlice()
	for _, v := range snap {
		if c.tree.cmp(v, pivot) >= 0 {
			if !action(v) {
				return
			}
		}
	}
}

// DescendFrom iterates elements <= pivot descending over snapshot.
func (c *concurrentTreeSet[T]) DescendFrom(pivot T, action func(element T) bool) {
	snap := c.ToSlice()
	for _, v := range slices.Backward(snap) {
		if c.tree.cmp(v, pivot) <= 0 {
			if !action(v) {
				return
			}
		}
	}
}

// Reversed returns a descending sequence (snapshot).
func (c *concurrentTreeSet[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		snap := c.ToSlice()
		for _, v := range slices.Backward(snap) {
			if !yield(v) {
				return
			}
		}
	}
}

// SubSet returns a new set containing elements in [from, to] (snapshot sorted).
func (c *concurrentTreeSet[T]) SubSet(from, to T) SortedSet[T] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.SubSet(from, to)
}

// HeadSet returns elements < to (or <= to if inclusive) (snapshot sorted).
func (c *concurrentTreeSet[T]) HeadSet(to T, inclusive bool) SortedSet[T] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.HeadSet(to, inclusive)
}

// TailSet returns elements > from (or >= from if inclusive) (snapshot sorted).
func (c *concurrentTreeSet[T]) TailSet(from T, inclusive bool) SortedSet[T] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.TailSet(from, inclusive)
}

// Rank returns the 0-based rank of x; -1 if not present.
func (c *concurrentTreeSet[T]) Rank(x T) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.Rank(x)
}

// GetByRank returns the element at rank (0-based).
func (c *concurrentTreeSet[T]) GetByRank(rank int) (T, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.GetByRank(rank)
}

// CloneSorted returns a shallow copy as SortedSet.
func (c *concurrentTreeSet[T]) CloneSorted() SortedSet[T] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tree.CloneSorted()
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes elements in ascending order as a JSON array.
//
// NOTE: The comparator is NOT serialized. Decode into a set constructed with
// NewConcurrentTreeSet.
func (c *concurrentTreeSet[T]) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(c.tree.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes into the receiver using its existing comparator, so construct
// the set with NewConcurrentTreeSet before decoding.
func (c *concurrentTreeSet[T]) UnmarshalJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tree == nil {
		return errors.New("unmarshal concurrent treeset: no comparator; construct the set with NewConcurrentTreeSet before decoding")
	}
	return c.tree.UnmarshalJSON(data)
}

// GobEncode implements gob.GobEncoder.
// Serializes elements in ascending order.
//
// NOTE: The comparator is NOT serialized. Decode into a set constructed with
// NewConcurrentTreeSet.
func (c *concurrentTreeSet[T]) GobEncode() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(c.tree.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes into the receiver using its existing comparator, so construct
// the set with NewConcurrentTreeSet before decoding.
func (c *concurrentTreeSet[T]) GobDecode(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tree == nil {
		return errors.New("unmarshal concurrent treeset: no comparator; construct the set with NewConcurrentTreeSet before decoding")
	}
	return c.tree.GobDecode(data)
}

// Conformance.
var (
	_ ConcurrentSortedSet[int] = (*concurrentTreeSet[int])(nil)
)
