package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ==========================
// 1. HashSet Serialization Tests
// ==========================

type HashSetSerializationTestSuite struct {
	suite.Suite
}

func (s *HashSetSerializationTestSuite) TestEmptyHashSet() {
	s.Run("JSON", func() {
		original := NewHashSet[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewHashSet[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *HashSetSerializationTestSuite) TestSingleElement() {
	s.Run("JSON", func() {
		original := NewHashSetFrom(42)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		s.True(restored.Contains(42), "Should contain element")
	})

	s.Run("Gob", func() {
		original := NewHashSetFrom(42)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		s.True(restored.Contains(42), "Should contain element")
	})
}

func (s *HashSetSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewHashSetFrom(1, 2, 3, 4, 5)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(1, 2, 3, 4, 5), "Should contain all elements")
	})

	s.Run("Gob", func() {
		original := NewHashSetFrom(1, 2, 3, 4, 5)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(1, 2, 3, 4, 5), "Should contain all elements")
	})
}

func (s *HashSetSerializationTestSuite) TestRoundTrip() {
	s.Run("JSON", func() {
		original := NewHashSetFrom(-10, 0, 42, 100, 999)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.True(original.Equals(restored), "Should be equal after round-trip")
	})

	s.Run("Gob", func() {
		original := NewHashSetFrom(-10, 0, 42, 100, 999)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.True(original.Equals(restored), "Should be equal after round-trip")
	})
}

func TestHashSetSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(HashSetSerializationTestSuite))
}

// ==========================
// 2. ArrayList Serialization Tests
// ==========================

type ArrayListSerializationTestSuite struct {
	suite.Suite
}

func (s *ArrayListSerializationTestSuite) TestEmptyList() {
	s.Run("JSON", func() {
		original := NewArrayList[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewArrayList[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ArrayListSerializationTestSuite) TestSingleElement() {
	s.Run("JSON", func() {
		original := NewArrayListFrom(42)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		val, ok := restored.Get(0)
		s.Require().True(ok, "Should retrieve element")
		s.Equal(42, val, "Value should match")
	})

	s.Run("Gob", func() {
		original := NewArrayListFrom(42)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		val, ok := restored.Get(0)
		s.Require().True(ok, "Should retrieve element")
		s.Equal(42, val, "Value should match")
	})
}

func (s *ArrayListSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewArrayListFrom(1, 2, 3, 4, 5)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for i := range original.Size() {
			origVal, _ := original.Get(i)
			restVal, _ := restored.Get(i)
			s.Equal(origVal, restVal, "Element at index %d should match", i)
		}
	})

	s.Run("Gob", func() {
		original := NewArrayListFrom(1, 2, 3, 4, 5)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for i := range original.Size() {
			origVal, _ := original.Get(i)
			restVal, _ := restored.Get(i)
			s.Equal(origVal, restVal, "Element at index %d should match", i)
		}
	})
}

func (s *ArrayListSerializationTestSuite) TestRoundTripPreservesOrder() {
	s.Run("JSON", func() {
		original := NewArrayListFrom(5, 3, 1, 4, 2)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewArrayListFrom(5, 3, 1, 4, 2)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestArrayListSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ArrayListSerializationTestSuite))
}

// ==========================
// 3. LinkedList Serialization Tests
// ==========================

type LinkedListSerializationTestSuite struct {
	suite.Suite
}

func (s *LinkedListSerializationTestSuite) TestEmptyList() {
	s.Run("JSON", func() {
		original := NewLinkedList[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewLinkedList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewLinkedList[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewLinkedList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *LinkedListSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewLinkedListFrom(10, 20, 30, 40, 50)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewLinkedList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewLinkedListFrom(10, 20, 30, 40, 50)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewLinkedList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestLinkedListSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(LinkedListSerializationTestSuite))
}

// ==========================
// 4. SyncList Serialization Tests
// ==========================

type SyncListSerializationTestSuite struct {
	suite.Suite
}

func (s *SyncListSerializationTestSuite) TestEmptyList() {
	s.Run("JSON", func() {
		original := NewSyncList[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewSyncList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewSyncList[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewSyncList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *SyncListSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewSyncListFrom(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewSyncList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewSyncListFrom(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewSyncList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestSyncListSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(SyncListSerializationTestSuite))
}

// ==========================
// 5. CowList Serialization Tests
// ==========================

type CowListSerializationTestSuite struct {
	suite.Suite
}

func (s *CowListSerializationTestSuite) TestEmptyList() {
	s.Run("JSON", func() {
		original := NewCOWList[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewCOWList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewCOWList[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewCOWList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *CowListSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewCOWListFrom(100, 200, 300)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewCOWList[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewCOWListFrom(100, 200, 300)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewCOWList[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestCowListSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(CowListSerializationTestSuite))
}

// ==========================
// 6. LockFreeList Serialization Tests
// ==========================

type LockFreeListSerializationTestSuite struct {
	suite.Suite
}

func (s *LockFreeListSerializationTestSuite) TestEmptyList() {
	s.Run("JSON", func() {
		original := NewLockFreeListOrdered[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewLockFreeListOrdered[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewLockFreeListOrdered[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewLockFreeListOrdered[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *LockFreeListSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewLockFreeListFrom(func(a, b int) bool { return a == b }, 7, 8, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewLockFreeList(func(a, b int) bool { return a == b })
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewLockFreeListFrom(func(a, b int) bool { return a == b }, 7, 8, 9)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewLockFreeList(func(a, b int) bool { return a == b })
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestLockFreeListSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(LockFreeListSerializationTestSuite))
}

// ==========================
// 7. HashMap Serialization Tests
// ==========================

type HashMapSerializationTestSuite struct {
	suite.Suite
}

func (s *HashMapSerializationTestSuite) TestEmptyMap() {
	s.Run("JSON", func() {
		original := NewHashMap[string, int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashMap[string, int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewHashMap[string, int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashMap[string, int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *HashMapSerializationTestSuite) TestSingleEntry() {
	s.Run("JSON", func() {
		original := NewHashMap[string, int]()
		original.Put("answer", 42)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashMap[string, int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		val, ok := restored.Get("answer")
		s.Require().True(ok, "Key should exist")
		s.Equal(42, val, "Value should match")
	})

	s.Run("Gob", func() {
		original := NewHashMap[string, int]()
		original.Put("answer", 42)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashMap[string, int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(1, restored.Size(), "Size should be one")
		val, ok := restored.Get("answer")
		s.Require().True(ok, "Key should exist")
		s.Equal(42, val, "Value should match")
	})
}

func (s *HashMapSerializationTestSuite) TestMultipleEntries() {
	s.Run("JSON", func() {
		original := NewHashMap[string, int]()
		original.Put("one", 1)
		original.Put("two", 2)
		original.Put("three", 3)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashMap[string, int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []string{"one", "two", "three"} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %s should exist", key)
			s.Equal(origVal, restVal, "Value for key %s should match", key)
		}
	})

	s.Run("Gob", func() {
		original := NewHashMap[string, int]()
		original.Put("one", 1)
		original.Put("two", 2)
		original.Put("three", 3)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewHashMap[string, int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []string{"one", "two", "three"} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %s should exist", key)
			s.Equal(origVal, restVal, "Value for key %s should match", key)
		}
	})
}

// Regression: stdlib decoders merge into an existing map, which used to
// leave stale entries behind; every codec path must replace wholesale.
func (s *HashMapSerializationTestSuite) TestDecodeReplacesExistingEntries() {
	s.Run("JSON", func() {
		original := NewHashMap[string, int]()
		original.Put("fresh", 1)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewHashMap[string, int]()
		restored.Put("stale", 99)
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal should succeed")
		s.Equal(1, restored.Size(), "decode should replace, not merge")
		s.False(restored.ContainsKey("stale"), "pre-existing entries should be gone")
	})

	s.Run("Gob", func() {
		original := NewHashMap[string, int]()
		original.Put("fresh", 1)
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewHashMap[string, int]()
		restored.Put("stale", 99)
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode should succeed")
		s.Equal(1, restored.Size(), "decode should replace, not merge")
		s.False(restored.ContainsKey("stale"), "pre-existing entries should be gone")
	})
}

func TestHashMapSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(HashMapSerializationTestSuite))
}

// ==========================
// 8. ArrayStack Serialization Tests
// ==========================

type ArrayStackSerializationTestSuite struct {
	suite.Suite
}

func (s *ArrayStackSerializationTestSuite) TestEmptyStack() {
	s.Run("JSON", func() {
		original := NewArrayStack[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayStack[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewArrayStack[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayStack[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ArrayStackSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewArrayStack[int]()
		original.PushAll(1, 2, 3, 4, 5)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayStack[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewArrayStack[int]()
		original.PushAll(1, 2, 3, 4, 5)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayStack[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestArrayStackSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ArrayStackSerializationTestSuite))
}

// ==========================
// 9. ArrayQueue Serialization Tests
// ==========================

type ArrayQueueSerializationTestSuite struct {
	suite.Suite
}

func (s *ArrayQueueSerializationTestSuite) TestEmptyQueue() {
	s.Run("JSON", func() {
		original := NewArrayQueue[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayQueue[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewArrayQueue[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayQueue[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ArrayQueueSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewArrayQueueFrom(10, 20, 30, 40, 50)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayQueue[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewArrayQueueFrom(10, 20, 30, 40, 50)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayQueue[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestArrayQueueSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ArrayQueueSerializationTestSuite))
}

// ==========================
// 10. ArrayDeque Serialization Tests
// ==========================

type ArrayDequeSerializationTestSuite struct {
	suite.Suite
}

func (s *ArrayDequeSerializationTestSuite) TestEmptyDeque() {
	s.Run("JSON", func() {
		original := NewArrayDeque[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayDeque[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewArrayDeque[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayDeque[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ArrayDequeSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewArrayDequeFrom(5, 10, 15, 20)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewArrayDeque[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})

	s.Run("Gob", func() {
		original := NewArrayDequeFrom(5, 10, 15, 20)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewArrayDeque[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Order should be preserved")
	})
}

func TestArrayDequeSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ArrayDequeSerializationTestSuite))
}

// ==========================
// 11. ConcurrentHashSet Serialization Tests
// ==========================

type ConcurrentHashSetSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentHashSetSerializationTestSuite) TestEmptySet() {
	s.Run("JSON", func() {
		original := NewConcurrentHashSet[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewConcurrentHashSet[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ConcurrentHashSetSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewConcurrentHashSetFrom(100, 200, 300, 400)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentHashSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(100, 200, 300, 400), "Should contain all elements")
	})

	s.Run("Gob", func() {
		original := NewConcurrentHashSetFrom(100, 200, 300, 400)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentHashSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(100, 200, 300, 400), "Should contain all elements")
	})
}

func TestConcurrentHashSetSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentHashSetSerializationTestSuite))
}

// ==========================
// 12. ConcurrentHashMap Serialization Tests
// ==========================

type ConcurrentHashMapSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentHashMapSerializationTestSuite) TestEmptyMap() {
	s.Run("JSON", func() {
		original := NewConcurrentHashMap[string, int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentHashMap[string, int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewConcurrentHashMap[string, int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentHashMap[string, int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ConcurrentHashMapSerializationTestSuite) TestMultipleEntries() {
	s.Run("JSON", func() {
		original := NewConcurrentHashMap[string, int]()
		original.Put("alpha", 1)
		original.Put("beta", 2)
		original.Put("gamma", 3)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentHashMap[string, int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []string{"alpha", "beta", "gamma"} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %s should exist", key)
			s.Equal(origVal, restVal, "Value for key %s should match", key)
		}
	})

	s.Run("Gob", func() {
		original := NewConcurrentHashMap[string, int]()
		original.Put("alpha", 1)
		original.Put("beta", 2)
		original.Put("gamma", 3)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentHashMap[string, int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []string{"alpha", "beta", "gamma"} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %s should exist", key)
			s.Equal(origVal, restVal, "Value for key %s should match", key)
		}
	})
}

func TestConcurrentHashMapSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentHashMapSerializationTestSuite))
}

// ==========================
// 13. ConcurrentSkipSet Serialization Tests
// ==========================

type ConcurrentSkipSetSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentSkipSetSerializationTestSuite) TestEmptySet() {
	s.Run("JSON", func() {
		original := NewConcurrentSkipSet[int]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentSkipSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewConcurrentSkipSet[int]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentSkipSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ConcurrentSkipSetSerializationTestSuite) TestMultipleElements() {
	s.Run("JSON", func() {
		original := NewConcurrentSkipSetFrom(5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentSkipSet[int]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(1, 2, 5, 8, 9), "Should contain all elements")
	})

	s.Run("Gob", func() {
		original := NewConcurrentSkipSetFrom(5, 2, 8, 1, 9)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentSkipSet[int]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.True(restored.ContainsAll(1, 2, 5, 8, 9), "Should contain all elements")
	})
}

func TestConcurrentSkipSetSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentSkipSetSerializationTestSuite))
}

// ==========================
// 14. ConcurrentSkipMap Serialization Tests
// ==========================

type ConcurrentSkipMapSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentSkipMapSerializationTestSuite) TestEmptyMap() {
	s.Run("JSON", func() {
		original := NewConcurrentSkipMap[int, string]()
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentSkipMap[int, string]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})

	s.Run("Gob", func() {
		original := NewConcurrentSkipMap[int, string]()
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentSkipMap[int, string]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(0, restored.Size(), "Size should be zero")
	})
}

func (s *ConcurrentSkipMapSerializationTestSuite) TestMultipleEntries() {
	s.Run("JSON", func() {
		original := NewConcurrentSkipMap[int, string]()
		original.Put(1, "one")
		original.Put(2, "two")
		original.Put(3, "three")
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentSkipMap[int, string]()
		err = json.Unmarshal(data, restored)
		s.Require().NoError(err, "Unmarshal should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []int{1, 2, 3} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %d should exist", key)
			s.Equal(origVal, restVal, "Value for key %d should match", key)
		}
	})

	s.Run("Gob", func() {
		original := NewConcurrentSkipMap[int, string]()
		original.Put(1, "one")
		original.Put(2, "two")
		original.Put(3, "three")
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored := NewConcurrentSkipMap[int, string]()
		err = gob.NewDecoder(&buf).Decode(restored)
		s.Require().NoError(err, "Gob decode should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []int{1, 2, 3} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %d should exist", key)
			s.Equal(origVal, restVal, "Value for key %d should match", key)
		}
	})
}

func TestConcurrentSkipMapSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentSkipMapSerializationTestSuite))
}

// ==========================
// 15. TreeSet Serialization Tests (requires comparator)
// ==========================

type TreeSetSerializationTestSuite struct {
	suite.Suite
}

func (s *TreeSetSerializationTestSuite) TestDirectUnmarshalIntoConstructed() {
	s.Run("JSON", func() {
		original := NewTreeSetFrom(CompareFunc[int](), 5, 2, 8)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewTreeSet(CompareFunc[int]())
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal into constructed set should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Round trip should preserve elements")
	})

	s.Run("Gob", func() {
		original := NewTreeSetFrom(CompareFunc[int](), 5, 2, 8)
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewTreeSet(CompareFunc[int]())
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode into constructed set should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Round trip should preserve elements")
	})
}

func (s *TreeSetSerializationTestSuite) TestOrderedTypeWithHelper() {
	s.Run("JSON", func() {
		original := NewTreeSetFrom(CompareFunc[int](), 5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalTreeSetOrderedJSON[int](data)
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Sorted order should match")
	})

	s.Run("Gob", func() {
		original := NewTreeSetFrom(CompareFunc[int](), 5, 2, 8, 1, 9)

		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalTreeSetOrderedGob[int](buf.Bytes())
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Sorted order should match")
	})
}

func (s *TreeSetSerializationTestSuite) TestCustomComparatorWithHelper() {
	s.Run("JSON", func() {
		reverseCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewTreeSetFrom(reverseCompare, 5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalTreeSetJSON(data, reverseCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Reverse sorted order should match")
	})

	s.Run("Gob", func() {
		reverseCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewTreeSetFrom(reverseCompare, 5, 2, 8, 1, 9)
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalTreeSetGob(buf.Bytes(), reverseCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Reverse sorted order should match")
	})
}

func TestTreeSetSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(TreeSetSerializationTestSuite))
}

// ==========================
// 16. TreeMap Serialization Tests (requires comparator)
// ==========================

type TreeMapSerializationTestSuite struct {
	suite.Suite
}

func (s *TreeMapSerializationTestSuite) TestDirectUnmarshalIntoConstructed() {
	s.Run("JSON", func() {
		original := NewTreeMap[int, string](CompareFunc[int]())
		original.Put(1, "one")
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewTreeMap[int, string](CompareFunc[int]())
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal into constructed map should succeed")
		s.Equal(original.Entries(), restored.Entries(), "Round trip should preserve entries")
	})

	s.Run("Gob", func() {
		original := NewTreeMap[int, string](CompareFunc[int]())
		original.Put(1, "one")
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewTreeMap[int, string](CompareFunc[int]())
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode into constructed map should succeed")
		s.Equal(original.Entries(), restored.Entries(), "Round trip should preserve entries")
	})
}

func (s *TreeMapSerializationTestSuite) TestOrderedKeyTypeWithHelper() {
	s.Run("JSON", func() {
		original := NewTreeMapOrdered[int, string]()
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalTreeMapOrderedJSON[int, string](data)
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []int{1, 2, 3} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %d should exist", key)
			s.Equal(origVal, restVal, "Value for key %d should match", key)
		}
	})

	s.Run("Gob", func() {
		original := NewTreeMapOrdered[int, string]()
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalTreeMapOrderedGob[int, string](buf.Bytes())
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		for _, key := range []int{1, 2, 3} {
			origVal, _ := original.Get(key)
			restVal, ok := restored.Get(key)
			s.Require().True(ok, "Key %d should exist", key)
			s.Equal(origVal, restVal, "Value for key %d should match", key)
		}
	})
}

func (s *TreeMapSerializationTestSuite) TestCustomComparatorWithHelper() {
	s.Run("JSON", func() {
		reverseCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewTreeMap[int, string](reverseCompare)
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalTreeMapJSON[int, string](data, reverseCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.Keys(), restored.Keys(), "Reverse sorted key order should match")
	})

	s.Run("Gob", func() {
		reverseCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewTreeMap[int, string](reverseCompare)
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")

		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalTreeMapGob[int, string](buf.Bytes(), reverseCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")
		s.Equal(original.Keys(), restored.Keys(), "Reverse sorted key order should match")
	})
}

func TestTreeMapSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(TreeMapSerializationTestSuite))
}

// ==========================
// 17. PriorityQueue Serialization Tests (requires comparator)
// ==========================

type PriorityQueueSerializationTestSuite struct {
	suite.Suite
}

func (s *PriorityQueueSerializationTestSuite) TestDirectUnmarshalIntoConstructed() {
	s.Run("JSON", func() {
		original := NewPriorityQueueFrom(CompareFunc[int](), 5, 2, 8)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewPriorityQueueOrdered[int]()
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal into constructed queue should succeed")
		s.Equal(original.ToSortedSlice(), restored.ToSortedSlice(), "Round trip should preserve elements")
	})

	s.Run("Gob", func() {
		original := NewPriorityQueueFrom(CompareFunc[int](), 5, 2, 8)
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewPriorityQueueOrdered[int]()
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode into constructed queue should succeed")
		s.Equal(original.ToSortedSlice(), restored.ToSortedSlice(), "Round trip should preserve elements")
	})
}

func (s *PriorityQueueSerializationTestSuite) TestOrderedTypeWithHelper() {
	s.Run("JSON", func() {
		original := NewPriorityQueueFrom(CompareFunc[int](), 5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalPriorityQueueOrderedJSON[int](data)
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		// Verify all elements are present by popping them
		origSlice := original.ToSortedSlice()
		restSlice := restored.ToSortedSlice()
		s.Equal(origSlice, restSlice, "Elements should match")
	})

	s.Run("Gob", func() {
		original := NewPriorityQueueFrom(CompareFunc[int](), 5, 2, 8, 1, 9)

		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalPriorityQueueOrderedGob[int](buf.Bytes())
		s.Require().NoError(err, "Unmarshal with helper should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		origSlice := original.ToSortedSlice()
		restSlice := restored.ToSortedSlice()
		s.Equal(origSlice, restSlice, "Elements should match")
	})
}

func (s *PriorityQueueSerializationTestSuite) TestCustomComparatorWithHelper() {
	s.Run("JSON", func() {
		maxHeapCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewPriorityQueueFrom(maxHeapCompare, 5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored, err := UnmarshalPriorityQueueJSON(data, maxHeapCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		origSlice := original.ToSortedSlice()
		restSlice := restored.ToSortedSlice()
		s.Equal(origSlice, restSlice, "Max-heap order should match")
	})

	s.Run("Gob", func() {
		maxHeapCompare := func(a, b int) int {
			return CompareFunc[int]()(b, a)
		}
		original := NewPriorityQueueFrom(maxHeapCompare, 5, 2, 8, 1, 9)

		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(original)
		s.Require().NoError(err, "Gob encode should succeed")

		restored, err := UnmarshalPriorityQueueGob(buf.Bytes(), maxHeapCompare)
		s.Require().NoError(err, "Unmarshal with custom comparator should succeed")
		s.Equal(original.Size(), restored.Size(), "Size should match")

		origSlice := original.ToSortedSlice()
		restSlice := restored.ToSortedSlice()
		s.Equal(origSlice, restSlice, "Max-heap order should match")
	})
}

func TestPriorityQueueSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(PriorityQueueSerializationTestSuite))
}

// ==========================
// 18. ConcurrentTreeSet Serialization Tests (requires comparator)
// ==========================

type ConcurrentTreeSetSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentTreeSetSerializationTestSuite) TestDirectUnmarshalIntoConstructed() {
	s.Run("JSON", func() {
		original := NewConcurrentTreeSetFrom(CompareFunc[int](), 5, 2, 8, 1, 9)
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentTreeSet(CompareFunc[int]())
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal into constructed set should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Round trip should preserve elements")
	})

	s.Run("Gob", func() {
		original := NewConcurrentTreeSetFrom(CompareFunc[int](), 5, 2, 8, 1, 9)
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewConcurrentTreeSet(CompareFunc[int]())
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode into constructed set should succeed")
		s.Equal(original.ToSlice(), restored.ToSlice(), "Round trip should preserve elements")
	})
}

func TestConcurrentTreeSetSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentTreeSetSerializationTestSuite))
}

// ==========================
// 19. ConcurrentTreeMap Serialization Tests (requires comparator)
// ==========================

type ConcurrentTreeMapSerializationTestSuite struct {
	suite.Suite
}

func (s *ConcurrentTreeMapSerializationTestSuite) TestDirectUnmarshalIntoConstructed() {
	s.Run("JSON", func() {
		original := NewConcurrentTreeMapOrdered[int, string]()
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")
		data, err := json.Marshal(original)
		s.Require().NoError(err, "Marshal should succeed")

		restored := NewConcurrentTreeMapOrdered[int, string]()
		s.Require().NoError(json.Unmarshal(data, restored), "Unmarshal into constructed map should succeed")
		s.Equal(original.Entries(), restored.Entries(), "Round trip should preserve entries")
	})

	s.Run("Gob", func() {
		original := NewConcurrentTreeMapOrdered[int, string]()
		original.Put(3, "three")
		original.Put(1, "one")
		original.Put(2, "two")
		var buf bytes.Buffer
		s.Require().NoError(gob.NewEncoder(&buf).Encode(original), "Gob encode should succeed")

		restored := NewConcurrentTreeMapOrdered[int, string]()
		s.Require().NoError(gob.NewDecoder(&buf).Decode(restored), "Gob decode into constructed map should succeed")
		s.Equal(original.Entries(), restored.Entries(), "Round trip should preserve entries")
	})
}

func TestConcurrentTreeMapSerializationTestSuite(t *testing.T) {
	suite.Run(t, new(ConcurrentTreeMapSerializationTestSuite))
}

// ==========================
// Error Handling Tests
// ==========================

type SerializationErrorHandlingTestSuite struct {
	suite.Suite
}

// Test invalid JSON data for direct serialization types.
func (s *SerializationErrorHandlingTestSuite) TestInvalidJSONData() {
	s.Run("HashSet", func() {
		set := NewHashSet[int]()
		err := json.Unmarshal([]byte("invalid json"), set)
		s.Error(err, "Should fail on invalid JSON")
	})

	s.Run("ArrayList", func() {
		list := NewArrayList[int]()
		err := json.Unmarshal([]byte("{not an array}"), list)
		s.Error(err, "Should fail on invalid JSON")
	})

	s.Run("HashMap", func() {
		m := NewHashMap[string, int]()
		err := json.Unmarshal([]byte("not json"), m)
		s.Error(err, "Should fail on invalid JSON")
	})
}

// Test invalid Gob data for direct serialization types.
func (s *SerializationErrorHandlingTestSuite) TestInvalidGobData() {
	s.Run("HashSet", func() {
		set := NewHashSet[int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(set)
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("ArrayList", func() {
		list := NewArrayList[int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(list)
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("HashMap", func() {
		m := NewHashMap[string, int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(m)
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("ArrayStack", func() {
		stack := NewArrayStack[int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(stack)
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("ArrayQueue", func() {
		queue := NewArrayQueue[int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(queue)
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("ArrayDeque", func() {
		deque := NewArrayDeque[int]()
		err := gob.NewDecoder(bytes.NewReader([]byte("invalid gob data"))).Decode(deque)
		s.Error(err, "Should fail on invalid Gob data")
	})
}

// Test nil comparator for helper functions.
func (s *SerializationErrorHandlingTestSuite) TestNilComparatorError() {
	s.Run("UnmarshalTreeSetJSON", func() {
		data, _ := json.Marshal([]int{1, 2, 3})
		_, err := UnmarshalTreeSetJSON[int](data, nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})

	s.Run("UnmarshalTreeMapJSON", func() {
		data, _ := json.Marshal(serializableMap[int, string]{
			Entries: []serializableEntry[int, string]{{Key: 1, Value: "one"}},
		})
		_, err := UnmarshalTreeMapJSON[int, string](data, nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})

	s.Run("UnmarshalTreeSetGob", func() {
		var buf bytes.Buffer
		_ = gob.NewEncoder(&buf).Encode([]int{1, 2, 3})
		_, err := UnmarshalTreeSetGob[int](buf.Bytes(), nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})

	s.Run("UnmarshalTreeMapGob", func() {
		entries := []serializableEntry[int, string]{{Key: 1, Value: "one"}}
		var buf bytes.Buffer
		_ = gob.NewEncoder(&buf).Encode(entries)
		_, err := UnmarshalTreeMapGob[int, string](buf.Bytes(), nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})

	s.Run("UnmarshalPriorityQueueJSON", func() {
		data, _ := json.Marshal([]int{1, 2, 3})
		_, err := UnmarshalPriorityQueueJSON[int](data, nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})

	s.Run("UnmarshalPriorityQueueGob", func() {
		var buf bytes.Buffer
		_ = gob.NewEncoder(&buf).Encode([]int{1, 2, 3})
		_, err := UnmarshalPriorityQueueGob[int](buf.Bytes(), nil)
		s.Require().Error(err, "Should fail with nil comparator")
		s.Contains(err.Error(), "comparator required")
	})
}

// Test invalid JSON data for helper functions.
func (s *SerializationErrorHandlingTestSuite) TestHelperFunctionsWithInvalidJSON() {
	s.Run("UnmarshalTreeSetJSON", func() {
		_, err := UnmarshalTreeSetJSON([]byte("invalid json"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid JSON")
	})

	s.Run("UnmarshalTreeMapJSON", func() {
		_, err := UnmarshalTreeMapJSON[int, string]([]byte("invalid json"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid JSON")
	})

	s.Run("UnmarshalPriorityQueueJSON", func() {
		_, err := UnmarshalPriorityQueueJSON([]byte("invalid json"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid JSON")
	})
}

// Test invalid Gob data for helper functions.
func (s *SerializationErrorHandlingTestSuite) TestHelperFunctionsWithInvalidGob() {
	s.Run("UnmarshalTreeSetGob", func() {
		_, err := UnmarshalTreeSetGob([]byte("invalid gob"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("UnmarshalTreeMapGob", func() {
		_, err := UnmarshalTreeMapGob[int, string]([]byte("invalid gob"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid Gob data")
	})

	s.Run("UnmarshalPriorityQueueGob", func() {
		_, err := UnmarshalPriorityQueueGob([]byte("invalid gob"), CompareFunc[int]())
		s.Error(err, "Should fail on invalid Gob data")
	})
}

func TestSerializationErrorHandlingTestSuite(t *testing.T) {
	suite.Run(t, new(SerializationErrorHandlingTestSuite))
}

// A zero-value comparator-carrying collection cannot decode; the error should
// point at the constructor.
func (s *SerializationErrorHandlingTestSuite) TestDecodeWithoutComparator() {
	s.Run("TreeSet", func() {
		var zero treeSet[int]
		s.Require().ErrorContains(zero.UnmarshalJSON([]byte("[1]")), "no comparator", "JSON decode should require a comparator")
		s.Require().ErrorContains(zero.GobDecode(nil), "no comparator", "Gob decode should require a comparator")
	})

	s.Run("TreeMap", func() {
		var zero treeMap[int, string]
		s.Require().ErrorContains(zero.UnmarshalJSON([]byte("{}")), "no comparator", "JSON decode should require a comparator")
		s.Require().ErrorContains(zero.GobDecode(nil), "no comparator", "Gob decode should require a comparator")
	})

	s.Run("PriorityQueue", func() {
		var zero priorityQueue[int]
		s.Require().ErrorContains(zero.UnmarshalJSON([]byte("[1]")), "no comparator", "JSON decode should require a comparator")
		s.Require().ErrorContains(zero.GobDecode(nil), "no comparator", "Gob decode should require a comparator")
	})

	s.Run("ConcurrentTreeSet", func() {
		var zero concurrentTreeSet[int]
		s.Require().ErrorContains(zero.UnmarshalJSON([]byte("[1]")), "no comparator", "JSON decode should require a comparator")
		s.Require().ErrorContains(zero.GobDecode(nil), "no comparator", "Gob decode should require a comparator")
	})

	s.Run("ConcurrentTreeMap", func() {
		var zero concurrentTreeMap[int, string]
		s.Require().ErrorContains(zero.UnmarshalJSON([]byte("{}")), "no comparator", "JSON decode should require a comparator")
		s.Require().ErrorContains(zero.GobDecode(nil), "no comparator", "Gob decode should require a comparator")
	})
}
