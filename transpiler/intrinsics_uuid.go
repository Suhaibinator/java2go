package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func uuidRuntimeTypeExpr(javaType string, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	base, args := parseJavaTypeString(strings.TrimSpace(javaType))
	if stripJavaQualifier(base) != "UUID" || len(args) != 0 {
		return nil, false
	}
	if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return nil, false
	}
	for _, parameter := range typeParams {
		if base == parameter {
			return nil, false
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known || owner != "java.util.UUID" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaUUID", ctx)}, true
}
func uuidArgumentsApplicable(node *sitter.Node, parameters []string, ctx Ctx, source []byte) bool {
	if invocationArgumentCount(node) != len(parameters) {
		return false
	}
	for i, parameter := range parameters {
		if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, i), parameter, ctx, source) {
			return false
		}
	}
	return true
}
func init() {
	registerIntrinsicOwner("java.util.UUID", true)
	registerConstructorNodeIntrinsic("UUID", func(typeArgs []ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		parameters := []string{"long", "long"}
		if len(typeArgs) != 0 || len(args) != 2 || !uuidArgumentsApplicable(node, parameters, ctx, source) {
			return unsupportedIntrinsicValue(node, "java.util.UUID", source, ctx)
		}
		for i := range args {
			args[i] = coerceArgumentToExpectedType(args[i], invocationArgumentNode(node, i), parameters[i], ctx, source)
		}
		return stdjavaCall(ctx, "NewJavaUUID", args...)
	})
	for _, factory := range []struct{ method, parameter, helper string }{{"fromString", "java.lang.String", "UUIDFromStringJavaString"}, {"nameUUIDFromBytes", "byte[]", "UUIDNameFromBytes"}} {
		registerStaticIntrinsicImportSignature("UUID", factory.method, factory.parameter)
		registerStaticIntrinsicResultType("UUID", factory.method, "java.util.UUID")
		registerStaticIntrinsicExpectedArguments("UUID", factory.method, func(node *sitter.Node, ctx Ctx, source []byte) []string {
			if uuidArgumentsApplicable(node, []string{factory.parameter}, ctx, source) {
				return []string{factory.parameter}
			}
			return nil
		})
		registerStaticNodeIntrinsic("UUID", factory.method, func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if len(args) != 1 || !uuidArgumentsApplicable(node, []string{factory.parameter}, ctx, source) {
				return unsupportedIntrinsicValue(node, "java.util.UUID", source, ctx)
			}
			arg := coerceArgumentToExpectedType(args[0], invocationArgumentNode(node, 0), factory.parameter, ctx, source)
			return stdjavaCall(ctx, factory.helper, arg)
		})
	}
	for _, method := range []struct {
		java, goName, result string
		parameters           []string
	}{
		{"getMostSignificantBits", "GetMostSignificantBits", "long", nil},
		{"getLeastSignificantBits", "GetLeastSignificantBits", "long", nil},
		{"hashCode", "HashCode", "int", nil},
		{"equals", "Equals", "boolean", []string{"java.lang.Object"}},
		{"compareTo", "CompareTo", "int", []string{"java.util.UUID"}},
		{"toString", "StringJava2goExecution", "java.lang.String", nil},
	} {
		registerInstanceIntrinsicResultType("UUID", method.java, method.result)
		registerInstanceNodeIntrinsic("UUID", method.java, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !uuidArgumentsApplicable(node, method.parameters, ctx, source) {
				return unsupportedIntrinsicValue(node, method.result, source, ctx)
			}
			args := make([]ast.Expr, 0, len(method.parameters))
			for i, parameter := range method.parameters {
				argument := invocationArgumentNode(node, i)
				argumentCtx := ctx.Clone()
				argumentCtx.expectedType = parameter
				argumentCtx.expectedTypeRoot = argument
				arg := ParseExpr(argument, source, argumentCtx)
				args = append(args, coerceArgumentToExpectedType(arg, argument, parameter, ctx, source))
			}
			if method.java == "toString" {
				args = append(args, intrinsicExecutionExpr(ctx))
			}
			return methodCall(recv, method.goName, args...)
		})
	}
	registerStaticIntrinsicImportSignature("UUID", "randomUUID")
	registerStaticIntrinsicResultType("UUID", "randomUUID", "java.util.UUID")
	registerStaticNodeIntrinsic("UUID", "randomUUID", func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		return unsupportedIntrinsicValue(node, "java.util.UUID", source, ctx)
	})
	for _, method := range []struct{ name, result string }{{"version", "int"}, {"variant", "int"}, {"timestamp", "long"}, {"clockSequence", "int"}, {"node", "long"}} {
		registerInstanceIntrinsicResultType("UUID", method.name, method.result)
		registerInstanceNodeIntrinsic("UUID", method.name, func(_ ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			return unsupportedIntrinsicValue(node, method.result, source, ctx)
		})
	}
}
