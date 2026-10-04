package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// A protocol bridge has an erased signature; its source body keeps its normal
// result and distinct execution entry, so concrete calls retain Java typing.
func iterationSourceImplementationName(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if method == nil || method.IsStatic || method.IsPrivate || len(method.Parameters) != 0 || len(method.TypeParameters) != 0 {
		return ""
	}
	target := "java.util.Iterator"
	switch method.OriginalName {
	case "iterator":
		target = "java.lang.Iterable"
	case "next", "hasNext", "remove":
	default:
		return ""
	}
	if _, ok := sourceIterationContract(owner, target, ctx); !ok {
		return ""
	}
	return collisionSafeExecutionIdentifier(method.Name+"Java2goIterationBodyExecution", owner)
}

func generateIterationBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.IsInterface || scope.Class == nil {
		return nil
	}
	type signature struct{ target, java, bridge, result string }
	contracts := []signature{
		{"java.lang.Iterable", "iterator", "IteratorJava2goExecution", "java.util.Iterator"},
		{"java.util.Iterator", "hasNext", "HasNextJava2goExecution", "boolean"},
		{"java.util.Iterator", "next", "NextJava2goExecution", "java.lang.Object"},
		{"java.util.Iterator", "remove", "IteratorRemoveJava2goExecution", "void"},
	}
	var declarations []ast.Decl
	for _, contract := range contracts {
		if _, ok := sourceIterationContract(scope, contract.target, ctx); !ok {
			continue
		}
		var method *symbol.Definition
		var owner *symbol.ClassScope
		seen := map[*symbol.ClassScope]bool{}
		for current := scope; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
			seen[current] = true
			for _, candidate := range current.Methods {
				if candidate != nil && candidate.OriginalName == contract.java && !candidate.IsStatic && !candidate.IsPrivate && len(candidate.Parameters) == 0 && len(candidate.TypeParameters) == 0 {
					method, owner = candidate, current
					break
				}
			}
			if method != nil {
				break
			}
		}
		if method == nil || !method.HasBody {
			continue
		}
		receiver := ast.NewIdent("receiver")
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: []ast.Expr{ast.NewIdent("execution")}}
		var results *ast.FieldList
		body := &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: call}}}
		if contract.result != "void" {
			results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(contract.result, inScopeTypeParameters(ctx), ctx)}}}
			body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}
		}
		declarations = append(declarations, &ast.FuncDecl{
			Name: ast.NewIdent(contract.bridge),
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(scope.GoTypeParameterNames()))}}}},
			Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: results},
			Body: body,
		})
	}
	return declarations
}

func sourceIterationInterfaceIDs(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if scope == nil {
		return nil
	}
	declaring := classScopeCtx(scope, ctx)
	var result []ast.Expr
	for _, parent := range append(append([]string(nil), scope.ImplementedInterfaces...), scope.Superclass) {
		if owner := canonicalIterationOwner(parent, declaring); owner != "" {
			result = append(result, javaTypeIDLiteral(owner, ctx))
		}
	}
	return result
}

// A plain method-reference factory is evaluated once when the adapter is made,
// never once per iterator request. Execution-aware references bypass this path.
func iterationPlainFactoryCallback(factory ast.Expr, ctx Ctx) ast.Expr {
	result := &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("JavaIterator", ctx)}}}
	plainType := &ast.FuncType{Params: &ast.FieldList{}, Results: result}
	executionType := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: result}
	callback := &ast.FuncLit{
		Type: executionType,
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: ast.NewIdent("factory")}}}}},
	}
	capture := &ast.FuncLit{
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("factory")}, Type: plainType}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: executionType}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{callback}}}},
	}
	return &ast.CallExpr{Fun: capture, Args: []ast.Expr{factory}}
}

// Hoisted classes are created after ResolveFile's inherited-family pass. Adopt
// the already-resolved selector of a genuine override before reserving protocol
// names. All local and anonymous classes must preserve their inherited selector
// families, including ordinary interfaces with globally reserved spellings.
func resolveSyntheticInheritedMethodNames(scope *symbol.ClassScope, ctx Ctx) {
	var ancestors []*symbol.ClassScope
	seen := map[*symbol.ClassScope]bool{scope: true}
	queue := []*symbol.ClassScope{scope}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		parents := append([]*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(ctx, current)}, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		for _, parent := range parents {
			if parent == nil || seen[parent] {
				continue
			}
			seen[parent] = true
			ancestors = append(ancestors, parent)
			queue = append(queue, parent)
		}
	}
	matched := map[*symbol.Definition]bool{}
	for _, method := range scope.Methods {
		if method == nil || method.IsStatic || method.IsPrivate || method.Constructor {
			continue
		}
		for _, ancestor := range ancestors {
			for _, inherited := range ancestor.Methods {
				if inherited == nil || inherited.IsStatic || inherited.IsPrivate || inherited.Constructor {
					continue
				}
				if inheritedOverloadOverrides(scope, method, ancestor, inherited) {
					method.Rename(inherited.Name)
					matched[method] = true
					break
				}
			}
			if matched[method] {
				break
			}
		}
	}
	used := map[string]struct{}{}
	for _, field := range scope.Fields {
		if field != nil {
			used[field.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method != nil {
			used[method.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method == nil || method.IsStatic || method.Constructor || matched[method] || !iterationProtocolReservedSelector(method.Name) {
			continue
		}
		name := nextSyntheticMemberName(method.Name, used)
		method.Rename(name)
		used[name] = struct{}{}
	}
}
