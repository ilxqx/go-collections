package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"iter"
	"slices"
)

// arrayQueue is a FIFO queue backed by a slice with a head index.
// Dequeues advance the head; occasionally compacts to reclaim space.
type arrayQueue[T any] struct {
	data []T
	head int
}

// NewArrayQueue creates an empty queue.
func NewArrayQueue[T any]() Queue[T] {
	return &arrayQueue[T]{data: make([]T, 0), head: 0}
}

// NewArrayQueueWithCapacity creates a queue with capacity hint.
func NewArrayQueueWithCapacity[T any](capacity int) Queue[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &arrayQueue[T]{data: make([]T, 0, capacity), head: 0}
}

// NewArrayQueueFrom creates a queue initialized with elements (front to back).
func NewArrayQueueFrom[T any](elements ...T) Queue[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return &arrayQueue[T]{data: cp, head: 0}
}

// Size returns the number of elements.
func (q *arrayQueue[T]) Size() int { return len(q.data) - q.head }

// IsEmpty reports whether the queue is empty.
func (q *arrayQueue[T]) IsEmpty() bool    { return q.Size() == 0 }
func (q *arrayQueue[T]) IsNotEmpty() bool { return !q.IsEmpty() }

// Clear removes all elements (retains capacity).
func (q *arrayQueue[T]) Clear() {
	clear(q.data)
	q.data = q.data[:0]
	q.head = 0
}

// String returns a concise representation.
func (q *arrayQueue[T]) String() string {
	return formatCollection("arrayQueue", q.Seq())
}

// Enqueue adds an element to the back of the queue.
func (q *arrayQueue[T]) Enqueue(element T) {
	q.data = append(q.data, element)
}

// EnqueueAll adds all elements to the back.
func (q *arrayQueue[T]) EnqueueAll(elements ...T) {
	if len(elements) == 0 {
		return
	}
	// Pre-grow to reduce reallocations on large batches.
	q.data = slices.Grow(q.data, len(elements))
	q.data = append(q.data, elements...)
}

// Dequeue removes and returns the front element, or (zero, false) if empty.
func (q *arrayQueue[T]) Dequeue() (T, bool) {
	if q.head >= len(q.data) {
		var zero T
		return zero, false
	}
	v := q.data[q.head]
	// Clear out the slot to avoid retaining references until compaction.
	var zero T
	q.data[q.head] = zero
	q.head++
	// Compact if head is large relative to slice to avoid unbounded growth.
	// Threshold at > len/2 keeps amortized O(1) dequeue while limiting memory retention
	// after bursty traffic where head advances significantly.
	if q.head > len(q.data)/2 {
		q.compact()
	}
	return v, true
}

// Peek returns the front element without removing it, or (zero, false) if empty.
func (q *arrayQueue[T]) Peek() (T, bool) {
	if q.head >= len(q.data) {
		var zero T
		return zero, false
	}
	return q.data[q.head], true
}

// ToSlice returns elements from front to back (snapshot).
func (q *arrayQueue[T]) ToSlice() []T {
	return slices.Clone(q.data[q.head:])
}

// Seq returns a sequence from front to back.
func (q *arrayQueue[T]) Seq() iter.Seq[T] {
	return slices.Values(q.data[q.head:])
}

// compact reclaims consumed head space.
// If capacity significantly exceeds live elements after peak usage, it allocates
// a smaller slice to release memory. Otherwise, it shifts in-place to avoid allocation.
func (q *arrayQueue[T]) compact() {
	if q.head == 0 {
		return
	}
	live := len(q.data) - q.head
	capData := cap(q.data)

	// Shrink capacity if live elements are few relative to capacity.
	// This handles "peak then low-water" scenarios where memory would otherwise be retained.
	// Only shrink if capacity is meaningful (> 64) to avoid thrashing on small queues.
	if live < capData/4 && capData > 64 {
		newData := make([]T, live)
		copy(newData, q.data[q.head:])
		q.data = newData
		q.head = 0
		return
	}

	// Shift in-place to avoid allocation for normal compaction.
	copy(q.data[:live], q.data[q.head:])
	// Clear the vacated tail so the backing array does not retain references
	// to elements past the new length.
	clear(q.data[live:])
	q.data = q.data[:live]
	q.head = 0
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes from front to back as a JSON array.
func (q *arrayQueue[T]) MarshalJSON() ([]byte, error) {
	// Encoders do not retain the slice, so no defensive copy is needed.
	return json.Marshal(q.data[q.head:])
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (q *arrayQueue[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	q.data = slice
	q.head = 0
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (q *arrayQueue[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, q.data[q.head:])
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
func (q *arrayQueue[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	q.data = slice
	q.head = 0
	return nil
}

// GobEncode implements gob.GobEncoder.
func (q *arrayQueue[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(q.data[q.head:]); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (q *arrayQueue[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	q.data = slice
	q.head = 0
	return nil
}

// Compile-time conformance.
var (
	_ Queue[int] = (*arrayQueue[int])(nil)
)
