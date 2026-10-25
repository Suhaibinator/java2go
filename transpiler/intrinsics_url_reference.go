package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func urlRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(typeArgs) != 0 || len(arguments) != 0 || containsString(typeParams, base) || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || platformTypeOwner(base, nil, ctx, nil) != "java.net.URL" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaURL", ctx)}, true
}

func urlReferenceAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
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
	if platformTypeOwner(actualBase, nil, ctx, nil) != "java.net.URL" {
		return false
	}
	switch qualifyDeclaredNominalReference(expectedBase, ctx) {
	case "java.net.URL", "java.lang.Object", "java.io.Serializable":
		return true
	}
	return false
}

func urlConstructorNodeExpr(typeArgs []ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if node == nil || node.ChildByFieldName("type") == nil || platformTypeOwner(node.ChildByFieldName("type").Content(source), node, ctx, source) != "java.net.URL" {
		return nil
	}
	argument := invocationArgumentNode(node, 0)
	if len(typeArgs) != 0 || len(args) != 1 || !intrinsicInvocationConversionApplicable(argument, "java.lang.String", ctx, source) {
		return unsupportedIntrinsicValue(node, "java.net.URL constructor", source, ctx)
	}
	text := coerceArgumentToExpectedType(args[0], argument, "java.lang.String", ctx, source)
	return stdjavaCall(ctx, "NewJavaURLStringExecution", intrinsicExecutionExpr(ctx), text)
}

// A freshly constructed reference has no parameter/field declaration from
// which the general platform receiver helper can recover an import namespace.
// Prove its written type at that actual creation; never infer identity from a
// display name. All other expressions retain the existing declaration proof.
func urlReceiverOwned(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(node) == 1 {
		return urlReceiverOwned(nodeutil.SemanticNamedChild(node, 0), ctx, source)
	}
	if node.Type() == "object_creation_expression" {
		typ := node.ChildByFieldName("type")
		return typ != nil && platformTypeOwner(typ.Content(source), node, ctx, source) == "java.net.URL"
	}
	return platformReceiverOwner(node, ctx, source) == "java.net.URL"
}

func urlExceptionConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	owner, known := builtinThrowableReferenceName(className, ctx)
	if !known || owner != "java.net.MalformedURLException" {
		return nil
	}
	if len(args) == 0 {
		return stdjavaCall(ctx, "NewJavaMalformedURLExceptionMessage", ast.NewIdent("nil"))
	}
	argument := nodeutil.SemanticNamedChild(arguments, 0)
	if len(args) != 1 || arguments == nil || !intrinsicInvocationConversionApplicable(argument, "java.lang.String", ctx, source) {
		return unsupportedIntrinsicValue(arguments, "java.net.MalformedURLException constructor", source, ctx)
	}
	return stdjavaCall(ctx, "NewJavaMalformedURLExceptionMessage", coerceArgumentToExpectedType(args[0], argument, "java.lang.String", ctx, source))
}

func init() {
	// Whole Properties replacement remains an unimplemented platform service.
	// Individual String properties use the separate canonical property service.
	for _, configuration := range []struct {
		name, result string
		arity        int
	}{{"setProperties", "void", 1}} {
		registerStaticNodeIntrinsic("System", configuration.name, func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if object := node.ChildByFieldName("object"); object != nil {
				if platformTypeOwner(object.Content(source), object, ctx, source) != "java.lang.System" {
					return nil
				}
			} else if imported := resolveStaticImportedMethod(node, ctx, source); imported.source != nil || imported.intrinsic != "System" {
				return nil
			}
			return unsupportedIntrinsicValue(node, configuration.result, source, ctx)
		})
		unimplementedPlatformMembers = append(unimplementedPlatformMembers, platformMemberContract{owner: "java.lang.System", name: configuration.name, arity: configuration.arity, result: configuration.result})
	}
	registerIntrinsicOwner("java.net.URL", true)
	registerIntrinsicOwner("java.net.MalformedURLException", true)
	registerConstructorNodeIntrinsic("URL", urlConstructorNodeExpr)
	for _, method := range []struct{ java, goName string }{{"toString", "StringJava2goExecution"}, {"toExternalForm", "ToExternalFormJava2goExecution"}} {
		registerInstanceIntrinsicResultType("URL", method.java, "java.lang.String")
		registerInstanceNodeIntrinsic("URL", method.java, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !urlReceiverOwned(node.ChildByFieldName("object"), ctx, source) {
				return nil
			}
			if invocationArgumentCount(node) != 0 {
				return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
			}
			return methodCall(receiver, method.goName, intrinsicExecutionExpr(ctx))
		})
	}
	// Refusal records cover unadmitted public URL declarations independently
	// of the service's native representation.
	for _, member := range []struct{ name, result string }{
		{"getQuery", "java.lang.String"}, {"getPath", "java.lang.String"}, {"getUserInfo", "java.lang.String"}, {"getAuthority", "java.lang.String"}, {"getPort", "int"}, {"getDefaultPort", "int"}, {"getProtocol", "java.lang.String"}, {"getHost", "java.lang.String"}, {"getFile", "java.lang.String"}, {"getRef", "java.lang.String"},
		{"equals", "boolean"}, {"hashCode", "int"}, {"sameFile", "boolean"}, {"toURI", "java.net.URI"}, {"openConnection", "java.net.URLConnection"}, {"openStream", "java.io.InputStream"}, {"getContent", "java.lang.Object"}, {"setURLStreamHandlerFactory", "void"},
	} {
		registerInstanceNodeIntrinsic("URL", member.name, func(_ ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !urlReceiverOwned(node.ChildByFieldName("object"), ctx, source) {
				return nil
			}
			return unsupportedIntrinsicValue(node, member.result, source, ctx)
		})
		arities := []int{0}
		switch member.name {
		case "equals", "sameFile", "setURLStreamHandlerFactory":
			arities = []int{1}
		case "openConnection", "getContent":
			arities = []int{0, 1}
		}
		for _, arity := range arities {
			unimplementedPlatformMembers = append(unimplementedPlatformMembers, platformMemberContract{owner: "java.net.URL", name: member.name, arity: arity, result: member.result})
		}
	}
	for _, method := range []string{"of", "setURLStreamHandlerFactory"} {
		registerStaticNodeIntrinsic("URL", method, func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
			if object := node.ChildByFieldName("object"); object != nil {
				if platformTypeOwner(object.Content(source), object, ctx, source) != "java.net.URL" {
					return nil
				}
			} else if imported := resolveStaticImportedMethod(node, ctx, source); imported.source != nil || imported.intrinsic != "URL" {
				return nil
			}
			return unsupportedIntrinsicValue(node, "java.net.URL static service", source, ctx)
		})
	}
}
