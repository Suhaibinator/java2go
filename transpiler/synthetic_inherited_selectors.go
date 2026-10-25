package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"

	"github.com/NickyBoy89/java2go/symbol"
)

// A known unnamed package differs from absent declaration ownership. In
// particular, Enclosing alone cannot certify the package of a synthesized AST.
func syntheticInheritedDeclaringPackage(scope *symbol.ClassScope, ctx Ctx) (string, bool) {
	if intrinsicLexicalDeclaringFile(scope, ctx.currentFile, ctx) {
		return ctx.currentFile.Package, true
	}
	if file := findFileScopeForClassScope(scope); file != nil {
		return file.Package, true
	}
	return "", false
}

func syntheticInheritedMethodOverrides(owner *symbol.ClassScope, method *symbol.Definition, ancestor *symbol.ClassScope, inherited *symbol.Definition, ctx Ctx) bool {
	if method == nil || inherited == nil || method.IsStatic || method.IsPrivate || method.Constructor || inherited.IsStatic || inherited.IsPrivate || inherited.Constructor || method.OriginalName != inherited.OriginalName || len(method.Parameters) != len(inherited.Parameters) {
		return false
	}
	if directOwnerOverrideBridgeVisibility(ancestor, inherited) == directOwnerOverrideBridgePackagePrivate {
		left, leftKnown := syntheticInheritedDeclaringPackage(owner, ctx)
		right, rightKnown := syntheticInheritedDeclaringPackage(ancestor, ctx)
		if !leftKnown || !rightKnown || left != right {
			return false
		}
	}
	return inheritedOverloadSignatureOverrides(owner, method, ancestor, inherited, ctx)
}

// Reserve every proved inherited role before choosing the body's selector.
// A distinct Java method whose Go spelling collides with an interface role
// keeps its own identity under a fresh selector; source calls follow its Def.
func reserveSyntheticInheritedSelectorRoles(scope *symbol.ClassScope, ancestors []*symbol.ClassScope, ctx Ctx) {
	roles := map[string]*symbol.Definition{}
	used := map[string]struct{}{}
	for _, field := range scope.Fields {
		if field != nil {
			used[field.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method == nil {
			continue
		}
		used[method.Name] = struct{}{}
		for _, ancestor := range ancestors {
			for _, inherited := range ancestor.Methods {
				if !syntheticInheritedMethodOverrides(scope, method, ancestor, inherited, ctx) {
					continue
				}
				selectors := []string{inherited.Name, executionImplementationName(inherited, ancestor, classScopeCtx(ancestor, ctx))}
				for _, selector := range selectors {
					if other := roles[selector]; other != nil && other != method {
						reportUnsupported("synthetic inherited selector collision", method.DeclarationNode, ctx.currentFile.Source, ctx)
						continue
					}
					roles[selector] = method
					used[selector] = struct{}{}
				}
			}
		}
	}
	for _, field := range scope.Fields {
		if field != nil && roles[field.Name] != nil {
			reportUnsupported("synthetic inherited field selector collision", field.DeclarationNode, ctx.currentFile.Source, ctx)
		}
	}
	for _, method := range scope.Methods {
		if method == nil {
			continue
		}
		if role := roles[method.Name]; role != nil && role != method {
			name := nextSyntheticMemberName(method.Name, used)
			method.Rename(name)
			used[name] = struct{}{}
		}
	}
}

// One Java declaration can satisfy both a public interface selector and a
// package-private superclass selector. Keep both physical entries, forwarding
// the caller's execution and receiver to the one emitted implementation.
func appendSyntheticInheritedSelectorAliases(declarations []ast.Decl, method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) []ast.Decl {
	if method == nil || method.IsStatic || method.IsPrivate || method.Constructor {
		return declarations
	}
	targetName := executionImplementationName(method, owner, ctx)
	var target *ast.FuncDecl
	emitted := map[string]bool{}
	for _, declaration := range declarations {
		if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Name != nil {
			emitted[fn.Name.Name] = true
			if fn.Name.Name == targetName {
				target = fn
			}
		}
	}
	if target == nil || target.Recv == nil || target.Type == nil || target.Type.Params == nil || len(target.Type.Params.List) == 0 {
		return declarations
	}
	for _, ancestor := range inheritedOverloadAncestorsInContext(owner, ctx) {
		for _, inherited := range ancestor.Methods {
			if !syntheticInheritedMethodOverrides(owner, method, ancestor, inherited, ctx) {
				continue
			}
			aliasName := inherited.Name
			aliasExecution := executionImplementationName(inherited, ancestor, classScopeCtx(ancestor, ctx))
			if emitted[aliasName] && emitted[aliasExecution] {
				continue
			}
			if !syntheticInheritedAliasABI(target, inherited, ancestor, ctx) {
				reportUnsupported("synthetic inherited selector ABI", method.DeclarationNode, ctx.currentFile.Source, ctx)
				continue
			}
			params := cloneFieldList(target.Type.Params)
			params.List = params.List[1:]
			if !emitted[aliasExecution] {
				call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(target.Recv.List[0].Names[0].Name), Sel: ast.NewIdent(targetName)}, Args: methodCallArgs(target.Type.Params)}
				if len(params.List) > 0 {
					if _, variadic := params.List[len(params.List)-1].Type.(*ast.Ellipsis); variadic {
						call.Ellipsis = token.Pos(1)
					}
				}
				var statement ast.Stmt = &ast.ExprStmt{X: call}
				if target.Type.Results != nil && len(target.Type.Results.List) > 0 {
					statement = &ast.ReturnStmt{Results: []ast.Expr{call}}
				}
				declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(aliasExecution), Recv: cloneFieldList(target.Recv), Type: &ast.FuncType{Params: cloneFieldList(target.Type.Params), Results: cloneFieldList(target.Type.Results)}, Body: &ast.BlockStmt{List: []ast.Stmt{statement}}})
				emitted[aliasExecution] = true
			}
			if !emitted[aliasName] {
				function := &ast.SelectorExpr{X: ast.NewIdent(target.Recv.List[0].Names[0].Name), Sel: ast.NewIdent(aliasExecution)}
				declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(aliasName), Recv: cloneFieldList(target.Recv), Type: &ast.FuncType{Params: params, Results: cloneFieldList(target.Type.Results)}, Body: executionWrapperBody(function, params, target.Type.Results, ctx)})
				emitted[aliasName] = true
			}
		}
	}
	return declarations
}

func inheritedOverloadAncestorsInContext(scope *symbol.ClassScope, ctx Ctx) []*symbol.ClassScope {
	var result []*symbol.ClassScope
	seen := map[*symbol.ClassScope]bool{scope: true}
	queue := []*symbol.ClassScope{scope}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		parents := append([]*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(ctx, current)}, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		for _, parent := range parents {
			if parent != nil && !seen[parent] {
				seen[parent] = true
				result = append(result, parent)
				queue = append(queue, parent)
			}
		}
	}
	return result
}

func syntheticInheritedAliasABI(target *ast.FuncDecl, inherited *symbol.Definition, ancestor *symbol.ClassScope, ctx Ctx) bool {
	params := target.Type.Params.List[1:]
	if len(params) != len(inherited.Parameters) {
		return false
	}
	inheritedCtx := classScopeCtx(ancestor, ctx)
	typeParams := ancestor.GoTypeParameterNames()
	for index, parameter := range inherited.Parameters {
		want := executionParameterTypeExpr(inherited, index, parameter.OriginalType, typeParams, inheritedCtx)
		if !syntheticInheritedTypeEqual(params[index].Type, want) {
			return false
		}
	}
	want := inherited.OriginalType
	if want == "" || want == "void" {
		return target.Type.Results == nil || len(target.Type.Results.List) == 0
	}
	return target.Type.Results != nil && len(target.Type.Results.List) == 1 && syntheticInheritedTypeEqual(target.Type.Results.List[0].Type, javaTypeStringToGoTypeExpr(want, typeParams, inheritedCtx))
}

func syntheticInheritedTypeEqual(left, right ast.Expr) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	var a, b bytes.Buffer
	if format.Node(&a, token.NewFileSet(), left) != nil || format.Node(&b, token.NewFileSet(), right) != nil {
		return false
	}
	return a.String() == b.String()
}
