package stdjava

import "testing"

func streamValues[T any](s Stream[T]) []T { return s.ToSlice() }

func assertInt32Slice(t *testing.T, got []int32, want ...int32) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestStreamDistinctKeepsFirstOccurrence(t *testing.T) {
	got := streamValues(StreamDistinct(NewStream[int32](5, 3, 5, 1, 3)))
	assertInt32Slice(t, got, 5, 3, 1)
}

func TestStreamSkip(t *testing.T) {
	assertInt32Slice(t, streamValues(NewStream[int32](1, 2, 3, 4).Skip(2)), 3, 4)
	assertInt32Slice(t, streamValues(NewStream[int32](1, 2).Skip(0)), 1, 2)
	if got := NewStream[int32](1, 2).Skip(99).Count(); got != 0 {
		t.Fatalf("Skip past the end left %d elements, want 0", got)
	}
}

func TestStreamSkipRejectsNegativeCount(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("Skip(-1) did not panic")
		} else if !CaughtAs(recovered, "IllegalArgumentException") {
			t.Fatalf("Skip(-1) panicked with %v, want IllegalArgumentException", recovered)
		}
	}()
	NewStream[int32](1).Skip(-1)
}

func TestStreamPeekObservesWithoutChanging(t *testing.T) {
	seen := []int32{}
	got := streamValues(NewStream[int32](1, 2, 3).Peek(func(v int32) { seen = append(seen, v) }))
	assertInt32Slice(t, got, 1, 2, 3)
	assertInt32Slice(t, seen, 1, 2, 3)
}

func TestStreamFindFirstAndFindAny(t *testing.T) {
	if got := NewStream[int32](7, 8).FindFirst().Get(); got != 7 {
		t.Fatalf("FindFirst = %d, want 7", got)
	}
	if got := NewStream[int32](7, 8).FindAny().Get(); got != 7 {
		t.Fatalf("FindAny = %d, want 7", got)
	}
	if NewStream[int32]().FindFirst().IsPresent() {
		t.Fatal("FindFirst on an empty stream reported a value")
	}
}

func TestStreamFlatMapConcatenates(t *testing.T) {
	got := streamValues(StreamFlatMap(NewStream("ab", "cd"), func(s string) Stream[string] {
		return NewStream(s[:1], s[1:])
	}))
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// Java's sorted() is stable, which is observable once the comparator does not
// distinguish every element.
func TestStreamSortedWithIsStable(t *testing.T) {
	type entry struct {
		key   int32
		label string
	}
	sorted := StreamSortedWith(
		NewStream(entry{2, "first"}, entry{1, "second"}, entry{2, "third"}),
		ComparatorComparing(func(e entry) int32 { return e.key }),
	)
	want := []string{"second", "first", "third"}
	for i, e := range sorted.ToSlice() {
		if e.label != want[i] {
			t.Fatalf("StreamSortedWith result[%d] = %q, want %q", i, e.label, want[i])
		}
	}
}

func TestStreamSortedNaturalUsesCompareTo(t *testing.T) {
	got := streamValues(StreamSorted(NewStream[int32](3, 1, 2)))
	assertInt32Slice(t, got, 1, 2, 3)
}

func TestStreamMinMax(t *testing.T) {
	s := NewStream[int32](5, 1, 9)
	if got := StreamMin(s).Get(); got != 1 {
		t.Fatalf("StreamMin = %d, want 1", got)
	}
	if got := StreamMax(s).Get(); got != 9 {
		t.Fatalf("StreamMax = %d, want 9", got)
	}
	if StreamMin(NewStream[int32]()).IsPresent() {
		t.Fatal("StreamMin on an empty stream reported a value")
	}
	descending := ComparatorComparing(func(v int32) int32 { return v }).Reversed()
	if got := StreamMinWith(s, descending).Get(); got != 9 {
		t.Fatalf("StreamMinWith(descending) = %d, want 9", got)
	}
}

// Collections.max/min keep the earlier element on a tie; so do the stream forms.
func TestStreamMinMaxKeepFirstOnTies(t *testing.T) {
	type entry struct {
		key   int32
		label string
	}
	byKey := ComparatorComparing(func(e entry) int32 { return e.key })
	s := NewStream(entry{1, "firstMin"}, entry{9, "firstMax"}, entry{1, "laterMin"}, entry{9, "laterMax"})
	if got := StreamMinWith(s, byKey).Get().label; got != "firstMin" {
		t.Fatalf("StreamMinWith = %q, want %q", got, "firstMin")
	}
	if got := StreamMaxWith(s, byKey).Get().label; got != "firstMax" {
		t.Fatalf("StreamMaxWith = %q, want %q", got, "firstMax")
	}
}

func TestStreamReduceArities(t *testing.T) {
	add := func(a, b int32) int32 { return a + b }
	if got := StreamReduceOptional(NewStream[int32](1, 2, 3), add).Get(); got != 6 {
		t.Fatalf("StreamReduceOptional = %d, want 6", got)
	}
	if StreamReduceOptional(NewStream[int32](), add).IsPresent() {
		t.Fatal("StreamReduceOptional on an empty stream reported a value")
	}
	if got := StreamReduce(NewStream[int32](1, 2, 3), 10, add); got != 16 {
		t.Fatalf("StreamReduce = %d, want 16", got)
	}
	combined := StreamReduceCombining(NewStream("a", "bb"), 0,
		func(acc int32, s string) int32 { return acc + int32(len(s)) },
		func(a, b int32) int32 { return a + b })
	if combined != 3 {
		t.Fatalf("StreamReduceCombining = %d, want 3", combined)
	}
}

func TestStreamParallelIsSequential(t *testing.T) {
	got := streamValues(NewStream[int32](3, 1, 2).Parallel().Sequential().Unordered())
	assertInt32Slice(t, got, 3, 1, 2)
}
