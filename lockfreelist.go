package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
)

// lockFreeList is a lock-free concurrent linked list implementation.
// Uses Compare-and-Swap (CAS) operations for thread-safe modifications.
// Based on Harris's lock-free linked list algorithm with logical deletion.
//
// Characteristics:
//   - Lock-free: Progress guaranteed even if some threads are delayed
//   - High throughput under contention
//   - Suitable for high-concurrency scenarios
//
// Atomicity:
//   - ATOMIC: Contains, Add (at head), Size (approximate)
//   - BEST-EFFORT: Remove, Insert (may retry under contention)
//   - NON-ATOMIC: Bulk operations, iteration (snapshot semantics)
//
// ABA Risk: PhysicalDelete uses sync.Pool for node recycling with raw pointer CAS.
// This may have ABA risk in high-concurrency scenarios where nodes are rapidly
// recycled and reused. For strict correctness requirements, consider:
//   - Disabling node pooling (trade GC pressure for safety), or
//   - Using version-tagged pointers (requires unsafe), or
//   - Keeping current design for approximate/near-realtime use cases.
//
// Note: This implementation provides the full List[T] interface.
// Size is approximate due to concurrent modifications; random access Get(index) is O(n).
// Some operations (iteration, bulk) use snapshot semantics for consistency.
type lockFreeList[T any] struct {
	head     atomic.Pointer[lfNode[T]]
	tail     atomic.Pointer[lfNode[T]]
	size     atomic.Int64
	eq       Equaler[T]
	nodePool sync.Pool
}

type lfNode[T any] struct {
	value   T
	next    atomic.Pointer[lfNode[T]]
	deleted atomic.Bool // Logical deletion marker
}

// NewLockFreeList creates a new lock-free list.
// The equaler function is used for element comparison.
func NewLockFreeList[T any](eq Equaler[T]) List[T] {
	l := &lockFreeList[T]{
		eq: eq,
		nodePool: sync.Pool{
			New: func() any { return &lfNode[T]{} },
		},
	}
	// Create sentinel nodes
	head := &lfNode[T]{}
	tail := &lfNode[T]{}
	head.next.Store(tail)
	l.head.Store(head)
	l.tail.Store(tail)
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

// newNode creates or reuses a node from the pool.
func (l *lockFreeList[T]) newNode(value T) *lfNode[T] {
	node := l.nodePool.Get().(*lfNode[T])
	node.value = value
	node.next.Store(nil)
	node.deleted.Store(false)
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

// Clear removes all elements (not truly lock-free, uses snapshot).
func (l *lockFreeList[T]) Clear() {
	// Reset to sentinel nodes
	head := &lfNode[T]{}
	tail := &lfNode[T]{}
	head.next.Store(tail)
	l.head.Store(head)
	l.tail.Store(tail)
	l.size.Store(0)
}

// ToSlice returns a snapshot of all elements.
func (l *lockFreeList[T]) ToSlice() []T {
	size := l.Size()
	if size == 0 {
		return nil
	}
	result := make([]T, 0, size)
	head := l.head.Load()
	tail := l.tail.Load()

	curr := head.next.Load()
	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			result = append(result, curr.value)
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

	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()
	i := 0

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if i == index {
				return curr.value, true
			}
			i++
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// Set replaces the element at index (O(n) operation).
func (l *lockFreeList[T]) Set(index int, element T) (T, bool) {
	if index < 0 {
		var zero T
		return zero, false
	}

	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()
	i := 0

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if i == index {
				old := curr.value
				curr.value = element
				return old, true
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			return curr.value, true
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// Last returns the last element (O(n) operation).
func (l *lockFreeList[T]) Last() (T, bool) {
	var last *lfNode[T]
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			last = curr
		}
		curr = curr.next.Load()
	}

	if last != nil {
		return last.value, true
	}
	var zero T
	return zero, false
}

// Add appends the element at the end.
func (l *lockFreeList[T]) Add(element T) {
	newNode := l.newNode(element)
	tail := l.tail.Load()

	for {
		// Find the actual last node (before tail)
		head := l.head.Load()
		pred := head
		curr := head.next.Load()

		for curr != nil && curr != tail {
			pred = curr
			curr = curr.next.Load()
		}

		// Try to insert before tail
		newNode.next.Store(tail)
		if pred.next.CompareAndSwap(tail, newNode) {
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

	newNode := l.newNode(element)

	for {
		head := l.head.Load()
		tail := l.tail.Load()
		pred := head
		curr := head.next.Load()
		i := 0

		// Find the node at position (index-1) to insert after it
		for curr != nil && curr != tail {
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
		if i != index-1 && curr == tail && i < index-1 {
			return false // Index out of bounds
		}

		next := pred.next.Load()
		newNode.next.Store(next)
		if pred.next.CompareAndSwap(next, newNode) {
			l.size.Add(1)
			return true
		}
		// CAS failed, retry
	}
}

// insertAtHead inserts at the beginning.
func (l *lockFreeList[T]) insertAtHead(element T) bool {
	newNode := l.newNode(element)

	for {
		head := l.head.Load()
		first := head.next.Load()
		newNode.next.Store(first)
		if head.next.CompareAndSwap(first, newNode) {
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
		head := l.head.Load()
		tail := l.tail.Load()
		curr := head.next.Load()
		i := 0

		for curr != nil && curr != tail {
			if !curr.deleted.Load() {
				if i == index {
					// Logically delete
					if curr.deleted.CompareAndSwap(false, true) {
						l.size.Add(-1)
						return curr.value, true
					}
					// Someone else deleted it, retry
					break
				}
				i++
			}
			curr = curr.next.Load()
		}

		if curr == nil || curr == tail {
			var zero T
			return zero, false
		}
	}
}

// Remove removes the first occurrence of element.
func (l *lockFreeList[T]) Remove(element T, eq Equaler[T]) bool {
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() && eq(curr.value, element) {
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if curr.deleted.CompareAndSwap(false, true) {
				l.size.Add(-1)
				return curr.value, true
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
		head := l.head.Load()
		tail := l.tail.Load()
		curr := head.next.Load()

		for curr != nil && curr != tail {
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
			return last.value, true
		}
		// Retry if CAS failed
	}
}

// RemoveFunc removes all elements satisfying predicate.
func (l *lockFreeList[T]) RemoveFunc(predicate func(element T) bool) int {
	removed := 0
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() && predicate(curr.value) {
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() && !predicate(curr.value) {
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()
	i := 0

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if eq(curr.value, element) {
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()
	lastIdx := -1
	i := 0

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if eq(curr.value, element) {
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() && predicate(curr.value) {
			return curr.value, true
		}
		curr = curr.next.Load()
	}

	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *lockFreeList[T]) FindIndex(predicate func(element T) bool) int {
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()
	i := 0

	for curr != nil && curr != tail {
		if !curr.deleted.Load() {
			if predicate(curr.value) {
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

// Sort sorts elements (creates a new internal structure).
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
	head := l.head.Load()
	tail := l.tail.Load()
	curr := head.next.Load()

	for curr != nil && curr != tail {
		if !curr.deleted.Load() && !predicate(curr.value) {
			return false
		}
		curr = curr.next.Load()
	}
	return true
}

// PhysicalDelete removes logically deleted nodes (garbage collection).
// This should be called periodically to reclaim memory.
func (l *lockFreeList[T]) PhysicalDelete() {
	head := l.head.Load()
	tail := l.tail.Load()
	pred := head
	curr := head.next.Load()

	for curr != nil && curr != tail {
		next := curr.next.Load()
		if curr.deleted.Load() {
			// Try to unlink
			pred.next.CompareAndSwap(curr, next)
			// Return node to pool
			var zero T
			curr.value = zero
			curr.next.Store(nil)
			l.nodePool.Put(curr)
		} else {
			pred = curr
		}
		curr = next
	}
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

// Compile-time conformance
var _ List[int] = (*lockFreeList[int])(nil)
