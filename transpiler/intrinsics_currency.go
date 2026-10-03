package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func currencyRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
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
	if !known || owner != "java.util.Currency" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaCurrency", ctx)}, true
}

// This service contributes only Currency's declared nominal edges. Arrays,
// parameterized spellings, source declarations, and binders keep their own
// assignability rules; sharing a simple name does not identify a JDK type.
func currencyReferenceAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
	bases := make([]string, 2)
	for index, javaType := range []string{actual, expected} {
		element, rank := javaArrayTypeParts(javaType)
		base, arguments := parseJavaTypeString(strings.TrimSpace(element))
		if rank != 0 || len(arguments) != 0 || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
			return false
		}
		for _, parameter := range candidateTypeParams {
			if base == parameter {
				return false
			}
		}
		bases[index] = base
	}
	owner, known := canonicalIntrinsicOwner(bases[0], ctx)
	if !known || owner != "java.util.Currency" {
		return false
	}
	if overrideBridgeCanonicalObjectResult(expected, bases[1], ctx) {
		return true
	}
	if owner, known := canonicalIntrinsicOwner(bases[1], ctx); known && owner == "java.util.Currency" {
		return true
	}
	if bases[1] == "java.io.Serializable" {
		return true
	}
	if bases[1] != "Serializable" || ctx.currentFile == nil {
		return false
	}
	if imported, explicit := ctx.currentFile.Imports[bases[1]]; explicit {
		return imported == "java.io"
	}
	for _, imported := range intrinsicOnDemandImports(ctx) {
		if imported == "java.io" {
			return true
		}
	}
	return false
}

func currencyStringInvocationApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if invocationArgumentCount(node) != 1 {
		return false
	}
	argument := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	// getInstance(Locale) remains outside the service. Its real presence makes
	// a bare null ambiguous; an explicitly typed String null is applicable.
	if argument == nil || argument.Type() == "null_literal" {
		return false
	}
	return intrinsicInvocationConversionApplicable(argument, "java.lang.String", ctx, source)
}

func init() {
	registerIntrinsicOwner("java.util.Currency", true)
	registerStaticIntrinsicImportSignature("Currency", "getInstance", "java.lang.String")
	registerStaticIntrinsicResultType("Currency", "getInstance", "java.util.Currency")
	registerStaticIntrinsicExpectedArguments("Currency", "getInstance", func(node *sitter.Node, ctx Ctx, source []byte) []string {
		if currencyStringInvocationApplicable(node, ctx, source) {
			return []string{"java.lang.String"}
		}
		return nil
	})
	registerStaticNodeIntrinsic("Currency", "getInstance", func(node *sitter.Node, arguments []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		if len(arguments) != 1 || !currencyStringInvocationApplicable(node, ctx, source) {
			return unsupportedIntrinsicValue(node, "java.util.Currency", source, ctx)
		}
		argument := coerceArgumentToExpectedType(arguments[0], invocationArgumentNode(node, 0), "java.lang.String", ctx, source)
		return stdjavaCall(ctx, "CurrencyGetInstanceJavaString", argument)
	})
	registerInstanceNodeIntrinsic("Currency", "getInstance", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if !currencyStringInvocationApplicable(node, ctx, source) {
			return unsupportedIntrinsicValue(node, "java.util.Currency", source, ctx)
		}
		arguments := parseTypedIntrinsicInvocationArguments(node, node.ChildByFieldName("object"), "Currency", "getInstance", source, ctx)
		return &ast.CallExpr{Fun: &ast.FuncLit{
			Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr("java.util.Currency", nil, ctx)}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{receiver}},
				&ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, "CurrencyGetInstanceJavaString", arguments[0])}},
			}},
		}}
	})
	registerInstanceIntrinsicResultType("Currency", "getInstance", "java.util.Currency")
	// Currency inherits Object's identity methods; it declares neither value
	// equality nor a content hash. Use the existing Object service directly.
	registerInstanceIntrinsicResultType("Currency", "hashCode", "int")
	registerInstanceNodeIntrinsic("Currency", "hashCode", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if invocationArgumentCount(node) != 0 {
			return unsupportedIntrinsicValue(node, "int", source, ctx)
		}
		return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), receiver)
	})
	registerInstanceIntrinsicResultType("Currency", "equals", "boolean")
	registerInstanceNodeIntrinsic("Currency", "equals", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if invocationArgumentCount(node) != 1 {
			return unsupportedIntrinsicValue(node, "boolean", source, ctx)
		}
		argument := invocationArgumentNode(node, 0)
		argumentCtx := ctx.Clone()
		argumentCtx.expectedType, argumentCtx.expectedTypeRoot = "java.lang.Object", argument
		return stdjavaCall(ctx, "ObjectEqualsExecution", intrinsicExecutionExpr(ctx), receiver, ParseExpr(argument, source, argumentCtx))
	})
	for _, method := range []string{"getCurrencyCode", "toString"} {
		registerInstanceIntrinsicResultType("Currency", method, "java.lang.String")
		registerInstanceNodeIntrinsic("Currency", method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != 0 {
				return unsupportedIntrinsicValue(node, "java.lang.String", source, ctx)
			}
			if method == "toString" {
				return methodCall(receiver, "StringJava2goExecution", intrinsicExecutionExpr(ctx))
			}
			return methodCall(receiver, "GetCurrencyCode")
		})
	}
	registerConstructorNodeIntrinsic("Currency", func(_ []ast.Expr, _ []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		return unsupportedIntrinsicValue(node, "java.util.Currency", source, ctx)
	})
	for _, unavailable := range []struct{ method, result string }{
		{"getDefaultFractionDigits", "int"}, {"getNumericCode", "int"}, {"getNumericCodeAsString", "java.lang.String"},
		{"getSymbol", "java.lang.String"}, {"getDisplayName", "java.lang.String"},
	} {
		registerInstanceIntrinsicResultType("Currency", unavailable.method, unavailable.result)
		registerInstanceNodeIntrinsic("Currency", unavailable.method, func(_ ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			return unsupportedIntrinsicValue(node, unavailable.result, source, ctx)
		})
	}
	registerStaticNodeIntrinsic("Currency", "getAvailableCurrencies", func(node *sitter.Node, _ []ast.Expr, ctx Ctx, source []byte) ast.Expr {
		return unsupportedIntrinsicValue(node, "java.lang.Object", source, ctx)
	})
}
