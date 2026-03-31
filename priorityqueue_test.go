package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPriorityQueue_Basic(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()
	assert.True(t, pq.IsEmpty(), "New queue should be empty")
	assert.Equal(t, 0, pq.Size(), "Empty queue size should be 0")

	pq.Push(3)
	pq.Push(1)
	pq.Push(4)
	pq.Push(1)
	pq.Push(5)

	assert.False(t, pq.IsEmpty(), "Queue should not be empty after pushes")
	assert.Equal(t, 5, pq.Size(), "Size should reflect number of pushes")

	// Min-heap: smallest first
	v, ok := pq.Peek()
	require.True(t, ok, "Peek should succeed on non-empty queue")
	assert.Equal(t, 1, v, "Peek should return smallest element")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 1, v, "First pop should return smallest element")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 1, v, "Second pop should return next smallest")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 3, v, "Third pop should return 3")
}

func TestPriorityQueue_MaxHeap(t *testing.T) {
	t.Parallel()
	pq := NewMaxPriorityQueue[int]()

	pq.Push(3)
	pq.Push(1)
	pq.Push(4)
	pq.Push(1)
	pq.Push(5)

	// Max-heap: largest first
	v, ok := pq.Peek()
	require.True(t, ok, "Peek should succeed on non-empty queue")
	assert.Equal(t, 5, v, "Peek should return largest element")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 5, v, "First pop should return largest")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 4, v, "Second pop should return next largest")

	v, ok = pq.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 3, v, "Third pop should return 3")
}

func TestPriorityQueue_Empty(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()

	_, ok := pq.Peek()
	assert.False(t, ok, "Peek on empty should fail")

	_, ok = pq.Pop()
	assert.False(t, ok, "Pop on empty should fail")
}

func TestPriorityQueue_Clear(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()
	pq.PushAll(1, 2, 3, 4, 5)

	assert.Equal(t, 5, pq.Size(), "Size should be 5 after PushAll")

	pq.Clear()
	assert.True(t, pq.IsEmpty(), "Queue should be empty after Clear")
	assert.Equal(t, 0, pq.Size(), "Size should be 0 after Clear")
}

func TestPriorityQueue_PushAll(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()
	pq.PushAll(5, 3, 1, 4, 2)

	assert.Equal(t, 5, pq.Size(), "Size should be 5 after PushAll")

	// Should pop in sorted order (min-heap)
	v, _ := pq.Pop()
	assert.Equal(t, 1, v, "First pop should be smallest")
	v, _ = pq.Pop()
	assert.Equal(t, 2, v, "Second pop should be next smallest")
}

func TestPriorityQueue_From(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueFrom(CompareFunc[int](), 5, 3, 1, 4, 2)

	assert.Equal(t, 5, pq.Size(), "Size should equal number of inputs")

	v, _ := pq.Peek()
	assert.Equal(t, 1, v, "Peek should return smallest element")
}

func TestPriorityQueue_ToSlice(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueFrom(CompareFunc[int](), 5, 3, 1, 4, 2)

	slice := pq.ToSlice()
	assert.Equal(t, 5, len(slice), "ToSlice should return all elements")
	// ToSlice returns heap order, not sorted
}

func TestPriorityQueue_ToSortedSlice(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueFrom(CompareFunc[int](), 5, 3, 1, 4, 2)

	sorted := pq.ToSortedSlice()
	assert.Equal(t, []int{1, 2, 3, 4, 5}, sorted, "ToSortedSlice should return ascending order")
}

func TestPriorityQueue_Seq(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueFrom(CompareFunc[int](), 3, 1, 2)

	var collected []int
	for v := range pq.Seq() {
		collected = append(collected, v)
	}
	assert.Equal(t, 3, len(collected), "Seq should iterate all elements")
}

func TestPriorityQueue_String(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueFrom(CompareFunc[int](), 1, 2, 3)
	s := pq.String()
	assert.Contains(t, s, "priorityQueue", "String should include type name")
}

func TestPriorityQueue_CustomComparator(t *testing.T) {
	t.Parallel()
	// Priority by string length (shorter = higher priority)
	pq := NewPriorityQueue(func(a, b string) int {
		return len(a) - len(b)
	})

	pq.Push("hello")
	pq.Push("hi")
	pq.Push("hey")
	pq.Push("h")

	v, _ := pq.Pop()
	assert.Equal(t, "h", v, "Shortest string should be popped first")

	v, _ = pq.Pop()
	assert.Equal(t, "hi", v, "Next shortest should be popped")

	v, _ = pq.Pop()
	assert.Equal(t, "hey", v, "Then 'hey'")

	v, _ = pq.Pop()
	assert.Equal(t, "hello", v, "Longest should be popped last")
}

func TestPriorityQueue_WithCapacity(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueWithCapacity(CompareFunc[int](), 100)
	assert.True(t, pq.IsEmpty(), "New queue should be empty")

	pq.Push(1)
	assert.Equal(t, 1, pq.Size(), "Size should be 1 after one push")
}

func TestPriorityQueue_WithCapacityNegative(t *testing.T) {
	t.Parallel()
	// Test with negative capacity (should be normalized to 0)
	pq := NewPriorityQueueWithCapacity(CompareFunc[int](), -10)
	assert.True(t, pq.IsEmpty(), "Queue with negative capacity should be empty")

	pq.Push(1)
	pq.Push(2)
	assert.Equal(t, 2, pq.Size(), "Should be able to add elements despite negative initial capacity")
}

func TestPriorityQueue_Panic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		var nilCmp Comparator[int]
		NewPriorityQueue(nilCmp)
	}, "NewPriorityQueue should panic on nil comparator")

	assert.Panics(t, func() {
		var nilCmp Comparator[int]
		NewPriorityQueueWithCapacity(nilCmp, 10)
	}, "NewPriorityQueueWithCapacity should panic on nil comparator")

	assert.Panics(t, func() {
		var nilCmp Comparator[int]
		NewPriorityQueueFrom(nilCmp, 1, 2, 3)
	}, "NewPriorityQueueFrom should panic on nil comparator")
}

func TestPriorityQueue_LargeDataset(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()

	// Push 1000 elements in reverse order
	for i := 1000; i > 0; i-- {
		pq.Push(i)
	}

	assert.Equal(t, 1000, pq.Size(), "Size should be 1000 after pushes")

	// Pop should return in sorted order
	for i := 1; i <= 1000; i++ {
		v, ok := pq.Pop()
		require.True(t, ok, "Pop should succeed")
		assert.Equal(t, i, v, "Popped value should be increasing")
	}

	assert.True(t, pq.IsEmpty(), "Queue should be empty after popping all")
}

func TestPriorityQueue_Duplicates(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueueOrdered[int]()
	pq.PushAll(3, 3, 3, 2, 2, 1, 1)

	assert.Equal(t, 7, pq.Size(), "Size should include duplicates")

	// All duplicates should be preserved
	v, _ := pq.Pop()
	assert.Equal(t, 1, v, "First duplicate")
	v, _ = pq.Pop()
	assert.Equal(t, 1, v, "Second duplicate")
	v, _ = pq.Pop()
	assert.Equal(t, 2, v, "Third element should be 2")
	v, _ = pq.Pop()
	assert.Equal(t, 2, v, "Fourth element should be 2")
	v, _ = pq.Pop()
	assert.Equal(t, 3, v, "Fifth element should be 3")
	v, _ = pq.Pop()
	assert.Equal(t, 3, v, "Sixth element should be 3")
	v, _ = pq.Pop()
	assert.Equal(t, 3, v, "Final elements should be 3")
}

func TestPriorityQueue_IsNotEmpty(t *testing.T) {
	t.Parallel()
	pq := NewPriorityQueue[int](func(a, b int) int { return a - b })
	assert.False(t, pq.IsNotEmpty(), "new priority queue should not be non-empty")
	pq.Push(1)
	assert.True(t, pq.IsNotEmpty(), "priority queue with element should be non-empty")
}
