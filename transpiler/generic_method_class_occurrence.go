package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

// A Class argument describes the represented Java type, not a different Class
// allocation layout. Reuse that physical proof only for an actual method's
// declaration-owned slot. A negative index of -1 denotes its result.
func canonicalGenericMethodClassSlot(method *symbol.Definition, index int, _ Ctx) bool {
	if method == nil || index < -1 || index >= len(method.Parameters) {
		return false
	}
	declaring, registered := genericMethodOwnerContext(method)
	owner := declaring.currentClass
	if !registered || owner == nil || owner.Class == nil || declaring.currentFile == nil {
		return false
	}
	file := declaring.currentFile
	classNode := owner.Class.DeclarationNode
	methodNode := method.DeclarationNode
	if file.ParsedRoot() == nil || classNode == nil || methodNode == nil || methodNode.Type() != "method_declaration" || methodNode.HasError() || !threadSourceNodeWithin(classNode, file.ParsedRoot()) || !sourceReflectionDeclaredMethod(owner, method) {
		return false
	}
	source := file.Source
	if methodNode.EndByte() > uint32(len(source)) {
		return false
	}
	className := classNode.ChildByFieldName("name")
	methodName := methodNode.ChildByFieldName("name")
	if className == nil || methodName == nil || className.Content(source) != owner.Class.OriginalName || methodName.Content(source) != method.OriginalName {
		return false
	}

	// The declaration identities are carried by the registered symbols, while the
	// retained header proves their Java slots. An owner T cannot stand in for a
	// distinct method T, even if counterfeit metadata shares all its spellings.
	if !genericMethodClassHeaderOwned(method, source) {
		return false
	}
	syntax := symbol.ExtractTypeParameters(methodNode.ChildByFieldName("type_parameters"), source)
	if len(syntax) == 0 || len(syntax) != len(method.TypeParameters) {
		return false
	}
	seen := make(map[*symbol.TypeParamDeclaration]bool, len(syntax))
	for i, parameter := range method.TypeParameters {
		if parameter.Declaration == nil || parameter.Declaration.SourceName != parameter.Name || parameter.Name != syntax[i].Name || seen[parameter.Declaration] || len(parameter.Bounds) != len(syntax[i].Bounds) {
			return false
		}
		seen[parameter.Declaration] = true
		for j, bound := range parameter.Bounds {
			if bound.Original != syntax[i].Bounds[j].Original {
				return false
			}
		}
		for scope := owner; scope != nil; scope = scope.Enclosing {
			for _, carried := range scope.TypeParameters {
				if carried.Declaration == parameter.Declaration {
					return false
				}
			}
		}
	}

	declaring.localScope = method
	expectedBindings := symbol.VisibleTypeParamBindings(symbol.MergeTypeParams(owner.TypeParameters, method.TypeParameters))
	coherent := func(member *symbol.Definition, typ, name *sitter.Node) bool {
		if member == nil || typ == nil || !threadSourceNodeWithin(typ, methodNode) || typ.Content(source) != member.OriginalType || len(member.TypeParameterBindings) != len(expectedBindings) {
			return false
		}
		if name != nil && name.Content(source) != member.OriginalName {
			return false
		}
		for key, identity := range expectedBindings {
			if identity == nil || member.TypeParameterBindings[key] != identity || visibleTypeParameterDeclarationForJavaType(key, declaring) != identity {
				return false
			}
		}
		return true
	}
	formals := nodeutil.SemanticNamedChildrenOf(methodNode.ChildByFieldName("parameters"))
	if len(formals) != len(method.Parameters) {
		return false
	}
	// Today's Definition omits post-declarator dimensions. Decline that spelling
	// rather than use its scalar type child to certify another physical rank.
	for _, child := range nodeutil.SemanticNamedChildrenOf(methodNode) {
		if child.Type() == "dimensions" {
			return false
		}
	}
	result := methodNode.ChildByFieldName("type")
	if !coherent(method, result, nil) {
		return false
	}
	occurrence := result
	member := method
	for i, formal := range formals {
		if formal.Type() != "formal_parameter" {
			return false
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(formal) {
			if child.Type() == "dimensions" {
				return false
			}
		}
		typ, name := nodeutil.JavaParameterNodes(formal)
		if name == nil || !coherent(method.Parameters[i], typ, name) {
			return false
		}
		if index == i {
			occurrence = typ
			member = method.Parameters[i]
		}
	}
	return canonicalClassArgumentIndependent(member.OriginalType, declaring, occurrence)
}

// A fresh anonymous method is not a registered owner. Its separate certificate
// starts at the actual creation and alpha-matches complete declaration origins
// to the registered source interface entry before publishing any body context.
func canonicalAnonymousGenericClassBody(creation, objectType, body, declaration *sitter.Node, implementation, selected *symbol.Definition, ctx Ctx) *genericInterfaceMethodBodyABI {
	if creation == nil || objectType == nil || body == nil || declaration == nil || implementation == nil || implementation.DeclarationNode == nil || len(implementation.TypeParameters) == 0 || selected == nil || ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentFile == nil {
		return nil
	}
	owner := ctx.currentClass
	file := ctx.currentFile
	if findFileScopeForClassScope(owner, ctx) != file || file.ParsedRoot() == nil || !threadSourceNodeWithin(owner.Class.DeclarationNode, file.ParsedRoot()) || !threadSourceNodeWithin(creation, file.ParsedRoot()) || creation.Type() != "object_creation_expression" || creation.ChildByFieldName("type") == nil || !objectType.Equal(creation.ChildByFieldName("type")) || body.Parent() == nil || !body.Parent().Equal(creation) || body.Type() != "class_body" || declaration.Parent() == nil || !declaration.Parent().Equal(body) || !implementation.DeclarationNode.Equal(declaration) {
		return nil
	}
	// No class/method capture substitution is inferred for this new admission.
	if len(owner.TypeParameters) != 0 || len(ctx.syntheticTypeParameters) != 0 {
		return nil
	}
	nearest := false
	for parent := creation.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Equal(owner.Class.DeclarationNode) {
			nearest = true
			break
		}
		switch parent.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "object_creation_expression":
			return nil
		case "method_declaration", "constructor_declaration":
			if len(nodeutil.SemanticNamedChildrenOf(parent.ChildByFieldName("type_parameters"))) != 0 {
				return nil
			}
			actual := false
			for _, method := range owner.Methods {
				if method != nil && method.DeclarationNode != nil && method.DeclarationNode.Equal(parent) {
					actual = ctx.localScope == method
					break
				}
			}
			if !actual {
				return nil
			}
		}
	}
	if !nearest {
		return nil
	}
	members := nodeutil.SemanticNamedChildrenOf(body)
	if len(members) != 1 || !members[0].Equal(declaration) || declaration.Type() != "method_declaration" || declaration.HasError() || declaration.EndByte() > uint32(len(file.Source)) {
		return nil
	}
	if declaration.ChildByFieldName("body") == nil || declarationHasModifier(declaration, "synchronized") || anonymousSAMBodyNeedsObjectIdentity(declaration.ChildByFieldName("body"), file.Source) {
		return nil
	}
	actualName := declaration.ChildByFieldName("name")
	if actualName == nil || actualName.Content(file.Source) != implementation.OriginalName || implementation.OriginalName != selected.OriginalName {
		return nil
	}
	lookup := ctx.Clone()
	lookup.importAliases = nil
	lookup.usedImports = nil
	lookup.sourceOwnership = nil
	lookup.genericFamilies = nil
	base, args := parseJavaTypeString(objectType.Content(file.Source))
	target := resolveClassScopeByQualifiedName(lookup, base)
	if len(args) != 0 || target == nil || !target.IsInterface || len(target.TypeParameters) != 0 {
		return nil
	}
	method, _ := resolveFunctionalInterfaceMethod(lookup, objectType.Content(file.Source))
	if method != selected {
		return nil
	}
	declaring, registered := genericMethodOwnerContext(selected)
	if !registered || declaring.currentClass != target || !sourceReflectionDeclaredMethod(target, selected) {
		return nil
	}
	entry := planGenericMethodEntryABI(target, selected, declaring)
	if entry == nil {
		return nil
	}
	if !genericMethodClassHeaderOwned(implementation, file.Source) {
		return nil
	}
	syntax := symbol.ExtractTypeParameters(declaration.ChildByFieldName("type_parameters"), file.Source)
	if len(syntax) == 0 || len(syntax) != len(implementation.TypeParameters) || len(syntax) != len(selected.TypeParameters) {
		return nil
	}
	alpha := map[*symbol.TypeParamDeclaration]*symbol.TypeParamDeclaration{}
	for index, parameter := range implementation.TypeParameters {
		contract := selected.TypeParameters[index]
		if parameter.Declaration == nil || contract.Declaration == nil || parameter.Declaration == contract.Declaration || parameter.Declaration.SourceName != parameter.Name || parameter.Name != syntax[index].Name || len(parameter.Bounds) != 0 || len(syntax[index].Bounds) != 0 || alpha[parameter.Declaration] != nil {
			return nil
		}
		for scope := owner; scope != nil; scope = scope.Enclosing {
			for _, outer := range scope.TypeParameters {
				if outer.Declaration == parameter.Declaration {
					return nil
				}
			}
		}
		// A freshly extracted anonymous binder cannot also own a registered
		// class or method slot, including a different contract slot with the
		// same spelling. Read the existing declarations; add no registry.
		for _, declaredOwner := range allSourceClassScopes() {
			if classScopeCarriesTypeParameterDeclaration(declaredOwner, parameter.Declaration) {
				return nil
			}
			for _, declaredMethod := range declaredOwner.Methods {
				if declaredMethod == nil {
					continue
				}
				for _, declaredParameter := range declaredMethod.TypeParameters {
					if declaredParameter.Declaration == parameter.Declaration {
						return nil
					}
				}
			}
		}
		alpha[parameter.Declaration] = contract.Declaration
	}
	expected := symbol.VisibleTypeParamBindings(implementation.TypeParameters)
	local := lookup.Clone()
	local.localScope = implementation
	coherent := func(member *symbol.Definition, typ, name *sitter.Node) bool {
		if member == nil || typ == nil || !threadSourceNodeWithin(typ, declaration) || typ.Content(file.Source) != member.OriginalType || len(member.TypeParameterBindings) != len(expected) || member.DirectTypeParameter != symbol.DirectTypeParamForJavaType(member.OriginalType, implementation.TypeParameters) {
			return false
		}
		if name != nil && name.Content(file.Source) != member.OriginalName {
			return false
		}
		for key, identity := range expected {
			if identity == nil || member.TypeParameterBindings[key] != identity || visibleTypeParameterDeclarationForJavaType(key, local) != identity {
				return false
			}
		}
		return true
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(declaration) {
		if child.Type() == "dimensions" {
			return nil
		}
	}
	result := declaration.ChildByFieldName("type")
	if !coherent(implementation, result, nil) {
		return nil
	}
	formals := nodeutil.SemanticNamedChildrenOf(declaration.ChildByFieldName("parameters"))
	if len(formals) != len(implementation.Parameters) || len(formals) != len(selected.Parameters) {
		return nil
	}
	hasClass := canonicalClassArgumentIndependent(implementation.OriginalType, local, result)
	var remap func(inferredJavaTypeOrigin) inferredJavaTypeOrigin
	remap = func(origin inferredJavaTypeOrigin) inferredJavaTypeOrigin {
		if mapped := alpha[origin.parameter]; mapped != nil {
			origin.parameter = mapped
		}
		args := make([]inferredJavaTypeOrigin, len(origin.arguments))
		for i, arg := range origin.arguments {
			args[i] = remap(arg)
		}
		origin.arguments = args
		return origin
	}
	sameSlot := func(actual, contract *symbol.Definition) bool {
		a := remap(anonymousClassMethodDeclaredOrigin(definitionDeclaredJavaType(actual), local))
		b := anonymousClassMethodDeclaredOrigin(definitionDeclaredJavaType(contract), declaring)
		return sourceOriginAncestryComplete(a) && sourceOriginAncestryComplete(b) && sameJavaTypeOrigin(a, b)
	}
	if !sameSlot(implementation, selected) {
		return nil
	}
	for index, formal := range formals {
		if formal.Type() != "formal_parameter" {
			return nil
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(formal) {
			if child.Type() == "dimensions" {
				return nil
			}
		}
		typ, name := nodeutil.JavaParameterNodes(formal)
		member := implementation.Parameters[index]
		if name == nil || !coherent(member, typ, name) || !sameSlot(member, selected.Parameters[index]) {
			return nil
		}
		hasClass = hasClass || canonicalClassArgumentIndependent(member.OriginalType, local, typ)
	}
	if !hasClass {
		return nil
	}
	return planGenericInterfaceMethodBodyABI(entry, implementation, declaration, local)
}

// The inference origin reader projects lower wildcards for expression use.
// Signature comparison must retain the actual bound's declaration instead.
// Variance itself is still checked by the unchanged body's alpha signature.
func anonymousClassMethodDeclaredOrigin(typ symbol.JavaType, ctx Ctx) inferredJavaTypeOrigin {
	raw := strings.TrimSpace(typ.Original)
	if raw == "?" {
		return inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: raw, wildcard: true}
	}
	if strings.HasPrefix(raw, "?") {
		rest := strings.TrimSpace(strings.TrimPrefix(raw, "?"))
		words := strings.Fields(rest)
		if len(words) < 2 || (words[0] != "extends" && words[0] != "super") {
			return inferredJavaTypeOrigin{javaType: raw, unresolved: true}
		}
		typ.Original = strings.TrimSpace(strings.TrimPrefix(rest, words[0]))
		origin := anonymousClassMethodDeclaredOrigin(typ, ctx)
		origin.wildcard = true
		return origin
	}

	origin := declaredJavaTypeOrigin(typ, ctx)
	component, _ := javaArrayTypeParts(raw)
	_, args := parseJavaTypeString(component)
	// Declaration analysis can prepend actual carried enclosing arguments to
	// the explicit syntax arguments. Keep those facts when restoring wildcard
	// bounds; replacing the entire list would erase enclosing-instance origins.
	if len(args) > len(origin.arguments) {
		origin.unresolved = true
		return origin
	}
	carried := len(origin.arguments) - len(args)
	arguments := append([]inferredJavaTypeOrigin(nil), origin.arguments[:carried]...)
	for _, arg := range args {
		arguments = append(arguments, anonymousClassMethodDeclaredOrigin(symbol.JavaType{Original: arg, TypeParameterBindings: typ.TypeParameterBindings}, ctx))
	}
	origin.arguments = arguments
	return origin
}

// Header authority is the actual grammar role, independently of the symbol
// extractor. Unsupported annotated type parameters/bounds stay unproved: an
// annotation is not a type-variable name or an implicit Object bound.
func genericMethodClassHeaderOwned(method *symbol.Definition, source []byte) bool {
	if method == nil || method.DeclarationNode == nil {
		return false
	}
	parameters := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("type_parameters"))
	if len(parameters) != len(method.TypeParameters) {
		return false
	}
	var plain func(*sitter.Node) bool
	plain = func(node *sitter.Node) bool {
		if node == nil {
			return false
		}
		switch node.Type() {
		case "annotation", "marker_annotation", "annotated_type":
			return false
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			if !plain(child) {
				return false
			}
		}
		return true
	}
	for index, node := range parameters {
		if node.Type() != "type_parameter" || !plain(node) {
			return false
		}
		var name *sitter.Node
		var bounds []*sitter.Node
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			switch child.Type() {
			case "type_identifier":
				if name != nil {
					return false
				}
				name = child
			case "type_bound":
				for _, bound := range nodeutil.SemanticNamedChildrenOf(child) {
					switch bound.Type() {
					case "type_identifier", "scoped_type_identifier", "generic_type", "array_type":
					default:
						return false
					}
					bounds = append(bounds, bound)
				}
			default:
				return false
			}
		}
		parameter := method.TypeParameters[index]
		if name == nil || name.EndByte() > uint32(len(source)) || name.Content(source) != parameter.Name || len(bounds) != len(parameter.Bounds) {
			return false
		}
		for i, bound := range bounds {
			if bound.EndByte() > uint32(len(source)) || bound.Content(source) != parameter.Bounds[i].Original {
				return false
			}
		}
	}
	return true
}
