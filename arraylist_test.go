package collections

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArrayList_BasicCRUD(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	assert.True(t, l.IsEmpty(), "New list should be empty")
	assert.Equal(t, 0, l.Size(), "Empty list should have size 0")
	l.Add(1)
	l.AddAll(2, 3)
	l.AddSeq(seqOf([]int{4, 5}))
	assert.Equal(t, 5, l.Size(), "Size should equal number of added elements")
	v, ok := l.Get(0)
	require.True(t, ok, "Get(0) should succeed")
	assert.Equal(t, 1, v, "Get(0) should return first element")
	old, ok := l.Set(1, 20)
	require.True(t, ok, "Set should succeed for valid index")
	assert.Equal(t, 2, old, "Set should return old value")
	v, ok = l.First()
	require.True(t, ok, "First should succeed on non-empty list")
	assert.Equal(t, 1, v, "First should return first element")
	v, ok = l.Last()
	require.True(t, ok, "Last should succeed on non-empty list")
	assert.Equal(t, 5, v, "Last should return last element")
	require.True(t, l.Insert(0, 0), "Insert at head should succeed")
	require.True(t, l.InsertAll(2, 7, 8), "InsertAll should succeed")
	removed, ok := l.RemoveAt(2) // remove 7
	require.True(t, ok, "RemoveAt should succeed for valid index")
	assert.Equal(t, 7, removed, "RemoveAt should return removed element")
	assert.True(t, l.Remove(20, eqV[int]), "Remove should succeed for present element")
	rf := l.RemoveFunc(func(x int) bool { return x%2 == 0 })
	assert.Positive(t, rf, "RemoveFunc expected to remove evens")
	l = NewArrayListFrom(1, 2, 3, 4, 5)
	rr := l.RetainFunc(func(x int) bool { return x > 3 })
	assert.Equal(t, 3, rr, "RetainFunc should remove elements <= 3")
	assert.Equal(t, 2, l.Size(), "Size should reflect retained elements")
}

func TestArrayList_SearchViewSortAggregate(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3, 2, 1)
	assert.Equal(t, 1, l.IndexOf(2, eqV[int]), "IndexOf should find first 2 at index 1")
	assert.Equal(t, 3, l.LastIndexOf(2, eqV[int]), "LastIndexOf should find last 2 at index 3")
	assert.True(t, l.Contains(3, eqV[int]), "Contains should be true for present value")
	v, ok := l.Find(func(x int) bool { return x%2 == 0 })
	require.True(t, ok, "Find should locate an even element")
	assert.Equal(t, 0, v%2, "Found element should be even")
	assert.Equal(t, 2, l.FindIndex(func(x int) bool { return x == 3 }), "FindIndex should locate 3 at index 2")
	sub := l.SubList(1, 4).ToSlice()
	assert.True(t, slices.Equal(sub, []int{2, 3, 2}), "SubList=%v", sub)
	// Sort ascending for test
	l.Sort(cmp.Compare)
	got := l.ToSlice()
	assert.True(t, slices.IsSortedFunc(got, cmp.Compare), "Sort ascending failed: %v", got)
	assert.True(t, l.Any(func(x int) bool { return x == 1 }), "Any should be true for value 1")
	assert.True(t, l.Every(func(x int) bool { return x >= 1 }), "Every should be true for all values >= 1")
	// Seq and Reversed consistencies
	seqVals := make([]int, 0, l.Size())
	for v := range l.Seq() {
		seqVals = append(seqVals, v)
	}
	revVals := make([]int, 0, l.Size())
	for v := range l.Reversed() {
		revVals = append(revVals, v)
	}
	revCopy := slices.Clone(revVals)
	slices.Reverse(revCopy)
	assert.Len(t, revVals, len(seqVals), "Seq and Reversed should yield same count")
	assert.True(t, slices.Equal(seqVals, revCopy), "Seq vs Reversed mismatch: %v vs %v", seqVals, revVals)
}

func TestArrayList_InsertAtEnd(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3)
	require.True(t, l.Insert(l.Size(), 4), "Insert should succeed for valid index")
	got := l.ToSlice()
	want := []int{1, 2, 3, 4}
	assert.Len(t, got, len(want), "Slice lengths should match")
	for i, v := range want {
		assert.Equal(t, v, got[i], "Values should match, got=%v want=%v", got, want)
	}
}

func TestArrayList_InsertOutOfBounds(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3)
	assert.False(t, l.Insert(l.Size()+1, 9), "Insert out of bounds should fail")
}

func TestArrayList_NegativeIndex(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	assert.False(t, l.Insert(-1, 1), "Insert negative index should fail")
	_, ok := l.Set(-1, 0)
	require.False(t, ok, "Set should fail for negative index")
	_, ok = l.Get(-1)
	require.False(t, ok, "Get should fail for negative index")
}

func TestArrayList_RemoveFromEmpty(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	_, ok := l.RemoveAt(0)
	require.False(t, ok, "RemoveAt should fail on empty list")
	_, ok = l.RemoveFirst()
	require.False(t, ok, "RemoveFirst on empty should fail")
	_, ok = l.RemoveLast()
	require.False(t, ok, "RemoveLast on empty should fail")
}

func TestArrayList_SubListBoundaries(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3)
	// invalid ranges return empty list
	assert.Equal(t, 0, l.SubList(-1, 2).Size(), "Negative from should return empty list")
	assert.Equal(t, 0, l.SubList(0, 4).Size(), "To beyond size should return empty list")
	assert.Equal(t, 0, l.SubList(3, 2).Size(), "From > to should return empty list")
	// valid boundary [0,0) -> empty
	assert.Equal(t, 0, l.SubList(0, 0).Size(), "[0,0) should be empty")
	// full
	assert.Equal(t, 3, l.SubList(0, 3).Size(), "Full slice size mismatch")
}

func TestArrayList_Clear(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3, 4, 5)
	assert.False(t, l.IsEmpty(), "List should start non-empty")
	l.Clear()
	assert.True(t, l.IsEmpty(), "List should be empty after Clear")
	assert.Equal(t, 0, l.Size(), "Size should be 0 after Clear")
}

func TestArrayList_ForEach(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3, 4, 5)
	sum := 0
	l.ForEach(func(v int) bool {
		sum += v
		return true
	})
	assert.Equal(t, 15, sum, "ForEach should visit all elements")
	// Test early termination
	count := 0
	l.ForEach(func(_ int) bool {
		count++
		return count < 3
	})
	assert.Equal(t, 3, count, "ForEach should stop when callback returns false")
}

func TestArrayList_Clone(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3)
	clone := l.Clone()
	clone.Add(4)
	assert.Equal(t, 3, l.Size(), "Original should be unaffected by clone mutation")
	assert.Equal(t, 4, clone.Size(), "Clone size should reflect added element")
}

func TestArrayList_Filter(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3, 4, 5, 6)
	evens := l.Filter(func(v int) bool { return v%2 == 0 })
	slice := evens.ToSlice()
	assert.Len(t, slice, 3, "Filter should keep three evens")
	for _, v := range slice {
		assert.Equal(t, 0, v%2, "All filtered values should be even")
	}
}

func TestArrayList_NewWithCapacityAndString(t *testing.T) {
	t.Parallel()
	// Test with zero capacity
	l0 := NewArrayListWithCapacity[int](0)
	assert.True(t, l0.IsEmpty(), "New list with zero capacity should be empty")
	l0.Add(1)
	assert.Equal(t, 1, l0.Size(), "Size should be 1 after Add")

	// Test with specific capacity
	l := NewArrayListWithCapacity[int](10)
	assert.True(t, l.IsEmpty(), "New list with capacity should be empty")
	l.Add(1)
	assert.Equal(t, 1, l.Size(), "Size should be 1 after Add")

	// Add more elements to trigger growth
	for i := 2; i <= 20; i++ {
		l.Add(i)
	}
	assert.Equal(t, 20, l.Size(), "Size should be 20 after adding 20 elements")

	// String
	str := l.String()
	assert.Contains(t, str, "arrayList", "String should include type name")
	assert.Contains(t, str, "1", "String should include element value")
}

func TestArrayList_FirstEmpty(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	_, ok := l.First()
	assert.False(t, ok, "First should return false on empty list")
}

func TestArrayList_LastEmpty(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	_, ok := l.Last()
	assert.False(t, ok, "Last should return false on empty list")
}

func TestArrayList_FindNotFound(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3)
	_, ok := l.Find(func(x int) bool { return x > 100 })
	assert.False(t, ok, "Find should return false when no element satisfies predicate")
}

func TestArrayList_FindEmpty(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	_, ok := l.Find(func(x int) bool { return x > 0 })
	assert.False(t, ok, "Find on empty list should return false")
}

func TestArrayList_NewArrayListWithCapacityNegative(t *testing.T) {
	t.Parallel()
	// Test with negative capacity
	l := NewArrayListWithCapacity[int](-5)
	assert.True(t, l.IsEmpty(), "List with negative capacity should be empty")
	l.Add(1)
	assert.Equal(t, 1, l.Size(), "Size should be 1 after Add")
}

func TestArrayList_InsertAllEmpty(t *testing.T) {
	t.Parallel()
	// Test InsertAll with empty elements
	l := NewArrayListFrom(1, 2, 3)
	assert.True(t, l.InsertAll(1), "InsertAll with empty elements should succeed")
	assert.Equal(t, 3, l.Size(), "Size should remain unchanged")
}

func TestArrayList_InsertAllAtEnd(t *testing.T) {
	t.Parallel()
	// Test InsertAll at the end
	l := NewArrayListFrom(1, 2)
	assert.True(t, l.InsertAll(2, 3, 4), "InsertAll at end should succeed")
	assert.Equal(t, 4, l.Size(), "Size should be 4")
	v, _ := l.Get(3)
	assert.Equal(t, 4, v, "Last element should be 4")
}

func TestArrayList_RemoveNotFound(t *testing.T) {
	t.Parallel()
	// Test Remove when element doesn't exist
	l := NewArrayListFrom(1, 2, 3)
	assert.False(t, l.Remove(99, eqV[int]), "Remove should return false for non-existent element")
	assert.Equal(t, 3, l.Size(), "Size should remain unchanged")
}

func TestArrayList_IndexOfNotFound(t *testing.T) {
	t.Parallel()
	// Test IndexOf when element doesn't exist
	l := NewArrayListFrom(1, 2, 3)
	assert.Equal(t, -1, l.IndexOf(99, eqV[int]), "IndexOf should return -1 for non-existent element")
}

func TestArrayList_LastIndexOfNotFound(t *testing.T) {
	t.Parallel()
	// Test LastIndexOf when element doesn't exist
	l := NewArrayListFrom(1, 2, 3)
	assert.Equal(t, -1, l.LastIndexOf(99, eqV[int]), "LastIndexOf should return -1 for non-existent element")
}

func TestArrayList_FindIndexNotFound(t *testing.T) {
	t.Parallel()
	// Test FindIndex when no element satisfies predicate
	l := NewArrayListFrom(1, 2, 3)
	assert.Equal(t, -1, l.FindIndex(func(x int) bool { return x > 100 }), "FindIndex should return -1 when no element satisfies predicate")
}

func TestArrayList_FindIndexEmpty(t *testing.T) {
	t.Parallel()
	// Test FindIndex on empty list
	l := NewArrayList[int]()
	assert.Equal(t, -1, l.FindIndex(func(_ int) bool { return true }), "FindIndex should return -1 on empty list")
}

func TestArrayList_EveryEmpty(t *testing.T) {
	t.Parallel()
	// Test Every on empty list (should return true)
	l := NewArrayList[int]()
	assert.True(t, l.Every(func(x int) bool { return x > 0 }), "Every should return true on empty list")
}

func TestArrayList_EveryFalse(t *testing.T) {
	t.Parallel()
	// Test Every when not all elements satisfy predicate
	l := NewArrayListFrom(1, 2, 3)
	assert.False(t, l.Every(func(x int) bool { return x > 2 }), "Every should return false when not all elements satisfy predicate")
}

func TestArrayList_ReversedEmpty(t *testing.T) {
	t.Parallel()
	// Test Reversed on empty list
	l := NewArrayList[int]()
	count := 0
	for range l.Reversed() {
		count++
	}
	assert.Equal(t, 0, count, "Reversed should yield no elements on empty list")
}

func TestArrayList_ReversedSingle(t *testing.T) {
	t.Parallel()
	// Test Reversed with single element
	l := NewArrayListFrom(42)
	count := 0
	for v := range l.Reversed() {
		count++
		assert.Equal(t, 42, v, "Reversed should yield the single element")
	}
	assert.Equal(t, 1, count, "Reversed should yield exactly one element")
}

func TestArrayList_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	l := NewArrayListFrom(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	// Test early exit in Reversed
	collected := make([]int, 0)
	for v := range l.Reversed() {
		collected = append(collected, v)
		if v <= 6 {
			break // Early exit
		}
	}
	assert.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Reversed should support early exit")
}

func TestArrayList_IsNotEmpty(t *testing.T) {
	t.Parallel()
	l := NewArrayList[int]()
	assert.False(t, l.IsNotEmpty(), "new list should not be non-empty")
	l.Add(1)
	assert.True(t, l.IsNotEmpty(), "list with element should be non-empty")
}
