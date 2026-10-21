package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// sourceInterfaceVirtualEntry is separate from the hidden implementation body:
// a parent subobject must dispatch interface calls through its initialized self,
// while explicit super calls must continue to enter the parent body directly.
type sourceInterfaceVirtualEntry struct {
	owner     *symbol.ClassScope
	method    *symbol.Definition
	name      string
	signature *ast.FuncType
}

func planSourceInterfaceVirtualEntry(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceInterfaceVirtualEntry {
	if owner == nil || owner.Class == nil || !owner.IsInterface || len(owner.TypeParameters) != 0 || method == nil || method.DeclarationNode == nil || method.DeclarationNode.Type() != "method_declaration" || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper || len(method.TypeParameters) != 0 {
		return nil
	}
	declared := false
	for _, candidate := range owner.Methods {
		if candidate == method {
			declared = true
			break
		}
	}
	if !declared {
		return nil
	}
	signature := sourceInterfaceVirtualSignature(owner, method, ctx)
	if signature == nil {
		return nil
	}
	identity := javaClassBinaryName(owner) + "\x00" + method.OriginalName
	for index := range method.Parameters {
		identity += "\x00" + qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner)
	}
	identity += "\x01" + qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
	base := "Java2goInterfaceVirtual_" + fmt.Sprintf("%X", []byte(identity))
	name := base
	for suffix := 0; ; suffix++ {
		name = base
		if suffix > 0 {
			name += "_" + strconv.Itoa(suffix)
		}
		// Scan the source graph as well as Definitions: locals and anonymous
		// declarations have not necessarily been hoisted at this emission point.
		if !generatedIdentifierExists(name, nil) && !sourceMethodIdentifierExists(name, ctx) && !superclassEmbeddingSourceIdentifierExists(name, ctx) {
			break
		}
	}
	return &sourceInterfaceVirtualEntry{owner: owner, method: method, name: name, signature: signature}
}

// Written types are qualified in the declaration's scope, then lowered in the
// consuming file so imports and aliases belong to that file. No new erasure is
// admitted here; generic owner/method contracts are deliberately excluded.
func sourceInterfaceVirtualSignature(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *ast.FuncType {
	params := &ast.FieldList{}
	for index, parameter := range method.Parameters {
		if parameter == nil {
			return nil
		}
		javaType := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner)
		parameterType := abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)
		parameterType = rawUnboundReceiverParameterType(owner, method, index, javaType, parameterType, ctx)
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: parameterType})
	}
	var results *ast.FieldList
	if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
		javaType := qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
		results = &ast.FieldList{List: []*ast.Field{{Type: abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)}}}
	}
	public := &ast.Field{Type: &ast.FuncType{Params: params, Results: results}}
	return executionMethodField(public, method, owner, ctx).Type.(*ast.FuncType)
}

func sourceInterfaceVirtualOptionalType(plan *sourceInterfaceVirtualEntry) ast.Expr {
	if plan == nil {
		return nil
	}
	return &ast.InterfaceType{Methods: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(plan.name)}, Type: plan.signature}}}}
}

func sourceInterfaceVirtualCallBranch(plan *sourceInterfaceVirtualEntry, receiver ast.Expr, args []ast.Expr, execution ast.Expr, results *ast.FieldList, used map[string]struct{}, convert func(ast.Expr) ast.Expr) []ast.Stmt {
	if plan == nil {
		return nil
	}
	dynamic := synchronizedUniqueLocalName("__java2goVirtualInterfaceReceiver", used)
	present := synchronizedUniqueLocalName("__java2goHasVirtualInterfaceReceiver", used)
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(dynamic), Sel: ast.NewIdent(plan.name)}, Args: append([]ast.Expr{execution}, args...)}
	markVariadicForwardCall(call, plan.method)
	var result ast.Expr = call
	if convert != nil {
		result = convert(result)
	}
	body := []ast.Stmt{invocationClosureCallStatement(result, results)}
	if results == nil || len(results.List) == 0 {
		body = append(body, &ast.ReturnStmt{})
	}
	return []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent(dynamic), ast.NewIdent(present)}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.TypeAssertExpr{X: &ast.CallExpr{Fun: &ast.InterfaceType{Methods: &ast.FieldList{}}, Args: []ast.Expr{receiver}}, Type: sourceInterfaceVirtualOptionalType(plan)}}}, &ast.IfStmt{Cond: ast.NewIdent(present), Body: &ast.BlockStmt{List: body}}}
}

func sourceInterfaceVirtualImplementation(scope *symbol.ClassScope, plan *sourceInterfaceVirtualEntry, ctx Ctx) *methodResolution {
	seen := map[*symbol.ClassScope]bool{}
	for owner := scope; owner != nil && !seen[owner]; owner = resolveSuperclassScopeInDeclaringContext(ctx, owner) {
		seen[owner] = true
		if owner.IsInterface || len(owner.TypeParameters) != 0 || owner.Class == nil || !classNeedsVirtualDispatch(owner, ctx) {
			continue
		}
		parameters, result := mappedOverrideSourceSignature(plan.owner, plan.method, nil)
		for _, method := range directDeclaredOverrides(plan.owner, owner, plan.method, parameters, result, ctx) {
			// A copied interface default is not a declared class body.
			body := owner.Class.DeclarationNode
			if body == nil || method.DeclarationNode.Parent() == nil || !method.DeclarationNode.Parent().Equal(body.ChildByFieldName("body")) {
				continue
			}
			return &methodResolution{owner: owner, def: method, receiverScope: scope}
		}
	}
	return nil
}

func generateSourceInterfaceVirtualEntries(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || len(scope.TypeParameters) != 0 {
		return nil
	}
	// Planning maps physical types before applicability is known. Keep those
	// imports separate from both the caller and the eventual emitted entry.
	importContext := func(input Ctx) Ctx {
		result := input.Clone()
		if input.importAliases != nil {
			result.importAliases = make(map[string]string, len(input.importAliases))
			for path, alias := range input.importAliases {
				result.importAliases[path] = alias
			}
		}
		if input.usedImports != nil {
			result.usedImports = make(map[string]bool, len(input.usedImports))
			for path, used := range input.usedImports {
				result.usedImports[path] = used
			}
		}
		return result
	}
	interfaces := sourceInterfaceAncestors(scope, ctx)
	var declarations []ast.Decl
	emitted := map[string]bool{}
	for _, iface := range interfaces {
		for _, method := range iface.Methods {
			proofCtx := importContext(ctx)
			plan := planSourceInterfaceVirtualEntry(iface, method, proofCtx)
			if plan == nil || emitted[plan.name] {
				continue
			}
			resolution := sourceInterfaceVirtualImplementation(scope, plan, proofCtx)
			if resolution == nil {
				continue
			}
			implementation := sourceInterfaceVirtualSignature(resolution.owner, resolution.def, proofCtx)
			if implementation == nil || len(implementation.Params.List) != len(plan.signature.Params.List) {
				continue
			}
			matching := true
			for index, parameter := range implementation.Params.List {
				if !reflect.DeepEqual(parameter.Type, plan.signature.Params.List[index].Type) {
					matching = false
					break
				}
			}
			if !matching {
				continue
			}
			sameResult := reflect.DeepEqual(implementation.Results, plan.signature.Results)
			actual := qualifyJavaTypeInDeclaringContext(resolution.def.OriginalType, resolution.owner)
			expected := qualifyJavaTypeInDeclaringContext(plan.method.OriginalType, plan.owner)
			if !sameResult && (!overrideBridgeResultCompatible(actual, resolution.owner, expected, plan.owner, proofCtx) || (!overrideBridgePlainResultWideningSupported(expected, plan.owner, proofCtx) && !overrideBridgeConcreteResultWideningSupported(actual, resolution.owner, expected, plan.owner, proofCtx))) {
				continue
			}
			// Rebuild the accepted plan rather than retaining any AST or aliases
			// from proof-only implementation signatures and compatibility checks.
			emissionCtx := importContext(ctx)
			plan = planSourceInterfaceVirtualEntry(iface, method, emissionCtx)
			if plan == nil {
				continue
			}
			signature := &ast.FuncType{Params: cloneFieldList(plan.signature.Params), Results: cloneFieldList(plan.signature.Results)}
			used := map[string]struct{}{}
			for _, name := range scope.GoTypeParameterNames() {
				used[name] = struct{}{}
			}
			for _, field := range signature.Params.List {
				for _, name := range field.Names {
					used[name.Name] = struct{}{}
				}
			}
			receiverName := synchronizedUniqueLocalName("__java2goInterfaceParent", used)
			receiver := ast.NewIdent(receiverName)
			storage := sourceClassViewExpr(scope, resolution.owner, receiver, emissionCtx)
			if storage == nil {
				continue
			}
			executionName := signature.Params.List[0].Names[0].Name
			callCtx := emissionCtx.Clone()
			callCtx.executionContextName = executionName
			arguments := fieldListNames(&ast.FieldList{List: signature.Params.List[1:]})
			project := func(call ast.Expr) ast.Expr {
				if !sameResult && !overrideBridgePlainResultWideningSupported(expected, plan.owner, emissionCtx) {
					return overrideBridgeConcreteResultProjection(call, actual, resolution.owner, expected, plan.owner, emissionCtx)
				}
				return call
			}
			virtual := virtualDispatchMethodCall(storage, resolution, arguments, false, callCtx)
			if virtual == nil {
				continue
			}
			direct := &ast.CallExpr{Fun: &ast.SelectorExpr{X: storage, Sel: ast.NewIdent(executionImplementationName(resolution.def, resolution.owner, callCtx))}, Args: append([]ast.Expr{ast.NewIdent(executionName)}, arguments...)}
			markVariadicForwardCall(direct, resolution.def)
			body := []ast.Stmt{instanceMethodNilReceiverGuard(receiverName)}
			fallback := []ast.Stmt{invocationClosureCallStatement(project(direct), signature.Results)}
			if signature.Results == nil || len(signature.Results.List) == 0 {
				fallback = append(fallback, &ast.ReturnStmt{})
			}
			body = append(body, &ast.IfStmt{Cond: &ast.BinaryExpr{X: &ast.SelectorExpr{X: storage, Sel: ast.NewIdent(classDispatchFieldName(resolution.owner))}, Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: fallback}}, invocationClosureCallStatement(project(virtual), signature.Results))
			declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(plan.name), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: ast.NewIdent(scope.Class.Name)}}}}, Type: signature, Body: &ast.BlockStmt{List: body}})
			for path, alias := range emissionCtx.importAliases {
				if ctx.importAliases != nil {
					ctx.importAliases[path] = alias
				}
			}
			for path, used := range emissionCtx.usedImports {
				if ctx.usedImports != nil {
					ctx.usedImports[path] = used
				}
			}
			emitted[plan.name] = true
		}
	}
	return declarations
}

func sourceInterfaceAncestors(scope *symbol.ClassScope, ctx Ctx) []*symbol.ClassScope {
	queue := []*symbol.ClassScope{scope}
	seen := map[*symbol.ClassScope]bool{}
	var result []*symbol.ClassScope
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] {
			continue
		}
		seen[current] = true
		if current.IsInterface {
			result = append(result, current)
		}
		queue = append(queue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		if !current.IsInterface {
			queue = append(queue, resolveSuperclassScopeInDeclaringContext(ctx, current))
		}
	}
	return result
}

// The statically retained parent can have ambiguous promoted public methods
// (an inherited class body and an interface default carrier). The fallback is
// the original Java public interface contract, never that parent methodset.
func sourceInterfaceVirtualPublicReceiver(plan *sourceInterfaceVirtualEntry, receiver ast.Expr, ctx Ctx) ast.Expr {
	if plan == nil {
		return receiver
	}
	return &ast.TypeAssertExpr{X: &ast.CallExpr{Fun: &ast.InterfaceType{Methods: &ast.FieldList{}}, Args: []ast.Expr{stdjavaCall(ctx, "ReferenceRequireNonNull", receiver)}}, Type: qualifiedNameExpr(plan.owner.Class.Name, findJavaPackageForClassScope(plan.owner), ctx)}
}
