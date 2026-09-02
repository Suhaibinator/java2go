package stdjava

import "testing"

func TestStreamToSetKeepsFirstOccurrence(t *testing.T) {
	set := StreamToSet(NewStream("b", "a", "b"))
	if set.Size() != 2 {
		t.Fatalf("StreamToSet size = %d, want 2", set.Size())
	}
	if got := set.Slice(); got[0] != "b" || got[1] != "a" {
		t.Fatalf("StreamToSet = %v, want insertion order [b a]", got)
	}
}

func TestStreamJoiningArities(t *testing.T) {
	s := NewStream("a", "b", "c")
	if got := StreamJoining(s, "", "", ""); got != "abc" {
		t.Fatalf("joining() = %q, want %q", got, "abc")
	}
	if got := StreamJoining(s, ", ", "", ""); got != "a, b, c" {
		t.Fatalf("joining(sep) = %q, want %q", got, "a, b, c")
	}
	if got := StreamJoining(s, ", ", "[", "]"); got != "[a, b, c]" {
		t.Fatalf("joining(sep, prefix, suffix) = %q, want %q", got, "[a, b, c]")
	}
	// An empty stream still gets the prefix and suffix, as Java's does.
	if got := StreamJoining(NewStream[string](), ",", "[", "]"); got != "[]" {
		t.Fatalf("joining on an empty stream = %q, want %q", got, "[]")
	}
}

// Java's toMap throws on a duplicate key rather than overwriting.
func TestStreamToMapRejectsDuplicateKeys(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("toMap with a duplicate key did not panic")
		}
		if !CaughtAs(recovered, "IllegalStateException") {
			t.Fatalf("panicked with %v, want IllegalStateException", recovered)
		}
	}()
	StreamToMap(NewStream("aa", "ab"),
		func(s string) string { return s[:1] },
		func(s string) string { return s })
}

// The merge function receives the existing value first, as Java's does.
func TestStreamToMapMergingOrder(t *testing.T) {
	merged := StreamToMapMerging(NewStream("aa", "ab"),
		func(s string) string { return s[:1] },
		func(s string) string { return s },
		func(existing, incoming string) string { return existing + "|" + incoming })
	if got := merged.Get("a"); got != "aa|ab" {
		t.Fatalf("merged value = %q, want %q", got, "aa|ab")
	}
}

func TestStreamGroupingByKeepsEncounterOrderWithinGroups(t *testing.T) {
	grouped := StreamGroupingBy(NewStream("fig", "pear", "kiwi"), func(s string) int32 { return int32(len(s)) })
	four := grouped.Get(4)
	if four.Size() != 2 || four.Get(0) != "pear" || four.Get(1) != "kiwi" {
		t.Fatalf("group = %v, want [pear kiwi] in encounter order", four.Slice())
	}
	if grouped.Get(3).Size() != 1 {
		t.Fatalf("group of length 3 = %v, want one element", grouped.Get(3).Slice())
	}
}

func TestStreamGroupingByDownstream(t *testing.T) {
	counts := StreamGroupingByDownstream(
		NewStream("fig", "pear", "kiwi"),
		func(s string) int32 { return int32(len(s)) },
		func(group Stream[string]) int64 { return group.Count() },
	)
	if got := counts.Get(4); got != 2 {
		t.Fatalf("count for length 4 = %d, want 2", got)
	}
	if got := counts.Get(3); got != 1 {
		t.Fatalf("count for length 3 = %d, want 1", got)
	}
}

// Java's partitioningBy always returns both entries, even when one side is
// empty, and reports false before true.
func TestStreamPartitioningByAlwaysHasBothSides(t *testing.T) {
	partitioned := StreamPartitioningBy(NewStream("aa", "bb"), func(s string) bool { return len(s) > 5 })
	if partitioned.Size() != 2 {
		t.Fatalf("partition size = %d, want 2", partitioned.Size())
	}
	if partitioned.Get(true).Size() != 0 {
		t.Fatalf("true side = %v, want empty", partitioned.Get(true).Slice())
	}
	if partitioned.Get(false).Size() != 2 {
		t.Fatalf("false side = %v, want both elements", partitioned.Get(false).Slice())
	}
	if keys := partitioned.KeySet(); len(keys) != 2 || keys[0] != false {
		t.Fatalf("partition keys = %v, want false first", keys)
	}
}

func TestStreamSummingAndAveraging(t *testing.T) {
	words := NewStream("a", "bbb")
	if got := StreamSummingOf(words, func(s string) int32 { return int32(len(s)) }); got != 4 {
		t.Fatalf("StreamSummingOf = %d, want 4", got)
	}
	if got := StreamAveragingOf(words, func(s string) int32 { return int32(len(s)) }); got != 2.0 {
		t.Fatalf("StreamAveragingOf = %v, want 2", got)
	}
	// Java's averaging collectors report 0.0 for an empty stream, unlike
	// IntStream.average, which reports an empty OptionalDouble.
	if got := StreamAveragingOf(NewStream[string](), func(s string) int32 { return 0 }); got != 0 {
		t.Fatalf("StreamAveragingOf on an empty stream = %v, want 0", got)
	}
}

func TestStreamCounting(t *testing.T) {
	if got := StreamCounting(NewStream("a", "b")); got != 2 {
		t.Fatalf("StreamCounting = %d, want 2", got)
	}
}

// K18: the three-argument reduce reduces to a type unrelated to the element
// type. The combiner is accepted and never invoked, since evaluation is
// sequential.
func TestStreamReduceCombining(t *testing.T) {
	combinerCalls := 0
	got := StreamReduceCombining(NewStream[int32](1, 2, 3), "",
		func(acc string, value int32) string { return acc + "x" },
		func(a, b string) string { combinerCalls++; return a + b })
	if got != "xxx" {
		t.Fatalf("StreamReduceCombining = %q, want %q", got, "xxx")
	}
	if combinerCalls != 0 {
		t.Fatalf("combiner invoked %d times, want 0 in sequential evaluation", combinerCalls)
	}
}
