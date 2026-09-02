package stdjava

import (
	"math"
	"testing"
)

func TestIntStreamRanges(t *testing.T) {
	assertInt32Slice(t, IntStreamRange(0, 4).ToSlice(), 0, 1, 2, 3)
	assertInt32Slice(t, IntStreamRangeClosed(1, 4).ToSlice(), 1, 2, 3, 4)
	// An empty or reversed range yields nothing, as in Java.
	if got := IntStreamRange(5, 5).Count(); got != 0 {
		t.Fatalf("IntStreamRange(5, 5) yielded %d elements, want 0", got)
	}
	if got := IntStreamRange(5, 1).Count(); got != 0 {
		t.Fatalf("IntStreamRange(5, 1) yielded %d elements, want 0", got)
	}
	if got := IntStreamRangeClosed(5, 1).Count(); got != 0 {
		t.Fatalf("IntStreamRangeClosed(5, 1) yielded %d elements, want 0", got)
	}
	// A single-element closed range must not loop forever at the bound.
	if got := IntStreamRangeClosed(3, 3).ToSlice(); len(got) != 1 || got[0] != 3 {
		t.Fatalf("IntStreamRangeClosed(3, 3) = %v, want [3]", got)
	}
}

func TestLongStreamRanges(t *testing.T) {
	if got := LongStreamRangeClosed(1, 5); StreamSum(got) != 15 {
		t.Fatalf("LongStreamRangeClosed(1, 5) sums to %d, want 15", StreamSum(got))
	}
	if got := LongStreamRange(0, 0).Count(); got != 0 {
		t.Fatalf("LongStreamRange(0, 0) yielded %d elements, want 0", got)
	}
}

func TestStreamSumAndAverage(t *testing.T) {
	if got := StreamSum(NewStream[int32](1, 2, 3)); got != 6 {
		t.Fatalf("StreamSum = %d, want 6", got)
	}
	if got := StreamAverage(NewStream[int32](1, 2, 3, 4)).Get(); got != 2.5 {
		t.Fatalf("StreamAverage = %v, want 2.5", got)
	}
	if StreamAverage(NewStream[int32]()).IsPresent() {
		t.Fatal("StreamAverage on an empty stream reported a value")
	}
}

// Java sums an IntStream in int arithmetic, so the total wraps at 32 bits.
func TestStreamSumWrapsAtTheElementWidth(t *testing.T) {
	const maxInt32 int32 = 2147483647
	if got := StreamSum(NewStream[int32](maxInt32, 1)); got != -2147483648 {
		t.Fatalf("StreamSum overflow = %d, want -2147483648", got)
	}
}

func TestStreamConcatAndEmpty(t *testing.T) {
	joined := StreamConcat(NewStream("a", "b"), NewStream("c"))
	if got := joined.ToSlice(); len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("StreamConcat = %v, want [a b c]", got)
	}
	if got := StreamEmpty[string]().Count(); got != 0 {
		t.Fatalf("StreamEmpty yielded %d elements, want 0", got)
	}
}

func TestStreamOfArray(t *testing.T) {
	primitive := NewPrimitiveArray[int32](3, PrimitiveIntTypeID)
	primitive.Elements[0] = 4
	primitive.Elements[1] = 1
	primitive.Elements[2] = 3
	if got := StreamSum(StreamOfArray[int32](primitive)); got != 8 {
		t.Fatalf("StreamOfArray over a primitive array sums to %d, want 8", got)
	}

	reference := ReferenceArrayLiteral(StringTypeID, "a", "bb")
	if got := StreamOfArray[string](reference).ToSlice(); len(got) != 2 || got[1] != "bb" {
		t.Fatalf("StreamOfArray over a reference array = %v, want [a bb]", got)
	}
}

func TestStreamOfArrayRejectsNil(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("StreamOfArray(nil) did not panic")
		} else if !CaughtAs(recovered, "NullPointerException") {
			t.Fatalf("panicked with %v, want NullPointerException", recovered)
		}
	}()
	StreamOfArray[int32](nil)
}

func TestStreamWideningConversions(t *testing.T) {
	if got := StreamSum(StreamAsLongStream(NewStream[int32](1, 2, 3))); got != int64(6) {
		t.Fatalf("StreamAsLongStream sums to %d, want 6", got)
	}
	if got := StreamSum(StreamAsDoubleStream(NewStream[int32](1, 2, 3))); got != 6.0 {
		t.Fatalf("StreamAsDoubleStream sums to %v, want 6", got)
	}
}

func TestStringCharsStream(t *testing.T) {
	if got := StringCharsStream("abc").Count(); got != 3 {
		t.Fatalf("StringCharsStream(\"abc\") yielded %d elements, want 3", got)
	}
	if got := StreamSum(StringCharsStream("abc")); got != 294 {
		t.Fatalf("StringCharsStream sum = %d, want 294", got)
	}
	if got := StringCharsStream("").Count(); got != 0 {
		t.Fatalf("StringCharsStream(\"\") yielded %d elements, want 0", got)
	}
}

func TestStreamSummaryStatistics(t *testing.T) {
	stats := StreamSummaryStatistics(NewStream[int32](1, 2, 3, 4))
	if stats.GetCount() != 4 {
		t.Fatalf("GetCount = %d, want 4", stats.GetCount())
	}
	if stats.GetSum() != 10 {
		t.Fatalf("GetSum = %d, want 10", stats.GetSum())
	}
	if stats.GetMin() != 1 || stats.GetMax() != 4 {
		t.Fatalf("GetMin/GetMax = %d/%d, want 1/4", stats.GetMin(), stats.GetMax())
	}
	if stats.GetAverage() != 2.5 {
		t.Fatalf("GetAverage = %v, want 2.5", stats.GetAverage())
	}
}

// Java reports Integer.MAX_VALUE / MIN_VALUE for an empty int stream's min/max.
func TestStreamSummaryStatisticsEmptySentinels(t *testing.T) {
	stats := StreamSummaryStatistics(NewStream[int32]())
	if stats.GetCount() != 0 || stats.GetSum() != 0 {
		t.Fatalf("empty stats count/sum = %d/%d, want 0/0", stats.GetCount(), stats.GetSum())
	}
	if stats.GetMin() != 2147483647 {
		t.Fatalf("empty GetMin = %d, want Integer.MAX_VALUE", stats.GetMin())
	}
	if stats.GetMax() != -2147483648 {
		t.Fatalf("empty GetMax = %d, want Integer.MIN_VALUE", stats.GetMax())
	}
	if stats.GetAverage() != 0 {
		t.Fatalf("empty GetAverage = %v, want 0", stats.GetAverage())
	}
}

// The sum must be accumulated at the element's own width. Held in a float64 it
// silently lost precision above 2^53, and the extremes were truncated to int32.
func TestStreamSummaryStatisticsLongPrecision(t *testing.T) {
	const large int64 = 1 << 60
	stats := StreamSummaryStatistics(NewStream[int64](large, 1))
	if got := stats.GetSum(); got != large+1 {
		t.Fatalf("GetSum = %d, want %d", got, large+1)
	}
	if got := stats.GetMax(); got != large {
		t.Fatalf("GetMax = %d, want %d", got, large)
	}
	if got := stats.GetMin(); got != 1 {
		t.Fatalf("GetMin = %d, want 1", got)
	}
}

// Java's LongSummaryStatistics and DoubleSummaryStatistics report their own
// empty sentinels, the latter being the infinities.
func TestStreamSummaryStatisticsEmptySentinelsPerElementType(t *testing.T) {
	longStats := StreamSummaryStatistics(NewStream[int64]())
	if longStats.GetMin() != math.MaxInt64 || longStats.GetMax() != math.MinInt64 {
		t.Fatalf("empty long sentinels = %d/%d, want Long.MAX_VALUE/MIN_VALUE",
			longStats.GetMin(), longStats.GetMax())
	}
	doubleStats := StreamSummaryStatistics(NewStream[float64]())
	if !math.IsInf(doubleStats.GetMin(), 1) || !math.IsInf(doubleStats.GetMax(), -1) {
		t.Fatalf("empty double sentinels = %v/%v, want +Inf/-Inf",
			doubleStats.GetMin(), doubleStats.GetMax())
	}
}

func TestStreamSummaryStatisticsDouble(t *testing.T) {
	stats := StreamSummaryStatistics(NewStream[float64](1.5, 2.5))
	if got := stats.GetSum(); got != 4.0 {
		t.Fatalf("GetSum = %v, want 4", got)
	}
	if got := stats.GetAverage(); got != 2.0 {
		t.Fatalf("GetAverage = %v, want 2", got)
	}
}
