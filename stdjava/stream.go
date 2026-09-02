package stdjava

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
	// Java throws IllegalArgumentException for a negative limit. Without this
	// check the make below fails with "makeslice: len out of range", which the
	// panic normalizer maps to NegativeArraySizeException, so a Java
	// `catch (IllegalArgumentException e)` would not catch it.
	if maxSize < 0 {
		panic(NewIllegalArgumentException("Stream.limit requires a non-negative count"))
	}
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
// Stream.sorted(). Java's sort is stable, which is observable once the element
// type's ordering does not distinguish every element.
func StreamSorted[T any](s Stream[T]) Stream[T] {
	out := s.ToSlice()
	SortSliceStableNatural(out)
	return Stream[T]{elements: out}
}

// StreamSortedWith returns a stream sorted by an explicit comparator, matching
// Stream.sorted(Comparator). It is stable, as Java's is.
func StreamSortedWith[T any](s Stream[T], c Comparator[T]) Stream[T] {
	out := s.ToSlice()
	SortSliceWith(out, c)
	return Stream[T]{elements: out}
}

// Skip returns a stream with the first n elements discarded, matching
// Stream.skip.
func (s Stream[T]) Skip(n int64) Stream[T] {
	if n < 0 {
		panic(NewIllegalArgumentException("Stream.skip requires a non-negative count"))
	}
	if n >= int64(len(s.elements)) {
		return Stream[T]{}
	}
	out := make([]T, int64(len(s.elements))-n)
	copy(out, s.elements[n:])
	return Stream[T]{elements: out}
}

// Peek returns an equivalent stream, invoking action on each element as it is
// consumed, matching Stream.peek.
//
// Because this runtime is eager, peek's action runs over every element when the
// pipeline stage is built, whereas Java runs it only for the elements a terminal
// operation actually pulls. A peek with observable side effects downstream of a
// short-circuiting terminal can therefore see more elements than in Java.
func (s Stream[T]) Peek(action func(T)) Stream[T] {
	for _, e := range s.elements {
		action(e)
	}
	return s
}

// FindFirst returns the first element, matching Stream.findFirst.
func (s Stream[T]) FindFirst() Optional[T] {
	if len(s.elements) == 0 {
		return Optional[T]{}
	}
	return OptionalOf(s.elements[0])
}

// FindAny returns some element, matching Stream.findAny. Evaluation here is
// sequential, so it returns the first, which Java's contract permits.
func (s Stream[T]) FindAny() Optional[T] {
	return s.FindFirst()
}

// Sequential and Parallel return the stream unchanged. Parallel streams are not
// modelled; everything runs sequentially, which preserves every ordering
// guarantee Java makes and only forgoes the concurrency.
func (s Stream[T]) Sequential() Stream[T] { return s }
func (s Stream[T]) Parallel() Stream[T]   { return s }

// Unordered returns the stream unchanged; this runtime always keeps encounter
// order, which Java permits an unordered stream to do.
func (s Stream[T]) Unordered() Stream[T] { return s }

// StreamDistinct returns a stream with duplicates removed, keeping the first
// occurrence of each, matching Stream.distinct. The element type must be a Go
// comparable type, which is what Java's equals/hashCode contract maps onto here.
func StreamDistinct[T comparable](s Stream[T]) Stream[T] {
	seen := make(map[T]struct{}, len(s.elements))
	out := make([]T, 0, len(s.elements))
	for _, e := range s.elements {
		if _, duplicate := seen[e]; duplicate {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return Stream[T]{elements: out}
}

// StreamFlatMap maps each element to a stream and concatenates the results,
// matching Stream.flatMap.
func StreamFlatMap[T, R any](s Stream[T], mapper func(T) Stream[R]) Stream[R] {
	out := []R{}
	for _, e := range s.elements {
		out = append(out, mapper(e).elements...)
	}
	return Stream[R]{elements: out}
}

// StreamMin and StreamMax return the smallest/largest element by natural
// ordering, matching Stream.min/max with a natural-order comparator. Java
// returns the first such element, so ties keep the earlier one.
func StreamMin[T any](s Stream[T]) Optional[T] {
	return StreamMinWith(s, nil)
}

func StreamMax[T any](s Stream[T]) Optional[T] {
	return StreamMaxWith(s, nil)
}

// StreamMinWith returns the smallest element under an explicit comparator,
// matching Stream.min(Comparator). A nil comparator means natural ordering.
func StreamMinWith[T any](s Stream[T], c Comparator[T]) Optional[T] {
	if len(s.elements) == 0 {
		return Optional[T]{}
	}
	best := s.elements[0]
	for _, e := range s.elements[1:] {
		if streamCompare(c, e, best) < 0 {
			best = e
		}
	}
	return OptionalOf(best)
}

// StreamMaxWith returns the largest element under an explicit comparator,
// matching Stream.max(Comparator).
func StreamMaxWith[T any](s Stream[T], c Comparator[T]) Optional[T] {
	if len(s.elements) == 0 {
		return Optional[T]{}
	}
	best := s.elements[0]
	for _, e := range s.elements[1:] {
		if streamCompare(c, e, best) > 0 {
			best = e
		}
	}
	return OptionalOf(best)
}

// streamCompare applies a comparator, falling back to natural ordering when it
// is nil.
func streamCompare[T any](c Comparator[T], a, b T) int32 {
	if c == nil {
		return javaCompareValues(a, b)
	}
	return c(a, b)
}

// StreamReduceOptional performs a reduction with no identity value, matching
// Stream.reduce(accumulator). An empty stream reduces to an empty Optional.
func StreamReduceOptional[T any](s Stream[T], accumulator func(T, T) T) Optional[T] {
	if len(s.elements) == 0 {
		return Optional[T]{}
	}
	result := s.elements[0]
	for _, e := range s.elements[1:] {
		result = accumulator(result, e)
	}
	return OptionalOf(result)
}

// StreamReduceCombining performs a reduction to a result type unrelated to the
// element type, matching Stream.reduce(identity, accumulator, combiner). The
// combiner exists in Java only to merge partial results across parallel splits;
// evaluation here is sequential, so it is accepted and never invoked.
func StreamReduceCombining[T, R any](s Stream[T], identity R, accumulator func(R, T) R, combiner func(R, R) R) R {
	result := identity
	for _, e := range s.elements {
		result = accumulator(result, e)
	}
	return result
}
