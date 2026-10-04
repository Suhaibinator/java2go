package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strings"
)

// Arrays is a non-generic nominal utility declaration, even when a Java value
// is used to qualify one of its static methods. A historical basename default
// is not declaration evidence.
func canonicalArraysUtilityType(javaType string, ctx Ctx) bool {
	element, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(element)
	if rank != 0 || len(arguments) != 0 || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	if !strings.Contains(base, ".") && visibleTypeParameterDeclarationForJavaType(base, ctx) != nil {
		return false
	}
	if base == "java.util.Arrays" {
		return true
	}
	if base != "Arrays" || ctx.currentFile == nil {
		return false
	}
	if pkg, present := ctx.currentFile.Imports[base]; present {
		return pkg == "java.util"
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known || owner != "java.util.Arrays" {
		return false
	}
	imported := false
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		if pkg == "java.util" {
			imported = true
		}
		if pkg != "java.util" && resolveClassScopeByQualifiedName(ctx, pkg+".Arrays") != nil {
			return false
		}
	}
	return imported
}

func arraysUtilityRuntimeTypeExpr(javaType string, arguments, parameters []string, ctx Ctx) (ast.Expr, bool) {
	if len(arguments) != 0 || !canonicalArraysUtilityType(javaType, ctx) {
		return nil, false
	}
	for _, parameter := range parameters {
		if javaType == parameter {
			return nil, false
		}
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("JavaArrays", ctx)}, true
}

// Retained source declaration/binder identity wins over a spelling resolved in
// the consuming namespace. A canonical external declaration has no source
// nominal scope; its already-qualified origin must not be rebound to a caller's
// nested type with the same written path.
func arraysUtilityReceiverCanonical(object *sitter.Node, ctx Ctx, source []byte) bool {
	if object == nil {
		return false
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(object, ctx, source, &origin)
	return known && arraysUtilityOriginCanonical(javaType, origin, ctx)
}

func arraysUtilityOriginCanonical(javaType string, origin inferredJavaTypeOrigin, ctx Ctx) bool {
	if origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	if origin.declaringOwner != nil {
		proof := ctx.Clone()
		proof.currentFile = nil
		proof.syntheticTypeParameters = nil
		declaring := classScopeCtx(origin.declaringOwner, proof)
		if declaring.currentFile == nil {
			return false
		}
		return canonicalArraysUtilityType(origin.javaType, declaring)
	}
	if origin.javaType == "java.util.Arrays" {
		return true
	}
	return canonicalArraysUtilityType(javaType, ctx)
}

// This is expression qualification of a STATIC method, not virtual dispatch.
// The shared selector proves the declaration and byte[]/int/int signature;
// typed argument parsing preserves boxing, unboxing and mutation snapshots.
func lowerArraysUtilityByteCopyRange(qualifier ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if !arraysByteCopyRangeSelected(invocation, ctx, source) {
		return nil
	}
	args := parseTypedIntrinsicInvocationArguments(invocation, invocation.ChildByFieldName("object"), "Arrays", "copyOfRange", source, ctx)
	if len(args) != 3 {
		return nil
	}
	call := stdjavaCall(ctx, "ArraysByteCopyOfRange", args...)
	results := &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr("byte[]", inScopeTypeParameters(ctx), ctx)}}}
	return stageStaticQualifierWithResults(qualifier, call, results)
}

// Shared mechanical core of the original source static-call stager. The caller
// owns declaration/result selection and supplies its already-parsed qualifier.
func stageStaticQualifierWithResults(qualifier, call ast.Expr, results *ast.FieldList) ast.Expr {
	body := []ast.Stmt{&ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{qualifier},
	}}
	body = append(body, invocationClosureCallStatement(call, results))
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: results},
		Body: &ast.BlockStmt{List: body},
	}}
}
