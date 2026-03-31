package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrayStack_BasicAndOrder(t *testing.T) {
	t.Parallel()
	s := NewArrayStack[int]()
	assert.True(t, s.IsEmpty(), "New stack should be empty")
	s.Push(1)
	s.PushAll(2, 3)
	assert.Equal(t, 3, s.Size(), "Size should be 3 after Push and PushAll")
	v, ok := s.Peek()
	require.True(t, ok, "Peek should succeed on non-empty stack")
	assert.Equal(t, 3, v, "Peek should return top element")
	// Seq should yield top->bottom: 3,2,1
	i := 3
	for v := range s.Seq() {
		assert.Equal(t, i, v, "Seq should iterate top-to-bottom")
		i--
	}
	// ToSlice bottom->top
	ts := s.ToSlice()
	assert.Equal(t, 1, ts[0], "ToSlice[0] should be bottom element")
	assert.Equal(t, 3, ts[2], "ToSlice[last] should be top element")
	v, ok = s.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 3, v, "First Pop should return top element")
	v, ok = s.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 2, v, "Second Pop should return next element")
	v, ok = s.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 1, v, "Third Pop should return bottom element")
	_, ok = s.Pop()
	require.False(t, ok, "Pop on empty should fail")
}

func TestArrayStack_Clear(t *testing.T) {
	t.Parallel()
	s := NewArrayStackFrom(1, 2, 3)
	assert.False(t, s.IsEmpty(), "Stack should start non-empty")
	s.Clear()
	assert.True(t, s.IsEmpty(), "Stack should be empty after Clear")
	assert.Equal(t, 0, s.Size(), "Size should be 0 after Clear")
}

func TestArrayStack_WithCapacity(t *testing.T) {
	t.Parallel()
	s := NewArrayStackWithCapacity[int](10)
	assert.True(t, s.IsEmpty(), "New stack with capacity should be empty")
	s.Push(1)
	assert.Equal(t, 1, s.Size(), "Size should be 1 after Push")
}

func TestArrayStack_From(t *testing.T) {
	t.Parallel()
	s := NewArrayStackFrom(1, 2, 3)
	assert.Equal(t, 3, s.Size(), "Size should equal number of inputs")
	v, _ := s.Pop()
	assert.Equal(t, 3, v, "Pop should return last input")
}

func TestArrayStack_String(t *testing.T) {
	t.Parallel()
	s := NewArrayStack[int]()
	s.Push(1)
	str := s.String()
	assert.Contains(t, str, "arrayStack", "String should include type name")
	assert.Contains(t, str, "1", "String should render element value")
}

func TestArrayStack_PeekEmpty(t *testing.T) {
	t.Parallel()
	s := NewArrayStack[int]()
	_, ok := s.Peek()
	assert.False(t, ok, "Peek should return false on empty stack")
}

func TestArrayStack_WithCapacityZero(t *testing.T) {
	t.Parallel()
	s := NewArrayStackWithCapacity[int](0)
	assert.True(t, s.IsEmpty(), "Stack with zero capacity should be empty")
	s.Push(1)
	assert.Equal(t, 1, s.Size(), "Size should be 1 after Push")
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed")
	assert.Equal(t, 1, v, "Should return pushed value")
}

func TestArrayStack_WithCapacityNegative(t *testing.T) {
	t.Parallel()
	s := NewArrayStackWithCapacity[int](-5)
	assert.True(t, s.IsEmpty(), "Stack with negative capacity should be empty")
	s.Push(42)
	assert.Equal(t, 1, s.Size(), "Size should be 1 after Push")
}

func TestArrayStack_SeqEmpty(t *testing.T) {
	t.Parallel()
	// Test Seq on empty stack
	s := NewArrayStack[int]()
	count := 0
	for range s.Seq() {
		count++
	}
	assert.Equal(t, 0, count, "Seq should yield no elements on empty stack")
}

func TestArrayStack_SeqSingle(t *testing.T) {
	t.Parallel()
	// Test Seq with single element
	s := NewArrayStack[int]()
	s.Push(42)
	count := 0
	for v := range s.Seq() {
		count++
		assert.Equal(t, 42, v, "Seq should yield the single element")
	}
	assert.Equal(t, 1, count, "Seq should yield exactly one element")
}

func TestArrayStack_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewArrayStackFrom(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	// Test early exit in Seq (iterates top-to-bottom: 10,9,8...)
	collected := make([]int, 0)
	for v := range s.Seq() {
		collected = append(collected, v)
		if v <= 6 {
			break // Early exit
		}
	}
	assert.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Seq should support early exit")
}

func TestArrayStack_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewArrayStack[int]()
	assert.False(t, s.IsNotEmpty(), "new stack should not be non-empty")
	s.Push(1)
	assert.True(t, s.IsNotEmpty(), "stack with element should be non-empty")
}
