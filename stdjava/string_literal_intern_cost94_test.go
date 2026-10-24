package stdjava

import (
	"runtime"
	"slices"
	"sync"
	"testing"
)

var literalInternCostResult *JavaString

// Inputs and canonical representatives are prepared before the warmed measure.
// Resolving a previously rooted literal must not allocate a discarded wrapper
// or copy its immutable UTF16 payload. Results remain consumed and observable.
func TestLiteralInternWarmedAllocationBudget94(t *testing.T) {
	long := make([]uint16, 4096)
	for i := range long {
		long[i] = uint16(0x3bb)
	}
	short := slices.Clone(long[:64])
	for _, row := range []struct {
		name  string
		units []uint16
	}{
		{"empty", nil}, {"short-64", short}, {"long-4096", long}, {"isolated-surrogates", []uint16{0, 'A', 0xd800, 'B', 0xdfff, 0xd83d, 0xde00}},
	} {
		t.Run(row.name, func(t *testing.T) {
			pool := newStringInternPool(nil)
			want := pool.literal(row.units)
			hash := want.HashCode()
			allocations := testing.AllocsPerRun(200, func() { literalInternCostResult = pool.literal(row.units) })
			if literalInternCostResult != want || !slices.Equal(literalInternCostResult.units, row.units) || literalInternCostResult.HashCode() != hash {
				t.Fatal("literal identity/UTF16/hash changed")
			}
			t.Logf("warmed literal allocations/call=%.0f frozen maximum=0 UTF16units=%d", allocations, len(row.units))
			if allocations > 0 {
				t.Fatalf("discarded literal candidate allocations %.0f > frozen0", allocations)
			}
			runtime.KeepAlive(pool)
			runtime.KeepAlive(want)
		})
	}
}
func TestLiteralInternDynamicAdoptionInputIsolation94(t *testing.T) {
	pool := newStringInternPool(nil)
	units := []uint16{0, 'A', 0xd800, 'B'}
	want := slices.Clone(units)
	dynamic := NewJavaStringUTF16(units)
	if pool.intern(dynamic) != dynamic || pool.literal(units) != dynamic {
		t.Fatal("literal did not adopt existing dynamic representative")
	}
	units[1] = 'X'
	literal := pool.literal(units)
	if literal == dynamic || !slices.Equal(dynamic.units, want) || !slices.Equal(literal.units, units) {
		t.Fatal("host slice mutation changed canonical content")
	}
	units[1] = 'A'
	if pool.literal(units) != dynamic {
		t.Fatal("existing fullUTF16 literal identity")
	}
	runtime.KeepAlive(pool)
	runtime.KeepAlive(dynamic)
	runtime.KeepAlive(literal)
}
func TestLiteralInternConcurrentResolution94(t *testing.T) {
	pool := newStringInternPool(nil)
	const count = 32
	units := make([][]uint16, count)
	results := make([]*JavaString, count)
	for i := range units {
		units[i] = []uint16{'A', 0xd800, 0, 'B', 0xdfff}
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range units {
		workers.Add(1)
		go func(i int) { defer workers.Done(); <-start; results[i] = pool.literal(units[i]) }(i)
	}
	close(start)
	workers.Wait()
	want := results[0]
	for i, result := range results {
		if result == nil || result != want || !slices.Equal(result.units, units[i]) {
			t.Fatal("concurrent fullUTF16 canonical representative")
		}
	}
	pool.index.mu.Lock()
	roots := 0
	for _, group := range pool.literals {
		roots += len(group)
	}
	pool.index.mu.Unlock()
	if roots != 1 {
		t.Fatalf("duplicate literal roots %d", roots)
	}
	runtime.KeepAlive(pool)
	runtime.KeepAlive(results)
}
