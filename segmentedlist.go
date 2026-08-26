package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
	"slices"
	"sync"
)

// segmentedList is a concurrent list using segmented locking.
// - Divides the list into segments, each with its own lock
// - Provides better write concurrency than COWList
// - Suitable for balanced read/write workloads
//
// Atomicity:
//   - ATOMIC: Get, Set, Add (within segment), Contains
//   - NON-ATOMIC: Insert/Remove that cross segments, bulk operations
//   - Iteration provides snapshot semantics
type segmentedList[T any] struct {
	segments     []*segment[T]
	segmentCount int
	mu           sync.RWMutex // Global lock for structural changes
}

type segment[T any] struct {
	data []T
	mu   sync.RWMutex
}

const defaultSegmentCount = 16

// NewSegmentedList creates a new segmented list with default segment count.
func NewSegmentedList[T any]() List[T] {
	return newSegmentedList[T](defaultSegmentCount)
}

// NewSegmentedListWithSegments creates a segmented list with specified segment count.
func NewSegmentedListWithSegments[T any](segmentCount int) List[T] {
	return newSegmentedList[T](segmentCount)
}

// newSegmentedList creates the concrete list for internal callers.
func newSegmentedList[T any](segmentCount int) *segmentedList[T] {
	if segmentCount < 1 {
		segmentCount = 1
	}
	l := &segmentedList[T]{
		segments:     make([]*segment[T], segmentCount),
		segmentCount: segmentCount,
	}
	for i := range segmentCount {
		l.segments[i] = &segment[T]{data: make([]T, 0)}
	}
	return l
}

// NewSegmentedListFrom creates a segmented list from elements.
func NewSegmentedListFrom[T any](elements ...T) List[T] {
	l := newSegmentedList[T](defaultSegmentCount)
	l.AddAll(elements...)
	return l
}

// Size returns the total number of elements.
func (l *segmentedList[T]) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	total := 0
	for _, seg := range l.segments {
		seg.mu.RLock()
		total += len(seg.data)
		seg.mu.RUnlock()
	}
	return total
}

// IsEmpty reports whether the list is empty.
func (l *segmentedList[T]) IsEmpty() bool {
	return l.Size() == 0
}

// IsNotEmpty reports whether the list contains at least one element.
func (l *segmentedList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements.
func (l *segmentedList[T]) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, seg := range l.segments {
		seg.mu.Lock()
		clear(seg.data)
		seg.data = seg.data[:0]
		seg.mu.Unlock()
	}
}

// locateIndex finds which segment and local index contains the global index.
// Must be called with at least a read lock on l.mu.
func (l *segmentedList[T]) locateIndex(globalIndex int) (segIdx, localIdx int, ok bool) {
	if globalIndex < 0 {
		return -1, -1, false
	}

	offset := 0
	for i, seg := range l.segments {
		seg.mu.RLock()
		segLen := len(seg.data)
		seg.mu.RUnlock()

		if globalIndex < offset+segLen {
			return i, globalIndex - offset, true
		}
		offset += segLen
	}
	return -1, -1, false
}

// locateIndexForInsert finds where to insert at global index.
// Returns segment index and local index.
func (l *segmentedList[T]) locateIndexForInsert(globalIndex int) (segIdx, localIdx int, ok bool) {
	if globalIndex < 0 {
		return -1, -1, false
	}

	offset := 0
	for i, seg := range l.segments {
		seg.mu.RLock()
		segLen := len(seg.data)
		seg.mu.RUnlock()

		if globalIndex <= offset+segLen {
			return i, globalIndex - offset, true
		}
		offset += segLen
	}
	return -1, -1, false
}

// ToSlice returns a snapshot of all elements.
func (l *segmentedList[T]) ToSlice() []T {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// First pass: calculate total length to pre-allocate
	total := 0
	for _, seg := range l.segments {
		seg.mu.RLock()
		total += len(seg.data)
		seg.mu.RUnlock()
	}

	// Second pass: copy data with pre-allocated slice
	result := make([]T, 0, total)
	for _, seg := range l.segments {
		seg.mu.RLock()
		result = append(result, seg.data...)
		seg.mu.RUnlock()
	}
	return result
}

// String returns a string representation.
func (l *segmentedList[T]) String() string {
	return formatCollection("segmentedList", l.Seq())
}

// Seq returns a sequence of elements (snapshot).
func (l *segmentedList[T]) Seq() iter.Seq[T] {
	snap := l.ToSlice()
	return slices.Values(snap)
}

// ForEach applies action to each element.
func (l *segmentedList[T]) ForEach(action func(element T) bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, seg := range l.segments {
		seg.mu.RLock()
		data := slices.Clone(seg.data)
		seg.mu.RUnlock()

		for _, v := range data {
			if !action(v) {
				return
			}
		}
	}
}

// Get returns the element at index.
func (l *segmentedList[T]) Get(index int) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	segIdx, localIdx, ok := l.locateIndex(index)
	if !ok {
		var zero T
		return zero, false
	}

	seg := l.segments[segIdx]
	seg.mu.RLock()
	defer seg.mu.RUnlock()

	if localIdx >= len(seg.data) {
		var zero T
		return zero, false
	}
	return seg.data[localIdx], true
}

// Set replaces the element at index.
func (l *segmentedList[T]) Set(index int, element T) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	segIdx, localIdx, ok := l.locateIndex(index)
	if !ok {
		var zero T
		return zero, false
	}

	seg := l.segments[segIdx]
	seg.mu.Lock()
	defer seg.mu.Unlock()

	if localIdx >= len(seg.data) {
		var zero T
		return zero, false
	}
	old := seg.data[localIdx]
	seg.data[localIdx] = element
	return old, true
}

// First returns the first element.
func (l *segmentedList[T]) First() (T, bool) {
	return l.Get(0)
}

// Last returns the last element.
func (l *segmentedList[T]) Last() (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for i := len(l.segments) - 1; i >= 0; i-- {
		seg := l.segments[i]
		seg.mu.RLock()
		if len(seg.data) > 0 {
			v := seg.data[len(seg.data)-1]
			seg.mu.RUnlock()
			return v, true
		}
		seg.mu.RUnlock()
	}
	var zero T
	return zero, false
}

// Add appends the element to the last segment.
func (l *segmentedList[T]) Add(element T) {
	l.mu.RLock()
	seg := l.segments[len(l.segments)-1]
	l.mu.RUnlock()

	seg.mu.Lock()
	seg.data = append(seg.data, element)
	seg.mu.Unlock()
}

// AddAll appends all elements.
func (l *segmentedList[T]) AddAll(elements ...T) {
	if len(elements) == 0 {
		return
	}
	l.mu.RLock()
	seg := l.segments[len(l.segments)-1]
	l.mu.RUnlock()

	seg.mu.Lock()
	seg.data = append(seg.data, elements...)
	seg.mu.Unlock()
}

// AddSeq appends all elements from the sequence.
func (l *segmentedList[T]) AddSeq(seq iter.Seq[T]) {
	var buf []T
	for v := range seq {
		buf = append(buf, v)
	}
	l.AddAll(buf...)
}

// Insert inserts the element at index.
func (l *segmentedList[T]) Insert(index int, element T) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	segIdx, localIdx, ok := l.locateIndexForInsert(index)
	if !ok {
		return false
	}

	seg := l.segments[segIdx]
	seg.mu.Lock()
	defer seg.mu.Unlock()

	if localIdx > len(seg.data) {
		return false
	}

	seg.data = slices.Insert(seg.data, localIdx, element)
	return true
}

// InsertAll inserts all elements at index.
func (l *segmentedList[T]) InsertAll(index int, elements ...T) bool {
	if len(elements) == 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	segIdx, localIdx, ok := l.locateIndexForInsert(index)
	if !ok {
		return false
	}

	seg := l.segments[segIdx]
	seg.mu.Lock()
	defer seg.mu.Unlock()

	if localIdx > len(seg.data) {
		return false
	}

	seg.data = slices.Insert(seg.data, localIdx, elements...)
	return true
}

// RemoveAt removes the element at index.
func (l *segmentedList[T]) RemoveAt(index int) (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	segIdx, localIdx, ok := l.locateIndex(index)
	if !ok {
		var zero T
		return zero, false
	}

	seg := l.segments[segIdx]
	seg.mu.Lock()
	defer seg.mu.Unlock()

	if localIdx >= len(seg.data) {
		var zero T
		return zero, false
	}

	removed := seg.data[localIdx]
	oldLen := len(seg.data)
	seg.data = slices.Delete(seg.data, localIdx, localIdx+1)
	clear(seg.data[len(seg.data):oldLen])
	return removed, true
}

// Remove removes the first occurrence of element.
func (l *segmentedList[T]) Remove(element T, eq Equaler[T]) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, seg := range l.segments {
		seg.mu.Lock()
		for i, v := range seg.data {
			if eq(v, element) {
				oldLen := len(seg.data)
				seg.data = slices.Delete(seg.data, i, i+1)
				clear(seg.data[len(seg.data):oldLen])
				seg.mu.Unlock()
				return true
			}
		}
		seg.mu.Unlock()
	}
	return false
}

// RemoveFirst removes and returns the first element.
func (l *segmentedList[T]) RemoveFirst() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, seg := range l.segments {
		seg.mu.Lock()
		if len(seg.data) > 0 {
			removed := seg.data[0]
			// Clear first slot to avoid retaining references.
			var zero T
			seg.data[0] = zero
			seg.data = seg.data[1:]
			seg.mu.Unlock()
			return removed, true
		}
		seg.mu.Unlock()
	}
	var zero T
	return zero, false
}

// RemoveLast removes and returns the last element.
func (l *segmentedList[T]) RemoveLast() (T, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i := len(l.segments) - 1; i >= 0; i-- {
		seg := l.segments[i]
		seg.mu.Lock()
		if len(seg.data) > 0 {
			removed := seg.data[len(seg.data)-1]
			// Clear last slot to avoid retaining references.
			var zero T
			seg.data[len(seg.data)-1] = zero
			seg.data = seg.data[:len(seg.data)-1]
			seg.mu.Unlock()
			return removed, true
		}
		seg.mu.Unlock()
	}
	var zero T
	return zero, false
}

// RemoveFunc removes all elements satisfying predicate.
func (l *segmentedList[T]) RemoveFunc(predicate func(element T) bool) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	for _, seg := range l.segments {
		seg.mu.Lock()
		// Filter in-place
		n := 0
		for _, v := range seg.data {
			if predicate(v) {
				removed++
			} else {
				seg.data[n] = v
				n++
			}
		}
		clear(seg.data[n:])
		seg.data = seg.data[:n]
		seg.mu.Unlock()
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate.
func (l *segmentedList[T]) RetainFunc(predicate func(element T) bool) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	for _, seg := range l.segments {
		seg.mu.Lock()
		// Filter in-place
		n := 0
		for _, v := range seg.data {
			if !predicate(v) {
				removed++
			} else {
				seg.data[n] = v
				n++
			}
		}
		clear(seg.data[n:])
		seg.data = seg.data[:n]
		seg.mu.Unlock()
	}
	return removed
}

// IndexOf returns the index of the first occurrence.
func (l *segmentedList[T]) IndexOf(element T, eq Equaler[T]) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	offset := 0
	for _, seg := range l.segments {
		seg.mu.RLock()
		for i, v := range seg.data {
			if eq(v, element) {
				seg.mu.RUnlock()
				return offset + i
			}
		}
		offset += len(seg.data)
		seg.mu.RUnlock()
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence.
func (l *segmentedList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Calculate total offset first
	offsets := make([]int, len(l.segments))
	offset := 0
	for i, seg := range l.segments {
		offsets[i] = offset
		seg.mu.RLock()
		offset += len(seg.data)
		seg.mu.RUnlock()
	}

	for i := len(l.segments) - 1; i >= 0; i-- {
		seg := l.segments[i]
		seg.mu.RLock()
		for j := len(seg.data) - 1; j >= 0; j-- {
			if eq(seg.data[j], element) {
				seg.mu.RUnlock()
				return offsets[i] + j
			}
		}
		seg.mu.RUnlock()
	}
	return -1
}

// Contains reports whether element exists.
func (l *segmentedList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate.
func (l *segmentedList[T]) Find(predicate func(element T) bool) (T, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, seg := range l.segments {
		seg.mu.RLock()
		for _, v := range seg.data {
			if predicate(v) {
				seg.mu.RUnlock()
				return v, true
			}
		}
		seg.mu.RUnlock()
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate.
func (l *segmentedList[T]) FindIndex(predicate func(element T) bool) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	offset := 0
	for _, seg := range l.segments {
		seg.mu.RLock()
		for i, v := range seg.data {
			if predicate(v) {
				seg.mu.RUnlock()
				return offset + i
			}
		}
		offset += len(seg.data)
		seg.mu.RUnlock()
	}
	return -1
}

// SubList returns a new list containing elements in [from, to).
func (l *segmentedList[T]) SubList(from, to int) List[T] {
	snap := l.ToSlice()
	if from < 0 || to > len(snap) || from > to {
		return NewSegmentedList[T]()
	}
	return NewSegmentedListFrom(snap[from:to]...)
}

// Reversed returns a sequence iterating in reverse order.
func (l *segmentedList[T]) Reversed() iter.Seq[T] {
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
func (l *segmentedList[T]) Clone() List[T] {
	return NewSegmentedListFrom(l.ToSlice()...)
}

// Filter returns a new list of elements satisfying predicate.
func (l *segmentedList[T]) Filter(predicate func(element T) bool) List[T] {
	var result []T
	for _, v := range l.ToSlice() {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return NewSegmentedListFrom(result...)
}

// Sort sorts all elements in place.
func (l *segmentedList[T]) Sort(cmp Comparator[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Collect all data
	var all []T
	for _, seg := range l.segments {
		seg.mu.Lock()
		all = append(all, seg.data...)
		seg.mu.Unlock()
	}

	// Sort
	slices.SortFunc(all, cmp)

	// Redistribute to segments
	perSeg := (len(all) + l.segmentCount - 1) / l.segmentCount
	idx := 0
	for _, seg := range l.segments {
		seg.mu.Lock()
		end := min(idx+perSeg, len(all))
		if idx < len(all) {
			seg.data = slices.Clone(all[idx:end])
		} else {
			seg.data = make([]T, 0)
		}
		idx = end
		seg.mu.Unlock()
	}
}

// Any returns true if at least one element satisfies predicate.
func (l *segmentedList[T]) Any(predicate func(element T) bool) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, seg := range l.segments {
		seg.mu.RLock()
		found := slices.ContainsFunc(seg.data, predicate)
		seg.mu.RUnlock()
		if found {
			return true
		}
	}
	return false
}

// Every returns true if all elements satisfy predicate.
func (l *segmentedList[T]) Every(predicate func(element T) bool) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, seg := range l.segments {
		seg.mu.RLock()
		for _, v := range seg.data {
			if !predicate(v) {
				seg.mu.RUnlock()
				return false
			}
		}
		seg.mu.RUnlock()
	}
	return true
}

// ==========================
// Serialization
// ==========================

// MarshalJSON implements json.Marshaler.
// Serializes a snapshot of the list as a JSON array.
// NOTE: Provides snapshot consistency - the serialized data represents
// a consistent view but may be slightly stale.
func (l *segmentedList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *segmentedList[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	// Clear and rebuild without copying locks
	newList := newSegmentedList[T](defaultSegmentCount)
	for _, elem := range slice {
		newList.Add(elem)
	}
	l.segments = newList.segments
	l.segmentCount = newList.segmentCount
	return nil
}

// GobEncode implements gob.GobEncoder.
// Serializes a snapshot of the list.
func (l *segmentedList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
// Deserializes from gob data.
func (l *segmentedList[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	// Clear and rebuild without copying locks
	newList := newSegmentedList[T](defaultSegmentCount)
	for _, elem := range slice {
		newList.Add(elem)
	}
	l.segments = newList.segments
	l.segmentCount = newList.segmentCount
	return nil
}

// Compile-time conformance.
var _ List[int] = (*segmentedList[int])(nil)
