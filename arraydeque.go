package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"iter"
)

// arrayDeque is a double-ended queue implemented as a circular buffer.
// Push/Pop at both ends are O(1) amortized; grows by doubling when full.
type arrayDeque[T any] struct {
	buf  []T
	head int // index of first element
	tail int // index one past the last element
	size int
}

// NewArrayDeque creates an empty deque.
func NewArrayDeque[T any]() Deque[T] {
	return &arrayDeque[T]{buf: make([]T, 0)}
}

// NewArrayDequeWithCapacity creates a deque with capacity hint.
// If capacity <= 0, an empty buffer is created and will grow on first insert.
func NewArrayDequeWithCapacity[T any](capacity int) Deque[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &arrayDeque[T]{buf: make([]T, capacity)}
}

// NewArrayDequeFrom creates a deque from elements (front to back in given order).
func NewArrayDequeFrom[T any](elements ...T) Deque[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	// The buffer is exactly full, so the circular one-past-last index wraps to 0.
	return &arrayDeque[T]{buf: cp, head: 0, tail: 0, size: len(cp)}
}

// Size returns the number of elements.
func (d *arrayDeque[T]) Size() int { return d.size }

// IsEmpty reports whether the deque is empty.
func (d *arrayDeque[T]) IsEmpty() bool    { return d.size == 0 }
func (d *arrayDeque[T]) IsNotEmpty() bool { return !d.IsEmpty() }

// Clear removes all elements (retains capacity).
func (d *arrayDeque[T]) Clear() {
	clear(d.buf)
	d.head, d.tail, d.size = 0, 0, 0
}

// String returns a concise representation (front..back).
func (d *arrayDeque[T]) String() string {
	return formatCollection("arrayDeque", d.Seq())
}

// PushFront adds an element to the front.
func (d *arrayDeque[T]) PushFront(element T) {
	d.ensureCapacityForOne()
	d.head = d.mod(d.head-1, len(d.buf))
	d.buf[d.head] = element
	d.size++
	if d.size == 1 {
		d.tail = d.mod(d.head+1, len(d.buf))
	}
}

// PopFront removes and returns the front element.
func (d *arrayDeque[T]) PopFront() (T, bool) {
	if d.size == 0 {
		var zero T
		return zero, false
	}
	v := d.buf[d.head]
	// Clear slot to drop references promptly.
	var zero T
	d.buf[d.head] = zero
	d.head = d.mod(d.head+1, len(d.buf))
	d.size--
	return v, true
}

// PeekFront returns the front element without removing it.
func (d *arrayDeque[T]) PeekFront() (T, bool) {
	if d.size == 0 {
		var zero T
		return zero, false
	}
	return d.buf[d.head], true
}

// PushBack adds an element to the back.
func (d *arrayDeque[T]) PushBack(element T) {
	d.ensureCapacityForOne()
	d.buf[d.tail] = element
	d.tail = d.mod(d.tail+1, len(d.buf))
	d.size++
}

// PopBack removes and returns the back element.
func (d *arrayDeque[T]) PopBack() (T, bool) {
	if d.size == 0 {
		var zero T
		return zero, false
	}
	d.tail = d.mod(d.tail-1, len(d.buf))
	v := d.buf[d.tail]
	// Clear slot to drop references promptly.
	var zero T
	d.buf[d.tail] = zero
	d.size--
	return v, true
}

// PeekBack returns the back element without removing it.
func (d *arrayDeque[T]) PeekBack() (T, bool) {
	if d.size == 0 {
		var zero T
		return zero, false
	}
	idx := d.mod(d.tail-1, len(d.buf))
	return d.buf[idx], true
}

// ToSlice returns elements from front to back.
func (d *arrayDeque[T]) ToSlice() []T {
	out := make([]T, d.size)
	for i := range d.size {
		out[i] = d.at(i)
	}
	return out
}

// Seq returns a sequence from front to back.
func (d *arrayDeque[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := range d.size {
			if !yield(d.at(i)) {
				return
			}
		}
	}
}

// Reversed returns a sequence from back to front.
func (d *arrayDeque[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := d.size - 1; i >= 0; i-- {
			if !yield(d.at(i)) {
				return
			}
		}
	}
}

// ensureCapacityForOne ensures there is room for one more element.
func (d *arrayDeque[T]) ensureCapacityForOne() {
	if d.size < len(d.buf) {
		return
	}
	newCap := 1
	if len(d.buf) > 0 {
		newCap = len(d.buf) * 2
	}
	newBuf := make([]T, newCap)
	for i := range d.size {
		newBuf[i] = d.at(i)
	}
	d.buf = newBuf
	d.head = 0
	d.tail = d.size
}

// at returns the element at logical index i (0..size-1).
func (d *arrayDeque[T]) at(i int) T {
	return d.buf[d.mod(d.head+i, len(d.buf))]
}

// mod wraps n within [0,m).
func (*arrayDeque[T]) mod(n, m int) int {
	if m == 0 {
		return 0
	}
	n %= m
	if n < 0 {
		n += m
	}
	return n
}

// Note on growth strategy:
// We double the buffer capacity when full (geometric growth by 2x). This keeps amortized
// PushFront/PushBack at O(1) and reduces the number of resizes for long-lived deques.
// The copy during growth preserves logical order (front..back) in the new buffer.

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes from front to back as a JSON array.
func (d *arrayDeque[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (d *arrayDeque[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	d.replaceFromSlice(slice)
	return nil
}

// replaceFromSlice rebuilds the deque from slice; the buffer is exactly
// full, so tail wraps to 0.
func (d *arrayDeque[T]) replaceFromSlice(slice []T) {
	d.buf = make([]T, len(slice))
	copy(d.buf, slice)
	d.head = 0
	d.tail = 0
	d.size = len(slice)
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (d *arrayDeque[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, d.ToSlice())
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
func (d *arrayDeque[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	d.replaceFromSlice(slice)
	return nil
}

// GobEncode implements gob.GobEncoder.
func (d *arrayDeque[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(d.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (d *arrayDeque[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	d.replaceFromSlice(slice)
	return nil
}

// Compile-time conformance.
var (
	_ Deque[int] = (*arrayDeque[int])(nil)
)
