package stdjava

import (
	"cmp"
	"fmt"
	"reflect"
	"sort"
)

// This file holds shared helpers for the collection types and the static
// utility methods of java.util.Collections and java.util.Arrays.

// ObjectsEqual reports whether two values are equal using Java-style value
// equality. It approximates Object.equals with reflect.DeepEqual, which matches
// equals() for strings, boxed numbers, and value structs; it does not invoke a
// user-defined equals() method.
func ObjectsEqual[T any](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

// SortOrdered sorts a list of ordered elements in place, matching
// Collections.sort on a List of a Comparable type with natural ordering.
func SortOrdered[T cmp.Ordered](l *List[T]) {
	sort.Slice(l.elements, func(i, j int) bool {
		return l.elements[i] < l.elements[j]
	})
}

// ReverseList reverses a list in place, matching Collections.reverse.
func ReverseList[T any](l *List[T]) {
	for i, j := 0, len(l.elements)-1; i < j; i, j = i+1, j-1 {
		l.elements[i], l.elements[j] = l.elements[j], l.elements[i]
	}
}

// MaxOrdered returns the largest element of a list, matching
// Collections.max on a Collection of a Comparable type.
func MaxOrdered[T cmp.Ordered](l *List[T]) T {
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if e > best {
			best = e
		}
	}
	return best
}

// MinOrdered returns the smallest element of a list, matching Collections.min.
func MinOrdered[T cmp.Ordered](l *List[T]) T {
	best := l.elements[0]
	for _, e := range l.elements[1:] {
		if e < best {
			best = e
		}
	}
	return best
}

// EmptyList returns a new empty List, matching Collections.emptyList. The result
// is mutable (the Java immutability guarantee is not modelled).
func EmptyList[T any]() *List[T] {
	return NewList[T]()
}

// SingletonList returns a List containing one element, matching
// Collections.singletonList.
func SingletonList[T any](element T) *List[T] {
	return NewListFrom(element)
}

// UnmodifiableList returns the list unchanged, approximating
// Collections.unmodifiableList. The read-only guarantee is not enforced.
func UnmodifiableList[T any](l *List[T]) *List[T] {
	return l
}

// AsList returns a List backed by the given elements, matching Arrays.asList.
func AsList[T any](elements ...T) *List[T] {
	return NewListFrom(elements...)
}

// SortSlice sorts a slice of ordered elements in place, matching Arrays.sort.
func SortSlice[T cmp.Ordered](elements []T) {
	sort.Slice(elements, func(i, j int) bool {
		return elements[i] < elements[j]
	})
}

// SliceToString returns the Java Arrays.toString form of a slice, e.g.
// "[a, b, c]".
func SliceToString[T any](elements []T) string {
	parts := make([]string, len(elements))
	for i, e := range elements {
		parts[i] = fmt.Sprintf("%v", e)
	}
	out := "["
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out + "]"
}
