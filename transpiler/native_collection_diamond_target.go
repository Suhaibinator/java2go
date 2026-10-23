package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Project only ArrayList<E>'s declared Collection<E> parent at an actual
// target-typed diamond. This does not change collection storage or raw syntax.
func nativeCollectionDiamondTargetArguments(expected, constructed string, ctx Ctx, invocation *sitter.Node) ([]string, bool) {
	if ctx.currentFile == nil || invocation == nil || invocation.Type() != "object_creation_expression" || objectCreationClassBody(invocation) != nil || ctx.currentFile.ParsedRoot() == nil || !threadSourceNodeWithin(invocation, ctx.currentFile.ParsedRoot()) {
		return nil, false
	}
	root := unwrapParenthesizedExpressionNode(ctx.expectedTypeRoot)
	if root == nil || !root.Equal(invocation) || !expectedTypeTargetsExpression(ctx, invocation) || strings.TrimSpace(ctx.expectedType) != strings.TrimSpace(expected) {
		return nil, false
	}
	source := ctx.currentFile.Source
	if invocation.EndByte() > uint32(len(source)) || invocation.HasError() {
		return nil, false
	}
	// Retain the actual callable's TYPE namespace; a stale caller scope cannot
	// hide a method binder or an enclosing source type with a platform name.
	var callable *sitter.Node
	foundOwner := false
	for ancestor := invocation.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		if callable == nil && (ancestor.Type() == "method_declaration" || ancestor.Type() == "constructor_declaration") {
			callable = ancestor
		}
		if sourceGenericViewTypeDeclaration(ancestor) {
			if ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentClass.Class.DeclarationNode == nil || ancestor.HasError() || !ancestor.Equal(ctx.currentClass.Class.DeclarationNode) {
				return nil, false
			}
			foundOwner = true
			break
		}
	}
	if !foundOwner || (callable == nil && ctx.localScope != nil) || (callable != nil && (ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || !ctx.localScope.DeclarationNode.Equal(callable))) {
		return nil, false
	}
	typ := invocation.ChildByFieldName("type")
	if typ == nil || typ.Type() != "generic_type" || typ.NamedChildCount() == 0 {
		return nil, false
	}
	actualBase := typ.NamedChild(0).Content(source)
	if actualBase != constructed || strings.TrimSpace(typ.Content(source)[len(actualBase):]) != "<>" {
		return nil, false
	}
	// The expected view must come from this original variable declaration, not
	// a caller's retained context or an enclosing consuming expression.
	value := invocation
	for value.Parent() != nil && value.Parent().Type() == "parenthesized_expression" {
		value = value.Parent()
	}
	declarator := value.Parent()
	if declarator == nil || declarator.Type() != "variable_declarator" || declarator.ChildByFieldName("value") == nil || !declarator.ChildByFieldName("value").Equal(value) {
		return nil, false
	}
	declaration := declarator.Parent()
	if declaration == nil || (declaration.Type() != "local_variable_declaration" && declaration.Type() != "field_declaration") {
		return nil, false
	}
	target := declaration.ChildByFieldName("type")
	if target == nil || target.HasError() || strings.TrimSpace(target.Content(source)) != strings.TrimSpace(expected) {
		return nil, false
	}
	base, arguments := parseJavaTypeString(expected)
	if len(arguments) != 1 {
		return nil, false
	}
	// Match the owned-context convention used by canonical Class argument
	// proofs. Original AST edges establish local/member lifetime below; caller
	// hoisting and synthetic registries are not Java declaration authority.
	proofContext := ctx.Clone()
	proofContext.localClasses = nil
	proofContext.syntheticTypeParameters = nil
	var argumentNode *sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(target) {
		if child.Type() == "type_arguments" {
			actual := nodeutil.SemanticNamedChildrenOf(child)
			if len(actual) != 1 {
				return nil, false
			}
			argumentNode = actual[0]
		}
	}
	if argumentNode == nil || argumentNode.HasError() || strings.TrimSpace(argumentNode.Content(source)) != strings.TrimSpace(arguments[0]) {
		return nil, false
	}
	argumentBase, argumentRank := javaArrayTypeParts(arguments[0])
	_, nested := parseJavaTypeString(argumentBase)
	if len(nested) != 0 || strings.ContainsAny(arguments[0], "?<> ") {
		return nil, false
	}
	// A stale ctx can omit an unhoisted local/member argument type even while
	// retaining the right callable. Never recertify that spelling as java.lang.
	argumentPrefix := strings.SplitN(argumentBase, ".", 2)[0]
	if nativeCollectionDiamondCallerPrefixConflict(argumentPrefix, ctx) || nativeCollectionDiamondCallerPrefixConflict(strings.SplitN(base, ".", 2)[0], ctx) || nativeCollectionDiamondCallerPrefixConflict(strings.SplitN(constructed, ".", 2)[0], ctx) {
		return nil, false
	}
	if sourceGenericViewLocalTypeShadowed(argumentNode, argumentBase, source) || classServiceASTTypePrefix(argumentNode, argumentPrefix, source, proofContext) {
		return nil, false
	}
	// The original TYPE namespace supplies a reference argument, including a
	// declaration-owned binder or a reference/primitive array. A primitive scalar,
	// unresolved name, null, or void is not a Java generic argument.
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: arguments[0], TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(proofContext))}, proofContext)
	if !completeJavaTypeOrigin(origin) || origin.wildcard || len(origin.arguments) != 0 {
		return nil, false
	}
	if origin.parameter == nil && origin.nominalScope == nil && origin.externalName == "" {
		if _, primitive := javaPrimitiveType(argumentBase); !primitive || argumentRank == 0 {
			return nil, false
		}
	}
	if origin.arrayRank != argumentRank {
		return nil, false
	}
	if nativeCollectionDiamondOwner(base, target, proofContext) != "java.util.Collection" || nativeCollectionDiamondOwner(constructed, typ, proofContext) != "java.util.ArrayList" {
		return nil, false
	}
	return append([]string(nil), arguments...), true
}

// The producer receives textual arguments and the caller context. Refuse a
// conflicting caller prefix rather than certifying one origin and letting the
// downstream physical converter pick a different hoisted/synthetic binding.
func nativeCollectionDiamondCallerPrefixConflict(prefix string, ctx Ctx) bool {
	if ctx.localClasses[prefix] != nil {
		return true
	}
	for _, parameter := range ctx.syntheticTypeParameters {
		if parameter.Name == prefix || parameter.EmittedName() == prefix {
			return true
		}
	}
	return false
}

// Resolve TYPE namespace bindings at the original occurrence. A matching
// basename, prefix, or an uninventoried local type cannot certify a JDK edge.
func nativeCollectionDiamondOwner(base string, occurrence *sitter.Node, ctx Ctx) string {
	source := ctx.currentFile.Source
	prefix := strings.SplitN(base, ".", 2)[0]
	if visibleTypeParameterDeclarationForJavaType(prefix, ctx) != nil || sourceGenericViewLocalTypeShadowed(occurrence, base, source) || classServiceASTTypePrefix(occurrence, prefix, source, ctx) || classServiceSourceTypePrefix(prefix, ctx) {
		return ""
	}
	if strings.Contains(base, ".") {
		if _, imported := ctx.currentFile.Imports[prefix]; imported {
			return ""
		}
	}
	if classServiceUnknownImportPrefix(prefix, ctx) {
		return ""
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known || !intrinsicOwnerSupported(owner) {
		return ""
	}
	// Do not use the runtime protocol or an undirected collection family as
	// nominal ancestry. Further parent edges require their own declaration proof.
	switch owner {
	case "java.util.Collection", "java.util.ArrayList":
		return owner
	}
	return ""
}
