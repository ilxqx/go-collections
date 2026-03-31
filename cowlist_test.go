package collections

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intEq(a, b int) bool { return a == b }

func TestCOWList_Basic(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()

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

func TestCOWList_Set(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)

	old, ok := l.Set(1, 20)
	require.True(t, ok, "Set should succeed for valid index")
	assert.Equal(t, 2, old, "Set should return previous value")

	v, _ := l.Get(1)
	assert.Equal(t, 20, v, "Value at index should be updated")

	// Out of bounds
	_, ok = l.Set(10, 100)
	require.False(t, ok, "Set out of bounds should return false")
}

func TestCOWList_Insert(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 3)

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

func TestCOWList_InsertAll(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 5)

	require.True(t, l.InsertAll(1, 2, 3, 4), "InsertAll should succeed")

	expected := []int{1, 2, 3, 4, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll should insert in order")
}

func TestCOWList_Remove(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 2, 4)

	// Remove first occurrence
	assert.True(t, l.Remove(2, intEq), "Remove should delete first matching element")

	expected := []int{1, 3, 2, 4}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect first-match removal")

	// Remove non-existent
	assert.False(t, l.Remove(99, intEq), "Remove should return false when element not present")
}

func TestCOWList_RemoveAt(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)

	v, ok := l.RemoveAt(1)
	require.True(t, ok, "RemoveAt should succeed for valid index")
	assert.Equal(t, 2, v, "RemoveAt should return removed value")

	expected := []int{1, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect removal at index")
}

func TestCOWList_RemoveFirstLast(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)

	v, ok := l.RemoveFirst()
	require.True(t, ok, "RemoveFirst should succeed")
	assert.Equal(t, 1, v, "RemoveFirst should return first element")

	v, ok = l.RemoveLast()
	require.True(t, ok, "RemoveLast should succeed")
	assert.Equal(t, 3, v, "RemoveLast should return last element")

	assert.Equal(t, 1, l.Size(), "List should contain only middle element")
}

func TestCOWList_RemoveFunc(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RemoveFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "RemoveFunc should remove three evens")

	expected := []int{1, 3, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Only odds should remain")
}

func TestCOWList_RetainFunc(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RetainFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "RetainFunc should remove three odds")

	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Only evens should remain")
}

func TestCOWList_IndexOf(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 2, 4)

	assert.Equal(t, 1, l.IndexOf(2, intEq), "IndexOf should find first 2 at index 1")
	assert.Equal(t, 3, l.LastIndexOf(2, intEq), "LastIndexOf should find last 2 at index 3")
	assert.Equal(t, -1, l.IndexOf(99, intEq), "IndexOf should be -1 for missing value")
}

func TestCOWList_Contains(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)

	assert.True(t, l.Contains(2, intEq), "Contains should be true for present value")
	assert.False(t, l.Contains(99, intEq), "Contains should be false for missing value")
}

func TestCOWList_Find(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5)

	v, ok := l.Find(func(e int) bool { return e > 3 })
	require.True(t, ok, "Find should succeed for matching predicate")
	assert.Equal(t, 4, v, "Find should return first matching value")

	idx := l.FindIndex(func(e int) bool { return e > 3 })
	assert.Equal(t, 3, idx, "FindIndex should return first index > 3")
}

func TestCOWList_SubList(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5)

	sub := l.SubList(1, 4)
	expected := []int{2, 3, 4}
	assert.True(t, slices.Equal(sub.ToSlice(), expected), "SubList [1,4) should return middle slice")

	// Invalid range
	empty := l.SubList(3, 1)
	assert.True(t, empty.IsEmpty(), "Invalid range should return empty list")
}

func TestCOWList_Reversed(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)

	var result []int
	for v := range l.Reversed() {
		result = append(result, v)
	}

	expected := []int{3, 2, 1}
	assert.True(t, slices.Equal(result, expected), "Reversed should iterate from tail to head")
}

func TestCOWList_Clone(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	clone := l.Clone()

	l.Add(4)
	assert.Equal(t, 3, clone.Size(), "Clone should not be affected by original modifications")
}

func TestCOWList_Filter(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5, 6)

	filtered := l.Filter(func(e int) bool { return e%2 == 0 })
	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(filtered.ToSlice(), expected), "Filter should keep only evens")
}

func TestCOWList_Sort(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(3, 1, 4, 1, 5, 9, 2, 6)

	l.Sort(func(a, b int) int { return a - b })
	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Sort should produce ascending order")
}

func TestCOWList_AnyEvery(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(2, 4, 6, 8)

	assert.True(t, l.Every(func(e int) bool { return e%2 == 0 }), "Every should be true for evens")
	assert.False(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be false when none match")

	l.Add(3)
	assert.True(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be true after adding an odd")
}

func TestCOWList_Clear(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	l.Clear()

	assert.True(t, l.IsEmpty(), "List should be empty after Clear")
}

func TestCOWList_AddAll(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	l.AddAll(1, 2, 3)

	expected := []int{1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "AddAll should append all values")
}

func TestCOWList_AddSeq(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()

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
	emptySeq := func(yield func(int) bool) {}
	l.AddSeq(emptySeq)
	assert.Equal(t, 6, l.Size(), "Empty sequence should not add elements")
}

func TestCOWList_ForEach(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5)

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

func TestCOWList_Concurrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := NewCOWList[int]()
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

func TestCOWList_SnapshotConsistency(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5)

	// Take a snapshot via iteration
	var snapshot []int
	for v := range l.Seq() {
		snapshot = append(snapshot, v)
		if v == 2 {
			// Modify during iteration
			l.Add(6)
		}
	}

	// Snapshot should be consistent (not see the modification)
	expected := []int{1, 2, 3, 4, 5}
	assert.True(t, slices.Equal(snapshot, expected), "Snapshot should be %v, got %v", expected, snapshot)

	// But the list should have the new element
	assert.Equal(t, 6, l.Size(), "Size should be 6 after Add during iteration")
}

func TestCOWList_String(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	s := l.String()

	assert.Equal(t, "cowList{1, 2, 3}", s, "String should render type and values")
}

func TestCOWList_GetEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	_, ok := l.Get(0)
	assert.False(t, ok, "Get should return false on empty list")

	l = NewCOWListFrom(1, 2, 3)
	_, ok = l.Get(-1)
	assert.False(t, ok, "Get with negative index should return false")
	_, ok = l.Get(100)
	assert.False(t, ok, "Get with out of bounds index should return false")
}

func TestCOWList_FirstEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	_, ok := l.First()
	assert.False(t, ok, "First should return false on empty list")
}

func TestCOWList_LastEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	_, ok := l.Last()
	assert.False(t, ok, "Last should return false on empty list")
}

func TestCOWList_FindNotFound(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	_, ok := l.Find(func(e int) bool { return e > 100 })
	assert.False(t, ok, "Find should return false when no element satisfies predicate")

	// Empty list
	l2 := NewCOWList[int]()
	_, ok = l2.Find(func(e int) bool { return e > 0 })
	assert.False(t, ok, "Find on empty list should return false")
}

func TestCOWList_AddAllEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	l.AddAll() // Empty elements
	assert.Equal(t, 3, l.Size(), "AddAll with empty elements should not change size")
}

func TestCOWList_InsertAllEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	assert.True(t, l.InsertAll(1), "InsertAll with empty elements should return true")
	assert.Equal(t, 3, l.Size(), "InsertAll with empty elements should not change size")
}

func TestCOWList_InsertAllOutOfBounds(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	assert.False(t, l.InsertAll(-1, 4, 5), "InsertAll with negative index should return false")
	assert.False(t, l.InsertAll(100, 4, 5), "InsertAll with out of bounds index should return false")
	assert.Equal(t, 3, l.Size(), "Failed InsertAll should not change size")
}

func TestCOWList_RemoveAtEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	_, ok := l.RemoveAt(0)
	assert.False(t, ok, "RemoveAt on empty list should return false")
}

func TestCOWList_RemoveLastEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	_, ok := l.RemoveLast()
	assert.False(t, ok, "RemoveLast on empty list should return false")
}

func TestCOWList_LastIndexOfNotFound(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	assert.Equal(t, -1, l.LastIndexOf(99, intEq), "LastIndexOf should return -1 when not found")
}

func TestCOWList_FindIndexNotFound(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3)
	assert.Equal(t, -1, l.FindIndex(func(e int) bool { return e > 100 }), "FindIndex should return -1 when not found")
}

func TestCOWList_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	l := NewCOWListFrom(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

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

func BenchmarkCOWList_Read(b *testing.B) {
	l := NewCOWList[int]()
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

func BenchmarkCOWList_Write(b *testing.B) {
	l := NewCOWList[int]()

	b.ResetTimer()
	for b.Loop() {
		l.Add(0)
	}
}

func TestCowList_IsNotEmpty(t *testing.T) {
	t.Parallel()
	l := NewCOWList[int]()
	assert.False(t, l.IsNotEmpty(), "new list should not be non-empty")
	l.Add(1)
	assert.True(t, l.IsNotEmpty(), "list with element should be non-empty")
}
