package stdjava

import (
	"math"
	"testing"
)

// Java's Double.compare is a total order: NaN sorts after everything (including
// itself, so it moves during a sort) and -0.0 sorts before 0.0. Go's `<` and
// cmp.Compare agree with neither.
func TestJavaDoubleCompareMatchesJava(t *testing.T) {
	nan := math.NaN()
	for _, tc := range []struct {
		name        string
		left, right float64
		want        int32
	}{
		{"NaN after a number", nan, 1.0, 1},
		{"number before NaN", 1.0, nan, -1},
		{"NaN equals NaN", nan, nan, 0},
		{"NaN after positive infinity", nan, math.Inf(1), 1},
		{"negative zero before zero", math.Copysign(0, -1), 0.0, -1},
		{"zero after negative zero", 0.0, math.Copysign(0, -1), 1},
		{"zero equals zero", 0.0, 0.0, 0},
		{"ordinary less", 1.0, 2.0, -1},
		{"ordinary greater", 2.0, 1.0, 1},
		{"negative ordering", -2.0, -1.0, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := javaDoubleCompare(tc.left, tc.right); got != tc.want {
				t.Fatalf("javaDoubleCompare(%v, %v) = %d, want %d", tc.left, tc.right, got, tc.want)
			}
		})
	}
}

func TestJavaFloatCompareMatchesJava(t *testing.T) {
	nan := float32(math.NaN())
	if got := javaFloatCompare(nan, 1.0); got != 1 {
		t.Fatalf("javaFloatCompare(NaN, 1) = %d, want 1", got)
	}
	if got := javaFloatCompare(float32(math.Copysign(0, -1)), 0.0); got != -1 {
		t.Fatalf("javaFloatCompare(-0.0, 0.0) = %d, want -1", got)
	}
}

// The bug this pins: Collections.sort and Stream.sorted must agree, and both
// must match Java. Before the fix SortOrdered's fast path used Go's `<` (NaN
// stays put) while StreamSorted used cmp.Compare (NaN first); Java puts it last.
func TestSortOrderedAndStreamSortedAgreeOnNaN(t *testing.T) {
	nan := math.NaN()
	values := []float64{3.0, nan, 1.0}

	list := NewListFrom(values...)
	SortOrdered(list)
	streamed := StreamSorted(NewStream(values...)).ToSlice()

	// Java prints [1.0, 3.0, NaN] for both.
	if list.Get(0) != 1.0 || list.Get(1) != 3.0 || !math.IsNaN(list.Get(2)) {
		t.Fatalf("SortOrdered = %v, want [1 3 NaN]", list.Slice())
	}
	if streamed[0] != 1.0 || streamed[1] != 3.0 || !math.IsNaN(streamed[2]) {
		t.Fatalf("StreamSorted = %v, want [1 3 NaN]", streamed)
	}
}

func TestMinMaxOrderedPlaceNaNLikeJava(t *testing.T) {
	nan := math.NaN()
	list := NewListFrom(3.0, nan, 1.0)
	// Java: Collections.max is NaN, Collections.min is 1.0.
	if got := MaxOrdered(list); !math.IsNaN(got) {
		t.Fatalf("MaxOrdered = %v, want NaN", got)
	}
	if got := MinOrdered(list); got != 1.0 {
		t.Fatalf("MinOrdered = %v, want 1.0", got)
	}
}

func TestNaturalOrderComparatorUsesJavaFloatOrdering(t *testing.T) {
	if got := NaturalOrder[float64]()(math.NaN(), 1.0); got != 1 {
		t.Fatalf("NaturalOrder(NaN, 1.0) = %d, want 1", got)
	}
	if got := ComparatorComparing(func(v float64) float64 { return v })(math.NaN(), 1.0); got != 1 {
		t.Fatalf("ComparatorComparing(NaN, 1.0) = %d, want 1", got)
	}
}

// Arrays.sort(double[]) uses the same total order.
func TestSortArrayOnFloatsUsesJavaOrdering(t *testing.T) {
	array := NewPrimitiveArray[float64](3, PrimitiveDoubleTypeID)
	array.Elements[0] = 3.0
	array.Elements[1] = math.NaN()
	array.Elements[2] = 1.0
	SortArray(array)
	if array.Elements[0] != 1.0 || array.Elements[1] != 3.0 || !math.IsNaN(array.Elements[2]) {
		t.Fatalf("SortArray = %v, want [1 3 NaN]", array.Elements)
	}
}

// The previous nil test used an untyped nil, which took a different switch arm
// than a real Java `null` array does; the typed nil segfaulted.
func TestStreamOfArrayRejectsTypedNilPrimitiveArray(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("StreamOfArray on a typed-nil primitive array did not panic")
		}
		if !CaughtAs(recovered, "NullPointerException") {
			t.Fatalf("panicked with %v, want NullPointerException", recovered)
		}
	}()
	var nilArray *PrimitiveArray[int32]
	StreamOfArray[int32](nilArray)
}

func TestStreamOfArrayRejectsTypedNilReferenceArray(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("StreamOfArray on a typed-nil reference array did not panic")
		} else if !CaughtAs(recovered, "NullPointerException") {
			t.Fatalf("panicked with %v, want NullPointerException", recovered)
		}
	}()
	var nilArray *ReferenceArray
	StreamOfArray[string](nilArray)
}

// Java throws IllegalArgumentException for a negative limit, not the
// NegativeArraySizeException that a raw makeslice failure maps to.
func TestStreamLimitRejectsNegativeCount(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("Limit(-1) did not panic")
		}
		if !CaughtAs(recovered, "IllegalArgumentException") {
			t.Fatalf("Limit(-1) panicked with %v, want IllegalArgumentException", recovered)
		}
	}()
	NewStream[int32](1, 2).Limit(-1)
}

// Java documents a null comparator as meaning natural ordering.
func TestMinMaxWithNilComparatorUseNaturalOrdering(t *testing.T) {
	list := NewListFrom[int32](3, 1, 2)
	if got := MaxWith(list, nil); got != 3 {
		t.Fatalf("MaxWith(nil) = %d, want 3", got)
	}
	if got := MinWith(list, nil); got != 1 {
		t.Fatalf("MinWith(nil) = %d, want 1", got)
	}
}
