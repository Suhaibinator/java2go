package stdjava

import "strings"

// This file implements the set type that java.util.Set implementations
// (HashSet, TreeSet) are mapped onto. HashSet and TreeSet share one Go type;
// TreeSet's sorted iteration is not faithfully modelled (insertion order is
// used). Elements must be comparable, which holds for the Java types used as set
// elements in practice.

// Set is a generic set matching the subset of java.util.Set used by transpiled
// code. It is a pointer type so mutations are visible to all holders.
type Set[T comparable] struct {
	backing map[T]struct{}
	// order preserves insertion order for deterministic iteration.
	order []T
}

// NewSet returns an empty Set, matching `new HashSet<>()` / `new TreeSet<>()`.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{backing: make(map[T]struct{})}
}

// Add inserts element and returns true if it was not already present, matching
// Set.add.
func (s *Set[T]) Add(element T) bool {
	if _, exists := s.backing[element]; exists {
		return false
	}
	s.backing[element] = struct{}{}
	s.order = append(s.order, element)
	return true
}

// Contains reports whether element is present, matching Set.contains.
func (s *Set[T]) Contains(element T) bool {
	_, ok := s.backing[element]
	return ok
}

// Remove deletes element and returns true if it was present, matching
// Set.remove.
func (s *Set[T]) Remove(element T) bool {
	if _, ok := s.backing[element]; !ok {
		return false
	}
	delete(s.backing, element)
	for i, e := range s.order {
		if e == element {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// Size returns the number of elements, matching Set.size.
func (s *Set[T]) Size() int32 {
	return int32(len(s.backing))
}

// IsEmpty reports whether the set has no elements, matching Set.isEmpty.
func (s *Set[T]) IsEmpty() bool {
	return len(s.backing) == 0
}

// Clear removes all elements, matching Set.clear.
func (s *Set[T]) Clear() {
	s.backing = make(map[T]struct{})
	s.order = nil
}

// Slice returns the elements in insertion order for iteration. The transpiler
// lowers an enhanced-for over a Set to a range over this.
func (s *Set[T]) Slice() []T {
	cp := make([]T, len(s.order))
	copy(cp, s.order)
	return cp
}

// String returns the Java AbstractCollection.toString form, e.g. "[a, b]", in
// insertion order, so a Set printed via fmt matches Java's output.
func (s *Set[T]) String() string {
	parts := make([]string, len(s.order))
	for i, e := range s.order {
		parts[i] = StringValueOf(e)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
