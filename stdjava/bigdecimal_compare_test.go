package stdjava

import "testing"

var bigDecimalEqualScaleAllocationResult int32

func TestBigDecimalCompareToEqualScaleDoesNotAllocate(t *testing.T) {
	left := NewBigDecimal("123456789012345678901234567890.00")
	right := NewBigDecimal("123456789012345678901234567891.00")
	allocations := testing.AllocsPerRun(100, func() {
		bigDecimalEqualScaleAllocationResult = left.CompareTo(right)
	})
	if allocations != 0 {
		t.Fatalf("equal-scale compare allocated %g, want zero", allocations)
	}
	if bigDecimalEqualScaleAllocationResult != -1 {
		t.Fatal("incorrect result")
	}
}
