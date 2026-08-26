package collections

import (
	"cmp"
	"runtime"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentTreeSet_BasicOrdered(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_CustomComparator(t *testing.T) {
	t.Parallel()
	// reverse order comparator
	cmpRev := func(a, b int) int { return -cmp.Compare(a, b) }
	s := NewConcurrentTreeSet(cmpRev)
	s.AddAll(1, 2, 3)
	got := make([]int, 0)
	for v := range s.Seq() {
		got = append(got, v)
	}
	assert.True(t, slices.Equal(got, []int{3, 2, 1}), "Seq with reverse comparator=%v", got)
}

func TestConcurrentTreeSet_ConcurrentAddIfAbsent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_NavigationAndExtremes(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(10, 20, 30, 40)
	v, ok := s.First()
	require.True(t, ok, "First should succeed on non-empty set")
	assert.Equal(t, 10, v, "Sequence should match expected")
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

func TestConcurrentTreeSet_PopFirstLast(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(10, 20, 30, 40)
	v, ok := s.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 10, v, "Returned value should match expected")
	v, ok = s.PopLast()
	require.True(t, ok, "PopLast should succeed")
	require.Equal(t, 40, v, "Returned value should match expected")
	require.Equal(t, 2, s.Size(), "Size should be 2 after PopFirst and PopLast")
}

func TestConcurrentTreeSet_RangeAndRank(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_Views(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_SetOpsBasic(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	assert.True(t, s.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, s.Size(), "Size should be 0 for new set")
	assert.True(t, s.Add(1), "Add should succeed for new element")
	assert.False(t, s.Add(1), "Add should be false for duplicate")
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	added := s.AddAll(2, 3, 3)
	assert.GreaterOrEqual(t, added, 2, "Value should be at least expected")
	assert.True(t, s.Remove(2), "Remove should succeed for present element")
	assert.False(t, s.Remove(99), "Remove should be false for missing element")
}

func TestConcurrentTreeSet_Algebra(t *testing.T) {
	t.Parallel()
	a := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4)
	b := NewConcurrentTreeSetFrom(cmp.Compare[int], 3, 4, 5)
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

func TestConcurrentTreeSet_AscendDescend(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_CloneSorted(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(1, 2, 3)
	clone := s.CloneSorted()
	clone.Remove(1)
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	assert.Equal(t, 3, s.Size(), "Size should remain 3 after clone mutation")
	assert.Equal(t, 2, clone.Size(), "Clone size should match expected")
}

func TestConcurrentTreeSet_FilterFindAnyEvery(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_RemoveFuncRetainFunc(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5, 6)
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

func TestConcurrentTreeSet_ClearAndSize(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	assert.False(t, s.IsEmpty(), "IsEmpty should be false")
	assert.Equal(t, 3, s.Size(), "Size should be 3 before Clear")
	s.Clear()
	assert.True(t, s.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, s.Size(), "Size should be 0 after Clear")
}

func TestConcurrentTreeSet_ToSliceAndPop(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetFrom(cmp.Compare[int], 5, 3, 1, 4, 2)
	slice := s.ToSlice()
	slices.Sort(slice)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, slice, "Slice should match expected")
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
	assert.Contains(t, []int{1, 2, 3, 4, 5}, v, "Contains should be true for expected element")
	assert.Equal(t, 4, s.Size(), "Size should be 4 after Pop")
}

func TestConcurrentTreeSet_AddSeqRemoveSeq(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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
	assert.Equal(t, 2, removed, "Removed count should match expected")
	assert.Equal(t, 1, s.Size(), "Size should be 1 after RemoveSeq")
}

func TestConcurrentTreeSet_ContainsAllAny(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5)
	assert.True(t, s.ContainsAll(1, 3, 5), "ContainsAll should be true for expected elements")
	assert.False(t, s.ContainsAll(1, 6), "Should not contain element")
	assert.True(t, s.ContainsAny(1, 10, 20), "ContainsAny should be true for expected elements")
	assert.False(t, s.ContainsAny(10, 20, 30), "Should not contain element")
}

func TestConcurrentTreeSet_IsSubsetSupersetDisjoint(t *testing.T) {
	t.Parallel()
	a := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	b := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5)
	c := NewConcurrentTreeSetFrom(cmp.Compare[int], 4, 5, 6)
	assert.True(t, a.IsSubsetOf(b), "IsSubsetOf should be true")
	assert.False(t, b.IsSubsetOf(a), "IsSubsetOf should be false")
	assert.True(t, b.IsSupersetOf(a), "IsSupersetOf should be true")
	assert.True(t, a.IsProperSubsetOf(b), "IsProperSubsetOf should be true")
	assert.False(t, a.IsProperSubsetOf(a), "IsProperSubsetOf should be false")
	assert.True(t, a.IsDisjoint(c), "IsDisjoint should be true")
	assert.False(t, a.IsDisjoint(b), "IsDisjoint should be false")
}

func TestConcurrentTreeSet_Equals(t *testing.T) {
	t.Parallel()
	a := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	b := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	c := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 4)
	assert.True(t, a.Equals(b), "Equals should be true")
	assert.False(t, a.Equals(c), "Equals should be false")
}

// Race test colocated with concurrent tree set tests.
func TestConcurrentTreeSet_Races(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(_ *testing.T) {
		s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_RemoveAll(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5)
	removed := s.RemoveAll(1, 3, 5)
	assert.Equal(t, 3, removed, "Removed count should match expected")
	assert.Equal(t, 2, s.Size(), "Size should be 2 after RemoveAll")
	assert.False(t, s.Contains(1), "Should not contain element")
	assert.False(t, s.Contains(3), "Should not contain element")
	assert.False(t, s.Contains(5), "Should not contain element")
}

func TestConcurrentTreeSet_AddIfAbsentAndRemoveAndGet(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	assert.True(t, s.AddIfAbsent(1), "AddIfAbsent should be true for new element")
	assert.False(t, s.AddIfAbsent(1), "AddIfAbsent should be false for duplicate")
	v, ok := s.RemoveAndGet(1)
	require.True(t, ok, "RemoveAndGet should succeed for present element")
	assert.Equal(t, 1, v, "RemoveAndGet should return removed value")
	_, ok = s.RemoveAndGet(1)
	require.False(t, ok, "RemoveAndGet should be false for missing element")
}

func TestConcurrentTreeSet_ForEachEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// ForEach with early exit (return false)
	count := 0
	s.ForEach(func(_ int) bool {
		count++
		return false // Early exit after first iteration
	})
	require.Equal(t, 1, count, "ForEach should stop when callback returns false")
}

func TestConcurrentTreeSet_FindScenarios(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// Find when element exists
	v, ok := s.Find(func(e int) bool { return e > 3 })
	require.True(t, ok, "Find should succeed for matching element")
	require.Equal(t, 4, v, "Find should return first matching element")

	// Find when no element matches
	_, ok = s.Find(func(e int) bool { return e > 100 })
	require.False(t, ok, "Find should be false when no element matches")

	// Find in empty set
	emptySet := NewConcurrentTreeSetOrdered[int]()
	_, ok = emptySet.Find(func(_ int) bool { return true })
	require.False(t, ok, "Find should be false for empty set")
}

func TestConcurrentTreeSet_AscendScenarios(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// Ascend iterates all elements in ascending order
	allKeys := []int{}
	s.Ascend(func(e int) bool {
		allKeys = append(allKeys, e)
		return true
	})
	require.Equal(t, []int{1, 2, 3, 4, 5}, allKeys, "Ascend should iterate all elements in ascending order")

	// Ascend from pivot
	ascKeys := []int{}
	s.AscendFrom(3, func(e int) bool {
		ascKeys = append(ascKeys, e)
		return true
	})
	require.Equal(t, []int{3, 4, 5}, ascKeys, "AscendFrom should include pivot and elements greater than it")

	// Ascend on empty set
	emptySet := NewConcurrentTreeSetOrdered[int]()
	count := 0
	emptySet.Ascend(func(_ int) bool {
		count++
		return true
	})
	require.Equal(t, 0, count, "Ascend on empty set should not call callback")

	// Ascend with early termination
	earlyCount := 0
	s.Ascend(func(_ int) bool {
		earlyCount++
		return earlyCount < 3 // Stop after 3 iterations
	})
	require.Equal(t, 3, earlyCount, "Ascend should stop when action returns false")
}

func TestConcurrentTreeSet_CoverageSupplement(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	s.Add(1)
	s.Add(2)

	// String
	str := s.String()
	require.Contains(t, str, "concurrentTreeSet", "String should include type name") // camelCase implied

	// ForEach
	cnt := 0
	s.ForEach(func(_ int) bool {
		cnt++
		return true
	})
	require.Equal(t, 2, cnt, "Count should match expected")

	// Clone
	c := s.Clone()
	require.True(t, s.Equals(c), "Clone should be equal to original")

	// AddSeq
	s.AddSeq(func(yield func(int) bool) {
		if !yield(3) {
			return
		}
		yield(4)
	})
	require.Equal(t, 4, s.Size(), "Size should be 4 after AddSeq")

	// RemoveSeq
	s.RemoveSeq(func(yield func(int) bool) {
		yield(3)
	})
	require.Equal(t, 3, s.Size(), "Size should be 3 after RemoveSeq")

	// RemoveFunc
	s.RemoveFunc(func(e int) bool { return e == 4 })
	require.Equal(t, 2, s.Size(), "Size should be 2 after RemoveFunc")

	// RetainFunc
	s.RetainFunc(func(e int) bool { return e == 1 })
	require.Equal(t, 1, s.Size(), "Size should be 1 after RetainFunc")
	require.True(t, s.Contains(1), "Contains should be true for expected element")

	// Pop
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
	require.Equal(t, 1, v, "Returned value should match expected")

	// Relations
	s1 := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2)
	s2 := NewConcurrentTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	require.True(t, s1.IsSubsetOf(s2), "IsSubsetOf should be true")
	require.True(t, s2.IsSupersetOf(s1), "IsSupersetOf should be true")
	require.True(t, s1.IsProperSubsetOf(s2), "IsProperSubsetOf should be true")
	require.True(t, s2.IsProperSupersetOf(s1), "IsProperSupersetOf should be true")
	s3 := NewConcurrentTreeSetFrom(cmp.Compare[int], 4)
	require.True(t, s1.IsDisjoint(s3), "IsDisjoint should be true")

	// Find/Any/Every/Filter
	s2.Add(3)
	found, fok := s2.Find(func(e int) bool { return e == 2 })
	require.True(t, fok, "Find should succeed for matching element")
	require.Equal(t, 2, found, "Returned value should match expected")
	require.True(t, s2.Any(func(e int) bool { return e == 2 }), "Any should be true for matching element")
	require.True(t, s2.Every(func(e int) bool { return e > 0 }), "Every should be true when all match")

	filter := s2.Filter(func(e int) bool { return e == 1 })
	require.Equal(t, 1, filter.Size(), "Filter should keep one element")
}

func TestConcurrentTreeSet_AnyEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_DescendEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// Descend with early exit
	collected := make([]int, 0)
	s.Descend(func(e int) bool {
		collected = append(collected, e)
		return e > 6 // stop when e <= 6
	})
	require.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Descend should support early exit")
}

func TestConcurrentTreeSet_AscendFromEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_DescendFromEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// DescendFrom with early exit
	collected := make([]int, 0)
	s.DescendFrom(7, func(e int) bool {
		collected = append(collected, e)
		return e > 4 // stop when e <= 4
	})
	require.Equal(t, []int{7, 6, 5, 4}, collected, "DescendFrom should support early exit")
}

func TestConcurrentTreeSet_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	for i := 1; i <= 10; i++ {
		s.Add(i)
	}

	// Reversed with early exit
	collected := make([]int, 0)
	for v := range s.Reversed() {
		collected = append(collected, v)
		if v <= 6 {
			break
		}
	}
	require.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Reversed should support early exit")
}

func TestConcurrentTreeSet_EmptySetEdgeCases(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()

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

	// PopFirst/PopLast on empty set
	_, ok = s.PopFirst()
	require.False(t, ok, "PopFirst should return false on empty set")
	_, ok = s.PopLast()
	require.False(t, ok, "PopLast should return false on empty set")

	// Pop on empty set
	_, ok = s.Pop()
	require.False(t, ok, "Pop should return false on empty set")
}

func TestConcurrentTreeSet_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
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

func TestConcurrentTreeSet_RangeSeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSetOrdered[int]()
	for i := range 10 {
		s.Add(i + 1)
	}

	// RangeSeq with early exit
	collected := make([]int, 0)
	for v := range s.RangeSeq(3, 8) {
		collected = append(collected, v)
		if v >= 5 {
			break
		}
	}
	require.Equal(t, []int{3, 4, 5}, collected, "RangeSeq should support early exit")
}

func TestConcurrentTreeSet_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewConcurrentTreeSet[int](func(a, b int) int { return a - b })
	assert.False(t, s.IsNotEmpty(), "new set should not be non-empty")
	s.Add(1)
	assert.True(t, s.IsNotEmpty(), "set with element should be non-empty")
}
