package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
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

// siftUp moves the element at index i toward the root until the heap
// invariant holds. O(log n).
func (pq *priorityQueue[T]) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if pq.cmp(pq.data[i], pq.data[parent]) >= 0 {
			return
		}
		pq.data[i], pq.data[parent] = pq.data[parent], pq.data[i]
		i = parent
	}
}

// siftDown moves the element at index i toward the leaves until the heap
// invariant holds. O(log n).
func (pq *priorityQueue[T]) siftDown(i int) {
	n := len(pq.data)
	for {
		left := 2*i + 1
		if left >= n {
			return
		}
		smallest := left
		if right := left + 1; right < n && pq.cmp(pq.data[right], pq.data[left]) < 0 {
			smallest = right
		}
		if pq.cmp(pq.data[smallest], pq.data[i]) >= 0 {
			return
		}
		pq.data[i], pq.data[smallest] = pq.data[smallest], pq.data[i]
		i = smallest
	}
}

// heapify re-establishes the heap invariant over the whole backing slice. O(n).
func (pq *priorityQueue[T]) heapify() {
	for i := len(pq.data)/2 - 1; i >= 0; i-- {
		pq.siftDown(i)
	}
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
	pq.heapify()
	return pq
}

// NewMaxPriorityQueue creates a max-heap priority queue for Ordered types.
// Largest element has highest priority.
func NewMaxPriorityQueue[T Ordered]() PriorityQueue[T] {
	c := CompareFunc[T]()
	return NewPriorityQueue(func(a, b T) int {
		return c(b, a) // Reverse comparison
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
	pq.data = append(pq.data, element)
	pq.siftUp(len(pq.data) - 1)
}

// PushAll adds all elements to the queue.
func (pq *priorityQueue[T]) PushAll(elements ...T) {
	if len(elements) == 0 {
		return
	}
	// For batches comparable to the heap size, one O(n+m) heapify beats
	// m separate O(log n) sift-ups.
	if len(elements) >= len(pq.data) {
		pq.data = append(pq.data, elements...)
		pq.heapify()
		return
	}
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
	v := pq.data[0]
	n := len(pq.data) - 1
	pq.data[0] = pq.data[n]
	// Clear the vacated slot to drop references promptly.
	var zero T
	pq.data[n] = zero
	pq.data = pq.data[:n]
	pq.siftDown(0)
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
	pq.heapify()
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (pq *priorityQueue[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, pq.data)
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
// Same comparator contract as UnmarshalJSON.
func (pq *priorityQueue[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if pq.cmp == nil {
		return errors.New("unmarshal priorityqueue: no comparator; construct the queue with NewPriorityQueue before decoding, or use UnmarshalPriorityQueueJSON/UnmarshalPriorityQueueOrderedJSON")
	}
	var elements []T
	if err := jsonv2.UnmarshalDecode(dec, &elements); err != nil {
		return err
	}
	pq.data = elements
	// Re-establish the heap invariant: the source need not be heap-ordered.
	pq.heapify()
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
	pq.heapify()
	return nil
}

// Compile-time conformance.
var (
	_ PriorityQueue[int] = (*priorityQueue[int])(nil)
)
