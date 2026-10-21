package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestRawFunctionTargetDeclarationAndLifetime(t *testing.T) {
	h := setupParseHelper(t, `package proof;import java.util.function.Function;class Object{}class java{static class lang{static class Object{}}}class Use<X>{void body(){Function value=key->key;}}`)
	owner := h.File.Symbols.FindClassScope("Use")
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = owner.FindMethodByName("body", nil)
	ctx.expectedType = "Function"
	root := findNode(h.File.Ast, "lambda_expression")
	ctx.expectedTypeRoot = root
	plan := rawCanonicalFunctionLambdaTarget("Function", ctx)
	if plan == nil {
		t.Fatal("canonical imported raw lambda rejected")
	}
	for _, typ := range []string{"Function[]", "Function<String,String>", "Function<?> ", "other.Function", "java.util.Function"} {
		if rawCanonicalFunctionTarget(typ, ctx) != nil {
			t.Fatalf("foreign/array/parameterized target admitted: %s", typ)
		}
	}
	physical, ok := functionRuntimeTypeExpr("Function", nil, nil, ctx)
	if !ok {
		t.Fatal("missing raw carrier")
	}
	args := physical.(*ast.IndexListExpr).Indices
	for _, arg := range args {
		if id, ok := arg.(*ast.Ident); !ok || id.Name != "any" {
			t.Fatalf("implicit Object acquired source view: %#v", arg)
		}
	}
	explicit, ok := functionRuntimeTypeExpr("Function", []string{"Object", "Object"}, nil, ctx)
	if !ok {
		t.Fatal("explicit arguments rejected")
	}
	for _, arg := range explicit.(*ast.IndexListExpr).Indices {
		if _, ok := arg.(*ast.StarExpr); !ok {
			t.Fatalf("written source Object erased: %#v", arg)
		}
	}
	body := contextWithLambdaParameters(ctx, root.ChildByFieldName("parameters"), []string{"java.lang.Object"}, "java.lang.Object", h.File.Source)
	body.rawFunctionTarget = plan.body(body)
	if !body.rawFunctionTarget.active(body.Clone()) {
		t.Fatal("clone lost same-scope plan")
	}
	copied := *body.localScope
	foreign := body.Clone()
	foreign.localScope = &copied
	if foreign.rawFunctionTarget.active(foreign) {
		t.Fatal("copied body acquired plan")
	}
	foreign = body.Clone()
	foreign.currentClass = h.File.Symbols.FindClassScope("Object")
	if foreign.rawFunctionTarget.active(foreign) {
		t.Fatal("sibling owner acquired plan")
	}
	foreign = body.Clone()
	foreign.currentFile = &symbol.FileScope{}
	if foreign.rawFunctionTarget.active(foreign) {
		t.Fatal("foreign file acquired plan")
	}
	typed := body.Clone()
	typed.expectedType = "Function<String,String>"
	if rawCanonicalFunctionLambdaTarget(typed.expectedType, typed) != nil {
		t.Fatal("nested typed target acquired raw plan")
	}
	if rawCanonicalFunctionTarget("java.util.function.Function", ctx) != nil {
		t.Fatal("source java qualifier borrowed external declaration")
	}
}
func TestRawFunctionTargetSourceAndBinderRefusal(t *testing.T) {
	for _, source := range []string{`import java.util.function.*;interface Function{Object apply(Object key);}class Use{}`, `import java.util.function.Function;class Use<Function>{}`, `import foreign.Function;class Use{}`} {
		h := setupParseHelper(t, source)
		ctx := classScopeCtx(h.File.Symbols.FindClassScope("Use"), h.Ctx)
		if rawCanonicalFunctionTarget("Function", ctx) != nil {
			t.Fatal("source/binder/foreign declaration admitted")
		}
	}
}

func TestRawFunctionTargetExplicitParameterRefusal(t *testing.T) {
	for _, source := range []string{`import java.util.function.Function;class Object{}class Use{void body(){Function value=(Object key)->key;}}`, `import java.util.function.Function;class Use<Object>{void body(){Function value=(Object key)->key;}}`, `import java.util.function.Function;class Use{void body(){Function value=(Object key[])->key;}}`} {
		h := setupParseHelper(t, source)
		ctx := classScopeCtx(h.File.Symbols.FindClassScope("Use"), h.Ctx)
		ctx.localScope = ctx.currentClass.FindMethodByName("body", nil)
		ctx.expectedType = "Function"
		ctx.expectedTypeRoot = findNode(h.File.Ast, "lambda_expression")
		if rawCanonicalFunctionLambdaTarget("Function", ctx) != nil {
			t.Fatal("noncanonical explicit parameter acquired raw SAM")
		}
	}
}
