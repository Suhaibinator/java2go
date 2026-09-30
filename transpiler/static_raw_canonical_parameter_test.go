package transpiler

import (
	"testing"
)

func TestStaticRawCanonicalParametersKeepReadableViewsWithoutPhantomBinders(t *testing.T) {
	cases := []struct{ name, declaration, formal, want string }{
		{"raw", "T", "Cell cell", "Cell<Object>"},
		{"unbounded", "T", "Cell<?> cell", "Cell<Object>"},
		{"upper", "T", "Cell<? extends Number> cell", "Cell<Number>"},
		{"lower", "T", "Cell<? super Integer> cell", "Cell<Object>"},
		{"boundedRaw", "T extends Number", "Cell cell", "Cell<Number>"},
		{"dependentRaw", "T extends Number, U extends T", "Cell cell", "Cell<Number, Number>"},
		{"dependentWildcard", "T extends Number, U extends T", "Cell<Integer, ?> cell", "Cell<Integer, Integer>"},
		{"rawArray", "T", "Cell[] cell", "Cell<Object>[]"},
		{"wildcardArray", "T", "Cell<? extends Number>[][] cell", "Cell<Number>[][]"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, "class Cell<"+test.declaration+"> { T value; } class Probe { static void inspect("+test.formal+") {} }")
			target := helper.File.Symbols.FindClassScope("Cell")
			owner := helper.File.Symbols.FindClassScope("Probe")
			ctx := classScopeCtx(owner, helper.Ctx)
			if !canonicalGenericClass(target, classScopeCtx(target, ctx)) {
				t.Fatal("fixture must have an admitted canonical physical layout")
			}
			methods := owner.FindMethod().ByOriginalName("inspect")
			if len(methods) != 1 {
				t.Fatal("inspect method was not resolved")
			}
			synthetic, rewritten := synthesizeRawGenericFunctionParameters(methods[0], ctx)
			if len(synthetic) != 0 {
				t.Fatalf("nongeneric physical alias retained uninferable binders: %#v", synthetic)
			}
			if got := rewritten["cell"]; got != test.want {
				t.Fatalf("readable Java projection = %q, want %q", got, test.want)
			}
		})
	}
}

func TestStaticCanonicalParametersPreserveSourceMethodBinder(t *testing.T) {
	helper := setupParseHelper(t, "class Cell<T> { T value; } class Probe { static <T> T inspect(Cell<T> cell, T value) { return value; } static void demand(Cell raw) {} }")
	owner := helper.File.Symbols.FindClassScope("Probe")
	ctx := classScopeCtx(owner, helper.Ctx)
	methods := owner.FindMethod().ByOriginalName("inspect")
	synthetic, rewritten := synthesizeRawGenericFunctionParameters(methods[0], ctx)
	if len(synthetic) != 0 || len(rewritten) != 0 {
		t.Fatalf("explicit source method binder was rewritten: %#v, %#v", synthetic, rewritten)
	}
	if len(methods[0].TypeParameters) != 1 || methods[0].Parameters[1].DirectTypeParameter != methods[0].TypeParameters[0].Declaration {
		t.Fatal("source method binder identity changed")
	}
}

func TestStaticNoncanonicalWildcardParametersRetainInferableBinders(t *testing.T) {
	helper := setupParseHelper(t, "interface Root { int code(); } interface Extra {} class Cell<T extends Root & Extra> { T value; } class Probe { static void inspect(Cell<?> cell) {} }")
	owner := helper.File.Symbols.FindClassScope("Probe")
	ctx := classScopeCtx(owner, helper.Ctx)
	target := helper.File.Symbols.FindClassScope("Cell")
	if canonicalGenericClass(target, classScopeCtx(target, ctx)) {
		t.Fatal("intersection-bound fixture must retain its generic physical layout")
	}
	methods := owner.FindMethod().ByOriginalName("inspect")
	synthetic, rewritten := synthesizeRawGenericFunctionParameters(methods[0], ctx)
	if len(synthetic) != 1 || rewritten["cell"] != "Cell<CellT>" {
		t.Fatalf("noncanonical wildcard lost its inferable binder: %#v, %#v", synthetic, rewritten)
	}
}
