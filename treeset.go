package collections

import (
	"bytes"
	"cmp"
	"encoding/gob"
	"encoding/json"
	"errors"
	"iter"

	"github.com/tidwall/btree"
)

// treeSet is a sorted set backed by a B-Tree.
// Ordering is defined by the provided Comparator[T].
// Not concurrent-safe.
type treeSet[T any] struct {
	bt  *btree.BTreeG[T]
	cmp Comparator[T]
}

func newTreeSet[T any](c Comparator[T]) *treeSet[T] {
	if c == nil {
		panic("NewTreeSet: comparator must not be nil")
	}
	less := func(a, b T) bool { return c(a, b) < 0 }
	return &treeSet[T]{bt: btree.NewBTreeG(less), cmp: c}
}

// NewTreeSet creates an empty SortedSet with a custom comparator.
func NewTreeSet[T any](c Comparator[T]) SortedSet[T] {
	return newTreeSet(c)
}

// NewTreeSetOrdered creates an empty TreeSet for Ordered types.
func NewTreeSetOrdered[T Ordered]() SortedSet[T] {
	return newTreeSet(cmp.Compare[T])
}

// NewTreeSetFrom creates a TreeSet and inserts all elements.
func NewTreeSetFrom[T any](c Comparator[T], elements ...T) SortedSet[T] {
	ts := newTreeSet(c)
	for _, e := range elements {
		ts.add(e)
	}
	return ts
}

// Size returns the number of elements.
func (t *treeSet[T]) Size() int { return t.bt.Len() }

// IsEmpty reports whether the set is empty.
func (t *treeSet[T]) IsEmpty() bool    { return t.bt.Len() == 0 }
func (t *treeSet[T]) IsNotEmpty() bool { return !t.IsEmpty() }

// Clear removes all elements.
func (t *treeSet[T]) Clear() { t.bt.Clear() }

// ToSlice returns a snapshot of all elements in ascending order.
func (t *treeSet[T]) ToSlice() []T {
	out := make([]T, 0, t.bt.Len())
	t.bt.Scan(func(item T) bool {
		out = append(out, item)
		return true
	})
	return out
}

// String returns a concise representation.
func (t *treeSet[T]) String() string {
	return formatCollection("treeSet", t.Seq())
}

// Seq returns a sequence of elements in ascending order.
func (t *treeSet[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		t.bt.Scan(func(item T) bool {
			return yield(item)
		})
	}
}

// ForEach iterates elements ascending; stops early if action returns false.
func (t *treeSet[T]) ForEach(action func(element T) bool) {
	t.bt.Scan(func(item T) bool {
		return action(item)
	})
}

// add inserts element only if no comparator-equivalent element is present.
// The existing element is kept (never replaced), so a false return always
// means the set did not change.
func (t *treeSet[T]) add(element T) bool {
	if _, ok := t.bt.Get(element); ok {
		return false
	}
	t.bt.Set(element)
	return true
}

// Add inserts element if absent. Returns true if the set changed.
// If a comparator-equivalent element is already present, it is kept unchanged.
func (t *treeSet[T]) Add(element T) bool {
	return t.add(element)
}

// AddAll inserts all given elements. Returns number added.
func (t *treeSet[T]) AddAll(elements ...T) int {
	added := 0
	for _, e := range elements {
		if t.add(e) {
			added++
		}
	}
	return added
}

// AddSeq inserts all elements from the sequence. Returns number added.
func (t *treeSet[T]) AddSeq(seq iter.Seq[T]) int {
	added := 0
	for v := range seq {
		if t.add(v) {
			added++
		}
	}
	return added
}

// Remove deletes the element if present. Returns true if removed.
func (t *treeSet[T]) Remove(element T) bool {
	_, ok := t.bt.Delete(element)
	return ok
}

// RemoveAll deletes all given elements. Returns the number removed.
func (t *treeSet[T]) RemoveAll(elements ...T) int {
	removed := 0
	for _, e := range elements {
		if _, ok := t.bt.Delete(e); ok {
			removed++
		}
	}
	return removed
}

// RemoveSeq removes all elements from the sequence. Returns the number removed.
func (t *treeSet[T]) RemoveSeq(seq iter.Seq[T]) int {
	removed := 0
	for v := range seq {
		if _, ok := t.bt.Delete(v); ok {
			removed++
		}
	}
	return removed
}

// RemoveFunc removes elements satisfying predicate. Returns count removed.
func (t *treeSet[T]) RemoveFunc(predicate func(element T) bool) int {
	// Use a conservative capacity to avoid over-allocating on sparse removals.
	capacity := max(t.bt.Len()/4, 16)
	dels := make([]T, 0, capacity)
	t.bt.Scan(func(item T) bool {
		if predicate(item) {
			dels = append(dels, item)
		}
		return true
	})
	for _, v := range dels {
		t.bt.Delete(v)
	}
	return len(dels)
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (t *treeSet[T]) RetainFunc(predicate func(element T) bool) int {
	capacity := max(t.bt.Len()/4, 16)
	dels := make([]T, 0, capacity)
	t.bt.Scan(func(item T) bool {
		if !predicate(item) {
			dels = append(dels, item)
		}
		return true
	})
	for _, v := range dels {
		t.bt.Delete(v)
	}
	return len(dels)
}

// Pop removes and returns the smallest element. Returns (zero, false) if empty.
func (t *treeSet[T]) Pop() (T, bool) {
	return t.PopFirst()
}

// Contains reports whether element exists in the set.
func (t *treeSet[T]) Contains(element T) bool {
	_, ok := t.bt.Get(element)
	return ok
}

// ContainsAll reports whether all elements exist in the set.
func (t *treeSet[T]) ContainsAll(elements ...T) bool {
	for _, e := range elements {
		if _, ok := t.bt.Get(e); !ok {
			return false
		}
	}
	return true
}

// ContainsAny reports whether any of the elements exist in the set.
func (t *treeSet[T]) ContainsAny(elements ...T) bool {
	for _, e := range elements {
		if _, ok := t.bt.Get(e); ok {
			return true
		}
	}
	return false
}

// Union returns a new sorted set with elements from both sets.
func (t *treeSet[T]) Union(other Set[T]) Set[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		out.bt.Set(item)
		return true
	})
	for v := range other.Seq() {
		out.add(v)
	}
	return out
}

// Intersection returns a new sorted set with common elements.
func (t *treeSet[T]) Intersection(other Set[T]) Set[T] {
	small, large := Set[T](t), other
	if other.Size() < t.Size() {
		small, large = other, t
	}
	out := newTreeSet(t.cmp)
	for v := range small.Seq() {
		if large.Contains(v) {
			out.add(v)
		}
	}
	return out
}

// Difference returns a new sorted set with elements in t but not in other.
func (t *treeSet[T]) Difference(other Set[T]) Set[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			out.bt.Set(item)
		}
		return true
	})
	return out
}

// SymmetricDifference returns a new sorted set with elements in either set but not both.
func (t *treeSet[T]) SymmetricDifference(other Set[T]) Set[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			out.bt.Set(item)
		}
		return true
	})
	for v := range other.Seq() {
		if _, ok := t.bt.Get(v); !ok {
			out.add(v)
		}
	}
	return out
}

// IsSubsetOf reports whether all elements of t are in other.
func (t *treeSet[T]) IsSubsetOf(other Set[T]) bool {
	ok := true
	t.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			ok = false
			return false
		}
		return true
	})
	return ok
}

// IsSupersetOf reports whether t contains all elements of other.
func (t *treeSet[T]) IsSupersetOf(other Set[T]) bool {
	return other.IsSubsetOf(t)
}

// IsProperSubsetOf reports whether t is a strict subset of other.
func (t *treeSet[T]) IsProperSubsetOf(other Set[T]) bool {
	return t.Size() < other.Size() && t.IsSubsetOf(other)
}

// IsProperSupersetOf reports whether t is a strict superset of other.
func (t *treeSet[T]) IsProperSupersetOf(other Set[T]) bool {
	return t.Size() > other.Size() && t.IsSupersetOf(other)
}

// IsDisjoint reports whether t and other share no common elements.
func (t *treeSet[T]) IsDisjoint(other Set[T]) bool {
	if other.Size() < t.Size() {
		for v := range other.Seq() {
			if t.Contains(v) {
				return false
			}
		}
		return true
	}
	disjoint := true
	t.bt.Scan(func(item T) bool {
		if other.Contains(item) {
			disjoint = false
			return false
		}
		return true
	})
	return disjoint
}

// Equals reports whether t and other contain exactly the same elements.
func (t *treeSet[T]) Equals(other Set[T]) bool {
	if t.Size() != other.Size() {
		return false
	}
	ok := true
	t.bt.Scan(func(item T) bool {
		if !other.Contains(item) {
			ok = false
			return false
		}
		return true
	})
	return ok
}

// Clone returns a shallow copy (as Set).
func (t *treeSet[T]) Clone() Set[T] {
	return t.CloneSorted()
}

// Filter returns a new set with elements satisfying predicate.
func (t *treeSet[T]) Filter(predicate func(element T) bool) Set[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		if predicate(item) {
			out.bt.Set(item)
		}
		return true
	})
	return out
}

// Find returns the first element satisfying predicate (ascending order).
func (t *treeSet[T]) Find(predicate func(element T) bool) (T, bool) {
	var res T
	found := false
	t.bt.Scan(func(item T) bool {
		if predicate(item) {
			res = item
			found = true
			return false
		}
		return true
	})
	return res, found
}

// Any returns true if at least one element satisfies predicate.
func (t *treeSet[T]) Any(predicate func(element T) bool) bool {
	ans := false
	t.bt.Scan(func(item T) bool {
		if predicate(item) {
			ans = true
			return false
		}
		return true
	})
	return ans
}

// Every returns true if all elements satisfy predicate (true for empty set).
func (t *treeSet[T]) Every(predicate func(element T) bool) bool {
	all := true
	t.bt.Scan(func(item T) bool {
		if !predicate(item) {
			all = false
			return false
		}
		return true
	})
	return all
}

// First returns the smallest element, or (zero, false) if empty.
func (t *treeSet[T]) First() (T, bool) {
	return t.bt.Min()
}

// Last returns the largest element, or (zero, false) if empty.
func (t *treeSet[T]) Last() (T, bool) {
	return t.bt.Max()
}

// Min is an alias for First.
func (t *treeSet[T]) Min() (T, bool) { return t.First() }

// Max is an alias for Last.
func (t *treeSet[T]) Max() (T, bool) { return t.Last() }

// PopFirst removes and returns the smallest element.
func (t *treeSet[T]) PopFirst() (T, bool) {
	first, ok := t.bt.Min()
	if !ok {
		var zero T
		return zero, false
	}
	t.bt.Delete(first)
	return first, true
}

// PopLast removes and returns the largest element.
func (t *treeSet[T]) PopLast() (T, bool) {
	last, ok := t.bt.Max()
	if !ok {
		var zero T
		return zero, false
	}
	t.bt.Delete(last)
	return last, true
}

// Floor returns the greatest element <= x.
func (t *treeSet[T]) Floor(x T) (T, bool) {
	var res T
	found := false
	t.bt.Descend(x, func(item T) bool {
		res = item
		found = true
		return false
	})
	return res, found
}

// Ceiling returns the smallest element >= x.
func (t *treeSet[T]) Ceiling(x T) (T, bool) {
	var res T
	found := false
	t.bt.Ascend(x, func(item T) bool {
		res = item
		found = true
		return false
	})
	return res, found
}

// Lower returns the greatest element strictly < x.
func (t *treeSet[T]) Lower(x T) (T, bool) {
	var res T
	found := false
	t.bt.Descend(x, func(item T) bool {
		if t.cmp(item, x) < 0 {
			res = item
			found = true
			return false
		}
		return true // skip equal
	})
	return res, found
}

// Higher returns the smallest element strictly > x.
func (t *treeSet[T]) Higher(x T) (T, bool) {
	var res T
	found := false
	t.bt.Ascend(x, func(item T) bool {
		if t.cmp(item, x) > 0 {
			res = item
			found = true
			return false
		}
		return true // skip equal
	})
	return res, found
}

// Range iterates elements in [from, to] ascending.
func (t *treeSet[T]) Range(from, to T, action func(element T) bool) {
	if t.cmp(from, to) > 0 {
		return
	}
	t.bt.Ascend(from, func(item T) bool {
		if t.cmp(item, to) > 0 {
			return false
		}
		return action(item)
	})
}

// RangeSeq returns a sequence for elements in [from, to] ascending.
func (t *treeSet[T]) RangeSeq(from, to T) iter.Seq[T] {
	return func(yield func(T) bool) {
		if t.cmp(from, to) > 0 {
			return
		}
		t.bt.Ascend(from, func(item T) bool {
			if t.cmp(item, to) > 0 {
				return false
			}
			return yield(item)
		})
	}
}

// Ascend iterates all elements in ascending order.
func (t *treeSet[T]) Ascend(action func(element T) bool) {
	t.bt.Scan(action)
}

// Descend iterates all elements in descending order.
func (t *treeSet[T]) Descend(action func(element T) bool) {
	t.bt.Reverse(action)
}

// AscendFrom iterates elements >= pivot in ascending order.
func (t *treeSet[T]) AscendFrom(pivot T, action func(element T) bool) {
	t.bt.Ascend(pivot, action)
}

// DescendFrom iterates elements <= pivot in descending order.
func (t *treeSet[T]) DescendFrom(pivot T, action func(element T) bool) {
	t.bt.Descend(pivot, action)
}

// Reversed returns a descending sequence.
func (t *treeSet[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		t.bt.Reverse(func(item T) bool { return yield(item) })
	}
}

// SubSet returns a new set containing elements in [from, to].
func (t *treeSet[T]) SubSet(from, to T) SortedSet[T] {
	out := newTreeSet(t.cmp)
	if t.cmp(from, to) > 0 {
		return out
	}
	t.bt.Ascend(from, func(item T) bool {
		if t.cmp(item, to) > 0 {
			return false
		}
		out.bt.Set(item)
		return true
	})
	return out
}

// HeadSet returns elements < to (or <= to if inclusive).
func (t *treeSet[T]) HeadSet(to T, inclusive bool) SortedSet[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		c := t.cmp(item, to)
		if c < 0 || (inclusive && c == 0) {
			out.bt.Set(item)
			return true
		}
		return c < 0
	})
	return out
}

// TailSet returns elements > from (or >= from if inclusive).
func (t *treeSet[T]) TailSet(from T, inclusive bool) SortedSet[T] {
	out := newTreeSet(t.cmp)
	t.bt.Ascend(from, func(item T) bool {
		c := t.cmp(item, from)
		if c > 0 || (inclusive && c == 0) {
			out.bt.Set(item)
		}
		return true
	})
	return out
}

// Rank returns the 0-based rank (index) of x; -1 if not present. O(n): the
// backing B-tree offers no item-to-rank lookup, so this scans.
func (t *treeSet[T]) Rank(x T) int {
	idx := 0
	found := -1
	t.bt.Scan(func(item T) bool {
		if t.cmp(item, x) == 0 {
			found = idx
			return false
		}
		idx++
		return true
	})
	return found
}

// GetByRank returns the element at rank (0-based).
func (t *treeSet[T]) GetByRank(rank int) (T, bool) {
	return t.bt.GetAt(rank)
}

// CloneSorted returns a shallow copy as SortedSet.
func (t *treeSet[T]) CloneSorted() SortedSet[T] {
	out := newTreeSet(t.cmp)
	t.bt.Scan(func(item T) bool {
		out.bt.Set(item)
		return true
	})
	return out
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes elements in ascending order as a JSON array.
//
// NOTE: The comparator is NOT serialized. Decode into a set constructed with
// NewTreeSet, or use UnmarshalTreeSetJSON / UnmarshalTreeSetOrderedJSON.
func (t *treeSet[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes a JSON array into the receiver using its existing comparator,
// so construct the set with NewTreeSet (or a helper) before decoding.
func (t *treeSet[T]) UnmarshalJSON(data []byte) error {
	if t.cmp == nil {
		return errors.New("unmarshal treeset: no comparator; construct the set with NewTreeSet before decoding, or use UnmarshalTreeSetJSON/UnmarshalTreeSetOrderedJSON")
	}
	var elements []T
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}
	t.Clear()
	for _, e := range elements {
		t.add(e)
	}
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes elements in ascending order.
//
// NOTE: The comparator is NOT serialized. Decode into a set constructed with
// NewTreeSet, or use UnmarshalTreeSetGob / UnmarshalTreeSetOrderedGob.
func (t *treeSet[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(t.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes into the receiver using its existing comparator, so construct
// the set with NewTreeSet (or a helper) before decoding.
func (t *treeSet[T]) GobDecode(data []byte) error {
	if t.cmp == nil {
		return errors.New("unmarshal treeset: no comparator; construct the set with NewTreeSet before decoding, or use UnmarshalTreeSetGob/UnmarshalTreeSetOrderedGob")
	}
	var elements []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&elements); err != nil {
		return err
	}
	t.Clear()
	for _, e := range elements {
		t.add(e)
	}
	return nil
}

// Compile-time conformance check.
var (
	_ SortedSet[int] = (*treeSet[int])(nil)
)
