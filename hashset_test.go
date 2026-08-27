package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to make a Seq from slice.
func seqOf[T any](vals []T) func(func(T) bool) {
	return func(yield func(T) bool) {
		for _, v := range vals {
			if !yield(v) {
				return
			}
		}
	}
}

func TestHashSet_BasicCRUD(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 2, 3)
	assert.Equal(t, 3, s.Size(), "Size should ignore duplicates")
	assert.True(t, s.Contains(1) && s.Contains(2) && s.Contains(3), "Initial elements should be present")
	assert.False(t, s.Contains(9), "Absent element should not be contained")
	added := s.Add(2)
	assert.False(t, added, "Adding duplicate should return false")
	assert.True(t, s.Add(4), "Add new element should return true")
	assert.True(t, s.Remove(2), "Remove should succeed for present element")
	assert.False(t, s.Remove(2), "Remove should fail for already removed element")
	// Pop
	_, ok := s.Pop()
	require.True(t, ok, "Pop should succeed on non-empty set")
}

func TestHashSet_AddRemoveAllSeq(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	added := s.AddAll(1, 2, 3, 3)
	assert.Equal(t, 3, added, "AddAll should count unique inserts")
	added = s.AddSeq(seqOf([]int{3, 4, 5}))
	assert.Equal(t, 2, added, "AddSeq should add only new elements")
	removed := s.RemoveAll(5, 6)
	assert.Equal(t, 1, removed, "RemoveAll should remove only present elements")
	removed = s.RemoveSeq(seqOf([]int{1, 4, 7}))
	assert.Equal(t, 2, removed, "RemoveSeq should remove two present elements")
}

func TestHashSet_RemoveRetainFunc(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3, 4, 5)
	removed := s.RemoveFunc(func(v int) bool { return v%2 == 0 })
	require.Equal(t, 2, removed, "RemoveFunc should remove two evens")
	removed = s.RetainFunc(func(v int) bool { return v > 3 })          // keep >3
	assert.Equal(t, 2, removed, "Removed count should match expected") // removed 1,3
	assert.True(t, s.Contains(5), "Largest element should remain")
	assert.Equal(t, 1, s.Size(), "Size should be 1 after RetainFunc, state=%v", s.ToSlice())
}

func TestHashSet_SeqForEach(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3, 4)
	collected := make([]int, 0, s.Size())
	for v := range s.Seq() {
		collected = append(collected, v)
	}
	slices.Sort(collected)
	assert.True(t, slices.Equal(collected, []int{1, 2, 3, 4}), "Seq collected=%v", collected)
	n := 0
	s.ForEach(func(_ int) bool {
		n++
		return false // stop immediately
	})
	assert.Equal(t, 1, n, "ForEach should stop after first element")
}

func TestHashSet_Algebra(t *testing.T) {
	t.Parallel()
	a := NewHashSetFrom(1, 2, 3, 4)
	b := NewHashSetFrom(3, 4, 5)
	u := a.Union(b).ToSlice()
	i := a.Intersection(b).ToSlice()
	d := a.Difference(b).ToSlice()
	sd := a.SymmetricDifference(b).ToSlice()
	slices.Sort(u)
	slices.Sort(i)
	slices.Sort(d)
	slices.Sort(sd)
	assert.True(t, slices.Equal(u, []int{1, 2, 3, 4, 5}), "Union=%v", u)
	assert.True(t, slices.Equal(i, []int{3, 4}), "Intersection=%v", i)
	assert.True(t, slices.Equal(d, []int{1, 2}), "Difference=%v", d)
	assert.True(t, slices.Equal(sd, []int{1, 2, 5}), "SymmetricDifference=%v", sd)
}

func TestHashSet_RelationsAndSearch(t *testing.T) {
	t.Parallel()
	a := NewHashSetFrom(1, 2, 3)
	b := NewHashSetFrom(1, 2, 3, 4)
	c := NewHashSetFrom(4, 5)
	assert.True(t, a.IsSubsetOf(b), "Set A should be subset of set B")
	assert.True(t, b.IsSupersetOf(a), "Set B should be superset of set A")
	assert.True(t, a.IsProperSubsetOf(b), "Set A should be proper subset of set B")
	assert.True(t, b.IsProperSupersetOf(a), "Set B should be proper superset of set A")
	assert.True(t, a.IsDisjoint(c), "Set A and set C should be disjoint")
	assert.False(t, c.IsDisjoint(b), "Set C and set B should not be disjoint")
	assert.True(t, a.Equals(NewHashSetFrom(2, 1, 3)), "Equals should ignore order")
	v, ok := a.Find(func(x int) bool { return x%2 == 0 })
	require.True(t, ok, "Find should locate an even element")
	assert.Equal(t, 0, v%2, "Found value should be even")
	assert.True(t, a.Any(func(x int) bool { return x == 2 }), "Any should be true for existing element")
	assert.True(t, a.Every(func(x int) bool { return x >= 1 && x <= 3 }), "Every should be true for range")
}

func TestHashSet_EmptySingle(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	assert.True(t, s.IsEmpty(), "New set should be empty")
	assert.Equal(t, 0, s.Size(), "Empty set size should be 0")
	_, ok := s.Pop()
	require.False(t, ok, "Pop should fail on empty set")
	assert.True(t, s.Add(1), "Add should succeed for new element")
	assert.False(t, s.IsEmpty(), "Set should not be empty after add")
	assert.Equal(t, 1, s.Size(), "Size should be 1 after add")
	assert.True(t, s.Contains(1), "Contains should be true for inserted element")
}

func TestHashSet_Large(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	const n = 20000
	for i := range n {
		s.Add(i)
	}
	assert.Equal(t, n, s.Size(), "Size should equal number of inserted elements")
	for _, k := range []int{0, n / 2, n - 1} {
		require.Truef(t, s.Contains(k), "Missing %d", k)
	}
}

func TestHashSet_Clear(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3)
	assert.False(t, s.IsEmpty(), "Set should start non-empty")
	s.Clear()
	assert.True(t, s.IsEmpty(), "Set should be empty after Clear")
	assert.Equal(t, 0, s.Size(), "Size should be 0 after Clear")
}

func TestHashSet_ToSlice(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(5, 3, 1, 4, 2)
	slice := s.ToSlice()
	slices.Sort(slice)
	assert.Equal(t, []int{1, 2, 3, 4, 5}, slice, "ToSlice should contain all elements")
}

func TestHashSet_Clone(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3)
	clone := s.Clone()
	clone.Remove(1)
	assert.True(t, s.Contains(1), "Original set should be unchanged")
	assert.Equal(t, 3, s.Size(), "Original size should remain 3")
	assert.Equal(t, 2, clone.Size(), "Clone size should reflect removal")
}

func TestHashSet_Filter(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3, 4, 5, 6)
	filtered := s.Filter(func(v int) bool { return v%2 == 0 })
	assert.Equal(t, 3, filtered.Size(), "Filter should keep three evens")
	assert.True(t, filtered.Contains(2), "Filtered set should contain 2")
	assert.True(t, filtered.Contains(4), "Filtered set should contain 4")
	assert.True(t, filtered.Contains(6), "Filtered set should contain 6")
}

func TestHashSet_ContainsAllAny(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3, 4, 5)
	assert.True(t, s.ContainsAll(1, 3, 5), "ContainsAll should be true for present elements")
	assert.False(t, s.ContainsAll(1, 6), "ContainsAll should be false if any missing")
	assert.True(t, s.ContainsAny(1, 10, 20), "ContainsAny should be true if at least one present")
	assert.False(t, s.ContainsAny(10, 20, 30), "ContainsAny should be false if none present")
}

func TestHashSet_NewWithCapacity(t *testing.T) {
	t.Parallel()
	s := NewHashSetWithCapacity[int](10)
	require.True(t, s.IsEmpty(), "New set with capacity should be empty")
	s.Add(1)
	require.Equal(t, 1, s.Size(), "Size should be 1 after add")
}

func TestHashSet_WithCapacityZero(t *testing.T) {
	t.Parallel()
	s := NewHashSetWithCapacity[int](0)
	require.True(t, s.IsEmpty(), "Set with zero capacity should be empty")
	s.Add(1)
	require.Equal(t, 1, s.Size(), "Size should be 1 after add")
}

func TestHashSet_WithCapacityNegative(t *testing.T) {
	t.Parallel()
	s := NewHashSetWithCapacity[int](-5)
	require.True(t, s.IsEmpty(), "Set with negative capacity should be empty")
	s.Add(42)
	require.Equal(t, 1, s.Size(), "Size should be 1 after add")
}

func TestHashSet_FindEmpty(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	_, ok := s.Find(func(x int) bool { return x > 0 })
	require.False(t, ok, "Find on empty set should return false")
}

func TestHashSet_FindNotFound(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom(1, 2, 3)
	_, ok := s.Find(func(x int) bool { return x > 100 })
	require.False(t, ok, "Find should return false when no element satisfies predicate")
}

func TestHashSet_String(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	s.Add(1)
	str := s.String()
	require.Contains(t, str, "hashSet", "String should include type name")
	require.Contains(t, str, "1", "String should include element value")
}

func TestHashSet_IsNotEmpty(t *testing.T) {
	t.Parallel()
	s := NewHashSet[int]()
	assert.False(t, s.IsNotEmpty(), "new set should not be non-empty")
	s.Add(1)
	assert.True(t, s.IsNotEmpty(), "set with element should be non-empty")
}

// Regression: Intersection and IsDisjoint used to iterate whichever set was
// smaller, flipping which set's equality judged membership. The receiver's
// elements are now always tested with other.Contains.
func TestHashSet_IntersectionIndependentOfSizes(t *testing.T) {
	t.Parallel()
	ci := func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) }

	r1 := NewHashSetFrom("a").Intersection(NewTreeSetFrom(ci, "A", "x"))
	r2 := NewHashSetFrom("a", "b").Intersection(NewTreeSetFrom(ci, "A"))
	require.Equal(t, r1.Contains("a"), r2.Contains("a"),
		"whether a/A intersect must not depend on unrelated elements")
	require.True(t, r1.Contains("a"), "membership is judged by other.Contains (case-insensitive here)")

	require.Equal(t,
		NewHashSetFrom("a").IsDisjoint(NewTreeSetFrom(ci, "A", "x")),
		NewHashSetFrom("a", "b").IsDisjoint(NewTreeSetFrom(ci, "A")),
		"IsDisjoint must not depend on unrelated elements")
}

// Regression: decoding a payload holding a dynamically unhashable element
// (any satisfies comparable since Go 1.20) used to replace the backing map,
// store a prefix of the payload, then panic. It is now rejected up front
// with an error and the set is left untouched.
func TestHashSet_DecodeRejectsUnhashableElements(t *testing.T) {
	t.Parallel()
	s := NewHashSetFrom[any]("old")

	err := s.(json.Unmarshaler).UnmarshalJSON([]byte(`[1,[2]]`))
	require.ErrorContains(t, err, "unhashable", "the JSON payload must be rejected, not panic")
	require.True(t, s.Contains("old"), "a rejected payload must leave the set untouched")
	require.Equal(t, 1, s.Size(), "no payload element may leak into the set")

	gob.Register([]int{})
	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode([]any{1, []int{2}}), "the slice itself encodes fine")
	err = s.(gob.GobDecoder).GobDecode(buf.Bytes())
	require.ErrorContains(t, err, "unhashable", "the gob payload must be rejected, not panic")
	require.True(t, s.Contains("old"), "a rejected gob payload must leave the set untouched")
	require.Equal(t, 1, s.Size(), "no payload element may leak into the set")

	require.NoError(t, s.(json.Unmarshaler).UnmarshalJSON([]byte(`[1,2]`)), "a hashable payload should decode")
	require.True(t, s.Contains(float64(1)), "decoded elements should land")
	require.Equal(t, 2, s.Size(), "the hashable payload should replace the old contents")
}
