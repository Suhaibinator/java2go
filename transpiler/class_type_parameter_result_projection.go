package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// narrowDirectOwnerMethodResultReceiver inserts the checkcast Java performs
// when a parameterized class-type-parameter result becomes the receiver of a
// chained member selection. The completed inner call remains the first
// ObjectView argument, so its body and side effects happen before a possible
// ClassCastException.
func narrowDirectOwnerMethodResultReceiver(
	receiver ast.Expr,
	receiverNode *sitter.Node,
	outerResolution *methodResolution,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if receiver == nil || receiverNode == nil || receiverNode.Type() != "method_invocation" ||
		outerResolution == nil || outerResolution.def == nil || outerResolution.def.IsStatic {
		return receiver
	}

	innerTarget, innerResolution := resolvedInstanceInvocation(receiverNode, ctx, source)
	if innerTarget == nil || innerResolution == nil || innerResolution.def == nil || innerResolution.owner == nil {
		return receiver
	}
	use, ok := directOwnerTypeParameterForDefinition(innerResolution.owner, innerResolution.def)
	if !ok {
		return receiver
	}

	ownerArguments := invocationOwnerTypeArguments(innerTarget, innerResolution, ctx)
	sourceView, ok := ownerTypeParameterSourceArgument(innerResolution.owner, use.parameter, ownerArguments)
	if !ok || sourceView == "" {
		return receiver
	}
	sourceView, ok = readableOwnerResultProjectionType(sourceView)
	if !ok {
		return receiver
	}
	if visibleTypeParameterDeclarationForJavaType(sourceView, ctx) != nil {
		memberOwner := outerResolution.receiverScope
		if memberOwner == nil {
			memberOwner = outerResolution.owner
		}
		sourceView = javaInferenceTypeName(memberOwner)
	}

	// A raw receiver normalizes its source argument to the erasure and therefore
	// needs no additional cast. Any narrower concrete substituted view continues
	// below and is projected before the chained member selection.
	erasure := qualifyJavaTypeInDeclaringContext(use.erasure, innerResolution.owner)
	if javaInferenceTypeAssignable(erasure, sourceView, ctx) {
		return receiver
	}

	targetType := javaTypeStringToGoTypeExpr(sourceView, inScopeTypeParameters(ctx), ctx)
	descriptor, ok := javaTypeDescriptorExpr(sourceView, ctx)
	if !ok {
		return receiver
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{targetType}, []ast.Expr{receiver, descriptor})
}

// readableOwnerResultProjectionType keeps this first lowering slice honest:
// concrete rank-zero views have one representable Go checkcast target.
// Wildcards and parameterized views require additional capture or canonical
// runtime-type planning and are deliberately deferred.
func readableOwnerResultProjectionType(sourceView string) (string, bool) {
	sourceView = strings.TrimSpace(sourceView)
	if strings.HasPrefix(sourceView, "?") {
		return "", false
	}
	component, rank := javaArrayTypeParts(sourceView)
	if rank != 0 {
		return "", false
	}
	base, arguments := parseJavaTypeString(component)
	if strings.TrimSpace(base) == "" || len(arguments) != 0 {
		return "", false
	}
	return strings.TrimSpace(base), true
}

func resolvedInstanceInvocation(
	node *sitter.Node,
	ctx Ctx,
	source []byte,
) (*invocationTargetInfo, *methodResolution) {
	if node == nil || node.Type() != "method_invocation" {
		return nil, nil
	}
	objectNode := node.ChildByFieldName("object")
	nameNode := node.ChildByFieldName("name")
	if objectNode == nil || nameNode == nil || resolveClassScopeByIdentifier(ctx, source, objectNode) != nil {
		return nil, nil
	}
	target := resolveInvocationTarget(objectNode, ctx, source)
	if target == nil {
		return nil, nil
	}
	resolution, _ := findBestMethodForInvocationTarget(
		target,
		nameNode.Content(source),
		node.ChildByFieldName("arguments"),
		true,
		true,
		ctx,
		source,
	)
	if resolution == nil || resolution.def == nil || resolution.def.IsStatic {
		return nil, nil
	}
	return target, resolution
}

func ownerTypeParameterSourceArgument(
	owner *symbol.ClassScope,
	parameter symbol.TypeParam,
	arguments []string,
) (string, bool) {
	if owner == nil || parameter.Declaration == nil || len(arguments) != len(owner.TypeParameters) {
		return "", false
	}
	for index, candidate := range owner.TypeParameters {
		if candidate.Declaration == parameter.Declaration {
			return strings.TrimSpace(arguments[index]), true
		}
	}
	return "", false
}
