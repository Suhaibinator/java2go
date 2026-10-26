package transpiler

import (
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

// P1 attests an actual registered, named source declaration and its Object
// storage slot. Display names and Go protocol shape never establish this edge.
func sourceArrayListDeclaration(scope *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "class_declaration" {
		return Ctx{}, false
	}
	registered := false
	for _, actual := range allSourceClassScopes() {
		if actual == scope {
			registered = true
			break
		}
	}
	if !registered {
		return Ctx{}, false
	}
	declaring, owned := supplierSourceDeclarationCtx(scope, ctx)
	if !owned || len(scope.TypeParameters) != 0 && sourceArrayListScalarOwner(scope, declaring) == nil {
		return Ctx{}, false
	}
	return declaring, true
}

func sourceArrayListObjectOrigin(origin inferredJavaTypeOrigin) bool {
	return !origin.unresolved && !origin.wildcard && origin.externalName == "java.lang.Object" && origin.nominalScope == nil && origin.parameter == nil && origin.arrayRank == 0 && len(origin.arguments) == 0
}

func sourceArrayListCanonicalEdge(javaType string, ctx Ctx) bool {
	edge := declaredJavaTypeOrigin(symbol.JavaType{Original: javaType, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
	if edge.unresolved || edge.wildcard || edge.externalName != "java.util.ArrayList" || edge.nominalScope != nil || edge.parameter != nil || edge.arrayRank != 0 {
		return false
	}
	return len(edge.arguments) == 0 || len(edge.arguments) == 1 && sourceArrayListObjectOrigin(edge.arguments[0]) || sourceArrayListScalarEdge(javaType, ctx)
}

func sourceArrayListDirectRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || !sourceArrayListSuperclassCandidate(scope.Superclass) {
		return false
	}
	header, owned := sourceArrayListDeclaration(scope, ctx)
	return owned && sourceArrayListCanonicalEdge(scope.Superclass, header) && sourceArrayListNativeConstructors(scope, header) && !sourceArrayListUncoveredOverride(scope, header)
}

func sourceArrayListRoot(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		declaring, owned := sourceArrayListDeclaration(scope, ctx)
		if !owned || sourceArrayListUncoveredOverride(scope, declaring) {
			return nil
		}
		if sourceArrayListCanonicalEdge(scope.Superclass, declaring) {
			if sourceArrayListNativeConstructors(scope, declaring) {
				return scope
			}
			return nil
		}
		edge := declaredJavaTypeOrigin(symbol.JavaType{Original: scope.Superclass}, declaring)
		parent := resolveSuperclassScopeInDeclaringContext(declaring, scope)
		if edge.unresolved || edge.wildcard || edge.parameter != nil || edge.arrayRank != 0 || len(edge.arguments) != 0 || edge.nominalScope != parent {
			return nil
		}
		scope = parent
	}
	return nil
}

// Capacity/copy overloads require independent constructor-selection evidence.
// A this(...) chain may carry arguments; its terminal native super() is empty.
func sourceArrayListNativeConstructors(scope *symbol.ClassScope, ctx Ctx) bool {
	for _, method := range scope.Methods {
		if method == nil || !method.Constructor || method.DeclarationNode == nil {
			continue
		}
		body := method.DeclarationNode.ChildByFieldName("body")
		for _, child := range nodeutil.NamedChildrenOf(body) {
			if child.Type() != "explicit_constructor_invocation" {
				continue
			}
			constructor := child.ChildByFieldName("constructor")
			if constructor == nil {
				for _, candidate := range nodeutil.NamedChildrenOf(child) {
					if candidate.Type() == "this" || candidate.Type() == "super" {
						constructor = candidate
						break
					}
				}
			}
			if constructor != nil && constructor.Content(ctx.currentFile.Source) == "super" {
				arguments := child.ChildByFieldName("arguments")
				if arguments == nil {
					for _, candidate := range nodeutil.NamedChildrenOf(child) {
						if candidate.Type() == "argument_list" {
							arguments = candidate
							break
						}
					}
				}
				if arguments != nil && nodeutil.SemanticNamedChildCount(arguments) != 0 {
					return false
				}
			}
		}
	}
	return true
}

// Platform declarations in the ArrayList/Object slot family. Parameter origin
// equality distinguishes real overrides from unrelated business overloads.
// Only add(Object) has a bound native-alias virtual entry in P1.
var sourceArrayListPlatformSlots = []struct {
	name       string
	parameters []string
}{
	{"add", []string{"java.lang.Object"}}, {"add", []string{"int", "java.lang.Object"}},
	{"get", []string{"int"}}, {"set", []string{"int", "java.lang.Object"}}, {"size", nil}, {"isEmpty", nil},
	{"contains", []string{"java.lang.Object"}}, {"indexOf", []string{"java.lang.Object"}}, {"lastIndexOf", []string{"java.lang.Object"}},
	{"remove", []string{"int"}}, {"remove", []string{"java.lang.Object"}}, {"clear", nil},
	{"addAll", []string{"java.util.Collection<?>"}}, {"addAll", []string{"int", "java.util.Collection<?>"}},
	{"containsAll", []string{"java.util.Collection<?>"}}, {"removeAll", []string{"java.util.Collection<?>"}}, {"retainAll", []string{"java.util.Collection<?>"}},
	{"iterator", nil}, {"listIterator", nil}, {"listIterator", []string{"int"}}, {"subList", []string{"int", "int"}},
	{"toArray", nil}, {"toArray", []string{"java.lang.Object[]"}}, {"toArray", []string{"java.util.function.IntFunction<?>"}},
	{"removeIf", []string{"java.util.function.Predicate<?>"}}, {"replaceAll", []string{"java.util.function.UnaryOperator<?>"}},
	{"sort", []string{"java.util.Comparator<?>"}}, {"spliterator", nil}, {"forEach", []string{"java.util.function.Consumer<?>"}},
	{"stream", nil}, {"parallelStream", nil}, {"equals", []string{"java.lang.Object"}}, {"hashCode", nil}, {"toString", nil},
	{"clone", nil}, {"trimToSize", nil}, {"ensureCapacity", []string{"int"}}, {"removeRange", []string{"int", "int"}},
}

func sourceArrayListMethodOwned(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if method == nil || method.RuntimeDefault || method.Constructor || method.IsStatic || method.IsPrivate || method.DeclarationNode == nil {
		return false
	}
	node := method.DeclarationNode
	if (node.Type() != "method_declaration" && node.Type() != "abstract_method_declaration") || !threadSourceNodeWithin(node, owner.Class.DeclarationNode) || !threadSourceNodeWithin(node, ctx.currentFile.ParsedRoot()) {
		return false
	}
	name, typ := node.ChildByFieldName("name"), node.ChildByFieldName("type")
	if name == nil || typ == nil || (node.ChildByFieldName("body") != nil) != method.HasBody || name.Content(ctx.currentFile.Source) != method.OriginalName || typ.Content(ctx.currentFile.Source) != method.OriginalType {
		return false
	}
	if !sourceArrayListParameterRanksExact(method, ctx) {
		return false
	}
	for _, actual := range owner.Methods {
		if actual == method {
			return true
		}
	}
	return false
}

func sourceArrayListSlotMatches(method *symbol.Definition, parameters []string, ctx Ctx) bool {
	if !sourceArrayListParameterRanksExact(method, ctx) || len(method.Parameters) != len(parameters) {
		return false
	}
	for i, expected := range parameters {
		actual := declaredJavaTypeOrigin(definitionDeclaredJavaType(method.Parameters[i]), ctx)
		formal := declaredJavaTypeOrigin(symbol.JavaType{Original: expected}, ctx)
		if actual.unresolved || formal.unresolved {
			return false
		}
		// Platform collection/function formals are erased in an overriding Java
		// declaration. Their written generic wildcard bounds do not change the slot.
		if expected == "java.lang.Object[]" && actual.parameter != nil && actual.arrayRank == 1 && actual.nominalScope == nil {
			continue
		}
		if strings.Contains(expected, "<?>") {
			if actual.externalName != formal.externalName || actual.parameter != nil || actual.nominalScope != nil || actual.arrayRank != 0 {
				return false
			}
		} else if !sameJavaTypeOrigin(actual, formal) {
			return false
		}
	}
	return true
}

func sourceArrayListUncoveredOverride(scope *symbol.ClassScope, ctx Ctx) bool {
	for _, method := range scope.Methods {
		if method == nil || method.RuntimeDefault || method.Constructor || method.IsStatic || method.IsPrivate {
			continue
		}
		platformName := false
		for _, slot := range sourceArrayListPlatformSlots {
			if slot.name == method.OriginalName {
				platformName = true
				break
			}
		}
		if !platformName {
			continue
		}
		if !sourceArrayListMethodOwned(method, scope, ctx) {
			return true
		}
		methodCtx := ctx
		methodCtx.localScope = method
		for _, slot := range sourceArrayListPlatformSlots {
			if slot.name != method.OriginalName || !sourceArrayListSlotMatches(method, slot.parameters, methodCtx) {
				continue
			}
			if slot.name != "add" || len(slot.parameters) != 1 || !method.HasBody || len(method.TypeParameters) != 0 || method.OriginalType != "boolean" || !declarationHasModifier(method.DeclarationNode, "public") {
				return true
			}
		}
	}
	return false
}

func sourceArrayListSelectedAdd(scope *symbol.ClassScope, ctx Ctx) (*symbol.Definition, *symbol.ClassScope) {
	if sourceArrayListRoot(scope, ctx) == nil {
		return nil, nil
	}
	if scalar := sourceArrayListScalarOwner(scope, ctx); scalar != nil {
		return scalar.add, scope
	}
	for current := scope; current != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		declaring := classScopeCtx(current, ctx)
		for _, method := range current.Methods {
			if method == nil || !method.HasBody || method.OriginalName != "add" || method.IsStatic || method.IsPrivate || len(method.TypeParameters) != 0 || len(method.Parameters) != 1 || method.OriginalType != "boolean" {
				continue
			}
			if method.RuntimeDefault || sourceArrayListMethodOwned(method, current, declaring) && sourceArrayListSlotMatches(method, []string{"java.lang.Object"}, declaring) {
				return method, current
			}
		}
	}
	return nil, nil
}

func sourceArrayListDefaults(scope *symbol.ClassScope, ctx Ctx) []abstractCollectionDefault {
	if !sourceArrayListDirectRoot(scope, ctx) {
		return nil
	}
	element := "java.lang.Object"
	if scalar := sourceArrayListScalarOwner(scope, ctx); scalar != nil {
		element = scalar.element
	}
	return []abstractCollectionDefault{
		{"add", "boolean", "ArrayListBaseAddExecution", []string{element}},
		{"get", element, "ArrayListBaseGetExecution", []string{"int"}},
		{"size", "int", "ArrayListBaseSizeExecution", nil},
		{"iterator", "java.util.Iterator<" + element + ">", "IterableIteratorExecution", nil},
	}
}

func sourceArrayListIterationElementType(javaType, target string, ctx Ctx) (string, bool) {
	if target != "java.util.Collection" && target != "java.lang.Iterable" {
		return "", false
	}
	if sourceArrayListCanonicalEdge(javaType, ctx) {
		if scalar := sourceArrayListScalarOwner(ctx.currentClass, ctx); scalar != nil {
			return scalar.element, true
		}
		return "java.lang.Object", true
	}
	return "", false
}

// The refusal boundary follows the actual canonical parent declaration even
// when its element slot, constructors, or virtual overrides are outside P1.
// Source homonyms and binder shadows never acquire this platform ancestry.
func sourceArrayListCanonicalSourceAncestry(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		if scope.Class == nil || scope.IsInterface || scope.IsEnum || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "class_declaration" {
			return false
		}
		registered := false
		for _, actual := range allSourceClassScopes() {
			if actual == scope {
				registered = true
				break
			}
		}
		if !registered {
			return false
		}
		declaring, owned := supplierSourceDeclarationCtx(scope, ctx)
		if !owned {
			return false
		}
		edge := declaredJavaTypeOrigin(symbol.JavaType{Original: scope.Superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(scope.TypeParameters)}, declaring)
		if edge.parameter != nil || edge.arrayRank != 0 || edge.wildcard {
			return false
		}
		if edge.externalName == "java.util.ArrayList" && edge.nominalScope == nil {
			return true
		}
		parent := resolveSuperclassScopeInDeclaringContext(declaring, scope)
		if parent == nil || edge.nominalScope != parent {
			return false
		}
		scope = parent
	}
	return false
}

func sourceArrayListRefusesDeclaration(scope *symbol.ClassScope, ctx Ctx) bool {
	return sourceArrayListCanonicalSourceAncestry(scope, ctx) && sourceArrayListRoot(scope, ctx) == nil
}

// Source symbols retain the prefix TYPE spelling. After-name dimensions and
// spread syntax therefore need independent proof from this attached formal.
// Hidden rank is outside the supported platform-slot audit: refuse it rather
// than binding a business array overload as scalar add(Object), or missing an
// unsupported array platform override. Ordinary prefix-array slots retain the
// same origin comparison and refusal policy in sourceArrayListSlotMatches.
func sourceArrayListParameterRanksExact(method *symbol.Definition, ctx Ctx) bool {
	if method == nil || method.IsVariadic || method.DeclarationNode == nil || ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil {
		return false
	}
	node := method.DeclarationNode
	if (node.Type() != "method_declaration" && node.Type() != "abstract_method_declaration") || !threadSourceNodeWithin(node, ctx.currentFile.ParsedRoot()) || node.EndByte() > uint32(len(ctx.currentFile.Source)) {
		return false
	}
	parameters := node.ChildByFieldName("parameters")
	if parameters == nil {
		return false
	}
	written := nodeutil.SemanticNamedChildrenOf(parameters)
	if len(written) != len(method.Parameters) {
		return false
	}
	for i, parameter := range method.Parameters {
		formal := written[i]
		if formal == nil || formal.Type() != "formal_parameter" || !sourceArrayListNativeNoDimensions(formal) {
			return false
		}
		typ, name := nodeutil.JavaParameterNodes(formal)
		if parameter == nil || typ == nil || name == nil || typ.Content(ctx.currentFile.Source) != parameter.OriginalType || name.Content(ctx.currentFile.Source) != parameter.OriginalName {
			return false
		}
	}
	return true
}

// This is only a necessary spelling check for a direct native superclass.
// Comments, escape spellings and unknown syntax keep the complete source proof.
// No candidate is accepted here, and inherited source roots keep their walk.
func sourceArrayListSuperclassCandidate(javaType string) bool {
	return sourceNativeSuperclassCandidate(javaType, "ArrayList")
}

func sourceNativeSuperclassCandidate(javaType, simpleName string) bool {
	javaType = strings.TrimSpace(javaType)
	if javaType == "" {
		return false
	}
	var base strings.Builder
	depth := 0
	closed := false
	for _, ch := range javaType {
		if ch > 127 || ch == '/' || ch == '\\' || ch == '@' {
			return true
		}
		switch ch {
		case '<':
			if depth == 0 && closed {
				return true
			}
			depth++
		case '>':
			depth--
			if depth < 0 {
				return true
			}
			if depth == 0 {
				closed = true
			}
		default:
			if depth == 0 {
				if closed && ch != '.' {
					return true
				}
				closed = false
				base.WriteRune(ch)
			}
		}
	}
	if depth != 0 {
		return true
	}
	name := base.String()
	for _, part := range strings.Split(name, ".") {
		if !token.IsIdentifier(part) {
			return true
		}
	}
	return stripJavaQualifier(name) == simpleName
}
