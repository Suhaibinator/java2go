package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A written scalar reference T belongs to the actual canonical Stream.of use.
// Both the physical factory and derived Java receiver use this certificate.
func explicitStreamFactoryElement(invocation *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, string, bool) {
	var empty inferredJavaTypeOrigin
	if invocation == nil || !sourceArrayListNativeSourceContext(invocation, ctx, source) {
		return empty, "", false
	}
	if invocation.Type() != "method_invocation" || invocation.ChildByFieldName("name") == nil || invocation.ChildByFieldName("name").Content(source) != "of" {
		return empty, "", false
	}
	object := invocation.ChildByFieldName("object")
	selected, known := intrinsicStaticClassName(object, ctx, source)
	if !known || selected != "Stream" || staticIntrinsics[intrinsicKey{"Stream", "of"}] == nil || object == nil {
		return empty, "", false
	}
	// Prove the original type namespace before rendering a canonical spelling.
	// A simple dispatch key alone cannot exclude source, value or binder names.
	if sourceGenericViewLocalTypeShadowed(object, object.Content(source), source) {
		return empty, "", false
	}
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	owner := declaredJavaTypeOrigin(symbol.JavaType{Original: object.Content(source), TypeParameterBindings: bindings}, ctx)
	if owner.unresolved || owner.wildcard || owner.arrayRank != 0 || owner.parameter != nil || owner.nominalScope != nil || owner.externalName != "java.util.stream.Stream" || len(owner.arguments) != 0 {
		return empty, "", false
	}
	arguments := invocation.ChildByFieldName("type_arguments")
	if arguments == nil || arguments.NamedChildCount() != 1 {
		return empty, "", false
	}
	writtenType := arguments.NamedChild(0)
	if sourceGenericViewLocalTypeShadowed(writtenType, writtenType.Content(source), source) {
		return empty, "", false
	}
	element := declaredJavaTypeOrigin(symbol.JavaType{Original: writtenType.Content(source), TypeParameterBindings: bindings}, ctx)
	if !sourceArrayListNativeScalarOrigin(element) {
		return empty, "", false
	}
	written, retained, known := sourceDiamondOriginEdge(element)
	if !known {
		return empty, "", false
	}
	for _, declaration := range retained {
		found := false
		for _, parameter := range visibleTypeParameterDeclarations(ctx) {
			found = found || parameter.Declaration == declaration
		}
		if !found {
			return empty, "", false
		}
	}
	roundTrip := declaredJavaTypeOrigin(symbol.JavaType{Original: written, TypeParameterBindings: bindings}, ctx)
	if !sameJavaTypeOrigin(element, roundTrip) {
		return empty, "", false
	}
	return element, written, true
}

// The ordinary intrinsic result spelling is not a structured declaration
// origin. Retain the actual written element directly; re-reading a display
// spelling in a later caller could select a different same-named binder.
func explicitStreamFactoryResultType(invocation *sitter.Node, ctx Ctx, source []byte, origins ...*inferredJavaTypeOrigin) (string, bool) {
	// Most invocations cannot use this capability. This syntax-only filter reads
	// no token bytes and does not speculate about ordinary Stream inference.
	if invocation == nil || invocation.Type() != "method_invocation" || invocation.ChildByFieldName("object") == nil {
		return "", false
	}
	arguments := invocation.ChildByFieldName("type_arguments")
	if arguments == nil || arguments.NamedChildCount() != 1 {
		return "", false
	}
	if !sourceArrayListNativeSourceContext(invocation, ctx, source) || !sourceArrayListConstructorFileNamespace(ctx.currentFile) {
		return "", false
	}
	proof := ctx.Clone()
	proof.memberTypeHeaderOwner = nil
	proof.localClasses = nil
	proof.anonymousClasses = nil
	element, written, known := explicitStreamFactoryElement(invocation, proof, source)
	if !known {
		return "", false
	}
	java := "java.util.stream.Stream<" + written + ">"
	setInferredJavaTypeOrigin(origins, inferredJavaTypeOrigin{externalName: "java.util.stream.Stream", javaType: java, arguments: []inferredJavaTypeOrigin{element}})
	return java, true
}
