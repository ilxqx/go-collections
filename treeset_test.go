package collections

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Panic tests colocated with tree set unit tests.
func TestTreeSet_PanicOnNilComparator(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			require.Fail(t, "Expected panic on nil comparator")
		}
	}()
	_ = NewTreeSet[int](nil)
}

func TestTreeSet_BasicAndOrder(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(3, 1, 2, 2)
	got := s.ToSlice()
	slices.Sort(got)
	assert.True(t, slices.Equal(got, []int{1, 2, 3}), "ToSlice=%v", got)
	asc := make([]int, 0, 3)
	for v := range s.Seq() {
		asc = append(asc, v)
	}
	assert.True(t, slices.IsSorted(asc), "Seq not ascending: %v", asc)
	// Descending via Reversed
	dec := make([]int, 0, 3)
	for v := range s.Reversed() {
		dec = append(dec, v)
	}
	assert.Equal(t, 3, len(dec), "Reversed should yield three elements")
	assert.Equal(t, 3, dec[0], "Reversed first element should be max")
	assert.Equal(t, 1, dec[2], "Reversed last element should be min")
}

func TestTreeSet_NavigationAndExtremes(t *testing.T) {
	t.Parallel()
	s := NewTreeSet(func(a, b int) int { return cmp.Compare(a, b) })
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
	// PopFirst/PopLast
	v, ok = s.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 10, v, "Returned value should match expected")
	v, ok = s.PopLast()
	require.True(t, ok, "PopLast should succeed")
	require.Equal(t, 40, v, "Returned value should match expected")
}

func TestTreeSet_RangeAndRank(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
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
	require.Equal(t, 1, v, "Returned value should match expected")
}

func TestTreeSet_Views(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
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

// Basic Set ops (emptiness, add/contains/remove) and algebra on TreeSet.
func TestTreeSet_SetOpsBasic(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	// emptiness
	assert.True(t, s.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, s.Size(), "HeadSet without inclusive should exclude pivot")
	// add & contains
	assert.True(t, s.Add(1), "Add should succeed for new element")
	assert.False(t, s.Add(1), "Add should be false for duplicate") // duplicate
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	// AddAll count (at least 2 because 3 is duplicate later)
	added := s.AddAll(2, 3, 3)
	assert.GreaterOrEqual(t, added, 2, "Value should be at least expected")
	// Remove existing / non-existing
	assert.True(t, s.Remove(2), "Remove should succeed for present element")
	assert.False(t, s.Remove(99), "Remove should be false for missing element")
	// Algebra with another TreeSet
	other := NewTreeSetOrdered[int]()
	other.AddAll(1, 3, 4)
	union := s.Union(other)
	assert.True(t, union.ContainsAll(1, 3, 4), "ContainsAll should be true for expected elements")
	inter := s.Intersection(other)
	assert.True(t, inter.ContainsAll(1, 3), "ContainsAll should be true for expected elements")
	diff := other.Difference(s)
	assert.True(t, diff.Contains(4), "Contains should be true for expected element")
	sym := s.SymmetricDifference(other)
	assert.True(t, sym.Contains(4) && !sym.Contains(1), "Contains should be true for expected element")
}

func TestTreeSet_AscendDescend(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(5, 1, 3, 2, 4)
	// Ascend - should iterate in ascending order
	asc := make([]int, 0)
	s.Ascend(func(e int) bool {
		asc = append(asc, e)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 4, 5}, asc, "Ascend should iterate ascending order")

	// Descend - should iterate in descending order
	desc := make([]int, 0)
	s.Descend(func(e int) bool {
		desc = append(desc, e)
		return true
	})
	assert.Equal(t, []int{5, 4, 3, 2, 1}, desc, "Descend should iterate descending order")

	// AscendFrom - iterate elements >= 3 ascending
	ascFrom := make([]int, 0)
	s.AscendFrom(3, func(e int) bool {
		ascFrom = append(ascFrom, e)
		return true
	})
	assert.Equal(t, []int{3, 4, 5}, ascFrom, "AscendFrom should iterate from pivot upwards")

	// DescendFrom - iterate elements <= 3 descending
	descFrom := make([]int, 0)
	s.DescendFrom(3, func(e int) bool {
		descFrom = append(descFrom, e)
		return true
	})
	assert.Equal(t, []int{3, 2, 1}, descFrom, "DescendFrom should iterate from pivot downwards")

	// Test early termination
	count := 0
	s.Ascend(func(e int) bool {
		count++
		return count < 3
	})
	assert.Equal(t, 3, count, "Ascend should stop after early termination")
}

func TestTreeSet_CloneSorted(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(1, 2, 3)
	clone := s.CloneSorted()
	clone.Remove(1)
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	assert.Equal(t, 3, s.Size(), "Original size should remain 3")
	assert.Equal(t, 2, clone.Size(), "Clone size should reflect removal")
}

func TestTreeSet_CoverageSupplement(t *testing.T) {
	t.Parallel()
	s := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	assert.Equal(t, 3, s.Size(), "Size should be 3 for initial set")

	// String
	str := s.String()
	assert.Contains(t, str, "treeSet", "String should include type name")
	assert.Contains(t, str, "1", "String should include element values")

	// ForEach
	cnt := 0
	s.ForEach(func(e int) bool {
		cnt++
		return true
	})
	assert.Equal(t, 3, cnt, "ForEach should visit all elements")

	// Pop
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
	assert.Contains(t, []int{1, 2, 3}, v, "Contains should be true for expected element")
	assert.Equal(t, 2, s.Size(), "Size should be 2 after Pop")

	// AddSeq
	added := s.AddSeq(func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		yield(5)
	})
	assert.Equal(t, 2, added, "AddAll should count unique inserts")
	assert.True(t, s.ContainsAll(4, 5), "ContainsAll should be true for expected elements")

	// RemoveSeq
	removed := s.RemoveSeq(func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		yield(99) // not present
	})
	assert.Equal(t, 1, removed, "Difference should remove one element unique to other")
	assert.Equal(t, 3, s.Size(), "Size should remain 3 after set algebra")

	// RemoveFunc
	removed = s.RemoveFunc(func(e int) bool { return e == 5 })
	assert.Equal(t, 1, removed, "RemoveAll should remove one matching element")
	assert.False(t, s.Contains(5), "Should not contain element")

	// RetainFunc
	s.AddAll(10, 11)
	// Only retain > 5 (should remove existing small elements)
	removed = s.RetainFunc(func(e int) bool { return e > 5 })
	assert.Greater(t, removed, 0, "Value should be greater than expected")
	assert.True(t, s.ContainsAll(10, 11), "ContainsAll should be true for expected elements")
	assert.False(t, s.Contains(2), "Should not contain element") // was present before

	// Set Relations
	setA := NewTreeSetFrom(cmp.Compare[int], 1, 2)
	setB := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	setC := NewTreeSetFrom(cmp.Compare[int], 4, 5)

	assert.True(t, setA.IsSubsetOf(setB), "IsSubsetOf should be true")
	assert.True(t, setB.IsSupersetOf(setA), "IsSupersetOf should be true")
	assert.True(t, setA.IsProperSubsetOf(setB), "IsProperSubsetOf should be true")
	assert.True(t, setB.IsProperSupersetOf(setA), "IsProperSupersetOf should be true")
	assert.True(t, setA.IsDisjoint(setC), "IsDisjoint should be true")
	assert.False(t, setA.IsSubsetOf(setC), "IsSubsetOf should be false")

	// Clone
	c := setA.Clone()
	assert.True(t, c.Equals(setA), "Equals should be true")

	// Filter
	f := setB.Filter(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 1, f.Size(), "Filter should keep one element")
	assert.True(t, f.Contains(2), "Contains should be true for expected element")

	// Any, Every, Find
	assert.True(t, setB.Any(func(e int) bool { return e == 3 }), "Any should be true for matching element")
	assert.True(t, setB.Every(func(e int) bool { return e > 0 }), "Every should be true when all match")
	val, ok := setB.Find(func(e int) bool { return e == 2 })
	require.True(t, ok, "Find should succeed for matching element")
	assert.Equal(t, 2, val, "Find should return matching element")
}

func TestTreeSet_EmptyOperations(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()

	// PopFirst on empty set
	_, ok := s.PopFirst()
	require.False(t, ok, "PopFirst should return false on empty set")

	// PopLast on empty set
	_, ok = s.PopLast()
	require.False(t, ok, "PopLast should return false on empty set")

	// Lower on empty set
	_, ok = s.Lower(5)
	require.False(t, ok, "Lower should return false on empty set")

	// Higher on empty set
	_, ok = s.Higher(5)
	require.False(t, ok, "Higher should return false on empty set")

	// First/Last on empty set
	_, ok = s.First()
	require.False(t, ok, "First should return false on empty set")
	_, ok = s.Last()
	require.False(t, ok, "Last should return false on empty set")
}

func TestTreeSet_IsDisjoint(t *testing.T) {
	t.Parallel()
	// Test when other is smaller (other.Size() < t.Size())
	s1 := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5) // 5 elements
	s2 := NewTreeSetFrom(cmp.Compare[int], 6, 7)          // 2 elements (smaller)
	assert.True(t, s1.IsDisjoint(s2), "IsDisjoint should be true for disjoint sets (other smaller)")

	// Test when sets have common elements (other smaller)
	s3 := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5) // 5 elements
	s4 := NewTreeSetFrom(cmp.Compare[int], 3, 6)          // 2 elements (smaller, has common element)
	assert.False(t, s3.IsDisjoint(s4), "IsDisjoint should be false when sets share elements")

	// Test empty set with non-empty set
	sEmpty := NewTreeSetOrdered[int]()
	sNonEmpty := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	assert.True(t, sEmpty.IsDisjoint(sNonEmpty), "Empty set should be disjoint with any set")
	assert.True(t, sNonEmpty.IsDisjoint(sEmpty), "Any set should be disjoint with empty set")

	// Test both empty sets
	assert.True(t, sEmpty.IsDisjoint(sEmpty), "Two empty sets should be disjoint")
}

func TestTreeSet_LowerHigherBoundaries(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(1, 3, 5, 7, 9)

	// Lower - element strictly less than x
	_, ok := s.Lower(1)
	require.False(t, ok, "Lower(1) should return false (no element < 1)")

	v, ok := s.Lower(2)
	require.True(t, ok, "Lower(2) should find 1")
	require.Equal(t, 1, v, "Lower(2) should return 1")

	v, ok = s.Lower(3)
	require.True(t, ok, "Lower(3) should find element < 3")
	require.Equal(t, 1, v, "Lower(3) should return 1 (strictly less)")

	v, ok = s.Lower(10)
	require.True(t, ok, "Lower(10) should find 9")
	require.Equal(t, 9, v, "Lower(10) should return 9")

	// Higher - element strictly greater than x
	_, ok = s.Higher(9)
	require.False(t, ok, "Higher(9) should return false (no element > 9)")

	v, ok = s.Higher(8)
	require.True(t, ok, "Higher(8) should find 9")
	require.Equal(t, 9, v, "Higher(8) should return 9")

	_, ok = s.Higher(9)
	require.False(t, ok, "Higher(9) should return false (strictly greater)")

	v, ok = s.Higher(0)
	require.True(t, ok, "Higher(0) should find 1")
	require.Equal(t, 1, v, "Higher(0) should return 1")

	// Test with single element set
	sSingle := NewTreeSetOrdered[int]()
	sSingle.Add(5)
	_, ok = sSingle.Lower(5)
	require.False(t, ok, "Lower(5) on single element set should be false")
	_, ok = sSingle.Higher(5)
	require.False(t, ok, "Higher(5) on single element set should be false")
	v, ok = sSingle.Lower(10)
	require.True(t, ok, "Lower(10) on single element set should find 5")
	require.Equal(t, 5, v)
	v, ok = sSingle.Higher(0)
	require.True(t, ok, "Higher(0) on single element set should find 5")
	require.Equal(t, 5, v)
}

func TestTreeSet_EveryEmpty(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()

	// Every on empty set should return true (vacuous truth)
	result := s.Every(func(e int) bool { return e > 0 })
	require.True(t, result, "Every should return true for empty set")

	// Every with predicate that fails
	s.AddAll(1, 2, 3, 4, 5)
	result = s.Every(func(e int) bool { return e < 3 })
	require.False(t, result, "Every should return false when some elements don't match")

	// Every with all matching
	result = s.Every(func(e int) bool { return e > 0 })
	require.True(t, result, "Every should return true when all elements match")
}

func TestTreeSet_RangeSeqEarlyExit(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
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
	require.Equal(t, 0, len(empty), "RangeSeq should return empty when from > to")
}

func TestTreeSet_FirstLastEntry(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()

	// FirstEntry and LastEntry on empty set
	_, ok := s.First()
	require.False(t, ok, "First should fail on empty set")

	_, ok = s.Last()
	require.False(t, ok, "Last should fail on empty set")

	// Add elements and test
	s.AddAll(5, 2, 8, 1, 9)
	v, ok := s.First()
	require.True(t, ok, "First should succeed on non-empty set")
	require.Equal(t, 1, v, "First should return smallest element")

	v, ok = s.Last()
	require.True(t, ok, "Last should succeed on non-empty set")
	require.Equal(t, 9, v, "Last should return largest element")
}

func TestTreeSet_FloorCeilingNotFound(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(10, 20, 30)

	// Floor - no floor exists
	_, ok := s.Floor(5)
	require.False(t, ok, "Floor should fail when no element <= target")

	// Ceiling - no ceiling exists
	_, ok = s.Ceiling(40)
	require.False(t, ok, "Ceiling should fail when no element >= target")

	// Floor with exact match
	v, ok := s.Floor(20)
	require.True(t, ok, "Floor should find exact match")
	require.Equal(t, 20, v, "Floor should return exact match")

	// Ceiling with exact match
	v, ok = s.Ceiling(20)
	require.True(t, ok, "Ceiling should find exact match")
	require.Equal(t, 20, v, "Ceiling should return exact match")
}

func TestTreeSet_PopFirstLastEmpty(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()

	// PopFirst on empty
	_, ok := s.PopFirst()
	require.False(t, ok, "PopFirst should fail on empty set")

	// PopLast on empty
	_, ok = s.PopLast()
	require.False(t, ok, "PopLast should fail on empty set")

	// Add single element and pop
	s.Add(42)
	v, ok := s.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 42, v, "PopFirst should return the element")
	require.True(t, s.IsEmpty(), "Set should be empty after popping last element")
}

func TestTreeSet_GetByRankOutOfBounds(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
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
}

func TestTreeSet_RangeReversed(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// Range with from > to should return empty
	collected := make([]int, 0)
	s.Range(5, 1, func(e int) bool {
		collected = append(collected, e)
		return true
	})
	require.Equal(t, 0, len(collected), "Range should return empty when from > to")
}

func TestTreeSet_SubSetReversed(t *testing.T) {
	t.Parallel()
	s := NewTreeSetOrdered[int]()
	s.AddAll(1, 2, 3, 4, 5)

	// SubSet with from > to should return empty
	sub := s.SubSet(5, 1)
	require.Equal(t, 0, sub.Size(), "SubSet should return empty when from > to")
}

func TestTreeSet_IntersectionSizeOptimization(t *testing.T) {
	t.Parallel()
	// Test Intersection when other.Size() < t.Size() (triggers swap optimization)
	s1 := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) // large set
	s2 := NewTreeSetFrom(cmp.Compare[int], 3, 4, 5)                       // small set

	// Should iterate over s2 (smaller) and check against s1 (larger)
	inter := s1.Intersection(s2)
	require.Equal(t, 3, inter.Size(), "Intersection should contain 3 elements")
	require.True(t, inter.ContainsAll(3, 4, 5), "Intersection should contain expected elements")
}

func TestTreeSet_SymmetricDifferenceBothSides(t *testing.T) {
	t.Parallel()
	s1 := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	s2 := NewTreeSetFrom(cmp.Compare[int], 3, 4, 5)

	// SymmetricDifference should include elements unique to each set
	sym := s1.SymmetricDifference(s2)
	require.Equal(t, 4, sym.Size(), "SymmetricDifference should have 4 elements")
	require.True(t, sym.ContainsAll(1, 2, 4, 5), "Should contain elements from both sets")
	require.False(t, sym.Contains(3), "Should not contain common element")
}

func TestTreeSet_EqualsDifferentSizes(t *testing.T) {
	t.Parallel()
	s1 := NewTreeSetFrom(cmp.Compare[int], 1, 2, 3)
	s2 := NewTreeSetFrom(cmp.Compare[int], 1, 2)

	// Different sizes should return false immediately
	require.False(t, s1.Equals(s2), "Equals should return false for sets with different sizes")
}

func TestTreeSet_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewTreeSet[int](func(a, b int) int { return a - b })
	assert.False(t, s.IsNotEmpty(), "new set should not be non-empty")
	s.Add(1)
	assert.True(t, s.IsNotEmpty(), "set with element should be non-empty")
}
