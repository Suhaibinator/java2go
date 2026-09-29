package stdjava

import "strings"

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
	elements    []T
	array       *ReferenceArray
	elementType TypeID
	fixed       bool
	modCount    uint64
}

// NewList returns an empty List, matching `new ArrayList<>()` / `new LinkedList<>()`.
func NewList[T any]() *List[T] {
	return &List[T]{}
}

// NewListFrom returns a mutable List containing a copy of the given elements.
func NewListFrom[T any](elements ...T) *List[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return &List[T]{elements: cp}
}

// Add appends an element and returns true, matching List.add (which always
// returns true for a List).
func (l *List[T]) Add(element T) bool {
	l.requireResizable()
	l.modCount++
	l.elements = append(l.elements, element)
	return true
}

// Get returns the element at index, matching List.get.
func (l *List[T]) Get(index int32) T {
	if l.array != nil {
		return ReferenceArrayGet[T](l.array, index, l.elementType)
	}
	return l.elements[index]
}

// Set replaces the element at index and returns the previous value, matching
// List.set.
func (l *List[T]) Set(index int32, element T) T {
	old := l.Get(index)
	if l.array != nil {
		ReferenceArraySet(l.array, index, element)
		return old
	}
	l.elements[index] = element
	return old
}

// Size returns the number of elements, matching List.size.
func (l *List[T]) Size() int32 {
	if l.array != nil {
		return ReferenceArrayLength(l.array)
	}
	return int32(len(l.elements))
}

// IsEmpty reports whether the list has no elements, matching List.isEmpty.
func (l *List[T]) IsEmpty() bool {
	return l.Size() == 0
}

// Clear removes all elements, matching List.clear.
func (l *List[T]) Clear() {
	if l.Size() != 0 {
		l.requireResizable()
	}
	if !l.fixed {
		l.modCount++
	}
	l.elements = nil
}

// RemoveAt removes and returns the element at index, matching List.remove(int).
func (l *List[T]) RemoveAt(index int32) T {
	l.requireResizable()
	old := l.elements[index]
	l.modCount++
	l.elements = append(l.elements[:index], l.elements[index+1:]...)
	return old
}

// AddAll appends every element of other and returns true if any were added,
// matching List.addAll.
func (l *List[T]) AddAll(other *List[T]) bool {
	if other == nil {
		return false
	}
	if !l.fixed {
		l.modCount++
	}
	if other.Size() == 0 {
		return false
	}
	l.requireResizable()
	l.elements = append(l.elements, other.Slice()...)
	return true
}

// ToArray returns a copy of the backing slice, matching List.toArray.
func (l *List[T]) ToArray() []T {
	cp := NewArray[T](l.Size())
	copy(cp, l.Slice())
	return cp
}

// Contains reports whether the list holds an element equal to target, matching
// List.contains. Equality uses ObjectsEqual (Java-style value equality).
func (l *List[T]) Contains(target any, execution ...*Execution) bool {
	return l.IndexOf(target, execution...) >= 0
}

// IndexOf returns the index of the first element equal to target, or -1,
// matching List.indexOf.
func (l *List[T]) IndexOf(target any, execution ...*Execution) int32 {
	for i := int32(0); i < l.Size(); i++ {
		e := l.Get(i)
		if listElementEqual(target, e, optionalComparisonExecution(execution)) {
			return int32(i)
		}
	}
	return -1
}

// RemoveObject removes the first equal element, matching List.remove(Object).
// RemoveAt implements the separate Java overload that accepts an int index.
func (l *List[T]) RemoveObject(target any, execution ...*Execution) bool {
	index := l.IndexOf(target, execution...)
	if index < 0 {
		return false
	}
	l.RemoveAt(index)
	return true
}

// Slice returns the ordinary backing slice, or a converted snapshot for an
// array-backed view. Java iteration uses CollectionIterationElements so each
// array slot is read only when the iterator advances.
func (l *List[T]) Slice() []T {
	if l.array != nil {
		return ReferenceArrayElements[T](l.array, l.elementType)
	}
	return l.elements
}

// String returns the Java AbstractCollection.toString form, e.g. "[a, b, c]", so
// that a List printed via fmt (System.out.println) matches Java's output.
func (l *List[T]) String() string {
	parts := make([]string, l.Size())
	for i := int32(0); i < l.Size(); i++ {
		e := l.Get(i)
		parts[i] = StringValueOf(e)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// List search invokes query.equals(element) even for the same reference.
func listElementEqual(query, element any, execution *Execution) bool {
	if javaReferenceIsNull(query) {
		return javaReferenceIsNull(element)
	}
	return ObjectEqualsExecution(execution, query, element)
}

func (l *List[T]) requireResizable() {
	if l.fixed {
		panic(NewUnsupportedOperationException("fixed-size list"))
	}
}

// AsListArray preserves the shared Java array, including its covariant store
// check. Conversion to each requested Go view occurs when an element is read.
func AsListArray[T any](array *ReferenceArray, elementType TypeID) *List[T] {
	ReferenceArrayLength(array)
	return &List[T]{array: array, elementType: elementType, fixed: true}
}
