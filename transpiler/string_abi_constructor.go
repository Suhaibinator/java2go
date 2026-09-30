package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Constructor overload selection happens before erasure. Native Go text never
// participates in a Java copy constructor, and char[] never crosses an encoder.
func lowerCanonicalStringConstructor(_ []ast.Expr, args []ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	types := make([]string, len(args))
	for index := range args {
		var known bool
		types[index], known = inferExprJavaType(invocationArgumentNode(invocation, index), ctx, source)
		if !known {
			return nil
		}
	}
	return canonicalStringConstructorForTypes(args, types, ctx)
}

// Method references use the selected SAM argument types and share exactly the
// same constructor allocation/encoding decision as an ordinary invocation.
func canonicalStringConstructorForTypes(args []ast.Expr, types []string, ctx Ctx) ast.Expr {
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewJavaStringUTF16", ast.NewIdent("nil"))
	}
	if len(types) != len(args) {
		return nil
	}
	actual := types[0]
	convert := func(index int, expected string) ast.Expr {
		if converted, ok := convertJavaValue(args[index], types[index], expected, ctx); ok {
			return converted
		}
		return args[index]
	}
	switch {
	case isBuiltinJavaString(actual, ctx) && len(args) == 1:
		return stdjavaCall(ctx, "CopyJavaString", args[0])
	case (actual == "StringBuilder" || actual == "java.lang.StringBuilder" || actual == "StringBuffer" || actual == "java.lang.StringBuffer") && len(args) == 1 && resolveClassScopeByQualifiedName(ctx, actual) == nil:
		return methodCall(args[0], "ToJavaString")
	case actual == "char[]":
		if len(args) == 1 {
			return stdjavaCall(ctx, "JavaStringFromChars", args[0])
		}
		if len(args) == 3 {
			return stdjavaCall(ctx, "JavaStringFromCharsRange", args[0], convert(1, "int"), convert(2, "int"))
		}
	case actual == "byte[]":
		helper := "JavaStringFromBytes"
		callArgs := []ast.Expr{args[0]}
		charsetIndex := 1
		if len(args) == 3 || len(args) == 4 {
			helper = "JavaStringFromBytesRange"
			callArgs = append(callArgs, convert(1, "int"), convert(2, "int"))
			charsetIndex = 3
		} else if len(args) != 1 && len(args) != 2 {
			return nil
		}
		charset := stdjavaQualifiedExpr("UTF_8", ctx)
		if charsetIndex < len(args) {
			encodingType := types[charsetIndex]
			if isBuiltinJavaString(encodingType, ctx) {
				helper += "Named"
			}
			charset = args[charsetIndex]
		}
		return stdjavaCall(ctx, helper, append(callArgs, charset)...)
	}
	return nil
}
