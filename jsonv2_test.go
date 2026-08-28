package collections

import (
	"cmp"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time conformance: every serializable container speaks json/v2's
// streaming interfaces alongside the v1 ones.
var (
	_ jsonv2.MarshalerTo     = (*arrayList[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*arrayList[int])(nil)
	_ jsonv2.MarshalerTo     = (*cowList[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*cowList[int])(nil)
	_ jsonv2.MarshalerTo     = (*syncList[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*syncList[int])(nil)
	_ jsonv2.MarshalerTo     = (*lockFreeList[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*lockFreeList[int])(nil)
	_ jsonv2.MarshalerTo     = (*linkedList[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*linkedList[int])(nil)
	_ jsonv2.MarshalerTo     = (*arrayDeque[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*arrayDeque[int])(nil)
	_ jsonv2.MarshalerTo     = (*arrayStack[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*arrayStack[int])(nil)
	_ jsonv2.MarshalerTo     = (*arrayQueue[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*arrayQueue[int])(nil)
	_ jsonv2.MarshalerTo     = (*priorityQueue[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*priorityQueue[int])(nil)
	_ jsonv2.MarshalerTo     = (*hashSet[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*hashSet[int])(nil)
	_ jsonv2.MarshalerTo     = (*hashMap[string, int])(nil)
	_ jsonv2.UnmarshalerFrom = (*hashMap[string, int])(nil)
	_ jsonv2.MarshalerTo     = (*treeSet[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*treeSet[int])(nil)
	_ jsonv2.MarshalerTo     = (*treeMap[string, int])(nil)
	_ jsonv2.UnmarshalerFrom = (*treeMap[string, int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentHashMap[string, int])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentHashMap[string, int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentHashSet[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentHashSet[int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentTreeSet[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentTreeSet[int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentTreeMap[string, int])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentTreeMap[string, int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentSkipSet[int])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentSkipSet[int])(nil)
	_ jsonv2.MarshalerTo     = (*concurrentSkipMap[int, string])(nil)
	_ jsonv2.UnmarshalerFrom = (*concurrentSkipMap[int, string])(nil)
)

// checkJSONv2Parity marshals src with both json versions, cross-decodes each
// output with the other version, and compares the decoded contents: the two
// codecs must speak the same wire format.
func checkJSONv2Parity[C any](t *testing.T, src C, newDst func() C, contents func(C) any) {
	t.Helper()
	v1bytes, err := json.Marshal(src)
	require.NoError(t, err, "v1 marshal should succeed")
	v2bytes, err := jsonv2.Marshal(src)
	require.NoError(t, err, "v2 marshal should succeed")

	viaV1 := newDst()
	require.NoError(t, json.Unmarshal(v2bytes, viaV1), "v1 should decode the v2 output")
	assert.Equal(t, contents(src), contents(viaV1), "v2 output decoded by v1 should match the source")

	viaV2 := newDst()
	require.NoError(t, jsonv2.Unmarshal(v1bytes, viaV2), "v2 should decode the v1 output")
	assert.Equal(t, contents(src), contents(viaV2), "v1 output decoded by v2 should match the source")
}

func sortedElements[C interface{ ToSlice() []int }](c C) any {
	sl := c.ToSlice()
	slices.Sort(sl)
	return sl
}

func TestJSONv2_RoundTripParity(t *testing.T) {
	t.Parallel()
	intEq := func(a, b int) bool { return a == b }

	t.Run("ArrayList", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewArrayListFrom(1, 2, 3),
			NewArrayList[int], func(c List[int]) any { return c.ToSlice() })
	})
	t.Run("COWList", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewCOWListFrom(1, 2, 3),
			NewCOWList[int], func(c List[int]) any { return c.ToSlice() })
	})
	t.Run("SyncList", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewSyncListFrom(1, 2, 3),
			NewSyncList[int], func(c List[int]) any { return c.ToSlice() })
	})
	t.Run("LockFreeList", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewLockFreeListFrom(intEq, 1, 2, 3),
			func() List[int] { return NewLockFreeList(intEq) },
			func(c List[int]) any { return c.ToSlice() })
	})
	t.Run("LinkedList", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewLinkedListFrom(1, 2, 3),
			NewLinkedList[int], func(c List[int]) any { return c.ToSlice() })
	})
	t.Run("ArrayDeque", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewArrayDequeFrom(1, 2, 3),
			NewArrayDeque[int], func(c Deque[int]) any { return c.ToSlice() })
	})
	t.Run("ArrayStack", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewArrayStackFrom(1, 2, 3),
			NewArrayStack[int], func(c Stack[int]) any { return c.ToSlice() })
	})
	t.Run("ArrayQueue", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewArrayQueueFrom(1, 2, 3),
			NewArrayQueue[int], func(c Queue[int]) any { return c.ToSlice() })
	})
	t.Run("PriorityQueue", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewPriorityQueueFrom(cmp.Compare[int], 3, 1, 2),
			func() PriorityQueue[int] { return NewPriorityQueue(cmp.Compare[int]) },
			sortedElements[PriorityQueue[int]])
	})
	t.Run("HashSet", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewHashSetFrom(1, 2, 3),
			NewHashSet[int], sortedElements[Set[int]])
	})
	t.Run("HashMap", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewHashMapFrom(map[string]int{"a": 1, "b": 2}),
			NewHashMap[string, int], func(c Map[string, int]) any { return maps.Collect(c.Seq()) })
	})
	t.Run("TreeSet", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewTreeSetFrom(cmp.Compare[int], 3, 1, 2),
			NewTreeSetOrdered[int], sortedElements[SortedSet[int]])
	})
	t.Run("TreeMap", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewTreeMapFrom(cmp.Compare[string], map[string]int{"a": 1, "b": 2}),
			NewTreeMapOrdered[string, int], func(c SortedMap[string, int]) any { return maps.Collect(c.Seq()) })
	})
	t.Run("ConcurrentHashMap", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewConcurrentHashMapFrom(map[string]int{"a": 1, "b": 2}),
			NewConcurrentHashMap[string, int], func(c ConcurrentMap[string, int]) any { return maps.Collect(c.Seq()) })
	})
	t.Run("ConcurrentHashSet", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewConcurrentHashSetFrom(1, 2, 3),
			NewConcurrentHashSet[int], sortedElements[ConcurrentSet[int]])
	})
	t.Run("ConcurrentTreeSet", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewConcurrentTreeSetFrom(cmp.Compare[int], 3, 1, 2),
			NewConcurrentTreeSetOrdered[int], sortedElements[ConcurrentSortedSet[int]])
	})
	t.Run("ConcurrentTreeMap", func(t *testing.T) {
		t.Parallel()
		src := NewConcurrentTreeMapOrdered[string, int]()
		src.Put("a", 1)
		src.Put("b", 2)
		checkJSONv2Parity(t, src,
			NewConcurrentTreeMapOrdered[string, int],
			func(c ConcurrentSortedMap[string, int]) any { return maps.Collect(c.Seq()) })
	})
	t.Run("ConcurrentSkipSet", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewConcurrentSkipSetFrom(1, 2, 3),
			NewConcurrentSkipSet[int], sortedElements[ConcurrentSortedSet[int]])
	})
	t.Run("ConcurrentSkipMap", func(t *testing.T) {
		t.Parallel()
		checkJSONv2Parity(t, NewConcurrentSkipMapFrom(map[int]string{1: "one", 2: "two"}),
			NewConcurrentSkipMap[int, string],
			func(c ConcurrentSortedMap[int, string]) any { return maps.Collect(c.Seq()) })
	})
}

// The v2 decode path must enforce the same contracts as the v1 path: the
// comparator-carrying containers reject a zero-value receiver, and the hash
// sets reject dynamically unhashable elements without mutating themselves.
func TestJSONv2_DecodeContracts(t *testing.T) {
	t.Parallel()

	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`[1]`), &treeSet[int]{}),
		"no comparator", "a comparator-less tree set must be rejected")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`{"entries":[]}`), &treeMap[int, int]{}),
		"no comparator", "a comparator-less tree map must be rejected")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`[1]`), &priorityQueue[int]{}),
		"no comparator", "a comparator-less priority queue must be rejected")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`[1]`), &concurrentTreeSet[int]{}),
		"no comparator", "a comparator-less concurrent tree set must be rejected")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`{"entries":[]}`), &concurrentTreeMap[int, int]{}),
		"no comparator", "a comparator-less concurrent tree map must be rejected")

	hs := NewHashSetFrom[any]("old")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`[1,[2]]`), hs),
		"unhashable", "an unhashable element must be rejected on the v2 path")
	assert.True(t, hs.Contains("old"), "a rejected payload must leave the set untouched")
	assert.Equal(t, 1, hs.Size(), "no payload element may leak into the set")

	chs := NewConcurrentHashSetFrom[any]("old")
	require.ErrorContains(t, jsonv2.Unmarshal([]byte(`[1,[2]]`), chs),
		"unhashable", "an unhashable element must be rejected on the v2 path")
	assert.True(t, chs.Contains("old"), "a rejected payload must leave the set untouched")
	assert.Equal(t, 1, chs.Size(), "no payload element may leak into the set")
}

// Containers held in struct fields stream through v2 without an intermediate
// buffer: marshal dispatches MarshalJSONTo and unmarshal dispatches
// UnmarshalJSONFrom on the pre-populated field values.
func TestJSONv2_NestedContainerFields(t *testing.T) {
	t.Parallel()
	type doc struct {
		L List[int]        `json:"l"`
		M Map[string, int] `json:"m"`
	}
	src := doc{L: NewArrayListFrom(1, 2), M: NewHashMapFrom(map[string]int{"a": 1})}
	data, err := jsonv2.Marshal(src)
	require.NoError(t, err, "marshaling nested containers should succeed")

	dst := doc{L: NewArrayList[int](), M: NewHashMap[string, int]()}
	require.NoError(t, jsonv2.Unmarshal(data, &dst), "unmarshaling into pre-populated fields should succeed")
	assert.Equal(t, []int{1, 2}, dst.L.ToSlice(), "the list field should round-trip")
	assert.Equal(t, map[string]int{"a": 1}, maps.Collect(dst.M.Seq()), "the map field should round-trip")
}
