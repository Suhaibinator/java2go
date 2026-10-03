package transpiler

import (
	"go/ast"
	"reflect"
	"testing"
)

// The header's declared nominal family and arguments must survive names that
// begin only in the implementing body. The emitted SAM still resolves its
// executable method in that body's declaration context.
func TestSourceNativeFunctionalHeaderDeclarationContexts(t *testing.T) {
	for _, tc := range []struct {
		name, source, family string
		arguments            []string
		bodyFunction         bool
	}{
		{"imported header and body member", `import java.util.function.Function; class Impl implements Function<String,Integer>{public Integer apply(String value){return 1;} static class Function<T,R>{}}`, "Function", []string{"java.lang.String", "java.lang.Integer"}, true},
		{"qualified header", `class Impl implements java.util.function.Function<String,Integer>{public Integer apply(String value){return 1;} static class Function<T,R>{}}`, "Function", []string{"java.lang.String", "java.lang.Integer"}, true},
		{"header arguments exclude body members", `import java.util.function.Function; class Impl implements Function<String,Integer>{public java.lang.Integer apply(java.lang.String value){return 1;} static class Function<T,R>{} static class String{} static class Integer{}}`, "Function", []string{"java.lang.String", "java.lang.Integer"}, true},
		{"inherited source interface", `import java.util.function.Function; interface Provider<U> extends Function<U,Integer>{class Function<T,R>{}} class Impl implements Provider<String>{public Integer apply(String value){return 1;}}`, "Function", []string{"java.lang.String", "java.lang.Integer"}, false},
		{"inherited executable superclass", `import java.util.function.Function; class Base implements Function<String,Integer>{public Integer apply(String value){return 1;} static class Function<T,R>{}} class Impl extends Base{}`, "Function", []string{"java.lang.String", "java.lang.Integer"}, true},
		{"other family with body member", `import java.util.function.BiFunction; class Impl implements BiFunction<String,String,Integer>{public Integer apply(String left,String right){return 1;} static class BiFunction<T,U,R>{}}`, "BiFunction", []string{"java.lang.String", "java.lang.String", "java.lang.Integer"}, false},
		{"primitive family with body member", `import java.util.function.IntUnaryOperator; class Impl implements IntUnaryOperator{public int applyAsInt(int value){return value+1;} static class IntUnaryOperator{}}`, "IntUnaryOperator", nil, false},
		{"source shadow", `interface Function<T,R>{R apply(T value);} class Impl implements Function<String,Integer>{public Integer apply(String value){return 1;}}`, "", nil, false},
		{"shape only", `class Impl{public Integer apply(String value){return 1;} static class Function<T,R>{}}`, "", nil, true},
		{"external same simple name", `class Impl implements external.Function<String,Integer>{public Integer apply(String value){return 1;}}`, "", nil, false},
		{"enclosing source member", `import java.util.function.Function; class Outer{interface Function<T,R>{R apply(T value);} static class Impl implements Function<String,Integer>{public Integer apply(String value){return 1;}}}`, "", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, "package headerprobe; "+tc.source+` class Caller {<Function> void run(){}}`)
			target := h.File.Symbols.FindClassScope("Impl")
			if target == nil {
				t.Fatal("missing implementation declaration")
			}
			ctx := h.Ctx.Clone()
			// Borrow an unrelated consumer with its own same-named method binder.
			// Contract discovery must reenter each original edge owner's header.
			ctx.currentClass = h.File.Symbols.FindClassScope("Caller")
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("missing caller binder control")
			}
			ctx.localScope = methods[0]
			contracts := sourceNativeFunctionalContracts(target, ctx)
			wantCount := 0
			if tc.family != "" {
				wantCount = 1
			}
			if len(contracts) != wantCount {
				t.Fatalf("resolved contracts=%v, want %d declared families", contracts, wantCount)
			}
			if wantCount != 0 {
				c := contracts[0]
				if c.family != tc.family || !reflect.DeepEqual(c.arguments, tc.arguments) {
					t.Fatalf("resolved contract=%v, want %s%v", c, tc.family, tc.arguments)
				}
				if sourceNativeFunctionalViewExpr(target, c, ast.NewIdent("receiver"), classScopeCtx(target, ctx)) == nil {
					t.Fatal("declared nominal edge lacks executable source SAM view")
				}
			}
			wantFunction := []string(nil)
			if tc.family == "Function" {
				wantFunction = tc.arguments
			}
			if got := sourceFunctionContract(target, ctx); !reflect.DeepEqual(got, wantFunction) {
				t.Fatalf("legacy Function bridge contract=%v, want %v", got, wantFunction)
			}
			body := classScopeCtx(target, ctx)
			member := resolveClassScopeByQualifiedName(body, "Function")
			if tc.bodyFunction && member == nil {
				t.Fatal("body-only or enclosing source Function lookup lost its declaration")
			}
		})
	}
}
