package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkedList_Basic(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()
	assert.True(t, l.IsEmpty(), "New list should be empty")
	assert.Equal(t, 0, l.Size(), "Empty list should have size 0")

	l.Add(1)
	l.Add(2)
	l.Add(3)

	assert.False(t, l.IsEmpty(), "List should not be empty after adds")
	assert.Equal(t, 3, l.Size(), "Size should equal number of added elements")

	v, ok := l.Get(0)
	require.True(t, ok, "Get(0) should succeed")
	assert.Equal(t, 1, v, "Get(0) should return first element")

	v, ok = l.Get(1)
	require.True(t, ok, "Get(1) should succeed")
	assert.Equal(t, 2, v, "Get(1) should return second element")

	v, ok = l.Get(2)
	require.True(t, ok, "Get(2) should succeed")
	assert.Equal(t, 3, v, "Get(2) should return third element")

	_, ok = l.Get(3)
	assert.False(t, ok, "Get(out of range) should fail")

	_, ok = l.Get(-1)
	assert.False(t, ok, "Get(negative) should fail")
}

func TestLinkedList_AddFirst(t *testing.T) {
	t.Parallel()
	l, ok := NewLinkedList[int]().(*linkedList[int])
	require.True(t, ok, "NewLinkedList should return *linkedList")
	l.AddFirst(3)
	l.AddFirst(2)
	l.AddFirst(1)

	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "AddFirst should prepend elements")
}

func TestLinkedList_FirstLast(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()

	_, ok := l.First()
	assert.False(t, ok, "First on empty should fail")

	_, ok = l.Last()
	assert.False(t, ok, "Last on empty should fail")

	l.Add(1)
	l.Add(2)
	l.Add(3)

	v, ok := l.First()
	require.True(t, ok, "First should succeed after adds")
	assert.Equal(t, 1, v, "First should return first element")

	v, ok = l.Last()
	require.True(t, ok, "Last should succeed after adds")
	assert.Equal(t, 3, v, "Last should return last element")
}

func TestLinkedList_Set(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)

	old, ok := l.Set(1, 20)
	require.True(t, ok, "Set should succeed for valid index")
	assert.Equal(t, 2, old, "Set should return previous value")

	v, _ := l.Get(1)
	assert.Equal(t, 20, v, "Value at index should be updated")

	_, ok = l.Set(10, 100)
	assert.False(t, ok, "Set out of range should fail")
}

func TestLinkedList_Insert(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 3)

	ok := l.Insert(1, 2)
	require.True(t, ok, "Insert in middle should succeed")
	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "List should reflect insertion")

	ok = l.Insert(0, 0)
	require.True(t, ok, "Insert at head should succeed")
	assert.Equal(t, []int{0, 1, 2, 3}, l.ToSlice(), "Insert at head should prepend")

	ok = l.Insert(4, 4)
	require.True(t, ok, "Insert at tail should succeed")
	assert.Equal(t, []int{0, 1, 2, 3, 4}, l.ToSlice(), "Insert at tail should append")

	ok = l.Insert(-1, -1)
	assert.False(t, ok, "Insert negative index should fail")

	ok = l.Insert(100, 100)
	assert.False(t, ok, "Insert out of range should fail")
}

func TestLinkedList_InsertAll(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 5)

	ok := l.InsertAll(1, 2, 3, 4)
	require.True(t, ok, "InsertAll should succeed")
	assert.Equal(t, []int{1, 2, 3, 4, 5}, l.ToSlice(), "InsertAll should put elements at index")
}

func TestLinkedList_InsertAllEmpty(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)
	ok := l.InsertAll(1) // empty elements
	assert.True(t, ok, "InsertAll with no elements should succeed")
	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "List should be unchanged")

	// InsertAll at head with empty
	ok = l.InsertAll(0)
	assert.True(t, ok, "InsertAll at head with no elements should succeed")
	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "List should be unchanged")
}

func TestLinkedList_InsertAllOutOfBounds(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)
	ok := l.InsertAll(-1, 4, 5)
	assert.False(t, ok, "InsertAll with negative index should fail")

	ok = l.InsertAll(10, 4, 5)
	assert.False(t, ok, "InsertAll with index > size should fail")

	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "List should be unchanged")
}

func TestLinkedList_RemoveAt(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5)

	v, ok := l.RemoveAt(2)
	require.True(t, ok, "RemoveAt should succeed")
	assert.Equal(t, 3, v, "RemoveAt should return removed element")
	assert.Equal(t, []int{1, 2, 4, 5}, l.ToSlice(), "List should reflect removal")

	v, ok = l.RemoveAt(0)
	require.True(t, ok, "RemoveAt head should succeed")
	assert.Equal(t, 1, v, "RemoveAt head returns first element")
	assert.Equal(t, []int{2, 4, 5}, l.ToSlice(), "List should reflect head removal")

	v, ok = l.RemoveAt(2)
	require.True(t, ok, "RemoveAt tail should succeed")
	assert.Equal(t, 5, v, "RemoveAt tail returns last element")
	assert.Equal(t, []int{2, 4}, l.ToSlice(), "List should reflect tail removal")

	_, ok = l.RemoveAt(10)
	assert.False(t, ok, "RemoveAt out of range should fail")
}

func TestLinkedList_Remove(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 2, 1)
	eq := EqualFunc[int]()

	ok := l.Remove(2, eq)
	require.True(t, ok, "Remove existing element should succeed")
	assert.Equal(t, []int{1, 3, 2, 1}, l.ToSlice(), "List should remove first matching element")

	ok = l.Remove(100, eq)
	assert.False(t, ok, "Remove non-existent element should fail")
}

func TestLinkedList_RemoveFirstLast(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)

	v, ok := l.RemoveFirst()
	require.True(t, ok, "RemoveFirst should succeed")
	assert.Equal(t, 1, v, "RemoveFirst should return first element")

	v, ok = l.RemoveLast()
	require.True(t, ok, "RemoveLast should succeed")
	assert.Equal(t, 3, v, "RemoveLast should return last element")

	assert.Equal(t, []int{2}, l.ToSlice(), "List should contain only middle element now")

	l.Clear()
	_, ok = l.RemoveFirst()
	assert.False(t, ok, "RemoveFirst on empty should fail")

	_, ok = l.RemoveLast()
	assert.False(t, ok, "RemoveLast on empty should fail")
}

func TestLinkedList_RemoveFunc(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RemoveFunc(func(v int) bool { return v%2 == 0 })
	assert.Equal(t, 3, removed, "RemoveFunc should remove three evens")
	assert.Equal(t, []int{1, 3, 5}, l.ToSlice(), "Only odds should remain")
}

func TestLinkedList_RetainFunc(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5, 6)

	removed := l.RetainFunc(func(v int) bool { return v%2 == 0 })
	assert.Equal(t, 3, removed, "RetainFunc should remove three odds")
	assert.Equal(t, []int{2, 4, 6}, l.ToSlice(), "Only evens should remain")
}

func TestLinkedList_IndexOf(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 2, 1)
	eq := EqualFunc[int]()

	assert.Equal(t, 1, l.IndexOf(2, eq), "IndexOf should find first match")
	assert.Equal(t, 3, l.LastIndexOf(2, eq), "LastIndexOf should find last match")
	assert.Equal(t, -1, l.IndexOf(100, eq), "IndexOf should return -1 when not found")
}

func TestLinkedList_Contains(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)
	eq := EqualFunc[int]()

	assert.True(t, l.Contains(2, eq), "Contains should be true for present value")
	assert.False(t, l.Contains(100, eq), "Contains should be false for missing value")
}

func TestLinkedList_Find(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5)

	v, ok := l.Find(func(n int) bool { return n > 3 })
	require.True(t, ok, "Find should locate value > 3")
	assert.Equal(t, 4, v, "Find should return first matching value")

	_, ok = l.Find(func(n int) bool { return n > 100 })
	assert.False(t, ok, "Find should fail when predicate never matches")

	idx := l.FindIndex(func(n int) bool { return n > 3 })
	assert.Equal(t, 3, idx, "FindIndex should return first index > 3")

	idx = l.FindIndex(func(n int) bool { return n > 100 })
	assert.Equal(t, -1, idx, "FindIndex should return -1 if not found")
}

func TestLinkedList_SubList(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5)

	sub := l.SubList(1, 4)
	assert.Equal(t, []int{2, 3, 4}, sub.ToSlice(), "SubList [1,4) should return middle slice")

	sub = l.SubList(-1, 3)
	assert.Equal(t, 0, sub.Size(), "Invalid from should produce empty list")

	sub = l.SubList(3, 2)
	assert.Equal(t, 0, sub.Size(), "From > to should produce empty list")
}

func TestLinkedList_Reversed(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)

	var reversed []int
	for v := range l.Reversed() {
		reversed = append(reversed, v)
	}
	assert.Equal(t, []int{3, 2, 1}, reversed, "Reversed should iterate from tail to head")
}

func TestLinkedList_Clone(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)
	clone := l.Clone()

	assert.Equal(t, l.ToSlice(), clone.ToSlice(), "Clone should copy elements")

	l.Add(4)
	assert.NotEqual(t, l.Size(), clone.Size(), "Clone should be independent")
}

func TestLinkedList_Filter(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5, 6)

	evens := l.Filter(func(v int) bool { return v%2 == 0 })
	assert.Equal(t, []int{2, 4, 6}, evens.ToSlice(), "Filter should keep only evens")
}

func TestLinkedList_Sort(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(5, 2, 8, 1, 9, 3)
	l.Sort(CompareFunc[int]())

	assert.Equal(t, []int{1, 2, 3, 5, 8, 9}, l.ToSlice(), "Sort should produce ascending order")
}

func TestLinkedList_SortEmpty(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()
	l.Sort(CompareFunc[int]())
	assert.Equal(t, 0, l.Size(), "Sorting empty list should keep size 0")
}

func TestLinkedList_SortSingle(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(42)
	l.Sort(CompareFunc[int]())
	assert.Equal(t, []int{42}, l.ToSlice(), "Sorting single-element list is a no-op")
}

func TestLinkedList_AnyEvery(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(2, 4, 6, 8)

	assert.True(t, l.Any(func(v int) bool { return v > 5 }), "Any should return true for v>5")
	assert.False(t, l.Any(func(v int) bool { return v > 100 }), "Any should return false if none match")

	assert.True(t, l.Every(func(v int) bool { return v%2 == 0 }), "Every should be true for evens")
	assert.False(t, l.Every(func(v int) bool { return v > 5 }), "Every should be false if any <= 5")
}

func TestLinkedList_Seq(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)

	var collected []int
	for v := range l.Seq() {
		collected = append(collected, v)
	}
	assert.Equal(t, []int{1, 2, 3}, collected, "Seq should iterate in insertion order")
}

func TestLinkedList_ForEach(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3, 4, 5)

	var sum int
	l.ForEach(func(v int) bool {
		sum += v
		return v < 3
	})
	assert.Equal(t, 6, sum, "ForEach should stop after v<3 becomes false")
}

func TestLinkedList_String(t *testing.T) {
	t.Parallel()
	l := NewLinkedListFrom(1, 2, 3)
	s := l.String()
	assert.Contains(t, s, "linkedList", "String should include type name")
	assert.Contains(t, s, "1", "String should include element values")
	assert.Contains(t, s, "2", "String should include element values")
	assert.Contains(t, s, "3", "String should include element values")
}

func TestLinkedList_AddAll(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()
	l.AddAll(1, 2, 3)
	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "AddAll should append elements")
}

func TestLinkedList_AddSeq(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()
	other := NewLinkedListFrom(1, 2, 3)
	l.AddSeq(other.Seq())
	assert.Equal(t, []int{1, 2, 3}, l.ToSlice(), "AddSeq should append all elements from seq")
}

func TestLinkedList_IsNotEmpty(t *testing.T) {
	t.Parallel()
	l := NewLinkedList[int]()
	assert.False(t, l.IsNotEmpty(), "new list should not be non-empty")
	l.Add(1)
	assert.True(t, l.IsNotEmpty(), "list with element should be non-empty")
}
