package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Library intrinsics have no source methodResolution, so their receivers need
// the same delayed field/result checkcast explicitly. Selecting Integer's
// intValue on a polluted Box<Integer> must fail even though Number's runtime
// accessor would also accept the Long stored through a raw alias.
func projectDirectOwnerErasedIntrinsicReceiver(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if receiver == nil || node == nil {
		return receiver
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return receiver
	}
	var sourceView, erasure string
	var known bool
	switch node.Type() {
	case "field_access":
		sourceView, erasure, known = directOwnerFieldAccessView(node, ctx, source)
	case "method_invocation":
		sourceView, erasure, known = directOwnerMethodResultView(node, ctx, source)
	}
	if !known {
		return receiver
	}
	_, wrapper := builtinJavaWrapperPrimitive(sourceView, ctx)
	if !wrapper && (stripJavaQualifier(sourceView) != "String" || resolveClassScopeByQualifiedName(ctx, sourceView) != nil) {
		return receiver
	}
	return projectDirectOwnerErasedView(receiver, sourceView, erasure, ctx)
}

// narrowDirectOwnerFieldResultReceiver inserts the checkcast javac performs
// when a field whose declaration type is a class-owned type parameter becomes
// the receiver of a member selection. The complete selector is evaluated as
// ObjectView's first argument, so receiver side effects precede a failing cast
// and the selected method is never entered after that failure.
func narrowDirectOwnerFieldResultReceiver(
	receiver ast.Expr,
	receiverNode *sitter.Node,
	outerResolution *methodResolution,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if receiver == nil || receiverNode == nil || receiverNode.Type() != "field_access" ||
		outerResolution == nil || outerResolution.def == nil || outerResolution.def.IsStatic {
		return receiver
	}

	sourceView, erasure, ok := directOwnerFieldAccessView(receiverNode, ctx, source)
	if !ok {
		return receiver
	}
	targetView := sourceView
	if visibleTypeParameterDeclarationForJavaType(sourceView, ctx) != nil {
		memberOwner := outerResolution.receiverScope
		if memberOwner == nil {
			memberOwner = outerResolution.owner
		}
		targetView = javaInferenceTypeName(memberOwner)
	}
	return projectDirectOwnerErasedView(receiver, targetView, erasure, ctx)
}

// narrowDirectOwnerFieldSelectionReceiver is the field-selection counterpart
// of the method-receiver projection above. A concrete substituted view casts to
// that concrete type before selecting its field; a live intersection type casts
// only to the bound that declares the selected field.
func narrowDirectOwnerFieldSelectionReceiver(
	receiver ast.Expr,
	receiverNode *sitter.Node,
	outerResolution *fieldResolution,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if receiver == nil || receiverNode == nil || receiverNode.Type() != "field_access" ||
		outerResolution == nil || outerResolution.def == nil || outerResolution.def.IsStatic {
		return receiver
	}
	sourceView, erasure, ok := directOwnerFieldAccessView(receiverNode, ctx, source)
	if !ok {
		return receiver
	}
	targetView := sourceView
	if visibleTypeParameterDeclarationForJavaType(sourceView, ctx) != nil {
		targetView = javaInferenceTypeName(outerResolution.owner)
	}
	return projectDirectOwnerErasedView(receiver, targetView, erasure, ctx)
}

// projectDirectOwnerErasedExpressionForExpected restores the source-level view
// of an erased field or method result only when the expression itself occupies
// a target-typed context that cannot accept the erasure. Assigning to Object or
// the first bound consequently remains cast-free, while assigning to a concrete
// substituted type performs the delayed Java checkcast.
func projectDirectOwnerErasedExpressionForExpected(
	expr ast.Expr,
	node *sitter.Node,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if expr == nil || node == nil || strings.TrimSpace(ctx.expectedType) == "" ||
		!expectedTypeTargetsExpression(ctx, node) {
		return expr
	}
	if currentErasedCallableOwnerTypeParameter(ctx.expectedType, ctx) {
		// A local/return target spelled with the current generic declaration's
		// own parameter has the same erased JVM representation. Narrow only when
		// that value later leaves the generic body through a concrete source view.
		return expr
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return expr
	}

	var sourceView, erasure string
	var ok bool
	switch node.Type() {
	case "field_access":
		sourceView, erasure, ok = directOwnerFieldAccessView(node, ctx, source)
	case "method_invocation":
		sourceView, erasure, ok = directOwnerMethodResultView(node, ctx, source)
	}
	target := ctx.expectedType
	if _, primitive := javaPrimitiveType(target); primitive {
		// Unboxing first checks the expression's reference view, then extracts
		// and widens the primitive. ObjectView must never target a primitive ID.
		if _, boxed := javaUnboxingPrimitive(sourceView, ctx); !boxed {
			return expr
		}
		target = sourceView
	}
	if !ok || javaInferenceTypeAssignable(erasure, target, ctx) {
		return expr
	}
	return projectDirectOwnerErasedView(expr, target, erasure, ctx)
}

func currentErasedCallableOwnerTypeParameter(javaType string, ctx Ctx) bool {
	_, ok := currentErasedCallableOwnerTypeParameterErasure(javaType, ctx)
	return ok
}

func currentErasedCallableOwnerTypeParameterErasure(javaType string, ctx Ctx) (string, bool) {
	physical := genericFamilyPhysicalJavaType(javaType, ctx)
	if physical != javaType {
		return physical, true
	}
	if ctx.currentClass == nil || ctx.localScope == nil ||
		!directOwnerCallableMethodEligible(ctx.currentClass, ctx.localScope, ctx) {
		return "", false
	}
	declaration := visibleTypeParameterDeclarationForJavaType(javaType, ctx)
	if declaration == nil || !methodDirectlyUsesTypeParameterDeclaration(ctx.localScope, declaration) ||
		!ownerTypeParameterCallableShapeSupported(declaration, ctx) {
		return "", false
	}
	for _, parameter := range ctx.currentClass.TypeParameters {
		if parameter.Declaration == declaration {
			erasure := qualifyJavaTypeInDeclaringContext(
				rawTypeParameterErasure(parameter, ctx.currentClass.TypeParameters),
				ctx.currentClass,
			)
			if !javaTypeHasInterfaceRepresentation(erasure, ctx) && !leafObjectErasure(ctx.currentClass, erasure, ctx) {
				return "", false
			}
			return erasure, true
		}
	}
	return "", false
}

func projectDirectOwnerErasedMethodReferenceResult(
	expr ast.Expr,
	resolution *methodResolution,
	target *invocationTargetInfo,
	ctx Ctx,
) ast.Expr {
	if expr == nil || resolution == nil || resolution.owner == nil || resolution.def == nil {
		return expr
	}
	erasure, ok := directOwnerOrdinaryMethodInterfaceErasure(resolution.owner, resolution.def, ctx)
	if !ok {
		return expr
	}
	// The shared SAM signature also carries runtime functional interfaces such
	// as Supplier, which have no source declaration in the symbol graph.
	_, targetView := methodReferenceJavaSignature(ctx)
	if targetView == "" || targetView == "void" {
		return expr
	}
	if _, primitive := javaPrimitiveType(targetView); primitive {
		// Restore the referenced method's reference result before the regular
		// unboxing/widening adapter. An Integer result consumed as long still
		// needs an Integer checkcast, not a Long checkcast.
		targetView = methodReferenceDeclaredResultType(resolution, target, ctx)
	}
	if javaInferenceTypeAssignable(erasure, targetView, ctx) {
		return expr
	}
	return projectDirectOwnerErasedView(expr, targetView, erasure, ctx)
}

func directOwnerFieldAccessView(
	node *sitter.Node,
	ctx Ctx,
	source []byte,
) (string, string, bool) {
	if node == nil || node.Type() != "field_access" {
		return "", "", false
	}
	objectNode := node.ChildByFieldName("object")
	fieldNode := node.ChildByFieldName("field")
	if objectNode == nil || fieldNode == nil {
		return "", "", false
	}
	target := resolveInvocationTarget(objectNode, ctx, source)
	if target == nil || target.classScope == nil {
		return "", "", false
	}
	resolution := findFieldResolutionInHierarchy(target.classScope, fieldNode.Content(source), ctx)
	if resolution == nil || resolution.owner == nil || resolution.def == nil || resolution.def.IsStatic {
		return "", "", false
	}
	erasure, ok := directOwnerOrdinaryFieldInterfaceErasure(resolution.owner, resolution.def, ctx)
	if !ok {
		return "", "", false
	}
	// Preserve the field's source view through inherited and wildcard receivers.
	// instantiatedFieldJavaType performs the declaration-site positional mapping
	// and turns a readable `? extends Root` capture into Root.
	sourceView := instantiatedFieldJavaType(
		target.classScope,
		target.classJavaTypeArgs,
		resolution,
		ctx,
	)
	if strings.TrimSpace(sourceView) == "" {
		return "", "", false
	}
	return sourceView, erasure, true
}

func directOwnerMethodResultView(
	node *sitter.Node,
	ctx Ctx,
	source []byte,
) (string, string, bool) {
	target, resolution := resolvedInstanceInvocation(node, ctx, source)
	if target == nil || resolution == nil || resolution.owner == nil || resolution.def == nil {
		return "", "", false
	}
	erasure, ok := directOwnerOrdinaryMethodInterfaceErasure(resolution.owner, resolution.def, ctx)
	if !ok {
		return "", "", false
	}
	use, _ := directOwnerTypeParameterForDefinition(resolution.owner, resolution.def)
	ownerArguments := invocationOwnerTypeArguments(target, resolution, ctx)
	sourceView, ok := ownerTypeParameterSourceArgument(resolution.owner, use.parameter, ownerArguments)
	if !ok {
		return "", "", false
	}
	return sourceView, erasure, true
}

func projectDirectOwnerErasedView(
	expr ast.Expr,
	targetView string,
	erasure string,
	ctx Ctx,
) ast.Expr {
	targetView, ok := readableOwnerResultProjectionType(targetView)
	if !ok || javaInferenceTypeAssignable(erasure, targetView, ctx) {
		return expr
	}
	targetType := javaTypeStringToGoTypeExpr(targetView, inScopeTypeParameters(ctx), ctx)
	descriptor, ok := javaTypeDescriptorExpr(targetView, ctx)
	if !ok {
		return expr
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{targetType}, []ast.Expr{expr, descriptor})
}
