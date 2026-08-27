package collections

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// ==========================
// Serialization Helpers
// ==========================

// serializableEntry is used for serializing maps with non-comparable keys.
type serializableEntry[K, V any] struct {
	Key   K `json:"key"`
	Value V `json:"value"`
}

// serializableMap wraps map entries for JSON serialization.
type serializableMap[K, V any] struct {
	Entries []serializableEntry[K, V] `json:"entries"`
}

// validateHashable returns an error naming the first element of elems that
// cannot be used as a hash-map key. T may be an interface or contain one
// (any satisfies comparable since Go 1.20), so a decoded payload can carry a
// dynamically unhashable value — storing it would panic after the receiver
// was already partially mutated, so decode paths reject the whole payload up
// front. A nil interface value is a valid key.
func validateHashable[T comparable](elems []T) error {
	for i, e := range elems {
		if rv := reflect.ValueOf(e); rv.IsValid() && !rv.Comparable() {
			return fmt.Errorf("element %d has unhashable type %T", i, e)
		}
	}
	return nil
}

// ==========================
// TreeSet/TreeMap Deserialization Helpers
// ==========================
//
// Comparator-carrying collections (TreeSet, TreeMap, PriorityQueue and their
// concurrent wrappers) can only be gob-decoded into a receiver that already
// has its comparator: either construct one and Decode into it, or use the
// Unmarshal*Gob helpers below. This does not extend to nesting — a field of
// such a type inside another gob-encoded struct is decoded into a zero-value
// receiver with no comparator and fails with a "no comparator" error. Decode
// nested collections separately from raw bytes instead.

// UnmarshalTreeSetJSON deserializes a TreeSet from JSON.
// Requires a comparator to be provided.
func UnmarshalTreeSetJSON[T any](data []byte, comparator Comparator[T]) (SortedSet[T], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal treeset: comparator required")
	}

	var elements []T
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("unmarshal treeset: %w", err)
	}

	set := NewTreeSet(comparator)
	set.AddAll(elements...)
	return set, nil
}

// UnmarshalTreeSetOrderedJSON deserializes a TreeSet from JSON for Ordered types.
// Uses cmp.Compare as the default comparator.
func UnmarshalTreeSetOrderedJSON[T Ordered](data []byte) (SortedSet[T], error) {
	return UnmarshalTreeSetJSON(data, CompareFunc[T]())
}

// UnmarshalTreeMapJSON deserializes a TreeMap from JSON.
// Requires a comparator to be provided.
func UnmarshalTreeMapJSON[K, V any](data []byte, comparator Comparator[K]) (SortedMap[K, V], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal treemap: comparator required")
	}

	var wrapped serializableMap[K, V]
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, fmt.Errorf("unmarshal treemap: %w", err)
	}

	m := NewTreeMap[K, V](comparator)
	for _, entry := range wrapped.Entries {
		m.Put(entry.Key, entry.Value)
	}
	return m, nil
}

// UnmarshalTreeMapOrderedJSON deserializes a TreeMap from JSON for Ordered key types.
// Uses cmp.Compare as the default comparator for keys.
func UnmarshalTreeMapOrderedJSON[K Ordered, V any](data []byte) (SortedMap[K, V], error) {
	return UnmarshalTreeMapJSON[K, V](data, CompareFunc[K]())
}

// UnmarshalTreeSetGob deserializes a TreeSet from Gob.
// Requires a comparator to be provided. The data is the output of a standard
// gob.Encoder run over the set, i.e. gob.NewEncoder(w).Encode(set).
func UnmarshalTreeSetGob[T any](data []byte, comparator Comparator[T]) (SortedSet[T], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal treeset gob: comparator required")
	}

	set := newTreeSet(comparator)
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(set); err != nil {
		return nil, fmt.Errorf("unmarshal treeset gob: %w", err)
	}
	return set, nil
}

// UnmarshalTreeSetOrderedGob deserializes a TreeSet from Gob for Ordered types.
// Uses cmp.Compare as the default comparator.
func UnmarshalTreeSetOrderedGob[T Ordered](data []byte) (SortedSet[T], error) {
	return UnmarshalTreeSetGob(data, CompareFunc[T]())
}

// UnmarshalTreeMapGob deserializes a TreeMap from Gob.
// Requires a comparator to be provided. The data is the output of a standard
// gob.Encoder run over the map, i.e. gob.NewEncoder(w).Encode(m).
func UnmarshalTreeMapGob[K, V any](data []byte, comparator Comparator[K]) (SortedMap[K, V], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal treemap gob: comparator required")
	}

	m := newTreeMap[K, V](comparator)
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(m); err != nil {
		return nil, fmt.Errorf("unmarshal treemap gob: %w", err)
	}
	return m, nil
}

// UnmarshalTreeMapOrderedGob deserializes a TreeMap from Gob for Ordered key types.
// Uses cmp.Compare as the default comparator for keys.
func UnmarshalTreeMapOrderedGob[K Ordered, V any](data []byte) (SortedMap[K, V], error) {
	return UnmarshalTreeMapGob[K, V](data, CompareFunc[K]())
}

// UnmarshalPriorityQueueJSON deserializes a PriorityQueue from JSON.
// Requires a comparator to be provided.
func UnmarshalPriorityQueueJSON[T any](data []byte, comparator Comparator[T]) (PriorityQueue[T], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal priorityqueue: comparator required")
	}

	var elements []T
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("unmarshal priorityqueue: %w", err)
	}

	pq := NewPriorityQueue(comparator)
	pq.PushAll(elements...)
	return pq, nil
}

// UnmarshalPriorityQueueOrderedJSON deserializes a PriorityQueue from JSON for Ordered types.
// Uses cmp.Compare as the default comparator.
func UnmarshalPriorityQueueOrderedJSON[T Ordered](data []byte) (PriorityQueue[T], error) {
	return UnmarshalPriorityQueueJSON(data, CompareFunc[T]())
}

// UnmarshalPriorityQueueGob deserializes a PriorityQueue from Gob.
// Requires a comparator to be provided. The data is the output of a standard
// gob.Encoder run over the queue, i.e. gob.NewEncoder(w).Encode(pq).
func UnmarshalPriorityQueueGob[T any](data []byte, comparator Comparator[T]) (PriorityQueue[T], error) {
	if comparator == nil {
		return nil, errors.New("unmarshal priorityqueue gob: comparator required")
	}

	pq := newPriorityQueue(comparator)
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(pq); err != nil {
		return nil, fmt.Errorf("unmarshal priorityqueue gob: %w", err)
	}
	return pq, nil
}

// UnmarshalPriorityQueueOrderedGob deserializes a PriorityQueue from Gob for Ordered types.
// Uses cmp.Compare as the default comparator.
func UnmarshalPriorityQueueOrderedGob[T Ordered](data []byte) (PriorityQueue[T], error) {
	return UnmarshalPriorityQueueGob(data, CompareFunc[T]())
}
