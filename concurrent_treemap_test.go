package collections

import (
	"cmp"
	"runtime"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/require"
)

func TestConcurrentTreeMap_BasicOrdered(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(2, "b")
	m.Put(1, "a")
	m.Put(3, "c")
	require.True(t, slices.Equal(m.Keys(), []int{1, 2, 3}), "Keys=%v", m.Keys())
	rev := make([]int, 0)
	for k := range m.Reversed() {
		rev = append(rev, k)
	}
	require.True(t, slices.Equal(rev, []int{3, 2, 1}), "Reversed=%v", rev)
	e, ok := m.FirstEntry()
	require.True(t, ok, "FirstEntry should succeed on non-empty map")
	require.Equal(t, 1, e.Key, "Key should match expected")
	require.Equal(t, "a", e.Value, "FirstEntry should return value for smallest key")
	e, ok = m.LastEntry()
	require.True(t, ok, "LastEntry should succeed on non-empty map")
	require.Equal(t, 3, e.Key, "Key should match expected")
	require.Equal(t, "c", e.Value, "LastEntry should return value for largest key")
}

func TestConcurrentTreeMap_CustomComparator(t *testing.T) {
	t.Parallel()
	cmpRev := func(a, b int) int { return -cmp.Compare(a, b) }
	m := NewConcurrentTreeMap[int, int](cmpRev)
	for i := 1; i <= 3; i++ {
		m.Put(i, i)
	}
	require.True(t, slices.Equal(m.Keys(), []int{3, 2, 1}), "Keys with reverse comparator=%v", m.Keys())
}

func TestConcurrentTreeMap_ConcurrentPutIfAbsentAndAtomics(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		m := NewConcurrentTreeMapOrdered[int, int]()
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
		// first CAS should fail (no key)
		require.False(t, m.CompareAndSwap(n+2, 0, 1, eqV[int]), "CompareAndSwap should be false on mismatch")
		m.Put(n+2, 10)
		require.True(t, m.CompareAndSwap(n+2, 10, 11, eqV[int]), "CompareAndSwap should succeed")
		require.True(t, m.CompareAndDelete(n+2, 11, eqV[int]), "CompareAndDelete should succeed")
		_, ok := m.Get(n + 2)
		require.False(t, ok, "Key should be deleted")

		// RemoveAndGet
		m.Put(n+3, 123)
		v, ok = m.RemoveAndGet(n + 3)
		require.True(t, ok, "RemoveAndGet should succeed for present key")
		require.Equal(t, 123, v, "Returned value should match expected")
		_, ok = m.Get(n + 3)
		require.False(t, ok, "Get should fail after RemoveAndGet removed the key")
	})
}

// Race test colocated with concurrent tree map tests.
func TestConcurrentTreeMap_Races(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(_ *testing.T) {
		m := NewConcurrentTreeMapOrdered[int, int]()
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
						m.ReplaceAll(func(_, val int) int { return val })
					}
					if i%100 == 0 {
						count := 0
						m.Range(-1<<31, 1<<31-1, func(_, _ int) bool {
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

func TestConcurrentTreeMap_NavigationAndRange(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 5; i++ {
		m.Put(i*10, i)
	}
	k, ok := m.FirstKey()
	require.True(t, ok, "FirstKey should succeed on non-empty map")
	require.Equal(t, 10, k, "Returned value should match expected")
	k, ok = m.LastKey()
	require.True(t, ok, "LastKey should succeed on non-empty map")
	require.Equal(t, 50, k, "Returned value should match expected")
	k, ok = m.FloorKey(35)
	require.True(t, ok, "FloorKey should find 30 for key 35")
	require.Equal(t, 30, k, "Returned value should match expected")
	k, ok = m.CeilingKey(35)
	require.True(t, ok, "CeilingKey should find 40 for key 35")
	require.Equal(t, 40, k, "Returned value should match expected")
	k, ok = m.LowerKey(30)
	require.True(t, ok, "LowerKey should succeed for key 30")
	require.Equal(t, 20, k, "Returned value should match expected")
	k, ok = m.HigherKey(30)
	require.True(t, ok, "HigherKey should succeed for key 30")
	require.Equal(t, 40, k, "Returned value should match expected")

	e, ok := m.FloorEntry(35)
	require.True(t, ok, "FloorEntry should succeed")
	require.Equal(t, 30, e.Key, "Key should match expected")
	e, ok = m.CeilingEntry(35)
	require.True(t, ok, "CeilingEntry should succeed")
	require.Equal(t, 40, e.Key, "Key should match expected")
	e, ok = m.LowerEntry(30)
	require.True(t, ok, "LowerEntry should succeed")
	require.Equal(t, 20, e.Key, "Key should match expected")
	e, ok = m.HigherEntry(30)
	require.True(t, ok, "HigherEntry should succeed")
	require.Equal(t, 40, e.Key, "Key should match expected")

	// Range
	rs := make([]int, 0)
	m.Range(15, 45, func(k, _ int) bool {
		rs = append(rs, k)
		return true
	})
	require.True(t, slices.Equal(rs, []int{20, 30, 40}), "Range=%v", rs)
	// Rank and GetByRank
	require.Equal(t, 2, m.RankOfKey(30), "Rank should match expected")
	e, ok = m.GetByRank(1)
	require.True(t, ok, "GetByRank should succeed for valid rank")
	require.Equal(t, 20, e.Key, "Key should match expected")
}

func TestConcurrentTreeMap_Views(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	m.Put(10, 1)
	m.Put(20, 2)
	m.Put(30, 3)
	m.Put(40, 4)
	sub := m.SubMap(15, 35).Keys()
	head := m.HeadMap(25, false).Keys()
	tail := m.TailMap(25, true).Keys()
	require.True(t, slices.Equal(sub, []int{20, 30}), "SubMap=%v", sub)
	require.True(t, slices.Equal(head, []int{10, 20}), "HeadMap=%v", head)
	require.True(t, slices.Equal(tail, []int{30, 40}), "TailMap=%v", tail)
}

func TestConcurrentTreeMap_ClearAndClone(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapFrom(cmp.Compare, map[int]string{1: "a", 2: "b", 3: "c"})
	require.False(t, m.IsEmpty(), "IsEmpty should be false")
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

func TestConcurrentTreeMap_PutAllAndPutSeq(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	m.Put(1, 10)
	m2 := NewConcurrentTreeMapOrdered[int, int]()
	m2.Put(1, 100)
	m2.Put(2, 200)
	m.PutAll(m2)
	require.Equal(t, 2, m.Size(), "Size should be 2 after PutAll")
	v, _ := m.Get(1)
	require.Equal(t, 100, v, "Returned value should match expected")
	// PutSeq
	m.PutSeq(func(yield func(int, int) bool) {
		if !yield(3, 300) {
			return
		}
		yield(4, 400)
	})
	require.Equal(t, 4, m.Size(), "Size should be 4 after PutSeq")
}

func TestConcurrentTreeMap_ComputeAndMerge(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	v, ok := m.Compute(1, func(_ int, old string, exists bool) (string, bool) {
		if exists {
			return old + "_updated", true
		}
		return "new", true
	})
	require.True(t, ok, "Compute should update when key exists and keep=true")
	require.Equal(t, "a_updated", v, "Compute should return updated value")
	// ComputeIfAbsent
	v = m.ComputeIfAbsent(2, func(_ int) string { return "b" })
	require.Equal(t, "b", v, "ComputeIfAbsent should return inserted value")
	// ComputeIfPresent
	v, ok = m.ComputeIfPresent(1, func(_ int, old string) (string, bool) {
		return old + "_again", true
	})
	require.True(t, ok, "ComputeIfPresent should update and return true for existing key")
	require.Equal(t, "a_updated_again", v, "ComputeIfPresent should return updated value")

	// Merge
	v, ok = m.Merge(3, "c", func(old, newValue string) (string, bool) {
		return old + "_" + newValue, true
	})
	require.True(t, ok, "Merge should insert and return true for missing key")
	require.Equal(t, "c", v, "Merge should return inserted value for missing key")
}

func TestConcurrentTreeMap_ReplaceOperations(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	old, ok := m.Replace(1, "aa")
	require.True(t, ok, "Replace should succeed for existing key")
	require.Equal(t, "a", old, "Replace should return previous value")
	v, _ := m.Get(1)
	require.Equal(t, "aa", v, "Replace should update stored value")
	// ReplaceIf
	require.True(t, m.ReplaceIf(1, "aa", "aaa", eqV[string]), "ReplaceIf should succeed when current value matches")
	require.False(t, m.ReplaceIf(1, "wrong", "xxx", eqV[string]), "ReplaceIf should be false when current value mismatches")
	// ReplaceAll
	m.ReplaceAll(func(_ int, v string) string { return v + "!" })
	v, _ = m.Get(1)
	require.Equal(t, "aaa!", v, "ReplaceAll should update stored value")
}

func TestConcurrentTreeMap_PopFirstLast(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
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
}

func TestConcurrentTreeMap_FilterAndEquals(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	filtered := m.Filter(func(k int, _ string) bool { return k > 1 })
	require.Equal(t, 2, filtered.Size(), "Filter should keep two keys greater than 1")
	// Equals
	m2 := NewConcurrentTreeMapOrdered[int, string]()
	m2.Put(1, "a")
	m2.Put(2, "b")
	m2.Put(3, "c")
	require.True(t, m.Equals(m2, eqV[string]), "Equals should be true")
}

func TestConcurrentTreeMap_ForEachEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")

	// ForEach with early exit (return false)
	count := 0
	m.ForEach(func(_ int, _ string) bool {
		count++
		return false // Early exit after first iteration
	})
	require.Equal(t, 1, count, "ForEach should stop when callback returns false")
}

func TestConcurrentTreeMap_AscendScenarios(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	m.Put(4, "d")
	m.Put(5, "e")

	// Ascend iterates all elements in ascending key order
	allKeys := []int{}
	m.Ascend(func(k int, _ string) bool {
		allKeys = append(allKeys, k)
		return true
	})
	require.Equal(t, []int{1, 2, 3, 4, 5}, allKeys, "Ascend should iterate all elements in ascending order")

	// Ascend from key 3
	ascKeys := []int{}
	m.AscendFrom(3, func(k int, _ string) bool {
		ascKeys = append(ascKeys, k)
		return true
	})
	require.Equal(t, []int{3, 4, 5}, ascKeys, "AscendFrom should include starting key and keys greater than it")

	// Ascend on empty map
	emptyMap := NewConcurrentTreeMapOrdered[int, string]()
	ascCount := 0
	emptyMap.Ascend(func(_ int, _ string) bool {
		ascCount++
		return true
	})
	require.Equal(t, 0, ascCount, "Ascend on empty map should not call callback")

	// Ascend with early termination
	earlyCount := 0
	m.Ascend(func(_ int, _ string) bool {
		earlyCount++
		return earlyCount < 3 // Stop after 3 iterations
	})
	require.Equal(t, 3, earlyCount, "Ascend should stop when action returns false")
}

func TestConcurrentTreeMap_CoverageSupplement(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// GetOrDefault
	require.Equal(t, "a", m.GetOrDefault(1, "x"), "GetOrDefault should return stored value")
	require.Equal(t, "x", m.GetOrDefault(99, "x"), "GetOrDefault should return fallback for missing key")

	// RemoveIf
	require.True(t, m.RemoveIf(1, "a", eqV[string]), "RemoveIf should remove when key/value match")
	require.False(t, m.RemoveIf(1, "a", eqV[string]), "RemoveIf should be false after entry is removed")
	require.False(t, m.RemoveIf(2, "wrong", eqV[string]), "RemoveIf should be false when value mismatches")

	// ContainsKey/Value
	require.True(t, m.ContainsKey(2), "ContainsKey should be true for present key")
	require.False(t, m.ContainsKey(1), "Should not contain element")
	require.True(t, m.ContainsValue("b", eqV[string]), "ContainsValue should find existing value")
	require.False(t, m.ContainsValue("z", eqV[string]), "ContainsValue should be false for missing value")

	// RemoveAll
	m.Put(3, "c")
	m.Put(4, "d")
	// keys: 2, 3, 4
	require.Equal(t, 2, m.RemoveAll(2, 3), "Returned value should match expected")
	require.Equal(t, 1, m.Size(), "Size should be 1 after RemoveAll")

	// RemoveSeq
	m.Put(5, "e")
	removed := m.RemoveSeq(func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		yield(99)
	})
	require.Equal(t, 1, removed, "Removed count should match expected")

	// RemoveFunc
	m.Put(6, "f")
	removed = m.RemoveFunc(func(k int, _ string) bool { return k == 6 })
	require.Equal(t, 1, removed, "Removed count should match expected")

	// Iteration
	m.Put(10, "ten")
	m.Put(20, "twenty")

	// ForEach
	cnt := 0
	m.ForEach(func(_ int, _ string) bool {
		cnt++
		return true
	})
	require.Equal(t, 3, cnt, "Count should match expected")

	// SeqKeys, SeqValues
	kc := 0
	for range m.SeqKeys() {
		kc++
	}
	require.Equal(t, 3, kc, "Returned value should match expected")

	vc := 0
	for range m.SeqValues() {
		vc++
	}
	require.Equal(t, 3, vc, "Returned value should match expected")

	// String
	str := m.String()
	require.Contains(t, str, "concurrentTreeMap", "String should include type name")

	// LowerKey, HigherKey
	m.Put(1, "one")
	m.Put(3, "three")
	// keys: 1, 5, 10, 20 (Wait, 5, 10, 20 were remaining? Yes)
	// keys currently: 5, 10, 20. +1, +3 -> 1, 3, 5, 10, 20

	lk, lok := m.LowerKey(4)
	require.True(t, lok, "LowerKey should succeed for existing lower key")
	require.Equal(t, 3, lk, "Returned value should match expected")
	hk, hok := m.HigherKey(4)
	require.True(t, hok, "HigherKey should succeed for existing higher key")
	require.Equal(t, 5, hk, "Returned value should match expected")

	// FloorEntry, CeilingEntry, LowerEntry, HigherEntry
	fe, fok := m.FloorEntry(4)
	require.True(t, fok, "FloorEntry should succeed")
	require.Equal(t, 3, fe.Key, "Key should match expected")

	// Ascend, Descend
	ascCnt := 0
	m.Ascend(func(_ int, _ string) bool {
		ascCnt++
		return true
	})
	require.Equal(t, 5, ascCnt, "Sequence should match expected") // 1, 3, 5, 10, 20
	descCnt := 0
	m.Descend(func(_ int, _ string) bool {
		descCnt++
		return true
	})
	require.Equal(t, 5, descCnt, "Sequence should match expected")

	// AscendFrom, DescendFrom
	m.AscendFrom(5, func(_ int, _ string) bool { return true })
	m.DescendFrom(5, func(_ int, _ string) bool { return true })

	// RangeSeq, RangeFrom, RangeTo
	for range m.RangeSeq(1, 100) {
	}
	m.RangeFrom(1, func(_ int, _ string) bool { return true })
	m.RangeTo(100, func(_ int, _ string) bool { return true })

	// Reversed
	for range m.Reversed() {
	}

	// CloneSorted
	cs := m.CloneSorted()
	require.Equal(t, m.Size(), cs.Size(), "CloneSorted size should match source size")

	// Merge
	m.Put(25, "a")
	// Merge existing
	v, ok := m.Merge(25, "b", func(old, newValue string) (string, bool) {
		return old + newValue, true
	})
	require.True(t, ok, "Merge should update and return true for existing key")
	require.Equal(t, "ab", v, "Merge should return merged value for existing key")
	// Merge existing -> remove
	_, ok = m.Merge(25, "c", func(_, _ string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "Merge should return false when function indicates removal")
	require.False(t, m.ContainsKey(25), "Should not contain element")
	// Merge absent
	v, ok = m.Merge(30, "new", func(_, newValue string) (string, bool) {
		return newValue, true // shouldn't be called
	})
	require.True(t, ok, "Merge should insert and return true for missing key")
	require.Equal(t, "new", v, "Merge should return new value for missing key")

	// ComputeIfPresent
	m.Put(10, "ten")
	// Present -> update
	v, ok = m.ComputeIfPresent(10, func(_ int, old string) (string, bool) {
		return old + "_updated", true
	})
	require.True(t, ok, "ComputeIfPresent should update and return true for existing key")
	require.Equal(t, "ten_updated", v, "Compute should return updated value")
	// Present -> remove
	_, ok = m.ComputeIfPresent(10, func(_ int, _ string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "ComputeIfPresent should return false when function indicates removal")
	require.False(t, m.ContainsKey(10), "Should not contain element")
	// Absent
	_, ok = m.ComputeIfPresent(999, func(_ int, _ string) (string, bool) {
		return "new", true
	})
	require.False(t, ok, "ComputeIfPresent should be false for missing key")

	// ReplaceIf
	m.Put(40, "val")
	require.True(t, m.ReplaceIf(40, "val", "newVal", eqV[string]), "ReplaceIf should succeed when current value matches")
	require.False(t, m.ReplaceIf(40, "wrong", "xxx", eqV[string]), "ReplaceIf should be false when current value mismatches")

	// ReplaceAll
	m.ReplaceAll(func(_ int, v string) string {
		return v + "!"
	})
	val, _ := m.Get(40)
	require.Equal(t, "newVal!", val, "ReplaceAll should update stored value")
	t.Log("CoverageSupplement finished")
	// require.Fail(t, "Force fail to see log")
}

func TestConcurrentTreeMap_SeqKeysEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	for i := 1; i <= 10; i++ {
		m.Put(i, string(rune('a'+i-1)))
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

func TestConcurrentTreeMap_SeqValuesEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	m.Put(4, "d")

	// SeqValues with early exit
	collected := make([]string, 0)
	for v := range m.SeqValues() {
		collected = append(collected, v)
		if len(collected) >= 2 {
			break
		}
	}
	require.Len(t, collected, 2, "SeqValues should support early exit")
}

func TestConcurrentTreeMap_RangeFromToEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// RangeFrom with early exit
	collected := make([]int, 0)
	m.RangeFrom(3, func(k, _ int) bool {
		collected = append(collected, k)
		return k < 6 // stop when k >= 6
	})
	require.Equal(t, []int{3, 4, 5, 6}, collected, "RangeFrom should support early exit")

	// RangeTo with early exit
	collected2 := make([]int, 0)
	m.RangeTo(7, func(k, _ int) bool {
		collected2 = append(collected2, k)
		return k < 5 // stop when k >= 5
	})
	require.Equal(t, []int{1, 2, 3, 4, 5}, collected2, "RangeTo should support early exit")
}

func TestConcurrentTreeMap_DescendAscendFromEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 8; i++ {
		m.Put(i, i*10)
	}

	// Descend with early exit
	collected := make([]int, 0)
	m.Descend(func(k, _ int) bool {
		collected = append(collected, k)
		return k > 5 // stop when k <= 5
	})
	require.Equal(t, []int{8, 7, 6, 5}, collected, "Descend should support early exit")

	// AscendFrom with early exit
	collected2 := make([]int, 0)
	m.AscendFrom(3, func(k, _ int) bool {
		collected2 = append(collected2, k)
		return k < 6 // stop when k >= 6
	})
	require.Equal(t, []int{3, 4, 5, 6}, collected2, "AscendFrom should support early exit")

	// DescendFrom with early exit
	collected3 := make([]int, 0)
	m.DescendFrom(6, func(k, _ int) bool {
		collected3 = append(collected3, k)
		return k > 3 // stop when k <= 3
	})
	require.Equal(t, []int{6, 5, 4, 3}, collected3, "DescendFrom should support early exit")
}

func TestConcurrentTreeMap_ReversedEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 6; i++ {
		m.Put(i, i*10)
	}

	// Reversed with early exit
	collected := make([]int, 0)
	for k := range m.Reversed() {
		collected = append(collected, k)
		if k <= 3 {
			break
		}
	}
	require.Equal(t, []int{6, 5, 4, 3}, collected, "Reversed should support early exit")
}

func TestConcurrentTreeMap_GetOrComputeExisting(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "existing")

	// GetOrCompute when key already exists
	v, computed := m.GetOrCompute(1, func() string { return "new" })
	require.False(t, computed, "GetOrCompute should not compute for existing key")
	require.Equal(t, "existing", v, "GetOrCompute should return existing value")
}

func TestConcurrentTreeMap_CompareAndDeleteMismatch(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()
	m.Put(1, "value")

	// CompareAndDelete with mismatched value
	require.False(t, m.CompareAndDelete(1, "wrong", eqV[string]), "CompareAndDelete should fail when value mismatches")
	require.True(t, m.ContainsKey(1), "Key should still exist after failed CompareAndDelete")
}

func TestConcurrentTreeMap_EmptyMapEdgeCases(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, string]()

	// FirstEntry/LastEntry on empty map
	_, ok := m.FirstEntry()
	require.False(t, ok, "FirstEntry should return false on empty map")
	_, ok = m.LastEntry()
	require.False(t, ok, "LastEntry should return false on empty map")

	// PopFirst/PopLast on empty map
	_, ok = m.PopFirst()
	require.False(t, ok, "PopFirst should return false on empty map")
	_, ok = m.PopLast()
	require.False(t, ok, "PopLast should return false on empty map")

	// FirstKey/LastKey on empty map
	_, ok = m.FirstKey()
	require.False(t, ok, "FirstKey should return false on empty map")
	_, ok = m.LastKey()
	require.False(t, ok, "LastKey should return false on empty map")
}

func TestConcurrentTreeMap_RangeSeqEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// RangeSeq with early exit
	collected := make([]int, 0)
	for k := range m.RangeSeq(3, 8) {
		collected = append(collected, k)
		if k >= 5 {
			break
		}
	}
	require.Equal(t, []int{3, 4, 5}, collected, "RangeSeq should support early exit")
}

func TestConcurrentTreeMap_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMapOrdered[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// Seq with early exit
	collected := make([]int, 0)
	for k := range m.Seq() {
		collected = append(collected, k)
		if k >= 3 {
			break
		}
	}
	require.Equal(t, []int{1, 2, 3}, collected, "Seq should support early exit")
}

func TestConcurrentTreeMap_IsNotEmpty(t *testing.T) {
	t.Parallel()
	m := NewConcurrentTreeMap[int, string](func(a, b int) int { return a - b })
	require.False(t, m.IsNotEmpty(), "new map should not be non-empty")
	m.Put(1, "a")
	require.True(t, m.IsNotEmpty(), "map with entry should be non-empty")
}
