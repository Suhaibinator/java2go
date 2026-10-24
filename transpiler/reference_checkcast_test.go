package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestReferenceCheckCastNativeFallbackProducer(t *testing.T) {
	h := setupParseHelper(t, `class CastProbe {java.lang.StringBuffer cast(Object value){return (java.lang.StringBuffer)value;}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("CastProbe")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("cast")[0]
	node := findNode(ctx.localScope.DeclarationNode, "cast_expression")
	if node == nil {
		t.Fatal("actual cast syntax missing")
	}
	expr := ParseExpr(node, h.File.Source, ctx)
	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	got := rendered.String()
	if !strings.Contains(got, "ReferenceCheckCast[") || !strings.Contains(got, `"java.lang.StringBuffer"`) {
		t.Fatalf("native nominal check absent before shared Go carrier assertion: %s", got)
	}
}

func checkCastRendered(t *testing.T, origin inferredJavaTypeOrigin, ctx Ctx) string {
	t.Helper()
	expr := nullableReferenceAssertion(&ast.CallExpr{Fun: ast.NewIdent("effect")}, ast.NewIdent("any"), origin, ctx)
	var rendered bytes.Buffer
	if err := format.Node(&rendered, token.NewFileSet(), expr); err != nil {
		t.Fatal(err)
	}
	if strings.Count(rendered.String(), "effect()") != 1 {
		t.Fatal("cast duplicated operand")
	}
	return rendered.String()
}

func TestReferenceCheckCastCapturedBoundUsesDeclarationContext(t *testing.T) {
	for _, name := range []string{"Bound", "Runnable"} {
		t.Run(name, func(t *testing.T) {
			h := setupParseHelper(t, strings.ReplaceAll(`package castprobe; class Bound {} class Outer<T extends Bound>{class Inner<Bound>{}}`, "Bound", name))
			outer := h.File.Symbols.FindClassScope("Outer")
			inner := outer.FindClassScope("Inner")
			ctx := h.Ctx.Clone()
			ctx.currentClass = inner
			origin := inferredJavaTypeOrigin{parameter: outer.TypeParameters[0].Declaration, javaType: "T"}
			got := checkCastRendered(t, origin, ctx)
			if !strings.Contains(got, "ReferenceCheckCast[") || !strings.Contains(got, `"castprobe.`+name+`"`) || strings.Contains(got, "ObjectTypeID") {
				t.Fatalf("outer bound recaptured in consumer's shadowed namespace: %s", got)
			}
			origin.arrayRank = 2
			got = checkCastRendered(t, origin, ctx)
			if strings.Count(got, "ArrayTypeID(") != 2 || !strings.Contains(got, `"castprobe.`+name+`"`) {
				t.Fatalf("bound array rank lost: %s", got)
			}

		})
	}
}

func TestReferenceCheckCastUnprovedAndCyclicOriginShield(t *testing.T) {
	if got := checkCastRendered(t, inferredJavaTypeOrigin{javaType: "Missing", unresolved: true}, Ctx{}); strings.Contains(got, "ReferenceCheckCast[") {
		t.Fatal("unproved spelling gained nominal descriptor")
	}
	parameter := symbol.NewTypeParam("T", nil)
	parameter.Bounds = []symbol.JavaType{{Original: "T", TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"T": parameter.Declaration}}}
	ctx := Ctx{syntheticTypeParameters: []symbol.TypeParam{parameter}}
	if got := checkCastRendered(t, inferredJavaTypeOrigin{parameter: parameter.Declaration, javaType: "T"}, ctx); strings.Contains(got, "ReferenceCheckCast[") {
		t.Fatal("cyclic declaration fabricated nominal target")
	}
}

func TestReferenceCheckCastRuntimeAliasSourceAndUnknownShields(t *testing.T) {
	for _, row := range []struct {
		name, source string
		sourceType   bool
	}{
		{"source", `package castprobe; class Runnable{} class CastProbe {Object cast(Object value){return (Runnable)value;}}`, true},
		{"foreign", `import foreign.Runnable; class CastProbe {Object cast(Object value){return (Runnable)value;}}`, false},
		{"unknown", `class CastProbe {Object cast(Object value){return (foreign.Runnable)value;}}`, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = h.File.Symbols.FindClassScope("CastProbe")
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("cast")[0]
			node := findNode(ctx.localScope.DeclarationNode, "cast_expression")
			var rendered bytes.Buffer
			if err := format.Node(&rendered, token.NewFileSet(), ParseExpr(node, h.File.Source, ctx)); err != nil {
				t.Fatal(err)
			}
			got := rendered.String()
			if strings.Contains(got, "RunnableTypeID") {
				t.Fatalf("source/unknown target acquired native identity: %s", got)
			}
			if row.sourceType && !strings.Contains(got, `"castprobe.Runnable"`) {
				t.Fatalf("source nominal descriptor lost: %s", got)
			}
			if !row.sourceType && strings.Contains(got, "ReferenceCheckCast[") {
				t.Fatalf("unproved import/name acquired descriptor: %s", got)
			}
		})
	}
}

func TestReferenceCheckCastCanonicalRunnableTarget(t *testing.T) {
	for _, written := range []string{"Runnable", "java.lang.Runnable"} {
		h := setupParseHelper(t, `class CastProbe {Object cast(Object value){return (`+written+`)value;}}`)
		ctx := h.Ctx.Clone()
		ctx.currentClass = h.File.Symbols.FindClassScope("CastProbe")
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("cast")[0]
		node := findNode(ctx.localScope.DeclarationNode, "cast_expression")
		var rendered bytes.Buffer
		if err := format.Node(&rendered, token.NewFileSet(), ParseExpr(node, h.File.Source, ctx)); err != nil {
			t.Fatal(err)
		}
		got := rendered.String()
		if !strings.Contains(got, "ReferenceCheckCast[") || !strings.Contains(got, `"java.lang.Runnable"`) {
			t.Fatalf("written native target lost canonical declaration: %s", got)
		}
	}
}
