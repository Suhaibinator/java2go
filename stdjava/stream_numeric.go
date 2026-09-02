package stdjava

import "math"

// This file implements the java.util.stream primitive-stream surface. IntStream,
// LongStream and DoubleStream are modelled as Stream[int32], Stream[int64] and
// Stream[float64] rather than as separate types, so every operation on
// Stream[T] is available on them without duplication.
//
// The conversions between them (boxed, asLongStream, mapToInt, ...) therefore
// either widen the element type or are identities, and are spelled out below so
// generated code reads like the Java it came from.

// IntStreamRange returns the ints from startInclusive up to but excluding
// endExclusive, matching IntStream.range. An empty or reversed range yields no
// elements, as in Java.
func IntStreamRange(startInclusive, endExclusive int32) Stream[int32] {
	if endExclusive <= startInclusive {
		return Stream[int32]{}
	}
	out := make([]int32, 0, endExclusive-startInclusive)
	for value := startInclusive; value < endExclusive; value++ {
		out = append(out, value)
	}
	return Stream[int32]{elements: out}
}

// IntStreamRangeClosed is IntStreamRange with the upper bound included,
// matching IntStream.rangeClosed.
func IntStreamRangeClosed(startInclusive, endInclusive int32) Stream[int32] {
	if endInclusive < startInclusive {
		return Stream[int32]{}
	}
	out := make([]int32, 0, endInclusive-startInclusive+1)
	for value := startInclusive; ; value++ {
		out = append(out, value)
		if value == endInclusive {
			break
		}
	}
	return Stream[int32]{elements: out}
}

// LongStreamRange and LongStreamRangeClosed are the 64-bit forms, matching
// LongStream.range and LongStream.rangeClosed.
func LongStreamRange(startInclusive, endExclusive int64) Stream[int64] {
	if endExclusive <= startInclusive {
		return Stream[int64]{}
	}
	out := make([]int64, 0, endExclusive-startInclusive)
	for value := startInclusive; value < endExclusive; value++ {
		out = append(out, value)
	}
	return Stream[int64]{elements: out}
}

func LongStreamRangeClosed(startInclusive, endInclusive int64) Stream[int64] {
	if endInclusive < startInclusive {
		return Stream[int64]{}
	}
	out := make([]int64, 0, endInclusive-startInclusive+1)
	for value := startInclusive; ; value++ {
		out = append(out, value)
		if value == endInclusive {
			break
		}
	}
	return Stream[int64]{elements: out}
}

// StreamEmpty returns a stream with no elements, matching Stream.empty.
func StreamEmpty[T any]() Stream[T] {
	return Stream[T]{}
}

// StreamConcat returns the elements of first followed by those of second,
// matching Stream.concat.
func StreamConcat[T any](first, second Stream[T]) Stream[T] {
	out := make([]T, 0, len(first.elements)+len(second.elements))
	out = append(out, first.elements...)
	out = append(out, second.elements...)
	return Stream[T]{elements: out}
}

// StreamOfArray returns a stream over a Java array, matching Arrays.stream. It
// accepts both array representations: a primitive array's elements are already
// typed, while a reference array erases them to `any`, so each is converted to
// the requested element type.
func StreamOfArray[T any](array any) Stream[T] {
	switch values := array.(type) {
	case nil:
		panic(NewNullPointerException("Arrays.stream on null"))
	case *PrimitiveArray[T]:
		// A `null` Java array reaches here as a TYPED nil, which the `case nil`
		// arm above does not match (that one only catches an untyped nil
		// interface). Without this check the dereference below is a segfault
		// rather than the NullPointerException Java throws.
		if values == nil {
			panic(NewNullPointerException("Arrays.stream on null"))
		}
		return StreamOfSlice(values.Elements)
	case *ReferenceArray:
		if values == nil {
			panic(NewNullPointerException("Arrays.stream on null"))
		}
		out := make([]T, 0, len(values.elements))
		for _, element := range values.elements {
			typed, ok := element.(T)
			if !ok {
				panic(NewClassCastException("Arrays.stream element does not match the requested type"))
			}
			out = append(out, typed)
		}
		return Stream[T]{elements: out}
	case []T:
		return StreamOfSlice(values)
	}
	panic(NewIllegalArgumentException("Arrays.stream requires an array"))
}

// StringCharsStream returns a stream of the string's characters as ints,
// matching String.chars. It reuses the existing rune view from StringChars, so
// it inherits that function's documented approximation: Java yields one element
// per UTF-16 code unit and this yields one per rune, which agree for BMP
// characters.
func StringCharsStream(s string) Stream[int32] {
	runes := StringChars(s)
	out := make([]int32, len(runes))
	for index, r := range runes {
		out[index] = int32(r)
	}
	return Stream[int32]{elements: out}
}

// StreamBoxed returns the stream unchanged, matching IntStream.boxed and its
// siblings. A primitive stream is already a Stream of the boxed type's Go
// counterpart here, so boxing is an identity.
func StreamBoxed[T any](s Stream[T]) Stream[T] { return s }

// StreamSum returns the sum of a numeric stream's elements, matching
// IntStream.sum and its siblings. Java sums an IntStream in int arithmetic, so
// this wraps at the element type's width exactly as Java does.
func StreamSum[T JavaNumber](s Stream[T]) T {
	var total T
	for _, e := range s.elements {
		total += e
	}
	return total
}

// StreamAverage returns the arithmetic mean of a numeric stream, matching
// IntStream.average. It is empty for an empty stream, and the sum is
// accumulated in float64 to match Java, which averages in double.
func StreamAverage[T JavaNumber](s Stream[T]) Optional[float64] {
	if len(s.elements) == 0 {
		return Optional[float64]{}
	}
	total := 0.0
	for _, e := range s.elements {
		total += float64(e)
	}
	return OptionalOf(total / float64(len(s.elements)))
}

// StreamToIntSlice and friends widen a numeric stream, matching
// IntStream.asLongStream and asDoubleStream.
func StreamAsLongStream[T JavaNumber](s Stream[T]) Stream[int64] {
	out := make([]int64, len(s.elements))
	for index, e := range s.elements {
		out[index] = int64(e)
	}
	return Stream[int64]{elements: out}
}

func StreamAsDoubleStream[T JavaNumber](s Stream[T]) Stream[float64] {
	out := make([]float64, len(s.elements))
	for index, e := range s.elements {
		out[index] = float64(e)
	}
	return Stream[float64]{elements: out}
}

// SummaryStatistics is the result of IntStream.summaryStatistics and its long
// and double siblings.
//
// Java has three distinct classes here and their accessors differ in type:
// IntSummaryStatistics.getMin returns int while LongSummaryStatistics.getMin
// returns long, and DoubleSummaryStatistics reports its sum and extremes as
// double with infinite empty sentinels. One generic type carries all three, so
// the element type decides what the accessors return.
//
// The sum is accumulated at the element's own width rather than in float64,
// which would silently lose precision above 2^53 for a long stream.
type SummaryStatistics[T JavaNumber] struct {
	count int64
	sum   T
	min   T
	max   T
	empty bool
}

// StreamSummaryStatistics collects count, sum, min and max in one pass,
// matching IntStream.summaryStatistics.
func StreamSummaryStatistics[T JavaNumber](s Stream[T]) *SummaryStatistics[T] {
	stats := &SummaryStatistics[T]{empty: true}
	for _, e := range s.elements {
		if stats.empty {
			stats.min = e
			stats.max = e
			stats.empty = false
		} else {
			if javaCompareValues(e, stats.min) < 0 {
				stats.min = e
			}
			if javaCompareValues(e, stats.max) > 0 {
				stats.max = e
			}
		}
		stats.count++
		stats.sum += e
	}
	return stats
}

// GetCount matches getCount on all three Java classes.
func (s *SummaryStatistics[T]) GetCount() int64 { return s.count }

// GetSum returns the sum at the element's own width. Java widens an int stream's
// sum to long, which the transpiler reflects in the call's declared result type.
func (s *SummaryStatistics[T]) GetSum() T { return s.sum }

// GetMin and GetMax report Java's empty-stream sentinels: the element type's
// extremes, which for a double stream are the infinities.
func (s *SummaryStatistics[T]) GetMin() T {
	if s.empty {
		return summaryEmptyMin[T]()
	}
	return s.min
}

func (s *SummaryStatistics[T]) GetMax() T {
	if s.empty {
		return summaryEmptyMax[T]()
	}
	return s.max
}

// GetAverage is zero for an empty stream, as Java's is, and is computed in
// double as Java computes it.
func (s *SummaryStatistics[T]) GetAverage() float64 {
	if s.count == 0 {
		return 0
	}
	return float64(s.sum) / float64(s.count)
}

// summaryEmptyMin and summaryEmptyMax return the sentinel Java reports for an
// empty stream: Integer.MAX_VALUE / MIN_VALUE for an int stream, the long
// equivalents for a long stream, and the infinities for a double stream.
func summaryEmptyMin[T JavaNumber]() T {
	var zero T
	switch any(zero).(type) {
	case float32:
		return T(float32(math.Inf(1)))
	case float64:
		return T(math.Inf(1))
	}
	// Routed through a variable because a constant conversion inside a generic
	// must fit every type in the constraint's type set, and these do not.
	sentinel := int64(math.MaxInt32)
	switch any(zero).(type) {
	case int64:
		sentinel = math.MaxInt64
	case int16:
		sentinel = math.MaxInt16
	case int8:
		sentinel = math.MaxInt8
	}
	return T(sentinel)
}

func summaryEmptyMax[T JavaNumber]() T {
	var zero T
	switch any(zero).(type) {
	case float32:
		return T(float32(math.Inf(-1)))
	case float64:
		return T(math.Inf(-1))
	}
	sentinel := int64(math.MinInt32)
	switch any(zero).(type) {
	case int64:
		sentinel = math.MinInt64
	case int16:
		sentinel = math.MinInt16
	case int8:
		sentinel = math.MinInt8
	}
	return T(sentinel)
}
