package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"iter"
)

// node is a doubly-linked list node.
type node[T any] struct {
	value T
	prev  *node[T]
	next  *node[T]
}

// linkedList is a doubly-linked list implementation of List[T].
// - O(1) insertion/removal at both ends
// - O(1) insertion/removal at a known position
// - O(n) random access by index
// - O(n) search
type linkedList[T any] struct {
	head *node[T]
	tail *node[T]
	size int
}

// NewLinkedList creates an empty LinkedList.
func NewLinkedList[T any]() List[T] {
	return &linkedList[T]{}
}

// NewLinkedListFrom creates a LinkedList containing the provided elements.
func NewLinkedListFrom[T any](elements ...T) List[T] {
	l := &linkedList[T]{}
	for _, e := range elements {
		l.Add(e)
	}
	return l
}

// Size returns the number of elements.
func (l *linkedList[T]) Size() int { return l.size }

// IsEmpty reports whether the list is empty.
func (l *linkedList[T]) IsEmpty() bool    { return l.size == 0 }
func (l *linkedList[T]) IsNotEmpty() bool { return !l.IsEmpty() }

// Clear removes all elements.
func (l *linkedList[T]) Clear() {
	l.head = nil
	l.tail = nil
	l.size = 0
}

// ToSlice returns a snapshot copy of the elements in order.
func (l *linkedList[T]) ToSlice() []T {
	out := make([]T, 0, l.size)
	for n := l.head; n != nil; n = n.next {
		out = append(out, n.value)
	}
	return out
}

// String returns a concise representation.
func (l *linkedList[T]) String() string {
	return formatCollection("linkedList", l.Seq())
}

// Seq returns a sequence of elements in order.
func (l *linkedList[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := l.head; n != nil; n = n.next {
			if !yield(n.value) {
				return
			}
		}
	}
}

// ForEach applies action to each element; stops early if action returns false.
func (l *linkedList[T]) ForEach(action func(element T) bool) {
	for n := l.head; n != nil; n = n.next {
		if !action(n.value) {
			return
		}
	}
}

// nodeAt returns the node at the given index, or nil if out of bounds.
func (l *linkedList[T]) nodeAt(index int) *node[T] {
	if index < 0 || index >= l.size {
		return nil
	}
	// Optimize by starting from the closer end
	if index < l.size/2 {
		n := l.head
		for range index {
			n = n.next
		}
		return n
	}
	n := l.tail
	for i := l.size - 1; i > index; i-- {
		n = n.prev
	}
	return n
}

// Get returns the element at index, or (zero, false) if out of bounds.
func (l *linkedList[T]) Get(index int) (T, bool) {
	n := l.nodeAt(index)
	if n == nil {
		var zero T
		return zero, false
	}
	return n.value, true
}

// Set replaces the element at index. Returns (oldValue, true) if successful.
func (l *linkedList[T]) Set(index int, element T) (T, bool) {
	n := l.nodeAt(index)
	if n == nil {
		var zero T
		return zero, false
	}
	old := n.value
	n.value = element
	return old, true
}

// First returns the first element, or (zero, false) if empty.
func (l *linkedList[T]) First() (T, bool) {
	if l.head == nil {
		var zero T
		return zero, false
	}
	return l.head.value, true
}

// Last returns the last element, or (zero, false) if empty.
func (l *linkedList[T]) Last() (T, bool) {
	if l.tail == nil {
		var zero T
		return zero, false
	}
	return l.tail.value, true
}

// Add appends the element to the end. O(1).
func (l *linkedList[T]) Add(element T) {
	n := &node[T]{value: element, prev: l.tail}
	if l.tail != nil {
		l.tail.next = n
	} else {
		l.head = n
	}
	l.tail = n
	l.size++
}

// AddAll appends all elements to the end.
func (l *linkedList[T]) AddAll(elements ...T) {
	for _, e := range elements {
		l.Add(e)
	}
}

// AddSeq appends all elements from the sequence.
func (l *linkedList[T]) AddSeq(seq iter.Seq[T]) {
	for v := range seq {
		l.Add(v)
	}
}

// AddFirst adds an element to the front. O(1).
func (l *linkedList[T]) AddFirst(element T) {
	n := &node[T]{value: element, next: l.head}
	if l.head != nil {
		l.head.prev = n
	} else {
		l.tail = n
	}
	l.head = n
	l.size++
}

// Insert inserts the element at index, shifting subsequent elements right.
// Returns false if index is out of bounds.
func (l *linkedList[T]) Insert(index int, element T) bool {
	if index < 0 || index > l.size {
		return false
	}
	if index == 0 {
		l.AddFirst(element)
		return true
	}
	if index == l.size {
		l.Add(element)
		return true
	}
	// Insert before the node at index
	target := l.nodeAt(index)
	n := &node[T]{value: element, prev: target.prev, next: target}
	target.prev.next = n
	target.prev = n
	l.size++
	return true
}

// InsertAll inserts all elements at index, shifting subsequent elements right.
func (l *linkedList[T]) InsertAll(index int, elements ...T) bool {
	if index < 0 || index > l.size {
		return false
	}
	for i, e := range elements {
		if !l.Insert(index+i, e) {
			return false
		}
	}
	return true
}

// removeNode removes a node from the list.
func (l *linkedList[T]) removeNode(n *node[T]) T {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
	l.size--
	return n.value
}

// RemoveAt removes the element at index. Returns (removed, true) if successful.
func (l *linkedList[T]) RemoveAt(index int) (T, bool) {
	n := l.nodeAt(index)
	if n == nil {
		var zero T
		return zero, false
	}
	return l.removeNode(n), true
}

// Remove removes the first occurrence of element. Returns true if removed.
func (l *linkedList[T]) Remove(element T, eq Equaler[T]) bool {
	for n := l.head; n != nil; n = n.next {
		if eq(n.value, element) {
			l.removeNode(n)
			return true
		}
	}
	return false
}

// RemoveFirst removes and returns the first element. O(1).
func (l *linkedList[T]) RemoveFirst() (T, bool) {
	if l.head == nil {
		var zero T
		return zero, false
	}
	return l.removeNode(l.head), true
}

// RemoveLast removes and returns the last element. O(1).
func (l *linkedList[T]) RemoveLast() (T, bool) {
	if l.tail == nil {
		var zero T
		return zero, false
	}
	return l.removeNode(l.tail), true
}

// RemoveFunc removes all elements satisfying predicate. Returns count removed.
func (l *linkedList[T]) RemoveFunc(predicate func(element T) bool) int {
	removed := 0
	n := l.head
	for n != nil {
		next := n.next
		if predicate(n.value) {
			l.removeNode(n)
			removed++
		}
		n = next
	}
	return removed
}

// RetainFunc keeps only elements satisfying predicate. Returns count removed.
func (l *linkedList[T]) RetainFunc(predicate func(element T) bool) int {
	removed := 0
	n := l.head
	for n != nil {
		next := n.next
		if !predicate(n.value) {
			l.removeNode(n)
			removed++
		}
		n = next
	}
	return removed
}

// IndexOf returns the index of the first occurrence, or -1.
func (l *linkedList[T]) IndexOf(element T, eq Equaler[T]) int {
	i := 0
	for n := l.head; n != nil; n = n.next {
		if eq(n.value, element) {
			return i
		}
		i++
	}
	return -1
}

// LastIndexOf returns the index of the last occurrence, or -1.
func (l *linkedList[T]) LastIndexOf(element T, eq Equaler[T]) int {
	i := l.size - 1
	for n := l.tail; n != nil; n = n.prev {
		if eq(n.value, element) {
			return i
		}
		i--
	}
	return -1
}

// Contains reports whether element exists using eq for comparison.
func (l *linkedList[T]) Contains(element T, eq Equaler[T]) bool {
	return l.IndexOf(element, eq) >= 0
}

// Find returns the first element satisfying predicate, or (zero, false).
func (l *linkedList[T]) Find(predicate func(element T) bool) (T, bool) {
	for n := l.head; n != nil; n = n.next {
		if predicate(n.value) {
			return n.value, true
		}
	}
	var zero T
	return zero, false
}

// FindIndex returns the index of the first element satisfying predicate, or -1.
func (l *linkedList[T]) FindIndex(predicate func(element T) bool) int {
	i := 0
	for n := l.head; n != nil; n = n.next {
		if predicate(n.value) {
			return i
		}
		i++
	}
	return -1
}

// SubList returns a new list containing elements in [from, to).
func (l *linkedList[T]) SubList(from, to int) List[T] {
	if from < 0 || to > l.size || from > to {
		return NewLinkedList[T]()
	}
	out := &linkedList[T]{}
	n := l.nodeAt(from)
	for i := from; i < to && n != nil; i++ {
		out.Add(n.value)
		n = n.next
	}
	return out
}

// Reversed returns a sequence iterating in reverse order.
func (l *linkedList[T]) Reversed() iter.Seq[T] {
	return func(yield func(T) bool) {
		for n := l.tail; n != nil; n = n.prev {
			if !yield(n.value) {
				return
			}
		}
	}
}

// Clone returns a shallow copy.
func (l *linkedList[T]) Clone() List[T] {
	out := &linkedList[T]{}
	for n := l.head; n != nil; n = n.next {
		out.Add(n.value)
	}
	return out
}

// Filter returns a new list of elements satisfying predicate.
func (l *linkedList[T]) Filter(predicate func(element T) bool) List[T] {
	out := &linkedList[T]{}
	for n := l.head; n != nil; n = n.next {
		if predicate(n.value) {
			out.Add(n.value)
		}
	}
	return out
}

// Sort sorts elements in place using the comparator.
// Uses merge sort for O(n log n) time complexity.
func (l *linkedList[T]) Sort(cmp Comparator[T]) {
	if l.size <= 1 {
		return
	}
	l.head = l.mergeSort(l.head, cmp)
	// Rebuild prev pointers and find tail
	var prev *node[T]
	for n := l.head; n != nil; n = n.next {
		n.prev = prev
		if n.next == nil {
			l.tail = n
		}
		prev = n
	}
}

// mergeSort performs merge sort on the linked list.
func (l *linkedList[T]) mergeSort(head *node[T], cmp Comparator[T]) *node[T] {
	if head == nil || head.next == nil {
		return head
	}
	// Find middle
	slow, fast := head, head.next
	for fast != nil && fast.next != nil {
		slow = slow.next
		fast = fast.next.next
	}
	mid := slow.next
	slow.next = nil
	// Sort both halves
	left := l.mergeSort(head, cmp)
	right := l.mergeSort(mid, cmp)
	// Merge
	return l.merge(left, right, cmp)
}

// merge merges two sorted lists.
func (l *linkedList[T]) merge(left, right *node[T], cmp Comparator[T]) *node[T] {
	dummy := &node[T]{}
	curr := dummy
	for left != nil && right != nil {
		if cmp(left.value, right.value) <= 0 {
			curr.next = left
			left = left.next
		} else {
			curr.next = right
			right = right.next
		}
		curr = curr.next
	}
	if left != nil {
		curr.next = left
	} else {
		curr.next = right
	}
	return dummy.next
}

// Any returns true if at least one element satisfies predicate.
func (l *linkedList[T]) Any(predicate func(element T) bool) bool {
	for n := l.head; n != nil; n = n.next {
		if predicate(n.value) {
			return true
		}
	}
	return false
}

// Every returns true if all elements satisfy predicate.
func (l *linkedList[T]) Every(predicate func(element T) bool) bool {
	for n := l.head; n != nil; n = n.next {
		if !predicate(n.value) {
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
func (l *linkedList[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.ToSlice())
}

// UnmarshalJSON implements json.Unmarshaler.
// Deserializes from a JSON array.
func (l *linkedList[T]) UnmarshalJSON(data []byte) error {
	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}
	// Clear existing list
	l.head = nil
	l.tail = nil
	l.size = 0
	// Add all elements
	for _, elem := range slice {
		l.Add(elem)
	}
	return nil
}

// GobEncode implements gob.GobEncoder.
func (l *linkedList[T]) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(l.ToSlice()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder.
func (l *linkedList[T]) GobDecode(data []byte) error {
	var slice []T
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&slice); err != nil {
		return err
	}
	// Clear existing list
	l.head = nil
	l.tail = nil
	l.size = 0
	// Add all elements
	for _, elem := range slice {
		l.Add(elem)
	}
	return nil
}

// Compile-time conformance
var (
	_ List[int] = (*linkedList[int])(nil)
)
