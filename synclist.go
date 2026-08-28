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
)

// syncList is a concurrent list backed by a slice guarded by a single RWMutex.
// It replaces the former SegmentedList, whose per-segment locks all funneled
// appends into the last segment and delivered no real write concurrency — one
// lock is simpler and faster.
//
// Atomicity:
//   - ATOMIC: every single method call runs under the lock
//   - Iteration methods (Seq/ForEach/Reversed/Filter) copy a snapshot first,
//     so user callbacks never run while the lock is held
//   - Every other callback (the Equaler of Remove/IndexOf/Contains, the
//     predicates of Find/RemoveFunc/RetainFunc/Every, the Comparator of
//     Sort) runs while the lock is held — that is what makes those calls
//     atomic. Such a callback must not call back into the same list, or it
//     will deadlock.
type syncList[T any] struct {
	mu   sync.RWMutex
	data []T
}

// NewSyncList creates a new empty mutex-guarded list.
//
// Every method call is atomic under an internal lock. Iteration methods
// (Seq/ForEach/Reversed/Filter) run user callbacks on a snapshot, outside
// the lock; every other callback (the Equaler of Remove/IndexOf/Contains,
// the predicates of Find/RemoveFunc/RetainFunc/Every, the Comparator of
// Sort) runs while the lock is held and must not call back into the same
// list, or it will deadlock.
func NewSyncList[T any]() List[T] {
	return &syncList[T]{}
}

// NewSyncListFrom creates a mutex-guarded list from elements.
// See NewSyncList for the locking contract.
func NewSyncListFrom[T any](elements ...T) List[T] {
	l := &syncList[T]{data: make([]T, len(elements))}
	copy(l.data, elements)
	return l
}

// Size returns the number of elements.
func (l *syncList[T]) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.data)
}

// IsEmpty reports whether the list is empty.
func (l *syncList[T]) IsEmpty() bool {
	return l.Size() == 0
}

// IsNotEmpty reports whether the list contains at least one element.
func (l *syncList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements (retains capacity).
func (l *syncList[T]) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	clear(l.data)
	l.data = l.data[:0]
}

// ToSlice returns a snapshot of all elements.
func (l *syncList[T]) ToSlice() []T {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return slices.Clone(l.data)
}

// String returns a string representation.
func (l *syncList[T]) String() string {
	return formatCollection("syncList", l.Seq())
}

// Seq returns a sequence of elements (snapshot).
func (l *syncList[T]) Seq() iter.Seq[T] {
	snap := l.ToSlice()
	return slices.Values(snap)
}

// ForEach applies action to each element of a snapshot.
func (l *syncList[T]) ForEach(action func(element T) bool) {
	for _, v := range l.ToSlice() {
		if !action(v) {
			return
		}
	}
}

// Get returns the element at index.
func (l *syncList[T]) Get(index int) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	return l.data[index], true
}

// Set replaces the element at index.
func (l *syncList[T]) Set(index int, element T) (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	old := l.data[index]
	l.data[index] = element
	return old, true
}

// First returns the first element.
func (l *syncList[T]) First() (T, bool) {
	return l.Get(0)
}

// Last returns the last element.
func (l *syncList[T]) Last() (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.data) == 0 {
		var zero T
		return zero, false
	}
	return l.data[len(l.data)-1], true
}

// Add appends the element.
func (l *syncList[T]) Add(element T) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = append(l.data, element)
}

// AddAll appends all elements.
func (l *syncList[T]) AddAll(elements ...T) {
	if len(elements) == 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = append(l.data, elements...)
}

// AddSeq appends all elements from the sequence.
func (l *syncList[T]) AddSeq(seq iter.Seq[T]) {
	var buf []T
	for v := range seq {
		buf = append(buf, v)
	}
	l.AddAll(buf...)
}

// Insert inserts the element at index.
func (l *syncList[T]) Insert(index int, element T) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if index < 0 || index > len(l.data) {
		return false
	}
	l.data = slices.Insert(l.data, index, element)
	return true
}

// InsertAll inserts all elements at index.
func (l *syncList[T]) InsertAll(index int, elements ...T) bool {
	if len(elements) == 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if index < 0 || index > len(l.data) {
		return false
	}
	l.data = slices.Insert(l.data, index, elements...)
	return true
}

// RemoveAt removes the element at index.
func (l *syncList[T]) RemoveAt(index int) (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if index < 0 || index >= len(l.data) {
		var zero T
		return zero, false
	}
	removed := l.data[index]
	l.data = slices.Delete(l.data, index, index+1)
	return removed, true
}

// Remove removes the first occurrence of element.
func (l *syncList[T]) Remove(element T, eq Equaler[T]) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, v := range l.data {
		if eq(v, element) {
			l.data = slices.Delete(l.data, i, i+1)
			return true
		}
	}
	return false
}

// RemoveFirst removes and returns the first element.
func (l *syncList[T]) RemoveFirst() (T, bool) {
	return l.RemoveAt(0)
}

// RemoveLast removes and returns the last element.
func (l *syncList[T]) RemoveLast() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.data) == 0 {
		var zero T
		return zero, false
	}
	last := len(l.data) - 1
	removed := l.data[last]
	// Clear the vacated slot to avoid retaining references.
	var zero T
	l.data[last] = zero
	l.data = l.data[:last]
	return removed, true
}

// RemoveFunc removes all elements satisfying predicate.
func (l *syncList[T]) RemoveFunc(predicate func(element T) bool) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	n := 0
	for _, v := range l.data {
		if predicate(v) {
			removed++
		} else {
			l.data[n] = v
			n++
		}
	}
	clear(l.data[n:])
	l.data = l.data[:n]
	return removed
}

// RetainFunc keeps only elements satisfying predicate.
func (l *syncList[T]) RetainFunc(predicate func(element T) bool) int {
	return l.RemoveFunc(func(element T) bool { return !predicate(element) })
}

// IndexOf returns the index of the first occurrence.
func (l *syncList[T]) IndexOf(element T, eq Equaler[T]) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return slices.IndexFunc(l.data, func(v T) bool { return eq(v, element) })
}

// LastIndexOf returns the index of the last occurrence.
func (l *syncList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for i, v := range slices.Backward(l.data) {
		if eq(v, element) {
			return i
		}
	}
	return -1
}

// Contains reports whether element exists.
func (l *syncList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate.
func (l *syncList[T]) Find(predicate func(element T) bool) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, v := range l.data {
		if predicate(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *syncList[T]) FindIndex(predicate func(element T) bool) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return slices.IndexFunc(l.data, predicate)
}

// SubList returns a new list containing elements in [from, to).
func (l *syncList[T]) SubList(from, to int) List[T] {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if from < 0 || to > len(l.data) || from > to {
		return NewSyncList[T]()
	}
	return &syncList[T]{data: slices.Clone(l.data[from:to])}
}

// Reversed returns a sequence iterating in reverse order.
func (l *syncList[T]) Reversed() iter.Seq[T] {
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
func (l *syncList[T]) Clone() List[T] {
	return &syncList[T]{data: l.ToSlice()}
}

// Filter returns a new list of elements satisfying predicate.
func (l *syncList[T]) Filter(predicate func(element T) bool) List[T] {
	snap := l.ToSlice()
	result := make([]T, 0, len(snap))
	for _, v := range snap {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return &syncList[T]{data: result}
}

// Sort sorts all elements in place.
func (l *syncList[T]) Sort(cmp Comparator[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	slices.SortFunc(l.data, cmp)
}

// Any returns true if at least one element satisfies predicate.
func (l *syncList[T]) Any(predicate func(element T) bool) bool {
	_, ok := l.Find(predicate)
	return ok
}

// Every returns true if all elements satisfy predicate.
func (l *syncList[T]) Every(predicate func(element T) bool) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
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
// Serializes a snapshot of the list as a JSON array.
func (l *syncList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *syncList[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = slice
	return nil
}

// MarshalJSONTo implements jsonv2.MarshalerTo.
// Streams the same JSON as MarshalJSON into enc.
func (l *syncList[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	return jsonv2.MarshalEncode(enc, l.ToSlice())
}

// UnmarshalJSONFrom implements jsonv2.UnmarshalerFrom.
// Accepts the same JSON as UnmarshalJSON, streamed from dec.
func (l *syncList[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var slice []T
	if err := jsonv2.UnmarshalDecode(dec, &slice); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = slice
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the list.
func (l *syncList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (l *syncList[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.data = slice
	return nil
}

// Compile-time conformance.
var _ List[int] = (*syncList[int])(nil)
