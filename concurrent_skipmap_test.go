package collections

import (
	"runtime"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestConcurrentSkipMap_BasicOrdered(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(2, "b")
	m.Put(1, "a")
	m.Put(3, "c")
	require.True(t, slices.Equal(m.Keys(), []int{1, 2, 3}), "Keys=%v", m.Keys())
	// Reversed
	rev := make([]int, 0)
	for k := range m.Reversed() {
		rev = append(rev, k)
	}
	require.True(t, slices.Equal(rev, []int{3, 2, 1}), "Reversed=%v", rev)
	// First/Last
	e, ok := m.FirstEntry()
	require.True(t, ok, "FirstEntry should succeed on non-empty map")
	require.Equal(t, 1, e.Key, "Key should match expected")
	require.Equal(t, "a", e.Value, "FirstEntry should return value for smallest key")
	e, ok = m.LastEntry()
	require.True(t, ok, "LastEntry should succeed on non-empty map")
	require.Equal(t, 3, e.Key, "Key should match expected")
	require.Equal(t, "c", e.Value, "LastEntry should return value for largest key")
}

func TestConcurrentSkipMap_PutIfAbsentAndAtomics(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		m := NewConcurrentSkipMap[int, int]()
		n := 1000
		workers := runtime.GOMAXPROCS(0) * 2
		for range workers {
			go func() {
				for i := range n {
					m.PutIfAbsent(i, i)
				}
			}()
		}
		synctest.Wait()
		for _, k := range []int{0, n / 2, n - 1} {
			v, ok := m.Get(k)
			require.Truef(t, ok, "Missing key %d", k)
			require.Equal(t, k, v, "Returned value should match expected")
		}
		// Atomics
		v, computed := m.GetOrCompute(n+1, func() int { return 42 })
		require.True(t, computed, "GetOrCompute should compute for missing key")
		require.Equal(t, 42, v, "Returned value should match expected")
		// CAS
		require.False(t, m.CompareAndSwap(n+2, 0, 1, eqV[int]), "CompareAndSwap should be false on mismatch")
		m.Put(n+2, 10)
		require.True(t, m.CompareAndSwap(n+2, 10, 11, eqV[int]), "CompareAndSwap should succeed")
		require.True(t, m.CompareAndDelete(n+2, 11, eqV[int]), "CompareAndDelete should succeed")
		_, ok := m.Get(n + 2)
		require.False(t, ok, "Key should be deleted")
	})
}

func TestConcurrentSkipMap_CompareAndDeleteNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")

	// CompareAndDelete with non-existing key
	require.False(t, m.CompareAndDelete(99, "value", eqV[string]), "CompareAndDelete should fail for non-existing key")

	// CompareAndDelete with existing key but wrong value
	require.False(t, m.CompareAndDelete(1, "wrong_value", eqV[string]), "CompareAndDelete should fail when value doesn't match")

	// CompareAndDelete with existing key and matching value
	require.True(t, m.CompareAndDelete(1, "a", eqV[string]), "CompareAndDelete should succeed when value matches")

	// Verify key is deleted
	_, ok := m.Get(1)
	require.False(t, ok, "Key should be deleted after CompareAndDelete")
}

func TestConcurrentSkipMap_Navigation(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i*10, i)
	}
	k, ok := m.FirstKey()
	require.True(t, ok, "FirstKey should succeed on non-empty map")
	require.Equal(t, 10, k, "Returned value should match expected")
	k, ok = m.LastKey()
	require.True(t, ok, "LastKey should succeed on non-empty map")
	require.Equal(t, 50, k, "Returned value should match expected")
	k, ok = m.FloorKey(33)
	require.True(t, ok, "FloorKey should find 30 for key 33")
	require.Equal(t, 30, k, "Returned value should match expected")
	k, ok = m.CeilingKey(33)
	require.True(t, ok, "CeilingKey should find 40 for key 33")
	require.Equal(t, 40, k, "Returned value should match expected")
	_, ok = m.LowerKey(10)
	require.False(t, ok, "LowerKey(10) should be none (no element < 10)")
	_, ok = m.HigherKey(50)
	require.False(t, ok, "HigherKey(50) should be none (no element > 50)")
	// Test with element removed
	m.Remove(10)
	_, ok = m.LowerKey(20)
	require.False(t, ok, "LowerKey(20) should be none after removing 10")
}

func TestConcurrentSkipMap_RangeRankAndViews(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
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
	require.True(t, slices.Equal(rs, []int{20, 30}), "Range=%v", rs)
	// RangeSeq
	r2 := make([]int, 0)
	for k := range m.RangeSeq(15, 35) {
		r2 = append(r2, k)
	}
	require.True(t, slices.Equal(r2, []int{20, 30}), "RangeSeq=%v", r2)
	// Rank
	require.Equal(t, 2, m.RankOfKey(30), "Rank should match expected")
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

func TestConcurrentSkipMap_ClearAndClone(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMapFrom(map[int]string{1: "a", 2: "b", 3: "c"})
	require.False(t, m.IsEmpty(), "IsEmpty should be false")
	require.Equal(t, 3, m.Size(), "Size should be 3 before Clear")
	// Clone
	clone := m.Clone()
	clone.Put(4, "d")
	require.Equal(t, 3, m.Size(), "Size should remain 3 after clone mutation")
	require.Equal(t, 4, clone.Size(), "Clone size should match expected")
	// CloneSorted
	sortedClone := m.CloneSorted()
	require.Equal(t, 3, sortedClone.Size(), "Sorted clone size should match expected")
	// Clear
	m.Clear()
	require.True(t, m.IsEmpty(), "IsEmpty should be true")
	require.Equal(t, 0, m.Size(), "Size should be 0 after Clear")
}

func TestConcurrentSkipMap_PutAllAndPutSeq(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	other := NewConcurrentSkipMap[int, int]()
	other.Put(1, 10)
	other.Put(2, 20)
	m.PutAll(other)
	require.Equal(t, 2, m.Size(), "Size should be 2 after PutAll")
	// PutSeq
	m.PutSeq(func(yield func(int, int) bool) {
		if !yield(3, 30) {
			return
		}
		yield(4, 40)
	})
	require.Equal(t, 4, m.Size(), "Size should be 4 after PutSeq")
}

func TestConcurrentSkipMap_RemoveAllAndFunc(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	// RemoveAll
	removed := m.RemoveAll(1, 2)
	require.Equal(t, 2, removed, "Removed count should match expected")
	require.Equal(t, 1, m.Size(), "Size should be 1 after RemoveAll")
	// RemoveFunc
	m.Put(4, "d")
	m.Put(5, "e")
	count := m.RemoveFunc(func(k int, v string) bool { return k > 3 })
	require.Equal(t, 2, count, "Count should match expected")
	require.Equal(t, 1, m.Size(), "Size should be 1 after RemoveFunc")
}

func TestConcurrentSkipMap_ComputeAndMerge(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "one")
	// Compute
	v, ok := m.Compute(1, func(k int, old string, exists bool) (string, bool) {
		if exists {
			return old + "_updated", true
		}
		return "new", true
	})
	require.True(t, ok, "Compute should update when key exists and keep=true")
	require.Equal(t, "one_updated", v, "Compute should return updated value")
	// ComputeIfAbsent
	v = m.ComputeIfAbsent(2, func(k int) string { return "two" })
	require.Equal(t, "two", v, "ComputeIfAbsent should return inserted value")
	v = m.ComputeIfAbsent(2, func(k int) string { return "two_again" })
	require.Equal(t, "two", v, "ComputeIfAbsent should return existing value") // should not change
	// ComputeIfPresent
	v, ok = m.ComputeIfPresent(1, func(k int, old string) (string, bool) {
		return old + "_modified", true
	})
	require.True(t, ok, "ComputeIfPresent should update and return true for existing key")
	require.Equal(t, "one_updated_modified", v, "ComputeIfPresent should return updated value")
	// Merge
	v, ok = m.Merge(3, "three", func(old, new string) (string, bool) {
		return old + "_" + new, true
	})
	require.True(t, ok, "Merge should insert and return true for missing key")
	require.Equal(t, "three", v, "Merge should return inserted value for missing key")
	v, ok = m.Merge(1, "one", func(old, new string) (string, bool) {
		return old + "_merged", true
	})
	require.True(t, ok, "Merge should update and return true for existing key")
	require.Equal(t, "one_updated_modified_merged", v, "Merge should return merged value for existing key")
}

func TestConcurrentSkipMap_ComputeNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "one")

	// Compute on non-existing key (exists=false, keep=false) -> should do nothing
	_, ok := m.Compute(99, func(k int, old string, exists bool) (string, bool) {
		return "should_not_be_added", false
	})
	require.False(t, ok, "Compute should return false when key doesn't exist and keep=false")
	require.False(t, m.ContainsKey(99), "New key should not be added when keep=false")

	// Compute on non-existing key (exists=false, keep=true) -> should add
	v, ok := m.Compute(99, func(k int, old string, exists bool) (string, bool) {
		return "new_value", true
	})
	require.True(t, ok, "Compute should return true when key is added")
	require.Equal(t, "new_value", v, "Compute should return new value")
	require.True(t, m.ContainsKey(99), "Key should be added")
}

func TestConcurrentSkipMap_ComputeIfPresentNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "one")

	// ComputeIfPresent on non-existing key -> should return false
	_, ok := m.ComputeIfPresent(99, func(k int, old string) (string, bool) {
		return "new", true
	})
	require.False(t, ok, "ComputeIfPresent should return false for non-existing key")

	// ComputeIfPresent with keep=false -> should remove key
	m.Put(2, "two")
	_, ok = m.ComputeIfPresent(2, func(k int, old string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "ComputeIfPresent should return false when keep=false")
	require.False(t, m.ContainsKey(2), "Key should be removed when keep=false")
}

func TestConcurrentSkipMap_ReplaceOperations(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	// Replace
	v, ok := m.Replace(1, "aa")
	require.True(t, ok, "Replace should succeed for existing key")
	require.Equal(t, "a", v, "Replace should return previous value")
	v, _ = m.Get(1)
	require.Equal(t, "aa", v, "Replace should update stored value")
	// ReplaceIf
	require.True(t, m.ReplaceIf(1, "aa", "aaa", eqV[string]), "ReplaceIf should succeed when current value matches")
	require.False(t, m.ReplaceIf(1, "wrong", "xxx", eqV[string]), "ReplaceIf should be false when current value mismatches")
	// ReplaceAll
	m.ReplaceAll(func(k int, v string) string { return v + "_replaced" })
	v, _ = m.Get(1)
	require.Equal(t, "aaa_replaced", v, "ReplaceAll should update stored value")

	// Replace non-existing key
	_, ok = m.Replace(99, "new")
	require.False(t, ok, "Replace should return false for non-existing key")
}

func TestConcurrentSkipMap_PopFirstLast(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	m.Put(10, 1)
	m.Put(20, 2)
	m.Put(30, 3)
	e, ok := m.PopFirst()
	require.True(t, ok, "PopFirst should succeed")
	require.Equal(t, 10, e.Key, "Key should match expected")
	require.Equal(t, 1, e.Value, "Entry value should match expected")
	e, ok = m.PopLast()
	require.True(t, ok, "PopLast should succeed")
	require.Equal(t, 30, e.Key, "Key should match expected")
	require.Equal(t, 3, e.Value, "Entry value should match expected")
	require.Equal(t, 1, m.Size(), "Size should be 1 after PopFirst and PopLast")

	// Pop from empty map
	m.Clear()
	_, ok = m.PopFirst()
	require.False(t, ok, "PopFirst should fail on empty map")
	_, ok = m.PopLast()
	require.False(t, ok, "PopLast should fail on empty map")
}

func TestConcurrentSkipMap_RangeFromAndTo(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i, i*10)
	}
	// RangeFrom
	rf := make([]int, 0)
	m.RangeFrom(3, func(k, v int) bool {
		rf = append(rf, k)
		return true
	})
	require.Equal(t, []int{3, 4, 5}, rf, "Slice should match expected")
	// RangeTo
	rt := make([]int, 0)
	m.RangeTo(3, func(k, v int) bool {
		rt = append(rt, k)
		return true
	})
	require.Equal(t, []int{1, 2, 3}, rt, "Slice should match expected")
}

func TestConcurrentSkipMap_AscendDescend(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
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
	require.Equal(t, []int{1, 2, 3, 4, 5}, asc, "Slice should match expected")
	// Descend
	desc := make([]int, 0)
	m.Descend(func(k, v int) bool {
		desc = append(desc, k)
		return true
	})
	require.Equal(t, []int{5, 4, 3, 2, 1}, desc, "Slice should match expected")
	// AscendFrom
	af := make([]int, 0)
	m.AscendFrom(3, func(k, v int) bool {
		af = append(af, k)
		return true
	})
	require.Equal(t, []int{3, 4, 5}, af, "Slice should match expected")
	// DescendFrom
	df := make([]int, 0)
	m.DescendFrom(3, func(k, v int) bool {
		df = append(df, k)
		return true
	})
	require.Equal(t, []int{3, 2, 1}, df, "Slice should match expected")
}

func TestConcurrentSkipMap_EntryNavigation(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i*10, i)
	}
	e, ok := m.FloorEntry(35)
	require.True(t, ok, "FloorEntry should succeed")
	require.Equal(t, 30, e.Key, "Key should match expected")
	e, ok = m.CeilingEntry(35)
	require.True(t, ok, "CeilingEntry should succeed")
	require.Equal(t, 40, e.Key, "Key should match expected")
	e, ok = m.LowerEntry(25)
	require.True(t, ok, "LowerEntry should succeed")
	require.Equal(t, 20, e.Key, "Key should match expected")
	e, ok = m.HigherEntry(25)
	require.True(t, ok, "HigherEntry should succeed")
	require.Equal(t, 30, e.Key, "Key should match expected")
}

func TestConcurrentSkipMap_FilterAndEquals(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	// Filter
	filtered := m.Filter(func(k int, v string) bool { return k > 1 })
	require.Equal(t, 2, filtered.Size(), "Filter should keep two keys greater than 1")
	// Equals
	m2 := NewConcurrentSkipMap[int, string]()
	m2.Put(1, "a")
	m2.Put(2, "b")
	m2.Put(3, "c")
	require.True(t, m.Equals(m2, eqV[string]), "Equals should be true")
	m3 := NewConcurrentSkipMap[int, string]()
	m3.Put(1, "a")
	m3.Put(2, "b")
	m3.Put(3, "different")
	require.False(t, m.Equals(m3, eqV[string]), "Equals should be false")
}

func TestConcurrentSkipMap_Races(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		m := NewConcurrentSkipMap[int, int]()
		workers := runtime.GOMAXPROCS(0) * 2
		iters := 500
		for w := range workers {
			go func(id int) {
				for i := range iters {
					k := (id * 131) + i
					switch i % 4 {
					case 0:
						m.Put(k, i)
					case 1:
						m.Remove(k)
					case 2:
						m.Get(k)
					default:
						m.ReplaceAll(func(key, val int) int { return val })
					}
					if i%100 == 0 {
						count := 0
						m.Range(-1<<31, 1<<31-1, func(k, v int) bool {
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

func TestConcurrentSkipMap_ContainsValueAndRemoveIf(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "a")
	require.True(t, m.ContainsValue("a", eqV[string]), "ContainsValue should find existing value")
	require.False(t, m.ContainsValue("z", eqV[string]), "ContainsValue should be false for missing value")
	// RemoveIf
	require.True(t, m.RemoveIf(2, "b", eqV[string]), "RemoveIf should remove when key/value match")
	require.False(t, m.RemoveIf(2, "wrong", eqV[string]), "RemoveIf should be false when value mismatches")
	require.Equal(t, 2, m.Size(), "Size should be 2 after RemoveIf")
}

func TestConcurrentSkipMap_RemoveAndGet(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	v, ok := m.RemoveAndGet(1)
	require.True(t, ok, "RemoveAndGet should succeed for present key")
	require.Equal(t, "a", v, "RemoveAndGet should return removed value")
	_, ok = m.Get(1)
	require.False(t, ok, "Get should fail after RemoveAndGet removed the key")

	// RemoveAndGet on non-existing key
	_, ok = m.RemoveAndGet(99)
	require.False(t, ok, "RemoveAndGet should fail for non-existing key")
}

func TestConcurrentSkipMap_RemoveSeq(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	m.Put(4, "d")
	m.Put(5, "e")
	removed := m.RemoveSeq(func(yield func(int) bool) {
		if !yield(1) {
			return
		}
		if !yield(3) {
			return
		}
		if !yield(5) {
			return
		}
	})
	require.Equal(t, 3, removed, "Removed count should match expected")
	require.Equal(t, 2, m.Size(), "Size should be 2 after RemoveSeq")
}

func TestConcurrentSkipMap_CoverageSupplement(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// ForEach
	cnt := 0
	m.ForEach(func(k int, v string) bool {
		cnt++
		return true
	})
	require.Equal(t, 2, cnt, "Count should match expected")

	// SeqKeys, SeqValues
	kc := 0
	for range m.SeqKeys() {
		kc++
	}
	require.Equal(t, 2, kc, "Returned value should match expected")

	vc := 0
	for range m.SeqValues() {
		vc++
	}
	require.Equal(t, 2, vc, "Returned value should match expected")

	// String
	str := m.String()
	require.Contains(t, str, "concurrentSkipMap", "String should include type name")
}

func TestConcurrentSkipMap_GetOrDefault(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")

	require.Equal(t, "a", m.GetOrDefault(1, "default"), "GetOrDefault should return stored value")
	require.Equal(t, "default", m.GetOrDefault(99, "default"), "GetOrDefault should return default for missing key")
}

func TestConcurrentSkipMap_EmptyNavigation(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()

	_, ok := m.FirstKey()
	require.False(t, ok, "FirstKey should fail on empty map")

	_, ok = m.LastKey()
	require.False(t, ok, "LastKey should fail on empty map")

	_, ok = m.FirstEntry()
	require.False(t, ok, "FirstEntry should fail on empty map")

	_, ok = m.LastEntry()
	require.False(t, ok, "LastEntry should fail on empty map")

	_, ok = m.FloorKey(10)
	require.False(t, ok, "FloorKey should fail on empty map")

	_, ok = m.CeilingKey(10)
	require.False(t, ok, "CeilingKey should fail on empty map")
}

func TestConcurrentSkipMap_Values(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(3, "c")
	m.Put(1, "a")
	m.Put(2, "b")

	vals := m.Values()
	require.Equal(t, 3, len(vals), "Values should return all values")
	require.Equal(t, []string{"a", "b", "c"}, vals, "Values should be ordered by keys")

	// Empty map
	empty := NewConcurrentSkipMap[int, string]()
	require.Equal(t, 0, len(empty.Values()), "Empty map should return empty slice")
}

func TestConcurrentSkipMap_MergeRemove(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "old")

	// Merge with keep=false should remove the key
	v, ok := m.Merge(1, "new", func(old, new string) (string, bool) {
		return "", false // keep=false
	})
	require.False(t, ok, "Merge should return false when keep=false")
	require.Equal(t, "", v, "Merge should return zero value when keep=false")
	require.False(t, m.ContainsKey(1), "Key should be removed when Merge returns keep=false")

	// Merge on non-existing key with keep=false (should insert)
	v, ok = m.Merge(2, "value", func(old, new string) (string, bool) {
		return "", false
	})
	require.True(t, ok, "Merge should insert value for non-existing key")
	require.Equal(t, "value", v, "Merge should return inserted value")
	require.True(t, m.ContainsKey(2), "Key should exist after Merge")
}

func TestConcurrentSkipMap_EntryNavigationNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	m.Put(10, 1)
	m.Put(30, 3)

	// FloorEntry - no floor exists
	_, ok := m.FloorEntry(5)
	require.False(t, ok, "FloorEntry should fail when no key <= target")

	// CeilingEntry - no ceiling exists
	_, ok = m.CeilingEntry(40)
	require.False(t, ok, "CeilingEntry should fail when no key >= target")

	// LowerEntry - no lower exists
	_, ok = m.LowerEntry(10)
	require.False(t, ok, "LowerEntry should fail when no key < target")

	// HigherEntry - no higher exists
	_, ok = m.HigherEntry(30)
	require.False(t, ok, "HigherEntry should fail when no key > target")
}

func TestConcurrentSkipMap_RangeSeqEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// RangeSeq with early exit
	collected := make([]int, 0)
	for k := range m.RangeSeq(2, 8) {
		collected = append(collected, k)
		if k >= 5 {
			break // early exit
		}
	}
	require.Equal(t, []int{2, 3, 4, 5}, collected, "RangeSeq should support early exit")
}

func TestConcurrentSkipMap_DescendEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	m.Put(1, 10)
	m.Put(2, 20)
	m.Put(3, 30)
	m.Put(4, 40)

	// Descend with early exit
	collected := make([]int, 0)
	m.Descend(func(k, v int) bool {
		collected = append(collected, k)
		return k > 2 // stop when k <= 2
	})
	require.Equal(t, []int{4, 3, 2}, collected, "Descend should support early exit")
}

func TestConcurrentSkipMap_GetByRankOutOfBounds(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// Valid rank
	e, ok := m.GetByRank(1)
	require.True(t, ok, "GetByRank should succeed for valid rank")
	require.Equal(t, 2, e.Key, "GetByRank(1) should return second element")

	// Out of bounds rank
	_, ok = m.GetByRank(5)
	require.False(t, ok, "GetByRank should fail for rank >= size")

	// Negative rank already tested, but let's verify
	_, ok = m.GetByRank(-1)
	require.False(t, ok, "GetByRank should fail for negative rank")
}

func TestConcurrentSkipMap_ComputeKeepFalse(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	m.Put(1, "old")

	// Compute with keep=false on existing key (should remove)
	v, ok := m.Compute(1, func(k int, old string, exists bool) (string, bool) {
		require.True(t, exists, "Key should exist")
		require.Equal(t, "old", old, "Old value should match")
		return "", false // keep=false
	})
	require.False(t, ok, "Compute should return false when keep=false")
	require.Equal(t, "", v, "Compute should return zero value when keep=false")
	require.False(t, m.ContainsKey(1), "Key should be removed when Compute returns keep=false")

	// Verify size decreased
	require.Equal(t, 0, m.Size(), "Size should be 0 after removal")
}

func TestConcurrentSkipMap_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// Seq with early exit
	collected := make([]int, 0)
	for k := range m.Seq() {
		collected = append(collected, k)
		if k >= 5 {
			break
		}
	}
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected, "Seq should support early exit")
}

func TestConcurrentSkipMap_SeqKeysEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// SeqKeys with early exit
	collected := make([]int, 0)
	for k := range m.SeqKeys() {
		collected = append(collected, k)
		if k >= 5 {
			break
		}
	}
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected, "SeqKeys should support early exit")
}

func TestConcurrentSkipMap_SeqValuesEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i, i*10)
	}

	// SeqValues with early exit
	collected := make([]int, 0)
	for v := range m.SeqValues() {
		collected = append(collected, v)
		if len(collected) >= 3 {
			break
		}
	}
	require.Equal(t, 3, len(collected), "SeqValues should support early exit")
}

func TestConcurrentSkipMap_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// Reversed with early exit
	collected := make([]int, 0)
	for k := range m.Reversed() {
		collected = append(collected, k)
		if k <= 6 {
			break
		}
	}
	require.Equal(t, []int{10, 9, 8, 7, 6}, collected, "Reversed should support early exit")
}

func TestConcurrentSkipMap_FoarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// ForEach with early exit
	collected := make([]int, 0)
	m.ForEach(func(k, v int) bool {
		collected = append(collected, k)
		return k < 5 // stop when k >= 5
	})
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected, "ForEach should support early exit")
}

func TestConcurrentSkipMap_IsNotEmpty(t *testing.T) {
	t.Parallel()
	m := NewConcurrentSkipMap[int, string]()
	require.False(t, m.IsNotEmpty(), "new map should not be non-empty")
	m.Put(1, "a")
	require.True(t, m.IsNotEmpty(), "map with entry should be non-empty")
}
