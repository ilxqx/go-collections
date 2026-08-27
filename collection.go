package collections

import (
	"cmp"
	"fmt"
	"iter"
	"strings"
)

// =========================
// Core Interfaces
// =========================

// Collection is the root interface for all collections.
// It exposes minimal, common capabilities that do not assume ordering.
type Collection[T any] interface {
	// Size returns the number of elements.
	Size() int
	// IsEmpty reports whether the collection contains no elements.
	IsEmpty() bool
	// IsNotEmpty reports whether the collection contains at least one element.
	IsNotEmpty() bool
	// Clear removes all elements from the collection.
	Clear()
	// ToSlice returns a snapshot slice of all elements.
	// The order depends on the concrete implementation.
	ToSlice() []T
	// String returns a string representation of the collection.
	String() string
}

// Iterable represents a collection that can be iterated.
// It provides Go 1.23+ for-range support via iter.Seq.
type Iterable[T any] interface {
	// Seq returns a sequence for use with for-range:
	//   for v := range c.Seq() { ... }
	Seq() iter.Seq[T]
	// ForEach applies action to each element.
	// Iteration stops early if action returns false.
	ForEach(action func(element T) bool)
}

// ============
// Set Interfaces
// ============

// Set is an unordered collection of unique elements.
// Ordering is not guaranteed unless a SortedSet is used.
type Set[T any] interface {
	Collection[T]
	Iterable[T]

	// --- Modification ---
	// Add inserts the element if absent. Returns true if the set changed.
	Add(element T) bool
	// AddAll inserts all given elements. Returns the number of elements added.
	AddAll(elements ...T) int
	// AddSeq inserts all elements from the sequence. Returns the number added.
	AddSeq(seq iter.Seq[T]) int
	// Remove deletes the element if present. Returns true if removed.
	Remove(element T) bool
	// RemoveAll deletes all given elements. Returns the number removed.
	RemoveAll(elements ...T) int
	// RemoveSeq removes all elements from the sequence. Returns the number removed.
	RemoveSeq(seq iter.Seq[T]) int
	// RemoveFunc removes elements satisfying predicate. Returns count removed.
	RemoveFunc(predicate func(element T) bool) int
	// RetainFunc keeps only elements satisfying predicate. Returns count removed.
	RetainFunc(predicate func(element T) bool) int
	// Pop removes and returns an arbitrary element. Returns (zero, false) if empty.
	Pop() (T, bool)

	// --- Query ---
	// Contains reports whether element exists in the set.
	Contains(element T) bool
	// ContainsAll reports whether all elements exist in the set.
	ContainsAll(elements ...T) bool
	// ContainsAny reports whether any of the elements exist in the set.
	ContainsAny(elements ...T) bool

	// --- Set Algebra ---
	// Union returns a new set: s ∪ other.
	Union(other Set[T]) Set[T]
	// Intersection returns a new set: s ∩ other.
	Intersection(other Set[T]) Set[T]
	// Difference returns a new set: s - other.
	Difference(other Set[T]) Set[T]
	// SymmetricDifference returns a new set: (s - other) ∪ (other - s).
	SymmetricDifference(other Set[T]) Set[T]

	// --- Relations ---
	// IsSubsetOf reports whether all elements of s are in other.
	IsSubsetOf(other Set[T]) bool
	// IsSupersetOf reports whether s contains all elements of other.
	IsSupersetOf(other Set[T]) bool
	// IsProperSubsetOf reports whether s ⊂ other (subset but not equal).
	IsProperSubsetOf(other Set[T]) bool
	// IsProperSupersetOf reports whether s ⊃ other (superset but not equal).
	IsProperSupersetOf(other Set[T]) bool
	// IsDisjoint reports whether s and other have no elements in common.
	IsDisjoint(other Set[T]) bool
	// Equals reports whether s and other contain exactly the same elements.
	//
	// Equals and the other cross-set predicates and operations test the
	// receiver's elements with other.Contains, i.e. other's equality (a
	// comparable type's ==, a TreeSet's comparator) decides membership.
	// The results are deterministic, but when the two sets disagree on
	// which elements are equal — say a case-insensitive TreeSet and a
	// HashSet — they can be asymmetric (a.Equals(b) != b.Equals(a)); only
	// combine sets that share an equivalence relation.
	Equals(other Set[T]) bool

	// --- Transformations ---
	// Clone returns a shallow copy.
	Clone() Set[T]
	// Filter returns a new set of elements that satisfy predicate.
	Filter(predicate func(element T) bool) Set[T]

	// --- Search ---
	// Find returns the first element satisfying predicate, or (zero, false).
	Find(predicate func(element T) bool) (T, bool)
	// Any returns true if at least one element satisfies predicate.
	Any(predicate func(element T) bool) bool
	// Every returns true if all elements satisfy predicate (true for empty set).
	Every(predicate func(element T) bool) bool
}

// SortedSet is a Set that maintains elements in sorted order.
// Implementations may require T to satisfy Ordered, or accept a Comparator[T].
type SortedSet[T any] interface {
	Set[T]

	// --- Extremes ---
	// First returns the smallest element, or (zero, false) if empty.
	First() (T, bool)
	// Last returns the largest element, or (zero, false) if empty.
	Last() (T, bool)
	// Min is an alias of First for compatibility with other libraries.
	Min() (T, bool)
	// Max is an alias of Last for compatibility with other libraries.
	Max() (T, bool)
	// PopFirst removes and returns the smallest element.
	PopFirst() (T, bool)
	// PopLast removes and returns the largest element.
	PopLast() (T, bool)

	// --- Navigation ---
	// Floor returns the greatest element <= x, or (zero, false).
	Floor(x T) (T, bool)
	// Ceiling returns the smallest element >= x, or (zero, false).
	Ceiling(x T) (T, bool)
	// Lower returns the greatest element < x, or (zero, false).
	Lower(x T) (T, bool)
	// Higher returns the smallest element > x, or (zero, false).
	Higher(x T) (T, bool)

	// --- Range Iteration ---
	// Range iterates elements in [from, to] ascending.
	// If from > to, no elements are visited.
	Range(from, to T, action func(element T) bool)
	// RangeSeq returns a sequence for elements in [from, to] ascending.
	// If from > to, returns an empty sequence.
	RangeSeq(from, to T) iter.Seq[T]

	// --- Ordered Iteration ---
	// Ascend iterates all elements in ascending order.
	Ascend(action func(element T) bool)
	// Descend iterates all elements in descending order.
	Descend(action func(element T) bool)
	// AscendFrom iterates elements >= pivot in ascending order.
	AscendFrom(pivot T, action func(element T) bool)
	// DescendFrom iterates elements <= pivot in descending order.
	DescendFrom(pivot T, action func(element T) bool)
	// Reversed returns a descending sequence for for-range.
	Reversed() iter.Seq[T]

	// --- Views ---
	// SubSet returns a new set containing elements in [from, to].
	SubSet(from, to T) SortedSet[T]
	// HeadSet returns elements < to (or <= if inclusive).
	HeadSet(to T, inclusive bool) SortedSet[T]
	// TailSet returns elements > from (or >= if inclusive).
	TailSet(from T, inclusive bool) SortedSet[T]

	// --- Rank Operations ---
	// Rank returns the 0-based rank of x, or -1 if not present.
	Rank(x T) int
	// GetByRank returns the element at 0-based rank, or (zero, false).
	GetByRank(rank int) (T, bool)

	// --- Sorted Clone ---
	// CloneSorted returns a shallow copy as SortedSet.
	CloneSorted() SortedSet[T]
}

// ============
// List Interface
// ============

// List is an ordered collection that allows duplicate elements.
// Elements are accessible by integer index (0-based).
type List[T any] interface {
	Collection[T]
	Iterable[T]

	// --- Positional Access ---
	// Get returns the element at index, or (zero, false) if out of bounds.
	Get(index int) (T, bool)
	// Set replaces the element at index. Returns (oldValue, true) if successful.
	Set(index int, element T) (T, bool)
	// First returns the first element, or (zero, false) if empty.
	First() (T, bool)
	// Last returns the last element, or (zero, false) if empty.
	Last() (T, bool)

	// --- Modification ---
	// Add appends the element to the end.
	Add(element T)
	// AddAll appends all elements to the end.
	AddAll(elements ...T)
	// AddSeq appends all elements from the sequence.
	AddSeq(seq iter.Seq[T])
	// Insert inserts the element at index, shifting subsequent elements right.
	// Returns false if index is out of bounds.
	Insert(index int, element T) bool
	// InsertAll inserts all elements at index, shifting subsequent elements right.
	InsertAll(index int, elements ...T) bool
	// RemoveAt removes the element at index. Returns (removed, true) if successful.
	RemoveAt(index int) (T, bool)
	// Remove removes the first occurrence of element. Returns true if removed.
	Remove(element T, eq Equaler[T]) bool
	// RemoveFirst removes and returns the first element.
	RemoveFirst() (T, bool)
	// RemoveLast removes and returns the last element.
	RemoveLast() (T, bool)
	// RemoveFunc removes all elements satisfying predicate. Returns count removed.
	RemoveFunc(predicate func(element T) bool) int
	// RetainFunc keeps only elements satisfying predicate. Returns count removed.
	RetainFunc(predicate func(element T) bool) int

	// --- Search ---
	// IndexOf returns the index of the first occurrence, or -1.
	IndexOf(element T, eq Equaler[T]) int
	// LastIndexOf returns the index of the last occurrence, or -1.
	LastIndexOf(element T, eq Equaler[T]) int
	// Contains reports whether element exists using eq for comparison.
	Contains(element T, eq Equaler[T]) bool
	// Find returns the first element satisfying predicate, or (zero, false).
	Find(predicate func(element T) bool) (T, bool)
	// FindIndex returns the index of the first element satisfying predicate, or -1.
	FindIndex(predicate func(element T) bool) int

	// --- Views ---
	// SubList returns a new list containing elements in [from, to).
	SubList(from, to int) List[T]
	// Reversed returns a sequence iterating in reverse order.
	Reversed() iter.Seq[T]

	// --- Transformations ---
	// Clone returns a shallow copy.
	Clone() List[T]
	// Filter returns a new list of elements satisfying predicate.
	Filter(predicate func(element T) bool) List[T]
	// Sort sorts elements in place using the comparator.
	Sort(comparator Comparator[T])

	// --- Aggregation ---
	// Any returns true if at least one element satisfies predicate.
	Any(predicate func(element T) bool) bool
	// Every returns true if all elements satisfy predicate.
	Every(predicate func(element T) bool) bool
}

// ============
// Stack Interface
// ============

// Stack is a LIFO (last-in-first-out) collection.
type Stack[T any] interface {
	// Size returns the number of elements.
	Size() int
	// IsEmpty reports whether the stack is empty.
	IsEmpty() bool
	// IsNotEmpty reports whether the stack contains at least one element.
	IsNotEmpty() bool
	// Clear removes all elements.
	Clear()
	// String returns a string representation.
	String() string

	// Push adds an element to the top of the stack.
	Push(element T)
	// PushAll adds all elements to the top (last element becomes top).
	PushAll(elements ...T)
	// Pop removes and returns the top element, or (zero, false) if empty.
	Pop() (T, bool)
	// Peek returns the top element without removing it, or (zero, false) if empty.
	Peek() (T, bool)

	// ToSlice returns elements from bottom to top.
	ToSlice() []T
	// Seq returns a sequence from top to bottom (LIFO order).
	Seq() iter.Seq[T]
}

// ============
// Queue Interface
// ============

// Queue is a FIFO (first-in-first-out) collection.
type Queue[T any] interface {
	// Size returns the number of elements.
	Size() int
	// IsEmpty reports whether the queue is empty.
	IsEmpty() bool
	// IsNotEmpty reports whether the queue contains at least one element.
	IsNotEmpty() bool
	// Clear removes all elements.
	Clear()
	// String returns a string representation.
	String() string

	// Enqueue adds an element to the back of the queue.
	Enqueue(element T)
	// EnqueueAll adds all elements to the back.
	EnqueueAll(elements ...T)
	// Dequeue removes and returns the front element, or (zero, false) if empty.
	Dequeue() (T, bool)
	// Peek returns the front element without removing it, or (zero, false) if empty.
	Peek() (T, bool)

	// ToSlice returns elements from front to back.
	ToSlice() []T
	// Seq returns a sequence from front to back (FIFO order).
	Seq() iter.Seq[T]
}

// ============
// Deque Interface
// ============

// Deque is a double-ended queue supporting insertion and removal at both ends.
type Deque[T any] interface {
	// Size returns the number of elements.
	Size() int
	// IsEmpty reports whether the deque is empty.
	IsEmpty() bool
	// IsNotEmpty reports whether the deque contains at least one element.
	IsNotEmpty() bool
	// Clear removes all elements.
	Clear()
	// String returns a string representation.
	String() string

	// --- Front Operations ---
	// PushFront adds an element to the front.
	PushFront(element T)
	// PopFront removes and returns the front element, or (zero, false) if empty.
	PopFront() (T, bool)
	// PeekFront returns the front element without removing it, or (zero, false) if empty.
	PeekFront() (T, bool)

	// --- Back Operations ---
	// PushBack adds an element to the back.
	PushBack(element T)
	// PopBack removes and returns the back element, or (zero, false) if empty.
	PopBack() (T, bool)
	// PeekBack returns the back element without removing it, or (zero, false) if empty.
	PeekBack() (T, bool)

	// ToSlice returns elements from front to back.
	ToSlice() []T
	// Seq returns a sequence from front to back.
	Seq() iter.Seq[T]
	// Reversed returns a sequence from back to front.
	Reversed() iter.Seq[T]
}

// ============
// PriorityQueue Interface
// ============

// PriorityQueue is a queue where elements are ordered by priority.
// The element with the highest priority (according to the comparator) is always at the front.
// By default, a min-heap is used (smallest element has highest priority).
// Use a reverse comparator for max-heap behavior.
type PriorityQueue[T any] interface {
	// Size returns the number of elements.
	Size() int
	// IsEmpty reports whether the queue is empty.
	IsEmpty() bool
	// IsNotEmpty reports whether the queue contains at least one element.
	IsNotEmpty() bool
	// Clear removes all elements.
	Clear()
	// String returns a string representation.
	String() string

	// Push adds an element to the queue. O(log n).
	Push(element T)
	// PushAll adds all elements to the queue.
	PushAll(elements ...T)
	// Pop removes and returns the highest-priority element, or (zero, false) if empty. O(log n).
	Pop() (T, bool)
	// Peek returns the highest-priority element without removing it, or (zero, false) if empty. O(1).
	Peek() (T, bool)

	// ToSlice returns elements in heap order (not sorted).
	ToSlice() []T
	// ToSortedSlice returns elements in priority order (sorted).
	ToSortedSlice() []T
	// Seq returns a sequence in heap order (not sorted).
	Seq() iter.Seq[T]
}

// ============
// Map Interfaces
// ============

// Entry represents a key-value pair.
type Entry[K, V any] struct {
	Key   K
	Value V
}

// Unpack returns the key and value for convenient destructuring.
func (e Entry[K, V]) Unpack() (K, V) { return e.Key, e.Value }

// Map is a collection that maps keys to values (unique keys).
type Map[K, V any] interface {
	// --- Basic Info ---
	// Size returns the number of entries.
	Size() int
	// IsEmpty reports whether the map has no entries.
	IsEmpty() bool
	// IsNotEmpty reports whether the map contains at least one entry.
	IsNotEmpty() bool
	// Clear removes all entries.
	Clear()
	// String returns a string representation.
	String() string

	// --- Basic Operations ---
	// Put associates value with key. Returns (oldValue, true) if key existed.
	Put(key K, value V) (V, bool)
	// PutIfAbsent stores value only if key is absent. Returns (existingOrNew, inserted).
	PutIfAbsent(key K, value V) (V, bool)
	// PutAll copies all entries from other into this map.
	PutAll(other Map[K, V])
	// PutSeq copies entries from a sequence. Returns number of keys changed.
	PutSeq(seq iter.Seq2[K, V]) int
	// Get returns (value, true) if key present; otherwise (zero, false).
	Get(key K) (V, bool)
	// GetOrDefault returns value for key or defaultValue if absent.
	GetOrDefault(key K, defaultValue V) V
	// Remove deletes key. Returns (oldValue, true) if key existed.
	Remove(key K) (V, bool)
	// RemoveIf deletes only if (key, value) matches. Returns true if removed.
	RemoveIf(key K, value V, eq Equaler[V]) bool

	// --- Query ---
	// ContainsKey reports whether key exists.
	ContainsKey(key K) bool
	// ContainsValue reports whether value exists (O(n)).
	ContainsValue(value V, eq Equaler[V]) bool

	// --- Bulk Operations ---
	// RemoveAll removes all specified keys. Returns count removed.
	RemoveAll(keys ...K) int
	// RemoveSeq removes keys from the sequence. Returns count removed.
	RemoveSeq(seq iter.Seq[K]) int
	// RemoveFunc removes entries where predicate returns true. Returns count removed.
	RemoveFunc(predicate func(key K, value V) bool) int

	// --- Compute Operations ---
	// In concurrent implementations the callbacks below may run while an
	// internal lock is held; each implementation documents whether a
	// callback may call back into the same map.
	//
	// Compute recomputes mapping for key.
	// If keep==false, the key is removed.
	Compute(key K, remapping func(key K, oldValue V, exists bool) (newValue V, keep bool)) (V, bool)
	// ComputeIfAbsent computes and stores value if key is absent.
	ComputeIfAbsent(key K, mapping func(key K) V) V
	// ComputeIfPresent recomputes value if key is present. If keep==false, removes key.
	ComputeIfPresent(key K, remapping func(key K, oldValue V) (newValue V, keep bool)) (V, bool)
	// Merge merges value with existing. If keep==false, removes key.
	Merge(key K, value V, remapping func(oldValue, newValue V) (mergedValue V, keep bool)) (V, bool)

	// --- Replace Operations ---
	// Replace sets value only if key is present. Returns (oldValue, true) if replaced.
	Replace(key K, value V) (V, bool)
	// ReplaceIf replaces only if current value equals oldValue. Returns true if replaced.
	ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool
	// ReplaceAll replaces each value with function(key, value).
	ReplaceAll(function func(key K, value V) V)

	// --- Views ---
	// Keys returns all keys as a slice.
	Keys() []K
	// Values returns all values as a slice.
	Values() []V
	// Entries returns all entries as a slice.
	Entries() []Entry[K, V]

	// --- Iteration ---
	// ForEach iterates over entries. Stops early if action returns false.
	ForEach(action func(key K, value V) bool)
	// Seq returns a sequence of (key, value) pairs.
	Seq() iter.Seq2[K, V]
	// SeqKeys returns a sequence of keys.
	SeqKeys() iter.Seq[K]
	// SeqValues returns a sequence of values.
	SeqValues() iter.Seq[V]

	// --- Transformations ---
	// Clone returns a shallow copy.
	Clone() Map[K, V]
	// Filter returns a new map with entries satisfying predicate.
	Filter(predicate func(key K, value V) bool) Map[K, V]

	// --- Comparison ---
	// Equals reports whether both maps contain the same entries.
	//
	// Key membership is tested with each map's own key equality (a
	// comparable type's ==, a TreeMap's comparator). When the two maps
	// disagree on which keys are equal, the result can be asymmetric
	// (a.Equals(b) != b.Equals(a)); only compare maps that share a key
	// equivalence relation.
	Equals(other Map[K, V], valueEq Equaler[V]) bool
}

// GoMapView marks implementations that can expose a native Go map snapshot.
// This requires comparable keys and is intentionally separate from Map[K,V].
type GoMapView[K comparable, V any] interface {
	// ToGoMap returns a standard Go map[K]V snapshot.
	ToGoMap() map[K]V
}

// SortedMap is a Map that maintains entries in sorted key order.
type SortedMap[K, V any] interface {
	Map[K, V]

	// --- Extremes ---
	// FirstKey returns the smallest key, or (zero, false) if empty.
	FirstKey() (K, bool)
	// LastKey returns the largest key, or (zero, false) if empty.
	LastKey() (K, bool)
	// FirstEntry returns the entry with the smallest key.
	FirstEntry() (Entry[K, V], bool)
	// LastEntry returns the entry with the largest key.
	LastEntry() (Entry[K, V], bool)
	// PopFirst removes and returns the smallest-key entry.
	PopFirst() (Entry[K, V], bool)
	// PopLast removes and returns the largest-key entry.
	PopLast() (Entry[K, V], bool)

	// --- Navigation ---
	// FloorKey returns the greatest key <= k, or (zero, false).
	FloorKey(k K) (K, bool)
	// CeilingKey returns the smallest key >= k, or (zero, false).
	CeilingKey(k K) (K, bool)
	// LowerKey returns the greatest key < k, or (zero, false).
	LowerKey(k K) (K, bool)
	// HigherKey returns the smallest key > k, or (zero, false).
	HigherKey(k K) (K, bool)
	// FloorEntry returns entry with greatest key <= k.
	FloorEntry(k K) (Entry[K, V], bool)
	// CeilingEntry returns entry with smallest key >= k.
	CeilingEntry(k K) (Entry[K, V], bool)
	// LowerEntry returns entry with greatest key < k.
	LowerEntry(k K) (Entry[K, V], bool)
	// HigherEntry returns entry with smallest key > k.
	HigherEntry(k K) (Entry[K, V], bool)

	// --- Range Iteration ---
	// Range iterates entries with keys in [from, to] ascending.
	// If from > to, no elements are visited.
	Range(from, to K, action func(key K, value V) bool)
	// RangeSeq returns a sequence for entries with keys in [from, to] ascending.
	// If from > to, returns an empty sequence.
	RangeSeq(from, to K) iter.Seq2[K, V]
	// RangeFrom iterates entries with keys >= from.
	RangeFrom(from K, action func(key K, value V) bool)
	// RangeTo iterates entries with keys <= to.
	RangeTo(to K, action func(key K, value V) bool)

	// --- Ordered Iteration ---
	// Ascend iterates all entries in ascending key order.
	Ascend(action func(key K, value V) bool)
	// Descend iterates all entries in descending key order.
	Descend(action func(key K, value V) bool)
	// AscendFrom iterates entries with keys >= pivot ascending.
	AscendFrom(pivot K, action func(key K, value V) bool)
	// DescendFrom iterates entries with keys <= pivot descending.
	DescendFrom(pivot K, action func(key K, value V) bool)
	// Reversed returns a sequence iterating in descending key order.
	Reversed() iter.Seq2[K, V]

	// --- Views ---
	// SubMap returns entries with keys in [from, to].
	SubMap(from, to K) SortedMap[K, V]
	// HeadMap returns entries with keys < to (or <= if inclusive).
	HeadMap(to K, inclusive bool) SortedMap[K, V]
	// TailMap returns entries with keys > from (or >= if inclusive).
	TailMap(from K, inclusive bool) SortedMap[K, V]

	// --- Rank Operations ---
	// RankOfKey returns the 0-based rank of key, or -1 if not present.
	RankOfKey(key K) int
	// GetByRank returns the entry at rank, or (Entry{}, false).
	GetByRank(rank int) (Entry[K, V], bool)

	// --- Sorted Clone ---
	// CloneSorted returns a shallow copy as SortedMap.
	CloneSorted() SortedMap[K, V]
}

// ==========================
// Concurrent-safe Interfaces
// ==========================
//
// Atomicity Guide:
//
// Operations are classified into three categories:
//
// 1. ATOMIC (✓): Guaranteed atomic in all implementations.
//    - Single-key reads: Get, Contains, ContainsKey
//    - Single-key writes: Add, Remove
//    - Atomic compound operations: AddIfAbsent, RemoveAndGet, PutIfAbsent,
//      GetOrCompute (its store is atomic everywhere; see GetOrCompute for
//      where the compute callback runs and the reentrancy rule)
//
// 2. BEST-EFFORT (~): Atomic in the mutex- and hash-based implementations;
//    may race in the lock-free skip-list implementations.
//    - Put: the write itself is always atomic, but the skip-list variants
//      load the previous value and then store, so the returned old value can
//      be stale when two writers hit the same key.
//    - CompareAndSwap, CompareAndDelete, RemoveIf, ReplaceIf
//    - These use load-then-modify patterns that can race under high contention.
//
// 3. NON-ATOMIC (✗): Never atomic as a whole; provide snapshot semantics.
//    - Bulk operations: AddAll, RemoveAll, PutAll, RemoveFunc
//    - Iteration: Seq, ForEach, Range, Keys, Values, Entries
//    - Set algebra: Union, Intersection, Difference
//    - Aggregations: Size (approximate), IsEmpty, Equals

// ConcurrentSet extends Set with atomic operations for concurrent use.
// All methods are safe for concurrent calls from multiple goroutines.
//
// Atomicity guarantees:
//   - ATOMIC: Add, Remove, Contains, AddIfAbsent, RemoveAndGet, Pop
//   - BEST-EFFORT: RemoveFunc, RetainFunc (snapshot + individual removes)
//   - NON-ATOMIC: AddAll, RemoveAll, AddSeq, RemoveSeq, Union, Intersection, etc.
type ConcurrentSet[T any] interface {
	Set[T]

	// AddIfAbsent atomically inserts element only if currently absent.
	// ATOMIC: This is a single atomic operation.
	// Returns true if the element was added.
	AddIfAbsent(element T) bool

	// RemoveAndGet atomically removes and returns the element if present.
	// ATOMIC: This is a single atomic operation.
	RemoveAndGet(element T) (T, bool)
}

// ConcurrentSortedSet extends SortedSet with atomic operations.
// It embeds ConcurrentSet to inherit atomic methods.
//
// Additional atomicity notes for sorted operations:
//   - First, Last, Floor, Ceiling, Lower, Higher: atomic in the mutex-guarded
//     tree implementation; the lock-free skip-list implementation answers them
//     by scanning, so the result is a best-effort snapshot.
//   - BEST-EFFORT: PopFirst, PopLast (may retry under contention)
//   - NON-ATOMIC: Range, Ascend, Descend, SubSet, HeadSet, TailSet
type ConcurrentSortedSet[T any] interface {
	SortedSet[T]
	ConcurrentSet[T]
}

// ConcurrentMap extends Map with atomic operations for concurrent use.
// All methods are safe for concurrent calls from multiple goroutines.
//
// Atomicity guarantees:
//   - ATOMIC: Get, Remove, ContainsKey, PutIfAbsent, RemoveAndGet, GetOrCompute
//   - Put: the write is atomic everywhere, but in ConcurrentSkipMap the
//     returned old value comes from a separate load and can be stale when
//     two writers hit the same key.
//   - BEST-EFFORT: CompareAndSwap, CompareAndDelete, RemoveIf, ReplaceIf
//     (atomic in ConcurrentHashMap and ConcurrentTreeMap; may race in
//     ConcurrentSkipMap)
//   - NON-ATOMIC: PutAll, RemoveAll, RemoveFunc, ReplaceAll, Keys, Values, Entries, Seq
//
// Implementation notes:
//   - ConcurrentHashMap (xsync.MapOf): All operations are strictly atomic.
//   - ConcurrentSkipMap (lock-free skip list): Single-key ops are atomic;
//     compound ops — Put's old-value report, CompareAndSwap and the like —
//     use load-then-modify and may race.
type ConcurrentMap[K, V any] interface {
	Map[K, V]

	// GetOrCompute returns the existing value or computes and stores a new one.
	// Returns (value, true) if computed (key was absent).
	// The store is atomic in every implementation; where compute runs differs:
	//   - ConcurrentHashMap and ConcurrentTreeMap run it under an internal
	//     lock — it must not call back into the same map. A panic in it is
	//     propagated with the map left unchanged and usable.
	//   - ConcurrentSkipMap runs it without locks — it may call back into
	//     the map, and when two writers race on the same absent key both
	//     may compute, with the loser's result discarded.
	GetOrCompute(key K, compute func() V) (V, bool)

	// RemoveAndGet atomically removes and returns the value for key.
	// ATOMIC: This is a single atomic operation.
	RemoveAndGet(key K) (V, bool)

	// CompareAndSwap replaces value if current equals old.
	// BEST-EFFORT: Atomic in ConcurrentHashMap; may race in ConcurrentSkipMap.
	CompareAndSwap(key K, oldValue, newValue V, eq Equaler[V]) bool

	// CompareAndDelete deletes entry if current value equals provided.
	// BEST-EFFORT: Atomic in ConcurrentHashMap; may race in ConcurrentSkipMap.
	CompareAndDelete(key K, value V, eq Equaler[V]) bool
}

// ConcurrentSortedMap extends SortedMap with atomic operations.
// It embeds ConcurrentMap to inherit atomic methods.
//
// Additional atomicity notes for sorted operations:
//   - FirstKey, LastKey, FirstEntry, LastEntry, FloorKey, CeilingKey: atomic
//     in the mutex-guarded tree implementation; the lock-free skip-list
//     implementation answers them by scanning, so the result is a
//     best-effort snapshot.
//   - BEST-EFFORT: PopFirst, PopLast (may retry under contention)
//   - NON-ATOMIC: Range, Ascend, Descend, SubMap, HeadMap, TailMap
type ConcurrentSortedMap[K, V any] interface {
	SortedMap[K, V]
	ConcurrentMap[K, V]
}

// ==========================
// Common String() formatters
// ==========================

// formatCollection renders a collection in the form: name{a, b, c}
// The provided seq controls iteration ordering.
func formatCollection[T any](name string, seq iter.Seq[T]) string {
	var b strings.Builder
	b.WriteString(cmp.Or(name, "collection"))
	b.WriteString("{")
	first := true
	for v := range seq {
		if !first {
			b.WriteString(", ")
		}
		first = false
		_, _ = fmt.Fprintf(&b, "%v", v)
	}
	b.WriteString("}")
	return b.String()
}

// formatMap renders a map in the form: name{k:v, ...}
// The provided seq controls iteration ordering.
func formatMap[K, V any](name string, seq iter.Seq2[K, V]) string {
	var b strings.Builder
	b.WriteString(cmp.Or(name, "map"))
	b.WriteString("{")
	first := true
	seq(func(k K, v V) bool {
		if !first {
			b.WriteString(", ")
		}
		first = false
		_, _ = fmt.Fprintf(&b, "%v:%v", k, v)
		return true
	})
	b.WriteString("}")
	return b.String()
}
