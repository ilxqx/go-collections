package collections

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Panic tests colocated with tree map unit tests.
func TestTreeMap_PanicOnNilComparator(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			require.Fail(t, "Expected panic on nil comparator")
		}
	}()
	_ = NewTreeMap[int, int](nil)
}

func TestTreeMap_BasicAndOrder(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(2, "b")
	m.Put(1, "a")
	m.Put(3, "c")
	ks := m.Keys()
	assert.True(t, slices.IsSorted(ks), "Keys not sorted: %v", ks)
	dec := make([]int, 0, 3)
	for k := range m.Reversed() {
		dec = append(dec, k)
	}
	assert.Equal(t, 3, len(dec), "Reversed should yield three keys")
	assert.Equal(t, 3, dec[0], "Reversed first key should be max")
	assert.Equal(t, 1, dec[2], "Reversed last key should be min")
}

func TestTreeMap_NavigationAndExtremes(t *testing.T) {
	t.Parallel()
	m := NewTreeMap[int, int](func(a, b int) int { return cmp.Compare(a, b) })
	for i := 1; i <= 5; i++ {
		m.Put(i*10, i)
	}
	k, ok := m.FirstKey()
	require.True(t, ok, "FirstKey should succeed on non-empty map")
	assert.Equal(t, 10, k, "FirstKey should return smallest key")
	k, ok = m.LastKey()
	require.True(t, ok, "LastKey should succeed on non-empty map")
	assert.Equal(t, 50, k, "LastKey should return largest key")
	e, ok := m.FirstEntry()
	require.True(t, ok, "FirstEntry should succeed on non-empty map")
	assert.Equal(t, 10, e.Key, "FirstEntry should have smallest key")
	assert.Equal(t, 1, e.Value, "FirstEntry should have corresponding value")
	e, ok = m.LastEntry()
	require.True(t, ok, "LastEntry should succeed on non-empty map")
	assert.Equal(t, 50, e.Key, "LastEntry should have largest key")
	assert.Equal(t, 5, e.Value, "LastEntry should have corresponding value")
	e, ok = m.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	assert.Equal(t, 10, e.Key, "PopFirst should return smallest key")
	e, ok = m.PopLast()
	require.True(t, ok, "PopLast should succeed")
	assert.Equal(t, 50, e.Key, "PopLast should return largest key")
	k, ok = m.FloorKey(33)
	require.True(t, ok, "FloorKey should find 30 for key 33")
	assert.Equal(t, 30, k, "FloorKey should return floor key")
	k, ok = m.CeilingKey(33)
	require.True(t, ok, "CeilingKey should find 40 for key 33")
	assert.Equal(t, 40, k, "CeilingKey should return ceiling key")
	_, ok = m.LowerKey(20)
	require.False(t, ok, "LowerKey(20) should be none after PopFirst")
	_, ok = m.HigherKey(40)
	require.False(t, ok, "HigherKey(40) should be none after PopLast")
}

func TestTreeMap_RangeRankAndViews(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	m.Put(10, 1)
	m.Put(20, 2)
	m.Put(30, 3)
	m.Put(40, 4)
	// Range
	rs := make([]int, 0)
	m.Range(15, 35, func(k, v int) bool {
		rs = append(rs, k)
		return true
	})
	assert.True(t, slices.Equal(rs, []int{20, 30}), "Range=%v", rs)
	// Rank
	assert.Equal(t, 2, m.RankOfKey(30), "Rank should match expected")
	e, ok := m.GetByRank(1)
	require.True(t, ok, "GetByRank should succeed for valid rank")
	require.Equal(t, 20, e.Key, "Key should match expected")
	// SubMap, HeadMap, TailMap
	sub := m.SubMap(15, 35).Keys()
	head := m.HeadMap(25, false).Keys()
	tail := m.TailMap(25, true).Keys()
	require.True(t, slices.Equal(sub, []int{20, 30}), "SubMap=%v", sub)
	require.True(t, slices.Equal(head, []int{10, 20}), "HeadMap=%v", head)
	require.True(t, slices.Equal(tail, []int{30, 40}), "TailMap=%v", tail)
}

func TestTreeMap_AscendDescend(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	m.Put(5, 50)
	m.Put(1, 10)
	m.Put(3, 30)
	m.Put(2, 20)
	m.Put(4, 40)
	// Ascend
	asc := make([]int, 0)
	m.Ascend(func(k, v int) bool {
		asc = append(asc, k)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 4, 5}, asc, "Ascend should iterate ascending order")
	// Descend
	desc := make([]int, 0)
	m.Descend(func(k, v int) bool {
		desc = append(desc, k)
		return true
	})
	assert.Equal(t, []int{5, 4, 3, 2, 1}, desc, "Descend should iterate descending order")
	// AscendFrom
	af := make([]int, 0)
	m.AscendFrom(3, func(k, v int) bool {
		af = append(af, k)
		return true
	})
	assert.Equal(t, []int{3, 4, 5}, af, "AscendFrom should iterate from pivot upwards")
	// DescendFrom
	df := make([]int, 0)
	m.DescendFrom(3, func(k, v int) bool {
		df = append(df, k)
		return true
	})
	assert.Equal(t, []int{3, 2, 1}, df, "DescendFrom should iterate from pivot downwards")
	// Test early termination
	count := 0
	m.Ascend(func(k, v int) bool {
		count++
		return count < 3
	})
	assert.Equal(t, 3, count, "Ascend should stop after early termination")
}

func TestTreeMap_RangeFromAndTo(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i, i*10)
	}
	// RangeFrom
	rf := make([]int, 0)
	m.RangeFrom(3, func(k, v int) bool {
		rf = append(rf, k)
		return true
	})
	assert.Equal(t, []int{3, 4, 5}, rf, "Slice should match expected")
	// RangeTo
	rt := make([]int, 0)
	m.RangeTo(3, func(k, v int) bool {
		rt = append(rt, k)
		return true
	})
	assert.Equal(t, []int{1, 2, 3}, rt, "Slice should match expected")
}

func TestTreeMap_RangeSeq(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	m.Put(10, 1)
	m.Put(20, 2)
	m.Put(30, 3)
	m.Put(40, 4)
	m.Put(50, 5)
	rs := make([]int, 0)
	for k := range m.RangeSeq(15, 45) {
		rs = append(rs, k)
	}
	assert.Equal(t, []int{20, 30, 40}, rs, "Slice should match expected")
}

func TestTreeMap_EntryNavigation(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i*10, i)
	}
	e, ok := m.FloorEntry(35)
	require.True(t, ok, "FloorEntry should succeed")
	assert.Equal(t, 30, e.Key, "Key should match expected")
	e, ok = m.CeilingEntry(35)
	require.True(t, ok, "CeilingEntry should succeed")
	assert.Equal(t, 40, e.Key, "Key should match expected")
	e, ok = m.LowerEntry(25)
	require.True(t, ok, "LowerEntry should succeed")
	assert.Equal(t, 20, e.Key, "Key should match expected")
	e, ok = m.HigherEntry(25)
	require.True(t, ok, "HigherEntry should succeed")
	assert.Equal(t, 30, e.Key, "Key should match expected")
}

func TestTreeMap_CloneAndFilter(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	// Clone
	clone := m.Clone()
	clone.Put(4, "d")
	assert.Equal(t, 3, m.Size(), "Size should remain 3 after clone mutation")
	assert.Equal(t, 4, clone.Size(), "Clone size should match expected")
	// CloneSorted
	sortedClone := m.CloneSorted()
	assert.Equal(t, 3, sortedClone.Size(), "Sorted clone size should match expected")
	// Filter
	filtered := m.Filter(func(k int, v string) bool { return k > 1 })
	assert.Equal(t, 2, filtered.Size(), "Filter should keep two keys greater than 1")
}

func TestTreeMap_Equals(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	m2 := NewTreeMapOrdered[int, string]()
	m2.Put(1, "a")
	m2.Put(2, "b")
	m2.Put(3, "c")
	assert.True(t, m.Equals(m2, eqV[string]), "Equals should be true")
	m3 := NewTreeMapOrdered[int, string]()
	m3.Put(1, "a")
	m3.Put(2, "b")
	m3.Put(3, "different")
	assert.False(t, m.Equals(m3, eqV[string]), "Equals should be false")
}

func TestTreeMap_Clear(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	assert.False(t, m.IsEmpty(), "IsEmpty should be false")
	m.Clear()
	assert.True(t, m.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, m.Size(), "Size should be 0 after Clear")
}

func TestTreeMap_PutAllAndPutSeq(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	m2 := NewTreeMapOrdered[int, int]()
	m2.Put(1, 10)
	m2.Put(2, 20)
	m.PutAll(m2)
	assert.Equal(t, 2, m.Size(), "Size should be 2 after PutAll")
	// PutSeq
	m.PutSeq(func(yield func(int, int) bool) {
		if !yield(3, 30) {
			return
		}
		yield(4, 40)
	})
	assert.Equal(t, 4, m.Size(), "Size should be 4 after PutSeq")
}

func TestTreeMap_RemoveFuncAndCompute(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	// RemoveFunc
	count := m.RemoveFunc(func(k int, v string) bool { return k > 1 })
	assert.Equal(t, 2, count, "Count should match expected")
	assert.Equal(t, 1, m.Size(), "Size should be 1 after RemoveFunc")
	// Compute
	m.Put(4, "d")
	v, ok := m.Compute(4, func(k int, old string, exists bool) (string, bool) {
		if exists {
			return old + "_updated", true
		}
		return "new", true
	})
	require.True(t, ok, "Compute should succeed for existing key")
	assert.Equal(t, "d_updated", v, "Compute should return updated value")
}

func TestTreeMap_ComputeRemoveKey(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// Compute on existing key -> remove (keep=false)
	m.Put(3, "c")
	_, ok := m.Compute(3, func(k int, old string, exists bool) (string, bool) {
		return "", false // remove the key
	})
	require.False(t, ok, "Compute should return false when key is removed")
	assert.False(t, m.ContainsKey(3), "Key should be removed")

	// Compute on non-existing key (exists=false, keep=false) -> should do nothing
	_, ok = m.Compute(99, func(k int, old string, exists bool) (string, bool) {
		return "should_not_be_added", false
	})
	require.False(t, ok, "Compute should return false when key doesn't exist")
	assert.False(t, m.ContainsKey(99), "New key should not be added when keep=false")
}

func TestTreeMap_Replace(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// Replace existing key
	old, ok := m.Replace(1, "aa")
	require.True(t, ok, "Replace should succeed for existing key")
	assert.Equal(t, "a", old, "Replace should return old value")
	v, _ := m.Get(1)
	assert.Equal(t, "aa", v, "Value should be updated")

	// Replace non-existing key
	_, ok = m.Replace(99, "new")
	require.False(t, ok, "Replace should fail for non-existing key")
	assert.False(t, m.ContainsKey(99), "New key should not be added")
}

func TestTreeMap_CoverageSupplement(t *testing.T) {
	t.Parallel()
	// NewTreeMapFrom
	m := NewTreeMapFrom(func(a, b int) int { return cmp.Compare(a, b) }, map[int]string{1: "a", 2: "b"})
	assert.Equal(t, 2, m.Size(), "Size should be 2 for initial map")
	assert.True(t, m.ContainsKey(1), "ContainsKey should be true for present key")

	// String
	str := m.String()
	assert.Contains(t, str, "treeMap", "String should include type name")
	assert.Contains(t, str, "1:a", "String should include entry values")

	// GetOrDefault
	assert.Equal(t, "b", m.GetOrDefault(2, "x"), "GetOrDefault should return stored value")
	assert.Equal(t, "x", m.GetOrDefault(99, "x"), "GetOrDefault should return fallback for missing key")

	// RemoveIf
	assert.True(t, m.RemoveIf(1, "a", eqV[string]), "RemoveIf should remove when key/value match")
	assert.False(t, m.RemoveIf(1, "a", eqV[string]), "RemoveIf should be false after entry is removed") // already removed
	assert.False(t, m.RemoveIf(2, "wrong", eqV[string]), "RemoveIf should be false when value mismatches")

	// ContainsValue
	m.Put(3, "c")
	assert.True(t, m.ContainsValue("c", eqV[string]), "ContainsValue should find existing value")
	assert.False(t, m.ContainsValue("z", eqV[string]), "ContainsValue should be false for missing value")

	// RemoveAll, RemoveSeq
	m.PutAll(NewTreeMapFrom(cmp.Compare[int], map[int]string{4: "d", 5: "e", 6: "f"}))
	// keys: 2, 3, 4, 5, 6
	removed := m.RemoveAll(2, 3)
	assert.Equal(t, 2, removed, "Removed count should match expected")
	assert.Equal(t, 3, m.Size(), "Size should be 3 after RemoveAll") // 4, 5, 6 remaining
	removed = m.RemoveSeq(func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		yield(99) // not present
	})
	assert.Equal(t, 1, removed, "Removed count should match expected")
	assert.Equal(t, 2, m.Size(), "Size should be 2 after RemoveSeq") // 5, 6 remaining

	// Values, Keys, Entries
	ks := m.Keys()
	vs := m.Values()
	es := m.Entries()
	assert.Equal(t, 2, len(ks), "Keys length should match expected")
	assert.Equal(t, 2, len(vs), "Values length should match expected")
	assert.Equal(t, 2, len(es), "Entries length should match expected")

	// ForEach
	cnt := 0
	m.ForEach(func(k int, v string) bool {
		cnt++
		return true
	})
	assert.Equal(t, 2, cnt, "Count should match expected")

	// SeqKeys, SeqValues
	kSeq := make([]int, 0)
	for k := range m.SeqKeys() {
		kSeq = append(kSeq, k)
	}
	assert.Len(t, kSeq, 2, "Keys length should match expected")
	vSeq := make([]string, 0)
	for v := range m.SeqValues() {
		vSeq = append(vSeq, v)
	}
	assert.Len(t, vSeq, 2, "Values length should match expected")

	// ComputeIfPresent
	m.Put(10, "ten")
	// Present -> update
	v, ok := m.ComputeIfPresent(10, func(k int, old string) (string, bool) {
		return old + "_updated", true
	})
	require.True(t, ok, "ComputeIfPresent should update and return true for existing key")
	assert.Equal(t, "ten_updated", v, "Compute should return updated value")
	// Present -> remove
	_, ok = m.ComputeIfPresent(10, func(k int, old string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "ComputeIfPresent should return false when function indicates removal")
	assert.False(t, m.ContainsKey(10), "Should not contain element")
	// Absent
	_, ok = m.ComputeIfPresent(999, func(k int, old string) (string, bool) {
		return "new", true
	})
	require.False(t, ok, "ComputeIfPresent should be false for missing key")

	// Merge
	m.Put(20, "a")
	// Merge existing
	v, ok = m.Merge(20, "b", func(old, new string) (string, bool) {
		return old + new, true
	})
	require.True(t, ok, "Merge should update and return true for existing key")
	assert.Equal(t, "ab", v, "Merge should return merged value for existing key")
	// Merge existing -> remove
	_, ok = m.Merge(20, "c", func(old, new string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "Merge should return false when function indicates removal")
	assert.False(t, m.ContainsKey(20), "Should not contain element")
	// Merge absent
	v, ok = m.Merge(30, "new", func(old, new string) (string, bool) {
		return new, true // shouldn't be called
	})
	require.True(t, ok, "Merge should insert and return true for missing key")
	assert.Equal(t, "new", v, "Merge should return new value for missing key")

	// ReplaceIf
	m.Put(40, "val")
	assert.True(t, m.ReplaceIf(40, "val", "newVal", eqV[string]), "ReplaceIf should succeed when current value matches")
	assert.False(t, m.ReplaceIf(40, "wrong", "xxx", eqV[string]), "ReplaceIf should be false when current value mismatches")

	// ReplaceAll
	m.ReplaceAll(func(k int, v string) string {
		return v + "!"
	})
	val, _ := m.Get(40)
	assert.Equal(t, "newVal!", val, "ReplaceAll should update stored value")
}

func TestTreeMap_EmptyMapEdgeCases(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()

	// FirstEntry/LastEntry on empty map
	_, ok := m.FirstEntry()
	assert.False(t, ok, "FirstEntry should return false on empty map")
	_, ok = m.LastEntry()
	assert.False(t, ok, "LastEntry should return false on empty map")

	// PopFirst/PopLast on empty map
	_, ok = m.PopFirst()
	assert.False(t, ok, "PopFirst should return false on empty map")
	_, ok = m.PopLast()
	assert.False(t, ok, "PopLast should return false on empty map")

	// Entry navigation functions when not found
	m.Put(5, "five")
	m.Put(10, "ten")
	m.Put(15, "fifteen")

	_, ok = m.FloorEntry(3) // less than min
	assert.False(t, ok, "FloorEntry should return false when key is less than minimum")
	_, ok = m.CeilingEntry(20) // greater than max
	assert.False(t, ok, "CeilingEntry should return false when key is greater than maximum")
	_, ok = m.LowerEntry(5) // lower than min
	assert.False(t, ok, "LowerEntry should return false when no lower key exists")
	_, ok = m.HigherEntry(15) // higher than max
	assert.False(t, ok, "HigherEntry should return false when no higher key exists")
}

func TestTreeMap_RangeReversed(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, int]()
	m.Put(1, 10)
	m.Put(2, 20)
	m.Put(3, 30)

	// Range with from > to should return empty
	rs := make([]int, 0)
	m.Range(5, 1, func(k, v int) bool {
		rs = append(rs, k)
		return true
	})
	assert.Equal(t, 0, len(rs), "Range should return empty when from > to")

	// RangeSeq with from > to should return empty
	rseq := make([]int, 0)
	for k := range m.RangeSeq(5, 1) {
		rseq = append(rseq, k)
	}
	assert.Equal(t, 0, len(rseq), "RangeSeq should return empty when from > to")

	// SubMap with from > to should return empty
	sub := m.SubMap(5, 1)
	assert.Equal(t, 0, sub.Size(), "SubMap should return empty when from > to")
}

func TestTreeMap_GetByRankEdgeCases(t *testing.T) {
	t.Parallel()
	m := NewTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")

	// Valid rank
	e, ok := m.GetByRank(0)
	require.True(t, ok, "GetByRank(0) should succeed")
	assert.Equal(t, 1, e.Key, "First element key should be 1")

	// Out of bounds rank
	_, ok = m.GetByRank(10)
	assert.False(t, ok, "GetByRank should return false for out of bounds rank")

	// Negative rank (handled by btree, returns false)
	_, ok = m.GetByRank(-1)
	assert.False(t, ok, "GetByRank should return false for negative rank")
}

func TestTreeMap_EqualsDifferentSizes(t *testing.T) {
	t.Parallel()
	m1 := NewTreeMapOrdered[int, string]()
	m1.Put(1, "a")
	m1.Put(2, "b")

	m2 := NewTreeMapOrdered[int, string]()
	m2.Put(1, "a")

	// Different sizes should return false immediately
	assert.False(t, m1.Equals(m2, eqV[string]), "Equals should return false for maps with different sizes")
}

func TestTreeMap_IsNotEmpty(t *testing.T) {
	t.Parallel()
	m := NewTreeMap[int, string](func(a, b int) int { return a - b })
	assert.False(t, m.IsNotEmpty(), "new map should not be non-empty")
	m.Put(1, "a")
	assert.True(t, m.IsNotEmpty(), "map with entry should be non-empty")
}
