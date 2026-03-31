package collections

import (
	"runtime"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentHashSet_Basic(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSet[int]()
	assert.True(t, s.IsEmpty(), "New set should be empty")
	assert.True(t, s.Add(1), "Add should succeed for new element")
	assert.False(t, s.Add(1), "Add duplicate should be false")
	assert.True(t, s.Contains(1), "Contains should be true for expected element")
	r, ok := s.RemoveAndGet(1)
	require.True(t, ok, "RemoveAndGet should succeed for present element")
	assert.Equal(t, 1, r, "RemoveAndGet should return removed value")
	assert.False(t, s.Contains(1), "Should not contain element")
	// RemoveAndGet non-existing
	_, ok = s.RemoveAndGet(99)
	require.False(t, ok, "RemoveAndGet should be false for missing element")
}

func TestConcurrentHashSet_ConcurrentAddIfAbsent(t *testing.T) {
t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		s := NewConcurrentHashSet[int]()
		n := 1000
		workers := runtime.GOMAXPROCS(0) * 2
		for w := range workers {
			go func(id int) {
				for i := range n {
					s.AddIfAbsent(i)
				}
			}(w)
		}
		synctest.Wait()
		// validate presence of a few keys deterministically
		for _, k := range []int{0, n / 2, n - 1} {
			assert.Truef(t, s.Contains(k), "Missing key %d", k)
		}
	})
}

func TestConcurrentHashSet_Algebra(t *testing.T) {
t.Parallel()
	a := NewConcurrentHashSetFrom(1, 2, 3, 4)
	b := NewConcurrentHashSetFrom(3, 4, 5)

	// Union
	u := a.Union(b).ToSlice()

	// Intersection
	i := a.Intersection(b).ToSlice()

	// Difference
	d := a.Difference(b).ToSlice()

	// SymmetricDifference
	sd := a.SymmetricDifference(b).ToSlice()

	expectContains := func(slice []int, x int) bool {
		return slices.Contains(slice, x)
	}

	assert.Truef(t, expectContains(u, 1) && expectContains(u, 5), "Union unexpected: %v", u)
	assert.Len(t, i, 2, "Intersection unexpected: %v", i)
	assert.True(t, expectContains(i, 3) && expectContains(i, 4), "Contains should be true for expected element")

	assert.Len(t, d, 2, "Difference unexpected: %v", d)
	assert.True(t, expectContains(d, 1) && expectContains(d, 2), "Contains should be true for expected element")

	assert.Len(t, sd, 3, "SymmetricDifference unexpected: %v", sd)
	assert.True(t, expectContains(sd, 1) && expectContains(sd, 2) && expectContains(sd, 5), "Contains should be true for expected element")
}

func TestConcurrentHashSet_ClearAndSize(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSetFrom(1, 2, 3)
	assert.False(t, s.IsEmpty(), "IsEmpty should be false")
	assert.Equal(t, 3, s.Size(), "Size should be 3 before Clear")

	// ToSlice
	slice := s.ToSlice()
	assert.Len(t, slice, 3, "ToSlice should return 3 elements")

	// String
	str := s.String()
	assert.Contains(t, str, "concurrentHashSet", "String should include type name")
	assert.Contains(t, str, "1", "String should include element values")

	s.Clear()
	assert.True(t, s.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, s.Size(), "Size should be 0 after Clear")
}

func TestConcurrentHashSet_PopAndRemove(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSetFrom(1, 2, 3)
	// Pop returns arbitrary element that is removed
	v, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
	assert.Contains(t, []int{1, 2, 3}, v, "Contains should be true for expected element")
	assert.Equal(t, 2, s.Size(), "Size should decrease by one after Pop")

	// Pop on empty
	s.Clear()
	_, ok = s.Pop()
	require.False(t, ok, "Pop should be false on empty set")

	s.Add(4)
	// Remove
	assert.True(t, s.Remove(4), "Remove should succeed for present element")
	assert.False(t, s.Remove(4), "Remove should be false for missing element")
}

func TestConcurrentHashSet_AddAndRemoveAll(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSet[int]()

	// AddAll
	added := s.AddAll(1, 2, 3)
	assert.Equal(t, 3, added, "AddAll should count unique inserts")
	added = s.AddAll(2, 4) // 2 exists, 4 is new
	assert.Equal(t, 1, added, "AddAll should add only new elements")
	assert.True(t, s.Contains(4), "Contains should be true for expected element")

	// AddSeq
	added = s.AddSeq(func(yield func(int) bool) {
		if !yield(5) {
			return
		}
		if !yield(6) {
			return
		}
		yield(1) // exists
	})
	assert.Equal(t, 2, added, "Added count should match expected")

	// RemoveAll
	removed := s.RemoveAll(1, 2, 99)
	assert.Equal(t, 2, removed, "RemoveAll should remove only present elements")
	assert.False(t, s.Contains(1), "Should not contain element")

	// RemoveSeq
	removed = s.RemoveSeq(func(yield func(int) bool) {
		if !yield(3) {
			return
		}
		if !yield(4) {
			return
		}
		yield(88)
	})
	assert.Equal(t, 2, removed, "RemoveSeq should remove yielded elements")
	assert.False(t, s.Contains(3), "Should not contain element")
}

func TestConcurrentHashSet_ContainsOperations(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSetFrom(1, 2, 3)

	// ContainsAll
	assert.True(t, s.ContainsAll(1, 2), "ContainsAll should be true for expected elements")
	assert.False(t, s.ContainsAll(1, 4), "Should not contain element")

	// ContainsAny
	assert.True(t, s.ContainsAny(1, 4), "ContainsAny should be true for expected elements")
	assert.False(t, s.ContainsAny(4, 5), "Should not contain element")
}

func TestConcurrentHashSet_SubsetSuperset(t *testing.T) {
t.Parallel()
	s1 := NewConcurrentHashSetFrom(1, 2)
	s2 := NewConcurrentHashSetFrom(1, 2, 3)
	s3 := NewConcurrentHashSetFrom(2, 3)
	s4 := NewConcurrentHashSetFrom(4, 5)

	// IsSubsetOf
	assert.True(t, s1.IsSubsetOf(s2), "IsSubsetOf should be true")
	assert.False(t, s2.IsSubsetOf(s1), "IsSubsetOf should be false")

	// IsSupersetOf
	assert.True(t, s2.IsSupersetOf(s1), "IsSupersetOf should be true")

	// IsProperSubsetOf
	assert.True(t, s1.IsProperSubsetOf(s2), "IsProperSubsetOf should be true")
	assert.False(t, s1.IsProperSubsetOf(s1), "IsProperSubsetOf should be false")

	// IsProperSupersetOf
	assert.True(t, s2.IsProperSupersetOf(s1), "IsProperSupersetOf should be true")

	// IsDisjoint
	assert.False(t, s1.IsDisjoint(s2), "IsDisjoint should be false")
	assert.False(t, s2.IsDisjoint(s3), "IsDisjoint should be false")
	assert.True(t, s1.IsDisjoint(s4), "IsDisjoint should be true")
}

func TestConcurrentHashSet_CloneFilterEquals(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSetFrom(1, 2, 3)

	// Clone
	c := s.Clone()
	assert.True(t, s.Equals(c), "Clone should be equal to original")

	c.Add(4)
	assert.False(t, s.Equals(c), "Equals should be false")

	// Filter
	even := s.Filter(func(e int) bool { return e%2 == 0 })
	assert.Equal(t, 1, even.Size(), "Filter should keep one even element")
	assert.True(t, even.Contains(2), "Contains should be true for expected element")

	// Equals comparison
	s2 := NewHashSetFrom(1, 2, 3)
	assert.True(t, s.Equals(s2), "Equals should ignore element order")
}

func TestConcurrentHashSet_IteratorsAndPredicates(t *testing.T) {
t.Parallel()
	s := NewConcurrentHashSetFrom(1, 2, 3, 4, 5)

	// ForEach
	cnt := 0
	s.ForEach(func(e int) bool {
		cnt++
		return true
	})
	assert.Equal(t, 5, cnt, "ForEach should visit all elements")

	// Seq
	cnt = 0
	for range s.Seq() {
		cnt++
	}
	assert.Equal(t, 5, cnt, "Seq should iterate all elements")

	// Find
	v, ok := s.Find(func(e int) bool { return e > 3 })
	require.True(t, ok, "Find should locate a value > 3")
	assert.Contains(t, []int{4, 5}, v, "Contains should be true for expected element")

	_, ok = s.Find(func(e int) bool { return e > 10 })
	require.False(t, ok, "Find should be false when predicate never matches")

	// Any
	assert.True(t, s.Any(func(e int) bool { return e == 3 }), "Any should be true for matching element")
	assert.False(t, s.Any(func(e int) bool { return e == 10 }), "Any should be false when no match")

	// Every
	assert.True(t, s.Every(func(e int) bool { return e > 0 }), "Every should be true when all match")
	assert.False(t, s.Every(func(e int) bool { return e < 5 }), "Every should be false when any fail")

	// RemoveFunc
	removed := s.RemoveFunc(func(e int) bool { return e%2 == 0 }) // remove 2, 4
	assert.Equal(t, 2, removed, "RemoveFunc should remove two evens")
	assert.False(t, s.Contains(2), "Should not contain element")

	// RetainFunc
	s.Add(2)
	s.Add(4)
	removed = s.RetainFunc(func(e int) bool { return e%2 != 0 }) // keep odds: 1, 3, 5; remove 2, 4
	assert.Equal(t, 2, removed, "RetainFunc should remove two evens")
	assert.False(t, s.Contains(2), "Should not contain element")
}

func TestConcurrentHashSet_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewConcurrentHashSet[int]()
	assert.False(t, s.IsNotEmpty(), "new set should not be non-empty")
	s.Add(1)
	assert.True(t, s.IsNotEmpty(), "set with element should be non-empty")
}
