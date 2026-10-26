package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

const intPredicateJavaType = "java.util.function.IntPredicate"

func intPredicateSourceMethodOwned(method *symbol.Definition, scope *symbol.ClassScope, contract sourceNativeFunctionalContract, ctx Ctx) bool {
	selected, owner, result := resolveSourceNativeSAM(scope, contract, ctx)
	if method == nil || method != selected || owner == nil || owner.Class == nil || !javaInferenceSameType(result, "boolean", ctx) || method.RequiresHelper || method.HelperName != "" || method.RuntimeDefault || method.IsVariadic || method.Constructor || method.IsStatic || method.IsPrivate || len(method.TypeParameters) != 0 || len(method.Parameters) != 1 || method.DeclarationNode == nil || !method.HasBody {
		return false
	}
	file := findFileScopeForClassScope(owner, ctx)
	node := method.DeclarationNode
	if file == nil || owner.Class.DeclarationNode == nil || node.Type() != "method_declaration" || node.HasError() || node.IsMissing() || node.HasChanges() || node.ChildByFieldName("body") == nil {
		return false
	}
	parent := node.Parent()
	if parent == nil || parent.Parent() == nil || !sameSourceNode(parent.Parent(), owner.Class.DeclarationNode) {
		return false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != "test" || (!owner.IsInterface && !declarationHasModifier(node, "public")) {
		return false
	}
	return javaInferenceSameType(definitionParameterJavaSignatureType(method, 0), "int", ctx)
}

func intPredicateDefaultInvocation(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if invocation == nil || invocation.Type() != "method_invocation" {
		return "", false
	}
	object, name := invocation.ChildByFieldName("object"), invocation.ChildByFieldName("name")
	if object == nil || name == nil {
		return "", false
	}
	method := name.Content(source)
	arity := invocationArgumentCount(invocation)
	if (method != "negate" || arity != 0) && ((method != "and" && method != "or") || arity != 1) {
		return "", false
	}
	typ, known := inferExprJavaType(object, ctx, source)
	if !known {
		return "", false
	}
	if nativeFunctionalFamily(typ, ctx) == "IntPredicate" {
		_, args := parseJavaTypeString(typ)
		if _, valid := nativeFunctionalContract("IntPredicate", args); !valid {
			return "", false
		}
	} else {
		base, _ := parseJavaTypeString(typ)
		scope := resolveClassScopeByQualifiedName(ctx, base)
		if scope == nil {
			return "", false
		}
		// An actual source override owns dispatch before the inherited JDK default.
		if selected := findBestMethodInHierarchy(scope, name.Content(source), invocation.ChildByFieldName("arguments"), true, false, ctx, source); selected != nil && selected.def != nil && selected.def.DeclarationNode != nil {
			return "", false
		}
		owned := false
		for _, contract := range sourceNativeFunctionalContracts(scope, ctx) {
			if contract.family != "IntPredicate" {
				continue
			}
			// An inherited default belongs to the static nominal edge. Its
			// concrete Test/default bodies are certified by the checked native view,
			// not by an abstract source receiver's method set.
			owned = true
		}
		if !owned {
			return "", false
		}
	}
	return method, true
}

// Source receivers expose inherited defaults only through a proved declared
// IntPredicate edge; source overrides keep ordinary source dispatch.
func intPredicateSourceDefaultReceiver(object *sitter.Node, method string, ctx Ctx, source []byte) bool {
	if object == nil || (method != "and" && method != "or" && method != "negate") {
		return false
	}
	invocation := object.Parent()
	if invocation == nil {
		return false
	}
	actual, known := intPredicateDefaultInvocation(invocation, ctx, source)
	return known && actual == method
}
func intPredicateExpectedArguments(invocation *sitter.Node, ctx Ctx, source []byte) []string {
	if method, known := intPredicateDefaultInvocation(invocation, ctx, source); known && (method == "and" || method == "or") {
		return []string{intPredicateJavaType}
	}
	return nil
}
func registerIntPredicateIntrinsics() {
	registerIntrinsicOwner(intPredicateJavaType, true)
	builtinFunctionalInterfaces["IntPredicate"] = builtinFunctionalInterface{parameterTypes: []string{"int"}, resultType: "boolean"}
	intrinsicFunctionalMethodNames["IntPredicate"] = "test"
	registerInstanceIntrinsicResultType("IntPredicate", "test", "boolean")
	registerInstanceIntrinsic("IntPredicate", "test", func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "CallIntPredicateExecution", intrinsicExecutionExpr(ctx), receiver, args[0])
	})
	for _, method := range []string{"and", "or", "negate"} {
		method := method
		registerInstanceIntrinsicDerivedResultType("IntPredicate", method, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			actual, known := intPredicateDefaultInvocation(node, ctx, source)
			return intPredicateJavaType, known && actual == method
		})
		registerInstanceNodeIntrinsic("IntPredicate", method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			actual, known := intPredicateDefaultInvocation(node, ctx, source)
			if !known || actual != method {
				return nil
			}
			object := node.ChildByFieldName("object")
			args := parseTypedIntrinsicInvocationArguments(node, object, "", method, source, ctx)
			count := 1
			if method == "negate" {
				count = 0
			}
			if len(args) != count {
				return nil
			}
			used := affineLoopUsedNames(node, source, ctx)
			recv := synchronizedUniqueLocalName("__java2goPredicateReceiver", used)
			receiverType, _ := inferExprJavaType(object, ctx, source)
			if nativeFunctionalFamily(receiverType, ctx) != "IntPredicate" {
				receiver = nativeFunctionalObjectProjection(receiver, intPredicateJavaType, ctx)
				if receiver == nil {
					return nil
				}
			}
			body := []ast.Stmt{intrinsicInvocationTypedLocal(recv, receiver, intPredicateJavaType, ctx)}
			values := []ast.Expr{intrinsicExecutionExpr(ctx), ast.NewIdent(recv)}
			if count == 1 {
				other := synchronizedUniqueLocalName("__java2goPredicateOther", used)
				body = append(body, intrinsicInvocationTypedLocal(other, args[0], intPredicateJavaType, ctx))
				values = append(values, ast.NewIdent(other))
			}
			runtimeMethod := map[string]string{"and": "And", "or": "Or", "negate": "Negate"}[method]
			body = append(body, &ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, "IntPredicate"+runtimeMethod+"Execution", values...)}})
			return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("IntPredicate", ctx)}}}}, Body: &ast.BlockStmt{List: body}}}
		})
	}
}

// The default-method signatures are fixed by IntPredicate. This lookup chooses
// the nearest class declaration, then a unique most-specific source interface.
// It never treats a source overload as the JDK override.
func intPredicateDefaultOverride(scope *symbol.ClassScope, name string, ctx Ctx) (*symbol.Definition, *symbol.ClassScope, bool) {
	arity := 1
	if name == "negate" {
		arity = 0
	}
	signature := func(owner *symbol.ClassScope, method *symbol.Definition) bool {
		if method == nil || method.OriginalName != name || method.IsStatic || method.IsPrivate || method.Constructor || len(method.Parameters) != arity {
			return false
		}
		if arity == 0 {
			return true
		}
		declared := classScopeCtx(owner, ctx)
		typ := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, 0), owner)
		return nativeFunctionalFamily(typ, declared) == "IntPredicate"
	}
	seen := map[*symbol.ClassScope]bool{}
	var classes []*symbol.ClassScope
	for owner := scope; owner != nil; owner = resolveSuperclassScopeInDeclaringContext(ctx, owner) {
		if seen[owner] {
			return nil, nil, false
		}
		seen[owner] = true
		classes = append(classes, owner)
		if owner.IsInterface {
			continue
		}
		var found *symbol.Definition
		for _, method := range owner.Methods {
			if signature(owner, method) {
				if found != nil {
					return nil, nil, false
				}
				found = method
			}
		}
		if found != nil {
			return found, owner, true
		}
	}
	candidates := map[*symbol.ClassScope]*symbol.Definition{}
	for _, owner := range classes {
		interfaces := transitiveImplementedInterfaceScopes(owner, ctx)
		if owner.IsInterface {
			interfaces = append(interfaces, owner)
		}
		for _, iface := range interfaces {
			for _, method := range iface.Methods {
				if signature(iface, method) {
					if old := candidates[iface]; old != nil && old != method {
						return nil, nil, false
					}
					candidates[iface] = method
				}
			}
		}
	}
	extends := func(child, parent *symbol.ClassScope) bool {
		for _, ancestor := range transitiveImplementedInterfaceScopes(child, ctx) {
			if ancestor == parent {
				return true
			}
		}
		return false
	}
	for parent := range candidates {
		for child := range candidates {
			if child != parent && extends(child, parent) {
				delete(candidates, parent)
				break
			}
		}
	}
	if len(candidates) > 1 {
		return nil, nil, false
	}
	for owner, method := range candidates {
		return method, owner, true
	}
	return nil, nil, true
}

func intPredicateSourceDefaults(scope *symbol.ClassScope, receiver ast.Expr, ctx Ctx) (ast.Expr, bool) {
	defaults := &ast.CompositeLit{Type: stdjavaQualifiedExpr("IntPredicateSourceDefaults", ctx)}
	for _, name := range []string{"and", "or", "negate"} {
		method, owner, valid := intPredicateDefaultOverride(scope, name, ctx)
		if !valid {
			return nil, false
		}
		if method == nil {
			continue
		}
		if owner == nil || owner.Class == nil || method.RequiresHelper || method.HelperName != "" || method.RuntimeDefault || method.IsVariadic || len(method.TypeParameters) != 0 || method.DeclarationNode == nil || !method.HasBody {
			return nil, false
		}
		file := findFileScopeForClassScope(owner, ctx)
		node := method.DeclarationNode
		if file == nil || node.Type() != "method_declaration" || node.HasError() || node.IsMissing() || node.HasChanges() || node.ChildByFieldName("body") == nil || (!owner.IsInterface && !declarationHasModifier(node, "public")) {
			return nil, false
		}
		parent := node.Parent()
		if parent == nil || parent.Parent() == nil || !sameSourceNode(parent.Parent(), owner.Class.DeclarationNode) {
			return nil, false
		}
		writtenName, writtenResult := node.ChildByFieldName("name"), node.ChildByFieldName("type")
		if writtenName == nil || writtenName.Content(file.Source) != method.OriginalName || writtenResult == nil || strings.TrimSpace(writtenResult.Content(file.Source)) != strings.TrimSpace(method.OriginalType) {
			return nil, false
		}
		parameters := node.ChildByFieldName("parameters")
		if parameters == nil {
			return nil, false
		}
		writtenCount := 0
		for i := 0; i < int(parameters.NamedChildCount()); i++ {
			parameter := parameters.NamedChild(i)
			if parameter.Type() == "line_comment" || parameter.Type() == "block_comment" {
				continue
			}
			if parameter.Type() != "formal_parameter" || writtenCount >= len(method.Parameters) {
				return nil, false
			}
			typ := parameter.ChildByFieldName("type")
			if typ == nil || strings.TrimSpace(typ.Content(file.Source)) != strings.TrimSpace(method.Parameters[writtenCount].OriginalType) {
				return nil, false
			}
			writtenCount++
		}
		if writtenCount != len(method.Parameters) {
			return nil, false
		}
		declared := classScopeCtx(owner, ctx)
		result := qualifyJavaTypeInDeclaringContext(method.OriginalType, owner)
		if !nativeFunctionalAssignable(result, intPredicateJavaType, declared) {
			return nil, false
		}
		target := receiver
		if !owner.IsInterface {
			target = sourceClassViewExpr(scope, owner, receiver, ctx)
			if target == nil {
				return nil, false
			}
		}
		params := &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}
		args := []ast.Expr{ast.NewIdent("execution")}
		if name != "negate" {
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent("other")}, Type: stdjavaQualifiedExpr("IntPredicate", ctx)})
			args = append(args, ast.NewIdent("other"))
		}
		call := ast.Expr(methodCall(target, executionImplementationName(method, owner, ctx), args...))
		if nativeFunctionalFamily(result, declared) != "IntPredicate" {
			call = nativeFunctionalObjectProjection(call, intPredicateJavaType, ctx)
			if call == nil {
				return nil, false
			}
		}
		closure := &ast.FuncLit{Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("IntPredicate", ctx)}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}}
		field := map[string]string{"and": "And", "or": "Or", "negate": "Negate"}[name]
		defaults.Elts = append(defaults.Elts, &ast.KeyValueExpr{Key: ast.NewIdent(field), Value: closure})
	}
	return defaults, true
}
