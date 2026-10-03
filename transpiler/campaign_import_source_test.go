package transpiler

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func campaignAssertImportText(t *testing.T, ctx Ctx, owner string) {
	t.Helper()
	wantStatic := []staticMethodImport{{owner: owner + ".Owner", member: "answer"}, {owner: owner + ".Owner", wildcard: true}}
	wantOrdinary := []string{owner}
	if got := staticMethodImports(ctx); !reflect.DeepEqual(got, wantStatic) {
		t.Fatalf("static imports=%v, want %v", got, wantStatic)
	}
	if got := ordinaryTypeOnDemandImports(ctx); !reflect.DeepEqual(got, wantOrdinary) {
		t.Fatalf("ordinary imports=%v, want %v", got, wantOrdinary)
	}
}

func campaignImportControlContext(t *testing.T, text string) Ctx {
	t.Helper()
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Active.java":    text,
		"alpha/Owner.java": "package alpha;public class Owner{public static int answer(){return 11;}}",
		"bravo/Owner.java": "package bravo;public class Owner{public static int answer(){return 19;}}",
		"extra/Owner.java": "package extra;public class Owner{public static int answer(){return 23;}}",
	})
	active := findQualifiedSourceClass("p.Active")
	ctx.currentFile = findFileScopeForClassScope(active)
	ctx.currentClass = active
	return ctx
}

func campaignImportSource(owner string) string {
	return "package p;import " + owner + ".*;import static " + owner + ".Owner.answer;import static " + owner + ".Owner.*;import " + owner + ".*;class Active<T>{class Member<U>{}}"
}

// Replace the parser tree while retaining the graph and FileScope identity.
func campaignReplaceImportTree(t *testing.T, ctx Ctx, text string) {
	t.Helper()
	file := parsing.SourceFile{Name: "Active.java", Source: []byte(text)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	symbols := file.ParseSymbols()
	ctx.currentFile.Source = file.Source
	ctx.currentFile.BaseClass = symbols.BaseClass
	ctx.currentFile.TopLevelClasses = symbols.TopLevelClasses
}

func TestCampaignImportSourceGuards(t *testing.T) {
	t.Run("same-tree-prefix-mutation-and-caller-slice-ownership", func(t *testing.T) {
		ctx := campaignImportControlContext(t, campaignImportSource("alpha"))
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		campaignAssertImportText(t, ctx, "alpha")
		static := staticMethodImports(ctx)
		ordinary := ordinaryTypeOnDemandImports(ctx)
		static[0].owner = "caller.poison"
		ordinary[0] = "caller.poison"
		campaignAssertImportText(t, ctx, "alpha")
		// Equal-length mutation preserves the tree and file identities. Cached text
		// must still follow the current source bytes used by the original walkers.
		copy(ctx.currentFile.Source, strings.ReplaceAll(string(ctx.currentFile.Source), "alpha", "bravo"))
		campaignAssertImportText(t, ctx, "bravo")
	})
	t.Run("same-graph-file-reparsed-and-appended-imports", func(t *testing.T) {
		ctx := campaignImportControlContext(t, "package p;class Active<T>{}")
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		if len(staticMethodImports(ctx)) != 0 || len(ordinaryTypeOnDemandImports(ctx)) != 0 {
			t.Fatal("empty header invented imports")
		}
		campaignReplaceImportTree(t, ctx, campaignImportSource("alpha"))
		campaignAssertImportText(t, ctx, "alpha")
		campaignReplaceImportTree(t, ctx, strings.Replace(campaignImportSource("bravo"), "class Active", "import extra.*;import static extra.Owner.answer;class Active", 1))
		if got := ordinaryTypeOnDemandImports(ctx); !reflect.DeepEqual(got, []string{"bravo", "extra"}) {
			t.Fatalf("reparsed appended ordinary imports=%v", got)
		}
		if got := staticMethodImports(ctx); len(got) != 3 || got[2].owner != "extra.Owner" || got[2].member != "answer" {
			t.Fatalf("reparsed appended static imports=%v", got)
		}
		campaignReplaceImportTree(t, ctx, "package p;class Active<T>{}")
		if len(staticMethodImports(ctx)) != 0 || len(ordinaryTypeOnDemandImports(ctx)) != 0 {
			t.Fatal("removed imports survived reparsing")
		}
	})
	t.Run("separate-graphs-and-standalone-contexts", func(t *testing.T) {
		first := campaignImportControlContext(t, campaignImportSource("alpha"))
		first.callableSubclasses = &callableSubclassSourceInventory{}
		campaignAssertImportText(t, first, "alpha")
		second := campaignImportControlContext(t, campaignImportSource("bravo"))
		second.callableSubclasses = first.callableSubclasses
		campaignAssertImportText(t, second, "bravo")
		standalone := second
		standalone.callableSubclasses = nil
		campaignAssertImportText(t, standalone, "bravo")
		if staticMethodImports(Ctx{}) != nil || ordinaryTypeOnDemandImports(Ctx{}) != nil {
			t.Fatal("missing file invented import facts")
		}
	})
}

func campaignImportBenchmarkContext(b *testing.B, padding int) Ctx {
	b.Helper()
	var text strings.Builder
	text.WriteString("package p;import static alpha.Owner.answer;import static alpha.Owner.*;")
	for i := 0; i < padding; i++ {
		fmt.Fprintf(&text, "import ordinary%03d.*;", i)
	}
	text.WriteString("class Active<T>{class Member<U>{}}")
	file := parsing.SourceFile{Name: "Active.java", Source: []byte(text.String())}
	if err := file.ParseAST(); err != nil {
		b.Fatal(err)
	}
	file.ParseSymbols()
	return Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}}
}

var campaignImportBenchmarkStatic []staticMethodImport
var campaignImportBenchmarkOrdinary []string

func BenchmarkCampaignImportSyntax(b *testing.B) {
	for _, padding := range []int{0, 64, 512} {
		for _, kind := range []string{"static", "ordinary"} {
			b.Run(fmt.Sprintf("%s/header%d", kind, padding), func(b *testing.B) {
				ctx := campaignImportBenchmarkContext(b, padding)
				_ = staticMethodImports(ctx)
				_ = ordinaryTypeOnDemandImports(ctx)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if kind == "static" {
						campaignImportBenchmarkStatic = staticMethodImports(ctx)
					} else {
						campaignImportBenchmarkOrdinary = ordinaryTypeOnDemandImports(ctx)
					}
				}
			})
		}
	}
}

func TestCampaignOrdinaryImportRepeatedSourceQueries(t *testing.T) {
	// The result and relative budget describe repeated source work on one exact
	// header, including the cold query. No exact implementation allocation count.
	ctx := campaignImportControlContext(t, "package p;"+strings.Repeat("import alpha.*;", 256)+"class Active{}")
	var got []string
	measure := func(queries int) float64 {
		return testing.AllocsPerRun(3, func() {
			ctx.callableSubclasses = &callableSubclassSourceInventory{}
			for i := 0; i < queries; i++ {
				got = ordinaryTypeOnDemandImports(ctx)
			}
		})
	}
	cold, repeated := measure(1), measure(32)
	if !reflect.DeepEqual(got, []string{"alpha"}) {
		t.Fatalf("repeated source queries changed imports: %v", got)
	}
	t.Logf("cold=%.0f repeated32=%.0f ratio=%.2f", cold, repeated, repeated/cold)
	if cold == 0 || repeated > 6*cold {
		t.Fatalf("ORDINARY_IMPORT_REPEATED_SOURCE_WALK: repeated32 %.0f >6*cold %.0f", repeated, cold)
	}
}
