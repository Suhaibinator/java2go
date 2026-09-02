package stdjava

import "strings"

// This file implements the java.util.stream.Collectors surface.
//
// A Collector is not modelled as a runtime value. `collect(Collectors.X(...))`
// is lowered at the call site into a direct call on the stream, because the
// element and result types a Collector carries are Java type arguments that a
// single Go type cannot hold. A collector used in a *downstream* position
// (groupingBy's second argument, mapping's) is lowered instead to a
// `func(Stream[T]) R`, which composes: a downstream collector is applied to each
// group's own stream.
//
// Documented divergence: Java's groupingBy and toMap return a HashMap, whose
// iteration order is unspecified. These return the insertion-ordered stdjava.Map
// (grouped by first appearance of each key), so generated code is deterministic
// where Java is not. Printing such a map directly can therefore agree with one
// JVM run and disagree with another; sort the keys before rendering.

// StreamToSet collects the elements into a Set, matching
// Collectors.toSet / toUnmodifiableSet. Java gives no ordering guarantee for
// toSet; stdjava.Set preserves insertion order.
func StreamToSet[T comparable](s Stream[T]) *Set[T] {
	out := NewSet[T]()
	for _, e := range s.elements {
		out.Add(e)
	}
	return out
}

// StreamJoining concatenates the elements with a separator between them and the
// given prefix and suffix, matching all three arities of Collectors.joining
// (the shorter forms pass empty strings).
func StreamJoining(s Stream[string], separator, prefix, suffix string) string {
	var builder strings.Builder
	builder.WriteString(prefix)
	for index, e := range s.elements {
		if index > 0 {
			builder.WriteString(separator)
		}
		builder.WriteString(e)
	}
	builder.WriteString(suffix)
	return builder.String()
}

// StreamToMap collects the elements into a map keyed and valued by the given
// extractors, matching Collectors.toMap(keyMapper, valueMapper). Java throws
// IllegalStateException on a duplicate key rather than overwriting, so this
// does too; the three-argument form supplies a merge function instead.
func StreamToMap[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V) *Map[K, V] {
	out := NewMap[K, V]()
	for _, e := range s.elements {
		k := key(e)
		if out.ContainsKey(k) {
			panic(NewIllegalStateException("Duplicate key"))
		}
		out.Put(k, value(e))
	}
	return out
}

// StreamToMapMerging is toMap with a merge function resolving duplicate keys,
// matching Collectors.toMap(keyMapper, valueMapper, mergeFunction). The merge
// receives the existing value first, as Java's does.
func StreamToMapMerging[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V, merge func(V, V) V) *Map[K, V] {
	out := NewMap[K, V]()
	for _, e := range s.elements {
		k := key(e)
		v := value(e)
		if out.ContainsKey(k) {
			v = merge(out.Get(k), v)
		}
		out.Put(k, v)
	}
	return out
}

// StreamGroupingBy groups the elements by a classifier, matching
// Collectors.groupingBy(classifier). Each group keeps encounter order.
func StreamGroupingBy[T any, K comparable](s Stream[T], classifier func(T) K) *Map[K, *List[T]] {
	out := NewMap[K, *List[T]]()
	for _, e := range s.elements {
		k := classifier(e)
		group := out.Get(k)
		if !out.ContainsKey(k) {
			group = NewList[T]()
			out.Put(k, group)
		}
		group.Add(e)
	}
	return out
}

// StreamGroupingByDownstream groups the elements and applies a downstream
// collector to each group, matching
// Collectors.groupingBy(classifier, downstream). The downstream is a function
// from the group's own stream to its collected result.
func StreamGroupingByDownstream[T any, K comparable, D any](s Stream[T], classifier func(T) K, downstream func(Stream[T]) D) *Map[K, D] {
	grouped := StreamGroupingBy(s, classifier)
	out := NewMap[K, D]()
	for _, key := range grouped.KeySet() {
		out.Put(key, downstream(StreamOfSlice(grouped.Get(key).Slice())))
	}
	return out
}

// StreamPartitioningBy splits the elements on a predicate, matching
// Collectors.partitioningBy. Java always returns both the false and true
// entries, even when one side is empty, and in that order.
func StreamPartitioningBy[T any](s Stream[T], predicate func(T) bool) *Map[bool, *List[T]] {
	out := NewMap[bool, *List[T]]()
	out.Put(false, NewList[T]())
	out.Put(true, NewList[T]())
	for _, e := range s.elements {
		out.Get(predicate(e)).Add(e)
	}
	return out
}

// StreamPartitioningByDownstream is partitioningBy with a downstream collector
// applied to each side.
func StreamPartitioningByDownstream[T any, D any](s Stream[T], predicate func(T) bool, downstream func(Stream[T]) D) *Map[bool, D] {
	partitioned := StreamPartitioningBy(s, predicate)
	out := NewMap[bool, D]()
	out.Put(false, downstream(StreamOfSlice(partitioned.Get(false).Slice())))
	out.Put(true, downstream(StreamOfSlice(partitioned.Get(true).Slice())))
	return out
}

// StreamCounting is Collectors.counting, which counts elements as a long.
func StreamCounting[T any](s Stream[T]) int64 { return s.Count() }

// StreamAveragingOf and StreamSummingOf back the averaging*/summing* collectors
// by mapping each element to its numeric contribution first. They exist so the
// lowering does not have to compose two calls at the call site.
func StreamSummingOf[T any, N JavaNumber](s Stream[T], value func(T) N) N {
	return StreamSum(StreamMap(s, value))
}

func StreamAveragingOf[T any, N JavaNumber](s Stream[T], value func(T) N) float64 {
	// Java's averaging collectors report 0.0 for an empty stream, unlike
	// IntStream.average, which reports an empty OptionalDouble.
	return StreamAverage(StreamMap(s, value)).OrElse(0)
}
