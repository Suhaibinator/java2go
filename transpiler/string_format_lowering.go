package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// String.format has two overloads and a Java Object... parameter. Fixed arrays
// are selected from static Java types; inspecting a dynamic array would flatten
// an (Object) array or conflate a null Object with a null Object[].
func lowerStringFormatInvocation(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	types := make([]string, len(args))
	for i := range types {
		var known bool
		types[i], known = inferExprJavaType(invocationArgumentNode(invocation, i), ctx, source)
		if !known {
			return unsupportedIntrinsicValue(invocation, "String", source, ctx)
		}
	}
	if result := canonicalStringFormatForTypes(args, types, ctx); result != nil {
		return result
	}
	return unsupportedIntrinsicValue(invocation, "String", source, ctx)
}

func lowerStringFormatReference(args []ast.Expr, ctx Ctx) ast.Expr {
	types, _ := methodReferenceJavaSignature(ctx)
	return canonicalStringFormatForTypes(args, types, ctx)
}

func canonicalStringFormatForTypes(args []ast.Expr, types []string, ctx Ctx) ast.Expr {
	if len(args) == 0 || len(args) != len(types) {
		return nil
	}
	offset := 1
	helper := "JavaStringFormatExecution"
	if builtinFormatLocaleType(types[0], ctx) {
		offset = 2
		helper = "JavaStringFormatLocaleExecution"
	}
	if len(args) < offset {
		return nil
	}
	formatType := types[offset-1]
	if !isBuiltinJavaString(formatType, ctx) && formatType != "null" && formatType != ternaryNullJavaType {
		return nil
	}
	var array ast.Expr
	if len(args) == offset+1 && javaInferenceTypeAssignable(types[offset], "java.lang.Object[]", ctx) {
		array = args[offset]
	} else {
		elements := append([]ast.Expr(nil), args[offset:]...)
		for i := range elements {
			converted, ok := convertJavaValue(elements[i], types[offset+i], "java.lang.Object", ctx)
			if ok {
				elements[i] = converted
			} else if _, primitive := javaPrimitiveType(types[offset+i]); primitive {
				return nil
			}
			// Reference widening needs no runtime conversion. Keeping the
			// original reference also preserves null and array identity.
		}
		array = generatedVarargsArrayLiteral("java.lang.Object", elements, ctx)
	}
	converted := append([]ast.Expr{intrinsicExecutionExpr(ctx)}, args[:offset]...)
	return stdjavaCall(ctx, helper, append(converted, array)...)
}

func builtinFormatLocaleType(javaType string, ctx Ctx) bool {
	javaType = strings.TrimSpace(javaType)
	if javaType != "Locale" && javaType != "java.util.Locale" {
		return false
	}
	if visibleTypeParameterDeclarationForJavaType(javaType, ctx) != nil || resolveClassScopeByQualifiedName(ctx, javaType) != nil {
		return false
	}
	if javaType == "Locale" && ctx.currentFile != nil {
		if owner, found := ctx.currentFile.Imports[javaType]; found && owner != "java.util" {
			return false
		}
	}
	return true
}
