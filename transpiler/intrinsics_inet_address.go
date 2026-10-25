package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

func inetAddressRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(typeArgs) != 0 || len(arguments) != 0 || containsString(typeParams, base) || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || platformTypeOwner(base, nil, ctx, nil) != "java.net.InetAddress" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaInetAddress", ctx)}, true
}
func inetAddressReferenceAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
	actualBase, actualRank := javaArrayTypeParts(actual)
	expectedBase, expectedRank := javaArrayTypeParts(expected)
	actualBase, actualArgs := parseJavaTypeString(strings.TrimSpace(actualBase))
	expectedBase, expectedArgs := parseJavaTypeString(strings.TrimSpace(expectedBase))
	if actualRank != 0 || expectedRank != 0 || len(actualArgs) != 0 || len(expectedArgs) != 0 {
		return false
	}
	for _, base := range []string{actualBase, expectedBase} {
		if containsString(candidateTypeParams, base) || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
			return false
		}
	}
	if platformTypeOwner(actualBase, nil, ctx, nil) != "java.net.InetAddress" {
		return false
	}
	switch qualifyDeclaredNominalReference(expectedBase, ctx) {
	case "java.net.InetAddress", "java.lang.Object", "java.io.Serializable":
		return true
	}
	return false
}
func inetAddressStaticOwned(node *sitter.Node, ctx Ctx, source []byte) bool {
	if object := node.ChildByFieldName("object"); object != nil {
		return platformTypeOwner(object.Content(source), object, ctx, source) == "java.net.InetAddress"
	}
	imported := resolveStaticImportedMethod(node, ctx, source)
	return imported.source == nil && imported.intrinsic == "InetAddress"
}
func inetAddressExceptionConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	owner, known := builtinThrowableReferenceName(className, ctx)
	if !known || owner != "java.net.UnknownHostException" {
		return nil
	}
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewJavaUnknownHostExceptionMessage", ast.NewIdent("nil"))
	}
	argument := nodeutil.SemanticNamedChild(arguments, 0)
	if len(args) != 1 || arguments == nil || !intrinsicInvocationConversionApplicable(argument, "java.lang.String", ctx, source) {
		return unsupportedIntrinsicValue(arguments, "java.net.UnknownHostException constructor", source, ctx)
	}
	return stdjavaCall(ctx, "NewJavaUnknownHostExceptionMessage", coerceArgumentToExpectedType(args[0], argument, "java.lang.String", ctx, source))
}
func init() {
	registerIntrinsicOwner("java.net.InetAddress", true)
	registerIntrinsicOwner("java.net.UnknownHostException", true)
	registerStaticIntrinsicResultType("InetAddress", "getByName", "java.net.InetAddress")
	registerStaticNodeIntrinsic("InetAddress", "getByName", func(node *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		if !inetAddressStaticOwned(node, ctx, source) {
			return nil
		}
		argument := invocationArgumentNode(node, 0)
		if len(args) != 1 || !intrinsicInvocationConversionApplicable(argument, "java.lang.String", ctx, source) {
			return unsupportedIntrinsicValue(node, "java.net.InetAddress", source, ctx)
		}
		return stdjavaCall(ctx, "JavaInetAddressGetByNameExecution", intrinsicExecutionExpr(ctx), coerceArgumentToExpectedType(args[0], argument, "java.lang.String", ctx, source))
	})
	registerInstanceIntrinsicResultType("InetAddress", "getHostAddress", "java.lang.String")
	registerInstanceNodeIntrinsic("InetAddress", "getHostAddress", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if platformReceiverOwner(node.ChildByFieldName("object"), ctx, source) != "java.net.InetAddress" {
			return nil
		}
		if invocationArgumentCount(node) != 0 {
			return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
		}
		return methodCall(receiver, "GetHostAddressJava2goExecution", intrinsicExecutionExpr(ctx))
	})

	for _, method := range []struct {
		name, helper, result string
		arity                int
	}{
		{"toString", "", "java.lang.String", 0},
		{"hashCode", "ObjectHashCodeExecution", "int", 0},
		{"equals", "ObjectEqualsExecution", "boolean", 1},
	} {
		registerInstanceIntrinsicResultType("InetAddress", method.name, method.result)
		registerInstanceNodeIntrinsic("InetAddress", method.name, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if platformReceiverOwner(node.ChildByFieldName("object"), ctx, source) != "java.net.InetAddress" {
				return nil
			}
			if invocationArgumentCount(node) != method.arity {
				return unsupportedIntrinsicValue(node, method.result, source, ctx)
			}
			if method.name == "toString" {
				return methodCall(receiver, "StringJava2goExecution", intrinsicExecutionExpr(ctx))
			}
			args := []ast.Expr{intrinsicExecutionExpr(ctx), receiver}
			if method.arity == 1 {
				argument := invocationArgumentNode(node, 0)
				argumentCtx := ctx.Clone()
				argumentCtx.expectedType, argumentCtx.expectedTypeRoot = "java.lang.Object", argument
				args = append(args, ParseExpr(argument, source, argumentCtx))
			}
			return stdjavaCall(ctx, method.helper, args...)
		})
	}
	unimplementedPlatformConstructors["java.net.InetAddress"] = true
	for _, member := range []struct {
		name, result string
		arity        int
	}{
		{"getHostName", "java.lang.String", 0}, {"getCanonicalHostName", "java.lang.String", 0}, {"getAddress", "byte[]", 0}, {"isAnyLocalAddress", "boolean", 0}, {"isLoopbackAddress", "boolean", 0}, {"isLinkLocalAddress", "boolean", 0}, {"isSiteLocalAddress", "boolean", 0}, {"isMulticastAddress", "boolean", 0}, {"isMCGlobal", "boolean", 0}, {"isMCNodeLocal", "boolean", 0}, {"isMCLinkLocal", "boolean", 0}, {"isMCSiteLocal", "boolean", 0}, {"isMCOrgLocal", "boolean", 0}, {"isReachable", "boolean", 1}, {"isReachable", "boolean", 3},
	} {
		registerInstanceNodeIntrinsic("InetAddress", member.name, func(_ ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if platformReceiverOwner(node.ChildByFieldName("object"), ctx, source) != "java.net.InetAddress" {
				return nil
			}
			return unsupportedIntrinsicValue(node, member.result, source, ctx)
		})
		unimplementedPlatformMembers = append(unimplementedPlatformMembers, platformMemberContract{owner: "java.net.InetAddress", name: member.name, arity: member.arity, result: member.result})
	}
	for _, name := range []string{"getAllByName", "getLocalHost", "getLoopbackAddress", "getByAddress", "ofLiteral"} {
		registerStaticNodeIntrinsic("InetAddress", name, func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if !inetAddressStaticOwned(node, ctx, source) {
				return nil
			}
			return unsupportedIntrinsicValue(node, "java.net.InetAddress", source, ctx)
		})
	}
}
