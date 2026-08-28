package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"
)

// arrayList is a growable array-backed implementation of List[T].
// - Preserves insertion order
// - Supports O(1) append and amortized growth
// - Insert/RemoveAt are O(n) due to shifting.
type arrayList[T any] struct {
	data []T
}

// NewArrayList creates an empty List.
func NewArrayList[T any]() List[T] {
	return &arrayList[T]{data: make([]T, 0)}
}

// NewArrayListWithCapacity creates a List with capacity hint.
func NewArrayListWithCapacity[T any](capacity int) List[T] {
	if capacity < 0 {
		capacity = 0
	}
	return &arrayList[T]{data: make([]T, 0, capacity)}
}

// NewArrayListFrom creates a List containing the provided elements.
func NewArrayListFrom[T any](elements ...T) List[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return &arrayList[T]{data: cp}
}

// Size returns the number of elements.
func (l *arrayList[T]) Size() int { return len(l.data) }

// IsEmpty reports whether the list is empty.
func (l *arrayList[T]) IsEmpty() bool    { return len(l.data) == 0 }
func (l *arrayList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements (capacity is retained).
func (l *arrayList[T]) Clear() {
	clear(l.data)
	l.data = l.data[:0]
}

// ToSlice returns a snapshot copy of the elements in order.
func (l *arrayList[T]) ToSlice() []T {
	return slices.Clone(l.data)
}

// String returns a concise representation.
func (l *arrayList[T]) String() string {
	return formatCollection("arrayList", l.Seq())
}

// Seq returns a sequence of elements in order.
func (l *arrayList[T]) Seq() iter.Seq[T] {
	return slices.Values(l.data)
}

// ForEach applies action to each element; stops early if action returns false.
func (l *arrayList[T]) ForEach(action func(element T) bool) {
	for _, v := range l.data {
		if !action(v) {
			return
		}
	}
}

// Get returns the element at index, or (zero, false) if out of bounds.
func (l *arrayList[T]) Get(index int) (T, bool) {
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	return l.data[index], true
}

// Set replaces the element at index. Returns (oldValue, true) if successful.
func (l *arrayList[T]) Set(index int, element T) (T, bool) {
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	old := l.data[index]
	l.data[index] = element
	return old, true
}

// First returns the first element, or (zero, false) if empty.
func (l *arrayList[T]) First() (T, bool) {
	if len(l.data) == 0 {
		var zero T
		return zero, false
	}
	return l.data[0], true
}

// Last returns the last element, or (zero, false) if empty.
func (l *arrayList[T]) Last() (T, bool) {
	n := len(l.data)
	if n == 0 {
		var zero T
		return zero, false
	}
	return l.data[n-1], true
}

// Add appends the element to the end.
func (l *arrayList[T]) Add(element T) { l.data = append(l.data, element) }

// AddAll appends all elements to the end.
func (l *arrayList[T]) AddAll(elements ...T) {
	l.data = append(l.data, elements...)
}

// AddSeq appends all elements from the sequence.
func (l *arrayList[T]) AddSeq(seq iter.Seq[T]) {
	l.data = slices.AppendSeq(l.data, seq)
}

// Insert inserts the element at index, shifting subsequent elements right.
// Returns false if index is out of bounds.
func (l *arrayList[T]) Insert(index int, element T) bool {
	if index < 0 || index > len(l.data) {
		return false
	}
	// Use standard library helper which handles growth + shift efficiently.
	l.data = slices.Insert(l.data, index, element)
	return true
}

// InsertAll inserts all elements at index, shifting subsequent elements right.
func (l *arrayList[T]) InsertAll(index int, elements ...T) bool {
	if index < 0 || index > len(l.data) {
		return false
	}
	n := len(elements)
	if n == 0 {
		return true
	}
	// Prefer slices.Insert to avoid manual double-copies and to keep code concise.
	l.data = slices.Insert(l.data, index, elements...)
	return true
}

// RemoveAt removes the element at index. Returns (removed, true) if successful.
func (l *arrayList[T]) RemoveAt(index int) (T, bool) {
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	removed := l.data[index]
	copy(l.data[index:], l.data[index+1:])
	// Clear last element to let GC reclaim referenced objects earlier.
	var zero T
	l.data[len(l.data)-1] = zero
	l.data = l.data[:len(l.data)-1]
	return removed, true
}

// Remove removes the first occurrence of element. Returns true if removed.
func (l *arrayList[T]) Remove(element T, eq Equaler[T]) bool {
	i := l.IndexOf(element, eq)
	if i < 0 {
		return false
	}
	_, _ = l.RemoveAt(i)
	return true
}

// RemoveFirst removes and returns the first element.
func (l *arrayList[T]) RemoveFirst() (T, bool) { return l.RemoveAt(0) }

// RemoveLast removes and returns the last element.
func (l *arrayList[T]) RemoveLast() (T, bool) { return l.RemoveAt(len(l.data) - 1) }

// RemoveFunc removes all elements satisfying predicate. Returns count removed.
func (l *arrayList[T]) RemoveFunc(predicate func(element T) bool) int {
	oldLen := len(l.data)
	j := 0
	for _, v := range l.data {
		if predicate(v) {
			continue
		}
		l.data[j] = v
		j++
	}
	// Clear the now-dead tail to avoid holding references.
	clear(l.data[j:oldLen])
	l.data = l.data[:j]
	return oldLen - j
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (l *arrayList[T]) RetainFunc(predicate func(element T) bool) int {
	oldLen := len(l.data)
	j := 0
	for _, v := range l.data {
		if !predicate(v) {
			continue
		}
		l.data[j] = v
		j++
	}
	// Clear the now-dead tail to avoid holding references.
	clear(l.data[j:oldLen])
	l.data = l.data[:j]
	return oldLen - j
}

// IndexOf returns the index of the first occurrence, or -1.
func (l *arrayList[T]) IndexOf(element T, eq Equaler[T]) int {
	for i, v := range l.data {
		if eq(v, element) {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence, or -1.
func (l *arrayList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	for i, v := range slices.Backward(l.data) {
		if eq(v, element) {
			return i
		}
	}
	return -1
}

// Contains reports whether element exists using eq for comparison.
func (l *arrayList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate, or (zero, false).
func (l *arrayList[T]) Find(predicate func(element T) bool) (T, bool) {
	for _, v := range l.data {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate, or -1.
func (l *arrayList[T]) FindIndex(predicate func(element T) bool) int {
	for i, v := range l.data {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// SubList returns a new list containing elements in [from, to).
func (l *arrayList[T]) SubList(from, to int) List[T] {
	n := len(l.data)
	if from < 0 || to > n || from > to {
		return NewArrayList[T]()
	}
	cp := make([]T, to-from)
	copy(cp, l.data[from:to])
	return &arrayList[T]{data: cp}
}

// Reversed returns a sequence iterating in reverse order.
func (l *arrayList[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, v := range slices.Backward(l.data) {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a shallow copy.
func (l *arrayList[T]) Clone() List[T] {
	return &arrayList[T]{data: l.ToSlice()}
}

// Filter returns a new list of elements satisfying predicate.
func (l *arrayList[T]) Filter(predicate func(element T) bool) List[T] {
	out := &arrayList[T]{data: make([]T, 0, len(l.data))}
	for _, v := range l.data {
		if predicate(v) {
			out.data = append(out.data, v)
		}
	}
	return out
}

// Sort sorts elements in place using the comparator.
func (l *arrayList[T]) Sort(cmp Comparator[T]) {
	slices.SortFunc(l.data, func(a, b T) int { return cmp(a, b) })
}

// Any returns true if at least one element satisfies predicate.
func (l *arrayList[T]) Any(predicate func(element T) bool) bool {
	return slices.ContainsFunc(l.data, predicate)
}

// Every returns true if all elements satisfy predicate.
func (l *arrayList[T]) Every(predicate func(element T) bool) bool {
	for _, v := range l.data {
		if !predicate(v) {
			return false
		}
	}
	return true
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes the list as a JSON array in order.
func (l *arrayList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.data)
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *arrayList[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.data)
}

// GobEncode implements gob.GobEncoder.
func (l *arrayList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (l *arrayList[T]) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewReader(data))
	return dec.Decode(&l.data)
}

// Compile-time conformance.
var (
	_ List[int] = (*arrayList[int])(nil)
)
