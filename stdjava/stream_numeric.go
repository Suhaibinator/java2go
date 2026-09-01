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

// IntSummaryStatistics is the result of IntStream.summaryStatistics. The long
// and double forms share it, since the accessors Java exposes are the same
// shape; the count and sum are held at their widest Java type.
type IntSummaryStatistics struct {
	count int64
	sum   float64
	min   float64
	max   float64
}

// StreamSummaryStatistics collects count, sum, min, max and average in one
// pass, matching IntStream.summaryStatistics. An empty stream reports Java's
// documented sentinels: a zero count and sum, with min and max at the element
// type's extremes.
func StreamSummaryStatistics[T JavaNumber](s Stream[T]) *IntSummaryStatistics {
	stats := &IntSummaryStatistics{
		min: math.Inf(1),
		max: math.Inf(-1),
	}
	for _, e := range s.elements {
		value := float64(e)
		stats.count++
		stats.sum += value
		if value < stats.min {
			stats.min = value
		}
		if value > stats.max {
			stats.max = value
		}
	}
	return stats
}

func (s *IntSummaryStatistics) GetCount() int64 { return s.count }
func (s *IntSummaryStatistics) GetSum() int64   { return int64(s.sum) }

// GetMin and GetMax report Java's empty-stream sentinels, Integer.MAX_VALUE and
// Integer.MIN_VALUE, when no element was seen.
func (s *IntSummaryStatistics) GetMin() int32 {
	if s.count == 0 {
		return math.MaxInt32
	}
	return int32(s.min)
}

func (s *IntSummaryStatistics) GetMax() int32 {
	if s.count == 0 {
		return math.MinInt32
	}
	return int32(s.max)
}

// GetAverage is zero for an empty stream, as Java's is.
func (s *IntSummaryStatistics) GetAverage() float64 {
	if s.count == 0 {
		return 0
	}
	return s.sum / float64(s.count)
}
