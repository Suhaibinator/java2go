package transpiler

import (
	"fmt"
	"slices"
	"testing"
)

func TestMathExtrema180RegisteredPrimitiveDeclarations(t *testing.T) {
	for _, method := range []string{"max", "min"} {
		defs := staticIntrinsicImportSignatures[intrinsicKey{"Math", method}]
		if len(defs) != 4 {
			t.Fatalf("canonical Math.%s requires its four JDK primitive overloads, got %d", method, len(defs))
		}
		seen := map[string]bool{}
		for _, def := range defs {
			parameters := def.OriginalParameterTypes()
			if def.OriginalName != method || !def.IsStatic || len(def.TypeParameters) != 0 || len(parameters) != 2 || parameters[0] != parameters[1] || !slices.Contains([]string{"int", "long", "float", "double"}, parameters[0]) || seen[parameters[0]] {
				t.Fatalf("unexpected actual canonical %s declaration: %+v parameters=%v", method, def, parameters)
			}
			seen[parameters[0]] = true
		}
	}
}

func TestMathExtrema180PrimitiveImportResolution(t *testing.T) {
	for _, method := range []string{"max", "min"} {
		for _, imported := range []string{method, "*"} {
			for _, tc := range []struct{ left, right, width string }{
				{"int", "int", "int"}, {"long", "long", "long"}, {"float", "float", "float"}, {"double", "double", "double"},
				{"long", "int", "long"}, {"int", "long", "long"}, {"long", "float", "float"}, {"float", "long", "float"},
				{"float", "double", "double"}, {"double", "float", "double"}, {"byte", "short", "int"}, {"char", "int", "int"},
			} {
				t.Run(method+"/"+imported+"/"+tc.left+"/"+tc.right, func(t *testing.T) {
					source := fmt.Sprintf("import static java.lang.Math.%s; class Use { %s call(%s a, %s b) { return %s(a,b); } }", imported, tc.width, tc.left, tc.right, method)
					helper := setupParseHelper(t, source)
					ctx := helper.Ctx.Clone()
					ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
					invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
					got := resolveStaticImportedMethod(invocation, ctx, []byte(source))
					if got.problem != "" || got.intrinsic != "Math" || got.source != nil {
						t.Fatalf("valid primitive invocation must resolve canonically: %+v", got)
					}
					if typ, known := inferExprJavaType(invocation, ctx, []byte(source)); !known || typ != tc.width {
						t.Fatalf("Java overload width = %s, inferred %q/%v", tc.width, typ, known)
					}
				})
			}
		}
	}
}

func TestMathExtrema180InvalidPrimitiveImports(t *testing.T) {
	for _, method := range []string{"max", "min"} {
		for _, expression := range []string{method + "(true,false)", method + "(\"a\",\"b\")", method + "(null,null)", method + "(1)", method + "(1,2,3)"} {
			t.Run(expression, func(t *testing.T) {
				source := "import static java.lang.Math.*; class Use { void call() { " + expression + "; } }"
				helper := setupParseHelper(t, source)
				ctx := helper.Ctx.Clone()
				ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
				invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
				if got := resolveStaticImportedMethod(invocation, ctx, []byte(source)); got.intrinsic != "" || got.source != nil || got.problem != "no applicable static method import "+method {
					t.Fatalf("invalid primitive invocation must remain rejected: %+v", got)
				}
			})
		}
	}
}
