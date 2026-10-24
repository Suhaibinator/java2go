package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// projectErasedFieldArrayOperand supplies javac's delayed array checkcast only
// at an array-consuming operation. Observing an erased field as Object remains
// cast-free. The entire field expression is evaluated once before the index,
// store RHS, or loop body; JavaArrayCast preserves null and array identity.
func projectErasedFieldArrayOperand(expr ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if expr == nil || node == nil || ctx.currentFile == nil {
		return expr
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || (node.Type() != "identifier" && node.Type() != "field_access") {
		return expr
	}
	proof := ctx.Clone()
	proof.expectedType, proof.expectedTypeRoot = "", nil
	proof.importAliases = make(map[string]string, len(ctx.importAliases))
	for path, alias := range ctx.importAliases {
		proof.importAliases[path] = alias
	}
	proof.usedImports = make(map[string]bool, len(ctx.usedImports))
	for path, used := range ctx.usedImports {
		proof.usedImports[path] = used
	}
	sourceView, erasure, admitted := directOwnerFieldAccessView(node, proof, source)
	if !admitted {
		return expr
	}
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(node, proof, source, &origin)
	if !known || !completeJavaTypeOrigin(origin) || origin.arrayRank <= 0 || origin.parameter != nil || origin.wildcard {
		return expr
	}
	if _, rank := javaArrayTypeParts(sourceView); rank != origin.arrayRank || javaInferenceTypeAssignable(erasure, sourceView, proof) {
		return expr
	}
	var physical ast.Expr
	if _, _, _, reified := expressionUsesReifiedReferenceArray(node, proof, source); reified {
		physical = reifiedReferenceArrayTypeExpr(ctx)
	} else if _, component, _, primitive := expressionUsesPrimitiveArray(node, proof, source); primitive {
		physical = &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("PrimitiveArray", ctx), []ast.Expr{component})}
	} else {
		return expr
	}
	descriptor, ok := declaredOriginDescriptor(origin, ctx, map[*symbol.TypeParamDeclaration]bool{})
	if !ok {
		return expr
	}
	return stdjavaGenericCall(ctx, "JavaArrayCast", []ast.Expr{physical}, []ast.Expr{expr, descriptor})
}
