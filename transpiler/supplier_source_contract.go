package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func init() { registerIntrinsicOwner("java.util.function.Supplier", true) }

// The independently retained parse root, actual registered scope and written
// header must agree before a source declaration can donate a Supplier view.
func supplierSourceDeclarationCtx(owner *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if certificate, anonymous := anonymousFunctionalLookup(owner, ctx); anonymous {
		if !anonymousFunctionalValid(certificate, ctx, false) {
			return Ctx{}, false
		}
		declaring := certificate.lexical.Clone()
		declaring.currentClass = certificate.scope
		declaring.localScope = nil
		return declaring, true
	}
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
		return Ctx{}, false
	}
	file := findFileScopeForClassScope(owner, ctx)
	node := owner.Class.DeclarationNode
	if file == nil || file.ParsedRoot() == nil || !receiverNodeWithin(node, file.ParsedRoot()) || node.EndByte() > uint32(len(file.Source)) {
		return Ctx{}, false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != owner.Class.OriginalName {
		return Ctx{}, false
	}
	var written []string
	if superclass := node.ChildByFieldName("superclass"); superclass != nil {
		types := supplierHeaderTypeNodes(superclass)
		if len(types) != 1 {
			return Ctx{}, false
		}
		written = append(written, types[0].Content(file.Source))
	} else {
		written = append(written, "")
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "super_interfaces" || child.Type() == "interfaces" || child.Type() == "extends_interfaces" {
			for _, typ := range supplierHeaderTypeNodes(child) {
				written = append(written, typ.Content(file.Source))
			}
		}
	}
	metadata := append([]string{owner.Superclass}, owner.ImplementedInterfaces...)
	if len(written) != len(metadata) {
		return Ctx{}, false
	}
	for i := range written {
		if written[i] != metadata[i] {
			return Ctx{}, false
		}
	}
	declared := owner.DeclaredTypeParameters
	if declared == nil {
		return Ctx{}, false
	}
	var parameterNames []string
	if parameters := node.ChildByFieldName("type_parameters"); parameters != nil {
		for _, parameter := range nodeutil.NamedChildrenOf(parameters) {
			if parameter.Type() == "type_parameter" && parameter.NamedChild(0) != nil {
				parameterNames = append(parameterNames, parameter.NamedChild(0).Content(file.Source))
			}
		}
	}
	if len(parameterNames) != len(declared) || len(declared) > len(owner.TypeParameters) {
		return Ctx{}, false
	}
	for i, name := range parameterNames {
		p := declared[i]
		physical := owner.TypeParameters[len(owner.TypeParameters)-len(declared)+i]
		if p.Declaration == nil || p.Name != name || p.Declaration.SourceName != name || physical.Declaration != p.Declaration {
			return Ctx{}, false
		}
	}
	declaring := ctx.Clone()
	declaring.currentFile = nil
	declaring.localScope = nil
	declaring.localClasses = nil
	declaring.syntheticTypeParameters = nil
	declaring = classHeaderTypeCtx(owner, declaring)
	return declaring, declaring.currentFile == file
}

func supplierHeaderTypeNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	switch node.Type() {
	case "type_identifier", "scoped_type_identifier", "generic_type", "array_type", "annotated_type":
		return []*sitter.Node{node}
	}
	var result []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		result = append(result, supplierHeaderTypeNodes(child)...)
	}
	return result
}

// Carried binders must belong to a containing source declaration, while this
// class's own ABI suffix retains its independently parsed declarations.
func supplierSourceParameterOwned(owner *symbol.ClassScope, parameter *symbol.TypeParamDeclaration, ctx Ctx) bool {
	if parameter == nil {
		return false
	}
	for _, candidate := range allSourceClassScopes() {
		declaring, owned := supplierSourceDeclarationCtx(candidate, ctx)
		if !owned {
			continue
		}
		if receiverNodeWithin(owner.Class.DeclarationNode, candidate.Class.DeclarationNode) {
			for _, p := range candidate.DeclaredTypeParameters {
				if p.Declaration == parameter {
					return true
				}
			}
		}
		for _, method := range candidate.Methods {
			if method == nil || method.DeclarationNode == nil || !receiverNodeWithin(method.DeclarationNode, declaring.currentFile.ParsedRoot()) || !receiverNodeWithin(owner.Class.DeclarationNode, method.DeclarationNode) {
				continue
			}
			for _, p := range method.TypeParameters {
				if p.Declaration == parameter {
					return true
				}
			}
		}
	}
	return false
}

func supplierSourceOriginOwned(origin inferredJavaTypeOrigin, parameters map[*symbol.TypeParamDeclaration]bool, ctx Ctx) bool {
	if !completeJavaTypeOrigin(origin) || origin.wildcard {
		return false
	}
	if origin.parameter != nil && !parameters[origin.parameter] {
		return false
	}
	if origin.nominalScope != nil {
		if _, owned := supplierSourceDeclarationCtx(origin.nominalScope, ctx); !owned {
			return false
		}
	}
	for _, argument := range origin.arguments {
		if !supplierSourceOriginOwned(argument, parameters, ctx) {
			return false
		}
	}
	return true
}

// Keep instantiated edges as declaration-bound origins. Display-name
// substitution used for code emission is independently checked against them.
func supplierSourceContractProof(scope *symbol.ClassScope, c sourceNativeFunctionalContract, ctx Ctx) (map[*symbol.ClassScope]map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin, bool) {
	declaring, owned := supplierSourceDeclarationCtx(scope, ctx)
	if !owned || c.family != "Supplier" || len(c.arguments) != 1 || len(c.parameters) != 0 || c.method != "get" {
		return nil, false
	}
	parameters := map[*symbol.TypeParamDeclaration]bool{}
	for _, p := range scope.TypeParameters {
		if !supplierSourceParameterOwned(scope, p.Declaration, ctx) {
			return nil, false
		}
		parameters[p.Declaration] = true
	}
	expected := declaredJavaTypeOrigin(symbol.JavaType{Original: c.result, TypeParameterBindings: symbol.VisibleTypeParamBindings(scope.TypeParameters)}, declaring)
	if !supplierSourceOriginOwned(expected, parameters, ctx) {
		return nil, false
	}
	bindingsByOwner := map[*symbol.ClassScope]map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
	argumentsByOwner := map[*symbol.ClassScope][]inferredJavaTypeOrigin{}
	active := map[*symbol.ClassScope]bool{}
	found := false
	var visit func(*symbol.ClassScope, []inferredJavaTypeOrigin) bool
	visit = func(owner *symbol.ClassScope, arguments []inferredJavaTypeOrigin) bool {
		if active[owner] {
			return false
		}
		if previous, seen := argumentsByOwner[owner]; seen {
			if len(previous) != len(arguments) {
				return false
			}
			for i := range previous {
				if !sameJavaTypeOrigin(previous[i], arguments[i]) {
					return false
				}
			}
			return true
		}
		header, owned := supplierSourceDeclarationCtx(owner, ctx)
		if !owned || len(arguments) != len(owner.TypeParameters) {
			return false
		}
		bindings := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
		for i, p := range owner.TypeParameters {
			if p.Declaration == nil || !supplierSourceParameterOwned(owner, p.Declaration, ctx) || !supplierSourceOriginOwned(arguments[i], parameters, ctx) {
				return false
			}
			bindings[p.Declaration] = arguments[i]
		}
		argumentsByOwner[owner] = arguments
		bindingsByOwner[owner] = bindings
		active[owner] = true
		defer delete(active, owner)
		for _, written := range append([]string{owner.Superclass}, owner.ImplementedInterfaces...) {
			if strings.TrimSpace(written) == "" {
				continue
			}
			edge := declaredJavaTypeOrigin(symbol.JavaType{Original: written, TypeParameterBindings: symbol.VisibleTypeParamBindings(owner.TypeParameters)}, header)
			edge = substituteJavaTypeOrigin(edge, bindings)
			if !supplierSourceOriginOwned(edge, parameters, ctx) || edge.arrayRank != 0 || edge.parameter != nil {
				return false
			}
			if edge.externalName == "java.util.function.Supplier" {
				if len(edge.arguments) != 1 || !sameJavaTypeOrigin(edge.arguments[0], expected) {
					return false
				}
				found = true
			} else if edge.nominalScope != nil && !visit(edge.nominalScope, edge.arguments) {
				return false
			}
		}
		return true
	}
	if !visit(scope, classParameterTypeOrigins(scope)) || !found {
		return nil, false
	}
	return bindingsByOwner, true
}

func supplierSourceContractOwned(scope *symbol.ClassScope, c sourceNativeFunctionalContract, ctx Ctx) bool {
	_, owned := supplierSourceContractProof(scope, c, ctx)
	return owned
}

func supplierSourceSelectedMethodOwned(scope, owner *symbol.ClassScope, method *symbol.Definition, c sourceNativeFunctionalContract, ctx Ctx) bool {
	if certificate, anonymous := anonymousFunctionalLookup(scope, ctx); anonymous {
		return certificate != nil && certificate.family == "Supplier" && anonymousFunctionalValid(certificate, ctx, true) &&
			owner == certificate.scope && method == certificate.selected && anonymousFunctionalContractEqual(certificate, c)
	}
	if _, anonymous := anonymousFunctionalLookup(owner, ctx); anonymous {
		return false
	}
	bindings, proved := supplierSourceContractProof(scope, c, ctx)
	declaring, owned := supplierSourceDeclarationCtx(owner, ctx)
	if !proved || !owned || method == nil || method.DeclarationNode == nil || method.IsStatic || method.IsPrivate || !method.HasBody || method.OriginalName != "get" || len(method.Parameters) != 0 || len(method.TypeParameters) != 0 {
		return false
	}
	ownerBindings, reachable := bindings[owner]
	if !reachable || !receiverNodeWithin(method.DeclarationNode, owner.Class.DeclarationNode) || !receiverNodeWithin(method.DeclarationNode, declaring.currentFile.ParsedRoot()) {
		return false
	}
	registered := false
	for _, candidate := range owner.Methods {
		registered = registered || candidate == method
	}
	node := method.DeclarationNode
	name, typ := node.ChildByFieldName("name"), node.ChildByFieldName("type")
	if !registered || node.ChildByFieldName("body") == nil || name == nil || typ == nil || name.Content(declaring.currentFile.Source) != "get" || typ.Content(declaring.currentFile.Source) != method.OriginalType {
		return false
	}
	actual := substituteJavaTypeOrigin(declaredJavaTypeOrigin(definitionDeclaredJavaType(method), declaring), ownerBindings)
	root, _ := supplierSourceDeclarationCtx(scope, ctx)
	expected := declaredJavaTypeOrigin(symbol.JavaType{Original: c.result, TypeParameterBindings: symbol.VisibleTypeParamBindings(scope.TypeParameters)}, root)
	if actual.externalName == "" && actual.nominalScope == nil && actual.parameter == nil && actual.arrayRank == 0 {
		return false
	}
	parameters := map[*symbol.TypeParamDeclaration]bool{}
	for _, p := range scope.TypeParameters {
		parameters[p.Declaration] = true
	}
	if !supplierSourceOriginOwned(actual, parameters, ctx) {
		return false
	}
	if sameJavaTypeOrigin(actual, expected) || supplierNativeMapResultAssignable(actual, expected, declaring) {
		return true
	}
	_, _, applicable, handled := declaredOriginConversion(actual, expected, declaring)
	return handled && applicable
}

// Native map ancestry already has a canonical declaration contract independent
// of the Collection-only generic-parent metadata. Its invariant slots still
// require exact origin identities; a same-spelled binder is not a proof.
func supplierNativeMapResultAssignable(actual, expected inferredJavaTypeOrigin, ctx Ctx) bool {
	if actual.nominalScope != nil || expected.nominalScope != nil || actual.parameter != nil || expected.parameter != nil || actual.arrayRank != 0 || expected.arrayRank != 0 || actual.wildcard || expected.wildcard || len(actual.arguments) != 2 || len(expected.arguments) != 2 {
		return false
	}
	a := canonicalMapReferenceOwner(actual.externalName, ctx)
	b := canonicalMapReferenceOwner(expected.externalName, ctx)
	if a == "" || b == "" || a != actual.externalName || b != expected.externalName || !mapOwnerReaches(a, b) {
		return false
	}
	for i := range actual.arguments {
		if !sameJavaTypeOrigin(actual.arguments[i], expected.arguments[i]) {
			return false
		}
	}
	return true
}
