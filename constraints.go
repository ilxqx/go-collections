package collections

import "cmp"

// Ordered is a constraint for types that support <, ==, > operators.
// It aliases cmp.Ordered from the standard library.
type Ordered = cmp.Ordered

// Comparator compares two values:
// - negative if a < b
// - zero     if a == b
// - positive if a > b.
type Comparator[T any] func(a, b T) int

// Equaler reports whether two values are equal.
type Equaler[T any] func(a, b T) bool

// EqualFunc returns a default Equaler for comparable types using ==.
func EqualFunc[T comparable]() Equaler[T] {
	return func(a, b T) bool { return a == b }
}

// CompareFunc returns a Comparator for Ordered types using cmp.Compare.
func CompareFunc[T Ordered]() Comparator[T] {
	return cmp.Compare[T]
}
