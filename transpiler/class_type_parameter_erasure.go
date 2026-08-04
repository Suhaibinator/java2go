package transpiler

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// directOwnerTypeParameterUse describes a rank-zero, bare use of a type
// parameter declared by the owning class. Java erases these uses at the class
// ABI boundary (fields, constructor parameters, and instance method
// parameters/results), while the parameterized source view determines where a
// caller must insert a narrowing cast.
//
// Declaration identity is deliberately authoritative. A method may shadow a
// class parameter with the same source name, and treating that method-owned T
// as the class-owned T would apply erasure at the wrong ABI boundary.
type directOwnerTypeParameterUse struct {
	parameter symbol.TypeParam
	erasure   string
}

// directOwnerTypeParameterForDefinition plans erasure for a definition whose
// complete Java type is one bare class type parameter. Arrays and nested uses
// such as T[] and List<T> have different runtime representations and are kept
// out of this first, intentionally narrow lowering block.
func directOwnerTypeParameterForDefinition(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
) (directOwnerTypeParameterUse, bool) {
	if owner == nil || definition == nil || definition.DirectTypeParameter == nil {
		return directOwnerTypeParameterUse{}, false
	}

	component, rank := javaArrayTypeParts(definition.OriginalType)
	if rank != 0 {
		return directOwnerTypeParameterUse{}, false
	}
	base, arguments := parseJavaTypeString(component)
	base = strings.TrimSpace(base)
	if base == "" || len(arguments) != 0 || strings.Contains(base, ".") {
		return directOwnerTypeParameterUse{}, false
	}

	for _, parameter := range owner.TypeParameters {
		// Identity, rather than spelling, distinguishes an owner parameter from a
		// same-named method/local parameter. Legacy parameters without declaration
		// identity cannot provide that proof and are conservatively excluded.
		if parameter.Declaration == nil || parameter.Declaration != definition.DirectTypeParameter {
			continue
		}
		if base != parameter.Name && base != parameter.EmittedName() {
			return directOwnerTypeParameterUse{}, false
		}
		return directOwnerTypeParameterUse{
			parameter: parameter,
			erasure:   rawTypeParameterErasure(parameter, owner.TypeParameters),
		}, true
	}

	return directOwnerTypeParameterUse{}, false
}

// directOwnerTypeParameterErasure is the string-level convenience form used
// by ABI and storage planners that do not need the declaration itself.
func directOwnerTypeParameterErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
) (string, bool) {
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok {
		return "", false
	}
	return use.erasure, true
}

func directOwnerInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok || len(use.parameter.Bounds) != 1 {
		return "", false
	}
	erasure := qualifyJavaTypeInDeclaringContext(use.erasure, owner)
	base, _ := parseJavaTypeString(erasure)
	erasureScope := resolveClassScopeByQualifiedName(ctx, base)
	if erasureScope == nil || !erasureScope.IsInterface {
		return "", false
	}
	return erasure, true
}

// directOwnerStandaloneInterfaceErasure limits the first physical member ABI
// slice to standalone concrete leaf classes. Generic interfaces, abstract
// classes, implemented interfaces, superclass overrides, and known subclasses
// need Java-style bridge methods before erased field/result storage and source
// views can safely coexist in Go.
func directOwnerStandaloneInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || definition == nil || definition.IsStatic || owner.IsInterface ||
		owner.IsAbstract || owner.IsEnum || owner.IsInner || len(owner.ImplementedInterfaces) != 0 ||
		strings.TrimSpace(owner.Superclass) != "" || classHasKnownSubclass(owner, ctx) {
		return "", false
	}
	for _, nested := range owner.Subclasses {
		if nested != nil && nested.IsInner {
			return "", false
		}
	}
	return directOwnerInterfaceErasure(owner, definition, ctx)
}

func directOwnerOrdinaryFieldInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || owner.Class == nil || definition == nil || definition.DeclarationNode == nil {
		return "", false
	}
	classNode := owner.Class.DeclarationNode
	if classNode == nil {
		return "", false
	}
	parent := classNode.Parent()
	if classNode.Type() != "class_declaration" || parent == nil ||
		(parent.Type() != "program" && parent.Type() != "class_body") ||
		definition.DeclarationNode.Type() != "field_declaration" {
		return "", false
	}
	if erasure, ok := directOwnerStandaloneInterfaceErasure(owner, definition, ctx); ok {
		return erasure, true
	}
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok || !ownerTypeParameterHasErasedCallableUse(owner, use.parameter.Declaration, ctx) {
		return "", false
	}
	return directOwnerInterfaceErasure(owner, definition, ctx)
}

func directOwnerOrdinaryMethodInterfaceErasure(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool) {
	if owner == nil || owner.Class == nil || definition == nil || definition.DeclarationNode == nil {
		return "", false
	}
	classNode := owner.Class.DeclarationNode
	if classNode == nil {
		return "", false
	}
	parent := classNode.Parent()
	if classNode.Type() != "class_declaration" || parent == nil ||
		(parent.Type() != "program" && parent.Type() != "class_body") ||
		definition.DeclarationNode.Type() != "method_declaration" {
		return "", false
	}
	if erasure, ok := directOwnerStandaloneInterfaceErasure(owner, definition, ctx); ok {
		return erasure, true
	}
	if !directOwnerCallableMethodEligible(owner, definition, ctx) {
		return "", false
	}
	return directOwnerInterfaceErasure(owner, definition, ctx)
}

// directOwnerTypeParameterMethodParameterType applies the same erased
// descriptor selected for a source method to one bare class-owned parameter.
// Java varargs erase as arrays and remain outside this rank-zero slice.
func directOwnerTypeParameterMethodParameterType(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	index int,
	declared ast.Expr,
	ctx Ctx,
) ast.Expr {
	erasure, ok := directOwnerMethodParameterInterfaceErasure(owner, method, index, ctx)
	if !ok {
		return declared
	}
	physical := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(physical, erasure, ctx)
}

func directOwnerMethodParameterInterfaceErasure(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	index int,
	ctx Ctx,
) (string, bool) {
	if method == nil || index < 0 || index >= len(method.Parameters) ||
		executionParameterIsVariadic(method, index) || !directOwnerCallableMethodEligible(owner, method, ctx) {
		return "", false
	}
	return directOwnerInterfaceErasure(owner, method.Parameters[index], ctx)
}

func directOwnerMethodHasErasedParameterABI(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if !directOwnerCallableMethodEligible(owner, method, ctx) {
		return false
	}
	for index := range method.Parameters {
		if executionParameterIsVariadic(method, index) {
			continue
		}
		if _, ok := directOwnerInterfaceErasure(owner, method.Parameters[index], ctx); ok {
			return true
		}
	}
	return false
}

func directOwnerMethodHasErasedCallableABI(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if !directOwnerCallableMethodEligible(owner, method, ctx) {
		return false
	}
	if _, ok := directOwnerInterfaceErasure(owner, method, ctx); ok {
		return true
	}
	return directOwnerMethodHasErasedParameterABI(owner, method, ctx)
}

// directOwnerCallableMethodEligible admits only concrete override families
// whose generated callable descriptor is uniform at every declaration. A
// concrete specialization such as Derived extends Base<First> needs a Java
// bridge (Numbered accept(Numbered) -> First accept(First)) and is deliberately
// excluded until that bridge is modeled. Generic interfaces and abstract or
// interface-implementing classes are likewise kept on their established Go ABI.
func directOwnerCallableMethodEligible(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if !directOwnerCallableMethodFamilyEligible(owner, method, ctx) {
		return false
	}
	for _, declaration := range methodDirectOwnerTypeParameterDeclarations(owner, method) {
		if !ownerTypeParameterCallableShapeSupported(declaration, ctx) {
			return false
		}
	}
	return true
}

func directOwnerCallableMethodFamilyEligible(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	if !ordinaryConcreteCallableOwner(owner) || !ordinarySourceMethod(owner, method) {
		return false
	}
	want, changed, ok := directOwnerCallablePhysicalSignature(owner, method, ctx)
	if !ok || !changed {
		return false
	}

	eligible := true
	visitAllClassScopes(func(candidate *symbol.ClassScope) bool {
		if candidate == nil || candidate == owner || !classScopesInheritanceRelated(candidate, owner, ctx) {
			return false
		}
		// Even an inheriting class that does not override this method can expose it
		// through an implemented interface or an abstract contract. Those Go method
		// sets need explicit Java bridge planning before their inherited ABI moves.
		if !ordinaryConcreteCallableOwner(candidate) {
			eligible = false
			return true
		}
		if classScopeDescendsFrom(candidate, owner, ctx) &&
			!descendantPreservesCallableOwnerParameters(candidate, owner, method, ctx) {
			eligible = false
			return true
		}
		for _, candidateMethod := range candidate.Methods {
			if candidateMethod == nil || candidateMethod.OriginalName != method.OriginalName ||
				len(candidateMethod.Parameters) != len(method.Parameters) || candidateMethod.Constructor ||
				candidateMethod.IsStatic || candidateMethod.IsPrivate || candidateMethod.RequiresHelper {
				continue
			}
			got, _, candidateOK := directOwnerCallablePhysicalSignature(candidate, candidateMethod, ctx)
			if !candidateOK || !sameStringSlice(got, want) {
				eligible = false
				return true
			}
		}
		return false
	})
	if classHasUnmodeledCallableSubclass(owner, ctx) {
		return false
	}
	return eligible
}

func classHasUnmodeledCallableSubclass(target *symbol.ClassScope, ctx Ctx) bool {
	if target == nil {
		return false
	}
	for _, owner := range allSourceClassScopes() {
		if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner)
		if file == nil {
			continue
		}
		ownerCtx := classScopeCtx(owner, ctx)
		var walk func(node *sitter.Node) bool
		walk = func(node *sitter.Node) bool {
			if node == nil {
				return false
			}
			var supertype *sitter.Node
			switch node.Type() {
			case "object_creation_expression":
				for _, child := range nodeutil.NamedChildrenOf(node) {
					if child.Type() == "class_body" {
						supertype = node.ChildByFieldName("type")
						break
					}
				}
			case "class_declaration":
				parent := node.Parent()
				if parent != nil && parent.Type() != "program" && parent.Type() != "class_body" {
					if superclass := node.ChildByFieldName("superclass"); superclass != nil {
						types := collectTypeNodes(superclass)
						if len(types) > 0 {
							supertype = types[0]
						}
					}
				}
			}
			if supertype != nil {
				base, _ := parseJavaTypeString(supertype.Content(file.Source))
				resolved := resolveClassScopeByQualifiedName(ownerCtx, base)
				if resolved == target || classScopeDescendsFrom(resolved, target, ownerCtx) {
					return true
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if walk(child) {
					return true
				}
			}
			return false
		}
		if walk(owner.Class.DeclarationNode) {
			return true
		}
	}
	return false
}

func methodDirectOwnerTypeParameterDeclarations(
	owner *symbol.ClassScope,
	method *symbol.Definition,
) []*symbol.TypeParamDeclaration {
	if owner == nil || method == nil {
		return nil
	}
	seen := map[*symbol.TypeParamDeclaration]struct{}{}
	var result []*symbol.TypeParamDeclaration
	definitions := append([]*symbol.Definition{method}, method.Parameters...)
	for _, definition := range definitions {
		use, ok := directOwnerTypeParameterForDefinition(owner, definition)
		if !ok || use.parameter.Declaration == nil {
			continue
		}
		if _, duplicate := seen[use.parameter.Declaration]; duplicate {
			continue
		}
		seen[use.parameter.Declaration] = struct{}{}
		result = append(result, use.parameter.Declaration)
	}
	return result
}

func descendantPreservesCallableOwnerParameters(
	descendant *symbol.ClassScope,
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) bool {
	if descendant == nil || owner == nil || method == nil {
		return false
	}
	mapped := mapClassTypeArgumentStringsToAncestor(
		descendant,
		descendant.GoTypeParameterNames(),
		owner,
		ctx,
	)
	if len(mapped) != len(owner.TypeParameters) {
		return false
	}
	descendantCtx := classScopeCtx(descendant, ctx)
	for _, declaration := range methodDirectOwnerTypeParameterDeclarations(owner, method) {
		ownerIndex := -1
		for index, parameter := range owner.TypeParameters {
			if parameter.Declaration == declaration {
				ownerIndex = index
				break
			}
		}
		if ownerIndex < 0 || ownerIndex >= len(mapped) {
			return false
		}
		mappedName := strings.TrimSpace(mapped[ownerIndex])
		var descendantParameter *symbol.TypeParam
		for index := range descendant.TypeParameters {
			parameter := &descendant.TypeParameters[index]
			if mappedName == parameter.Name || mappedName == parameter.EmittedName() {
				descendantParameter = parameter
				break
			}
		}
		if descendantParameter == nil {
			// A concrete specialization needs a synthetic erased bridge even when it
			// does not override the affected method.
			return false
		}
		ownerParameter := owner.TypeParameters[ownerIndex]
		ownerErasure := rawTypeParameterErasure(ownerParameter, owner.TypeParameters)
		descendantErasure := rawTypeParameterErasure(*descendantParameter, descendant.TypeParameters)
		if callablePhysicalTypeKey(qualifyJavaTypeInDeclaringContext(ownerErasure, owner), classScopeCtx(owner, ctx)) !=
			callablePhysicalTypeKey(qualifyJavaTypeInDeclaringContext(descendantErasure, descendant), descendantCtx) {
			return false
		}
	}
	return true
}

func ownerTypeParameterCallableShapeSupported(
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if declaration == nil {
		return false
	}
	declarations := callableTypeParameterDeclarationClosure(declaration, ctx)
	found := false
	supported := true
	visitAllClassScopes(func(scope *symbol.ClassScope) bool {
		for related := range declarations {
			if !classScopeCarriesTypeParameterDeclaration(scope, related) {
				continue
			}
			if !classInitializerTypeSyntaxSupportsCallableErasure(scope, related, ctx) {
				supported = false
				return true
			}
		}
		for _, method := range scope.Methods {
			if method == nil || method.IsStatic {
				continue
			}
			for related := range declarations {
				if !classScopeCarriesTypeParameterDeclaration(scope, related) {
					continue
				}
				if method.Constructor {
					if !constructorBodySupportsCallableErasure(scope, method, related, ctx) {
						supported = false
						return true
					}
					continue
				}
				// Body type syntax is not represented completely by Definition.Children.
				// Check every instance method in the connected representation family,
				// including siblings whose signature and locals never mention this slot.
				if !methodBodyTypeSyntaxSupportsCallableErasure(scope, method, related, true, ctx) {
					supported = false
					return true
				}
				if !definitionTreeReferencesTypeParameter(method, related) {
					continue
				}
				found = true
				if !definitionTreeUsesOnlyBareTypeParameter(method, related, scope, false, ctx) ||
					!directOwnerCallableMethodFamilyEligible(scope, method, ctx) {
					supported = false
					return true
				}
			}
		}
		for _, field := range scope.Fields {
			if field == nil {
				continue
			}
			for related := range declarations {
				if !definitionTreeReferencesTypeParameter(field, related) {
					continue
				}
				if !definitionTreeUsesOnlyBareTypeParameter(field, related, scope, false, ctx) {
					supported = false
					return true
				}
			}
		}
		return false
	})
	return found && supported
}

func classScopeCarriesTypeParameterDeclaration(scope *symbol.ClassScope, declaration *symbol.TypeParamDeclaration) bool {
	if scope == nil || declaration == nil {
		return false
	}
	for _, parameter := range scope.TypeParameters {
		if parameter.Declaration == declaration {
			return true
		}
	}
	return false
}

// Symbol definitions cover declared locals but do not record every type syntax
// in a body. In particular, new Holder<T>(...) can retain an instantiated Go T
// after the surrounding callable has moved to its erased interface ABI. Admit
// only the two body-local bare uses whose lowering is explicitly physicalized:
// local declarations and casts. All other exact owner-parameter type syntax is
// a conservative whole-plan gate.
func methodBodyTypeSyntaxSupportsCallableErasure(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
	allowPhysicalBareUses bool,
	ctx Ctx,
) bool {
	if owner == nil || method == nil || method.DeclarationNode == nil || declaration == nil {
		return false
	}
	body := method.DeclarationNode.ChildByFieldName("body")
	if body == nil {
		return false
	}
	file := findFileScopeForClassScope(owner)
	if file == nil {
		return false
	}
	source := file.Source
	methodCtx := classScopeCtx(owner, ctx)
	methodCtx.localScope = method

	var walk func(*sitter.Node) bool
	walk = func(node *sitter.Node) bool {
		if node == nil {
			return true
		}
		skip := (*sitter.Node)(nil)
		switch node.Type() {
		case "local_variable_declaration":
			typeNode := node.ChildByFieldName("type")
			if allowPhysicalBareUses && typeNode != nil {
				typeText := typeNode.Content(source)
				switch {
				case visibleTypeParameterDeclarationForJavaType(typeText, methodCtx) == declaration:
					// explicitLocalVariableType lowers an uninitialized/null local to
					// the erased physical interface; initialized non-null locals use :=.
					skip = typeNode
				case javaTypeUsesVisibleDeclarationOnlyAsCarriedInnerArgument(typeText, declaration, owner, methodCtx):
					// Outer<X>.Inner<Item> retains X only in the hidden enclosing
					// pointer slot. Callable erasure changes member values/method
					// descriptors, not that pointer's invariant Go instantiation.
					skip = typeNode
				}
			}
		case "cast_expression":
			if allowPhysicalBareUses && node.NamedChildCount() > 0 {
				typeNode := node.NamedChild(0)
				if visibleTypeParameterDeclarationForJavaType(typeNode.Content(source), methodCtx) == declaration {
					// ParseExpr lowers this exact cast target to the JVM erasure.
					skip = typeNode
				}
			}
		case "type_identifier":
			if visibleTypeParameterDeclarationForJavaType(node.Content(source), methodCtx) == declaration {
				return false
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if skip != nil && sameSourceNode(child, skip) {
				continue
			}
			if !walk(child) {
				return false
			}
		}
		return true
	}
	return walk(body)
}

// Field initializer expressions are lowered into a synthetic init method and
// therefore sit outside every source method body audited above. An apparently
// unrelated field type can still hide an instantiated use such as
//
//	Object hidden = new Holder<T>(outer);
//
// after outer's physical storage has moved from T to its erased interface.
// Conservatively retain the established representation whenever initializer
// type syntax resolves to the affected declaration. Direct initializer blocks
// receive the same treatment even though their lowering support is currently
// narrower than ordinary field initializers.
func classInitializerTypeSyntaxSupportsCallableErasure(
	owner *symbol.ClassScope,
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || declaration == nil {
		return false
	}
	body := owner.Class.DeclarationNode.ChildByFieldName("body")
	file := findFileScopeForClassScope(owner)
	if body == nil || file == nil {
		return false
	}
	source := file.Source
	initializerCtx := classScopeCtx(owner, ctx)

	initializerSupported := func(root *sitter.Node) bool {
		var walk func(*sitter.Node) bool
		walk = func(node *sitter.Node) bool {
			if node == nil {
				return true
			}
			// A nested/local/anonymous type owns a fresh lexical type-parameter
			// environment and may also capture this initializer's carried T. Its
			// synthesized ClassScope is not guaranteed to exist during this global
			// audit, so conservatively retain the established representation.
			switch node.Type() {
			case "class_declaration", "interface_declaration", "enum_declaration",
				"record_declaration", "annotation_type_declaration":
				return false
			case "class_body":
				parent := node.Parent()
				if parent != nil && (parent.Type() == "object_creation_expression" || parent.Type() == "enum_constant") {
					return false
				}
			}
			if node.Type() == "type_identifier" &&
				visibleTypeParameterDeclarationForJavaType(node.Content(source), initializerCtx) == declaration {
				return false
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		return walk(root)
	}

	var fieldInitializersSupported func(*sitter.Node) bool
	fieldInitializersSupported = func(node *sitter.Node) bool {
		if node == nil {
			return true
		}
		if node.Type() == "variable_declarator" {
			return initializerSupported(node.ChildByFieldName("value"))
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if !fieldInitializersSupported(child) {
				return false
			}
		}
		return true
	}

	for _, declarationNode := range nodeutil.NamedChildrenOf(body) {
		switch declarationNode.Type() {
		case "field_declaration":
			if !fieldInitializersSupported(declarationNode) {
				return false
			}
		case "block":
			if !initializerSupported(declarationNode) {
				return false
			}
		case "static_initializer":
			// Java forbids a static context from referring to a class-owned type
			// parameter. Any same-spelled T here belongs to a nested/local declaration
			// and must not gate the outer representation plan.
			continue
		}
	}
	return true
}

func constructorBodySupportsCallableErasure(
	owner *symbol.ClassScope,
	constructor *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if owner == nil || constructor == nil || !constructor.Constructor || declaration == nil {
		return false
	}
	// Constructor parameters keep their established T ABI in this patch. Local
	// storage and explicit body types cannot safely consume a newly-erased field.
	for _, child := range constructor.Children {
		if definitionTreeReferencesTypeParameter(child, declaration) {
			return false
		}
	}
	if !methodBodyTypeSyntaxSupportsCallableErasure(owner, constructor, declaration, false, ctx) {
		return false
	}
	body := constructor.DeclarationNode.ChildByFieldName("body")
	var containsTernary func(*sitter.Node) bool
	containsTernary = func(node *sitter.Node) bool {
		if node == nil {
			return false
		}
		if node.Type() == "ternary_expression" {
			return true
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if containsTernary(child) {
				return true
			}
		}
		return false
	}
	return !containsTernary(body)
}

// callableTypeParameterDeclarationClosure connects class parameters that are
// the same source slot across generic inheritance. Base<T> and Derived<X>
// extends Base<X> must migrate as one representation: allowing Base's field
// and methods to erase while Derived still contains an unsupported Box<X> use
// would leave the embedded subobject and override method sets on incompatible
// ABIs.
func callableTypeParameterDeclarationClosure(
	seed *symbol.TypeParamDeclaration,
	ctx Ctx,
) map[*symbol.TypeParamDeclaration]struct{} {
	result := map[*symbol.TypeParamDeclaration]struct{}{seed: {}}
	scopes := allSourceClassScopes()
	changed := true
	for changed {
		changed = false
		for _, descendant := range scopes {
			if descendant == nil {
				continue
			}
			for _, ancestor := range scopes {
				if ancestor == nil || descendant == ancestor ||
					!classScopeDescendsFrom(descendant, ancestor, ctx) {
					continue
				}
				mapped := mapClassTypeArgumentStringsToAncestor(
					descendant,
					descendant.GoTypeParameterNames(),
					ancestor,
					ctx,
				)
				for index, argument := range mapped {
					if index >= len(ancestor.TypeParameters) || ancestor.TypeParameters[index].Declaration == nil {
						continue
					}
					descendantParameter := bareClassTypeParameterForJavaType(descendant, argument)
					if descendantParameter == nil || descendantParameter.Declaration == nil {
						continue
					}
					ancestorDeclaration := ancestor.TypeParameters[index].Declaration
					_, ancestorPresent := result[ancestorDeclaration]
					_, descendantPresent := result[descendantParameter.Declaration]
					if !ancestorPresent && !descendantPresent {
						continue
					}
					if !ancestorPresent {
						result[ancestorDeclaration] = struct{}{}
						changed = true
					}
					if !descendantPresent {
						result[descendantParameter.Declaration] = struct{}{}
						changed = true
					}
				}
			}
		}
	}
	return result
}

func bareClassTypeParameterForJavaType(scope *symbol.ClassScope, javaType string) *symbol.TypeParam {
	if scope == nil {
		return nil
	}
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(component)
	base = strings.TrimSpace(base)
	if rank != 0 || len(arguments) != 0 || base == "" || strings.Contains(base, ".") {
		return nil
	}
	for index := range scope.TypeParameters {
		parameter := &scope.TypeParameters[index]
		if base == parameter.Name || base == parameter.EmittedName() {
			return parameter
		}
	}
	return nil
}

func definitionTreeReferencesTypeParameter(
	definition *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
) bool {
	if definition == nil || declaration == nil {
		return false
	}
	if definitionReferencesTypeParameterDeclaration(definition, declaration) {
		return true
	}
	for _, parameter := range definition.Parameters {
		if definitionTreeReferencesTypeParameter(parameter, declaration) {
			return true
		}
	}
	for _, child := range definition.Children {
		if definitionTreeReferencesTypeParameter(child, declaration) {
			return true
		}
	}
	return false
}

func definitionTreeUsesOnlyBareTypeParameter(
	definition *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
	owner *symbol.ClassScope,
	localDefinition bool,
	ctx Ctx,
) bool {
	if definition == nil || declaration == nil {
		return true
	}
	usesDeclaration := definitionReferencesTypeParameterDeclaration(definition, declaration)
	if usesDeclaration && definition.DirectTypeParameter != declaration &&
		(!localDefinition || !definitionUsesDeclarationOnlyAsCarriedInnerArgument(definition, declaration, owner, ctx)) {
		return false
	}
	if definition.DirectTypeParameter == declaration {
		component, rank := javaArrayTypeParts(definition.OriginalType)
		base, arguments := parseJavaTypeString(component)
		if rank != 0 || len(arguments) != 0 || strings.TrimSpace(base) == "" {
			return false
		}
	}
	for _, parameter := range definition.Parameters {
		if !definitionTreeUsesOnlyBareTypeParameter(parameter, declaration, owner, false, ctx) {
			return false
		}
	}
	for _, child := range definition.Children {
		if !definitionTreeUsesOnlyBareTypeParameter(child, declaration, owner, true, ctx) {
			return false
		}
	}
	return true
}

func definitionUsesDeclarationOnlyAsCarriedInnerArgument(
	definition *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
	owner *symbol.ClassScope,
	ctx Ctx,
) bool {
	if definition == nil || declaration == nil {
		return false
	}
	classify := func(javaType string) (bool, bool) {
		references := false
		for spelling, binding := range definition.TypeParameterBindings {
			if binding == declaration && javaTypeReferencesTypeParameter(javaType, spelling) {
				references = true
				break
			}
		}
		if !references {
			return false, false
		}
		component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
		base, arguments := parseJavaTypeString(component)
		base = strings.TrimSpace(base)
		return true, rank == 0 && len(arguments) == 0 && !strings.Contains(base, ".") &&
			definition.TypeParameterBindings[base] == declaration
	}
	return javaTypeUsesOnlyCarriedInnerArguments(definition.OriginalType, declaration, owner, ctx, classify)
}

func javaTypeUsesVisibleDeclarationOnlyAsCarriedInnerArgument(
	javaType string,
	declaration *symbol.TypeParamDeclaration,
	owner *symbol.ClassScope,
	ctx Ctx,
) bool {
	if declaration == nil {
		return false
	}
	classify := func(candidate string) (bool, bool) {
		references := false
		for _, spelling := range []string{declaration.SourceName, declaration.GoName} {
			if !javaTypeReferencesTypeParameter(candidate, spelling) {
				continue
			}
			if visibleTypeParameterDeclarationForJavaType(spelling, ctx) == declaration {
				references = true
				break
			}
		}
		if !references {
			return false, false
		}
		return true, visibleTypeParameterDeclarationForJavaType(candidate, ctx) == declaration
	}
	return javaTypeUsesOnlyCarriedInnerArguments(javaType, declaration, owner, ctx, classify)
}

func javaTypeUsesOnlyCarriedInnerArguments(
	javaType string,
	declaration *symbol.TypeParamDeclaration,
	owner *symbol.ClassScope,
	ctx Ctx,
	classify func(string) (references bool, bare bool),
) bool {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 || declaration == nil || classify == nil {
		return false
	}
	base, arguments := parseJavaTypeString(component)
	target := resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), base)
	if target == nil || !target.IsInner || target.Class == nil || target.Enclosing == nil ||
		target.Enclosing.Class == nil || len(arguments) == 0 ||
		!strings.HasSuffix(strings.TrimSpace(base), target.Enclosing.Class.OriginalName+"."+target.Class.OriginalName) {
		return false
	}
	hiddenCount := len(target.TypeParameters) - len(target.OwnTypeParameters())
	if hiddenCount <= 0 {
		return false
	}
	related := callableTypeParameterDeclarationClosure(declaration, ctx)
	found := false
	for index, argument := range arguments {
		references, bare := classify(argument)
		if !references {
			continue
		}
		found = true
		if !bare {
			return false
		}
		targetParameter := genericTargetParameterForArgument(target, len(arguments), index)
		if targetParameter == nil || targetParameter.Declaration == nil {
			return false
		}
		targetIndex := -1
		for parameterIndex := range target.TypeParameters {
			if target.TypeParameters[parameterIndex].Declaration == targetParameter.Declaration {
				targetIndex = parameterIndex
				break
			}
		}
		if targetIndex < 0 || targetIndex >= hiddenCount {
			return false
		}
		if _, connected := related[targetParameter.Declaration]; !connected {
			return false
		}
		if innerConstructorFormalReferencesDeclaration(target, targetParameter.Declaration) {
			return false
		}
		if innerSuperclassConstructorFormalConsumesCarriedDeclaration(target, targetParameter.Declaration, ctx) {
			return false
		}
	}
	return found
}

func innerConstructorFormalReferencesDeclaration(
	target *symbol.ClassScope,
	declaration *symbol.TypeParamDeclaration,
) bool {
	if target == nil || declaration == nil {
		return true
	}
	for _, constructor := range target.Methods {
		if constructor == nil || !constructor.Constructor {
			continue
		}
		for _, parameter := range constructor.Parameters {
			if definitionTreeReferencesTypeParameter(parameter, declaration) {
				return true
			}
		}
	}
	return false
}

func innerSuperclassConstructorFormalConsumesCarriedDeclaration(
	target *symbol.ClassScope,
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if target == nil || declaration == nil {
		return true
	}
	var carried *symbol.TypeParam
	for index := range target.TypeParameters {
		if target.TypeParameters[index].Declaration == declaration {
			carried = &target.TypeParameters[index]
			break
		}
	}
	if carried == nil {
		return true
	}

	seen := map[*symbol.ClassScope]struct{}{}
	for ancestor := resolveSuperclassScopeInDeclaringContext(ctx, target); ancestor != nil; ancestor = resolveSuperclassScopeInDeclaringContext(ctx, ancestor) {
		if _, duplicate := seen[ancestor]; duplicate {
			return true
		}
		seen[ancestor] = struct{}{}
		mapped := mapClassTypeArgumentStringsToAncestor(
			target,
			target.GoTypeParameterNames(),
			ancestor,
			ctx,
		)
		if len(ancestor.TypeParameters) > 0 && len(mapped) != len(ancestor.TypeParameters) {
			return true
		}
		affected := map[*symbol.TypeParamDeclaration]struct{}{}
		for index, parameter := range ancestor.TypeParameters {
			if parameter.Declaration == nil || index >= len(mapped) {
				continue
			}
			argument := mapped[index]
			if javaTypeReferencesTypeParameter(argument, carried.EmittedName()) ||
				(carried.EmittedName() != carried.Name && javaTypeReferencesTypeParameter(argument, carried.Name)) {
				affected[parameter.Declaration] = struct{}{}
			}
		}
		if len(affected) == 0 {
			continue
		}
		for _, constructor := range ancestor.Methods {
			if constructor == nil || !constructor.Constructor {
				continue
			}
			for _, formal := range constructor.Parameters {
				for ancestorDeclaration := range affected {
					if definitionTreeReferencesTypeParameter(formal, ancestorDeclaration) {
						return true
					}
				}
			}
		}
	}
	return false
}

// TypeParameterBindings records the complete lexical environment at a
// definition, not just the parameters which occur in its type. Always combine
// declaration identity with an occurrence check before treating a binding as
// a use; otherwise an unrelated int/String local can veto an entire callable
// family merely because it was declared inside a generic class.
func definitionReferencesTypeParameterDeclaration(
	definition *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
) bool {
	if definition == nil || declaration == nil {
		return false
	}
	if definition.DirectTypeParameter == declaration {
		return true
	}
	for spelling, binding := range definition.TypeParameterBindings {
		if binding != declaration {
			continue
		}
		if javaTypeReferencesTypeParameter(definition.OriginalType, spelling) ||
			javaTypeReferencesTypeParameter(definition.OriginalType, declaration.SourceName) ||
			javaTypeReferencesTypeParameter(definition.OriginalType, declaration.GoName) {
			return true
		}
	}
	return false
}

func ordinaryConcreteCallableOwner(owner *symbol.ClassScope) bool {
	return owner != nil && owner.Class != nil && !owner.IsInterface && !owner.IsAbstract &&
		!owner.IsEnum && len(owner.ImplementedInterfaces) == 0
}

func ordinarySourceMethod(owner *symbol.ClassScope, method *symbol.Definition) bool {
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || method == nil ||
		method.DeclarationNode == nil || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper {
		return false
	}
	classNode := owner.Class.DeclarationNode
	parent := classNode.Parent()
	return classNode.Type() == "class_declaration" && parent != nil &&
		(parent.Type() == "program" || parent.Type() == "class_body") &&
		method.DeclarationNode.Type() == "method_declaration"
}

func directOwnerCallablePhysicalSignature(
	owner *symbol.ClassScope,
	method *symbol.Definition,
	ctx Ctx,
) ([]string, bool, bool) {
	if owner == nil || method == nil {
		return nil, false, false
	}
	declarationCtx := classScopeCtx(owner, ctx)
	signature := make([]string, 0, len(method.Parameters)+1)
	changed := false
	for index, parameter := range method.Parameters {
		if parameter == nil || executionParameterIsVariadic(method, index) {
			return nil, false, false
		}
		physical, erased, ok := directOwnerCallablePhysicalType(owner, parameter, declarationCtx)
		if !ok {
			return nil, false, false
		}
		signature = append(signature, physical)
		changed = changed || erased
	}
	if strings.TrimSpace(method.OriginalType) == "" || strings.TrimSpace(method.OriginalType) == "void" {
		signature = append(signature, "void")
	} else {
		physical, erased, ok := directOwnerCallablePhysicalType(owner, method, declarationCtx)
		if !ok {
			return nil, false, false
		}
		signature = append(signature, physical)
		changed = changed || erased
	}
	return signature, changed, true
}

func directOwnerCallablePhysicalType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	ctx Ctx,
) (string, bool, bool) {
	if use, direct := directOwnerTypeParameterForDefinition(owner, definition); direct {
		erasure, ok := directOwnerInterfaceErasure(owner, definition, ctx)
		if !ok || use.parameter.Declaration == nil {
			return "", false, false
		}
		return callablePhysicalTypeKey(erasure, ctx), true, true
	}
	for _, parameter := range owner.TypeParameters {
		if parameter.Declaration == nil {
			continue
		}
		if definitionReferencesTypeParameterDeclaration(definition, parameter.Declaration) {
			// Nested/array owner-parameter uses need a wider ABI migration.
			return "", false, false
		}
	}
	return callablePhysicalTypeKey(qualifyJavaTypeInDeclaringContext(definitionJavaType(definition), owner), ctx), false, true
}

func callablePhysicalTypeKey(javaType string, ctx Ctx) string {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(component)
	if len(arguments) == 0 {
		if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil {
			return "scope:" + findJavaPackageForClassScope(scope) + ":" + scope.Class.Name + ":" + strconv.Itoa(rank)
		}
	}
	return strings.Join(strings.Fields(javaType), "")
}

func classScopeCtx(scope *symbol.ClassScope, ctx Ctx) Ctx {
	result := ctx.Clone()
	result.currentClass = scope
	result.localScope = nil
	if file := findFileScopeForClassScope(scope); file != nil {
		result.currentFile = file
	}
	return result
}

func classScopesInheritanceRelated(a, b *symbol.ClassScope, ctx Ctx) bool {
	return classScopeDescendsFrom(a, b, ctx) || classScopeDescendsFrom(b, a, ctx)
}

func classScopeDescendsFrom(candidate, target *symbol.ClassScope, ctx Ctx) bool {
	if candidate == nil || target == nil || candidate == target {
		return false
	}
	seen := map[*symbol.ClassScope]struct{}{}
	for current := candidate; current != nil; {
		currentCtx := classScopeCtx(current, ctx)
		parent := resolveSuperclassScope(currentCtx, current)
		if parent == target {
			return true
		}
		if parent == nil {
			return false
		}
		if _, duplicate := seen[parent]; duplicate {
			return false
		}
		seen[parent] = struct{}{}
		current = parent
	}
	return false
}

func visitAllClassScopes(visitor func(*symbol.ClassScope) bool) bool {
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				if visitClassScopes(top, visitor) {
					return true
				}
			}
		}
	}
	return false
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func ownerTypeParameterHasErasedCallableUse(
	owner *symbol.ClassScope,
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if owner == nil || declaration == nil {
		return false
	}
	relatedDeclarations := callableTypeParameterDeclarationClosure(declaration, ctx)
	foundEligible := false
	allEligible := true
	visitAllClassScopes(func(scope *symbol.ClassScope) bool {
		for _, method := range scope.Methods {
			referencesRelatedDeclaration := false
			for related := range relatedDeclarations {
				if methodDirectlyUsesTypeParameterDeclaration(method, related) {
					referencesRelatedDeclaration = true
					break
				}
			}
			if !referencesRelatedDeclaration {
				continue
			}
			if !directOwnerCallableMethodEligible(scope, method, ctx) {
				allEligible = false
				return true
			}
			foundEligible = true
		}
		return false
	})
	return foundEligible && allEligible
}

func methodDirectlyUsesTypeParameterDeclaration(
	method *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
) bool {
	if method == nil || declaration == nil || method.Constructor || method.IsStatic {
		return false
	}
	if method.DirectTypeParameter == declaration {
		return true
	}
	for _, parameter := range method.Parameters {
		if parameter != nil && parameter.DirectTypeParameter == declaration {
			return true
		}
	}
	return false
}

// directOwnerTypeParameterFieldStorageType gives a bare class-owned type
// parameter field the Java declaration's erased representation. Every raw and
// parameterized alias consequently shares one physical slot type; use-site
// lowering is responsible for restoring the narrower source view where javac
// would emit a checkcast.
func directOwnerTypeParameterFieldStorageType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	declared ast.Expr,
	ctx Ctx,
) ast.Expr {
	if definition == nil || definition.IsStatic {
		return declared
	}
	erasure, ok := directOwnerOrdinaryFieldInterfaceErasure(owner, definition, ctx)
	if !ok {
		return declared
	}
	// This first physical-storage slice is limited to generated interface
	// erasures. A Go interface can retain every implementing object's identity,
	// including heap pollution through a raw alias. Concrete-class erasures need
	// the broader canonical reference/subobject migration, while unbounded Object
	// fields retain the established generic Go API until that migration lands.
	storage := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(storage, erasure, ctx)
}

// directOwnerTypeParameterMethodResultType applies Java's erased callable ABI
// to a method whose complete result is one bare parameter declared by its
// owning class. The source-level parameterized view remains in symbol metadata;
// callers insert a concrete ObjectView only where javac would emit checkcast.
func directOwnerTypeParameterMethodResultType(
	owner *symbol.ClassScope,
	definition *symbol.Definition,
	declared ast.Expr,
	ctx Ctx,
) ast.Expr {
	if definition == nil || definition.IsStatic {
		return declared
	}
	erasure, ok := directOwnerOrdinaryMethodInterfaceErasure(owner, definition, ctx)
	if !ok {
		return declared
	}
	result := javaTypeStringToGoTypeExpr(erasure, inScopeTypeParameters(ctx), ctx)
	return abstractClassToInterface(result, erasure, ctx)
}
