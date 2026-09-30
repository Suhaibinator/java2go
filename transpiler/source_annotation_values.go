package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func sourceAnnotationProxyName(scope *symbol.ClassScope) string {
	return collisionSafeExecutionIdentifier("__java2goAnnotationValue"+symbol.GoIdentifier(scope.Class.Name), scope)
}
func sourceAnnotationFactoryName(scope *symbol.ClassScope) string {
	return symbol.GoIdentifier(collisionSafeExecutionIdentifier("Java2goAnnotationValueFactory"+scope.Class.Name, scope))
}
func sourceAnnotationMemberType(method *symbol.Definition, ctx Ctx) ast.Expr {
	return javaTypeStringToGoTypeExpr(method.OriginalType, nil, ctx)
}

// Annotation elements are ordinary implicitly public abstract source methods.
// Their concrete value proxies expose both the source and execution ABIs.
func sourceAnnotationDecls(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	declaring := classScopeCtx(scope, ctx)
	proxy := sourceAnnotationProxyName(scope)
	members := &ast.FieldList{}
	iface := &ast.FieldList{}
	companion := &ast.FieldList{}
	var methods []ast.Decl
	var constructorValues []ast.Expr
	params := []*ast.Field{executionParameterField("execution", ctx)}
	for index, method := range scope.Methods {
		if method.IsStatic || method.Constructor {
			continue
		}
		result := sourceAnnotationMemberType(method, declaring)
		plainType := &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: result}}}}
		execType := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: plainType.Results}
		execName := executionImplementationName(method, scope, declaring)
		iface.List = append(iface.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(symbol.GoIdentifier(method.Name))}, Type: plainType})
		companion.List = append(companion.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(symbol.GoIdentifier(execName))}, Type: execType})
		var value ast.Expr
		body := []ast.Stmt{}
		if method.OriginalName == "annotationType" && method.DeclarationNode == nil {
			value = stdjavaCall(ctx, "ClassLiteral", javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx))
		} else {
			fieldName := "member" + strconv.Itoa(index)
			members.List = append(members.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(fieldName)}, Type: result})
			argName := "value" + strconv.Itoa(index)
			params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent(argName)}, Type: result})
			constructorValues = append(constructorValues, &ast.KeyValueExpr{Key: ast.NewIdent(fieldName), Value: ast.NewIdent(argName)})
			value = &ast.SelectorExpr{X: ast.NewIdent("annotation"), Sel: ast.NewIdent(fieldName)}
			if strings.HasSuffix(strings.TrimSpace(method.OriginalType), "[]") {
				component := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(method.OriginalType), "[]"))
				if _, primitive := javaPrimitiveTypeIDExpr(component, declaring); primitive {
					clone := stdjavaGenericCall(ctx, "PrimitiveArrayLiteral", []ast.Expr{javaTypeStringToGoTypeExpr(component, nil, declaring)}, []ast.Expr{selectorCall(value, "ComponentType", nil), stdjavaCall(ctx, "PrimitiveArrayIterationElements", value)})
					clone.Ellipsis = 1
					body = append(body, &ast.IfStmt{Cond: &ast.BinaryExpr{X: stdjavaCall(ctx, "PrimitiveArrayLength", value), Op: token.GTR, Y: reflectionInteger(0)}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{clone}}}}})
				} else {
					clone := stdjavaCall(ctx, "ReferenceArrayLiteral", selectorCall(value, "ComponentType", nil), stdjavaCall(ctx, "ReferenceArrayIterationElements", value))
					clone.Ellipsis = 1
					body = append(body, &ast.IfStmt{Cond: &ast.BinaryExpr{X: stdjavaCall(ctx, "ReferenceArrayLength", value), Op: token.GTR, Y: reflectionInteger(0)}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{clone}}}}})
				}
			}
		}
		body = append(body, &ast.ReturnStmt{Results: []ast.Expr{value}})
		recv := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("annotation")}, Type: &ast.StarExpr{X: ast.NewIdent(proxy)}}}}
		methods = append(methods, &ast.FuncDecl{Name: ast.NewIdent(symbol.GoIdentifier(execName)), Recv: recv, Type: execType, Body: &ast.BlockStmt{List: body}})
		facadeCall := selectorCall(ast.NewIdent("annotation"), symbol.GoIdentifier(execName), []ast.Expr{newExecutionExpr(ctx)})
		methods = append(methods, &ast.FuncDecl{Name: ast.NewIdent(symbol.GoIdentifier(method.Name)), Recv: recv, Type: plainType, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{facadeCall}}}}})
	}
	out := []ast.Decl{genInterfaceInContext(scope.Class.Name, iface, nil, ctx)}
	if len(companion.List) > 0 {
		out = append(out, genInterfaceInContext(executionCompanionInterfaceName(scope), companion, nil, ctx))
	}
	out = append(out, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{&ast.TypeSpec{Name: ast.NewIdent(proxy), Type: &ast.StructType{Fields: members}}}})
	out = append(out, methods...)
	identity := sourceClassRuntimeTypeID(scope, ctx) + "$Java2goAnnotationValue"
	out = append(out, fixedJavaDynamicTypeDecl(proxy, identity, ctx))
	out = append(out, syntheticReferenceRegistrationDecl(proxy, identity, stdjavaQualifiedExpr("ObjectTypeID", ctx), []ast.Expr{javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx), stdjavaQualifiedExpr("AnnotationTypeID", ctx)}, ctx))
	out = append(out, &ast.FuncDecl{Name: ast.NewIdent(sourceAnnotationFactoryName(scope)), Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(symbol.GoIdentifier(scope.Class.Name))}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: ast.NewIdent(proxy), Elts: constructorValues}}}}}}})
	if registration := sourceClassRegistrationDecl(scope, ctx); registration != nil {
		out = append(out, registration)
	}
	return out
}

func sourceAnnotationDefaultNode(method *symbol.Definition) *sitter.Node {
	if method.DeclarationNode == nil {
		return nil
	}
	if value := method.DeclarationNode.ChildByFieldName("value"); value != nil {
		return value
	}
	name := method.DeclarationNode.ChildByFieldName("name")
	if name == nil {
		return nil
	}
	for _, child := range nodeutil.NamedChildrenOf(method.DeclarationNode) {
		if child.StartByte() > name.EndByte() && child.Type() != "dimensions" {
			return child
		}
	}
	return nil
}
func sourceAnnotationMemberValue(node *sitter.Node, javaType string, source []byte, ctx Ctx) (ast.Expr, bool) {
	if node == nil {
		return nil, false
	}
	ctx.expectedType = javaType
	ctx.expectedTypeRoot = node
	if strings.HasSuffix(strings.TrimSpace(javaType), "[]") {
		component := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(javaType), "[]"))
		nodes := []*sitter.Node{node}
		if node.Type() == "element_value_array_initializer" || node.Type() == "array_initializer" {
			nodes = nodeutil.NamedChildrenOf(node)
		}
		var values []ast.Expr
		for _, element := range nodes {
			value, ok := sourceAnnotationMemberValue(element, component, source, ctx)
			if !ok {
				return nil, false
			}
			values = append(values, value)
		}
		if id, primitive := javaPrimitiveTypeIDExpr(component, ctx); primitive {
			return stdjavaGenericCall(ctx, "PrimitiveArrayLiteral", []ast.Expr{javaTypeStringToGoTypeExpr(component, nil, ctx)}, append([]ast.Expr{id}, values...)), true
		}
		id, ok := javaTypeDescriptorExpr(component, ctx)
		if !ok {
			return nil, false
		}
		return stdjavaCall(ctx, "ReferenceArrayLiteral", append([]ast.Expr{id}, values...)...), true
	}
	value := ParseExpr(node, source, ctx)
	return coerceArgumentToExpectedType(value, node, javaType, ctx, source), value != nil
}

func sourceAnnotationDescriptors(node *sitter.Node, scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	out := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("AnnotationDescriptor", ctx)}}
	if node == nil {
		return out
	}
	caller := classScopeCtx(scope, ctx)
	file := findFileScopeForClassScope(scope)
	if file == nil {
		return out
	}
	for _, modifiers := range nodeutil.NamedChildrenOf(node) {
		if modifiers.Type() != "modifiers" {
			continue
		}
		for _, annotation := range nodeutil.NamedChildrenOf(modifiers) {
			if annotation.Type() != "annotation" && annotation.Type() != "marker_annotation" {
				continue
			}
			name := annotation.ChildByFieldName("name")
			if name == nil {
				continue
			}
			declaration := resolveClassScopeByQualifiedName(caller, name.Content(file.Source))
			if declaration == nil {
				continue
			}
			retention, _ := runtimeMarkerPolicy(declaration)
			if !retention {
				continue
			}
			values := map[string]*sitter.Node{}
			if args := annotation.ChildByFieldName("arguments"); args != nil {
				for _, arg := range nodeutil.NamedChildrenOf(args) {
					if arg.Type() == "element_value_pair" {
						key := arg.ChildByFieldName("key")
						if key == nil {
							key = arg.ChildByFieldName("name")
						}
						value := arg.ChildByFieldName("value")
						if key != nil && value != nil {
							values[key.Content(file.Source)] = value
						}
					} else {
						values["value"] = arg
					}
				}
			}
			args := []ast.Expr{ast.NewIdent("execution")}
			supported := true
			for _, member := range declaration.Methods {
				if member.DeclarationNode == nil || member.DeclarationNode.Type() != "annotation_type_element_declaration" {
					continue
				}
				valueNode := values[member.OriginalName]
				valueCtx := caller
				source := file.Source
				if valueNode == nil {
					valueNode = sourceAnnotationDefaultNode(member)
					valueCtx = classScopeCtx(declaration, ctx)
					source = findFileScopeForClassScope(declaration).Source
				}
				value, ok := sourceAnnotationMemberValue(valueNode, member.OriginalType, source, valueCtx)
				if !ok {
					supported = false
					break
				}
				args = append(args, value)
			}
			if !supported {
				continue
			}
			factory := qualifiedNameExpr(sourceAnnotationFactoryName(declaration), findJavaPackageForClassScope(declaration), ctx)
			callback := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField("execution", ctx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: factory, Args: args}}}}}}
			out.Elts = append(out.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Type", javaTypeIDLiteral(sourceClassRuntimeTypeID(declaration, ctx), ctx)), metadataKey("Factory", callback)}})
		}
	}
	return out
}
