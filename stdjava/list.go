package stdjava

// This file implements the slice-backed list type that java.util.List
// implementations (ArrayList, LinkedList) are mapped onto. The Java List
// interface and its common implementations share one Go type here; the
// distinction between array- and linked-list performance is not modelled.
//
// A List is a pointer type so that mutating methods (Add, Remove, ...) are
// visible to all holders of the reference, matching Java reference semantics.
// Enhanced-for over a List is lowered by the transpiler to range over Slice().

// List is a generic, slice-backed list matching the subset of java.util.List
// used by transpiled code.
type List[T any] struct {
	elements []T
}

// NewList returns an empty List, matching `new ArrayList<>()` / `new LinkedList<>()`.
func NewList[T any]() *List[T] {
	return &List[T]{}
}

// NewListFrom returns a List seeded with the given elements, used for
// Arrays.asList and List.of style construction.
func NewListFrom[T any](elements ...T) *List[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return &List[T]{elements: cp}
}

// Add appends an element and returns true, matching List.add (which always
// returns true for a List).
func (l *List[T]) Add(element T) bool {
	l.elements = append(l.elements, element)
	return true
}

// Get returns the element at index, matching List.get.
func (l *List[T]) Get(index int32) T {
	return l.elements[index]
}

// Set replaces the element at index and returns the previous value, matching
// List.set.
func (l *List[T]) Set(index int32, element T) T {
	old := l.elements[index]
	l.elements[index] = element
	return old
}

// Size returns the number of elements, matching List.size.
func (l *List[T]) Size() int32 {
	return int32(len(l.elements))
}

// IsEmpty reports whether the list has no elements, matching List.isEmpty.
func (l *List[T]) IsEmpty() bool {
	return len(l.elements) == 0
}

// Clear removes all elements, matching List.clear.
func (l *List[T]) Clear() {
	l.elements = nil
}

// RemoveAt removes and returns the element at index, matching List.remove(int).
func (l *List[T]) RemoveAt(index int32) T {
	old := l.elements[index]
	l.elements = append(l.elements[:index], l.elements[index+1:]...)
	return old
}

// AddAll appends every element of other and returns true if any were added,
// matching List.addAll.
func (l *List[T]) AddAll(other *List[T]) bool {
	if other == nil || len(other.elements) == 0 {
		return false
	}
	l.elements = append(l.elements, other.elements...)
	return true
}

// ToArray returns a copy of the backing slice, matching List.toArray.
func (l *List[T]) ToArray() []T {
	cp := make([]T, len(l.elements))
	copy(cp, l.elements)
	return cp
}

// Contains reports whether the list holds an element equal to target, matching
// List.contains. Equality uses ObjectsEqual (Java-style value equality).
func (l *List[T]) Contains(target T) bool {
	return l.IndexOf(target) >= 0
}

// IndexOf returns the index of the first element equal to target, or -1,
// matching List.indexOf.
func (l *List[T]) IndexOf(target T) int32 {
	for i, e := range l.elements {
		if ObjectsEqual(e, target) {
			return int32(i)
		}
	}
	return -1
}

// Slice returns the backing slice for iteration. The transpiler lowers an
// enhanced-for over a List to a range over this. The returned slice aliases the
// list's storage and must not be retained across mutations.
func (l *List[T]) Slice() []T {
	return l.elements
}
