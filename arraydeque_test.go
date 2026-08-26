package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrayDeque_BasicOps(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	assert.True(t, d.IsEmpty(), "New deque should be empty")
	d.PushBack(2)  // [2]
	d.PushFront(1) // [1,2]
	d.PushBack(3)  // [1,2,3]
	assert.Equal(t, 3, d.Size(), "Size should be 3 after three pushes")
	v, ok := d.PeekFront()
	require.True(t, ok, "PeekFront should succeed on non-empty deque")
	assert.Equal(t, 1, v, "PeekFront should return first element")
	v, ok = d.PeekBack()
	require.True(t, ok, "PeekBack should succeed on non-empty deque")
	assert.Equal(t, 3, v, "PeekBack should return last element")
	// Seq front->back: 1,2,3
	want := 1
	for v := range d.Seq() {
		assert.Equal(t, want, v, "Seq should iterate front-to-back in order")
		want++
	}
	// Reversed: 3,2,1
	want = 3
	for v := range d.Reversed() {
		assert.Equal(t, want, v, "Reversed should iterate back-to-front in order")
		want--
	}
	v, ok = d.PopFront()
	require.True(t, ok, "PopFront should succeed")
	assert.Equal(t, 1, v, "PopFront should return first element")
	v, ok = d.PopBack()
	require.True(t, ok, "PopBack should succeed")
	assert.Equal(t, 3, v, "PopBack should return last element")
	v, ok = d.PopFront()
	require.True(t, ok, "PopFront should succeed for remaining element")
	assert.Equal(t, 2, v, "Remaining element should be 2")
	_, ok = d.PopBack()
	require.False(t, ok, "PopBack on empty should fail")
}

func TestArrayDeque_Growth(t *testing.T) {
	t.Parallel()
	d := NewArrayDequeWithCapacity[int](2)
	// push enough to trigger growth and wrap-around
	for i := range 32 {
		if i%2 == 0 {
			d.PushFront(i)
		} else {
			d.PushBack(i)
		}
	}
	// sanity: size
	assert.Equal(t, 32, d.Size(), "Size should reflect total pushes")
	// pop all to ensure order is consistent with front/back operations
	count := 0
	for !d.IsEmpty() {
		_, ok := d.PopFront()
		require.True(t, ok, "PopFront should succeed until empty")
		count++
	}
	assert.Equal(t, 32, count, "Should have popped all elements")
}

func TestArrayDeque_Clear(t *testing.T) {
	t.Parallel()
	d := NewArrayDequeFrom(1, 2, 3)
	assert.False(t, d.IsEmpty(), "Deque should start non-empty")
	d.Clear()
	assert.True(t, d.IsEmpty(), "Deque should be empty after Clear")
	assert.Equal(t, 0, d.Size(), "Size should be 0 after Clear")
}

func TestArrayDeque_ToSlice(t *testing.T) {
	t.Parallel()
	d := NewArrayDequeFrom(1, 2, 3)
	slice := d.ToSlice()
	assert.Len(t, slice, 3, "ToSlice should return 3 elements")
	assert.Equal(t, 1, slice[0], "Front element should be first in slice")
	assert.Equal(t, 3, slice[2], "Back element should be last in slice")
}

func TestArrayDeque_From(t *testing.T) {
	t.Parallel()
	d := NewArrayDequeFrom(5, 4, 3)
	assert.Equal(t, 3, d.Size(), "Size should equal input length")
	v, _ := d.PopFront()
	assert.Equal(t, 5, v, "PopFront should return first input element")
}

func TestArrayDeque_String(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	d.PushBack(1)
	str := d.String()
	assert.Contains(t, str, "arrayDeque", "String should include type name")
	assert.Contains(t, str, "1", "String should render element value")
}

func TestArrayDeque_PeekFrontEmpty(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	_, ok := d.PeekFront()
	assert.False(t, ok, "PeekFront should return false on empty deque")
}

func TestArrayDeque_PeekBackEmpty(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	_, ok := d.PeekBack()
	assert.False(t, ok, "PeekBack should return false on empty deque")
}

func TestArrayDeque_WithCapacityZero(t *testing.T) {
	t.Parallel()
	// Test with zero capacity
	d := NewArrayDequeWithCapacity[int](0)
	assert.True(t, d.IsEmpty(), "Deque with zero capacity should be empty")
	d.PushBack(1)
	assert.Equal(t, 1, d.Size(), "Size should be 1 after push")
	v, ok := d.PopFront()
	require.True(t, ok, "PopFront should succeed")
	assert.Equal(t, 1, v, "Should return pushed value")
}

func TestArrayDeque_WithCapacityNegative(t *testing.T) {
	t.Parallel()
	// Test with negative capacity
	d := NewArrayDequeWithCapacity[int](-5)
	assert.True(t, d.IsEmpty(), "Deque with negative capacity should be empty")
	d.PushBack(42)
	assert.Equal(t, 1, d.Size(), "Size should be 1 after push")
}

func TestArrayDeque_PopFrontWithEmptyBuffer(t *testing.T) {
	t.Parallel()
	// Test PopFront when buffer is empty (len(d.buf) == 0)
	d := NewArrayDequeWithCapacity[int](0)
	d.PushBack(1)
	d.PushBack(2)
	// Pop all elements
	v, ok := d.PopFront()
	require.True(t, ok, "PopFront should succeed")
	assert.Equal(t, 1, v, "First element should be 1")
	v, ok = d.PopFront()
	require.True(t, ok, "PopFront should succeed")
	assert.Equal(t, 2, v, "Second element should be 2")
	// Now deque is empty
	_, ok = d.PopFront()
	assert.False(t, ok, "PopFront should fail on empty deque")
}

func TestArrayDeque_SeqWithEmpty(t *testing.T) {
	t.Parallel()
	// Test Seq when deque is empty
	d := NewArrayDeque[int]()
	count := 0
	for range d.Seq() {
		count++
	}
	assert.Equal(t, 0, count, "Seq should yield no elements on empty deque")
}

func TestArrayDeque_ReversedWithEmpty(t *testing.T) {
	t.Parallel()
	// Test Reversed when deque is empty
	d := NewArrayDeque[int]()
	count := 0
	for range d.Reversed() {
		count++
	}
	assert.Equal(t, 0, count, "Reversed should yield no elements on empty deque")
}

func TestArrayDeque_SeqWithSingleElement(t *testing.T) {
	t.Parallel()
	// Test Seq with single element
	d := NewArrayDeque[int]()
	d.PushBack(42)
	count := 0
	for v := range d.Seq() {
		count++
		assert.Equal(t, 42, v, "Seq should yield the single element")
	}
	assert.Equal(t, 1, count, "Seq should yield exactly one element")
}

func TestArrayDeque_ReversedWithSingleElement(t *testing.T) {
	t.Parallel()
	// Test Reversed with single element
	d := NewArrayDeque[int]()
	d.PushBack(42)
	count := 0
	for v := range d.Reversed() {
		count++
		assert.Equal(t, 42, v, "Reversed should yield the single element")
	}
	assert.Equal(t, 1, count, "Reversed should yield exactly one element")
}

func TestArrayDeque_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	for i := 1; i <= 10; i++ {
		d.PushBack(i)
	}

	// Test early exit in Seq
	collected := make([]int, 0)
	for v := range d.Seq() {
		collected = append(collected, v)
		if v >= 5 {
			break // Early exit
		}
	}
	assert.Equal(t, []int{1, 2, 3, 4, 5}, collected, "Seq should support early exit")
}

func TestArrayDeque_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	for i := 1; i <= 10; i++ {
		d.PushBack(i)
	}

	// Test early exit in Reversed
	collected := make([]int, 0)
	for v := range d.Reversed() {
		collected = append(collected, v)
		if v <= 6 {
			break // Early exit
		}
	}
	assert.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Reversed should support early exit")
}

func TestArrayDeque_IsNotEmpty(t *testing.T) {
	t.Parallel()
	d := NewArrayDeque[int]()
	assert.False(t, d.IsNotEmpty(), "new deque should not be non-empty")
	d.PushBack(1)
	assert.True(t, d.IsNotEmpty(), "deque with element should be non-empty")
}

// Regression: a deque whose buffer starts exactly full (NewArrayDequeFrom and
// the JSON/Gob decode paths) must keep the circular tail index in [0, len),
// or the first PushBack after a PopFront writes past the buffer.
func TestArrayDeque_FullInitialStatePushBack(t *testing.T) {
	t.Parallel()

	build := map[string]func(t *testing.T) Deque[int]{
		"NewArrayDequeFrom": func(_ *testing.T) Deque[int] {
			return NewArrayDequeFrom(1, 2, 3)
		},
		"UnmarshalJSON": func(t *testing.T) Deque[int] {
			d := NewArrayDeque[int]()
			require.NoError(t, json.Unmarshal([]byte("[1,2,3]"), d), "JSON decode should succeed")
			return d
		},
		"GobDecode": func(t *testing.T) Deque[int] {
			var buf bytes.Buffer
			require.NoError(t, gob.NewEncoder(&buf).Encode(NewArrayDequeFrom(1, 2, 3)), "gob encode should succeed")
			d := NewArrayDeque[int]()
			require.NoError(t, gob.NewDecoder(&buf).Decode(d), "gob decode should succeed")
			return d
		},
	}

	for name, mk := range build {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := mk(t)
			v, ok := d.PopFront()
			require.True(t, ok, "PopFront should succeed on full deque")
			assert.Equal(t, 1, v, "PopFront should return the front element")
			assert.NotPanics(t, func() { d.PushBack(4) }, "PushBack after PopFront should not panic")
			assert.Equal(t, []int{2, 3, 4}, d.ToSlice(), "contents should be front-to-back after wrap")
			d.PushFront(1)
			assert.Equal(t, []int{1, 2, 3, 4}, d.ToSlice(), "PushFront should restore the front")
		})
	}
}
