package transpiler

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Candidate source bounds are not permission to erase storage. Publication of
// one successful family attaches this closed declaration-owned slot inventory.
type genericFamilySourceBoundCertificate struct {
	definitions  map[*symbol.Definition]*symbol.TypeParamDeclaration
	constructors map[*symbol.Definition]bool
	slots        map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation
}

func genericFamilyConcreteSourceBound(parameter symbol.TypeParam, ctx Ctx, visiting map[*symbol.TypeParamDeclaration]bool) (*symbol.ClassScope, *symbol.FileScope, bool) {
	if parameter.Declaration == nil || visiting[parameter.Declaration] || len(parameter.Bounds) != 1 {
		return nil, nil, false
	}
	visiting[parameter.Declaration] = true
	defer delete(visiting, parameter.Declaration)
	bound := parameter.Bounds[0]
	base, rank := javaArrayTypeParts(strings.TrimSpace(bound.Original))
	base, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 {
		return nil, nil, false
	}
	if declaration := bound.TypeParameterBindings[base]; declaration != nil {
		for _, binding := range referenceTypeParameterBindings(ctx) {
			if binding.parameter.Declaration == declaration {
				return genericFamilyConcreteSourceBound(binding.parameter, binding.context, visiting)
			}
		}
		return nil, nil, false
	}
	origin := declaredJavaTypeOrigin(bound, ctx)
	scope := origin.nominalScope
	if origin.parameter != nil || origin.unresolved || scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.IsInterface || scope.IsAbstract || scope.IsEnum || len(scope.TypeParameters) != 0 {
		return nil, nil, false
	}
	file := findFileScopeForClassScope(scope, ctx)
	if file == nil || scope.Class.DeclarationNode.Type() != "class_declaration" {
		return nil, nil, false
	}
	return scope, file, true
}

func genericFamilySourceBoundHeader(scope *symbol.ClassScope, ctx Ctx) *symbol.TypeParamDeclaration {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return nil
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	declaring.currentFile = findFileScopeForClassScope(scope, ctx)
	declaring.syntheticTypeParameters = nil
	if declaring.currentFile == nil {
		return nil
	}
	for _, edge := range scope.ImplementedInterfaces {
		component, rank := javaArrayTypeParts(edge)
		_, args := parseJavaTypeString(component)
		if rank != 0 || len(args) != 1 || declarationOwnedExternalInterface(edge, scope, declaring) != "java.lang.Comparable" {
			continue
		}
		origin := declaredJavaTypeOrigin(symbol.JavaType{Original: args[0], TypeParameterBindings: symbol.VisibleTypeParamBindings(scope.TypeParameters)}, declaring)
		for _, parameter := range scope.TypeParameters {
			if parameter.Declaration != nil && origin.parameter == parameter.Declaration {
				if _, _, ok := genericFamilyConcreteSourceBound(parameter, declaring, map[*symbol.TypeParamDeclaration]bool{}); ok {
					return parameter.Declaration
				}
			}
		}
	}
	return nil
}

func genericFamilyCanonicalComparableHeader(typ string, ctx Ctx, occurrences ...*sitter.Node) bool {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || genericFamilySourceBoundHeader(scope, ctx) == nil {
		return false
	}
	component, rank := javaArrayTypeParts(typ)
	_, args := parseJavaTypeString(component)
	if rank != 0 || len(args) != 1 {
		return false
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	declaring.currentFile = findFileScopeForClassScope(scope, ctx)
	declaring.syntheticTypeParameters = nil
	if declaring.currentFile == nil || declarationOwnedExternalInterface(typ, scope, declaring) != "java.lang.Comparable" {
		return false
	}
	for _, edge := range scope.ImplementedInterfaces {
		if strings.TrimSpace(edge) == strings.TrimSpace(typ) {
			actual := canonicalClassFamilyHeaderTypeOccurrence(edge, scope.Class.DeclarationNode, declaring)
			for _, occurrence := range occurrences {
				if actual != nil && occurrence != nil && actual.Equal(occurrence) {
					return true
				}
			}
		}
	}
	return false
}

func certifyGenericFamilySourceBounds(plan *genericFamilyPlan, ctx Ctx) error {
	hasSource := false
	anchored := false
	for _, representation := range plan.representations {
		hasSource = hasSource || representation.sourceBound != nil
	}
	if !hasSource {
		return nil
	}
	for scope := range plan.members {
		declaration := genericFamilySourceBoundHeader(scope, ctx)
		representation, ok := plan.representations[declaration]
		anchored = anchored || (ok && representation.sourceBound != nil)
	}
	if !anchored {
		return fmt.Errorf("source-bound family has no declaration-owned Comparable header")
	}
	if len(plan.locals) != 0 || len(plan.anonymous) != 0 {
		return fmt.Errorf("source-bound local layout requires a closed slot proof")
	}
	certificate := &genericFamilySourceBoundCertificate{definitions: map[*symbol.Definition]*symbol.TypeParamDeclaration{}, constructors: map[*symbol.Definition]bool{}, slots: map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation{}}
	for declaration, representation := range plan.representations {
		if representation.sourceBound != nil {
			certificate.slots[declaration] = representation
		}
	}
	auditDefinition := func(owner *symbol.ClassScope, definition *symbol.Definition) error {
		if definition == nil {
			return nil
		}
		for declaration := range certificate.slots {
			if !definitionReferencesTypeParameterDeclaration(definition, declaration) {
				continue
			}
			use, ok := directOwnerTypeParameterForDefinition(owner, definition)
			if !ok || use.parameter.Declaration != declaration {
				return fmt.Errorf("source-bound value slot requires scalar declaration proof")
			}
			certificate.definitions[definition] = declaration
		}
		return nil
	}
	var auditChildren func(*symbol.ClassScope, *symbol.Definition) error
	auditChildren = func(owner *symbol.ClassScope, definition *symbol.Definition) error {
		if definition == nil {
			return nil
		}
		for _, child := range definition.Children {
			if err := auditDefinition(owner, child); err != nil {
				return err
			}
			if err := auditChildren(owner, child); err != nil {
				return err
			}
		}
		return nil
	}
	for owner := range plan.members {
		if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || owner.IsInner || findFileScopeForClassScope(owner, ctx) == nil {
			return fmt.Errorf("source-bound family has no named closed owner")
		}
		for _, field := range owner.Fields {
			if err := auditDefinition(owner, field); err != nil {
				return err
			}
			if certificate.definitions[field] != nil && field.DeclarationNode != nil {
				// The synthetic field-initializer body has no source constructor
				// definition in this certificate. Keep that layout unsupported.
				for _, declarator := range nodeutil.VariableDeclarators(field.DeclarationNode) {
					if declarator.ChildByFieldName("value") != nil {
						return fmt.Errorf("source-bound initializer has no callable storage proof")
					}
				}
			}
		}
		for _, method := range owner.Methods {
			if method == nil || method.DeclarationNode == nil {
				return fmt.Errorf("source-bound callable has no declaration origin")
			}
			if method.Constructor {
				for declaration := range certificate.slots {
					if !constructorBodySupportsCallableErasure(owner, method, declaration, ctx) {
						return fmt.Errorf("source-bound constructor storage remains unsupported")
					}
				}
				certificate.constructors[method] = true
			} else if err := auditDefinition(owner, method); err != nil {
				return err
			}
			for index, parameter := range method.Parameters {
				if err := auditDefinition(owner, parameter); err != nil {
					return err
				}
				if certificate.definitions[parameter] != nil && executionParameterIsVariadic(method, index) {
					return fmt.Errorf("source-bound array argument has no scalar storage proof")
				}
			}
			if err := auditChildren(owner, method); err != nil {
				return err
			}
		}
	}
	for declaration, representation := range certificate.slots {
		representation.sourceCertificate = certificate
		plan.representations[declaration] = representation
	}
	return nil
}

func genericFamilySourceBoundDefinition(owner *symbol.ClassScope, definition *symbol.Definition, ctx Ctx) (genericFamilyBinderRepresentation, bool) {
	empty := genericFamilyBinderRepresentation{}
	if owner == nil || definition == nil {
		return empty, false
	}
	use, ok := directOwnerTypeParameterForDefinition(owner, definition)
	if !ok {
		return empty, false
	}
	plan := canonicalGenericFamily(owner, ctx)
	if plan == nil {
		return empty, false
	}
	representation, ok := plan.representations[use.parameter.Declaration]
	if !ok || representation.sourceCertificate == nil || representation.sourceCertificate.definitions[definition] != use.parameter.Declaration {
		return empty, false
	}
	return representation, true
}

func genericFamilySourceBoundGoType(representation genericFamilyBinderRepresentation, ctx Ctx) ast.Expr {
	if representation.sourceBound == nil || representation.sourceFile == nil || representation.sourceCertificate == nil {
		return nil
	}
	return &ast.StarExpr{X: qualifiedNameExpr(representation.sourceBound.Class.Name, findJavaPackageForClassScope(representation.sourceBound), ctx)}
}

// Invocation parsing has already applied Java's logical conversion. This
// transport changes only the storage view, evaluates once and preserves null.
func genericFamilySourceBoundArguments(args []ast.Expr, resolution *methodResolution, nodes *sitter.Node, expected []string, ctx Ctx) {
	if resolution == nil || resolution.owner == nil || resolution.def == nil || nodes == nil {
		return
	}
	for index, parameter := range resolution.def.Parameters {
		if index >= len(args) || index >= len(expected) || executionParameterIsVariadic(resolution.def, index) {
			continue
		}
		representation, ok := genericFamilySourceBoundDefinition(resolution.owner, parameter, ctx)
		if !ok {
			continue
		}
		node := nodeutilArgument(nodes, index)
		if node == nil {
			continue
		}
		if isStaticallyNullReference(node) {
			args[index] = ast.NewIdent("nil")
			continue
		}
		logical := expected[index]
		component, rank := javaArrayTypeParts(logical)
		base, arguments := parseJavaTypeString(component)
		if rank != 0 || len(arguments) != 0 {
			continue
		}
		origin := declaredJavaTypeOrigin(symbol.JavaType{Original: logical, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
		if origin.parameter != nil {
			if caller := canonicalGenericFamily(ctx.currentClass, ctx); caller != nil {
				if physical, present := caller.representations[origin.parameter]; present && physical.sourceBound == representation.sourceBound && physical.sourceCertificate != nil {
					continue
				}
			}
			continue
		}
		actual := origin.nominalScope
		if actual == nil || actual == representation.sourceBound || !classInheritsFrom(actual, representation.sourceBound, ctx) {
			continue
		}
		args[index] = nullableSuperclassEdgeView(args[index], base, representation.erasure, actual, representation.sourceBound, ctx)
	}
}

func nodeutilArgument(nodes *sitter.Node, index int) *sitter.Node {
	children := nodeutil.SemanticNamedChildrenOf(nodes)
	if index < 0 || index >= len(children) {
		return nil
	}
	return children[index]
}

// Equality consumes the erased reference without the concrete logical checkcast
// that a later field selection or assignment requires. Only a certified scalar
// class slot can have this different physical representation.
func genericFamilySourceBoundIdentityOperand(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	var owner *symbol.ClassScope
	var definition *symbol.Definition
	switch node.Type() {
	case "identifier":
		if ctx.currentClass == nil || (ctx.localScope != nil && ctx.localScope.FindVariable(node.Content(source)) != nil) {
			return false
		}
		resolution := findFieldResolutionInHierarchy(ctx.currentClass, node.Content(source), ctx)
		if resolution != nil {
			owner, definition = resolution.owner, resolution.def
		}
	case "field_access":
		object, field := node.ChildByFieldName("object"), node.ChildByFieldName("field")
		if object == nil || field == nil {
			return false
		}
		target := resolveInvocationTarget(object, ctx, source)
		if target == nil || target.classScope == nil {
			return false
		}
		resolution := findFieldResolutionInHierarchy(target.classScope, field.Content(source), ctx)
		if resolution != nil {
			owner, definition = resolution.owner, resolution.def
		}
	case "method_invocation":
		_, resolution := resolvedInstanceInvocation(node, ctx, source)
		if resolution != nil {
			owner, definition = resolution.owner, resolution.def
		}
	default:
		return false
	}
	if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil || owner.Class.DeclarationNode.Type() != "class_declaration" || definition == nil || definition.DeclarationNode == nil || definition.IsStatic || definition.Constructor {
		return false
	}
	_, ok := genericFamilySourceBoundDefinition(owner, definition, ctx)
	return ok
}

func genericFamilySourceBoundHasArguments(resolution *methodResolution, ctx Ctx) bool {
	if resolution == nil || resolution.owner == nil || resolution.def == nil {
		return false
	}
	for _, parameter := range resolution.def.Parameters {
		if _, ok := genericFamilySourceBoundDefinition(resolution.owner, parameter, ctx); ok {
			return true
		}
	}
	return false
}

// Keep Java's logical conversion before physical transport. The one additional
// conversion is a certified source-subclass widening, which must preserve null
// just as the declared bound's JVM reference parameter does.
func parseGenericFamilySourceBoundArguments(resolution *methodResolution, nodes *sitter.Node, source []byte, ctx Ctx, expected []string) []ast.Expr {
	if nodes == nil {
		return nil
	}
	argumentNodes := nodeutil.SemanticNamedChildrenOf(nodes)
	laterEffects := javaLaterExpressionEffects(argumentNodes, source, ctx)
	args := make([]ast.Expr, 0, len(argumentNodes))
	for index, node := range argumentNodes {
		logical := ""
		argumentCtx := ctx.Clone()
		if index < len(expected) && strings.TrimSpace(expected[index]) != "" {
			logical = expected[index]
			argumentCtx.expectedType = logical
			argumentCtx.expectedTypeRoot = node
		}
		parsed := ParseExpr(node, source, argumentCtx)
		widened := false
		if index < len(resolution.def.Parameters) && !executionParameterIsVariadic(resolution.def, index) {
			if representation, ok := genericFamilySourceBoundDefinition(resolution.owner, resolution.def.Parameters[index], ctx); ok {
				expectedOrigin := declaredJavaTypeOrigin(symbol.JavaType{Original: logical, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
				var actualOrigin inferredJavaTypeOrigin
				actual, known := inferExprJavaType(node, argumentCtx, source, &actualOrigin)
				_, rank := javaArrayTypeParts(actual)
				if known && rank == 0 && expectedOrigin.nominalScope == representation.sourceBound && expectedOrigin.parameter == nil && !expectedOrigin.unresolved && actualOrigin.nominalScope != nil && actualOrigin.parameter == nil && !actualOrigin.unresolved && classInheritsFrom(actualOrigin.nominalScope, representation.sourceBound, ctx) {
					parsed = nullableSuperclassEdgeView(parsed, actual, logical, actualOrigin.nominalScope, representation.sourceBound, ctx)
					widened = true
				}
			}
		}
		if !widened {
			if logical != "" && usesNullableValueStorage(logical) && expressionAlwaysProducesNull(node) {
				if isJavaStringType(logical) {
					parsed = javaStringNullExpr(ctx)
				} else {
					parsed = zeroValueForType(javaTypeStringToGoTypeExpr(logical, inScopeTypeParameters(ctx), ctx))
				}
			}
			parsed = coerceArgumentToExpectedType(parsed, node, logical, ctx, source)
		}
		if laterEffects[index] {
			valueType := logical
			if valueType == "" {
				valueType, _ = inferExprJavaType(node, ctx, source)
			}
			parsed = snapshotJavaExpressionValueForType(parsed, valueType, ctx)
		}
		args = append(args, parsed)
	}
	return args
}
