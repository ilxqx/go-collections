package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"iter"
	"slices"
	"sync"
	"sync/atomic"
)

// cowList is a Copy-on-Write implementation of a concurrent list.
// - Read operations are lock-free and extremely fast
// - Write operations copy the entire underlying slice
// - Best suited for read-heavy workloads with infrequent writes
// - Iteration is always safe and sees a consistent snapshot
//
// Atomicity:
//   - ATOMIC: Get, First, Last, Size, Contains, IndexOf (all reads)
//   - NON-ATOMIC: Add, Set, Remove, Insert (copy entire slice)
//   - Iteration sees a consistent snapshot but may be stale
type cowList[T any] struct {
	data atomic.Pointer[[]T]
	mu   sync.Mutex // Only for write serialization
}

// NewCOWList creates an empty Copy-on-Write list.
func NewCOWList[T any]() List[T] {
	l := &cowList[T]{}
	empty := make([]T, 0)
	l.data.Store(&empty)
	return l
}

// NewCOWListFrom creates a COW list from elements.
func NewCOWListFrom[T any](elements ...T) List[T] {
	l := &cowList[T]{}
	cp := make([]T, len(elements))
	copy(cp, elements)
	l.data.Store(&cp)
	return l
}

// snapshot returns the current data slice (read-only).
func (l *cowList[T]) snapshot() []T {
	return *l.data.Load()
}

// Size returns the number of elements.
func (l *cowList[T]) Size() int { return len(l.snapshot()) }

// IsEmpty reports whether the list is empty.
func (l *cowList[T]) IsEmpty() bool    { return len(l.snapshot()) == 0 }
func (l *cowList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements.
func (l *cowList[T]) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	empty := make([]T, 0)
	l.data.Store(&empty)
}

// ToSlice returns a snapshot copy of the elements.
func (l *cowList[T]) ToSlice() []T {
	return slices.Clone(l.snapshot())
}

// String returns a concise representation.
func (l *cowList[T]) String() string {
	return formatCollection("cowList", l.Seq())
}

// Seq returns a sequence of elements (snapshot).
func (l *cowList[T]) Seq() iter.Seq[T] {
	snap := l.snapshot()
	return slices.Values(snap)
}

// ForEach applies action to each element.
func (l *cowList[T]) ForEach(action func(element T) bool) {
	for _, v := range l.snapshot() {
		if !action(v) {
			return
		}
	}
}

// Get returns the element at index.
func (l *cowList[T]) Get(index int) (T, bool) {
	snap := l.snapshot()
	if index < 0 || index >= len(snap) {
		var zero T
		return zero, false
	}
	return snap[index], true
}

// Set replaces the element at index (copies entire slice).
func (l *cowList[T]) Set(index int, element T) (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	if index < 0 || index >= len(snap) {
		var zero T
		return zero, false
	}

	old := snap[index]
	newData := slices.Clone(snap)
	newData[index] = element
	l.data.Store(&newData)
	return old, true
}

// First returns the first element.
func (l *cowList[T]) First() (T, bool) {
	snap := l.snapshot()
	if len(snap) == 0 {
		var zero T
		return zero, false
	}
	return snap[0], true
}

// Last returns the last element.
func (l *cowList[T]) Last() (T, bool) {
	snap := l.snapshot()
	if len(snap) == 0 {
		var zero T
		return zero, false
	}
	return snap[len(snap)-1], true
}

// Add appends the element (copies entire slice).
func (l *cowList[T]) Add(element T) {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	newData := make([]T, len(snap)+1)
	copy(newData, snap)
	newData[len(snap)] = element
	l.data.Store(&newData)
}

// AddAll appends all elements.
func (l *cowList[T]) AddAll(elements ...T) {
	if len(elements) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	newData := make([]T, len(snap)+len(elements))
	copy(newData, snap)
	copy(newData[len(snap):], elements)
	l.data.Store(&newData)
}

// AddSeq appends all elements from the sequence.
func (l *cowList[T]) AddSeq(seq iter.Seq[T]) {
	var buf []T
	for v := range seq {
		buf = append(buf, v)
	}
	l.AddAll(buf...)
}

// Insert inserts the element at index.
func (l *cowList[T]) Insert(index int, element T) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	if index < 0 || index > len(snap) {
		return false
	}

	newData := make([]T, len(snap)+1)
	copy(newData[:index], snap[:index])
	newData[index] = element
	copy(newData[index+1:], snap[index:])
	l.data.Store(&newData)
	return true
}

// InsertAll inserts all elements at index.
func (l *cowList[T]) InsertAll(index int, elements ...T) bool {
	if len(elements) == 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	if index < 0 || index > len(snap) {
		return false
	}

	newData := make([]T, len(snap)+len(elements))
	copy(newData[:index], snap[:index])
	copy(newData[index:index+len(elements)], elements)
	copy(newData[index+len(elements):], snap[index:])
	l.data.Store(&newData)
	return true
}

// RemoveAt removes the element at index.
func (l *cowList[T]) RemoveAt(index int) (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	if index < 0 || index >= len(snap) {
		var zero T
		return zero, false
	}

	removed := snap[index]
	newData := make([]T, len(snap)-1)
	copy(newData[:index], snap[:index])
	copy(newData[index:], snap[index+1:])
	l.data.Store(&newData)
	return removed, true
}

// Remove removes the first occurrence of element.
func (l *cowList[T]) Remove(element T, eq Equaler[T]) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	for i, v := range snap {
		if eq(v, element) {
			newData := make([]T, len(snap)-1)
			copy(newData[:i], snap[:i])
			copy(newData[i:], snap[i+1:])
			l.data.Store(&newData)
			return true
		}
	}
	return false
}

// RemoveFirst removes and returns the first element.
func (l *cowList[T]) RemoveFirst() (T, bool) {
	return l.RemoveAt(0)
}

// RemoveLast removes and returns the last element.
func (l *cowList[T]) RemoveLast() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	if len(snap) == 0 {
		var zero T
		return zero, false
	}

	removed := snap[len(snap)-1]
	newData := slices.Clone(snap[:len(snap)-1])
	l.data.Store(&newData)
	return removed, true
}

// RemoveFunc removes all elements satisfying predicate.
func (l *cowList[T]) RemoveFunc(predicate func(element T) bool) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	newData := make([]T, 0, len(snap))
	removed := 0
	for _, v := range snap {
		if predicate(v) {
			removed++
		} else {
			newData = append(newData, v)
		}
	}
	if removed > 0 {
		newData = slices.Clip(newData)
		l.data.Store(&newData)
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate.
func (l *cowList[T]) RetainFunc(predicate func(element T) bool) int {
	return l.RemoveFunc(func(element T) bool { return !predicate(element) })
}

// IndexOf returns the index of the first occurrence.
func (l *cowList[T]) IndexOf(element T, eq Equaler[T]) int {
	for i, v := range l.snapshot() {
		if eq(v, element) {
			return i
		}
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence.
func (l *cowList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	snap := l.snapshot()
	for i, v := range slices.Backward(snap) {
		if eq(v, element) {
			return i
		}
	}
	return -1
}

// Contains reports whether element exists.
func (l *cowList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate.
func (l *cowList[T]) Find(predicate func(element T) bool) (T, bool) {
	for _, v := range l.snapshot() {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *cowList[T]) FindIndex(predicate func(element T) bool) int {
	for i, v := range l.snapshot() {
		if predicate(v) {
			return i
		}
	}
	return -1
}

// SubList returns a new list containing elements in [from, to).
func (l *cowList[T]) SubList(from, to int) List[T] {
	snap := l.snapshot()
	if from < 0 || to > len(snap) || from > to {
		return NewCOWList[T]()
	}
	return NewCOWListFrom(snap[from:to]...)
}

// Reversed returns a sequence iterating in reverse order.
func (l *cowList[T]) Reversed() iter.Seq[T] {
	snap := l.snapshot()
	return func(yield func(T) bool) {
		for _, v := range slices.Backward(snap) {
			if !yield(v) {
				return
			}
		}
	}
}

// Clone returns a shallow copy.
func (l *cowList[T]) Clone() List[T] {
	return NewCOWListFrom(l.snapshot()...)
}

// Filter returns a new list of elements satisfying predicate.
func (l *cowList[T]) Filter(predicate func(element T) bool) List[T] {
	snap := l.snapshot()
	result := make([]T, 0, len(snap))
	for _, v := range snap {
		if predicate(v) {
			result = append(result, v)
		}
	}
	out := &cowList[T]{}
	out.data.Store(&result)
	return out
}

// Sort sorts elements in place.
func (l *cowList[T]) Sort(cmp Comparator[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()

	snap := l.snapshot()
	newData := slices.Clone(snap)
	slices.SortFunc(newData, cmp)
	l.data.Store(&newData)
}

// Any returns true if at least one element satisfies predicate.
func (l *cowList[T]) Any(predicate func(element T) bool) bool {
	return slices.ContainsFunc(l.snapshot(), predicate)
}

// Every returns true if all elements satisfy predicate.
func (l *cowList[T]) Every(predicate func(element T) bool) bool {
	for _, v := range l.snapshot() {
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
// Serializes a snapshot of the list as a JSON array.
func (l *cowList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.snapshot())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *cowList[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	// Replace the atomic pointer data
	l.data.Store(&slice)
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (l *cowList[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, l.snapshot())
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
func (l *cowList[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	l.data.Store(&slice)
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the list.
func (l *cowList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.snapshot()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (l *cowList[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	// Replace the atomic pointer data
	l.data.Store(&slice)
	return nil
}

// Compile-time conformance.
var _ List[int] = (*cowList[int])(nil)
