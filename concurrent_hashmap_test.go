package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentHashMap_Basic(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	assert.True(t, m.IsEmpty(), "New map should be empty")
	_, ok := m.Get(1)
	require.False(t, ok, "Unexpected key should not be present")
	old, replaced := m.Put(1, "a")
	require.False(t, replaced, "Put should report no replacement for new key")
	assert.Empty(t, old, "Put should return zero value for new key")
	v, ok := m.Get(1)
	require.True(t, ok, "Get should succeed for existing key")
	assert.Equal(t, "a", v, "Get should return stored value")
	v, inserted := m.PutIfAbsent(1, "x")
	require.False(t, inserted, "PutIfAbsent should report not inserted for existing key")
	assert.Equal(t, "a", v, "PutIfAbsent should return existing value")
	v, computed := m.GetOrCompute(2, func() string { return "b" })
	assert.True(t, computed, "GetOrCompute should compute on absent key")
	assert.Equal(t, "b", v, "GetOrCompute should return computed value")
	v, ok = m.RemoveAndGet(2)
	require.True(t, ok, "RemoveAndGet should succeed for present key")
	assert.Equal(t, "b", v, "RemoveAndGet should return removed value")
}

func TestConcurrentHashMap_CompareAndOps(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	m.Put(1, 10)
	// CAS success
	require.True(t, m.CompareAndSwap(1, 10, 11, eqV[int]), "CompareAndSwap should succeed")
	// CAS fail
	require.False(t, m.CompareAndSwap(1, 10, 12, eqV[int]), "CompareAndSwap should be false on mismatch")
	// CompareAndDelete success
	require.True(t, m.CompareAndDelete(1, 11, eqV[int]), "CompareAndDelete should succeed")
	_, ok := m.Get(1)
	require.False(t, ok, "Key 1 should be deleted")
	// CompareAndDelete fail
	m.Put(2, 20)
	require.False(t, m.CompareAndDelete(2, 21, eqV[int]), "CompareAndDelete should be false on mismatch")
	require.True(t, m.ContainsKey(2), "ContainsKey should be true for present key")
}

func TestConcurrentHashMap_ConcurrentPutIfAbsent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		m := NewConcurrentHashMap[int, int]()
		n := 1000
		workers := runtime.GOMAXPROCS(0) * 2
		for w := range workers {
			go func(_ int) {
				for i := range n {
					m.PutIfAbsent(i, i)
				}
			}(w)
		}
		synctest.Wait()
		// Validate presence
		for _, k := range []int{0, n / 2, n - 1} {
			v, ok := m.Get(k)
			require.Truef(t, ok, "Missing key %d", k)
			assert.Equal(t, k, v, "Value should equal key")
		}
	})
}

func TestConcurrentHashMap_ClearAndString(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMapFrom(map[int]string{1: "a", 2: "b"})
	assert.False(t, m.IsEmpty(), "IsEmpty should be false")

	// String
	str := m.String()
	assert.Contains(t, str, "concurrentHashMap", "String should include type name")
	assert.Contains(t, str, "1:a", "String should include entry values")
	assert.Contains(t, str, "2:b", "String should include entry values")

	m.Clear()
	assert.True(t, m.IsEmpty(), "IsEmpty should be true")
	assert.Equal(t, 0, m.Size(), "Size should be 0 after Clear")
}

func TestConcurrentHashMap_PutAllAndPutSeq(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	other := NewConcurrentHashMap[int, int]()
	other.Put(1, 10)
	other.Put(2, 20)
	m.PutAll(other)
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

func TestConcurrentHashMap_GetOrDefaultAndRemove(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	m.Put(1, "a")
	assert.Equal(t, "a", m.GetOrDefault(1, "default"), "GetOrDefault should return stored value")
	assert.Equal(t, "default", m.GetOrDefault(999, "default"), "GetOrDefault should return fallback for missing key")

	v, ok := m.Remove(1)
	require.True(t, ok, "Remove should succeed for existing key")
	assert.Equal(t, "a", v, "Remove should return previous value")
	_, ok = m.Get(1)
	require.False(t, ok, "Get should be false for removed key")
}

func TestConcurrentHashMap_ContainsAndRemoveAll(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")

	assert.True(t, m.ContainsKey(1), "ContainsKey should be true for present key")
	assert.False(t, m.ContainsKey(4), "Should not contain element")

	assert.True(t, m.ContainsValue("a", eqV[string]), "ContainsValue should find existing value")
	assert.False(t, m.ContainsValue("z", eqV[string]), "ContainsValue should be false for missing value")

	// RemoveIf
	removed := m.RemoveIf(1, "a", eqV[string])
	assert.True(t, removed, "RemoveIf should remove when key/value match")
	assert.False(t, m.ContainsKey(1), "Should not contain element")

	removed = m.RemoveIf(2, "xxx", eqV[string])
	assert.False(t, removed, "RemoveIf should be false when value mismatches")
	assert.True(t, m.ContainsKey(2), "ContainsKey should be true for present key")

	// RemoveAll
	count := m.RemoveAll(2, 3, 99)
	assert.Equal(t, 2, count, "Count should match expected")
	assert.True(t, m.IsEmpty(), "IsEmpty should be true")

	// RemoveSeq
	m.Put(4, "d")
	m.Put(5, "e")
	count = m.RemoveSeq(func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		if !yield(5) {
			return
		}
		yield(100)
	})
	assert.Equal(t, 2, count, "Count should match expected")
	assert.True(t, m.IsEmpty(), "IsEmpty should be true")

	// RemoveFunc
	m.Put(1, "a")
	m.Put(2, "b")
	m.Put(3, "c")
	count = m.RemoveFunc(func(k int, _ string) bool {
		return k%2 != 0 // remove odd keys: 1, 3
	})
	assert.Equal(t, 2, count, "Count should match expected")
	assert.Equal(t, 1, m.Size(), "Size should be 1 after removing odd keys")
	assert.True(t, m.ContainsKey(2), "ContainsKey should be true for present key")
}

func TestConcurrentHashMap_ComputeAndMerge(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()

	// Compute
	v, ok := m.Compute(1, func(_ int, _ int, exists bool) (int, bool) {
		assert.False(t, exists, "Exists should be false for first Compute on key 1")
		return 10, true
	})
	require.True(t, ok, "Compute should insert when key is absent and keep=true")
	assert.Equal(t, 10, v, "Sequence should match expected")

	v, ok = m.Compute(1, func(_ int, old int, exists bool) (int, bool) {
		assert.True(t, exists, "Exists should be true for subsequent Compute on key 1")
		return old + 5, true
	})
	require.True(t, ok, "Compute should update when key exists and keep=true")
	assert.Equal(t, 15, v, "Sequence should match expected")

	// Compute: delete
	_, ok = m.Compute(1, func(_ int, _ int, _ bool) (int, bool) {
		return 0, false
	})
	require.False(t, ok, "Compute should report removed when keep=false")
	assert.False(t, m.ContainsKey(1), "Should not contain element")

	// ComputeIfAbsent
	v = m.ComputeIfAbsent(2, func(_ int) int { return 20 })
	assert.Equal(t, 20, v, "Sequence should match expected")
	v = m.ComputeIfAbsent(2, func(_ int) int { return 999 }) // should not update
	assert.Equal(t, 20, v, "Sequence should match expected")

	// ComputeIfPresent
	v, ok = m.ComputeIfPresent(2, func(_ int, old int) (int, bool) {
		return old * 2, true
	})
	require.True(t, ok, "ComputeIfPresent should update and return true for existing key")
	assert.Equal(t, 40, v, "Sequence should match expected")

	_, ok = m.ComputeIfPresent(3, func(_ int, _ int) (int, bool) { return 0, true })
	require.False(t, ok, "ComputeIfPresent should be false for missing key")

	// ComputeIfPresent: delete
	_, ok = m.ComputeIfPresent(2, func(_ int, _ int) (int, bool) {
		return 0, false
	})
	require.False(t, ok, "ComputeIfPresent should be false when function indicates removal")
	assert.False(t, m.ContainsKey(2), "Should not contain element")

	// Merge
	m.Put(1, 10)
	v, ok = m.Merge(1, 20, func(old, newV int) (int, bool) {
		return old + newV, true
	})
	require.True(t, ok, "Merge should update and return true for existing key")
	assert.Equal(t, 30, v, "Sequence should match expected")

	v, ok = m.Merge(5, 50, func(_, _ int) (int, bool) { return 0, true }) // key absent
	require.True(t, ok, "Merge should insert and return true for missing key")
	assert.Equal(t, 50, v, "Sequence should match expected")

	// Merge: delete
	_, ok = m.Merge(1, 999, func(_, _ int) (int, bool) {
		return 0, false
	})
	require.False(t, ok, "Merge should return false when function indicates removal")
	assert.False(t, m.ContainsKey(1), "Should not contain element")
}

func TestConcurrentHashMap_ReplaceOperations(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	m.Put(1, "a")

	// Replace
	old, ok := m.Replace(1, "b")
	require.True(t, ok, "Replace should succeed for existing key")
	assert.Equal(t, "a", old, "Replace should return previous value")
	assert.Equal(t, "b", m.GetOrDefault(1, ""), "Replace should update stored value")

	_, ok = m.Replace(99, "z")
	require.False(t, ok, "Replace should be false for missing key")

	// ReplaceIf
	ok = m.ReplaceIf(1, "b", "c", eqV[string])
	require.True(t, ok, "ReplaceIf should succeed when current value matches")
	assert.Equal(t, "c", m.GetOrDefault(1, ""), "ReplaceIf should update stored value")

	ok = m.ReplaceIf(1, "x", "y", eqV[string])
	require.False(t, ok, "ReplaceIf should be false when current value mismatches")

	// ReplaceAll
	m.Put(2, "hello")
	m.ReplaceAll(func(_ int, v string) string {
		return strings.ToUpper(v)
	})
	assert.Equal(t, "C", m.GetOrDefault(1, ""), "ReplaceAll should update value for key 1")
	assert.Equal(t, "HELLO", m.GetOrDefault(2, ""), "ReplaceAll should update value for key 2")
}

func TestConcurrentHashMap_ViewsAndIterations(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	m.Put(1, 10)
	m.Put(2, 20)
	m.Put(3, 30)

	// Keys
	keys := m.Keys()
	assert.Len(t, keys, 3, "Keys should contain 3 entries")
	assert.Contains(t, keys, 1, "Contains should be true for expected element")

	// Values
	values := m.Values()
	assert.Len(t, values, 3, "Values should contain 3 entries")
	assert.Contains(t, values, 10, "Contains should be true for expected element")

	// Entries
	entries := m.Entries()
	assert.Len(t, entries, 3, "Entries should contain 3 entries")

	// ForEach
	count := 0
	m.ForEach(func(_, _ int) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count, "Count should match expected")

	// Seq
	count = 0
	for k, v := range m.Seq() {
		assert.Equal(t, k*10, v, "Seq should yield k,v with v=10*k")
		count++
	}
	assert.Equal(t, 3, count, "Count should match expected")

	// SeqKeys
	count = 0
	for range m.SeqKeys() {
		count++
	}
	assert.Equal(t, 3, count, "Count should match expected")

	// SeqValues
	count = 0
	for range m.SeqValues() {
		count++
	}
	assert.Equal(t, 3, count, "Count should match expected")
}

func TestConcurrentHashMap_CloneFilterEquals(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	m.Put(1, 1)
	m.Put(2, 2)
	m.Put(3, 3)

	// Clone
	c := m.Clone()
	assert.Equal(t, m.Size(), c.Size(), "Clone size should match source size")
	assert.True(t, m.Equals(c, eqV[int]), "Equals should be true")

	// Filter
	even := m.Filter(func(k, _ int) bool {
		return k%2 == 0
	})
	assert.Equal(t, 1, even.Size(), "Filter should keep one even key")
	assert.True(t, even.ContainsKey(2), "ContainsKey should be true for present key")

	// Equals
	m2 := NewHashMap[int, int]()
	m2.Put(1, 1)
	m2.Put(2, 2)
	m2.Put(3, 3)
	assert.True(t, m.Equals(m2, eqV[int]), "Equals should be true")

	m2.Put(4, 4)
	assert.False(t, m.Equals(m2, eqV[int]), "Equals should be false")
}

func TestConcurrentHashMap_CoverageSupplement(t *testing.T) {
	t.Parallel()
	// Cover cases where ForEach/Range stops early
	m := NewConcurrentHashMap[int, int]()
	for i := range 10 {
		m.Put(i, i)
	}

	count := 0
	m.ForEach(func(_, _ int) bool {
		count++
		return count < 5
	})
	assert.Equal(t, 5, count, "Count should match expected")

	// ContainsValue short-circuit
	assert.True(t, m.ContainsValue(0, eqV[int]), "ContainsValue should be true for present value")
}

func TestConcurrentHashMap_PutOverwrite(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()

	// First put - no previous value
	old, replaced := m.Put(1, "first")
	require.False(t, replaced, "Put should report no replacement for new key")
	require.Empty(t, old, "Put should return zero value for new key")

	// Second put - overwrites
	old, replaced = m.Put(1, "second")
	require.True(t, replaced, "Put should report replacement for existing key")
	require.Equal(t, "first", old, "Put should return previous value")

	// Verify new value is stored
	v, ok := m.Get(1)
	require.True(t, ok, "Get should succeed")
	require.Equal(t, "second", v, "Get should return updated value")
}

func TestConcurrentHashMap_ReplaceAllEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// ReplaceAll should process all entries
	m.ReplaceAll(func(_, v int) int {
		return v + 1
	})

	// Verify all values updated
	for i := 1; i <= 10; i++ {
		v, ok := m.Get(i)
		require.True(t, ok, "Get should succeed for key %d", i)
		require.Equal(t, i*10+1, v, "Value should be incremented")
	}

	// ReplaceAll on empty map
	empty := NewConcurrentHashMap[int, int]()
	empty.ReplaceAll(func(_, v int) int {
		return v * 2
	})
	require.Equal(t, 0, empty.Size(), "Empty map should remain empty")
}

func TestConcurrentHashMap_ComputeNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()

	// Compute on non-existing key with keep=false
	v, ok := m.Compute(1, func(_ int, _ string, exists bool) (string, bool) {
		require.False(t, exists, "Key should not exist")
		return "should_not_add", false
	})
	require.False(t, ok, "Compute should return false when keep=false")
	require.Empty(t, v, "Compute should return zero value")
	require.False(t, m.ContainsKey(1), "Key should not be added when keep=false")

	// Compute on non-existing key with keep=true
	v, ok = m.Compute(2, func(_ int, _ string, exists bool) (string, bool) {
		require.False(t, exists, "Key should not exist")
		return "new_value", true
	})
	require.True(t, ok, "Compute should return true when adding")
	require.Equal(t, "new_value", v, "Compute should return new value")
	require.True(t, m.ContainsKey(2), "Key should be added")
}

func TestConcurrentHashMap_SeqEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// Seq with early exit
	count := 0
	for range m.Seq() {
		count++
		if count >= 5 {
			break
		}
	}
	require.Equal(t, 5, count, "Seq should support early exit")
}

func TestConcurrentHashMap_SeqKeysEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// SeqKeys with early exit
	count := 0
	for range m.SeqKeys() {
		count++
		if count >= 5 {
			break
		}
	}
	require.Equal(t, 5, count, "SeqKeys should support early exit")
}

func TestConcurrentHashMap_SeqValuesEarlyExit(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	for i := 1; i <= 10; i++ {
		m.Put(i, i*10)
	}

	// SeqValues with early exit
	count := 0
	for range m.SeqValues() {
		count++
		if count >= 5 {
			break
		}
	}
	require.Equal(t, 5, count, "SeqValues should support early exit")
}

func TestConcurrentHashMap_RemoveAndGetNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	m.Put(1, "a")

	// RemoveAndGet on non-existing key
	_, ok := m.RemoveAndGet(99)
	require.False(t, ok, "RemoveAndGet should fail for non-existing key")
}

func TestConcurrentHashMap_CompareAndDeleteNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()

	// CompareAndDelete on non-existing key
	require.False(t, m.CompareAndDelete(99, "value", eqV[string]), "CompareAndDelete should fail for non-existing key")
}

func TestConcurrentHashMap_CompareAndSwapNotFound(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()

	// CompareAndSwap on non-existing key
	require.False(t, m.CompareAndSwap(99, "old", "new", eqV[string]), "CompareAndSwap should fail for non-existing key")
}

func TestConcurrentHashMap_GetOrComputeExisting(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, string]()
	m.Put(1, "existing")

	// GetOrCompute when key already exists
	v, computed := m.GetOrCompute(1, func() string { return "new" })
	require.False(t, computed, "GetOrCompute should not compute for existing key")
	require.Equal(t, "existing", v, "GetOrCompute should return existing value")
}

func TestConcurrentHashMap_IsNotEmpty(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[string, int]()
	assert.False(t, m.IsNotEmpty(), "new map should not be non-empty")
	m.Put("a", 1)
	assert.True(t, m.IsNotEmpty(), "map with entry should be non-empty")
}

// Regression: the Compute-based helpers answered del=false for an absent
// key, which the backing map treats as an insert — a zero value appeared
// for keys that ComputeIfPresent, RemoveIf, Replace, ReplaceIf,
// CompareAndSwap and CompareAndDelete never stored.
func TestConcurrentHashMap_AbsentKeyHelpersInsertNothing(t *testing.T) {
	t.Parallel()
	eq := func(a, b int) bool { return a == b }

	m := NewConcurrentHashMap[string, int]()
	_, ok := m.ComputeIfPresent("a", func(_ string, old int) (int, bool) { return old + 1, true })
	require.False(t, ok, "ComputeIfPresent on an absent key must report false")
	require.False(t, m.RemoveIf("b", 1, eq), "RemoveIf on an absent key must report false")
	_, ok = m.Replace("c", 5)
	require.False(t, ok, "Replace on an absent key must report false")
	require.False(t, m.ReplaceIf("d", 1, 2, eq), "ReplaceIf on an absent key must report false")
	require.False(t, m.CompareAndSwap("e", 1, 2, eq), "CompareAndSwap on an absent key must report false")
	require.False(t, m.CompareAndDelete("f", 1, eq), "CompareAndDelete on an absent key must report false")
	require.Equal(t, 0, m.Size(), "no helper may create the key it failed to find")
}

// Regression: a panic in a compute-family callback used to escape while the
// backing map's internal bucket lock was still held, permanently blocking
// every later operation on that bucket.
func TestConcurrentHashMap_CallbackPanicLeavesMapUsable(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[int, int]()
	m.Put(1, 10)

	require.PanicsWithValue(t, "boom", func() {
		m.GetOrCompute(2, func() int { panic("boom") })
	}, "GetOrCompute must propagate the callback panic")
	require.False(t, m.ContainsKey(2), "a panicking compute must store nothing")

	require.PanicsWithValue(t, "boom", func() {
		m.Compute(1, func(int, int, bool) (int, bool) { panic("boom") })
	}, "Compute must propagate the callback panic")
	require.PanicsWithValue(t, "boom", func() {
		m.ComputeIfAbsent(3, func(int) int { panic("boom") })
	}, "ComputeIfAbsent must propagate the callback panic")
	require.PanicsWithValue(t, "boom", func() {
		m.ComputeIfPresent(1, func(int, int) (int, bool) { panic("boom") })
	}, "ComputeIfPresent must propagate the callback panic")
	require.PanicsWithValue(t, "boom", func() {
		m.Merge(1, 5, func(int, int) (int, bool) { panic("boom") })
	}, "Merge must propagate the callback panic")
	require.PanicsWithValue(t, "boom", func() {
		m.RemoveIf(1, 10, func(int, int) bool { panic("boom") })
	}, "RemoveIf must propagate the eq panic")
	require.PanicsWithValue(t, "boom", func() {
		m.ReplaceIf(1, 10, 11, func(int, int) bool { panic("boom") })
	}, "ReplaceIf must propagate the eq panic")
	require.PanicsWithValue(t, "boom", func() {
		m.ReplaceAll(func(int, int) int { panic("boom") })
	}, "ReplaceAll must propagate the function panic")
	require.PanicsWithValue(t, "boom", func() {
		m.CompareAndSwap(1, 10, 11, func(int, int) bool { panic("boom") })
	}, "CompareAndSwap must propagate the eq panic")
	require.PanicsWithValue(t, "boom", func() {
		m.CompareAndDelete(1, 10, func(int, int) bool { panic("boom") })
	}, "CompareAndDelete must propagate the eq panic")

	v, ok := m.Get(1)
	require.True(t, ok, "existing entries must survive callback panics")
	require.Equal(t, 10, v, "a panicking callback must leave values unchanged")
	m.Put(4, 40)
	v, ok = m.Get(4)
	require.True(t, ok, "the map must stay usable after callback panics")
	require.Equal(t, 40, v, "writes after a callback panic must work")
}

// Regression: UnmarshalJSON and GobDecode replaced the backing xsync map
// pointer without synchronization, racing with every concurrent reader.
// Run with -race to exercise the guarantee.
func TestConcurrentHashMap_DecodeConcurrentWithReaders(t *testing.T) {
	t.Parallel()
	m := NewConcurrentHashMap[string, int]()
	m.Put("a", 1)
	jsonData := []byte(`{"b":2}`)
	gobData, err := m.(gob.GobEncoder).GobEncode()
	require.NoError(t, err, "encoding the map should succeed")

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 2000 {
			m.Get("a")
			m.ContainsKey("b")
		}
	})
	wg.Go(func() {
		for range 1000 {
			require.NoError(t, m.(json.Unmarshaler).UnmarshalJSON(jsonData))
			require.NoError(t, m.(gob.GobDecoder).GobDecode(gobData))
		}
	})
	wg.Wait()
}

// Regression: the callback panic guard used to test recover()'s value, which
// is nil for panic(nil) under GODEBUG=panicnil=1 — the panic was swallowed
// and a zero value written to the map. Detection now uses a completion flag.
func TestConcurrentHashMap_CallbackPanicNilCompat(t *testing.T) {
	t.Setenv("GODEBUG", "panicnil=1")
	// Verify the compat mode applies dynamically before relying on it.
	probe := any("sentinel")
	func() {
		defer func() { probe = recover() }()
		panic(nil)
	}()
	if probe != nil {
		t.Skipf("GODEBUG=panicnil=1 did not apply (recover returned %v)", probe)
	}

	mustPanic := func(name string, fn func()) {
		t.Helper()
		panicked := true
		func() {
			defer func() { _ = recover() }()
			fn()
			panicked = false
		}()
		require.True(t, panicked, "%s must propagate panic(nil), not swallow it", name)
	}

	m := NewConcurrentHashMap[string, int]()
	m.Put("k", 7)

	mustPanic("Compute", func() {
		m.Compute("k", func(string, int, bool) (int, bool) { panic(nil) })
	})
	v, ok := m.Get("k")
	require.True(t, ok, "the key should survive the panic")
	require.Equal(t, 7, v, "the existing value must not be overwritten with a zero value")

	mustPanic("Compute(absent)", func() {
		m.Compute("absent", func(string, int, bool) (int, bool) { panic(nil) })
	})
	require.False(t, m.ContainsKey("absent"), "no zero value must be inserted for the absent key")

	mustPanic("GetOrCompute", func() {
		m.GetOrCompute("absent", func() int { panic(nil) })
	})
	require.False(t, m.ContainsKey("absent"), "a canceled compute must store nothing")

	mustPanic("RemoveIf", func() {
		m.RemoveIf("k", 7, func(int, int) bool { panic(nil) })
	})
	require.True(t, m.ContainsKey("k"), "the entry must survive a panicking eq")

	old, existed := m.Put("k", 8)
	require.True(t, existed, "the map must stay usable after the panics")
	require.Equal(t, 7, old, "the stored value must be intact")
}

// Regression: gob allocates a zero-value concrete receiver when decoding an
// interface field; GobDecode then nil-panicked on the uninitialized backing map.
func TestConcurrentHashMap_GobInterfaceRoundTrip(t *testing.T) {
	t.Parallel()
	gob.Register(NewConcurrentHashMap[string, int]())
	type payload struct{ M Map[string, int] }
	src := payload{M: NewConcurrentHashMap[string, int]()}
	src.M.Put("a", 1)
	src.M.Put("b", 2)

	var buf bytes.Buffer
	require.NoError(t, gob.NewEncoder(&buf).Encode(src), "encoding the wrapper should succeed")
	var dst payload
	require.NoError(t, gob.NewDecoder(&buf).Decode(&dst), "decoding into a zero-value receiver should succeed")

	v, ok := dst.M.Get("a")
	require.True(t, ok, "the decoded map should contain key a")
	require.Equal(t, 1, v, "the decoded value should round-trip")
	require.Equal(t, 2, dst.M.Size(), "the decoded map should hold both entries")
	dst.M.Put("c", 3)
	require.True(t, dst.M.ContainsKey("c"), "the decoded map should accept writes")
}
