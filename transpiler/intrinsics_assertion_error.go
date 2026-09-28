package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	registerIntrinsicOwner("java.lang.AssertionError", true)
	// Throwable message accessors return nullable Java String references, not an
	// unknown Object; this matters when the caller compares the result with null.
	registerInstanceIntrinsicResultType("Throwable", "getMessage", "String")
	registerInstanceIntrinsicResultType("Throwable", "getLocalizedMessage", "String")
	for name := range builtinExceptionTypes {
		registerInstanceIntrinsicResultType(name, "getMessage", "String")
		registerInstanceIntrinsicResultType(name, "getLocalizedMessage", "String")
	}
}

func tryAssertionErrorConstructor(node *sitter.Node, className string, ctx Ctx, source []byte) (ast.Expr, bool) {
	return assertionErrorConstructorArguments(className, node.ChildByFieldName("arguments"), nil, ctx, source)
}

// The original argument nodes select Java overloads; pre-parsed arguments let
// superclass construction reuse its expressions without parsing them twice.
func assertionErrorConstructorArguments(className string, arguments *sitter.Node, parsed []ast.Expr, ctx Ctx, source []byte) (ast.Expr, bool) {
	if stripJavaQualifier(className) != "AssertionError" {
		return nil, false
	}
	owner, known := canonicalIntrinsicOwner(className, ctx)
	if !known {
		return nil, false
	}
	if owner != "java.lang.AssertionError" {
		return unsupportedIntrinsicOwnerValue(owner, arguments, source, ctx), true
	}
	count := 0
	if arguments != nil {
		count = int(arguments.NamedChildCount())
	}
	expected := []string{}
	helper := "NewAssertionErrorExecution"
	switch count {
	case 0:
	case 1:
		actual, known := inferExprJavaType(arguments.NamedChild(0), ctx, source)
		if !known {
			actual = "Object"
		}
		switch actual {
		case "char":
			helper = "NewAssertionErrorCharExecution"
			expected = []string{"char"}
		case "byte", "short", "int":
			expected = []string{"int"}
		case "boolean", "long", "float", "double":
			expected = []string{actual}
		default:
			expected = []string{"Object"}
		}
	case 2:
		expected = []string{"String", "Throwable"}
	default:
		return nil, false
	}
	if parsed == nil {
		parsed = parseArgumentListWithExpectedTypes(arguments, source, ctx, expected)
	} else {
		converted := make([]ast.Expr, len(parsed))
		for index, value := range parsed {
			converted[index] = coerceArgumentToExpectedType(value, arguments.NamedChild(index), expected[index], ctx, source)
		}
		parsed = converted
	}
	return stdjavaCall(ctx, helper, append([]ast.Expr{intrinsicExecutionExpr(ctx)}, parsed...)...), true
}
