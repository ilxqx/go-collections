package collections

import (
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockFreeList_Basic(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

	assert.True(t, l.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, l.Size(), "Size should be 0 for empty list")

	l.Add(1)
	l.Add(2)
	l.Add(3)

	assert.Equal(t, 3, l.Size(), "Size should be 3 after Add")

	v, ok := l.Get(1)
	require.True(t, ok, "Get should succeed for valid index")
	assert.Equal(t, 2, v, "Sequence should match expected")

	v, ok = l.First()
	require.True(t, ok, "First should succeed on non-empty list")
	assert.Equal(t, 1, v, "Sequence should match expected")

	v, ok = l.Last()
	require.True(t, ok, "Last should succeed on non-empty list")
	assert.Equal(t, 3, v, "Sequence should match expected")
}

func TestLockFreeList_Set(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	old, ok := l.Set(1, 20)
	require.True(t, ok, "Set should succeed for valid index")
	assert.Equal(t, 2, old, "Sequence should match expected")

	v, _ := l.Get(1)
	assert.Equal(t, 20, v, "Sequence should match expected")

	// Out of bounds
	_, ok = l.Set(10, 100)
	require.False(t, ok, "Set should fail for out-of-bounds index")
}

func TestLockFreeList_Insert(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 3)

	require.True(t, l.Insert(1, 2), "Insert should succeed for valid index")

	expected := []int{1, 2, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Insert should place value at index")

	// Insert at beginning
	l.Insert(0, 0)
	v, _ := l.First()
	assert.Equal(t, 0, v, "Sequence should match expected")
}

func TestLockFreeList_Remove(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 2, 4)

	// Remove first occurrence
	assert.True(t, l.Remove(2, intEq), "Remove should succeed for present element")

	expected := []int{1, 3, 2, 4}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect first-match removal")

	// Remove non-existent
	assert.False(t, l.Remove(99, intEq), "Remove should be false for missing element")
}

func TestLockFreeList_RemoveAt(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	v, ok := l.RemoveAt(1)
	require.True(t, ok, "RemoveAt should succeed for valid index")
	assert.Equal(t, 2, v, "Sequence should match expected")

	expected := []int{1, 3}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should reflect removal at index")
}

func TestLockFreeList_RemoveFirstLast(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	v, ok := l.RemoveFirst()
	require.True(t, ok, "RemoveFirst should succeed")
	assert.Equal(t, 1, v, "Sequence should match expected")

	v, ok = l.RemoveLast()
	require.True(t, ok, "RemoveLast should succeed")
	assert.Equal(t, 3, v, "Sequence should match expected")

	assert.Equal(t, 1, l.Size(), "Size should be 1 after removing first and last")
}

func TestLockFreeList_RemoveFunc(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5, 6)

	removed := l.RemoveFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "Removed count should match expected")

	expected := []int{1, 3, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should contain only odds")
}

func TestLockFreeList_RetainFunc(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5, 6)

	removed := l.RetainFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, removed, "Removed count should match expected")

	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "List should contain only evens")
}

func TestLockFreeList_IndexOf(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 2, 4)

	assert.Equal(t, 1, l.IndexOf(2, intEq), "Sequence should match expected")
	assert.Equal(t, 3, l.LastIndexOf(2, intEq), "Sequence should match expected")
	assert.Equal(t, -1, l.IndexOf(99, intEq), "Sequence should match expected")
}

func TestLockFreeList_Contains(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	assert.True(t, l.Contains(2, intEq), "Contains should be true for expected element")
	assert.False(t, l.Contains(99, intEq), "Should not contain element")
}

func TestLockFreeList_Find(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5)

	v, ok := l.Find(func(e int) bool { return e > 3 })
	require.True(t, ok, "Find should succeed for matching predicate")
	assert.Equal(t, 4, v, "Sequence should match expected")

	idx := l.FindIndex(func(e int) bool { return e > 3 })
	assert.Equal(t, 3, idx, "Sequence should match expected")
}

func TestLockFreeList_SubList(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5)

	sub := l.SubList(1, 4)
	expected := []int{2, 3, 4}
	assert.True(t, slices.Equal(sub.ToSlice(), expected), "SubList should return middle slice")

	// Invalid range
	empty := l.SubList(3, 1)
	assert.True(t, empty.IsEmpty(), "IsEmpty should be true")
}

func TestLockFreeList_Reversed(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	var result []int
	for v := range l.Reversed() {
		result = append(result, v)
	}

	expected := []int{3, 2, 1}
	assert.True(t, slices.Equal(result, expected), "Reversed should return descending order")
}

func TestLockFreeList_Clone(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)
	clone := l.Clone()

	l.Add(4)
	assert.Equal(t, 3, clone.Size(), "Clone should not be affected by original modifications")
}

func TestLockFreeList_Filter(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5, 6)

	filtered := l.Filter(func(e int) bool { return e%2 == 0 })
	expected := []int{2, 4, 6}
	assert.True(t, slices.Equal(filtered.ToSlice(), expected), "Filter should keep even values")
}

func TestLockFreeList_Sort(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(3, 1, 4, 1, 5, 9, 2, 6)

	l.Sort(func(a, b int) int { return a - b })
	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "Sort should order ascending")
}

func TestLockFreeList_AnyEvery(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(2, 4, 6, 8)

	assert.True(t, l.Every(func(e int) bool { return e%2 == 0 }), "Every should be true when all match")
	assert.False(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be false when no match")

	l.Add(3)
	assert.True(t, l.Any(func(e int) bool { return e%2 != 0 }), "Any should be true for matching element")
}

func TestLockFreeList_Clear(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)
	l.Clear()

	assert.True(t, l.IsEmpty(), "IsEmpty should be true")
}

func TestLockFreeList_Concurrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := NewLockFreeListOrdered[int]()
		const goroutines = 50
		const opsPerGoroutine = 100

		// Writers
		for i := range goroutines {
			go func(id int) {
				for j := range opsPerGoroutine {
					l.Add(id*opsPerGoroutine + j)
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
		actualSize := l.Size()
		// Size may be approximate due to concurrent modifications
		assert.InDelta(t, expectedSize, actualSize, float64(expectedSize)/10, "Expected size around %d, got %d", expectedSize, actualSize)
	})
}

func TestLockFreeList_ConcurrentRemove(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := NewLockFreeListOrdered[int]()

		// Add elements
		for i := range 1000 {
			l.Add(i)
		}

		// Concurrent removers
		for i := range 10 {
			go func(start int) {
				for j := start; j < 1000; j += 10 {
					l.Remove(j, intEq)
				}
			}(i)
		}

		synctest.Wait()

		assert.Equal(t, 0, l.Size(), "Size should be 0 after concurrent removals")
	})
}

func TestLockFreeList_String(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)
	s := l.String()

	assert.Equal(t, "lockFreeList{1, 2, 3}", s, "String should render type and values")
}

func TestLockFreeList_NonOrdered(t *testing.T) {
	t.Parallel()
	// Test NewLockFreeList with custom equal function (non-ordered)
	eq := func(a, b int) bool { return a == b }
	l := NewLockFreeList(eq)

	assert.True(t, l.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, l.Size(), "Size should be 0 for empty list")

	l.Add(3)
	l.Add(1)
	l.Add(2)

	assert.Equal(t, 3, l.Size(), "Size should be 3 after Add")

	// Non-ordered list - elements are added in order (FIFO)
	// Order in list: 3 -> 1 -> 2
	v, ok := l.First()
	require.True(t, ok, "First should succeed on non-empty list")
	assert.Equal(t, 3, v, "First should return oldest element")

	v, ok = l.Last()
	require.True(t, ok, "Last should succeed on non-empty list")
	assert.Equal(t, 2, v, "Last should return most recently added element")

	// Verify the order by checking all elements
	slice := l.ToSlice()
	assert.Equal(t, []int{3, 1, 2}, slice, "ToSlice should return elements in insertion order")
}

func TestLockFreeList_WithEqualer(t *testing.T) {
	t.Parallel()
	type person struct {
		name string
		age  int
	}

	eq := func(a, b person) bool { return a.name == b.name }
	l := NewLockFreeList(eq)

	l.Add(person{"Alice", 30})
	l.Add(person{"Bob", 25})

	// Should find by name only
	assert.True(t, l.Contains(person{"Alice", 0}, eq), "Contains should match by name only")
}

func BenchmarkLockFreeList_Add(b *testing.B) {
	l := NewLockFreeListOrdered[int]()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Add(i)
			i++
		}
	})
}

func BenchmarkLockFreeList_Read(b *testing.B) {
	l := NewLockFreeListOrdered[int]()
	for i := range 1000 {
		l.Add(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = l.First()
		}
	})
}

func BenchmarkLockFreeList_Contains(b *testing.B) {
	l := NewLockFreeListOrdered[int]()
	for i := range 1000 {
		l.Add(i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			l.Contains(i%1000, intEq)
			i++
		}
	})
}

func TestLockFreeList_AddSeq(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

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

func TestLockFreeList_InsertAll(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 5)

	// Insert in the middle
	require.True(t, l.InsertAll(1, 2, 3, 4), "InsertAll should succeed")
	expected := []int{1, 2, 3, 4, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll should insert in order")

	// Insert at beginning
	require.True(t, l.InsertAll(0, 0), "InsertAll should succeed")
	expected = []int{0, 1, 2, 3, 4, 5}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll at beginning should work")

	// Insert at end
	require.True(t, l.InsertAll(l.Size(), 6, 7), "InsertAll should succeed")
	expected = []int{0, 1, 2, 3, 4, 5, 6, 7}
	assert.True(t, slices.Equal(l.ToSlice(), expected), "InsertAll at end should work")

	// Empty elements
	require.True(t, l.InsertAll(0), "InsertAll should succeed")
	assert.Equal(t, 8, l.Size(), "Empty InsertAll should return true without changes")

	// Out of bounds
	require.False(t, l.InsertAll(100, 99), "InsertAll should fail for invalid index")
	assert.Equal(t, 8, l.Size(), "Out of bounds InsertAll should return false")
}

func TestLockFreeList_ForEach(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5)

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

func TestLockFreeList_PhysicalDelete(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	const n = 1000
	// Build list 0..n-1
	for i := range n {
		l.Add(i)
	}
	// Logically delete all even numbers
	removed := l.RemoveFunc(func(v int) bool { return v%2 == 0 })
	require.Equal(t, n/2, removed, "Removed count should equal half of the elements")

	// Snapshot should only contain odds
	snapBefore := l.ToSlice()
	require.Equal(t, n/2, len(snapBefore), "Snapshot should contain half the elements")
	for _, v := range snapBefore {
		assert.Equal(t, 1, v%2, "Snapshot before physical delete should contain only odd numbers")
	}

	// Physical deletion should unlink logically deleted nodes without changing visible semantics
	lf := l.(*lockFreeList[int])
	lf.PhysicalDelete()

	// Snapshot after physical delete remains the same (odds only)
	snapAfter := l.ToSlice()
	require.Equal(t, n/2, len(snapAfter), "Snapshot should still contain half the elements")
	for _, v := range snapAfter {
		assert.Equal(t, 1, v%2, "Snapshot after physical delete should contain only odd numbers")
	}

	// Idempotency: calling PhysicalDelete again should not change content
	lf.PhysicalDelete()
	snapAgain := l.ToSlice()
	require.Equal(t, n/2, len(snapAgain), "Snapshot should remain unchanged after repeat delete")
	for i := range snapAfter {
		assert.Equal(t, snapAfter[i], snapAgain[i], "Sequence should match expected")
	}
}

func TestLockFreeList_NewLockFreeListOrderedCoverage(t *testing.T) {
	t.Parallel()
	// This test specifically targets NewLockFreeListOrdered to improve coverage
	l := NewLockFreeListOrdered[string]()

	assert.True(t, l.IsEmpty(), "New ordered list should be empty")

	// Test with string type
	l.Add("apple")
	l.Add("banana")
	l.Add("cherry")

	assert.Equal(t, 3, l.Size(), "Size should be 3")
	assert.True(t, l.Contains("banana", func(a, b string) bool { return a == b }), "Should contain banana")

	// Test removal
	assert.True(t, l.Remove("banana", func(a, b string) bool { return a == b }), "Should remove banana")
	assert.Equal(t, 2, l.Size(), "Size should be 2 after removal")
}

func TestLockFreeList_GetNegativeIndex(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	_, ok := l.Get(-1)
	assert.False(t, ok, "Get with negative index should fail")
}

func TestLockFreeList_SetNegativeIndex(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	_, ok := l.Set(-1, 99)
	assert.False(t, ok, "Set with negative index should fail")
}

func TestLockFreeList_FirstEmpty(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

	_, ok := l.First()
	assert.False(t, ok, "First on empty list should fail")
}

func TestLockFreeList_RemoveAtNegativeIndex(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	_, ok := l.RemoveAt(-1)
	assert.False(t, ok, "RemoveAt with negative index should fail")
}

func TestLockFreeList_RemoveAtOutOfBounds(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3)

	_, ok := l.RemoveAt(10)
	assert.False(t, ok, "RemoveAt with out-of-bounds index should fail")
}

func TestLockFreeList_RemoveFirstEmpty(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

	_, ok := l.RemoveFirst()
	assert.False(t, ok, "RemoveFirst on empty list should fail")
}

func TestLockFreeList_ReversedEmpty(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

	count := 0
	for range l.Reversed() {
		count++
	}
	assert.Equal(t, 0, count, "Reversed on empty list should yield no elements")
}

func TestLockFreeList_EveryEmpty(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()

	result := l.Every(func(e int) bool { return e > 0 })
	assert.True(t, result, "Every on empty list should return true")
}

func TestLockFreeList_EveryFalse(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	l.AddAll(1, 2, 3, 4, 5)

	result := l.Every(func(e int) bool { return e > 3 })
	assert.False(t, result, "Every should return false when not all elements match")
}

func TestLockFreeList_IsNotEmpty(t *testing.T) {
	t.Parallel()
	l := NewLockFreeListOrdered[int]()
	assert.False(t, l.IsNotEmpty(), "new list should not be non-empty")
	l.Add(1)
	assert.True(t, l.IsNotEmpty(), "list with element should be non-empty")
}
