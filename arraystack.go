package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"
)

// arrayStack is a slice-backed LIFO stack.
// Top is at the end of the slice for O(1) push/pop.
type arrayStack[T any] struct {
	data []T
}

// NewArrayStack creates an empty Stack.
func NewArrayStack[T any]() Stack[T] {
	return &arrayStack[T]{data: make([]T, 0)}
}

// NewArrayStackWithCapacity creates a stack with capacity hint.
func NewArrayStackWithCapacity[T any](capacity int) Stack[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &arrayStack[T]{data: make([]T, 0, capacity)}
}

// NewArrayStackFrom creates a stack initialized with elements (bottom to top).
func NewArrayStackFrom[T any](elements ...T) Stack[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return &arrayStack[T]{data: cp}
}

// Size returns the number of elements.
func (s *arrayStack[T]) Size() int { return len(s.data) }

// IsEmpty reports whether empty.
func (s *arrayStack[T]) IsEmpty() bool    { return len(s.data) == 0 }
func (s *arrayStack[T]) IsNotEmpty() bool { return !s.IsEmpty() }

// Clear removes all elements (retains capacity).
func (s *arrayStack[T]) Clear() {
	clear(s.data)
	s.data = s.data[:0]
}

// String returns a concise representation (bottom..top).
func (s *arrayStack[T]) String() string {
	return formatCollection("arrayStack", s.Seq())
}

// Push adds an element to the top.
func (s *arrayStack[T]) Push(element T) { s.data = append(s.data, element) }

// PushAll adds all elements to the top (last becomes top).
func (s *arrayStack[T]) PushAll(elements ...T) { s.data = append(s.data, elements...) }

// Pop removes and returns the top element, or (zero, false) if empty.
func (s *arrayStack[T]) Pop() (T, bool) {
	n := len(s.data)
	if n == 0 {
		var zero T
		return zero, false
	}
	v := s.data[n-1]
	// Clear last slot before shrinking to help GC drop references promptly.
	var zero T
	s.data[n-1] = zero
	s.data = s.data[:n-1]
	return v, true
}

// Peek returns the top element without removing it, or (zero, false) if empty.
func (s *arrayStack[T]) Peek() (T, bool) {
	n := len(s.data)
	if n == 0 {
		var zero T
		return zero, false
	}
	return s.data[n-1], true
}

// ToSlice returns elements from bottom to top (snapshot).
func (s *arrayStack[T]) ToSlice() []T {
	return slices.Clone(s.data)
}

// Seq returns a sequence from top to bottom (LIFO order).
func (s *arrayStack[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range slices.Backward(s.data) {
			if !yield(v) {
				return
			}
		}
	}
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes from bottom to top as a JSON array.
func (s *arrayStack[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.data)
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (s *arrayStack[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.data)
}

// GobEncode implements gob.GobEncoder.
func (s *arrayStack[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s.data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (s *arrayStack[T]) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(&s.data)
}

// Compile-time conformance
var (
	_ Stack[int] = (*arrayStack[int])(nil)
)
