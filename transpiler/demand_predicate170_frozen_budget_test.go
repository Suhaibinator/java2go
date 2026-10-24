package transpiler

import "testing"

func TestDemandPredicate170FrozenCgoBudget(t *testing.T) {
	r := demandPredicate170Measure(t)
	if r.Scopes != 4 {
		t.Fatalf("input classes were discarded: %d", r.Scopes)
	}
	if r.Allocs > 85 || r.Bytes > 6656 {
		t.Errorf("allocation shield regressed: %.0f allocations / %d bytes", r.Allocs, r.Bytes)
	}
	if r.Cgo > 464 {
		t.Errorf("FROZEN_DEMAND_CGO_RED: %d actual Cgo calls exceed 464", r.Cgo)
	}
}
