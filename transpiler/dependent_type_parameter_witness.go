package transpiler

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// dependentTypeWitnessEdge is one direct Java type-variable bound such as
// T extends B. Source values may be converted to target values by the hidden
// function parameter. Declaration identity, rather than source spelling, keeps
// shadowed type parameters distinct after local/helper hoisting.
type dependentTypeWitnessEdge struct {
	source          symbol.TypeParam
	targetParameter *symbol.TypeParamDeclaration
	targetGoName    string
	targetJavaType  string
	name            string
}

type dependentTypeWitnessPlan struct {
	edges []dependentTypeWitnessEdge
}

// planConcreteDependentTypeWitnesses identifies the dependent edges that Go's
// constraint system cannot represent. Interface-rooted chains need no evidence:
// generated concrete pointers implement the interface directly. A concrete
// class root is different because Java subclasses are represented by embedded
// subobjects, not Go pointer subtyping.
func planConcreteDependentTypeWitnesses(def *symbol.Definition, source []byte, ctx Ctx) *dependentTypeWitnessPlan {
	if def == nil || (!def.IsStatic && !def.Constructor) || len(def.TypeParameters) == 0 {
		return nil
	}
	usedNames := affineLoopUsedNames(def.DeclarationNode, source, ctx)
	plan := &dependentTypeWitnessPlan{edges: concreteDependentTypeWitnessEdges(def, ctx)}
	for index := range plan.edges {
		baseName := fmt.Sprintf("__java2goDependentWitness%d", index+1)
		name := synchronizedUniqueLocalName(baseName, usedNames)
		usedNames[name] = struct{}{}
		plan.edges[index].name = name
	}
	if len(plan.edges) == 0 {
		return nil
	}
	return plan
}

func concreteDependentTypeWitnessEdges(def *symbol.Definition, ctx Ctx) []dependentTypeWitnessEdge {
	if def == nil || (!def.IsStatic && !def.Constructor) || len(def.TypeParameters) == 0 {
		return nil
	}
	planningCtx, owner := dependentTypeWitnessDefinitionContext(def, ctx)
	lookup := newTypeParameterLookup(def.TypeParameters)
	edges := make([]dependentTypeWitnessEdge, 0, len(def.TypeParameters))
	for _, parameter := range def.TypeParameters {
		if concreteTypeParameterRootScope(parameter, lookup, planningCtx, nil) == nil {
			continue
		}
		if edge, ok := concreteDependentTypeWitnessEdge(parameter, lookup, planningCtx); ok {
			if edge.targetParameter == nil && owner != nil {
				edge.targetJavaType = qualifyJavaTypeInDeclaringContext(edge.targetJavaType, owner)
			}
			edges = append(edges, edge)
		}
	}
	return edges
}

func dependentTypeWitnessDefinitionContext(def *symbol.Definition, fallback Ctx) (Ctx, *symbol.ClassScope) {
	if def == nil {
		return fallback, nil
	}
	var owner *symbol.ClassScope
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			for _, top := range file.TopLevelClasses {
				if visitClassScopes(top, func(scope *symbol.ClassScope) bool {
					for _, method := range scope.Methods {
						if method == def {
							owner = scope
							return true
						}
					}
					return false
				}) {
					break
				}
			}
			if owner != nil {
				break
			}
		}
		if owner != nil {
			break
		}
	}
	if owner == nil {
		return fallback, nil
	}
	result := fallback.Clone()
	result.currentClass = owner
	if file := findFileScopeForClassScope(owner); file != nil {
		result.currentFile = file
	}
	return result, owner
}

func methodTypeParameterRequiresConcreteWitness(
	def *symbol.Definition,
	declaration *symbol.TypeParamDeclaration,
	ctx Ctx,
) bool {
	if declaration == nil {
		return false
	}
	for _, edge := range concreteDependentTypeWitnessEdges(def, ctx) {
		if edge.source.Declaration == declaration {
			return true
		}
	}
	return false
}

func methodUsesConcreteDependentTypeWitnesses(def *symbol.Definition, ctx Ctx) bool {
	return len(concreteDependentTypeWitnessEdges(def, ctx)) > 0
}

func concreteDependentTypeWitnessEdge(
	parameter symbol.TypeParam,
	lookup typeParameterLookup,
	ctx Ctx,
) (dependentTypeWitnessEdge, bool) {
	for _, bound := range parameter.Bounds {
		base, arguments := parseJavaTypeString(strings.TrimSpace(bound.Original))
		if len(arguments) == 0 {
			if dependency, ok := lookup.resolve(bound, strings.TrimSpace(base)); ok {
				if concreteTypeParameterRootScope(dependency, lookup, ctx, nil) == nil {
					continue
				}
				return dependentTypeWitnessEdge{
					source:          parameter,
					targetParameter: dependency.Declaration,
					targetGoName:    dependency.EmittedName(),
					targetJavaType:  dependency.Name,
				}, true
			}
		}
		if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && !scope.IsInterface {
			return dependentTypeWitnessEdge{
				source:         parameter,
				targetJavaType: strings.TrimSpace(bound.Original),
			}, true
		}
	}
	return dependentTypeWitnessEdge{}, false
}

// concreteTypeParameterRootScope follows only declaration-resolved dependent
// bounds. It returns a non-interface class root, or nil for interface/Object,
// malformed, and cyclic chains.
func concreteTypeParameterRootScope(
	parameter symbol.TypeParam,
	lookup typeParameterLookup,
	ctx Ctx,
	visiting map[typeParameterIdentityKey]bool,
) *symbol.ClassScope {
	// Multiple/intersection and parameterized concrete bounds need one witness
	// per retained interface/nested type argument. Keep those shapes on the
	// existing constraint path until that complete ABI is emitted.
	if len(parameter.Bounds) != 1 {
		return nil
	}
	if visiting == nil {
		visiting = make(map[typeParameterIdentityKey]bool, len(lookup.byName))
	}
	identity := identityKeyForTypeParameter(parameter)
	if visiting[identity] {
		return nil
	}
	visiting[identity] = true
	defer delete(visiting, identity)

	for _, bound := range parameter.Bounds {
		base, arguments := parseJavaTypeString(strings.TrimSpace(bound.Original))
		if len(arguments) == 0 {
			if dependency, ok := lookup.resolve(bound, strings.TrimSpace(base)); ok {
				if root := concreteTypeParameterRootScope(dependency, lookup, ctx, visiting); root != nil {
					return root
				}
				continue
			}
		}
		if len(arguments) != 0 {
			return nil
		}
		scope := resolveClassScopeByQualifiedName(ctx, base)
		if scope != nil && !scope.IsInterface {
			return scope
		}
	}
	return nil
}

func (plan *dependentTypeWitnessPlan) hasSource(declaration *symbol.TypeParamDeclaration) bool {
	if plan == nil || declaration == nil {
		return false
	}
	for _, edge := range plan.edges {
		if edge.source.Declaration == declaration {
			return true
		}
	}
	return false
}

func dependentTypeWitnessParameterFields(plan *dependentTypeWitnessPlan, ctx Ctx) []*ast.Field {
	if plan == nil {
		return nil
	}
	fields := make([]*ast.Field, 0, len(plan.edges))
	for _, edge := range plan.edges {
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{{Name: edge.name}},
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Type: &ast.Ident{Name: edge.source.EmittedName()},
				}}},
				Results: &ast.FieldList{List: []*ast.Field{{
					Type: dependentTypeWitnessTargetExpr(edge, ctx),
				}}},
			},
		})
	}
	return fields
}

func dependentTypeWitnessTargetExpr(edge dependentTypeWitnessEdge, ctx Ctx) ast.Expr {
	if edge.targetGoName != "" {
		return &ast.Ident{Name: edge.targetGoName}
	}
	return javaTypeStringToGoTypeExpr(edge.targetJavaType, inScopeTypeParameters(ctx), ctx)
}

func visibleTypeParameterDeclarationForJavaType(javaType string, ctx Ctx) *symbol.TypeParamDeclaration {
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	if rank != 0 {
		return nil
	}
	base, arguments := parseJavaTypeString(base)
	if len(arguments) != 0 {
		return nil
	}
	parameter, ok := newTypeParameterLookup(visibleTypeParameterDeclarations(ctx)).resolve(
		symbol.JavaType{},
		strings.TrimSpace(base),
	)
	if !ok {
		return nil
	}
	return parameter.Declaration
}

func dependentTypeWitnessPathExpr(
	value ast.Expr,
	actualDeclaration *symbol.TypeParamDeclaration,
	expectedDeclaration *symbol.TypeParamDeclaration,
	expectedScope *symbol.ClassScope,
	ctx Ctx,
) (ast.Expr, bool) {
	plan := ctx.dependentTypeWitnesses
	if plan == nil || actualDeclaration == nil {
		return nil, false
	}
	result := value
	current := actualDeclaration
	seen := map[*symbol.TypeParamDeclaration]struct{}{}
	for current != nil {
		if expectedDeclaration != nil && current == expectedDeclaration {
			return result, true
		}
		if _, duplicate := seen[current]; duplicate {
			return nil, false
		}
		seen[current] = struct{}{}
		var selected *dependentTypeWitnessEdge
		for index := range plan.edges {
			if plan.edges[index].source.Declaration == current {
				selected = &plan.edges[index]
				break
			}
		}
		if selected == nil {
			return nil, false
		}
		result = &ast.CallExpr{Fun: &ast.Ident{Name: selected.name}, Args: []ast.Expr{result}}
		if selected.targetParameter != nil {
			current = selected.targetParameter
			continue
		}
		if expectedScope != nil {
			base, _ := parseJavaTypeString(selected.targetJavaType)
			if targetScope := resolveClassScopeByQualifiedName(ctx, base); targetScope == expectedScope {
				return result, true
			}
		}
		return nil, false
	}
	return nil, false
}

// plannedDependentTypeParameterWideningExpr uses the hidden evidence carried by
// the current generic method. The returned bool distinguishes "no plan" from a
// valid expression so the legacy interface-root bridge remains available.
func plannedDependentTypeParameterWideningExpr(
	value ast.Expr,
	actualJavaType string,
	expectedJavaType string,
	ctx Ctx,
) (ast.Expr, bool) {
	actual := visibleTypeParameterDeclarationForJavaType(actualJavaType, ctx)
	expected := visibleTypeParameterDeclarationForJavaType(expectedJavaType, ctx)
	if actual == nil || expected == nil {
		return nil, false
	}
	return dependentTypeWitnessPathExpr(value, actual, expected, nil, ctx)
}

// dependentTypeParameterConcreteViewExpr projects a concrete-rooted method
// type parameter to the generated superclass subobject that owns a selected
// field or method.
func dependentTypeParameterConcreteViewExpr(
	value ast.Expr,
	actualJavaType string,
	expectedScope *symbol.ClassScope,
	ctx Ctx,
) (ast.Expr, bool) {
	actual := visibleTypeParameterDeclarationForJavaType(actualJavaType, ctx)
	if actual == nil || expectedScope == nil {
		return nil, false
	}
	return dependentTypeWitnessPathExpr(value, actual, nil, expectedScope, ctx)
}

func projectDependentTypeParameterReceiver(
	value ast.Expr,
	receiverNode *sitter.Node,
	expectedScope *symbol.ClassScope,
	ctx Ctx,
	source []byte,
) ast.Expr {
	if value == nil || receiverNode == nil || expectedScope == nil || ctx.dependentTypeWitnesses == nil {
		return value
	}
	javaType, ok := inferExprJavaType(receiverNode, ctx, source)
	if !ok {
		return value
	}
	if projected, ok := dependentTypeParameterConcreteViewExpr(value, javaType, expectedScope, ctx); ok {
		return projected
	}
	return value
}

func methodInvocationTypeArgumentJavaTypes(
	def *symbol.Definition,
	invocationNode *sitter.Node,
	ctx Ctx,
	source []byte,
) []string {
	if def == nil || len(def.TypeParameters) == 0 {
		return nil
	}
	if invocationNode != nil {
		if typeArguments := invocationNode.ChildByFieldName("type_arguments"); typeArguments != nil {
			explicit := nodeutil.NamedChildrenOf(typeArguments)
			if len(explicit) == len(def.TypeParameters) {
				result := make([]string, len(explicit))
				for index, argument := range explicit {
					result[index] = strings.TrimSpace(argument.Content(source))
				}
				return result
			}
		}
	}

	bindings := genericArrayInvocationTypeBindings(def, invocationNode, ctx, source)
	result := make([]string, len(def.TypeParameters))
	for index, parameter := range def.TypeParameters {
		result[index] = strings.TrimSpace(bindings[parameter.Name])
		if result[index] == "" {
			result[index] = rawTypeParameterErasure(parameter, def.TypeParameters)
		}
	}
	return result
}

func dependentTypeWitnessInvocationArguments(
	def *symbol.Definition,
	invocationNode *sitter.Node,
	ctx Ctx,
	source []byte,
) []ast.Expr {
	edges := concreteDependentTypeWitnessEdges(def, ctx)
	if len(edges) == 0 {
		return nil
	}
	javaTypes := methodInvocationTypeArgumentJavaTypes(def, invocationNode, ctx, source)
	if len(javaTypes) != len(def.TypeParameters) {
		return nil
	}
	byDeclaration := make(map[*symbol.TypeParamDeclaration]string, len(javaTypes))
	for index, parameter := range def.TypeParameters {
		byDeclaration[parameter.Declaration] = javaTypes[index]
	}
	result := make([]ast.Expr, 0, len(edges))
	for _, edge := range edges {
		sourceJavaType := byDeclaration[edge.source.Declaration]
		targetJavaType := edge.targetJavaType
		if edge.targetParameter != nil {
			targetJavaType = byDeclaration[edge.targetParameter]
		}
		projection := dependentTypeProjectionWitnessExpr(sourceJavaType, targetJavaType, ctx)
		if projection == nil {
			return nil
		}
		result = append(result, projection)
	}
	return result
}

func dependentTypeProjectionWitnessExpr(sourceJavaType, targetJavaType string, ctx Ctx) ast.Expr {
	sourceJavaType = strings.TrimSpace(sourceJavaType)
	targetJavaType = strings.TrimSpace(targetJavaType)
	if sourceJavaType == "" || targetJavaType == "" {
		return nil
	}
	sourceGoType := javaTypeStringToGoTypeExpr(sourceJavaType, inScopeTypeParameters(ctx), ctx)
	targetGoType := javaTypeStringToGoTypeExpr(targetJavaType, inScopeTypeParameters(ctx), ctx)
	const valueName = "__java2goProjectedValue"
	value := &ast.Ident{Name: valueName}

	if javaInferenceSameType(sourceJavaType, targetJavaType, ctx) {
		return dependentTypeProjectionFunc(sourceGoType, targetGoType, value, false, ctx)
	}

	sourceDeclaration := visibleTypeParameterDeclarationForJavaType(sourceJavaType, ctx)
	targetDeclaration := visibleTypeParameterDeclarationForJavaType(targetJavaType, ctx)
	if sourceDeclaration != nil {
		var targetScope *symbol.ClassScope
		if targetDeclaration == nil {
			base, _ := parseJavaTypeString(targetJavaType)
			targetScope = resolveClassScopeByQualifiedName(ctx, base)
		}
		if projected, ok := dependentTypeWitnessPathExpr(
			value,
			sourceDeclaration,
			targetDeclaration,
			targetScope,
			ctx,
		); ok {
			return dependentTypeProjectionFunc(sourceGoType, targetGoType, projected, false, ctx)
		}
	}

	sourceBase, _ := parseJavaTypeString(sourceJavaType)
	targetBase, _ := parseJavaTypeString(targetJavaType)
	sourceScope := resolveClassScopeByQualifiedName(ctx, sourceBase)
	targetScope := resolveClassScopeByQualifiedName(ctx, targetBase)
	if sourceScope == nil || targetScope == nil || !javaReferenceTypeAssignable(sourceScope, targetScope, ctx) {
		return nil
	}
	projected := sourceClassViewExpr(sourceScope, targetScope, value, ctx)
	if projected == nil {
		return nil
	}
	return dependentTypeProjectionFunc(sourceGoType, targetGoType, projected, true, ctx)
}

func dependentTypeProjectionFunc(
	sourceGoType ast.Expr,
	targetGoType ast.Expr,
	projected ast.Expr,
	nullGuard bool,
	ctx Ctx,
) ast.Expr {
	const valueName = "__java2goProjectedValue"
	body := &ast.BlockStmt{}
	if nullGuard {
		body.List = append(body.List, &ast.IfStmt{
			Cond: stdjavaCall(ctx, "JavaReferenceEqual", &ast.Ident{Name: valueName}, &ast.Ident{Name: "nil"}),
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
				zeroValueForType(targetGoType),
			}}}},
		})
	}
	body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{projected}})
	return &ast.FuncLit{
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: valueName}},
				Type:  sourceGoType,
			}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: targetGoType}}},
		},
		Body: body,
	}
}
