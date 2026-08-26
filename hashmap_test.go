package collections

import (
	"cmp"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// helper to make a Seq2 from slices of keys/values (same length).
func seq2Of[K any, V any](keys []K, values []V) func(func(K, V) bool) {
	return func(yield func(K, V) bool) {
		for i := range keys {
			if !yield(keys[i], values[i]) {
				return
			}
		}
	}
}

func eqV[T comparable](a, b T) bool { return a == b }

func TestHashMap_BasicCRUD(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	require.True(t, m.IsEmpty(), "New map should be empty")
	require.Equal(t, 0, m.Size(), "Empty map size should be 0")
	_, ok := m.Get(1)
	require.False(t, ok, "Unexpected key should not be present")
	_, ok = m.Remove(1)
	require.False(t, ok, "Remove on absent key should be false")
	require.Equal(t, "x", m.GetOrDefault(1, "x"), "GetOrDefault should return fallback")
	old, replaced := m.Put(1, "a")
	require.False(t, replaced, "First Put should not report replaced")
	require.Empty(t, old, "Old value should be zero on first Put")
	old, replaced = m.Put(1, "b")
	require.True(t, replaced, "Second Put should report replaced")
	require.Equal(t, "a", old, "Old value should be returned")
	val, ok := m.Get(1)
	require.True(t, ok, "Get should succeed for existing key")
	require.Equal(t, "b", val, "Get should return latest value")
	// PutIfAbsent
	v, inserted := m.PutIfAbsent(1, "c")
	require.False(t, inserted, "PutIfAbsent should not insert when present")
	require.Equal(t, "b", v, "PutIfAbsent should return existing value")
	v, inserted = m.PutIfAbsent(2, "x")
	require.True(t, inserted, "PutIfAbsent should insert when absent")
	require.Equal(t, "x", v, "PutIfAbsent should return inserted value")
	// RemoveIf
	require.True(t, m.RemoveIf(2, "x", eqV[string]), "RemoveIf should remove matching entry")
	require.False(t, m.RemoveIf(2, "x", eqV[string]), "RemoveIf should fail when already removed")
}

func TestHashMap_PutSeq_RemoveSeq(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, int]()
	changed := m.PutSeq(seq2Of([]int{1, 2, 3, 2}, []int{10, 20, 30, 200}))
	// Keys touched: 1,2,3 -> 3 unique
	require.Equal(t, 3, changed, "PutSeq should report unique keys touched")
	require.Equal(t, 3, m.Size(), "Size should reflect unique keys")
	removed := m.RemoveSeq(seqOf([]int{2, 4}))
	require.Equal(t, 1, removed, "RemoveSeq should remove only present keys")
	require.Equal(t, 2, m.Size(), "Size should decrease accordingly")
}

func TestHashMap_Contains_Views_Iter(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	require.True(t, m.ContainsKey(1), "ContainsKey should be true for present key")
	require.False(t, m.ContainsKey(3), "ContainsKey should be false for absent key")
	require.True(t, m.ContainsValue("b", eqV[string]), "ContainsValue should find existing value")
	require.False(t, m.ContainsValue("x", eqV[string]), "ContainsValue should not find absent value")
	ks := m.Keys()
	slices.Sort(ks)
	require.True(t, slices.Equal(ks, []int{1, 2}), "Keys=%v", ks)
	vs := m.Values()
	slices.Sort(vs)
	require.True(t, slices.Equal(vs, []string{"a", "b"}), "Values=%v", vs)
	es := m.Entries()
	slices.SortFunc(es, func(a, b Entry[int, string]) int { return cmp.Compare(a.Key, b.Key) })
	require.Len(t, es, 2, "Entries length should be 2")
	require.Equal(t, 1, es[0].Key, "First entry should have smallest key")
	require.Equal(t, 2, es[1].Key, "Second entry should have largest key")
	// ForEach early stop
	cnt := 0
	m.ForEach(func(_ int, _ string) bool {
		cnt++
		return false // stop immediately
	})
	require.Equal(t, 1, cnt, "ForEach should stop after first iteration")
	// Seq/SeqKeys/SeqValues
	gotK := make([]int, 0, 2)
	for k := range m.SeqKeys() {
		gotK = append(gotK, k)
	}
	slices.Sort(gotK)
	require.True(t, slices.Equal(gotK, []int{1, 2}), "SeqKeys=%v", gotK)
	gotV := make([]string, 0, 2)
	for _, v := range m.Seq() {
		gotV = append(gotV, v)
	}
	slices.Sort(gotV)
	require.True(t, slices.Equal(gotV, []string{"a", "b"}), "Seq Values should match, got=%v", gotV)
	gotValues := make([]string, 0, 2)
	for v := range m.SeqValues() {
		gotValues = append(gotValues, v)
	}
	slices.Sort(gotValues)
	require.True(t, slices.Equal(gotValues, []string{"a", "b"}), "SeqValues=%v", gotValues)
}

func TestHashMap_ComputeVariants(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, int]()
	// ComputeIfAbsent
	v := m.ComputeIfAbsent(1, func(k int) int { return k * 10 })
	require.Equal(t, 10, v, "ComputeIfAbsent should compute value from key")
	require.Equal(t, 1, m.Size(), "Size should be 1 after ComputeIfAbsent")
	// Compute existing -> update
	nv, ok := m.Compute(1, func(_, old int, exists bool) (int, bool) {
		require.True(t, exists, "Exists expected true")
		return old + 1, true
	})
	require.True(t, ok, "Compute should keep key when keep=true")
	require.Equal(t, 11, nv, "Compute should update value")
	// Compute remove
	_, ok = m.Compute(1, func(_, _ int, _ bool) (int, bool) {
		return 0, false
	})
	require.False(t, ok, "Compute should report removed when keep=false")
	require.False(t, m.ContainsKey(1), "Key should be removed after keep=false")
	// ComputeIfPresent on absent
	_, ok = m.ComputeIfPresent(1, func(_, _ int) (int, bool) { return 0, true })
	require.False(t, ok, "ComputeIfPresent should be false on absent")
	// Merge
	m.Put(2, 5)
	nv, ok = m.Merge(2, 7, func(old, newValue int) (int, bool) { return old + newValue, true })
	require.True(t, ok, "Merge should keep key when keep=true")
	require.Equal(t, 12, nv, "Merge should combine values")
	_, ok = m.Merge(2, 0, func(_, _ int) (int, bool) { return 0, false })
	require.False(t, ok, "Merge should remove when keep=false")
	require.False(t, m.ContainsKey(2), "Key should be removed by Merge keep=false")
	// Merge on absent
	v, ok = m.Merge(3, 30, func(old, newValue int) (int, bool) { return old + newValue, true })
	require.True(t, ok, "Merge on absent should insert")
	require.Equal(t, 30, v, "Merge on absent should return inserted value")
}

func TestHashMap_ReplaceAndFilterAndEquals(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")
	old, replaced := m.Replace(1, "aa")
	require.True(t, replaced, "Replace should succeed for existing key")
	require.Equal(t, "a", old, "Replace should return previous value")
	require.False(t, m.ReplaceIf(2, "x", "bb", eqV[string]), "ReplaceIf should fail on mismatched oldValue")
	require.True(t, m.ReplaceIf(2, "b", "bb", eqV[string]), "ReplaceIf should succeed when current value matches")
	// ReplaceAll
	m.ReplaceAll(func(_ int, v string) string { return v + "!" })
	require.True(t, m.ContainsValue("aa!", eqV[string]), "ReplaceAll should update values")
	require.True(t, m.ContainsValue("bb!", eqV[string]), "ReplaceAll should update values")
	// Clone
	cp := m.Clone()
	require.True(t, m.Equals(cp, eqV[string]), "Clone should be equal to original")
	require.True(t, cp.Equals(m, eqV[string]), "Equality should be symmetric")
	// Filter
	f := m.Filter(func(k int, _ string) bool { return k == 1 })
	require.Equal(t, 1, f.Size(), "Filter should keep one key")
	require.True(t, f.ContainsKey(1), "Filter result should contain kept key")
	// ToGoMap is a snapshot (available via GoMapView)
	gm := m.(GoMapView[int, string]).ToGoMap()
	gm[3] = "zzz"
	require.False(t, m.ContainsKey(3), "ToGoMap must return snapshot, not live map")
}

func TestHashMap_EmptySingle(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	require.True(t, m.IsEmpty(), "New map should be empty")
	require.Equal(t, 0, m.Size(), "Empty map size should be 0")
	require.Equal(t, "x", m.GetOrDefault(1, "x"), "GetOrDefault should return fallback")
	_, ok := m.Remove(1)
	require.False(t, ok, "Remove on absent key should be false")
	old, replaced := m.Put(1, "a")
	require.False(t, replaced, "First Put should not replace")
	require.Empty(t, old, "Old value should be zero on first Put")
	v, ok := m.Get(1)
	require.True(t, ok, "Get should succeed for existing key")
	require.Equal(t, "a", v, "Get should return stored value")
}

func TestHashMap_Clear(t *testing.T) {
	t.Parallel()
	m := NewHashMapFrom(map[int]string{1: "a", 2: "b", 3: "c"})
	require.False(t, m.IsEmpty(), "Map should not be empty before Clear")
	m.Clear()
	require.True(t, m.IsEmpty(), "Map should be empty after Clear")
	require.Equal(t, 0, m.Size(), "Size should be 0 after Clear")
}

func TestHashMap_RemoveAll(t *testing.T) {
	t.Parallel()
	m := NewHashMapFrom(map[int]string{1: "a", 2: "b", 3: "c"})
	removed := m.RemoveAll(1, 2)
	require.Equal(t, 2, removed, "RemoveAll should remove two keys")
	require.Equal(t, 1, m.Size(), "Size should decrease accordingly")
	require.False(t, m.ContainsKey(1), "Key 1 should be removed")
	require.False(t, m.ContainsKey(2), "Key 2 should be removed")
	require.True(t, m.ContainsKey(3), "Key 3 should remain")
}

func TestHashMap_PutAll(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	m.Put(1, "a")
	m2 := NewHashMapFrom(map[int]string{1: "x", 2: "y"})
	m.PutAll(m2)
	require.Equal(t, 2, m.Size(), "PutAll should copy entries from other map")
	v, _ := m.Get(1)
	require.Equal(t, "x", v, "Value for key 1 should be overwritten")
	v, _ = m.Get(2)
	require.Equal(t, "y", v, "Key 2 should be copied")
}

func TestHashMap_RemoveFunc(t *testing.T) {
	t.Parallel()
	m := NewHashMapFrom(map[int]string{1: "a", 2: "b", 3: "c"})
	count := m.RemoveFunc(func(k int, _ string) bool { return k > 1 })
	require.Equal(t, 2, count, "RemoveFunc should remove keys > 1")
	require.Equal(t, 1, m.Size(), "Size should reflect removals")
	require.True(t, m.ContainsKey(1), "Key 1 should remain")
	require.False(t, m.ContainsKey(2), "Key 2 should be removed")
	require.False(t, m.ContainsKey(3), "Key 3 should be removed")
}

func TestHashMap_From(t *testing.T) {
	t.Parallel()
	m := NewHashMapFrom(map[int]string{1: "a", 2: "b"})
	require.Equal(t, 2, m.Size(), "Size should equal source map size")
	v, ok := m.Get(1)
	require.True(t, ok, "Get should succeed for existing key")
	require.Equal(t, "a", v, "Get should return copied value")
}

func TestHashMap_NewWithCapacity(t *testing.T) {
	t.Parallel()
	m := NewHashMapWithCapacity[int, string](10)
	require.True(t, m.IsEmpty(), "New map with capacity should be empty")
	m.Put(1, "a")
	require.Equal(t, 1, m.Size(), "Size should be 1 after Put")

	// Test with zero capacity
	m2 := NewHashMapWithCapacity[int, string](0)
	require.True(t, m2.IsEmpty(), "New map with 0 capacity should be empty")

	// Test with negative capacity (should be treated as 0)
	m3 := NewHashMapWithCapacity[int, string](-5)
	require.True(t, m3.IsEmpty(), "New map with negative capacity should be empty")
}

func TestHashMap_GetOrDefault(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()

	// Key not present - should return default
	defaultVal := "default"
	result := m.GetOrDefault(1, defaultVal)
	require.Equal(t, "default", result, "GetOrDefault should return default value for absent key")

	// Key present - should return actual value
	m.Put(1, "value1")
	result = m.GetOrDefault(1, defaultVal)
	require.Equal(t, "value1", result, "GetOrDefault should return actual value for present key")

	// Multiple keys
	m.Put(2, "value2")
	result = m.GetOrDefault(2, "default")
	require.Equal(t, "value2", result, "GetOrDefault should work with multiple keys")
	result = m.GetOrDefault(3, "default")
	require.Equal(t, "default", result, "GetOrDefault should return default for non-existent key")
}

func TestHashMap_Replace(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()

	// Replace on non-existent key should return false
	_, replaced := m.Replace(1, "value")
	require.False(t, replaced, "Replace should return false for absent key")
	require.False(t, m.ContainsKey(1), "Replace should not add key")

	// Replace on existing key should succeed
	m.Put(1, "oldValue")
	old, replaced := m.Replace(1, "newValue")
	require.True(t, replaced, "Replace should return true for present key")
	require.Equal(t, "oldValue", old, "Replace should return old value")

	v, _ := m.Get(1)
	require.Equal(t, "newValue", v, "Replace should update the value")

	// Replace multiple keys
	m.Put(2, "a")
	m.Put(3, "b")
	_, replaced = m.Replace(2, "aa")
	require.True(t, replaced, "Replace should succeed for key 2")
	_, replaced = m.Replace(3, "bb")
	require.True(t, replaced, "Replace should succeed for key 3")
	_, replaced = m.Replace(4, "cc")
	require.False(t, replaced, "Replace should fail for non-existent key 4")
}

func TestHashMap_ComputeIfPresent(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()

	// Test on absent key - should return (zero, false)
	_, ok := m.ComputeIfPresent(1, func(_ int, _ string) (string, bool) {
		return "new", true
	})
	require.False(t, ok, "ComputeIfPresent should return false for absent key")
	require.Equal(t, 0, m.Size(), "Size should remain 0 for absent key")

	// Test on existing key - keep=true (update)
	m.Put(1, "old")
	newVal, ok := m.ComputeIfPresent(1, func(_ int, _ string) (string, bool) {
		return "updated", true
	})
	require.True(t, ok, "ComputeIfPresent should return true when key exists")
	require.Equal(t, "updated", newVal, "ComputeIfPresent should return new value")
	v, _ := m.Get(1)
	require.Equal(t, "updated", v, "Value should be updated")

	// Test on existing key - keep=false (remove)
	_, ok = m.ComputeIfPresent(1, func(_ int, _ string) (string, bool) {
		return "", false
	})
	require.False(t, ok, "ComputeIfPresent should return false when key is removed")
	require.False(t, m.ContainsKey(1), "Key should be removed when keep=false")

	// Test multiple updates
	m.Put(2, "a")
	m.Put(3, "b")
	_, _ = m.ComputeIfPresent(2, func(_ int, _ string) (string, bool) {
		return "aa", true
	})
	_, _ = m.ComputeIfPresent(3, func(_ int, _ string) (string, bool) {
		return "bb", true
	})
	v2, _ := m.Get(2)
	v3, _ := m.Get(3)
	require.Equal(t, "aa", v2, "Key 2 should be updated")
	require.Equal(t, "bb", v3, "Key 3 should be updated")
}

func TestHashMap_String(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	m.Put(1, "a")
	s := m.String()
	require.Contains(t, s, "hashMap", "String should include type name")
	require.Contains(t, s, "1:a", "String should include entry representation")
}

func TestHashMap_RemoveNotFound(t *testing.T) {
	t.Parallel()
	m := NewHashMap[int, string]()
	m.Put(1, "a")
	m.Put(2, "b")

	// Remove existing key
	old, ok := m.Remove(1)
	require.True(t, ok, "Remove should succeed for existing key")
	require.Equal(t, "a", old, "Remove should return old value")

	// Remove non-existent key
	_, ok = m.Remove(99)
	require.False(t, ok, "Remove should fail for non-existent key")

	require.Equal(t, 1, m.Size(), "Size should be 1 after one removal")
}

func TestHashMap_IsNotEmpty(t *testing.T) {
	t.Parallel()
	m := NewHashMap[string, int]()
	require.False(t, m.IsNotEmpty(), "new map should not be non-empty")
	m.Put("a", 1)
	require.True(t, m.IsNotEmpty(), "map with entry should be non-empty")
}
