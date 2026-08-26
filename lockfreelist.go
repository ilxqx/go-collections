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
//   - ATOMIC: Contains, Add, Set (per element), Size (approximate)
//   - BEST-EFFORT: Remove, Insert (may retry under contention)
//   - NON-ATOMIC: Bulk operations, iteration (snapshot semantics)
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
//   - Size is approximate under concurrent modification; Get(index) is O(n).
type lockFreeList[T any] struct {
	head *lfNode[T] // permanent sentinel; head.next is the first element
	tail *lfNode[T] // permanent sentinel terminating the chain
	size atomic.Int64
	eq   Equaler[T]
}

type lfNode[T any] struct {
	value   atomic.Pointer[T]
	next    atomic.Pointer[lfNode[T]]
	deleted atomic.Bool // Logical deletion marker
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

// newLFNode creates a node holding value.
func newLFNode[T any](value T) *lfNode[T] {
	node := &lfNode[T]{}
	node.value.Store(&value)
	return node
}

// Size returns the approximate number of elements.
// Note: Due to concurrent modifications, this may not be exact.
func (l *lockFreeList[T]) Size() int {
	return int(l.size.Load())
}

// IsEmpty reports whether the list appears empty.
func (l *lockFreeList[T]) IsEmpty() bool {
	return l.Size() == 0
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
	l.size.Store(0)
}

// ToSlice returns a snapshot of all elements.
func (l *lockFreeList[T]) ToSlice() []T {
	size := l.Size()
	if size == 0 {
		return nil
	}
	result := make([]T, 0, size)
	curr := l.head.next.Load()
	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			result = append(result, *curr.value.Load())
		}
		curr = curr.next.Load()
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

	curr := l.head.next.Load()
	i := 0

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if i == index {
				return *curr.value.Load(), true
			}
			i++
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// Set replaces the element at index (O(n) operation).
// The value swap itself is atomic per node.
func (l *lockFreeList[T]) Set(index int, element T) (T, bool) {
	if index < 0 {
		var zero T
		return zero, false
	}

	curr := l.head.next.Load()
	i := 0

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if i == index {
				old := curr.value.Swap(&element)
				return *old, true
			}
			i++
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// First returns the first element.
func (l *lockFreeList[T]) First() (T, bool) {
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			return *curr.value.Load(), true
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// Last returns the last element (O(n) operation).
func (l *lockFreeList[T]) Last() (T, bool) {
	var last *lfNode[T]
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			last = curr
		}
		curr = curr.next.Load()
	}

	if last != nil {
		return *last.value.Load(), true
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
			l.size.Add(1)
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

// Insert inserts the element at index.
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

		// Find the node at position (index-1) to insert after it
		for curr != nil && curr != l.tail {
			if !curr.deleted.Load() {
				if i == index-1 {
					// Found the predecessor, insert after curr
					pred = curr
					break
				}
				i++
			}
			pred = curr
			curr = curr.next.Load()
		}

		// Check if index is out of bounds
		// Note: when i == index-1, we found the right position to insert after
		// When curr == tail && i < index-1, index is beyond current size
		if i != index-1 && curr == l.tail && i < index-1 {
			return false // Index out of bounds
		}

		next := pred.next.Load()
		node.next.Store(next)
		if pred.next.CompareAndSwap(next, node) {
			l.size.Add(1)
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
			l.size.Add(1)
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
	for i := len(elements) - 1; i >= 0; i-- {
		if !l.Insert(index, elements[i]) {
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

	for {
		curr := l.head.next.Load()
		i := 0

		for curr != nil && curr != l.tail {
			if !curr.deleted.Load() {
				if i == index {
					// Logically delete
					if curr.deleted.CompareAndSwap(false, true) {
						l.size.Add(-1)
						return *curr.value.Load(), true
					}
					// Someone else deleted it, retry
					break
				}
				i++
			}
			curr = curr.next.Load()
		}

		if curr == nil || curr == l.tail {
			var zero T
			return zero, false
		}
	}
}

// Remove removes the first occurrence of element.
func (l *lockFreeList[T]) Remove(element T, eq Equaler[T]) bool {
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() && eq(*curr.value.Load(), element) {
			if curr.deleted.CompareAndSwap(false, true) {
				l.size.Add(-1)
				return true
			}
			// Someone else deleted it, continue searching
		}
		curr = curr.next.Load()
	}
	return false
}

// RemoveFirst removes and returns the first element.
func (l *lockFreeList[T]) RemoveFirst() (T, bool) {
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if curr.deleted.CompareAndSwap(false, true) {
				l.size.Add(-1)
				return *curr.value.Load(), true
			}
			// Someone else deleted it, try next
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// RemoveLast removes and returns the last element.
func (l *lockFreeList[T]) RemoveLast() (T, bool) {
	for {
		var last *lfNode[T]
		curr := l.head.next.Load()

		for curr != nil && curr != l.tail {
			if !curr.deleted.Load() {
				last = curr
			}
			curr = curr.next.Load()
		}

		if last == nil {
			var zero T
			return zero, false
		}

		if last.deleted.CompareAndSwap(false, true) {
			l.size.Add(-1)
			return *last.value.Load(), true
		}
		// Retry if CAS failed
	}
}

// RemoveFunc removes all elements satisfying predicate.
func (l *lockFreeList[T]) RemoveFunc(predicate func(element T) bool) int {
	removed := 0
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() && predicate(*curr.value.Load()) {
			if curr.deleted.CompareAndSwap(false, true) {
				l.size.Add(-1)
				removed++
			}
		}
		curr = curr.next.Load()
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate.
func (l *lockFreeList[T]) RetainFunc(predicate func(element T) bool) int {
	removed := 0
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() && !predicate(*curr.value.Load()) {
			if curr.deleted.CompareAndSwap(false, true) {
				l.size.Add(-1)
				removed++
			}
		}
		curr = curr.next.Load()
	}
	return removed
}

// IndexOf returns the index of the first occurrence.
func (l *lockFreeList[T]) IndexOf(element T, eq Equaler[T]) int {
	curr := l.head.next.Load()
	i := 0

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if eq(*curr.value.Load(), element) {
				return i
			}
			i++
		}
		curr = curr.next.Load()
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence.
func (l *lockFreeList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	curr := l.head.next.Load()
	lastIdx := -1
	i := 0

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if eq(*curr.value.Load(), element) {
				lastIdx = i
			}
			i++
		}
		curr = curr.next.Load()
	}
	return lastIdx
}

// Contains reports whether element exists.
func (l *lockFreeList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate.
func (l *lockFreeList[T]) Find(predicate func(element T) bool) (T, bool) {
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if v := *curr.value.Load(); predicate(v) {
				return v, true
			}
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *lockFreeList[T]) FindIndex(predicate func(element T) bool) int {
	curr := l.head.next.Load()
	i := 0

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() {
			if predicate(*curr.value.Load()) {
				return i
			}
			i++
		}
		curr = curr.next.Load()
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
	curr := l.head.next.Load()

	for curr != nil && curr != l.tail {
		if !curr.deleted.Load() && !predicate(*curr.value.Load()) {
			return false
		}
		curr = curr.next.Load()
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
