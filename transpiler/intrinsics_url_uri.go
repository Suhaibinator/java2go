package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"go/ast"
	"go/token"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Only canonical URI declarations enter this slice. URL remains unimplemented.
func uriRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(typeArgs) != 0 || len(arguments) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return nil, false
	}
	for _, parameter := range typeParams {
		if base == parameter {
			return nil, false
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known || owner != "java.net.URI" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaURI", ctx)}, true
}

func uriReferenceAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
	actualElement, actualRank := javaArrayTypeParts(actual)
	expectedElement, expectedRank := javaArrayTypeParts(expected)
	actualBase, actualArgs := parseJavaTypeString(strings.TrimSpace(actualElement))
	expectedBase, expectedArgs := parseJavaTypeString(strings.TrimSpace(expectedElement))
	if actualRank != 0 || expectedRank != 0 || len(actualArgs) != 0 {
		return false
	}
	for _, base := range []string{actualBase, expectedBase} {
		if visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
			return false
		}
		for _, parameter := range candidateTypeParams {
			if base == parameter {
				return false
			}
		}
	}
	owner, known := canonicalIntrinsicOwner(actualBase, ctx)
	if !known || owner != "java.net.URI" {
		return false
	}
	expectedOwner := qualifyDeclaredNominalReference(expectedBase, ctx)
	switch expectedOwner {
	case "java.net.URI", "java.lang.Object", "java.io.Serializable":
		return len(expectedArgs) == 0
	case "java.lang.Comparable":
		if len(expectedArgs) == 0 {
			return true
		}
		if len(expectedArgs) != 1 {
			return false
		}
		argument := strings.TrimSpace(expectedArgs[0])
		if argument == "?" {
			return true
		}
		if bound, ok := strings.CutPrefix(argument, "? extends "); ok {
			return uriReferenceAssignable("java.net.URI", bound, candidateTypeParams, ctx)
		}
		if bound, ok := strings.CutPrefix(argument, "? super "); ok {
			return uriReferenceAssignable(bound, "java.net.URI", candidateTypeParams, ctx)
		}
		if containsString(candidateTypeParams, argument) {
			return true
		}
		return uriReferenceAssignable(argument, "java.net.URI", nil, ctx)
	}
	return false
}

// The exception inventory recognizes this type but its Java declarations have
// two or three parameters, rather than Throwable's ordinary message overloads.
func uriSyntaxExceptionConstructorExpr(className string, arguments *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	owner, known := builtinThrowableReferenceName(className, ctx)
	if !known || owner != "java.net.URISyntaxException" {
		return nil
	}
	parameters := []string{"java.lang.String", "java.lang.String"}
	if len(args) == 3 {
		parameters = append(parameters, "int")
	}
	if len(args) != len(parameters) || arguments == nil {
		return unsupportedIntrinsicValue(arguments, "java.net.URISyntaxException constructor", source, ctx)
	}
	for index, parameter := range parameters {
		argument := nodeutil.SemanticNamedChild(arguments, index)
		if argument == nil || !intrinsicInvocationConversionApplicable(argument, parameter, ctx, source) {
			return unsupportedIntrinsicValue(arguments, "java.net.URISyntaxException constructor", source, ctx)
		}
		args[index] = coerceArgumentToExpectedType(args[index], argument, parameter, ctx, source)
	}
	if len(args) == 2 {
		args = append(args, &ast.UnaryExpr{Op: token.SUB, X: &ast.BasicLit{Kind: token.INT, Value: "1"}})
	}
	return stdjavaCall(ctx, "NewJavaURISyntaxException", args...)
}

func init() {
	registerIntrinsicOwner("java.net.URI", true)
	registerIntrinsicOwner("java.net.URISyntaxException", true)
	registerConstructorNodeIntrinsic("URI", func(typeArgs []ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if len(typeArgs) != 0 || len(args) != 1 || !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, 0), "java.lang.String", ctx, source) {
			return unsupportedIntrinsicValue(node, "java.net.URI constructor", source, ctx)
		}
		text := coerceArgumentToExpectedType(args[0], invocationArgumentNode(node, 0), "java.lang.String", ctx, source)
		return stdjavaCall(ctx, "NewJavaURIStringExecution", intrinsicExecutionExpr(ctx), text)
	})
	for _, method := range []struct{ java, goName string }{{"toString", "StringJava2goExecution"}, {"toASCIIString", "ToASCIIStringJava2goExecution"}} {
		registerInstanceIntrinsicResultType("URI", method.java, "java.lang.String")
		registerInstanceNodeIntrinsic("URI", method.java, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != 0 {
				return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
			}
			return methodCall(receiver, method.goName, intrinsicExecutionExpr(ctx))
		})
	}
	for _, method := range []struct{ java, helper, result string }{{"getInput", "URISyntaxExceptionInput", "java.lang.String"}, {"getReason", "URISyntaxExceptionReason", "java.lang.String"}, {"getIndex", "URISyntaxExceptionIndex", "int"}} {
		registerInstanceIntrinsicResultType("URISyntaxException", method.java, method.result)
		registerInstanceNodeIntrinsic("URISyntaxException", method.java, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != 0 {
				return unsupportedIntrinsicValue(node, method.result, source, ctx)
			}
			return stdjavaCall(ctx, method.helper, receiver)
		})
	}
}
