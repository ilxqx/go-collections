package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrayQueue_BasicFIFO(t *testing.T) {
	t.Parallel()
	q := NewArrayQueue[int]()
	assert.True(t, q.IsEmpty(), "New queue should be empty")
	q.Enqueue(1)
	q.EnqueueAll(2, 3)
	assert.Equal(t, 3, q.Size(), "Size should be 3 after EnqueueAll")
	v, ok := q.Peek()
	require.True(t, ok, "Peek should succeed on non-empty queue")
	assert.Equal(t, 1, v, "Peek should return front element")
	// Seq front->back: 1,2,3
	want := 1
	for v := range q.Seq() {
		assert.Equal(t, want, v, "Seq should yield FIFO order")
		want++
	}
	v, ok = q.Dequeue()
	require.True(t, ok, "Dequeue should succeed")
	assert.Equal(t, 1, v, "Dequeue should return oldest element")
	v, ok = q.Dequeue()
	require.True(t, ok, "Dequeue should succeed")
	assert.Equal(t, 2, v, "Dequeue should return next element")
	v, ok = q.Dequeue()
	require.True(t, ok, "Dequeue should succeed")
	assert.Equal(t, 3, v, "Dequeue should return last element")
	_, ok = q.Dequeue()
	require.False(t, ok, "Dequeue on empty should fail")
}

func TestArrayQueue_GrowthAndCompact(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueWithCapacity[int](1)
	for i := range 100 {
		q.Enqueue(i)
	}
	for i := range 50 {
		v, ok := q.Dequeue()
		require.True(t, ok, "Dequeue should succeed during draining")
		assert.Equal(t, i, v, "Dequeue mismatch at %d", i)
	}
	// After many dequeues, ensure the remaining sequence is correct
	want := 50
	for v := range q.Seq() {
		assert.Equal(t, want, v, "Remaining sequence should continue from 50")
		want++
	}
	assert.Equal(t, 50, q.Size(), "Size should be 50 after draining half")
}

func TestArrayQueue_Clear(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueFrom(1, 2, 3)
	assert.False(t, q.IsEmpty(), "Queue should start non-empty")
	q.Clear()
	assert.True(t, q.IsEmpty(), "Queue should be empty after Clear")
	assert.Equal(t, 0, q.Size(), "Size should be 0 after Clear")
}

func TestArrayQueue_ToSlice(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueFrom(1, 2, 3)
	slice := q.ToSlice()
	assert.Equal(t, 3, len(slice), "ToSlice should return 3 elements")
	assert.Equal(t, 1, slice[0], "Front element should be first in slice")
	assert.Equal(t, 3, slice[2], "Back element should be last in slice")
}

func TestArrayQueue_From(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueFrom(5, 4, 3)
	assert.Equal(t, 3, q.Size(), "Size should equal number of inputs")
	v, _ := q.Dequeue()
	assert.Equal(t, 5, v, "Dequeue should return first input element")
}

func TestArrayQueue_String(t *testing.T) {
	t.Parallel()
	q := NewArrayQueue[int]()
	q.Enqueue(1)
	str := q.String()
	assert.Contains(t, str, "arrayQueue", "String should include type name")
	assert.Contains(t, str, "1", "String should render element value")
}

func TestArrayQueue_PeekEmpty(t *testing.T) {
	t.Parallel()
	q := NewArrayQueue[int]()
	_, ok := q.Peek()
	assert.False(t, ok, "Peek should return false on empty queue")
}

func TestArrayQueue_WithCapacityZero(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueWithCapacity[int](0)
	assert.True(t, q.IsEmpty(), "Queue with zero capacity should be empty")
	q.Enqueue(1)
	assert.Equal(t, 1, q.Size(), "Size should be 1 after Enqueue")
	v, ok := q.Dequeue()
	require.True(t, ok, "Dequeue should succeed")
	assert.Equal(t, 1, v, "Should return enqueued value")
}

func TestArrayQueue_WithCapacityNegative(t *testing.T) {
	t.Parallel()
	q := NewArrayQueueWithCapacity[int](-5)
	assert.True(t, q.IsEmpty(), "Queue with negative capacity should be empty")
	q.Enqueue(42)
	assert.Equal(t, 1, q.Size(), "Size should be 1 after Enqueue")
}

func TestArrayQueue_ShrinkAfterPeak(t *testing.T) {
	t.Parallel()
	// Test the shrink strategy: when head > 3/4 cap and live < 1/4 cap,
	// the capacity should shrink to release memory.
	q := NewArrayQueue[int]().(*arrayQueue[int])

	// Enqueue a large number of elements to create peak capacity
	const peakSize = 1000
	for i := range peakSize {
		q.Enqueue(i)
	}
	peakCap := cap(q.data)
	assert.GreaterOrEqual(t, peakCap, peakSize, "Peak capacity should be >= peak size")

	// Dequeue most elements, leaving very few
	// This should trigger the shrink strategy
	for range peakSize - 10 {
		q.Dequeue()
	}

	// After shrinking, capacity should be significantly smaller
	currentCap := cap(q.data)
	assert.Less(t, currentCap, peakCap/2,
		"Capacity should shrink after peak-then-low-water scenario: peak=%d, current=%d",
		peakCap, currentCap)

	// Verify remaining elements are still correct
	assert.Equal(t, 10, q.Size(), "Size should be 10 after draining to last elements")
	for i := peakSize - 10; i < peakSize; i++ {
		v, ok := q.Dequeue()
		require.True(t, ok, "Dequeue should succeed after shrink")
		assert.Equal(t, i, v, "Remaining elements should be in order")
	}
}

func TestArrayQueue_ShrinkTriggersReallocation(t *testing.T) {
	t.Parallel()
	// Test that shrink behavior triggers reallocation and subsequent
	// enqueue/dequeue operations have stable performance.
	q := NewArrayQueue[int]().(*arrayQueue[int])

	// Build up to a large capacity
	const peakSize = 500
	for i := range peakSize {
		q.Enqueue(i)
	}

	// Record capacity after peak
	peakCap := cap(q.data)
	require.GreaterOrEqual(t, peakCap, peakSize, "Peak capacity should be >= peak size")

	// Dequeue almost all elements to trigger shrink
	for range peakSize - 5 {
		q.Dequeue()
	}

	// After shrink, capacity should be reduced
	shrunkCap := cap(q.data)
	assert.Less(t, shrunkCap, peakCap/2,
		"Capacity should shrink significantly: peak=%d, shrunk=%d", peakCap, shrunkCap)

	// Enqueue more elements - should work correctly after shrink
	for i := range 50 {
		q.Enqueue(1000 + i)
	}

	// Verify queue still works correctly
	assert.Equal(t, 55, q.Size(), "Size should be remaining originals + new enqueues")

	// Dequeue and verify correct order
	for i := peakSize - 5; i < peakSize; i++ {
		v, ok := q.Dequeue()
		require.True(t, ok, "Dequeue should succeed")
		assert.Equal(t, i, v, "Remaining original elements should be in order")
	}
	for i := range 50 {
		v, ok := q.Dequeue()
		require.True(t, ok, "Dequeue should succeed")
		assert.Equal(t, 1000+i, v, "New elements should be in order")
	}
	assert.True(t, q.IsEmpty(), "Queue should be empty after draining")
}

func TestArrayQueue_ShrinkDoesNotThrashOnSmallQueues(t *testing.T) {
	t.Parallel()
	// Verify that small queues don't shrink (capacity <= 64 threshold)
	q := NewArrayQueue[int]().(*arrayQueue[int])

	// Use a small queue
	for i := range 50 {
		q.Enqueue(i)
	}
	initialCap := cap(q.data)

	// Dequeue most elements
	for range 45 {
		q.Dequeue()
	}

	// Small queues should not shrink to avoid thrashing
	// The threshold in compact() is cap > 64
	if initialCap <= 64 {
		// For small queues, capacity should remain (in-place shift, not shrink)
		assert.LessOrEqual(t, cap(q.data), initialCap,
			"Small queues should not allocate larger, but may compact in-place")
	}

	// Queue should still function correctly
	assert.Equal(t, 5, q.Size(), "Size should be number of remaining elements")
	for i := 45; i < 50; i++ {
		v, ok := q.Dequeue()
		require.True(t, ok, "Dequeue should succeed")
		assert.Equal(t, i, v, "Elements should be dequeued in order")
	}
}

func TestArrayQueue_ShrinkReducesAllocations(t *testing.T) {
	// Test that after shrink, subsequent enqueue/dequeue operations
	// require fewer allocations due to reduced capacity.
	// This protects the shrink optimization.

	const peakSize = 200
	const steadyOps = 50

	// Scenario: Build up to peak, drain to trigger shrink, then steady-state ops
	allocs := testing.AllocsPerRun(10, func() {
		q := NewArrayQueue[int]().(*arrayQueue[int])

		// Phase 1: Build up to peak capacity
		for i := range peakSize {
			q.Enqueue(i)
		}

		// Phase 2: Drain most elements to trigger shrink
		for range peakSize - 10 {
			q.Dequeue()
		}

		// Phase 3: Steady-state operations after shrink
		// These should use the shrunk capacity, not peak capacity
		for i := range steadyOps {
			q.Enqueue(1000 + i)
			if i%2 == 0 {
				q.Dequeue()
			}
		}
	})

	// The shrink optimization should keep allocations reasonable.
	// Without shrink, peak capacity would be retained and steady-state
	// ops might not need allocations (but memory footprint would be large).
	// With shrink, we trade a reallocation for lower memory.
	// This test ensures shrink doesn't cause excessive reallocations.
	t.Logf("AllocsPerRun for shrink scenario: %.1f", allocs)

	// Expect reasonable allocations (initial growth + one shrink + possible regrowth)
	// The exact number depends on Go's slice growth strategy
	assert.Less(t, allocs, float64(30),
		"Shrink optimization should not cause excessive allocations")
}

func TestArrayQueue_EnqueueAllEmpty(t *testing.T) {
	t.Parallel()
	// Test EnqueueAll with empty elements
	q := NewArrayQueueFrom(1, 2, 3)
	initialSize := q.Size()
	q.EnqueueAll() // No elements
	assert.Equal(t, initialSize, q.Size(), "Size should remain unchanged after EnqueueAll with empty elements")
}

func TestArrayQueue_IsNotEmpty(t *testing.T) {
	t.Parallel()
	q := NewArrayQueue[int]()
	assert.False(t, q.IsNotEmpty(), "new queue should not be non-empty")
	q.Enqueue(1)
	assert.True(t, q.IsNotEmpty(), "queue with element should be non-empty")
}
