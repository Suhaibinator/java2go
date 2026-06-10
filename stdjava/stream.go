package stdjava

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// This file implements the slice-backed stream type that java.util.stream.Stream
// is mapped onto.
//
// Divergence from Java semantics (documented approximation):
//   - Evaluation is EAGER, not lazy: every intermediate operation (filter, map,
//     sorted, limit) materializes a new slice immediately, whereas Java builds a
//     lazy pipeline evaluated only at the terminal operation. For pure functions
//     without side effects the observable result is identical; a pipeline whose
//     intermediate lambdas have side effects, or that relies on short-circuiting
//     to avoid evaluating elements, can observe a different number of lambda
//     invocations than Java.
//   - Streams here are single-type Go values, so Stream.map (T -> R) that changes
//     the element type is the free function StreamMap, not a method (a Go method
//     cannot introduce a new type parameter).
//   - Parallel streams are not modeled; everything runs sequentially.

// Stream is an eager, slice-backed stream of elements.
type Stream[T any] struct {
	elements []T
}

// NewStream returns a Stream over a copy of the given elements, matching
// Stream.of(...).
func NewStream[T any](elements ...T) Stream[T] {
	cp := make([]T, len(elements))
	copy(cp, elements)
	return Stream[T]{elements: cp}
}

// StreamOfSlice returns a Stream over a copy of a slice, used for
// Collection.stream() where the backing slice is already in hand.
func StreamOfSlice[T any](elements []T) Stream[T] {
	return NewStream(elements...)
}

// Filter returns a stream of the elements matching predicate, matching
// Stream.filter.
func (s Stream[T]) Filter(predicate func(T) bool) Stream[T] {
	out := make([]T, 0, len(s.elements))
	for _, e := range s.elements {
		if predicate(e) {
			out = append(out, e)
		}
	}
	return Stream[T]{elements: out}
}

// Limit returns a stream truncated to at most maxSize elements, matching
// Stream.limit.
func (s Stream[T]) Limit(maxSize int32) Stream[T] {
	if int(maxSize) >= len(s.elements) {
		return s
	}
	out := make([]T, maxSize)
	copy(out, s.elements[:maxSize])
	return Stream[T]{elements: out}
}

// ForEach invokes action for each element in encounter order, matching
// Stream.forEach.
func (s Stream[T]) ForEach(action func(T)) {
	for _, e := range s.elements {
		action(e)
	}
}

// Count returns the number of elements, matching Stream.count.
func (s Stream[T]) Count() int64 {
	return int64(len(s.elements))
}

// AnyMatch reports whether any element matches predicate, matching
// Stream.anyMatch.
func (s Stream[T]) AnyMatch(predicate func(T) bool) bool {
	for _, e := range s.elements {
		if predicate(e) {
			return true
		}
	}
	return false
}

// AllMatch reports whether every element matches predicate, matching
// Stream.allMatch.
func (s Stream[T]) AllMatch(predicate func(T) bool) bool {
	for _, e := range s.elements {
		if !predicate(e) {
			return false
		}
	}
	return true
}

// NoneMatch reports whether no element matches predicate, matching
// Stream.noneMatch.
func (s Stream[T]) NoneMatch(predicate func(T) bool) bool {
	return !s.AnyMatch(predicate)
}

// ToList collects the elements into a List, matching
// Stream.collect(Collectors.toList()) / Stream.toList().
func (s Stream[T]) ToList() *List[T] {
	return NewListFrom(s.elements...)
}

// ToSlice returns the elements as a slice (used internally and for toArray).
func (s Stream[T]) ToSlice() []T {
	cp := make([]T, len(s.elements))
	copy(cp, s.elements)
	return cp
}

// StreamMap applies mapper to each element, returning a stream of results. It is
// a free function because Go methods cannot introduce a new type parameter.
func StreamMap[T, R any](s Stream[T], mapper func(T) R) Stream[R] {
	out := make([]R, len(s.elements))
	for i, e := range s.elements {
		out[i] = mapper(e)
	}
	return Stream[R]{elements: out}
}

// StreamReduce performs a reduction with an identity value and an accumulator,
// matching Stream.reduce(identity, accumulator).
func StreamReduce[T any](s Stream[T], identity T, accumulator func(T, T) T) T {
	result := identity
	for _, e := range s.elements {
		result = accumulator(result, e)
	}
	return result
}

// StreamSorted returns a stream sorted by natural ordering, matching
// Stream.sorted() for a Comparable element type.
func StreamSorted[T constraints.Ordered](s Stream[T]) Stream[T] {
	out := make([]T, len(s.elements))
	copy(out, s.elements)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return Stream[T]{elements: out}
}
