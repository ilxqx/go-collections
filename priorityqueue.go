package collections

import (
	"bytes"
	"container/heap"
	"encoding/gob"
	"encoding/json"
	"errors"
	"iter"
	"slices"
)

// priorityQueue is a heap-based implementation of PriorityQueue[T].
// - O(log n) push and pop
// - O(1) peek
// - By default, smallest element has highest priority (min-heap)
// - Use a reverse comparator for max-heap behavior.
type priorityQueue[T any] struct {
	data []T
	cmp  Comparator[T]
}

// heapWrapper wraps priorityQueue to implement container/heap.Interface.
type heapWrapper[T any] struct {
	pq *priorityQueue[T]
}

func (h *heapWrapper[T]) Len() int { return len(h.pq.data) }

func (h *heapWrapper[T]) Less(i, j int) bool {
	return h.pq.cmp(h.pq.data[i], h.pq.data[j]) < 0
}

func (h *heapWrapper[T]) Swap(i, j int) {
	h.pq.data[i], h.pq.data[j] = h.pq.data[j], h.pq.data[i]
}

func (h *heapWrapper[T]) Push(x any) {
	h.pq.data = append(h.pq.data, x.(T))
}

func (h *heapWrapper[T]) Pop() any {
	n := len(h.pq.data)
	x := h.pq.data[n-1]
	var zero T
	h.pq.data[n-1] = zero
	h.pq.data = h.pq.data[:n-1]
	return x
}

// NewPriorityQueue creates an empty priority queue with a custom comparator.
// The comparator determines priority: elements with smaller comparison values
// have higher priority (min-heap). Use a reverse comparator for max-heap.
func NewPriorityQueue[T any](cmp Comparator[T]) PriorityQueue[T] {
	if cmp == nil {
		panic("NewPriorityQueue: comparator must not be nil")
	}
	return newPriorityQueue(cmp)
}

// newPriorityQueue creates the concrete queue for internal callers.
func newPriorityQueue[T any](cmp Comparator[T]) *priorityQueue[T] {
	return &priorityQueue[T]{
		data: make([]T, 0),
		cmp:  cmp,
	}
}

// NewPriorityQueueOrdered creates a min-heap priority queue for Ordered types.
// Smallest element has highest priority.
func NewPriorityQueueOrdered[T Ordered]() PriorityQueue[T] {
	return NewPriorityQueue(CompareFunc[T]())
}

// NewPriorityQueueWithCapacity creates a priority queue with capacity hint.
func NewPriorityQueueWithCapacity[T any](cmp Comparator[T], capacity int) PriorityQueue[T] {
	if cmp == nil {
		panic("NewPriorityQueueWithCapacity: comparator must not be nil")
	}
	if capacity < 0 {
		capacity = 0
	}
	return &priorityQueue[T]{
		data: make([]T, 0, capacity),
		cmp:  cmp,
	}
}

// NewPriorityQueueFrom creates a priority queue from elements.
func NewPriorityQueueFrom[T any](cmp Comparator[T], elements ...T) PriorityQueue[T] {
	if cmp == nil {
		panic("NewPriorityQueueFrom: comparator must not be nil")
	}
	pq := &priorityQueue[T]{
		data: make([]T, len(elements)),
		cmp:  cmp,
	}
	copy(pq.data, elements)
	heap.Init(&heapWrapper[T]{pq: pq})
	return pq
}

// NewMaxPriorityQueue creates a max-heap priority queue for Ordered types.
// Largest element has highest priority.
func NewMaxPriorityQueue[T Ordered]() PriorityQueue[T] {
	return NewPriorityQueue(func(a, b T) int {
		return CompareFunc[T]()(b, a) // Reverse comparison
	})
}

// Size returns the number of elements.
func (pq *priorityQueue[T]) Size() int { return len(pq.data) }

// IsEmpty reports whether the queue is empty.
func (pq *priorityQueue[T]) IsEmpty() bool    { return len(pq.data) == 0 }
func (pq *priorityQueue[T]) IsNotEmpty() bool { return !pq.IsEmpty() }

// Clear removes all elements (retains capacity).
func (pq *priorityQueue[T]) Clear() {
	clear(pq.data)
	pq.data = pq.data[:0]
}

// String returns a concise representation.
func (pq *priorityQueue[T]) String() string {
	return formatCollection("priorityQueue", pq.Seq())
}

// Push adds an element to the queue. O(log n).
func (pq *priorityQueue[T]) Push(element T) {
	heap.Push(&heapWrapper[T]{pq: pq}, element)
}

// PushAll adds all elements to the queue.
func (pq *priorityQueue[T]) PushAll(elements ...T) {
	for _, e := range elements {
		pq.Push(e)
	}
}

// Pop removes and returns the highest-priority element, or (zero, false) if empty. O(log n).
func (pq *priorityQueue[T]) Pop() (T, bool) {
	if len(pq.data) == 0 {
		var zero T
		return zero, false
	}
	v := heap.Pop(&heapWrapper[T]{pq: pq}).(T)
	return v, true
}

// Peek returns the highest-priority element without removing it, or (zero, false) if empty. O(1).
func (pq *priorityQueue[T]) Peek() (T, bool) {
	if len(pq.data) == 0 {
		var zero T
		return zero, false
	}
	return pq.data[0], true
}

// ToSlice returns elements in heap order (not sorted).
func (pq *priorityQueue[T]) ToSlice() []T {
	return slices.Clone(pq.data)
}

// ToSortedSlice returns elements in priority order (sorted).
func (pq *priorityQueue[T]) ToSortedSlice() []T {
	out := slices.Clone(pq.data)
	slices.SortFunc(out, pq.cmp)
	return out
}

// Seq returns a sequence in heap order (not sorted).
func (pq *priorityQueue[T]) Seq() iter.Seq[T] {
	return slices.Values(pq.data)
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes elements in heap order (not sorted).
//
// NOTE: The comparator is NOT serialized. Decode into a queue constructed
// with NewPriorityQueue, or use UnmarshalPriorityQueueJSON /
// UnmarshalPriorityQueueOrderedJSON.
func (pq *priorityQueue[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(pq.data)
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes into the receiver using its existing comparator, so construct
// the queue with NewPriorityQueue (or a helper) before decoding.
func (pq *priorityQueue[T]) UnmarshalJSON(data []byte) error {
	if pq.cmp == nil {
		return errors.New("unmarshal priorityqueue: no comparator; construct the queue with NewPriorityQueue before decoding, or use UnmarshalPriorityQueueJSON/UnmarshalPriorityQueueOrderedJSON")
	}
	var elements []T
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}
	pq.data = elements
	// Re-establish the heap invariant: the source need not be heap-ordered.
	heap.Init(&heapWrapper[T]{pq: pq})
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes elements in heap order (not sorted).
//
// NOTE: The comparator is NOT serialized. Decode into a queue constructed
// with NewPriorityQueue, or use UnmarshalPriorityQueueGob /
// UnmarshalPriorityQueueOrderedGob.
func (pq *priorityQueue[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(pq.data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes into the receiver using its existing comparator, so construct
// the queue with NewPriorityQueue (or a helper) before decoding.
func (pq *priorityQueue[T]) GobDecode(data []byte) error {
	if pq.cmp == nil {
		return errors.New("unmarshal priorityqueue: no comparator; construct the queue with NewPriorityQueue before decoding, or use UnmarshalPriorityQueueGob/UnmarshalPriorityQueueOrderedGob")
	}
	var elements []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&elements); err != nil {
		return err
	}
	pq.data = elements
	// Re-establish the heap invariant: the source need not be heap-ordered.
	heap.Init(&heapWrapper[T]{pq: pq})
	return nil
}

// Compile-time conformance.
var (
	_ PriorityQueue[int] = (*priorityQueue[int])(nil)
)
