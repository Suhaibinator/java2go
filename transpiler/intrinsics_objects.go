package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func objectsRequireNonNullResult(invocation *sitter.Node, ctx Ctx, source []byte) string {
	if explicit := invocation.ChildByFieldName("type_arguments"); explicit != nil && explicit.NamedChildCount() == 1 {
		return explicit.NamedChild(0).Content(source)
	}
	if actual, ok := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, source); ok && actual != "null" {
		return intrinsicReferenceJavaType(actual)
	}
	if ctx.expectedType != "" && expectedTypeTargetsExpression(ctx, invocation) {
		return intrinsicReferenceJavaType(ctx.expectedType)
	}
	return "java.lang.Object"
}

func objectsRequireNonNullExpected(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	count := invocationArgumentCount(invocation)
	if count < 1 || count > 2 {
		return nil
	}
	result := []string{objectsRequireNonNullResult(invocation, ctx, source)}
	if count == 2 {
		second := invocationArgumentNode(invocation, 1)
		actual, known := inferExprJavaType(second, ctx, source)
		if known && intrinsicStringBoundType(symbol.JavaType{Original: actual}, ctx, map[typeParameterIdentityKey]bool{}) {
			result = append(result, "java.lang.String")
		} else {
			result = append(result, "java.util.function.Supplier<java.lang.String>")
		}
	}
	return result
}

func lowerObjectsRequireNonNull(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	expected := objectsRequireNonNullExpected(invocation, ctx, source)
	if len(expected) == 0 {
		return nil
	}
	name := "ObjectsRequireNonNull"
	if len(expected) == 2 {
		name += "Message"
		if expected[1] != "java.lang.String" {
			name = "ObjectsRequireNonNullSupplier"
			args = append([]ast.Expr{executionExpr(ctx)}, args...)
		}
	}
	typeArg := javaTypeStringToGoTypeExpr(expected[0], inScopeTypeParameters(ctx), ctx)
	return stdjavaGenericCall(ctx, name, []ast.Expr{typeArg}, args)
}

func init() {
	registerIntrinsicOwner("java.util.Objects", true)
	// Invocation-aware dispatch selects the declared String/Supplier overload.
	registerStaticIntrinsic("Objects", "requireNonNull", func(_ ast.Expr, _ []ast.Expr, _ Ctx) ast.Expr { return nil })
	registerStaticIntrinsicDerivedResultType("Objects", "requireNonNull", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		return objectsRequireNonNullResult(invocation, ctx, source), true
	})
	for _, parameters := range [][]string{{"T"}, {"T", "java.lang.String"}, {"T", "java.util.function.Supplier<java.lang.String>"}} {
		def := &symbol.Definition{OriginalName: "requireNonNull", OriginalType: "T", IsStatic: true, TypeParameters: []symbol.TypeParam{symbol.NewTypeParam("T", nil)}}
		for _, parameter := range parameters {
			def.Parameters = append(def.Parameters, &symbol.Definition{OriginalType: parameter})
		}
		key := intrinsicKey{"Objects", "requireNonNull"}
		staticIntrinsicImportSignatures[key] = append(staticIntrinsicImportSignatures[key], def)
	}
}
