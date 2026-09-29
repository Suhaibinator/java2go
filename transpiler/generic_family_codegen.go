package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// This cache belongs to one file-render context, created after resolution.
// Cached plans contain only source declarations and are never extended in place
// with anonymous emission scopes. Ctx{} callers deliberately remain uncached.
type genericFamilyAnalysis struct {
	ready bool
	plans []*genericFamilyPlan
}

func canonicalGenericFamily(scope *symbol.ClassScope, ctx Ctx) *genericFamilyPlan {
	if scope == nil {
		return nil
	}
	var plans []*genericFamilyPlan
	if ctx.genericFamilies != nil {
		if !ctx.genericFamilies.ready {
			inventoryCtx := ctx.Clone()
			inventoryCtx.genericFamilies = nil
			ctx.genericFamilies.plans = discoverCanonicalGenericFamilies(inventoryCtx)
			ctx.genericFamilies.ready = true
		}
		plans = ctx.genericFamilies.plans
	} else {
		plans = discoverCanonicalGenericFamilies(ctx)
	}
	for _, plan := range plans {
		if _, member := plan.members[scope]; member {
			return plan
		}
		if scope.Class != nil && scope.Class.DeclarationNode != nil {
			file := findFileScopeForClassScope(scope.Enclosing)
			if file == nil {
				file = ctx.currentFile
			}
			key := genericFamilyKey(file, scope.Class.DeclarationNode)
			_, anonymous := plan.anonymous[key]
			_, local := plan.locals[key]
			if anonymous || local {
				instance := *plan
				instance.members = make(map[*symbol.ClassScope]struct{}, len(plan.members)+1)
				for member := range plan.members {
					instance.members[member] = struct{}{}
				}
				instance.members[scope] = struct{}{}
				instance.binders = make(map[*symbol.TypeParamDeclaration]struct{}, len(plan.binders))
				for declaration := range plan.binders {
					instance.binders[declaration] = struct{}{}
				}
				instance.representations = make(map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation, len(plan.representations))
				for declaration, representation := range plan.representations {
					instance.representations[declaration] = representation
				}
				if err := instance.addBinderRepresentations(scope, ctx); err != nil {
					return nil
				}
				return &instance
			}
		}
	}
	return nil
}

func discoverCanonicalGenericFamilies(ctx Ctx) []*genericFamilyPlan {
	seen := map[*symbol.ClassScope]bool{}
	var plans []*genericFamilyPlan
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			if method == nil || len(method.TypeParameters) == 0 {
				continue
			}
			types := []string{method.OriginalType}
			for _, parameter := range method.Parameters {
				types = append(types, parameter.OriginalType)
			}
			for _, typ := range types {
				base, args := parseJavaTypeString(typ)
				if len(args) == 0 {
					continue
				}
				demanded := false
				for _, parameter := range method.TypeParameters {
					demanded = demanded || javaTypeContainsParameter(typ, parameter.Name)
				}
				if !demanded {
					continue
				}
				seed := resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), base)
				if seed == nil || len(seed.TypeParameters) == 0 || leafObjectMemberErasureEligible(seed, ctx) {
					continue
				}
				if seen[seed] {
					continue
				}
				seen[seed] = true
				plan, err := planGenericFamily(seed, ctx)
				if err != nil {
					continue
				}

				plans = append(plans, plan)
			}
		}
	}
	return plans
}

func genericFamilyPhysicalJavaType(typ string, ctx Ctx) string {
	if ctx.localScope == nil || ctx.localScope.Constructor || ctx.localScope.IsStatic || canonicalGenericFamily(ctx.currentClass, ctx) == nil {
		return typ
	}
	plan := canonicalGenericFamily(ctx.currentClass, ctx)
	bindings := map[string]string{}
	for _, parameter := range ctx.currentClass.TypeParameters {
		if parameter.Declaration != nil && visibleTypeParameterDeclarationForJavaType(parameter.Name, ctx) == parameter.Declaration {
			representation, present := plan.representations[parameter.Declaration]
			if !present {
				continue
			}
			bindings[parameter.Name] = representation.erasure
			bindings[parameter.EmittedName()] = representation.erasure
		}
	}
	return substituteJavaTypeParameters(typ, bindings)
}

// Rewrite type syntax, never declaration/value identifiers. Receiver binders
// disappear with a canonical alias, but package-qualified selectors, field
// names and shadowing method binders keep their original identities.
func genericFamilyPhysicalGoType(expr ast.Expr, parameters []symbol.TypeParam, ctx Ctx) ast.Expr {
	if expr == nil {
		return nil
	}
	switch typ := expr.(type) {
	case *ast.Ident:
		for _, parameter := range parameters {
			if typ.Name == parameter.EmittedName() {
				if physical := genericFamilyBinderGoType(parameter, ctx); physical != nil {
					return physical
				}
			}
		}
	case *ast.StarExpr:
		typ.X = genericFamilyPhysicalGoType(typ.X, parameters, ctx)
	case *ast.ArrayType:
		typ.Elt = genericFamilyPhysicalGoType(typ.Elt, parameters, ctx)
	case *ast.MapType:
		typ.Key = genericFamilyPhysicalGoType(typ.Key, parameters, ctx)
		typ.Value = genericFamilyPhysicalGoType(typ.Value, parameters, ctx)
	case *ast.ChanType:
		typ.Value = genericFamilyPhysicalGoType(typ.Value, parameters, ctx)
	case *ast.IndexExpr:
		typ.X = genericFamilyPhysicalGoType(typ.X, parameters, ctx)
		typ.Index = genericFamilyPhysicalGoType(typ.Index, parameters, ctx)
	case *ast.IndexListExpr:
		typ.X = genericFamilyPhysicalGoType(typ.X, parameters, ctx)
		for index := range typ.Indices {
			typ.Indices[index] = genericFamilyPhysicalGoType(typ.Indices[index], parameters, ctx)
		}
	case *ast.FuncType:
		genericFamilyPhysicalFields(typ.Params, parameters, ctx)
		genericFamilyPhysicalFields(typ.Results, parameters, ctx)
	case *ast.InterfaceType:
		genericFamilyPhysicalFields(typ.Methods, parameters, ctx)
	case *ast.StructType:
		genericFamilyPhysicalFields(typ.Fields, parameters, ctx)
	case *ast.ParenExpr:
		typ.X = genericFamilyPhysicalGoType(typ.X, parameters, ctx)
	case *ast.Ellipsis:
		typ.Elt = genericFamilyPhysicalGoType(typ.Elt, parameters, ctx)
	}
	return expr
}

func genericFamilyPhysicalFields(fields *ast.FieldList, parameters []symbol.TypeParam, ctx Ctx) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		field.Type = genericFamilyPhysicalGoType(field.Type, parameters, ctx)
	}
}

func canonicalGenericInterfaceSpecs(name string, methods *ast.FieldList, parameters []symbol.TypeParam, ctx Ctx) []ast.Spec {
	if len(parameters) == 0 || canonicalGenericFamily(ctx.currentClass, ctx) == nil {
		return nil
	}
	genericFamilyPhysicalFields(methods, parameters, ctx)
	raw := availableCanonicalGenericName(name + "Java2goErased")
	return []ast.Spec{
		&ast.TypeSpec{Name: ast.NewIdent(raw), Type: &ast.InterfaceType{Methods: methods}},
		&ast.TypeSpec{Name: ast.NewIdent(name), TypeParams: &ast.FieldList{List: makeTypeParamFieldsInContext(parameters, ctx)}, Assign: token.Pos(1), Type: ast.NewIdent(raw)},
	}
}

func canonicalGenericReceiverBody(method *ast.FuncDecl, ctx Ctx) {
	if canonicalGenericFamily(ctx.currentClass, ctx) == nil {
		return
	}
	parameters := ctx.currentClass.TypeParameters
	genericFamilyPhysicalFields(method.Type.Params, parameters, ctx)
	genericFamilyPhysicalFields(method.Type.Results, parameters, ctx)
	ast.Inspect(method.Body, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.ValueSpec:
			value.Type = genericFamilyPhysicalGoType(value.Type, parameters, ctx)
		case *ast.TypeAssertExpr:
			value.Type = genericFamilyPhysicalGoType(value.Type, parameters, ctx)
		case *ast.CompositeLit:
			value.Type = genericFamilyPhysicalGoType(value.Type, parameters, ctx)
		case *ast.FuncLit:
			genericFamilyPhysicalGoType(value.Type, parameters, ctx)
		case *ast.CallExpr:
			switch fun := value.Fun.(type) {
			case *ast.Ident:
				if (fun.Name == "new" || fun.Name == "make") && len(value.Args) > 0 {
					value.Args[0] = genericFamilyPhysicalGoType(value.Args[0], parameters, ctx)
				}
			}
		}
		return true
	})
}

func genericFamilyMethodPlan(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (*directOwnerOverrideBridgeFamilyPlan, bool) {
	if owner == nil || method == nil || method.IsPrivate || method.IsStatic || method.Constructor || len(method.TypeParameters) != 0 {
		return nil, false
	}
	layout := canonicalGenericFamily(owner, ctx)
	if layout == nil {
		return nil, false
	}
	if len(methodDirectOwnerTypeParameterDeclarations(owner, method)) == 0 {
		return nil, false
	}
	plan := &directOwnerOverrideBridgeFamilyPlan{owner: owner, method: method, erasedResult: "void", requiresErasedView: true}
	for _, parameter := range method.Parameters {
		typ, _, ok := directOwnerOverrideBridgeErasedType(owner, parameter, ctx)
		if !ok {
			return nil, false
		}
		plan.erasedParameters = append(plan.erasedParameters, typ)
	}
	if !javaMethodResultIsVoid(method) {
		typ, _, ok := directOwnerOverrideBridgeErasedType(owner, method, ctx)
		if !ok {
			return nil, false
		}
		plan.erasedResult = typ
	}
	for descendant := range layout.members {
		if descendant == owner {
			continue
		}
		args := mapClassTypeArgumentStringsToAncestor(descendant, descendant.GoTypeParameterNames(), owner, ctx)
		if len(args) != len(owner.TypeParameters) {
			continue
		}
		parameters, result := mappedOverrideSourceSignature(owner, method, args)
		matches := directDeclaredOverrides(owner, descendant, method, parameters, result, ctx)
		for _, override := range matches {
			bridge, required, ok := planDirectOwnerSpecializedOverride(owner, method, descendant, override, args, parameters, result, plan.erasedParameters, plan.erasedResult, ctx)
			if !ok {
				return nil, false
			}
			if required {
				plan.overrides = append(plan.overrides, bridge)
			}
		}
	}
	return plan, true
}

func genericFamilySpecializedMethod(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (directOwnerSpecializedOverrideBridgeSelection, bool) {
	empty := directOwnerSpecializedOverrideBridgeSelection{}
	layout := canonicalGenericFamily(owner, ctx)
	if layout == nil {
		return empty, false
	}
	var selected directOwnerSpecializedOverrideBridgeSelection
	for ancestor := range layout.members {
		if ancestor == owner || len(ancestor.TypeParameters) == 0 {
			continue
		}
		args := mapClassTypeArgumentStringsToAncestor(owner, owner.GoTypeParameterNames(), ancestor, ctx)
		if len(args) != len(ancestor.TypeParameters) {
			continue
		}
		for _, candidate := range ancestor.Methods {
			if candidate == nil || candidate.OriginalName != method.OriginalName {
				continue
			}
			family, ok := genericFamilyMethodPlan(ancestor, candidate, ctx)
			if !ok {
				continue
			}
			parameters, result := mappedOverrideSourceSignature(ancestor, candidate, args)
			matches := directDeclaredOverrides(ancestor, owner, candidate, parameters, result, ctx)
			matched := false
			for _, match := range matches {
				matched = matched || match == method
			}
			if !matched {
				continue
			}
			bridge, required, ok := planDirectOwnerSpecializedOverride(ancestor, candidate, owner, method, args, parameters, result, family.erasedParameters, family.erasedResult, ctx)
			if !ok || !required {
				continue
			}
			if selected.family != nil && overrideBridgeFamilyDescriptorKey(selected.family, ctx) != overrideBridgeFamilyDescriptorKey(family, ctx) {
				return empty, false
			}
			selected = directOwnerSpecializedOverrideBridgeSelection{family: family, bridge: bridge}
		}
	}
	return selected, selected.family != nil
}

func genericFamilyPublicBridgeWrapper(declaration *ast.FuncDecl, bridge *ast.FuncDecl, executionName string, ctx Ctx) *ast.FuncDecl {
	params := cloneFieldList(bridge.Type.Params)
	params.List = params.List[1:]
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(declaration.Recv.List[0].Names[0].Name), Sel: ast.NewIdent(bridge.Name.Name)}, Args: append([]ast.Expr{newExecutionExpr(ctx)}, methodCallArgs(params)...)}
	var statement ast.Stmt = &ast.ExprStmt{X: call}
	if bridge.Type.Results != nil && len(bridge.Type.Results.List) > 0 {
		statement = &ast.ReturnStmt{Results: []ast.Expr{call}}
	}
	return &ast.FuncDecl{Doc: declaration.Doc, Name: ast.NewIdent(declaration.Name.Name), Recv: cloneFieldList(declaration.Recv), Type: &ast.FuncType{Params: params, Results: cloneFieldList(bridge.Type.Results)}, Body: &ast.BlockStmt{List: []ast.Stmt{statement}}}
}

func genericFamilyIsObjectErasure(owner *symbol.ClassScope, erasure string, ctx Ctx) bool {
	return (strings.TrimSpace(erasure) == "Object" || strings.TrimSpace(erasure) == "java.lang.Object") && canonicalGenericFamily(owner, ctx) != nil
}
