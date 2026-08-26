package collections

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentedList_Basic(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	assert.True(t, l.IsEmpty(), "New list should be empty")
	assert.Equal(t, 0, l.Size(), "Empty list should have size 0")

	l.Add(1)
	l.Add(2)
	l.Add(3)

	assert.Equal(t, 3, l.Size(), "Size should equal number of added elements")

	v, ok := l.Get(1)
	require.True(t, ok, "Get should succeed for valid index")
	assert.Equal(t, 2, v, "Get should return value at index")

	v, ok = l.First()
	require.True(t, ok, "First should succeed on non-empty list")
	assert.Equal(t, 1, v, "First should return first element")

	v, ok = l.Last()
	require.True(t, ok, "Last should succeed on non-empty list")
	assert.Equal(t, 3, v, "Last should return last element")
}

func TestSegmentedList_Set(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	old, ok := l.Set(1, 20)
	require.True(t, ok, "Set should succeed for valid index")
	assert.Equal(t, 2, old, "Set should return previous value")

	v, _ := l.Get(1)
	assert.Equal(t, 20, v, "Value at index should be updated")

	// Out of bounds
	_, ok = l.Set(10, 100)
	require.False(t, ok, "Set should fail for out-of-bounds index")
}

func TestSegmentedList_Insert(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 3)

	require.True(t, l.Insert(1, 2), "Insert should succeed for valid index")

	expected := []int{1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Insert should put value at index")

	// Insert at beginning
	l.Insert(0, 0)
	v, _ := l.First()
	assert.Equal(t, 0, v, "Insert at head should update first element")

	// Insert at end
	l.Insert(l.Size(), 4)
	v, _ = l.Last()
	assert.Equal(t, 4, v, "Insert at tail should update last element")
}

func TestSegmentedList_InsertAll(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 5)

	require.True(t, l.InsertAll(1, 2, 3, 4), "InsertAll should succeed")

	expected := []int{1, 2, 3, 4, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll should insert in order")
}

func TestSegmentedList_Remove(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 2, 4)

	// Remove first occurrence
	assert.True(t, l.Remove(2, intEq), "Remove should delete first matching element")

	expected := []int{1, 3, 2, 4}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect first-match removal")

	// Remove non-existent
	assert.False(t, l.Remove(99, intEq), "Remove should return false when element not present")
}

func TestSegmentedList_RemoveAt(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	v, ok := l.RemoveAt(1)
	require.True(t, ok, "RemoveAt should succeed for valid index")
	assert.Equal(t, 2, v, "RemoveAt should return removed value")

	expected := []int{1, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect removal at index")
}

func TestSegmentedList_RemoveFirstLast(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	v, ok := l.RemoveFirst()
	require.True(t, ok, "RemoveFirst should succeed")
	assert.Equal(t, 1, v, "RemoveFirst should return first element")

	v, ok = l.RemoveLast()
	require.True(t, ok, "RemoveLast should succeed")
	assert.Equal(t, 3, v, "RemoveLast should return last element")

	assert.Equal(t, 1, l.Size(), "List should contain only middle element")
}

func TestSegmentedList_RemoveFunc(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RemoveFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "RemoveFunc should remove three evens")

	expected := []int{1, 3, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Only odds should remain")
}

func TestSegmentedList_RetainFunc(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RetainFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "RetainFunc should remove three odds")

	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Only evens should remain")
}

func TestSegmentedList_IndexOf(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 2, 4)

	assert.Equal(t, 1, l.IndexOf(2, intEq), "IndexOf should find first 2 at index 1")
	assert.Equal(t, 3, l.LastIndexOf(2, intEq), "LastIndexOf should find last 2 at index 3")
	assert.Equal(t, -1, l.IndexOf(99, intEq), "IndexOf should be -1 for missing value")
}

func TestSegmentedList_Contains(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	assert.True(t, l.Contains(2, intEq), "Contains should be true for present value")
	assert.False(t, l.Contains(99, intEq), "Contains should be false for missing value")
}

func TestSegmentedList_Find(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	v, ok := l.Find(func(e int) bool { return e > 3 })
	require.True(t, ok, "Find should succeed for matching predicate")
	assert.Equal(t, 4, v, "Find should return first matching value")

	idx := l.FindIndex(func(e int) bool { return e > 3 })
	assert.Equal(t, 3, idx, "FindIndex should return first index > 3")
}

func TestSegmentedList_SubList(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	sub := l.SubList(1, 4)
	expected := []int{2, 3, 4}
	assert.True(t, slices.Equal(sub.ToSlice(), expected), "SubList [1,4) should return middle slice")

	// Invalid range
	empty := l.SubList(3, 1)
	assert.True(t, empty.IsEmpty(), "Invalid range should return empty list")
}

func TestSegmentedList_Reversed(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	var result []int
	for v := range l.Reversed() {
		result = append(result, v)
	}

	expected := []int{3, 2, 1}
	assert.True(t, slices.Equal(result, expected), "Reversed should iterate from tail to head")
}

func TestSegmentedList_Clone(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)
	clone := l.Clone()

	l.Add(4)
	assert.Equal(t, 3, clone.Size(), "Clone should not be affected by original modifications")
}

func TestSegmentedList_Filter(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5, 6)

	filtered := l.Filter(func(e int) bool { return e%2 == 0 })
	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(filtered.ToSlice(), expected), "Filter should keep only evens")
}

func TestSegmentedList_Sort(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(3, 1, 4, 1, 5, 9, 2, 6)

	l.Sort(func(a, b int) int { return a - b })
	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Sort should produce ascending order")
}

func TestSegmentedList_AnyEvery(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(2, 4, 6, 8)

	assert.True(t, l.Every(func(e int) bool { return e%2 == 0 }), "Every should be true for evens")
	assert.False(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be false when none match")

	l.Add(3)
	assert.True(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be true after adding an odd")
}

func TestSegmentedList_Clear(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)
	l.Clear()

	assert.True(t, l.IsEmpty(), "List should be empty after Clear")
}

func TestSegmentedList_AddAll(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()
	l.AddAll(1, 2, 3)

	expected := []int{1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "AddAll should append all values")
}

func TestSegmentedList_AddSeq(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// Create a sequence
	seq := func(yield func(int) bool) {
		for i := 1; i <= 3; i++ {
			if !yield(i) {
				return
			}
		}
	}

	l.AddSeq(seq)

	expected := []int{1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "AddSeq should add all values from sequence")

	// Add to non-empty list
	l.AddSeq(seq)
	expected = []int{1, 2, 3, 1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "AddSeq should append to existing elements")

	// Empty sequence
	emptySeq := func(_ func(int) bool) {}
	l.AddSeq(emptySeq)
	assert.Equal(t, 6, l.Size(), "Empty sequence should not add elements")
}

func TestSegmentedList_ForEach(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	var sum int
	var count int
	l.ForEach(func(e int) bool {
		sum += e
		count++
		return true // continue iteration
	})

	assert.Equal(t, 15, sum, "ForEach should sum all elements")
	assert.Equal(t, 5, count, "ForEach should visit all elements")

	// Test early termination
	var earlyCount int
	l.ForEach(func(e int) bool {
		earlyCount++
		return e < 3 // stop when reaching 3
	})
	assert.Equal(t, 3, earlyCount, "ForEach should stop when predicate returns false")
}

func TestSegmentedList_Concurrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := NewSegmentedList[int]()
		const goroutines = 100
		const opsPerGoroutine = 100

		// Writers
		for i := range goroutines {
			go func(base int) {
				for j := range opsPerGoroutine {
					l.Add(base*opsPerGoroutine + j)
				}
			}(i)
		}

		// Readers
		for range goroutines {
			go func() {
				for range opsPerGoroutine {
					_ = l.Size()
					_ = l.ToSlice()
					_, _ = l.First()
					_, _ = l.Last()
				}
			}()
		}

		synctest.Wait()

		expectedSize := goroutines * opsPerGoroutine
		assert.Equal(t, expectedSize, l.Size(), "Concurrent add/read should reach expected size")
	})
}

func TestSegmentedList_GetScenarios(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	// Get valid indices
	for i := range 5 {
		v, ok := l.Get(i)
		require.True(t, ok, "Get should succeed for valid index %d", i)
		require.Equal(t, i+1, v, "Get should return correct value at index %d", i)
	}

	// Get out of bounds
	_, ok := l.Get(5)
	require.False(t, ok, "Get should fail for index >= size")

	_, ok = l.Get(100)
	require.False(t, ok, "Get should fail for far out of bounds index")

	// Get negative index
	_, ok = l.Get(-1)
	require.False(t, ok, "Get should fail for negative index")

	// Get on empty list
	empty := NewSegmentedList[int]()
	_, ok = empty.Get(0)
	require.False(t, ok, "Get should fail on empty list")
}

func TestSegmentedList_CustomSegments(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListWithSegments[int](4)
	for i := range 100 {
		l.Add(i)
	}

	assert.Equal(t, 100, l.Size(), "Size should be 100 after adds")

	// Verify Get works across segments
	for i := range 100 {
		v, ok := l.Get(i)
		require.True(t, ok, "Get should succeed for valid index")
		assert.Equal(t, i, v, "Get should return correct value across segments")
	}
}

func TestSegmentedList_String(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)
	s := l.String()

	assert.Equal(t, "segmentedList{1, 2, 3}", s, "String should render type and values")
}

func BenchmarkSegmentedList_Read(b *testing.B) {
	l := NewSegmentedList[int]()
	for i := range 1000 {
		l.Add(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = l.Get(500)
		}
	})
}

func BenchmarkSegmentedList_Write(b *testing.B) {
	l := NewSegmentedList[int]()

	b.ResetTimer()
	for b.Loop() {
		l.Add(0)
	}
}

func BenchmarkSegmentedList_ConcurrentReadWrite(b *testing.B) {
	l := NewSegmentedList[int]()
	for i := range 1000 {
		l.Add(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				_, _ = l.Get(i % 1000)
			} else {
				l.Add(i)
			}
			i++
		}
	})
}

func TestSegmentedList_RemoveAtOutOfBounds(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	// Valid removal
	v, ok := l.RemoveAt(1)
	require.True(t, ok, "RemoveAt should succeed for valid index")
	require.Equal(t, 2, v, "RemoveAt should return removed value")

	// Out of bounds index
	_, ok = l.RemoveAt(10)
	require.False(t, ok, "RemoveAt should fail for index >= size")

	// Negative index
	_, ok = l.RemoveAt(-1)
	require.False(t, ok, "RemoveAt should fail for negative index")

	// Empty list
	empty := NewSegmentedList[int]()
	_, ok = empty.RemoveAt(0)
	require.False(t, ok, "RemoveAt should fail on empty list")
}

func TestSegmentedList_LastEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// Last on empty list
	_, ok := l.Last()
	require.False(t, ok, "Last should fail on empty list")

	// Add element and test
	l.Add(42)
	v, ok := l.Last()
	require.True(t, ok, "Last should succeed on non-empty list")
	require.Equal(t, 42, v, "Last should return the element")

	// Remove and verify empty again
	l.RemoveLast()
	_, ok = l.Last()
	require.False(t, ok, "Last should fail after removing last element")
}

func TestSegmentedList_InsertAllEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3)

	// InsertAll with empty elements (returns true, no-op)
	ok := l.InsertAll(1)
	require.True(t, ok, "InsertAll should return true for empty elements (no-op)")
	require.Equal(t, 3, l.Size(), "Size should remain unchanged")

	// InsertAll at out of bounds index
	ok = l.InsertAll(10, 4, 5)
	require.False(t, ok, "InsertAll should fail for out-of-bounds index")

	// InsertAll at valid index
	ok = l.InsertAll(1, 10, 20)
	require.True(t, ok, "InsertAll should succeed for valid index")
	expected := []int{1, 10, 20, 2, 3}
	require.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll should insert elements at index")
}

func TestSegmentedList_RemoveLastEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// RemoveLast on empty list
	_, ok := l.RemoveLast()
	require.False(t, ok, "RemoveLast should fail on empty list")

	// Add and remove
	l.Add(1)
	v, ok := l.RemoveLast()
	require.True(t, ok, "RemoveLast should succeed")
	require.Equal(t, 1, v, "RemoveLast should return the element")
	require.True(t, l.IsEmpty(), "List should be empty after removing last element")
}

func TestSegmentedList_RemoveFirstEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// RemoveFirst on empty list
	_, ok := l.RemoveFirst()
	require.False(t, ok, "RemoveFirst should fail on empty list")

	// Add and remove
	l.Add(1)
	v, ok := l.RemoveFirst()
	require.True(t, ok, "RemoveFirst should succeed")
	require.Equal(t, 1, v, "RemoveFirst should return the element")
	require.True(t, l.IsEmpty(), "List should be empty after removing first element")
}

func TestSegmentedList_ReversedEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// Reversed on empty list
	count := 0
	for range l.Reversed() {
		count++
	}
	require.Equal(t, 0, count, "Reversed should not yield elements for empty list")

	// Single element
	l.Add(42)
	var result []int
	for v := range l.Reversed() {
		result = append(result, v)
	}
	require.Equal(t, []int{42}, result, "Reversed should work for single element")
}

func TestSegmentedList_EveryEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()

	// Every on empty list (vacuous truth)
	result := l.Every(func(e int) bool { return e > 0 })
	require.True(t, result, "Every should return true for empty list")

	// Every with failing predicate
	l.AddAll(1, 2, 3, 4, 5)
	result = l.Every(func(e int) bool { return e < 3 })
	require.False(t, result, "Every should return false when some elements don't match")
}

func TestSegmentedList_FindNotFound(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	// Find with no match
	_, ok := l.Find(func(e int) bool { return e > 10 })
	require.False(t, ok, "Find should fail when no element matches")

	// FindIndex with no match
	idx := l.FindIndex(func(e int) bool { return e > 10 })
	require.Equal(t, -1, idx, "FindIndex should return -1 when no element matches")
}

func TestSegmentedList_LastIndexOfNotFound(t *testing.T) {
	t.Parallel()
	l := NewSegmentedListFrom(1, 2, 3, 4, 5)

	// LastIndexOf with no match
	idx := l.LastIndexOf(10, intEq)
	require.Equal(t, -1, idx, "LastIndexOf should return -1 for missing element")

	// LastIndexOf with match
	l.AddAll(2, 3, 2)
	idx = l.LastIndexOf(2, intEq)
	require.Equal(t, 7, idx, "LastIndexOf should return last occurrence index")
}

func TestSegmentedList_NewSegmentedListWithSegments(t *testing.T) {
	t.Parallel()

	// Normal segment size
	l := NewSegmentedListWithSegments[int](4)
	require.NotNil(t, l, "NewSegmentedListWithSegments should create list")
	require.True(t, l.IsEmpty(), "New list should be empty")

	// Very small segment size
	l2 := NewSegmentedListWithSegments[int](1)
	l2.AddAll(1, 2, 3, 4, 5)
	require.Equal(t, 5, l2.Size(), "Small segment size should work")

	// Zero or negative segment size should use default
	l3 := NewSegmentedListWithSegments[int](0)
	require.NotNil(t, l3, "Zero segment size should use default")
	l3.Add(1)
	require.Equal(t, 1, l3.Size(), "List with default segment should work")
}

func TestSegmentedList_IsNotEmpty(t *testing.T) {
	t.Parallel()
	l := NewSegmentedList[int]()
	assert.False(t, l.IsNotEmpty(), "new list should not be non-empty")
	l.Add(1)
	assert.True(t, l.IsNotEmpty(), "list with element should be non-empty")
}
