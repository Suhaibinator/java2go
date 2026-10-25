package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
)

// Only actual AbstractList class declarations donate their unbounded scalar
// owner binders. This reuses the existing native-body emitter, without adding
// native families to general callable, field, or container erasure eligibility.
func sourceAbstractListActualMethodOwned(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if method == nil || method.RuntimeDefault || method.Constructor || method.IsStatic || method.DeclarationNode == nil || owner == nil || owner.Class == nil || ctx.currentFile == nil {
		return false
	}
	node := method.DeclarationNode
	if node.Parent() == nil || node.Parent().Parent() == nil || !node.Parent().Parent().Equal(owner.Class.DeclarationNode) || node.Type() != "method_declaration" || !threadSourceNodeWithin(node, owner.Class.DeclarationNode) || !threadSourceNodeWithin(node, ctx.currentFile.ParsedRoot()) {
		return false
	}
	name, typ := node.ChildByFieldName("name"), node.ChildByFieldName("type")
	if name == nil || typ == nil || name.Content(ctx.currentFile.Source) != method.OriginalName || typ.Content(ctx.currentFile.Source) != method.OriginalType || (node.ChildByFieldName("body") != nil) != method.HasBody || method.IsStatic != declarationHasModifier(node, "static") || method.IsPrivate != declarationHasModifier(node, "private") || !sourceArrayListParameterRanksExact(method, ctx) {
		return false
	}
	count := 0
	for _, actual := range owner.Methods {
		if actual == method {
			count++
		}
	}
	return count == 1
}

func sourceAbstractListVirtualBody(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceNativeVirtualBodyABI {
	if method == nil || method.IsStatic || method.Constructor || len(method.TypeParameters) != 0 || sourceAbstractListRoot(owner, ctx) == nil {
		return nil
	}
	declaring, owned := sourceAbstractListDeclaration(owner, ctx)
	if !owned {
		return nil
	}
	attached := 0
	for _, actual := range owner.Methods {
		if actual == method {
			attached++
		}
	}
	if attached != 1 {
		return nil
	}
	if method.RuntimeDefault {
		if method.DeclarationNode != nil {
			return nil
		}
		if _, known := sourceAbstractListDefaultSpec(method, classScopeCtx(owner, declaring)); !known {
			return nil
		}
	} else if !sourceAbstractListActualMethodOwned(method, owner, declaring) || method.IsStatic != declarationHasModifier(method.DeclarationNode, "static") || method.IsPrivate != declarationHasModifier(method.DeclarationNode, "private") {
		return nil
	}
	declarations := map[*symbol.TypeParamDeclaration]bool{}
	for _, p := range owner.TypeParameters {
		if p.Declaration != nil && (len(p.Bounds) == 0 || len(p.Bounds) == 1 && qualifyDeclaredNominalReference(p.Bounds[0].Original, declaring) == "java.lang.Object") {
			declarations[p.Declaration] = true
		}
	}
	if len(declarations) == 0 {
		return nil
	}
	// Refuse copied or altered binding sidecars instead of erasing a different
	// written type just because it points at an otherwise genuine declaration.
	for _, def := range append(append([]*symbol.Definition(nil), method.Parameters...), method) {
		if def.DirectTypeParameter != nil && declarations[def.DirectTypeParameter] {
			if _, actual := directOwnerTypeParameterForDefinition(owner, def); !actual {
				return nil
			}
		}
	}
	plan := &sourceNativeVirtualBodyABI{owner: owner, method: method, file: declaring.currentFile, declarations: declarations, erasedResult: method.OriginalType}
	changed := false
	for _, parameter := range method.Parameters {
		typ := parameter.OriginalType
		if sourceNativeVirtualScalarDefinition(parameter, declarations) {
			typ = "java.lang.Object"
			changed = true
		}
		plan.erasedParameters = append(plan.erasedParameters, typ)
	}
	if sourceNativeVirtualScalarDefinition(method, declarations) {
		plan.erasedResult = "java.lang.Object"
		changed = true
	}
	// Even an Object/void signature can contain owner-E locals, casts or
	// field writes. Preserve its declared callable signature while rendering
	// the actual instance body against the same erased owner storage.
	if !changed && (method.RuntimeDefault || !method.HasBody) {
		return nil
	}
	return plan
}
func sourceAbstractListAttestDefaultBindings(scope *symbol.ClassScope, ctx Ctx) {
	if !sourceAbstractListDirectRoot(scope, ctx) {
		return
	}
	bindings := sourceNativeVirtualBindings(scope)
	for _, method := range scope.Methods {
		if method == nil || !method.RuntimeDefault || method.DeclarationNode != nil {
			continue
		}
		if _, known := sourceAbstractListDefaultSpec(method, ctx); !known {
			continue
		}
		for _, definition := range append(append([]*symbol.Definition(nil), method.Parameters...), method) {
			base, rank := javaArrayTypeParts(strings.TrimSpace(definition.OriginalType))
			name, args := parseJavaTypeString(base)
			if rank != 0 || len(args) != 0 {
				continue
			}
			declaration := bindings[name]
			if declaration == nil {
				continue
			}
			definition.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{name: declaration}
			definition.DirectTypeParameter = declaration
		}
	}
}
func sourceAbstractListInstantiatedMethodType(javaType string, owner, receiver *symbol.ClassScope, ctx Ctx) string {
	arguments := mapClassTypeArgumentStringsToAncestor(receiver, receiver.GoTypeParameterNames(), owner, ctx)
	bindings := map[string]string{}
	for i, p := range owner.TypeParameters {
		if i < len(arguments) {
			bindings[p.Name] = arguments[i]
			bindings[p.EmittedName()] = arguments[i]
		}
	}
	return substituteJavaTypeParams(qualifyJavaTypeInDeclaringContext(javaType, owner), bindings)
}
func sourceAbstractListSlotMatches(method *symbol.Definition, owner, receiver *symbol.ClassScope, ctx Ctx) bool {
	if method == nil {
		return false
	}
	if !method.RuntimeDefault {
		declaring, owned := sourceAbstractListDeclaration(owner, ctx)
		if !owned || !sourceAbstractListActualMethodOwned(method, owner, declaring) || method.IsStatic != declarationHasModifier(method.DeclarationNode, "static") || method.IsPrivate != declarationHasModifier(method.DeclarationNode, "private") {
			return false
		}
	}
	name := method.OriginalName
	element := sourceAbstractListElement(receiver, ctx)
	root := sourceAbstractListRoot(receiver, ctx)
	if root != nil {
		element = sourceAbstractListInstantiatedMethodType(element, root, receiver, ctx)
	}
	var expected []string
	switch name {
	case "get":
		expected = []string{"int"}
	case "set":
		expected = []string{"int", element}
	case "add":
		if len(method.Parameters) == 1 {
			expected = []string{element}
		} else {
			expected = []string{"int", element}
		}
	case "remove":
		if len(method.Parameters) == 1 && javaInferenceSameType(sourceAbstractListInstantiatedMethodType(method.Parameters[0].OriginalType, owner, receiver, ctx), "int", ctx) {
			expected = []string{"int"}
		} else {
			expected = []string{"java.lang.Object"}
		}
	case "indexOf", "lastIndexOf":
		expected = []string{"java.lang.Object"}
	case "listIterator":
		expected = []string{"int"}
	case "subList", "removeRange":
		expected = []string{"int", "int"}
	case "addAll":
		if len(method.Parameters) == 2 {
			expected = []string{"int", "java.util.Collection<?>"}
		} else {
			expected = []string{"java.util.Collection<?>"}
		}
	case "removeAll", "retainAll":
		expected = []string{"java.util.Collection<?>"}
	case "toArray":
		if len(method.Parameters) == 0 {
			return len(method.TypeParameters) == 0
		}
		if len(method.Parameters) != 1 || len(method.TypeParameters) > 1 {
			return false
		}
		actual := method.Parameters[0].OriginalType
		base, rank := javaArrayTypeParts(actual)
		if rank != 1 {
			return false
		}
		if len(method.TypeParameters) == 1 {
			return method.Parameters[0].TypeParameterBindings[base] == method.TypeParameters[0].Declaration && (len(method.TypeParameters[0].Bounds) == 0 || len(method.TypeParameters[0].Bounds) == 1 && qualifyDeclaredNominalReference(method.TypeParameters[0].Bounds[0].Original, classScopeCtx(owner, ctx)) == "java.lang.Object")
		}
		return javaInferenceSameType(actual, "java.lang.Object[]", classScopeCtx(owner, ctx))
	default:
		return false
	}
	if len(method.TypeParameters) != 0 || len(expected) != len(method.Parameters) {
		return false
	}
	for i, p := range expected {
		actual := sourceAbstractListInstantiatedMethodType(method.Parameters[i].OriginalType, owner, receiver, ctx)
		if strings.Contains(p, "<?>") {
			a, ar := parseJavaTypeString(actual)
			b, _ := parseJavaTypeString(p)
			if a != b || len(ar) > 1 {
				return false
			}
		} else if !javaInferenceSameType(actual, p, ctx) {
			return false
		}
	}
	return true
}

// A bare unbounded owner field has the same Object descriptor as the owned
// native source bodies that access it. Arrays and nested generic storage retain
// their existing representations; no other native family gains field erasure.
func sourceAbstractListScalarFieldErasure(owner *symbol.ClassScope, definition *symbol.Definition, ctx Ctx) (string, bool) {
	if definition == nil || definition.IsStatic || sourceAbstractListRoot(owner, ctx) == nil || !sourceArrayListNativeDeclaredScalar(definition, owner, ctx) {
		return "", false
	}
	use, owned := directOwnerTypeParameterForDefinition(owner, definition)
	if !owned {
		return "", false
	}
	declaring, known := sourceAbstractListDeclaration(owner, ctx)
	if !known {
		return "", false
	}
	if len(use.parameter.Bounds) != 0 && (len(use.parameter.Bounds) != 1 || qualifyDeclaredNominalReference(use.parameter.Bounds[0].Original, declaring) != "java.lang.Object") {
		return "", false
	}
	return "java.lang.Object", true
}

// Body erasure and callable descriptor erasure are separate contracts.
// A signature-free helper needs an erased body without donating a new bridge.
func sourceAbstractListErasedSignature(plan *sourceNativeVirtualBodyABI) bool {
	if plan == nil {
		return false
	}
	if sourceNativeVirtualScalarDefinition(plan.method, plan.declarations) {
		return true
	}
	for _, parameter := range plan.method.Parameters {
		if sourceNativeVirtualScalarDefinition(parameter, plan.declarations) {
			return true
		}
	}
	return false
}
