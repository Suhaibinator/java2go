package transpiler

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/symbol"
)

// Include creation/attestation/retirement of each owned batch, rather than
// treating declaration-inventory setup as free in a warm query measurement.
func qualifiedLookup195AmortizedWork(t *testing.T, count int) {
	t.Helper()
	sources := []string{`package p; class Use extends Base {}`, `package p; class Base {public static class Inherited {}}`}
	for index := 2; index < count; index++ {
		sources = append(sources, fmt.Sprintf(`package p; class Cell%d extends Base {public static class Member {}}`, index))
	}
	f := qualifiedLookup195Parsed(t, sources)
	last := f.files[len(f.files)-1].Symbols.BaseClass
	base := f.files[1].Symbols.BaseClass
	queries := []struct {
		name string
		want *symbol.ClassScope
	}{
		{fmt.Sprintf("Cell%d", count-1), last},
		{fmt.Sprintf("Cell%d.Member", count-1), last.Subclasses[0]},
		{fmt.Sprintf("p.Cell%d.Member", count-1), last.Subclasses[0]},
		{"Inherited", base.Subclasses[0]}, {"Missing", nil}, {"p.Absent.Member", nil},
	}
	const batches = 512
	whole := func() {
		session := cliownership.New()
		defer session.Close()
		invocation := newSyntaxInvocation(session, f.files)
		if invocation == nil {
			t.Fatal("owned standalone invocation unavailable")
		}
		defer invocation.close()
		ctx := f.ctx.Clone()
		ctx.syntaxInvocation = invocation
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		previous := activeResolutionFiles
		activeResolutionFiles = sourceOwnershipIndex(ctx)
		defer func() { activeResolutionFiles = previous }()
		for batch := 0; batch < batches; batch++ {
			for repeat := 0; repeat < 8; repeat++ {
				for _, row := range queries {
					if got := resolveClassScopeByQualifiedName(ctx, row.name); got != row.want {
						t.Fatalf("amortized query %q got%p want%p", row.name, got, row.want)
					}
				}
			}
		}
		runtime.KeepAlive(f)
	}
	whole() // Warm native wrappers; every measured call still owns fresh setup.
	allocations := testing.AllocsPerRun(1, whole)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	cgoBefore := runtime.NumCgoCall()
	whole()
	cgoCalls := runtime.NumCgoCall() - cgoBefore
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	data, err := json.Marshal(map[string]any{
		"schema": "qualified-lookup195-amortized-native-baseline-v1", "actual_parsed_files": count,
		"queries_per_batch": len(queries) * 8 * batches, "allocs_per_batch": allocations,
		"measured_batch_cgo_calls": cgoCalls, "measured_batch_totalalloc_bytes": after.TotalAlloc - before.TotalAlloc,
		"measured_batch_mallocs": after.Mallocs - before.Mallocs, "elapsed_nanoseconds_observation_only": elapsed.Nanoseconds(),
		"fresh_owned_invocation_attestation_and_retirement_included": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("QUALIFIED_LOOKUP195_AMORTIZED_BASELINE " + string(data))
	// Frozen only after the actual immutable public195 measurements. The new
	// index must improve complete-batch work, including its own setup cost.
	limits := map[int]struct {
		allocations float64
		cgoCalls    int64
	}{50: {403901, 666300}, 200: {410114, 674550}, 800: {434961, 707550}}
	limit, known := limits[count]
	if !known {
		t.Fatal("unbudgeted workload")
	}
	if allocations > limit.allocations {
		t.Errorf("FROZEN_QUALIFIED_LOOKUP195_ALLOCATION_WORK: observed %.0f > %.0f including fresh invocation setup", allocations, limit.allocations)
	}
	if cgoCalls > limit.cgoCalls {
		t.Errorf("FROZEN_QUALIFIED_LOOKUP195_CGO_NONREGRESSION: observed %d > %d", cgoCalls, limit.cgoCalls)
	}
	// Incremental budget frozen from matched public196 NO-index package-cwd
	// measurements, after its split fastpath independently passed old budgets.
	incrementalLimits := map[int]float64{50: 355977, 200: 362191, 800: 387037}
	if allocations > incrementalLimits[count] {
		t.Errorf("FROZEN_QUALIFIED_LOOKUP196_INCREMENTAL_ALLOCATION_WORK: observed %.0f > %.0f", allocations, incrementalLimits[count])
	}
	runtime.KeepAlive(f)
}

func TestQualifiedLookup195AmortizedWork50(t *testing.T)  { qualifiedLookup195AmortizedWork(t, 50) }
func TestQualifiedLookup195AmortizedWork200(t *testing.T) { qualifiedLookup195AmortizedWork(t, 200) }
func TestQualifiedLookup195AmortizedWork800(t *testing.T) { qualifiedLookup195AmortizedWork(t, 800) }
