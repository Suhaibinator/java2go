package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Dedicated canonical keys isolate interface dispatch from legacy native selectors.
func registerCanonicalMapPutIntrinsic() {
	for _, owner := range []string{"java.util.Map", "java.util.HashMap", "java.util.LinkedHashMap", "java.util.TreeMap", "java.util.concurrent.ConcurrentMap", "java.util.concurrent.ConcurrentHashMap"} {
		registerIntrinsicOwner(owner, true)
	}
	intrinsicOwnerKeys["java.util.Map"] = "JavaMap"
	intrinsicOwnerKeys["java.util.concurrent.ConcurrentMap"] = "JavaConcurrentMap"
	intrinsicOwnerKeys["java.util.concurrent.ConcurrentHashMap"] = "JavaConcurrentHashMap"
	for _, owner := range []string{"JavaMap", "JavaConcurrentMap", "JavaConcurrentHashMap"} {
		for _, op := range []struct {
			java, runtime string
			arity         int
			raw           bool
		}{
			{"get", "MapGetExecution", 1, true}, {"put", "MapPutExecution", 2, true}, {"remove", "MapRemoveExecution", 1, true},
			{"getOrDefault", "MapGetOrDefaultExecution", 2, true}, {"putIfAbsent", "MapPutIfAbsentExecution", 2, true}, {"putAll", "MapPutAllExecution", 1, false},
			{"size", "MapSizeExecution", 0, false}, {"isEmpty", "MapIsEmptyExecution", 0, false}, {"containsKey", "MapContainsKeyExecution", 1, false}, {"containsValue", "MapContainsValueExecution", 1, false},
			{"clear", "MapClearExecution", 0, false}, {"entrySet", "MapEntrySetExecution", 0, false}, {"keySet", "MapKeySetExecution", 0, false}, {"values", "MapValuesExecution", 0, false},
			{"equals", "ObjectEqualsExecution", 1, false}, {"hashCode", "ObjectHashCodeExecution", 0, false},
		} {
			registerInstanceNodeIntrinsic(owner, op.java, func(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if invocationArgumentCount(invocation) != op.arity {
					return nil
				}
				args := intrinsicArgs(invocation.ChildByFieldName("object"), op.java, source, ctx)
				call := stdjavaCall(ctx, op.runtime, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
				if !op.raw {
					return call
				}
				logical, known := inferExprJavaType(invocation, ctx, source)
				if !known {
					logical = "java.lang.Object"
				}
				return mapErasedResultProjection(call, invocation, logical, ctx, source)
			})
		}
		registerInstanceNodeIntrinsic(owner, "computeIfAbsent", func(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(invocation) != 2 {
				return nil
			}
			if owner == "JavaConcurrentHashMap" {
				return unsupportedIntrinsicOwnerValue("ConcurrentHashMap atomic computeIfAbsent", invocation, source, ctx)
			}
			args := parseTypedIntrinsicInvocationArguments(invocation, invocation.ChildByFieldName("object"), "", "computeIfAbsent", source, ctx, true)
			raw := mapComputeProtocolCall(receiver, args, invocation, ctx, source)
			logical, known := inferExprJavaType(invocation, ctx, source)
			if !known {
				logical = "java.lang.Object"
			}
			return mapErasedResultProjection(raw, invocation, logical, ctx, source)
		})
	}
}

// Preserve the original Function reference for source overrides. Only applying
// it converts the key to that Function declaration's parameter; its erased
// result reaches storage before the map consumer performs a checkcast.
func mapComputeProtocolCall(receiver ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	mappingNode := invocationArgumentNode(node, 1)
	mappingType, known := inferExprJavaType(mappingNode, ctx, source)
	expected := intrinsicInvocationExpectedArgumentTypes(node, node.ChildByFieldName("object"), "", "computeIfAbsent", ctx, source)
	if !known || mappingNode.Type() == "lambda_expression" || mappingNode.Type() == "method_reference" || isStaticallyNullReference(mappingNode) {
		mappingType = expected[1]
	}
	base, slots := parseJavaTypeString(mappingType)
	if !isExternalFunctionType(base, ctx) {
		if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
			for i := range slots {
				slots[i] = qualifyDeclaredReferenceType(symbol.JavaType{Original: slots[i]}, ctx)
			}
			bindings, valid := mapReferenceDeclarationBindings(scope, slots, ctx)
			if !valid {
				return unsupportedIntrinsicOwnerValue("Map Function declaration arguments", node, source, ctx)
			}
			slots = sourceFunctionContract(scope, ctx)
			for i := range slots {
				slots[i] = substituteJavaTypeParams(slots[i], bindings)
			}
		}
	}
	if len(slots) != 2 {
		slots = []string{"java.lang.Object", "java.lang.Object"}
	}
	mapper := ast.NewIdent("__java2goMapping")
	exec := ast.NewIdent("__java2goMapExecution")
	key := ast.NewIdent("__java2goMapKey")
	m := ast.NewIdent("__java2goMap")
	callback := mapMappingApplyExpression(mapper, "java.util.function.Function<"+slots[0]+","+slots[1]+">", ctx)
	call := stdjavaCall(ctx, "MapComputeIfAbsentExecution", exec, m, key, mapper, callback)
	signature := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(exec.Name, ctx), {Names: []*ast.Ident{m}, Type: stdjavaQualifiedExpr("JavaMap", ctx)}, {Names: []*ast.Ident{key}, Type: ast.NewIdent("any")}, {Names: []*ast.Ident{mapper}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: signature, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}}, Args: []ast.Expr{intrinsicExecutionExpr(ctx), receiver, args[0], args[1]}}
}

// A source declaration has exactly the Map generic contract it inherits.
func mapProtocolSourceContracts(scope *symbol.ClassScope, ctx Ctx) []abstractMapDefault {
	slots, found := sourceMapReferenceArguments(scope, "java.util.Map", ctx)
	if !found {
		return nil
	}
	if len(slots) != 2 {
		slots = []string{"java.lang.Object", "java.lang.Object"}
	}
	contracts := append([]abstractMapDefault{{java: "entrySet", result: "java.util.Set", role: "MapEntrySetJava2goExecution"}}, abstractMapDefaults...)
	contracts = append(contracts, abstractMapDefault{java: "putAll", result: "void", role: "MapPutAllJava2goExecution", parameters: []string{"java.util.Map<? extends K,? extends V>"}}, abstractMapDefault{java: "putIfAbsent", result: "V", algorithm: "MapDefaultPutIfAbsentExecution", dispatch: "MapPutIfAbsentExecution", role: "MapPutIfAbsentJava2goExecution", parameters: []string{"K", "V"}, rawResult: true}, abstractMapDefault{java: "getOrDefault", result: "V", algorithm: "MapDefaultGetOrDefaultExecution", dispatch: "MapGetOrDefaultExecution", role: "MapGetOrDefaultJava2goExecution", parameters: []string{"java.lang.Object", "V"}, rawResult: true}, abstractMapDefault{java: "computeIfAbsent", result: "V", algorithm: "MapDefaultComputeIfAbsentExecution", dispatch: "MapComputeIfAbsentExecution", role: "MapComputeIfAbsentJava2goExecution", parameters: []string{"K", "java.util.function.Function<? super K,? extends V>"}, rawResult: true})
	bindings := map[string]string{"K": slots[0], "V": slots[1]}
	for i := range contracts {
		contracts[i].result = substituteJavaTypeParams(contracts[i].result, bindings)
		contracts[i].parameters = append([]string(nil), contracts[i].parameters...)
		for j := range contracts[i].parameters {
			contracts[i].parameters[j] = substituteJavaTypeParams(contracts[i].parameters[j], bindings)
		}
	}
	return contracts
}

func mapMappingApplyType(ctx Ctx) *ast.FuncType {
	return &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)}}, {Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}
}
func mapMappingApplyExpression(mapping ast.Expr, mappingType string, ctx Ctx) ast.Expr {
	_, slots := parseJavaTypeString(mappingType)
	if len(slots) != 2 {
		slots = []string{"java.lang.Object", "java.lang.Object"}
	}
	keyType := readableWildcardProjection(slots[0])
	resultType := readableWildcardProjection(slots[1])
	e := ast.NewIdent("__java2goCallbackExecution")
	k := ast.NewIdent("__java2goCallbackKey")
	functionType := "java.util.function.Function<" + keyType + "," + resultType + ">"
	function := mapping
	if projected := nativeFunctionalObjectProjection(function, functionType, ctx); projected != nil {
		function = projected
	}
	applied := stdjavaGenericCall(ctx, "CallFunctionExecution", []ast.Expr{javaTypeStringToGoTypeExpr(keyType, inScopeTypeParameters(ctx), ctx), javaTypeStringToGoTypeExpr(resultType, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{e, function, abstractMapResultProjection(k, keyType, ctx)})
	return &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(e.Name, ctx), {Names: []*ast.Ident{k}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{applied}}}}}
}
