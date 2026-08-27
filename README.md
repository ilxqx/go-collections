# go-collections

[![Go Reference](https://pkg.go.dev/badge/github.com/coldsmirk/go-collections.svg)](https://pkg.go.dev/github.com/coldsmirk/go-collections)
[![Go Report Card](https://goreportcard.com/badge/github.com/coldsmirk/go-collections)](https://goreportcard.com/report/github.com/coldsmirk/go-collections)
[![Build Status](https://github.com/coldsmirk/go-collections/actions/workflows/test.yml/badge.svg)](https://github.com/coldsmirk/go-collections/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/coldsmirk/go-collections/branch/main/graph/badge.svg)](https://codecov.io/gh/coldsmirk/go-collections)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Generic, fast, and ergonomic collections for Go 1.25+. This library provides a comprehensive set of type-safe, generic data structures with clean APIs, consistent naming, and support for Go's `iter.Seq`/`iter.Seq2` for seamless for-range integration.

## Table of Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Constraints and Helpers](#constraints-and-helpers)
- [Sets](#sets)
  - [Set Interface](#set-interface)
  - [SortedSet Interface](#sortedset-interface)
  - [HashSet](#hashset)
  - [TreeSet](#treeset)
  - [ConcurrentHashSet](#concurrenthashset)
  - [ConcurrentTreeSet](#concurrenttreeset)
  - [ConcurrentSkipSet](#concurrentskipset)
- [Maps](#maps)
  - [Map Interface](#map-interface)
  - [SortedMap Interface](#sortedmap-interface)
  - [HashMap](#hashmap)
  - [TreeMap](#treemap)
  - [ConcurrentHashMap](#concurrenthashmap)
  - [ConcurrentTreeMap](#concurrenttreemap)
  - [ConcurrentSkipMap](#concurrentskipmap)
- [Lists](#lists)
  - [List Interface](#list-interface)
  - [ArrayList](#arraylist)
  - [LinkedList](#linkedlist)
  - [Concurrent List Implementations](#concurrent-list-implementations)
  - [Concurrent List Comparison](#concurrent-list-comparison)
- [Stacks](#stacks)
  - [Stack Interface](#stack-interface)
  - [ArrayStack](#arraystack)
- [Queues](#queues)
  - [Queue Interface](#queue-interface)
  - [ArrayQueue](#arrayqueue)
  - [PriorityQueue](#priorityqueue)
- [Deques](#deques)
  - [Deque Interface](#deque-interface)
  - [ArrayDeque](#arraydeque)
- [License](#license)

## Features

- **Type-Safe Generics**: All collections are fully generic with compile-time type safety
- **Interface Segregation**: Small, focused interfaces composed into higher-level ones
- **Go 1.25+ Iteration**: Native support for `iter.Seq` and `iter.Seq2` for seamless for-range loops
- **Consistent API**: Uniform naming across Set/Map/List; `(value, ok)` return patterns
- **Concurrent Variants**: Thread-safe collections with atomic operations
- **No Reflection**: Zero runtime reflection for maximum performance
- **Sorted Collections**: B-Tree and Skip List backed sorted sets and maps
- **Memory Efficient**: Optimized to prevent memory leaks with `slices.Delete` and proper capacity management

## Requirements

- Go 1.25 or later (uses `iter.Seq`/`iter.Seq2`, `testing.B.Loop`, built-in `min`/`max`)

## Installation

```bash
go get github.com/coldsmirk/go-collections
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/coldsmirk/go-collections"
)

func main() {
    // Create a HashSet
    set := collections.NewHashSetFrom(1, 2, 3, 4, 5)

    // Iterate using for-range with iter.Seq
    for v := range set.Seq() {
        fmt.Println(v)
    }

    // Set operations
    other := collections.NewHashSetFrom(4, 5, 6, 7)
    union := set.Union(other)
    intersection := set.Intersection(other)

    fmt.Printf("Union size: %d\n", union.Size())
    fmt.Printf("Intersection size: %d\n", intersection.Size())

    // Create a HashMap
    m := collections.NewHashMap[string, int]()
    m.Put("one", 1)
    m.Put("two", 2)

    if v, ok := m.Get("one"); ok {
        fmt.Printf("one = %d\n", v)
    }

    // Create a sorted TreeSet
    sorted := collections.NewTreeSetOrdered[int]()
    sorted.AddAll(5, 3, 1, 4, 2)

    // Iterate in sorted order
    for v := range sorted.Seq() {
        fmt.Println(v) // 1, 2, 3, 4, 5
    }
}
```

## Constraints and Helpers

### Type Constraints

```go
// Ordered is a constraint for types that support <, ==, > operators.
// Aliases cmp.Ordered from the standard library.
type Ordered = cmp.Ordered

// Comparator compares two values: negative if a < b, zero if a == b, positive if a > b.
type Comparator[T any] func(a, b T) int

// Equaler reports whether two values are equal.
type Equaler[T any] func(a, b T) bool
```

### Helper Functions

```go
// EqualFunc returns a default Equaler for comparable types using ==.
func EqualFunc[T comparable]() Equaler[T]

// CompareFunc returns a Comparator for Ordered types using cmp.Compare.
func CompareFunc[T Ordered]() Comparator[T]
```

**Usage:**

```go
// Using EqualFunc for list operations
list := collections.NewArrayListFrom(1, 2, 3, 2, 1)
list.Remove(2, collections.EqualFunc[int]())

// Using CompareFunc for sorting
list.Sort(collections.CompareFunc[int]())
```

---

## Sets

### Set Interface

`Set[T]` is an unordered collection of unique elements.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of elements |
| `IsEmpty() bool` | Reports whether the set is empty |
| `IsNotEmpty() bool` | Reports whether the set contains at least one element |
| `Clear()` | Removes all elements |
| `ToSlice() []T` | Returns a snapshot slice of all elements |
| `String() string` | Returns a string representation |
| `Seq() iter.Seq[T]` | Returns a sequence for for-range |
| `ForEach(action func(T) bool)` | Iterates elements; stops if action returns false |
| `Add(element T) bool` | Inserts element if absent; returns true if set changed |
| `AddAll(elements ...T) int` | Inserts all elements; returns count added |
| `AddSeq(seq iter.Seq[T]) int` | Inserts from sequence; returns count added |
| `Remove(element T) bool` | Removes element; returns true if removed |
| `RemoveAll(elements ...T) int` | Removes all elements; returns count removed |
| `RemoveSeq(seq iter.Seq[T]) int` | Removes from sequence; returns count removed |
| `RemoveFunc(predicate func(T) bool) int` | Removes elements satisfying predicate |
| `RetainFunc(predicate func(T) bool) int` | Keeps only elements satisfying predicate |
| `Pop() (T, bool)` | Removes and returns an arbitrary element |
| `Contains(element T) bool` | Reports whether element exists |
| `ContainsAll(elements ...T) bool` | Reports whether all elements exist |
| `ContainsAny(elements ...T) bool` | Reports whether any element exists |
| `Union(other Set[T]) Set[T]` | Returns s ∪ other |
| `Intersection(other Set[T]) Set[T]` | Returns s ∩ other |
| `Difference(other Set[T]) Set[T]` | Returns s - other |
| `SymmetricDifference(other Set[T]) Set[T]` | Returns (s - other) ∪ (other - s) |
| `IsSubsetOf(other Set[T]) bool` | Reports whether s ⊆ other |
| `IsSupersetOf(other Set[T]) bool` | Reports whether s ⊇ other |
| `IsProperSubsetOf(other Set[T]) bool` | Reports whether s ⊂ other |
| `IsProperSupersetOf(other Set[T]) bool` | Reports whether s ⊃ other |
| `IsDisjoint(other Set[T]) bool` | Reports whether s and other have no common elements |
| `Equals(other Set[T]) bool` | Reports whether sets are equal |
| `Clone() Set[T]` | Returns a shallow copy |
| `Filter(predicate func(T) bool) Set[T]` | Returns new set with matching elements |
| `Find(predicate func(T) bool) (T, bool)` | Returns first matching element |
| `Any(predicate func(T) bool) bool` | Returns true if any element matches |
| `Every(predicate func(T) bool) bool` | Returns true if all elements match |

### SortedSet Interface

`SortedSet[T]` extends `Set[T]` with sorted order operations.

| Method | Description |
|--------|-------------|
| `First() (T, bool)` | Returns the smallest element |
| `Last() (T, bool)` | Returns the largest element |
| `Min() (T, bool)` | Alias for First |
| `Max() (T, bool)` | Alias for Last |
| `PopFirst() (T, bool)` | Removes and returns the smallest element |
| `PopLast() (T, bool)` | Removes and returns the largest element |
| `Floor(x T) (T, bool)` | Returns greatest element ≤ x |
| `Ceiling(x T) (T, bool)` | Returns smallest element ≥ x |
| `Lower(x T) (T, bool)` | Returns greatest element < x |
| `Higher(x T) (T, bool)` | Returns smallest element > x |
| `Range(from, to T, action func(T) bool)` | Iterates elements in [from, to] |
| `RangeSeq(from, to T) iter.Seq[T]` | Returns sequence for [from, to] |
| `Ascend(action func(T) bool)` | Iterates in ascending order |
| `Descend(action func(T) bool)` | Iterates in descending order |
| `AscendFrom(pivot T, action func(T) bool)` | Iterates elements ≥ pivot ascending |
| `DescendFrom(pivot T, action func(T) bool)` | Iterates elements ≤ pivot descending |
| `Reversed() iter.Seq[T]` | Returns descending sequence |
| `SubSet(from, to T) SortedSet[T]` | Returns elements in [from, to] |
| `HeadSet(to T, inclusive bool) SortedSet[T]` | Returns elements < to (or ≤) |
| `TailSet(from T, inclusive bool) SortedSet[T]` | Returns elements > from (or ≥) |
| `Rank(x T) int` | Returns 0-based rank; -1 if absent |
| `GetByRank(rank int) (T, bool)` | Returns element at rank |
| `CloneSorted() SortedSet[T]` | Returns shallow copy as SortedSet |

### HashSet

Hash-based unordered set backed by `map[T]struct{}`.

**Constructors:**

```go
func NewHashSet[T comparable]() Set[T]
func NewHashSetWithCapacity[T comparable](capacity int) Set[T]
func NewHashSetFrom[T comparable](elements ...T) Set[T]
```

**Example:**

```go
set := collections.NewHashSetFrom("apple", "banana", "cherry")

set.Add("date")
set.Remove("banana")

for fruit := range set.Seq() {
    fmt.Println(fruit)
}

// Set algebra
other := collections.NewHashSetFrom("cherry", "date", "elderberry")
union := set.Union(other)
intersection := set.Intersection(other)
```

### TreeSet

Sorted set backed by B-Tree. Maintains elements in comparator order.

**Constructors:**

```go
func NewTreeSet[T any](c Comparator[T]) SortedSet[T]
func NewTreeSetOrdered[T Ordered]() SortedSet[T]
func NewTreeSetFrom[T any](c Comparator[T], elements ...T) SortedSet[T]
```

**Example:**

```go
// For Ordered types (int, string, etc.)
set := collections.NewTreeSetOrdered[int]()
set.AddAll(5, 3, 1, 4, 2)

first, _ := set.First()  // 1
last, _ := set.Last()    // 5

// Range query
set.Range(2, 4, func(v int) bool {
    fmt.Println(v)  // 2, 3, 4
    return true
})

// Custom comparator (reverse order)
reverseSet := collections.NewTreeSet(func(a, b int) int {
    return b - a  // Descending
})
```

### ConcurrentHashSet

Thread-safe hash set backed by `xsync.MapOf`.

**Constructors:**

```go
func NewConcurrentHashSet[T comparable]() ConcurrentSet[T]
func NewConcurrentHashSetFrom[T comparable](elements ...T) ConcurrentSet[T]
```

**Additional Methods (ConcurrentSet):**

| Method | Description |
|--------|-------------|
| `AddIfAbsent(element T) bool` | Atomically adds if absent |
| `RemoveAndGet(element T) (T, bool)` | Atomically removes and returns |

**Example:**

```go
set := collections.NewConcurrentHashSet[int]()

// Safe for concurrent use
go func() { set.Add(1) }()
go func() { set.Add(2) }()

// Atomic operations
if set.AddIfAbsent(3) {
    fmt.Println("3 was added")
}
```

### ConcurrentTreeSet

Thread-safe sorted set using RWMutex-protected TreeSet.

**Constructors:**

```go
func NewConcurrentTreeSet[T any](cmpT Comparator[T]) ConcurrentSortedSet[T]
func NewConcurrentTreeSetOrdered[T Ordered]() ConcurrentSortedSet[T]
func NewConcurrentTreeSetFrom[T any](cmpT Comparator[T], elements ...T) ConcurrentSortedSet[T]
```

### ConcurrentSkipSet

Lock-free concurrent sorted set backed by skip list. For `Ordered` types only.

**Atomicity Note:** Single-element operations (`Add`/`Contains`/`Remove`) are atomic. Navigation queries (`First`/`Last`/`Floor`/`Ceiling`/…) answer by scanning, so they return best-effort snapshots at O(n) cost. For strict semantics, use `ConcurrentTreeSet` instead.

**Constructors:**

```go
func NewConcurrentSkipSet[T Ordered]() ConcurrentSortedSet[T]
func NewConcurrentSkipSetFrom[T Ordered](elements ...T) ConcurrentSortedSet[T]
```

---

## Maps

### Map Interface

`Map[K, V]` is a collection that maps keys to values.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of entries |
| `IsEmpty() bool` | Reports whether the map is empty |
| `IsNotEmpty() bool` | Reports whether the map contains at least one entry |
| `Clear()` | Removes all entries |
| `String() string` | Returns a string representation |
| `Put(key K, value V) (V, bool)` | Associates value with key; returns (old, existed) |
| `PutIfAbsent(key K, value V) (V, bool)` | Stores if absent; returns (value, inserted) |
| `PutAll(other Map[K, V])` | Copies all entries from other |
| `PutSeq(seq iter.Seq2[K, V]) int` | Copies from sequence; returns keys changed |
| `Get(key K) (V, bool)` | Returns (value, exists) |
| `GetOrDefault(key K, defaultValue V) V` | Returns value or default |
| `Remove(key K) (V, bool)` | Removes key; returns (old, existed) |
| `RemoveIf(key K, value V, eq Equaler[V]) bool` | Removes if value matches |
| `ContainsKey(key K) bool` | Reports whether key exists |
| `ContainsValue(value V, eq Equaler[V]) bool` | Reports whether value exists (O(n)) |
| `RemoveKeys(keys ...K) int` | Removes keys; returns count |
| `RemoveKeysSeq(seq iter.Seq[K]) int` | Removes from sequence |
| `RemoveFunc(predicate func(K, V) bool) int` | Removes matching entries |
| `Compute(key K, remapping func(K, V, bool) (V, bool)) (V, bool)` | Recomputes mapping |
| `ComputeIfAbsent(key K, mapping func(K) V) V` | Computes if absent |
| `ComputeIfPresent(key K, remapping func(K, V) (V, bool)) (V, bool)` | Recomputes if present |
| `Merge(key K, value V, remapping func(V, V) (V, bool)) (V, bool)` | Merges with existing |
| `Replace(key K, value V) (V, bool)` | Replaces if present |
| `ReplaceIf(key K, oldValue, newValue V, eq Equaler[V]) bool` | Replaces if value matches |
| `ReplaceAll(function func(K, V) V)` | Replaces all values |
| `Keys() []K` | Returns all keys |
| `Values() []V` | Returns all values |
| `Entries() []Entry[K, V]` | Returns all entries |
| `ForEach(action func(K, V) bool)` | Iterates entries |
| `Seq() iter.Seq2[K, V]` | Returns (key, value) sequence |
| `SeqKeys() iter.Seq[K]` | Returns keys sequence |
| `SeqValues() iter.Seq[V]` | Returns values sequence |
| `Clone() Map[K, V]` | Returns shallow copy |
| `Filter(predicate func(K, V) bool) Map[K, V]` | Returns filtered map |
| `Equals(other Map[K, V], valueEq Equaler[V]) bool` | Reports equality |

### SortedMap Interface

`SortedMap[K, V]` extends `Map[K, V]` with sorted key operations.

| Method | Description |
|--------|-------------|
| `FirstKey() (K, bool)` | Returns smallest key |
| `LastKey() (K, bool)` | Returns largest key |
| `FirstEntry() (Entry[K, V], bool)` | Returns entry with smallest key |
| `LastEntry() (Entry[K, V], bool)` | Returns entry with largest key |
| `PopFirst() (Entry[K, V], bool)` | Removes smallest-key entry |
| `PopLast() (Entry[K, V], bool)` | Removes largest-key entry |
| `FloorKey(k K) (K, bool)` | Returns greatest key ≤ k |
| `CeilingKey(k K) (K, bool)` | Returns smallest key ≥ k |
| `LowerKey(k K) (K, bool)` | Returns greatest key < k |
| `HigherKey(k K) (K, bool)` | Returns smallest key > k |
| `FloorEntry(k K) (Entry[K, V], bool)` | Returns entry with greatest key ≤ k |
| `CeilingEntry(k K) (Entry[K, V], bool)` | Returns entry with smallest key ≥ k |
| `LowerEntry(k K) (Entry[K, V], bool)` | Returns entry with greatest key < k |
| `HigherEntry(k K) (Entry[K, V], bool)` | Returns entry with smallest key > k |
| `Range(from, to K, action func(K, V) bool)` | Iterates [from, to] |
| `RangeSeq(from, to K) iter.Seq2[K, V]` | Returns sequence for [from, to] |
| `RangeFrom(from K, action func(K, V) bool)` | Iterates keys ≥ from |
| `RangeTo(to K, action func(K, V) bool)` | Iterates keys ≤ to |
| `Ascend(action func(K, V) bool)` | Iterates ascending |
| `Descend(action func(K, V) bool)` | Iterates descending |
| `AscendFrom(pivot K, action func(K, V) bool)` | Iterates ≥ pivot ascending |
| `DescendFrom(pivot K, action func(K, V) bool)` | Iterates ≤ pivot descending |
| `Reversed() iter.Seq2[K, V]` | Returns descending sequence |
| `SubMap(from, to K) SortedMap[K, V]` | Returns entries in [from, to] |
| `HeadMap(to K, inclusive bool) SortedMap[K, V]` | Returns entries < to (or ≤) |
| `TailMap(from K, inclusive bool) SortedMap[K, V]` | Returns entries > from (or ≥) |
| `RankOfKey(key K) int` | Returns 0-based rank; -1 if absent |
| `GetByRank(rank int) (Entry[K, V], bool)` | Returns entry at rank |
| `CloneSorted() SortedMap[K, V]` | Returns shallow copy as SortedMap |

### Entry Type

```go
type Entry[K, V any] struct {
    Key   K
    Value V
}

// Unpack returns the key and value for convenient destructuring.
func (e Entry[K, V]) Unpack() (K, V)
```

### HashMap

Hash-based unordered map backed by `map[K]V`.

**Constructors:**

```go
func NewHashMap[K comparable, V any]() Map[K, V]
func NewHashMapWithCapacity[K comparable, V any](capacity int) Map[K, V]
func NewHashMapFrom[K comparable, V any](src map[K]V) Map[K, V]
```

**Additional Interface:**

```go
**Example:**

```go
m := collections.NewHashMap[string, int]()
m.Put("one", 1)
m.Put("two", 2)
m.Put("three", 3)

// Get with existence check
if v, ok := m.Get("two"); ok {
    fmt.Printf("two = %d\n", v)
}

// Iterate
for k, v := range m.Seq() {
    fmt.Printf("%s: %d\n", k, v)
}

// Compute operations
m.ComputeIfAbsent("four", func(k string) int {
    return len(k)  // 4
})
```

### TreeMap

Sorted map backed by B-Tree.

**Constructors:**

```go
func NewTreeMap[K any, V any](cmpK Comparator[K]) SortedMap[K, V]
func NewTreeMapOrdered[K Ordered, V any]() SortedMap[K, V]
func NewTreeMapFrom[K comparable, V any](cmpK Comparator[K], m map[K]V) SortedMap[K, V]
```

**Example:**

```go
m := collections.NewTreeMapOrdered[int, string]()
m.Put(3, "three")
m.Put(1, "one")
m.Put(2, "two")

// Iterate in sorted key order
for k, v := range m.Seq() {
    fmt.Printf("%d: %s\n", k, v)  // 1, 2, 3
}

// Navigation
floor, _ := m.FloorKey(2)      // 2
ceiling, _ := m.CeilingKey(2)  // 2
lower, _ := m.LowerKey(2)      // 1
higher, _ := m.HigherKey(2)    // 3
```

### ConcurrentHashMap

Thread-safe hash map backed by `xsync.MapOf`.

**Callback Note:** The compute-family callbacks (`GetOrCompute`, `Compute`, `ComputeIfAbsent`, `ComputeIfPresent`, `Merge`) run while an internal bucket lock is held: they must not call back into the same map, and should be short. A panic inside a callback is propagated after the lock is released and leaves the map unchanged and usable.

**Constructors:**

```go
func NewConcurrentHashMap[K comparable, V any]() ConcurrentMap[K, V]
func NewConcurrentHashMapFrom[K comparable, V any](src map[K]V) ConcurrentMap[K, V]
```

**Additional Methods (ConcurrentMap):**

| Method | Description |
|--------|-------------|
| `GetOrCompute(key K, compute func() V) (V, bool)` | Atomically get or compute |
| `RemoveAndGet(key K) (V, bool)` | Atomically removes and returns |
| `PutIfAbsent(key K, value V) (V, bool)` | Atomically stores if absent |
| `CompareAndSwap(key K, oldValue, newValue V, eq Equaler[V]) bool` | Atomic CAS |
| `CompareAndDelete(key K, value V, eq Equaler[V]) bool` | Atomic compare-and-delete |

**Example:**

```go
m := collections.NewConcurrentHashMap[string, int]()

// Safe for concurrent use
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(n int) {
        defer wg.Done()
        m.Put(fmt.Sprintf("key%d", n), n)
    }(i)
}
wg.Wait()

// Atomic operations
v, computed := m.GetOrCompute("expensive", func() int {
    return expensiveCalculation()
})
```

### ConcurrentTreeMap

Thread-safe sorted map using RWMutex-protected TreeMap.

**Constructors:**

```go
func NewConcurrentTreeMap[K any, V any](cmpK Comparator[K]) ConcurrentSortedMap[K, V]
func NewConcurrentTreeMapOrdered[K Ordered, V any]() ConcurrentSortedMap[K, V]
func NewConcurrentTreeMapFrom[K comparable, V any](cmpK Comparator[K], m map[K]V) ConcurrentSortedMap[K, V]
```

### ConcurrentSkipMap

Lock-free concurrent sorted map backed by skip list. For `Ordered` keys only.

**Atomicity Note:** Single-key operations (`Get`/`PutIfAbsent`/`RemoveAndGet`) are atomic. `Put` stores atomically, but reports the old value from a separate load, so the returned old value is **best-effort** when writers race on one key. `CompareAndSwap` and `CompareAndDelete` are likewise **best-effort** due to the lock-free nature of skip lists, and navigation queries (`FirstKey`/`LastKey`/`Floor`/`Ceiling`/…) as well as from-pivot range queries (`RangeFrom`/`AscendFrom`/`TailMap`/…) answer by scanning from the smallest key, so they cost O(n) regardless of the result size. `GetOrCompute`/`ComputeIfAbsent` run their compute callback without any lock — it may call back into the map and a panic in it stores nothing, but when two writers race on the same absent key both may compute and the loser's result is discarded. For strict semantics, use `ConcurrentTreeMap` instead.

**Constructors:**

```go
func NewConcurrentSkipMap[K Ordered, V any]() ConcurrentSortedMap[K, V]
func NewConcurrentSkipMapFrom[K Ordered, V any](src map[K]V) ConcurrentSortedMap[K, V]
```

---

## Lists

### List Interface

`List[T]` is an ordered collection with index-based access.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of elements |
| `IsEmpty() bool` | Reports whether the list is empty |
| `IsNotEmpty() bool` | Reports whether the list contains at least one element |
| `Clear()` | Removes all elements |
| `ToSlice() []T` | Returns a snapshot slice |
| `String() string` | Returns a string representation |
| `Seq() iter.Seq[T]` | Returns a sequence for for-range |
| `ForEach(action func(T) bool)` | Iterates elements |
| `Get(index int) (T, bool)` | Returns element at index |
| `Set(index int, element T) (T, bool)` | Replaces element; returns old |
| `First() (T, bool)` | Returns first element |
| `Last() (T, bool)` | Returns last element |
| `Add(element T)` | Appends element |
| `AddAll(elements ...T)` | Appends all elements |
| `AddSeq(seq iter.Seq[T])` | Appends from sequence |
| `Insert(index int, element T) bool` | Inserts at index |
| `InsertAll(index int, elements ...T) bool` | Inserts all at index |
| `RemoveAt(index int) (T, bool)` | Removes at index |
| `Remove(element T, eq Equaler[T]) bool` | Removes first occurrence |
| `RemoveFirst() (T, bool)` | Removes first element |
| `RemoveLast() (T, bool)` | Removes last element |
| `RemoveFunc(predicate func(T) bool) int` | Removes matching elements |
| `RetainFunc(predicate func(T) bool) int` | Keeps matching elements |
| `IndexOf(element T, eq Equaler[T]) int` | Returns first index; -1 if absent |
| `LastIndexOf(element T, eq Equaler[T]) int` | Returns last index; -1 if absent |
| `Contains(element T, eq Equaler[T]) bool` | Reports whether element exists |
| `Find(predicate func(T) bool) (T, bool)` | Returns first matching element |
| `FindIndex(predicate func(T) bool) int` | Returns first matching index |
| `SubList(from, to int) List[T]` | Returns elements in [from, to) |
| `Reversed() iter.Seq[T]` | Returns reverse sequence |
| `Clone() List[T]` | Returns shallow copy |
| `Filter(predicate func(T) bool) List[T]` | Returns filtered list |
| `Sort(comparator Comparator[T])` | Sorts in place |
| `Any(predicate func(T) bool) bool` | Returns true if any match |
| `Every(predicate func(T) bool) bool` | Returns true if all match |

### ArrayList

Dynamic array-backed list with O(1) append and O(n) insert/remove.

**Constructors:**

```go
func NewArrayList[T any]() List[T]
func NewArrayListWithCapacity[T any](capacity int) List[T]
func NewArrayListFrom[T any](elements ...T) List[T]
```

**Example:**

```go
list := collections.NewArrayListFrom(1, 2, 3)

list.Add(4)
list.Insert(0, 0)  // [0, 1, 2, 3, 4]

v, _ := list.Get(2)  // 2
list.Set(2, 20)      // [0, 1, 20, 3, 4]

// Sort
list.Sort(collections.CompareFunc[int]())

// Filter
evens := list.Filter(func(n int) bool { return n%2 == 0 })

// Iterate in reverse
for v := range list.Reversed() {
    fmt.Println(v)
}
```

### LinkedList

Doubly-linked list with O(1) head/tail operations and O(n) random access.

**When to use LinkedList vs ArrayList:**
- Use `LinkedList` when you frequently insert/remove at the beginning or need stable iterators
- Use `ArrayList` when you need fast random access by index or iterate sequentially

**Complexity:**
| Operation | LinkedList | ArrayList |
|-----------|------------|-----------|
| Add (append) | O(1) | O(1) amortized |
| AddFirst/RemoveFirst | O(1) | O(n) |
| Get/Set by index | O(n) | O(1) |
| Insert at index | O(n) | O(n) |

**Constructors:**

```go
func NewLinkedList[T any]() List[T]
func NewLinkedListFrom[T any](elements ...T) List[T]
```

**Example:**

```go
list := collections.NewLinkedListFrom("a", "b", "c")

// O(1) operations at ends
first, _ := list.RemoveFirst()  // "a"
list.Add("d")                   // append: O(1)

// O(n) random access
v, _ := list.Get(1)  // "c" (need to traverse)

// Iterate
for v := range list.Seq() {
    fmt.Println(v)  // "b", "c", "d"
}
```

### Concurrent List Implementations

For concurrent scenarios, three specialized list implementations are available:

#### COWList (Copy-on-Write List)

A copy-on-write list that provides lock-free reads and atomic writes by copying the entire underlying slice on modifications.

**Constructors:**

```go
func NewCOWList[T any]() List[T]
func NewCOWListFrom[T any](elements ...T) List[T]
```

#### SyncList

A concurrent list backed by a slice guarded by a single RWMutex. Every method call is atomic; iteration methods copy a snapshot first, so callbacks never run under the lock. (Replaces the former SegmentedList, whose segmented locks delivered no real write concurrency.)

**Constructors:**

```go
func NewSyncList[T any]() List[T]
func NewSyncListFrom[T any](elements ...T) List[T]
```

#### LockFreeList

A lock-free concurrent linked list using CAS operations with logical deletion (deleted nodes are marked and skipped, and reclaimed when `Clear` detaches the chain).

**Constructors:**

```go
func NewLockFreeList[T any](eq Equaler[T]) List[T]
func NewLockFreeListOrdered[T comparable]() List[T]
func NewLockFreeListFrom[T any](eq Equaler[T], elements ...T) List[T]
```

**Note:** `Add` walks the chain to append, so it is O(n) and bulk building is O(n²); removals are logical, so traversal cost also grows with removals until `Clear`. `Size` and `IsEmpty` traverse the chain (O(n)) — there is no element counter, because a counter maintained beside the chain cannot stay consistent with a concurrent `Clear`. Each node holds its value and deletion state in one atomic pointer, so `Set` and a removal of the same element linearize against each other. Best suited to small, hot lists where lock-free progress matters more than throughput at scale.

### Concurrent List Comparison

| Feature | COWList | SyncList | LockFreeList |
|---------|---------|----------|--------------|
| **Read Performance** | O(1) lock-free | O(1) under read lock | O(n) traversal |
| **Write Performance** | O(n) copy | O(1) amortized under write lock | O(n) walk + CAS |
| **Memory Overhead** | High (full copy on write) | Low | Medium (one node per element) |
| **Consistency** | Snapshot reads | Per-call atomic | Eventual (logical deletion) |
| **Best For** | Read-heavy, rare writes | Balanced read/write | Small hot lists under high contention |
| **Size Accuracy** | Exact | Exact | O(n) traversal; approximate under races |
| **Random Access** | O(1) | O(1) | O(n) |
| **Iteration** | Snapshot | Snapshot | Snapshot |

**When to use which:**
- **COWList**: Ideal for read-heavy workloads where writes are infrequent (e.g., configuration lists, caches)
- **SyncList**: Balanced read/write workloads; the simple, predictable default
- **LockFreeList**: Small, high-contention lists where progress guarantees matter more than exact counts

**Atomicity:**

| Operation | COWList | SyncList | LockFreeList |
|-----------|---------|----------|--------------|
| Get/Contains | Atomic (snapshot) | Atomic (lock) | Atomic |
| Set/Remove | Atomic (full copy) | Atomic (lock) | Atomic per node (single CAS) |
| Add/Insert | Atomic (full copy) | Atomic (lock) | Best-effort CAS |
| Size | Exact | Exact | O(n) traversal; approximate under races |
| Iteration | Snapshot | Snapshot | Snapshot |

---

## Stacks

### Stack Interface

`Stack[T]` is a LIFO (last-in-first-out) collection.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of elements |
| `IsEmpty() bool` | Reports whether the stack is empty |
| `IsNotEmpty() bool` | Reports whether the stack contains at least one element |
| `Clear()` | Removes all elements |
| `String() string` | Returns a string representation |
| `Push(element T)` | Adds element to top |
| `PushAll(elements ...T)` | Adds all to top (last becomes top) |
| `Pop() (T, bool)` | Removes and returns top |
| `Peek() (T, bool)` | Returns top without removing |
| `ToSlice() []T` | Returns elements bottom to top |
| `Seq() iter.Seq[T]` | Returns sequence top to bottom (LIFO) |

### ArrayStack

Slice-backed stack with O(1) push/pop.

**Constructors:**

```go
func NewArrayStack[T any]() Stack[T]
func NewArrayStackWithCapacity[T any](capacity int) Stack[T]
func NewArrayStackFrom[T any](elements ...T) Stack[T]
```

**Example:**

```go
stack := collections.NewArrayStack[int]()

stack.Push(1)
stack.Push(2)
stack.Push(3)

top, _ := stack.Peek()  // 3
v, _ := stack.Pop()     // 3
v, _ = stack.Pop()      // 2

// Iterate in LIFO order
for v := range stack.Seq() {
    fmt.Println(v)  // 1
}
```

---

## Queues

### Queue Interface

`Queue[T]` is a FIFO (first-in-first-out) collection.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of elements |
| `IsEmpty() bool` | Reports whether the queue is empty |
| `IsNotEmpty() bool` | Reports whether the queue contains at least one element |
| `Clear()` | Removes all elements |
| `String() string` | Returns a string representation |
| `Enqueue(element T)` | Adds element to back |
| `EnqueueAll(elements ...T)` | Adds all to back |
| `Dequeue() (T, bool)` | Removes and returns front |
| `Peek() (T, bool)` | Returns front without removing |
| `ToSlice() []T` | Returns elements front to back |
| `Seq() iter.Seq[T]` | Returns sequence front to back (FIFO) |

### ArrayQueue

Slice-backed queue with amortized O(1) operations.

**Constructors:**

```go
func NewArrayQueue[T any]() Queue[T]
func NewArrayQueueWithCapacity[T any](capacity int) Queue[T]
func NewArrayQueueFrom[T any](elements ...T) Queue[T]
```

**Example:**

```go
queue := collections.NewArrayQueue[string]()

queue.Enqueue("first")
queue.Enqueue("second")
queue.Enqueue("third")

front, _ := queue.Peek()    // "first"
v, _ := queue.Dequeue()     // "first"
v, _ = queue.Dequeue()      // "second"

// Iterate in FIFO order
for v := range queue.Seq() {
    fmt.Println(v)  // "third"
}
```

### PriorityQueue

Heap-based priority queue with O(log n) push/pop and O(1) peek. By default uses min-heap (smallest element has highest priority).

**Constructors:**

```go
func NewPriorityQueue[T any](cmp Comparator[T]) PriorityQueue[T]
func NewPriorityQueueOrdered[T Ordered]() PriorityQueue[T]           // min-heap
func NewMaxPriorityQueue[T Ordered]() PriorityQueue[T]               // max-heap
func NewPriorityQueueWithCapacity[T any](cmp Comparator[T], capacity int) PriorityQueue[T]
func NewPriorityQueueFrom[T any](cmp Comparator[T], elements ...T) PriorityQueue[T]
```

**ToSlice vs ToSortedSlice:**
- `ToSlice()` returns elements in heap order (not sorted) - O(n)
- `ToSortedSlice()` returns elements sorted by priority - O(n log n)

**Example (min-heap):**

```go
// Min-heap: smallest element has highest priority
pq := collections.NewPriorityQueueOrdered[int]()
pq.PushAll(5, 1, 3, 2, 4)

v, _ := pq.Pop()  // 1 (smallest)
v, _ = pq.Pop()   // 2
v, _ = pq.Peek()  // 3 (without removing)

// ToSlice: heap order (not sorted)
// ToSortedSlice: priority order (sorted)
sorted := pq.ToSortedSlice()  // [3, 4, 5]
```

**Example (max-heap):**

```go
// Max-heap: largest element has highest priority
pq := collections.NewMaxPriorityQueue[int]()
pq.PushAll(5, 1, 3, 2, 4)

v, _ := pq.Pop()  // 5 (largest)
v, _ = pq.Pop()   // 4
```

---

## Deques

### Deque Interface

`Deque[T]` is a double-ended queue supporting insertion and removal at both ends.

| Method | Description |
|--------|-------------|
| `Size() int` | Returns the number of elements |
| `IsEmpty() bool` | Reports whether the deque is empty |
| `IsNotEmpty() bool` | Reports whether the deque contains at least one element |
| `Clear()` | Removes all elements |
| `String() string` | Returns a string representation |
| `PushFront(element T)` | Adds element to front |
| `PopFront() (T, bool)` | Removes and returns front |
| `PeekFront() (T, bool)` | Returns front without removing |
| `PushBack(element T)` | Adds element to back |
| `PopBack() (T, bool)` | Removes and returns back |
| `PeekBack() (T, bool)` | Returns back without removing |
| `ToSlice() []T` | Returns elements front to back |
| `Seq() iter.Seq[T]` | Returns sequence front to back |
| `Reversed() iter.Seq[T]` | Returns sequence back to front |

### ArrayDeque

Circular buffer-backed deque with O(1) operations at both ends.

**Constructors:**

```go
func NewArrayDeque[T any]() Deque[T]
func NewArrayDequeWithCapacity[T any](capacity int) Deque[T]
func NewArrayDequeFrom[T any](elements ...T) Deque[T]
```

**Example:**

```go
deque := collections.NewArrayDeque[int]()

// Use as stack (LIFO)
deque.PushBack(1)
deque.PushBack(2)
v, _ := deque.PopBack()  // 2

// Use as queue (FIFO)
deque.PushBack(3)
v, _ = deque.PopFront()  // 1

// Double-ended operations
deque.PushFront(0)
deque.PushBack(4)

// Iterate both directions
for v := range deque.Seq() {
    fmt.Println(v)  // front to back
}

for v := range deque.Reversed() {
    fmt.Println(v)  // back to front
}
```

---

## License

MIT, see [LICENSE](LICENSE).
