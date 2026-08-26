package collections

import (
	"runtime"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentSkipSet_BasicOrdered(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(3, 1, 2, 2)
	got := make([]int, 0, s.Size())
	for v := range s.Seq() {
		got = append(got, v)
	}
	assert.True(t, slices.Equal(got, []int{1, 2, 3}), "Seq=%v", got)
	rev := make([]int, 0, 3)
	for v := range s.Reversed() {
		rev = append(rev, v)
	}
	assert.True(t, slices.Equal(rev, []int{3, 2, 1}), "Reversed=%v", rev)
	v, ok := s.First()
	require.True(t, ok, "First should succeed on non-empty set")
	assert.Equal(t, 1, v, "First should return smallest element")
	v, ok = s.Last()
	require.True(t, ok, "Last should succeed on non-empty set")
	assert.Equal(t, 3, v, "Last should return largest element")
}

func TestConcurrentSkipSet_ConcurrentAddIfAbsent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := NewConcurrentSkipSet[int]()
		n := 1000
		workers := runtime.GOMAXPROCS(0) * 2
		for range workers {
			go func() {
				for i := range n {
					s.AddIfAbsent(i)
				}
			}()
		}
		synctest.Wait()
		for _, k := range []int{0, n / 2, n - 1} {
			assert.Truef(t, s.Contains(k), "Missing key %d", k)
		}
	})
}

func TestConcurrentSkipSet_NavigationAndExtremes(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(10, 20, 30, 40)
	v, ok := s.First()
	require.True(t, ok, "First should succeed on non-empty set")
	assert.Equal(t, 10, v, "First should return smallest element")
	v, ok = s.Last()
	require.True(t, ok, "Last should succeed on non-empty set")
	assert.Equal(t, 40, v, "Last should return largest element")
	v, ok = s.Min()
	require.True(t, ok, "Min should succeed on non-empty set")
	assert.Equal(t, 10, v, "Min should return smallest element")
	v, ok = s.Max()
	require.True(t, ok, "Max should succeed on non-empty set")
	assert.Equal(t, 40, v, "Max should return largest element")
	v, ok = s.Floor(25)
	require.True(t, ok, "Floor should find 20 for key 25")
	require.Equal(t, 20, v, "Returned value should match expected")
	v, ok = s.Ceiling(25)
	require.True(t, ok, "Ceiling should find 30 for key 25")
	require.Equal(t, 30, v, "Returned value should match expected")
	_, ok = s.Lower(10)
	require.False(t, ok, "Lower(10) should be none")
	_, ok = s.Higher(40)
	require.False(t, ok, "Higher(40) should be none")
}

func TestConcurrentSkipSet_PopFirstLast(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(10, 20, 30, 40)
	v, ok := s.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 10, v, "Returned value should match expected")
	v, ok = s.PopLast()
	require.True(t, ok, "PopLast should succeed")
	require.Equal(t, 40, v, "Returned value should match expected")
	require.Equal(t, 2, s.Size(), "Size should be 2 after PopFirst and PopLast")
}

func TestConcurrentSkipSet_RangeAndRank(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5)
	r := make([]int, 0)
	s.Range(2, 4, func(e int) bool {
		r = append(r, e)
		return true
	})
	require.True(t, slices.Equal(r, []int{2, 3, 4}), "Range=%v", r)
	r2 := make([]int, 0)
	for v := range s.RangeSeq(3, 5) {
		r2 = append(r2, v)
	}
	require.True(t, slices.Equal(r2, []int{3, 4, 5}), "RangeSeq=%v", r2)
	require.Equal(t, 2, s.Rank(3), "Rank should match expected")
	v, ok := s.GetByRank(0)
	require.True(t, ok, "GetByRank should succeed for valid rank")
	require.Equal(t, 1, v, "GetByRank should return smallest element at rank 0")
}

func TestConcurrentSkipSet_Views(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5)
	head := s.HeadSet(3, false).ToSlice()
	tail := s.TailSet(3, true).ToSlice()
	sub := s.SubSet(2, 4).ToSlice()
	slices.Sort(head)
	slices.Sort(tail)
	slices.Sort(sub)
	require.True(t, slices.Equal(head, []int{1, 2}), "HeadSet=%v", head)
	require.True(t, slices.Equal(tail, []int{3, 4, 5}), "TailSet=%v", tail)
	require.True(t, slices.Equal(sub, []int{2, 3, 4}), "SubSet=%v", sub)
}

func TestConcurrentSkipSet_SetOpsBasic(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	assert.True(t, s.IsEmpty(), "New set should be empty")
	assert.Equal(t, 0, s.Size(), "Empty set size should be 0")
	assert.True(t, s.Add(1), "Add should succeed for new element")
	assert.False(t, s.Add(1), "Add duplicate should be false")
	assert.True(t, s.Contains(1), "Contains should be true for present element")
	added := s.AddAll(2, 3, 3)
	assert.GreaterOrEqual(t, added, 2, "Value should be at least expected")
	assert.True(t, s.Remove(2), "Remove should succeed for present element")
	assert.False(t, s.Remove(99), "Remove should be false for missing element")
}

func TestConcurrentSkipSet_Algebra(t *testing.T) {
	t.Parallel()
	a := NewConcurrentSkipSetFrom(1, 2, 3, 4)
	b := NewConcurrentSkipSetFrom(3, 4, 5)
	u := a.Union(b).ToSlice()
	i := a.Intersection(b).ToSlice()
	d := a.Difference(b).ToSlice()
	sd := a.SymmetricDifference(b).ToSlice()
	assert.Truef(t, slices.Contains(u, 1) && slices.Contains(u, 5), "Union unexpected: %v", u)
	assert.Len(t, i, 2, "Intersection unexpected: %v", i)
	assert.True(t, slices.Contains(i, 3) && slices.Contains(i, 4), "Contains should be true for expected element")
	assert.Len(t, d, 2, "Difference unexpected: %v", d)
	assert.True(t, slices.Contains(d, 1) && slices.Contains(d, 2), "Contains should be true for expected element")
	assert.Len(t, sd, 3, "SymmetricDifference unexpected: %v", sd)
	assert.True(t, slices.Contains(sd, 1) && slices.Contains(sd, 2) && slices.Contains(sd, 5), "Contains should be true for expected element")
}

func TestConcurrentSkipSet_AscendDescend(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(5, 1, 3, 2, 4)
	asc := make([]int, 0)
	s.Ascend(func(e int) bool {
		asc = append(asc, e)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 4, 5}, asc, "Ascend should iterate ascending order")
	desc := make([]int, 0)
	s.Descend(func(e int) bool {
		desc = append(desc, e)
		return true
	})
	assert.Equal(t, []int{5, 4, 3, 2, 1}, desc, "Descend should iterate descending order")
	af := make([]int, 0)
	s.AscendFrom(3, func(e int) bool {
		af = append(af, e)
		return true
	})
	assert.Equal(t, []int{3, 4, 5}, af, "AscendFrom should iterate from pivot upwards")
	df := make([]int, 0)
	s.DescendFrom(3, func(e int) bool {
		df = append(df, e)
		return true
	})
	assert.Equal(t, []int{3, 2, 1}, df, "DescendFrom should iterate from pivot downwards")
}

func TestConcurrentSkipSet_CloneSorted(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3)
	clone := s.CloneSorted()
	clone.Remove(1)
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	assert.Equal(t, 3, s.Size(), "Size should be 3 after AddSeq")
	assert.Equal(t, 2, clone.Size(), "Clone size should match expected")
}

func TestConcurrentSkipSet_FilterFindAnyEvery(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5, 6)
	filtered := s.Filter(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, filtered.Size(), "Filter should keep three even values")
	v, ok := filtered.Find(func(e int) bool { return e > 4 })
	require.True(t, ok, "Find should locate a value > 4")
	assert.Equal(t, 6, v, "Sequence should match expected")
	assert.True(t, s.Any(func(e int) bool { return e == 3 }), "Any should be true for matching element")
	assert.True(t, s.Every(func(e int) bool { return e >= 1 }), "Every should be true when all match")
	assert.False(t, s.Every(func(e int) bool { return e > 3 }), "Every should be false when any fail")
}

func TestConcurrentSkipSet_RemoveFuncRetainFunc(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(1, 2, 3, 4, 5, 6)
	count := s.RemoveFunc(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 3, count, "Count should match expected")
	assert.Equal(t, 3, s.Size(), "Size should be 3 after RemoveFunc")
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	assert.False(t, s.Contains(2), "Should not contain element")
	s.AddAll(7, 8, 9)
	count = s.RetainFunc(func(e int) bool { return e > 5 })
	// Keep 7, 8, 9 (all > 5), remove 1, 3, 5 = 3 removed
	assert.Equal(t, 3, count, "Count should match expected")
	assert.Equal(t, 3, s.Size(), "Size should be 3 after RetainFunc")
	assert.False(t, s.Contains(1), "Should not contain element")
	assert.False(t, s.Contains(3), "Should not contain element")
	assert.False(t, s.Contains(5), "Should not contain element")
	assert.True(t, s.Contains(7), "Contains should be true for expected element")
	assert.True(t, s.Contains(8), "Contains should be true for expected element")
	assert.True(t, s.Contains(9), "Contains should be true for expected element")
}

func TestConcurrentSkipSet_ClearAndSize(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(1, 2, 3)
	assert.False(t, s.IsEmpty(), "Set should start non-empty")
	assert.Equal(t, 3, s.Size(), "Size should be 3 before Clear")
	s.Clear()
	assert.True(t, s.IsEmpty(), "Set should be empty after Clear")
	assert.Equal(t, 0, s.Size(), "Size should be 0 after Clear")
}

func TestConcurrentSkipSet_ToSliceAndPop(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(5, 3, 1, 4, 2)
	slice := s.ToSlice()
	slices.Sort(slice)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, slice, "ToSlice should contain all elements in order")
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
	assert.Contains(t, []int{1, 2, 3, 4, 5}, v, "Contains should be true for expected element")
	assert.Equal(t, 4, s.Size(), "Size should decrease by one after Pop")
}

func TestConcurrentSkipSet_AddSeqRemoveSeq(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddSeq(func(yield func(int) bool) {
		if !yield(1) {
			return
		}
		if !yield(2) {
			return
		}
		yield(3)
	})
	assert.Equal(t, 3, s.Size(), "Size should be 3 after AddSeq")
	removed := s.RemoveSeq(func(yield func(int) bool) {
		if !yield(1) {
			return
		}
		yield(2)
	})
	assert.Equal(t, 2, removed, "RemoveSeq should remove yielded elements")
	assert.Equal(t, 1, s.Size(), "Size should reflect RemoveSeq")
}

func TestConcurrentSkipSet_ContainsAllAny(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(1, 2, 3, 4, 5)
	assert.True(t, s.ContainsAll(1, 3, 5), "ContainsAll should be true for expected elements")
	assert.False(t, s.ContainsAll(1, 6), "Should not contain element")
	assert.True(t, s.ContainsAny(1, 10, 20), "ContainsAny should be true for expected elements")
	assert.False(t, s.ContainsAny(10, 20, 30), "Should not contain element")
}

func TestConcurrentSkipSet_IsSubsetSupersetDisjoint(t *testing.T) {
	t.Parallel()
	a := NewConcurrentSkipSetFrom(1, 2, 3)
	b := NewConcurrentSkipSetFrom(1, 2, 3, 4, 5)
	c := NewConcurrentSkipSetFrom(4, 5, 6)
	assert.True(t, a.IsSubsetOf(b), "IsSubsetOf should be true")
	assert.False(t, b.IsSubsetOf(a), "IsSubsetOf should be false")
	assert.True(t, b.IsSupersetOf(a), "IsSupersetOf should be true")
	assert.True(t, a.IsProperSubsetOf(b), "IsProperSubsetOf should be true")
	assert.False(t, a.IsProperSubsetOf(a), "IsProperSubsetOf should be false")
	assert.True(t, a.IsDisjoint(c), "IsDisjoint should be true")
	assert.False(t, a.IsDisjoint(b), "IsDisjoint should be false")
}

func TestConcurrentSkipSet_Equals(t *testing.T) {
	t.Parallel()
	a := NewConcurrentSkipSetFrom(1, 2, 3)
	b := NewConcurrentSkipSetFrom(1, 2, 3)
	c := NewConcurrentSkipSetFrom(1, 2, 4)
	assert.True(t, a.Equals(b), "Equals should be true")
	assert.False(t, a.Equals(c), "Equals should be false")
}

func TestConcurrentSkipSet_Races(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(_ *testing.T) {
		s := NewConcurrentSkipSet[int]()
		workers := runtime.GOMAXPROCS(0) * 2
		iters := 500
		for w := range workers {
			go func(id int) {
				for i := range iters {
					k := (id * 997) ^ i
					switch i % 3 {
					case 0:
						s.Add(k)
					case 1:
						s.Remove(k)
					default:
						_ = s.Contains(k)
					}
					if i%100 == 0 {
						count := 0
						s.Range(-1<<31, 1<<31-1, func(_ int) bool {
							if count > 10 {
								return false
							}
							count++
							return true
						})
					}
				}
			}(w)
		}
		synctest.Wait()
	})
}

func TestConcurrentSkipSet_RemoveAll(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(1, 2, 3, 4, 5)
	removed := s.RemoveAll(1, 3, 5)
	assert.Equal(t, 3, removed, "RemoveAll should remove three keys")
	assert.Equal(t, 2, s.Size(), "Size should reflect removals")
	assert.False(t, s.Contains(1), "Should not contain element")
	assert.False(t, s.Contains(3), "Should not contain element")
	assert.False(t, s.Contains(5), "Should not contain element")
}

func TestConcurrentSkipSet_AddIfAbsentAndRemoveAndGet(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	assert.True(t, s.AddIfAbsent(1), "AddIfAbsent should be true for new element")
	assert.False(t, s.AddIfAbsent(1), "AddIfAbsent should be false for duplicate")
	v, ok := s.RemoveAndGet(1)
	require.True(t, ok, "RemoveAndGet should succeed for present element")
	assert.Equal(t, 1, v, "RemoveAndGet should return removed value")
	_, ok = s.RemoveAndGet(1)
	require.False(t, ok, "RemoveAndGet should be false for missing element")
}

func TestConcurrentSkipSet_String(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.Add(1)
	str := s.String()
	require.Contains(t, str, "concurrentSkipSet", "String should include type name")
	require.Contains(t, str, "1", "String should include element values")
}

func TestConcurrentSkipSet_CoverageSupplement(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSetFrom(1, 2, 3)

	// ForEach
	cnt := 0
	s.ForEach(func(_ int) bool {
		cnt++
		return true
	})
	require.Equal(t, 3, cnt, "Count should match expected")

	// Clone
	c := s.Clone()
	require.True(t, s.Equals(c), "Clone should be equal to original")

	// IsProperSupersetOf
	sub := NewConcurrentSkipSetFrom(1, 2)
	assert.True(t, s.IsProperSupersetOf(sub), "IsProperSupersetOf should be true")

	// Find
	found, fok := s.Find(func(e int) bool { return e == 2 })
	require.True(t, fok, "Find should succeed for matching element")
	assert.Equal(t, 2, found, "Find should return the matching element")

	// Filter
	filter := s.Filter(func(e int) bool { return e > 1 })
	assert.Equal(t, 2, filter.Size(), "Filter should keep elements > 1")

	// Lower, Higher extra checks
	// s has 1, 2, 3
	_, ok := s.Lower(1)
	require.False(t, ok, "Lower should be none for minimum element")
	v, ok := s.Lower(3)
	require.True(t, ok, "Lower should return predecessor for non-min element")
	assert.Equal(t, 2, v, "Lower should return predecessor value")

	v, ok = s.Higher(1)
	require.True(t, ok, "Higher should return successor for non-max element")
	assert.Equal(t, 2, v, "Higher should return successor value")
	_, ok = s.Higher(3)
	require.False(t, ok, "Higher should be none for maximum element")

	// RangeSeq - range is inclusive [from, to], includes 1, 2, 3
	count := 0
	for range s.RangeSeq(1, 3) {
		count++
	}
	assert.Equal(t, 3, count, "RangeSeq(1, 3) should include elements 1, 2, 3")
}

func TestConcurrentSkipSet_PopFirstLastEmpty(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()

	// PopFirst on empty set
	_, ok := s.PopFirst()
	require.False(t, ok, "PopFirst should fail on empty set")

	// PopLast on empty set
	_, ok = s.PopLast()
	require.False(t, ok, "PopLast should fail on empty set")

	// Add single element and pop
	s.Add(42)
	v, ok := s.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 42, v, "PopFirst should return the element")
	require.True(t, s.IsEmpty(), "Set should be empty after popping last element")
}

func TestConcurrentSkipSet_RangeSeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	// RangeSeq with early exit
	collected := make([]int, 0)
	for v := range s.RangeSeq(2, 8) {
		collected = append(collected, v)
		if v >= 5 {
			break // early exit
		}
	}
	require.Equal(t, []int{2, 3, 4, 5}, collected, "RangeSeq should support early exit")

	// RangeSeq empty range (from > to)
	empty := make([]int, 0)
	for v := range s.RangeSeq(8, 2) {
		empty = append(empty, v)
	}
	require.Empty(t, empty, "RangeSeq should return empty when from > to")
}

func TestConcurrentSkipSet_DescendEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// Descend with early exit
	collected := make([]int, 0)
	s.Descend(func(e int) bool {
		collected = append(collected, e)
		return e > 2 // stop when e <= 2
	})
	require.Equal(t, []int{5, 4, 3, 2}, collected, "Descend should support early exit")
}

func TestConcurrentSkipSet_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5, 6)

	// Reversed with early exit
	collected := make([]int, 0)
	for v := range s.Reversed() {
		collected = append(collected, v)
		if v <= 3 {
			break
		}
	}
	require.Equal(t, []int{6, 5, 4, 3}, collected, "Reversed should support early exit")
}

func TestConcurrentSkipSet_GetByRankOutOfBounds(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(10, 20, 30, 40, 50)

	// Valid rank
	v, ok := s.GetByRank(2)
	require.True(t, ok, "GetByRank should succeed for valid rank")
	require.Equal(t, 30, v, "GetByRank(2) should return third element")

	// Out of bounds rank
	_, ok = s.GetByRank(10)
	require.False(t, ok, "GetByRank should fail for rank >= size")

	// Negative rank
	_, ok = s.GetByRank(-1)
	require.False(t, ok, "GetByRank should fail for negative rank")

	// Empty set
	empty := NewConcurrentSkipSet[int]()
	_, ok = empty.GetByRank(0)
	require.False(t, ok, "GetByRank should fail on empty set")
}

func TestConcurrentSkipSet_RangeEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	// Range with early exit
	collected := make([]int, 0)
	s.Range(2, 8, func(e int) bool {
		collected = append(collected, e)
		return e < 5 // continue until e >= 5
	})
	require.Equal(t, []int{2, 3, 4, 5}, collected, "Range should support early exit")
}

func TestConcurrentSkipSet_DescendFromEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	s.AddAll(1, 2, 3, 4, 5, 6, 7, 8)

	// DescendFrom with early exit
	collected := make([]int, 0)
	s.DescendFrom(6, func(e int) bool {
		collected = append(collected, e)
		return e > 3 // stop when e <= 3
	})
	require.Equal(t, []int{6, 5, 4, 3}, collected, "DescendFrom should support early exit")
}

func TestConcurrentSkipSet_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// Seq with early exit
	collected := make([]int, 0)
	for v := range s.Seq() {
		collected = append(collected, v)
		if v >= 5 {
			break
		}
	}
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected, "Seq should support early exit")
}

func TestConcurrentSkipSet_EmptySetEdgeCases(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()

	// First/Last on empty set
	_, ok := s.First()
	require.False(t, ok, "First should return false on empty set")
	_, ok = s.Last()
	require.False(t, ok, "Last should return false on empty set")

	// Min/Max on empty set
	_, ok = s.Min()
	require.False(t, ok, "Min should return false on empty set")
	_, ok = s.Max()
	require.False(t, ok, "Max should return false on empty set")

	// Pop on empty set
	_, ok = s.Pop()
	require.False(t, ok, "Pop should return false on empty set")

	// Floor/Ceiling on empty set
	_, ok = s.Floor(10)
	require.False(t, ok, "Floor should return false on empty set")
	_, ok = s.Ceiling(10)
	require.False(t, ok, "Ceiling should return false on empty set")
}

func TestConcurrentSkipSet_AscendFromEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// AscendFrom with early exit
	collected := make([]int, 0)
	s.AscendFrom(5, func(e int) bool {
		collected = append(collected, e)
		return e < 8 // stop when e >= 8
	})
	require.Equal(t, []int{5, 6, 7, 8}, collected, "AscendFrom should support early exit")
}

func TestConcurrentSkipSet_ForEachEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// ForEach with early exit
	collected := make([]int, 0)
	s.ForEach(func(e int) bool {
		collected = append(collected, e)
		return e < 5 // stop when e >= 5
	})
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected, "ForEach should support early exit")
}

func TestConcurrentSkipSet_AnyEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// Any should stop early when predicate returns true
	callCount := 0
	result := s.Any(func(e int) bool {
		callCount++
		return e == 3 // Found at 3rd element
	})
	require.True(t, result, "Any should return true when element found")
	require.Equal(t, 3, callCount, "Any should stop early when predicate returns true")
}

func TestConcurrentSkipSet_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	assert.False(t, s.IsNotEmpty(), "new set should not be non-empty")
	s.Add(1)
	assert.True(t, s.IsNotEmpty(), "set with element should be non-empty")
}

// Regression: Clear used to swap the underlying skip list pointer without
// synchronization, racing with every concurrent accessor.
func TestConcurrentSkipSet_ClearConcurrentWithAdd(t *testing.T) {
	t.Parallel()
	s := NewConcurrentSkipSet[int]()
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 2000 {
			s.Add(i % 10)
		}
	})
	wg.Go(func() {
		for range 2000 {
			s.Clear()
		}
	})
	wg.Wait()
	s.Clear()
	require.Equal(t, 0, s.Size(), "quiescent Clear should leave the set empty")
}
