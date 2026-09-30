package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// builtinJavaWrapperPrimitive recognizes java.lang wrappers only after source
// declarations have had their normal opportunity to shadow the short name.
func builtinJavaWrapperPrimitive(javaType string, ctx Ctx) (string, bool) {
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 {
		return "", false
	}
	base, arguments := parseJavaTypeString(base)
	if len(arguments) != 0 || (strings.Contains(base, ".") && !strings.HasPrefix(base, "java.lang.")) {
		return "", false
	}
	if !strings.HasPrefix(base, "java.lang.") && ctx.currentFile != nil && resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	if !strings.Contains(base, ".") && visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return "", false
	}
	return ternaryBoxedPrimitive(base)
}

// javaUnboxingPrimitive follows only declared bounds that lead to a concrete
// java.lang wrapper. Number is deliberately insufficient: Java cannot unbox a
// Number reference. Declaration identity keeps shadowed bounds distinct.
func javaUnboxingPrimitive(javaType string, ctx Ctx) (string, bool) {
	if primitive, wrapper := builtinJavaWrapperPrimitive(javaType, ctx); wrapper {
		return primitive, true
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 {
		return "", false
	}
	base, arguments := parseJavaTypeString(base)
	if len(arguments) != 0 || strings.Contains(base, ".") {
		return "", false
	}
	lookup := newTypeParameterLookup(visibleTypeParameterDeclarations(ctx))
	parameter, found := lookup.resolve(symbol.JavaType{}, base)
	if !found {
		return "", false
	}
	visited := make(map[typeParameterIdentityKey]bool)
	var follow func(symbol.TypeParam) (string, bool)
	follow = func(parameter symbol.TypeParam) (string, bool) {
		identity := identityKeyForTypeParameter(parameter)
		if visited[identity] {
			return "", false
		}
		visited[identity] = true
		for _, bound := range parameter.Bounds {
			boundBase, arguments := parseJavaTypeString(strings.TrimSpace(bound.Original))
			if len(arguments) != 0 {
				continue
			}
			if dependency, found := lookup.resolve(bound, boundBase); found {
				if primitive, found := follow(dependency); found {
					return primitive, true
				}
				continue
			}
			if primitive, wrapper := builtinJavaWrapperPrimitive(boundBase, ctx); wrapper {
				return primitive, true
			}
		}
		return "", false
	}
	return follow(parameter)
}

func javaBoxExpr(expr ast.Expr, primitive string, ctx Ctx) ast.Expr {
	wrapper := ternaryBoxedJavaType(primitive)
	if wrapper == "" {
		return expr
	}
	// Inference of a constant must never choose the host-sized Go int for a
	// Java primitive. The factory's parameter also enforces the primitive ABI.
	if conversion := goPrimitiveConversionName(primitive); conversion != "" {
		expr = &ast.CallExpr{Fun: &ast.Ident{Name: conversion}, Args: []ast.Expr{expr}}
	}
	return stdjavaCall(ctx, "Box"+wrapper, expr)
}

func javaUnboxExpr(expr ast.Expr, wrapperJavaType string, ctx Ctx) ast.Expr {
	primitive, ok := javaUnboxingPrimitive(wrapperJavaType, ctx)
	if !ok {
		return expr
	}
	if _, direct := builtinJavaWrapperPrimitive(wrapperJavaType, ctx); !direct {
		wrapper := ternaryBoxedJavaType(primitive)
		wrapperType := javaTypeStringToGoTypeExpr("java.lang."+wrapper, inScopeTypeParameters(ctx), ctx)
		expr = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{wrapperType}, []ast.Expr{expr, stdjavaQualifiedExpr(wrapper+"TypeID", ctx)})
	}
	return stdjavaCall(ctx, "Unbox"+ternaryBoxedJavaType(primitive), expr)
}

// convertJavaValue applies the shared boxing, unboxing, and primitive widening
// portion of assignment or invocation conversion. Applicability is checked
// separately: invocation must never inherit assignment-only constant narrowing.
func convertJavaValue(expr ast.Expr, actualType, expectedType string, ctx Ctx) (ast.Expr, bool) {
	actualPrimitive, actualIsPrimitive := javaPrimitiveType(actualType)
	expectedPrimitive, expectedIsPrimitive := javaPrimitiveType(expectedType)
	if actualIsPrimitive && expectedIsPrimitive {
		if actualPrimitive == expectedPrimitive {
			return expr, true
		}
		if _, widening := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive); widening {
			return &ast.CallExpr{Fun: &ast.Ident{Name: goPrimitiveConversionName(expectedPrimitive)}, Args: []ast.Expr{expr}}, true
		}
		return expr, false
	}
	if actualIsPrimitive {
		if primitive, wrapper := builtinJavaWrapperPrimitive(expectedType, ctx); wrapper && primitive == actualPrimitive {
			return javaBoxExpr(expr, primitive, ctx), true
		}
		if builtinJavaReferenceAssignable("java.lang."+ternaryBoxedJavaType(actualPrimitive), expectedType, ctx) {
			return javaBoxExpr(expr, actualPrimitive, ctx), true
		}
	}
	if primitive, wrapper := javaUnboxingPrimitive(actualType, ctx); wrapper && expectedIsPrimitive {
		unboxed := javaUnboxExpr(expr, actualType, ctx)
		if primitive == expectedPrimitive {
			return unboxed, true
		}
		if _, widening := javaPrimitiveWideningDistance(primitive, expectedPrimitive); widening {
			return &ast.CallExpr{Fun: &ast.Ident{Name: goPrimitiveConversionName(expectedPrimitive)}, Args: []ast.Expr{unboxed}}, true
		}
	}
	return expr, false
}

func parseJavaBooleanExpr(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	value := ParseExpr(node, source, ctx)
	return coerceArgumentToExpectedType(value, node, "boolean", ctx, source)
}

func parseJavaIndexExpr(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	value := ParseExpr(node, source, ctx)
	return coerceArgumentToExpectedType(value, node, "int", ctx, source)
}

// snapshotJavaExpressionValue turns a mutable read into an ordered Go call.
// Untyped literals are left to their surrounding target context; existing
// calls already form a Go evaluation-order boundary.
func snapshotJavaExpressionValue(expr ast.Expr, ctx Ctx) ast.Expr {
	if javaGoExpressionIsConstant(expr) {
		return expr
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		if !javaGoExpressionIsTypeConversion(call.Fun) {
			return expr
		}
	}
	return stdjavaCall(ctx, "EvaluationValue", expr)
}

func snapshotJavaExpressionValueForType(expr ast.Expr, javaType string, ctx Ctx) ast.Expr {
	result := snapshotJavaExpressionValue(expr, ctx)
	if result == expr {
		return expr
	}
	if _, primitive := javaPrimitiveType(javaType); primitive {
		return stdjavaGenericCall(ctx, "EvaluationValue", []ast.Expr{javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{expr})
	}
	return result
}

func javaLaterExpressionEffects(nodes []*sitter.Node, source []byte, ctx Ctx) []bool {
	later := make([]bool, len(nodes))
	hasEffect := false
	for index := len(nodes) - 1; index >= 0; index-- {
		later[index] = hasEffect
		hasEffect = hasEffect || javaBinaryOperandMayHaveEffects(nodes[index], source, ctx)
	}
	return later
}

func javaGoExpressionIsTypeConversion(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return isPrimitiveCastTarget(value) || value.Name == "any" || value.Name == "string"
	case *ast.InterfaceType, *ast.StarExpr, *ast.ArrayType:
		return true
	}
	return false
}

func javaGoExpressionIsConstant(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.BasicLit:
		return true
	case *ast.Ident:
		if value.Name == "nil" || value.Name == "true" || value.Name == "false" {
			return true
		}
		if _, err := strconv.Unquote(value.Name); err == nil {
			return true
		}
		if _, err := strconv.ParseInt(value.Name, 0, 64); err == nil {
			return true
		}
		if _, err := strconv.ParseFloat(value.Name, 64); err == nil {
			return true
		}
	case *ast.ParenExpr:
		return javaGoExpressionIsConstant(value.X)
	case *ast.UnaryExpr:
		return (value.Op == token.ADD || value.Op == token.SUB || value.Op == token.XOR || value.Op == token.NOT) && javaGoExpressionIsConstant(value.X)
	case *ast.BinaryExpr:
		return javaGoExpressionIsConstant(value.X) && javaGoExpressionIsConstant(value.Y)
	case *ast.CallExpr:
		if javaGoExpressionIsTypeConversion(value.Fun) && len(value.Args) == 1 {
			return javaGoExpressionIsConstant(value.Args[0])
		}
	}
	return false
}

// javaLooseInvocationConversionCost implements only the extra conversions of
// phase two: boxing plus reference widening, or unboxing plus primitive
// widening. Widening followed by boxing and constant narrowing are excluded.
func javaLooseInvocationConversionCost(argNode *sitter.Node, expectedType string, typeParameters []string, ctx Ctx, source []byte) (int, bool, bool) {
	inferenceCtx := ctx.Clone()
	inferenceCtx.expectedType = ""
	inferenceCtx.expectedTypeRoot = nil
	actualType, known := inferExprJavaType(argNode, inferenceCtx, source)
	if !known {
		return 0, false, false
	}
	actualPrimitive, primitive := javaPrimitiveType(actualType)
	expectedPrimitive, primitiveExpected := javaPrimitiveType(expectedType)
	if primitive && !primitiveExpected {
		expectedBase, _ := parseJavaTypeString(expectedType)
		if containsString(typeParameters, stripJavaQualifier(expectedBase)) ||
			builtinJavaReferenceAssignable("java.lang."+ternaryBoxedJavaType(actualPrimitive), expectedType, ctx) {
			return 8, false, true
		}
	}
	if actualPrimitive, wrapper := javaUnboxingPrimitive(actualType, ctx); wrapper && primitiveExpected {
		if actualPrimitive == expectedPrimitive {
			return 8, false, true
		}
		if distance, widening := javaPrimitiveWideningDistance(actualPrimitive, expectedPrimitive); widening {
			return 8 + distance, false, true
		}
	}
	return 0, false, false
}
