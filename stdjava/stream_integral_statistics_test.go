package stdjava

import (
	"math"
	"testing"
)

type statisticsIntAlias int32
type statisticsLongAlias int64

func TestIntegralStatisticsUseLongAccumulator(t *testing.T) {
	stats := StreamSummaryStatistics(NewStream[int32](math.MaxInt32, math.MaxInt32))
	if got := stats.GetAverage(); got != float64(math.MaxInt32) {
		t.Fatalf("int stats average=%v want %v", got, float64(math.MaxInt32))
	}
	if got := integralSumLong(t, stats); got != 4294967294 {
		t.Fatalf("int stats long sum=%d", got)
	}
	aliases := StreamSummaryStatistics(NewStream[statisticsIntAlias](math.MaxInt32, math.MaxInt32))
	if integralSumLong(t, aliases) != 4294967294 || aliases.GetAverage() != float64(math.MaxInt32) {
		t.Fatal("signed alias accumulator used element width")
	}
	overflow := StreamSummaryStatistics(NewStream[statisticsLongAlias](math.MaxInt64, 1))
	if integralSumLong(t, overflow) != math.MinInt64 || overflow.GetAverage() != float64(math.MinInt64)/2 {
		t.Fatal("long accumulator must wrap at 64 bits")
	}
}

func TestIntegralStreamAverageAccumulatesBeforeDoubleConversion(t *testing.T) {
	stream := NewStream[int64](1<<53, 1, -(1 << 53))
	if got := StreamAverage(stream).Get(); got != 1.0/3 {
		t.Fatalf("long average=%v want %v", got, 1.0/3)
	}
	alias := NewStream[statisticsLongAlias](1<<53, 1, -(1 << 53))
	if got := StreamAverage(alias).Get(); got != 1.0/3 {
		t.Fatalf("aliased long average=%v", got)
	}
	if got := StreamAverage(NewStream[int64](math.MaxInt64, 1)).Get(); got != float64(math.MinInt64)/2 {
		t.Fatalf("overflow long average=%v", got)
	}
}

func integralSumLong(t *testing.T, value any) int64 {
	t.Helper()
	accessor, ok := value.(interface{ GetSumLong() int64 })
	if !ok {
		t.Fatal("integral statistics lack Java long sum accessor")
	}
	return accessor.GetSumLong()
}
