package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"
	"sync/atomic"
)

// lockFreeList is a lock-free concurrent linked list implementation.
// Uses Compare-and-Swap (CAS) operations for thread-safe modifications,
// with logical deletion (nodes are marked deleted, never unlinked).
//
// Characteristics:
//   - Lock-free: Progress guaranteed even if some threads are delayed
//   - High throughput under contention
//   - Suitable for small, hot lists in high-concurrency scenarios
//
// Atomicity:
//   - ATOMIC (per node): a node's value and its deletion state live in one
//     atomic pointer (nil = deleted), so Contains/Set/Remove observe and
//     change them in a single step — a Set and a removal of the same
//     element linearize against each other
//   - BEST-EFFORT: index-based operations (which element an index names can
//     shift under concurrent modification; they retry when they lose a race)
//   - NON-ATOMIC: bulk operations, iteration (snapshot semantics)
//
// Structure: head and tail are permanent sentinel nodes — they never change
// for the lifetime of the list, so every CAS races only on next pointers.
// Clear detaches the whole chain in one CAS and leaves reclamation to the
// garbage collector; there is no manual node recycling.
//
// Costs to be aware of:
//   - Add appends by walking the chain, so it is O(n) and building a large
//     list this way is O(n²).
//   - Removal is logical: deleted nodes stay in the chain (skipped by every
//     operation) until Clear detaches them, so traversal cost grows with the
//     number of removals since the last Clear.
//   - Size and IsEmpty traverse the chain (O(n), early exit for IsEmpty).
//     There is no element counter: a counter maintained beside the chain
//     cannot stay consistent with a concurrent Clear, so the chain itself is
//     the single source of truth. Get(index) is O(n).
type lockFreeList[T any] struct {
	head *lfNode[T] // permanent sentinel; head.next is the first element
	tail *lfNode[T] // permanent sentinel terminating the chain
	eq   Equaler[T]
}

// lfNode is a chain node. state holds the element while the node is live and
// nil once it is logically deleted, so value and deletion state are observed
// and changed in a single atomic step.
type lfNode[T any] struct {
	state atomic.Pointer[T]
	next  atomic.Pointer[lfNode[T]]
}

// NewLockFreeList creates a new lock-free list.
// The equaler function is used for element comparison.
func NewLockFreeList[T any](eq Equaler[T]) List[T] {
	l := &lockFreeList[T]{
		eq:   eq,
		head: &lfNode[T]{},
		tail: &lfNode[T]{},
	}
	l.head.next.Store(l.tail)
	return l
}

// NewLockFreeListOrdered creates a lock-free list for ordered types.
func NewLockFreeListOrdered[T comparable]() List[T] {
	return NewLockFreeList(func(a, b T) bool { return a == b })
}

// NewLockFreeListFrom creates a lock-free list from elements.
func NewLockFreeListFrom[T any](eq Equaler[T], elements ...T) List[T] {
	l := NewLockFreeList(eq)
	for _, e := range elements {
		l.Add(e)
	}
	return l
}

// newLFNode creates a live node holding value.
func newLFNode[T any](value T) *lfNode[T] {
	node := &lfNode[T]{}
	node.state.Store(&value)
	return node
}

// Size returns the number of elements by traversing the chain (O(n)).
// Under concurrent modification the count is a best-effort snapshot.
func (l *lockFreeList[T]) Size() int {
	n := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if curr.state.Load() != nil {
			n++
		}
	}
	return n
}

// IsEmpty reports whether the list appears empty (O(n) worst case, but stops
// at the first live element).
func (l *lockFreeList[T]) IsEmpty() bool {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if curr.state.Load() != nil {
			return false
		}
	}
	return true
}

// IsNotEmpty reports whether the list contains at least one element.
func (l *lockFreeList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements by detaching the chain from the head sentinel.
// The sentinels stay in place, so operations racing with Clear keep a valid
// chain to work on; an element added concurrently with Clear may be detached
// with the old chain (it linearizes before the Clear).
func (l *lockFreeList[T]) Clear() {
	for {
		first := l.head.next.Load()
		if l.head.next.CompareAndSwap(first, l.tail) {
			break
		}
	}
}

// ToSlice returns a snapshot of all elements.
func (l *lockFreeList[T]) ToSlice() []T {
	var result []T
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			result = append(result, *v)
		}
	}
	return result
}

// String returns a string representation.
func (l *lockFreeList[T]) String() string {
	return formatCollection("lockFreeList", l.Seq())
}

// Seq returns a sequence of elements (snapshot).
func (l *lockFreeList[T]) Seq() iter.Seq[T] {
	snap := l.ToSlice()
	return slices.Values(snap)
}

// ForEach applies action to each element.
func (l *lockFreeList[T]) ForEach(action func(element T) bool) {
	for v := range l.Seq() {
		if !action(v) {
			return
		}
	}
}

// Get returns the element at index (O(n) operation).
func (l *lockFreeList[T]) Get(index int) (T, bool) {
	if index < 0 {
		var zero T
		return zero, false
	}

	i := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			if i == index {
				return *v, true
			}
			i++
		}
	}

	var zero T
	return zero, false
}

// Set replaces the element at index (O(n) operation).
// The replacement is a single CAS on the node's state, so it linearizes
// against a concurrent removal of the same node: exactly one of them
// observes the old value.
func (l *lockFreeList[T]) Set(index int, element T) (T, bool) {
	if index < 0 {
		var zero T
		return zero, false
	}

retry:
	for {
		i := 0
		for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
			v := curr.state.Load()
			if v == nil {
				continue
			}
			if i != index {
				i++
				continue
			}
			for {
				if curr.state.CompareAndSwap(v, &element) {
					return *v, true
				}
				v = curr.state.Load()
				if v == nil {
					// The node was removed under us; the index now names a
					// different element, so resolve it again.
					continue retry
				}
			}
		}
		var zero T
		return zero, false
	}
}

// First returns the first element.
func (l *lockFreeList[T]) First() (T, bool) {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			return *v, true
		}
	}

	var zero T
	return zero, false
}

// Last returns the last element (O(n) operation).
func (l *lockFreeList[T]) Last() (T, bool) {
	var last *T
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			last = v
		}
	}

	if last != nil {
		return *last, true
	}
	var zero T
	return zero, false
}

// Add appends the element at the end (O(n): walks the chain to find it).
func (l *lockFreeList[T]) Add(element T) {
	node := newLFNode(element)

	for {
		// Find the actual last node (before the tail sentinel)
		pred := l.head
		curr := pred.next.Load()

		for curr != nil && curr != l.tail {
			pred = curr
			curr = curr.next.Load()
		}

		// Try to insert before tail
		node.next.Store(l.tail)
		if pred.next.CompareAndSwap(l.tail, node) {
			return
		}
		// CAS failed, retry
	}
}

// AddAll appends all elements.
func (l *lockFreeList[T]) AddAll(elements ...T) {
	for _, e := range elements {
		l.Add(e)
	}
}

// AddSeq appends all elements from the sequence.
func (l *lockFreeList[T]) AddSeq(seq iter.Seq[T]) {
	for v := range seq {
		l.Add(v)
	}
}

// Insert inserts the element at index. Valid indexes are 0 through Size().
func (l *lockFreeList[T]) Insert(index int, element T) bool {
	if index < 0 {
		return false
	}
	if index == 0 {
		return l.insertAtHead(element)
	}

	node := newLFNode(element)

	for {
		pred := l.head
		curr := pred.next.Load()
		i := 0
		found := false

		// Find the (index-1)-th live node to insert after it.
		for curr != nil && curr != l.tail {
			if curr.state.Load() != nil {
				if i == index-1 {
					pred = curr
					found = true
					break
				}
				i++
			}
			pred = curr
			curr = curr.next.Load()
		}

		if !found {
			// Fewer than index live elements: out of bounds.
			return false
		}

		next := pred.next.Load()
		node.next.Store(next)
		if pred.next.CompareAndSwap(next, node) {
			return true
		}
		// CAS failed, retry
	}
}

// insertAtHead inserts at the beginning.
func (l *lockFreeList[T]) insertAtHead(element T) bool {
	node := newLFNode(element)

	for {
		first := l.head.next.Load()
		node.next.Store(first)
		if l.head.next.CompareAndSwap(first, node) {
			return true
		}
	}
}

// InsertAll inserts all elements at index.
func (l *lockFreeList[T]) InsertAll(index int, elements ...T) bool {
	if len(elements) == 0 {
		return true
	}
	// Insert in reverse order to maintain order
	for _, e := range slices.Backward(elements) {
		if !l.Insert(index, e) {
			return false
		}
	}
	return true
}

// RemoveAt removes the element at index using logical deletion.
func (l *lockFreeList[T]) RemoveAt(index int) (T, bool) {
	if index < 0 {
		var zero T
		return zero, false
	}

retry:
	for {
		i := 0
		for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
			v := curr.state.Load()
			if v == nil {
				continue
			}
			if i != index {
				i++
				continue
			}
			if curr.state.CompareAndSwap(v, nil) {
				return *v, true
			}
			// Lost a race on this node; the index may now name a
			// different element, so resolve it again.
			continue retry
		}
		var zero T
		return zero, false
	}
}

// Remove removes the first occurrence of element.
func (l *lockFreeList[T]) Remove(element T, eq Equaler[T]) bool {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		for {
			v := curr.state.Load()
			if v == nil || !eq(*v, element) {
				break
			}
			if curr.state.CompareAndSwap(v, nil) {
				return true
			}
			// Lost a race on this node (removed or value swapped); re-read.
		}
	}
	return false
}

// RemoveFirst removes and returns the first element.
func (l *lockFreeList[T]) RemoveFirst() (T, bool) {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		for {
			v := curr.state.Load()
			if v == nil {
				break // Already removed; try the next node.
			}
			if curr.state.CompareAndSwap(v, nil) {
				return *v, true
			}
		}
	}

	var zero T
	return zero, false
}

// RemoveLast removes and returns the last element.
func (l *lockFreeList[T]) RemoveLast() (T, bool) {
	for {
		var (
			last    *lfNode[T]
			lastVal *T
		)
		for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
			if v := curr.state.Load(); v != nil {
				last, lastVal = curr, v
			}
		}

		if last == nil {
			var zero T
			return zero, false
		}

		if last.state.CompareAndSwap(lastVal, nil) {
			return *lastVal, true
		}
		// Retry if CAS failed
	}
}

// RemoveFunc removes all elements satisfying predicate.
func (l *lockFreeList[T]) RemoveFunc(predicate func(element T) bool) int {
	removed := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil && predicate(*v) {
			if curr.state.CompareAndSwap(v, nil) {
				removed++
			}
		}
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate.
func (l *lockFreeList[T]) RetainFunc(predicate func(element T) bool) int {
	removed := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil && !predicate(*v) {
			if curr.state.CompareAndSwap(v, nil) {
				removed++
			}
		}
	}
	return removed
}

// IndexOf returns the index of the first occurrence.
func (l *lockFreeList[T]) IndexOf(element T, eq Equaler[T]) int {
	i := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			if eq(*v, element) {
				return i
			}
			i++
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence.
func (l *lockFreeList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	lastIdx := -1
	i := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			if eq(*v, element) {
				lastIdx = i
			}
			i++
		}
	}
	return lastIdx
}

// Contains reports whether element exists.
func (l *lockFreeList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate.
func (l *lockFreeList[T]) Find(predicate func(element T) bool) (T, bool) {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil && predicate(*v) {
			return *v, true
		}
	}

	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *lockFreeList[T]) FindIndex(predicate func(element T) bool) int {
	i := 0
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil {
			if predicate(*v) {
				return i
			}
			i++
		}
	}
	return -1
}

// SubList returns a new list containing elements in [from, to).
func (l *lockFreeList[T]) SubList(from, to int) List[T] {
	snap := l.ToSlice()
	if from < 0 || to > len(snap) || from > to {
		return NewLockFreeList(l.eq)
	}
	return NewLockFreeListFrom(l.eq, snap[from:to]...)
}

// Reversed returns a sequence iterating in reverse order.
func (l *lockFreeList[T]) Reversed() iter.Seq[T] {
	snap := l.ToSlice()
	return func(yield func(T) bool) {
		for _, v := range slices.Backward(snap) {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a shallow copy.
func (l *lockFreeList[T]) Clone() List[T] {
	return NewLockFreeListFrom(l.eq, l.ToSlice()...)
}

// Filter returns a new list of elements satisfying predicate.
func (l *lockFreeList[T]) Filter(predicate func(element T) bool) List[T] {
	var result []T
	for _, v := range l.ToSlice() {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return NewLockFreeListFrom(l.eq, result...)
}

// Sort sorts elements (snapshot, clear and rebuild; not atomic as a whole).
func (l *lockFreeList[T]) Sort(cmp Comparator[T]) {
	snap := l.ToSlice()
	slices.SortFunc(snap, cmp)

	// Rebuild the list
	l.Clear()
	for _, v := range snap {
		l.Add(v)
	}
}

// Any returns true if at least one element satisfies predicate.
func (l *lockFreeList[T]) Any(predicate func(element T) bool) bool {
	_, ok := l.Find(predicate)
	return ok
}

// Every returns true if all elements satisfy predicate.
func (l *lockFreeList[T]) Every(predicate func(element T) bool) bool {
	for curr := l.head.next.Load(); curr != nil && curr != l.tail; curr = curr.next.Load() {
		if v := curr.state.Load(); v != nil && !predicate(*v) {
			return false
		}
	}
	return true
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes a snapshot of the list as a JSON array.
func (l *lockFreeList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *lockFreeList[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	// Clear and rebuild
	l.Clear()
	for _, elem := range slice {
		l.Add(elem)
	}
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the list.
func (l *lockFreeList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (l *lockFreeList[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	// Clear and rebuild
	l.Clear()
	for _, elem := range slice {
		l.Add(elem)
	}
	return nil
}

// Compile-time conformance.
var _ List[int] = (*lockFreeList[int])(nil)
