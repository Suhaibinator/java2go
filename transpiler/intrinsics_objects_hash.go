package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// The single Object[] parameter is invoked at fixed arity when the source
// argument widens to that array type. Object casts and primitive arrays instead
// become one element of a new variable-arity Object array.
func objectsHashFixedArray(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	if invocationArgumentCount(invocation) != 1 {
		return false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	actual, known := inferExprJavaType(invocationArgumentNode(invocation, 0), inference, source)
	return known && javaInferenceTypeAssignable(actual, "java.lang.Object[]", ctx)
}

func objectsHashExpected(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	if objectsHashFixedArray(invocation, ctx, source) {
		return []string{"java.lang.Object[]"}
	}
	expected := make([]string, invocationArgumentCount(invocation))
	for index := range expected {
		expected[index] = "java.lang.Object"
	}
	return expected
}

func lowerObjectsHash(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	var values ast.Expr
	if objectsHashFixedArray(invocation, ctx, source) {
		values = args[0]
	} else {
		values = generatedVarargsArrayLiteral("java.lang.Object", args, ctx)
	}
	return stdjavaCall(ctx, "ObjectsHashExecution", executionExpr(ctx), values)
}

func init() {
	registerStaticNodeIntrinsic("Objects", "hash", lowerObjectsHash)
	registerStaticIntrinsicExpectedArguments("Objects", "hash", objectsHashExpected)
	registerStaticIntrinsicResultType("Objects", "hash", "int")
	signature := &symbol.Definition{OriginalName: "hash", OriginalType: "int", IsStatic: true, IsVariadic: true, Parameters: []*symbol.Definition{{OriginalType: "java.lang.Object"}}}
	staticIntrinsicImportSignatures[intrinsicKey{"Objects", "hash"}] = append(staticIntrinsicImportSignatures[intrinsicKey{"Objects", "hash"}], signature)
}
