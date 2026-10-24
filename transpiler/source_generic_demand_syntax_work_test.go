package transpiler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// Bound warmed query growth by the actual live ownership guard and one fresh
// declaration-child context restoration per added root. Empty candidate routes
// must not repeatedly traverse method bodies that cannot demand a source view.
func TestSourceGenericDemandSyntaxWorkGrowth(t *testing.T) {
	const queries = 8
	type sample struct {
		Files, Candidates   int
		API, Guard, Routing demand188Row
	}
	var samples []sample
	for _, noise := range []int{0, 4, 8, 16} {
		func() {
			sources := map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": `package app;import p.Cell;public class Use{Cell<?> value;Object cast(Object value){return (Cell<?>)value;}}`}
			for f := 0; f < noise; f++ {
				var source strings.Builder
				fmt.Fprintf(&source, "package extra;class Body%d{", f)
				for m := 0; m < 8; m++ {
					fmt.Fprintf(&source, "int work%d(int value){", m)
					for x := 0; x < 128; x++ {
						source.WriteString("value++;")
					}
					source.WriteString("return value;}")
				}
				source.WriteString("}")
				sources[fmt.Sprintf("extra/Body%d.java", f)] = source.String()
			}
			ctx, _, _ := demand188Fixture(t, sources)
			want := sourceGenericViewDemandSeeds(ctx)
			if len(want) != 1 || want[0] != findQualifiedSourceClass("p.Cell") {
				t.Fatal("missing real source demand")
			}
			sourceGenericViewDemandSeeds(ctx)
			_, counts := demand188SyntaxWitness(ctx, false)
			api := demand188Measure(t, fmt.Sprintf("growth%d_actual", noise), queries, func() []*symbol.ClassScope {
				got := sourceGenericViewDemandSeeds(ctx)
				demand188Same(t, got, want)
				return got
			})
			guard := demand188Measure(t, fmt.Sprintf("growth%d_guard", noise), queries, func() []*symbol.ClassScope {
				if _, ok := ctx.syntaxInvocation.queryBodies("growth-budget"); !ok {
					t.Fatal("owned graph rejected")
				}
				return nil
			})
			routing := demand188Measure(t, fmt.Sprintf("growth%d_live-root-routing", noise), queries, func() []*symbol.ClassScope {
				for _, owner := range allSourceClassScopes() {
					if owner.Class == nil || owner.Class.DeclarationNode == nil {
						continue
					}
					live := classScopeCtx(owner, ctx)
					root := owner.Class.DeclarationNode
					if body := root.ChildByFieldName("body"); body != nil {
						_ = classDeclarationChildCtx(root, body, live)
					}
				}
				return nil
			})
			samples = append(samples, sample{noise, counts.Candidates, api, guard, routing})
		}()
	}
	base := samples[0]
	for _, got := range samples[1:] {
		if got.Candidates != base.Candidates {
			t.Fatal("legal added bodies changed candidate set")
		}
		actualGrowth := got.API.Cgo - base.API.Cgo
		budgetGrowth := (got.Guard.Cgo - base.Guard.Cgo) + (got.Routing.Cgo - base.Routing.Cgo)
		encoded, _ := json.Marshal(map[string]any{"AddedFiles": got.Files, "Queries": queries, "Candidates": got.Candidates, "ActualNativeGrowth": actualGrowth, "MeasuredGuardAndFreshRoutingBudget": budgetGrowth})
		t.Log("DEMAND_SYNTAX_GROWTH", string(encoded))
		if actualGrowth > budgetGrowth {
			t.Errorf("REPEATED_EMPTY_DEMAND_SYNTAX: actual native growth %d exceeds live guard+one fresh root route %d for %d added files", actualGrowth, budgetGrowth, got.Files)
		}
	}
}
