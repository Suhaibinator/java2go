package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

// This certificate is separate from the final, two-binder TreeMap contract.
// The source receiver keeps its invariant native parent; only the actual
// unbounded add(E) body has an erased scalar entry.
type sourceArrayListScalar struct {
	owner     *symbol.ClassScope
	file      *symbol.FileScope
	add       *symbol.Definition
	parameter *symbol.TypeParamDeclaration
	element   string
}

func sourceArrayListScalarOwner(owner *symbol.ClassScope, ctx Ctx) *sourceArrayListScalar {
	if owner == nil || owner.Class == nil || owner.IsInterface || owner.IsEnum || owner.IsAbstract || owner.IsInner || owner.Enclosing != nil || len(owner.ImplementedInterfaces) != 0 || owner.Class.DeclarationNode == nil || owner.Class.DeclarationNode.Type() != "class_declaration" {
		return nil
	}
	if strings.TrimSpace(owner.Superclass) == "" || declarationHasModifier(owner.Class.DeclarationNode, "abstract") || owner.Class.IsFinal != declarationHasModifier(owner.Class.DeclarationNode, "final") {
		return nil
	}
	registered := false
	for _, scope := range allSourceClassScopes() {
		if scope == owner {
			registered = true
			break
		}
	}
	if !registered || classHasKnownSubclass(owner, ctx) {
		return nil
	}
	declaring, owned := supplierSourceDeclarationCtx(owner, ctx)
	if !owned {
		return nil
	}
	edge := declaredJavaTypeOrigin(symbol.JavaType{Original: owner.Superclass, TypeParameterBindings: sourceNativeVirtualBindings(owner)}, declaring)
	if edge.unresolved || edge.wildcard || edge.parameter != nil || edge.nominalScope != nil || edge.externalName != "java.util.ArrayList" || edge.arrayRank != 0 || len(edge.arguments) != 1 {
		return nil
	}
	plan := &sourceArrayListScalar{owner: owner, file: declaring.currentFile}
	argument := edge.arguments[0]
	if argument.unresolved || argument.wildcard || argument.arrayRank != 0 || len(argument.arguments) != 0 || argument.nominalScope != nil {
		return nil
	}
	own := owner.OwnTypeParameters()
	if len(owner.TypeParameters) == 1 && len(own) == 1 && own[0].Declaration != nil && len(own[0].Bounds) == 0 && argument.parameter == own[0].Declaration && argument.externalName == "" {
		parameters := nodeutil.SemanticNamedChildrenOf(owner.Class.DeclarationNode.ChildByFieldName("type_parameters"))
		if len(parameters) != 1 || parameters[0].NamedChildCount() != 1 {
			return nil
		}
		plan.parameter = own[0].Declaration
		plan.element = own[0].Declaration.SourceName
	} else if len(owner.TypeParameters) == 0 && argument.parameter == nil && argument.externalName == "java.lang.String" && owner.Class.IsFinal && declarationHasModifier(owner.Class.DeclarationNode, "final") {
		plan.element = "java.lang.String"
	} else {
		return nil
	}
	for _, method := range owner.Methods {
		if method == nil {
			return nil
		}
		if method.RuntimeDefault {
			if method.DeclarationNode != nil {
				return nil
			}
			continue
		}
		// Explicit constructor and additional source-callable layouts need their
		// own ABI audit. The original roots use their implicit empty constructor.
		if method.Constructor {
			return nil
		}
		if method.OriginalName != "add" || plan.add != nil || !sourceArrayListMethodOwned(method, owner, declaring) || !method.HasBody || method.IsStatic || method.IsPrivate || method.IsVariadic || method.RequiresHelper || len(method.TypeParameters) != 0 || len(method.Parameters) != 1 || method.OriginalType != "boolean" || !declarationHasModifier(method.DeclarationNode, "public") {
			return nil
		}
		methodCtx := declaring
		methodCtx.localScope = method
		parameter := declaredJavaTypeOrigin(definitionDeclaredJavaType(method.Parameters[0]), methodCtx)
		if !sameJavaTypeOrigin(parameter, argument) {
			return nil
		}
		if plan.parameter != nil && !sourceNativeVirtualScalarDefinition(method.Parameters[0], map[*symbol.TypeParamDeclaration]bool{plan.parameter: true}) {
			return nil
		}
		plan.add = method
	}
	if plan.add == nil || !sourceArrayListScalarClassBody(plan, declaring) {
		return nil
	}
	return plan
}

// Audit the complete bounded class, including executable initializers, rather
// than just a selected signature. Counter names are immaterial. This first
// scalar slice admits primitive state updates followed by native super.add of
// the actual formal; arbitrary binder-bearing locals/callables stay refused.
func sourceArrayListScalarClassBody(plan *sourceArrayListScalar, ctx Ctx) bool {
	body := plan.owner.Class.DeclarationNode.ChildByFieldName("body")
	for _, child := range nodeutil.SemanticNamedChildrenOf(body) {
		if child.Type() == "method_declaration" {
			if !child.Equal(plan.add.DeclarationNode) {
				return false
			}
			continue
		}
		if child.Type() != "field_declaration" {
			return false
		}
		typ := child.ChildByFieldName("type")
		if typ == nil {
			return false
		}
		if _, primitive := javaPrimitiveType(typ.Content(ctx.currentFile.Source)); !primitive {
			return false
		}
		// Field initializers may hide effects or reified uses. Their independent
		// phase remains available to P1, but is outside this scalar body slice.
		for _, node := range nodeutil.SemanticNamedChildrenOf(child) {
			if node.Type() == "variable_declarator" {
				if node.ChildByFieldName("value") != nil {
					return false
				}
				for _, component := range nodeutil.SemanticNamedChildrenOf(node) {
					if component.Type() == "dimensions" {
						return false
					}
				}
			}
		}
	}
	for _, field := range plan.owner.Fields {
		if field == nil || field.DeclarationNode == nil || !threadSourceNodeWithin(field.DeclarationNode, plan.owner.Class.DeclarationNode) {
			return false
		}
		typ := field.DeclarationNode.ChildByFieldName("type")
		if typ == nil || typ.Content(ctx.currentFile.Source) != field.OriginalType || field.IsStatic != declarationHasModifier(field.DeclarationNode, "static") {
			return false
		}
		nameOwned := 0
		for _, declarator := range nodeutil.VariableDeclarators(field.DeclarationNode) {
			name := declarator.ChildByFieldName("name")
			if name != nil && name.Content(ctx.currentFile.Source) == field.OriginalName {
				nameOwned++
			}
		}
		if nameOwned != 1 {
			return false
		}
		if _, primitive := javaPrimitiveType(field.OriginalType); !primitive {
			return false
		}
	}
	statements := nodeutil.SemanticNamedChildrenOf(plan.add.DeclarationNode.ChildByFieldName("body"))
	if len(statements) == 0 {
		return false
	}
	for index, statement := range statements {
		if index == len(statements)-1 {
			if statement.Type() != "return_statement" || statement.NamedChildCount() != 1 {
				return false
			}
			call := statement.NamedChild(0)
			if call.Type() != "method_invocation" {
				return false
			}
			object, name, args := call.ChildByFieldName("object"), call.ChildByFieldName("name"), call.ChildByFieldName("arguments")
			if object == nil || object.Type() != "super" || name == nil || name.Content(ctx.currentFile.Source) != "add" || args == nil {
				return false
			}
			actual := nodeutil.SemanticNamedChildrenOf(args)
			if len(actual) != 1 || actual[0].Type() != "identifier" || actual[0].Content(ctx.currentFile.Source) != plan.add.Parameters[0].OriginalName {
				return false
			}
			continue
		}
		if statement.Type() != "expression_statement" || statement.NamedChildCount() != 1 {
			return false
		}
		update := statement.NamedChild(0)
		if update.Type() != "update_expression" || update.NamedChildCount() != 1 {
			return false
		}
		field := update.NamedChild(0)
		if field.Type() == "field_access" {
			object := field.ChildByFieldName("object")
			if object == nil || object.Type() != "this" {
				return false
			}
			field = field.ChildByFieldName("field")
		}
		if field == nil || field.Type() != "identifier" {
			return false
		}
		name := field.Content(ctx.currentFile.Source)
		if name == plan.add.Parameters[0].OriginalName {
			return false
		}
		found := 0
		for _, definition := range plan.owner.Fields {
			if definition.OriginalName == name && !definition.IsStatic && definition.OriginalType != "boolean" {
				found++
			}
		}
		if found != 1 {
			return false
		}
	}
	return true
}

func sourceArrayListScalarEdge(javaType string, ctx Ctx) bool {
	plan := sourceArrayListScalarOwner(ctx.currentClass, ctx)
	if plan == nil {
		return false
	}
	declaring, _ := supplierSourceDeclarationCtx(plan.owner, ctx)
	bindings := sourceNativeVirtualBindings(plan.owner)
	actual := declaredJavaTypeOrigin(symbol.JavaType{Original: javaType, TypeParameterBindings: bindings}, declaring)
	expected := declaredJavaTypeOrigin(symbol.JavaType{Original: plan.owner.Superclass, TypeParameterBindings: bindings}, declaring)
	return sameJavaTypeOrigin(actual, expected)
}

func sourceArrayListScalarBody(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceNativeVirtualBodyABI {
	if method == nil || method.RuntimeDefault || method.OriginalName != "add" {
		return nil
	}
	plan := sourceArrayListScalarOwner(owner, ctx)
	if plan == nil || plan.parameter == nil || method != plan.add {
		return nil
	}
	return &sourceNativeVirtualBodyABI{owner: owner, method: method, file: plan.file, declarations: map[*symbol.TypeParamDeclaration]bool{plan.parameter: true}, erasedParameters: []string{"java.lang.Object"}, erasedResult: "boolean"}
}

func sourceArrayListStorageTypeArgument(scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	plan := sourceArrayListScalarOwner(scope, ctx)
	if plan == nil {
		return ast.NewIdent("any")
	}
	declaring, _ := supplierSourceDeclarationCtx(scope, ctx)
	declaring.sourceNativeVirtualBody = nil
	return javaTypeStringToGoTypeExpr(plan.element, inScopeTypeParameters(declaring), declaring)
}

func sourceArrayListScalarAddArgument(scope *symbol.ClassScope, value ast.Expr, ctx Ctx) ast.Expr {
	plan := sourceArrayListScalarOwner(scope, ctx)
	if plan == nil || plan.parameter != nil {
		return value
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{sourceArrayListStorageTypeArgument(scope, ctx)}, []ast.Expr{value, stdjavaQualifiedExpr("StringTypeID", ctx)})
}

func sourceArrayListScalarDefaultGet(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *sourceArrayListScalar {
	if method == nil || !method.RuntimeDefault || method.DeclarationNode != nil || !method.HasBody || method.IsStatic || method.IsPrivate || method.IsFinal || method.Constructor || method.IsVariadic || method.RequiresHelper || len(method.TypeParameters) != 0 || len(method.Children) != 0 || method.OriginalName != "get" || method.Name != "Get" || len(method.Parameters) != 1 {
		return nil
	}
	formal := method.Parameters[0]
	if formal == nil || formal.OriginalType != "int" || formal.DeclarationNode != nil || formal.DirectTypeParameter != nil || len(formal.TypeParameterBindings) != 0 || formal.RuntimeDefault || formal.HasBody || formal.IsStatic || formal.IsPrivate || formal.IsFinal || len(formal.Children) != 0 || formal.Constructor || formal.IsVariadic || formal.RequiresHelper || len(formal.TypeParameters) != 0 || len(formal.Parameters) != 0 || !token.IsIdentifier(formal.Name) || formal.Name != formal.OriginalName {
		return nil
	}
	plan := sourceArrayListScalarOwner(owner, ctx)
	if plan == nil || method.OriginalType != plan.element {
		return nil
	}
	attached := 0
	for _, actual := range owner.Methods {
		if actual == method {
			attached++
		}
		if actual != method && actual.OriginalName == "get" {
			return nil
		}
	}
	if attached != 1 {
		return nil
	}
	return plan
}

func sourceArrayListScalarGetErasure(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (string, bool) {
	if sourceArrayListScalarDefaultGet(owner, method, ctx) != nil {
		return "java.lang.Object", true
	}
	return "", false
}

func sourceArrayListScalarGetExecution(definition *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if definition == nil || !definition.RuntimeDefault || definition.OriginalName != "get" {
		return false
	}
	if owner != nil {
		return sourceArrayListScalarDefaultGet(owner, definition, ctx) != nil
	}
	for _, scope := range allSourceClassScopes() {
		if sourceArrayListScalarDefaultGet(scope, definition, ctx) != nil {
			return true
		}
	}
	return false
}

func sourceArrayListScalarResultView(target *invocationTargetInfo, resolution *methodResolution, ctx Ctx) (string, string, bool) {
	if target == nil || resolution == nil || target.classScope != resolution.owner {
		return "", "", false
	}
	plan := sourceArrayListScalarDefaultGet(resolution.owner, resolution.def, ctx)
	if plan == nil {
		return "", "", false
	}
	if plan.parameter == nil {
		return "java.lang.String", "java.lang.Object", true
	}
	var origin inferredJavaTypeOrigin
	if target.rawGenericView {
		origin = inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}
	} else {
		if len(target.classTypeArgumentOrigins) != 1 {
			return "", "", false
		}
		origin = target.classTypeArgumentOrigins[0]
	}
	// Use declaration-owned receiver origins. Re-reading an inherited field's
	// written Item in the caller would capture a different Child.Item.
	if origin.unresolved || origin.wildcard || origin.arrayRank != 0 || len(origin.arguments) != 0 {
		return "", "", false
	}
	view := ""
	if origin.nominalScope != nil {
		view = javaInferenceTypeName(origin.nominalScope)
	} else if origin.parameter != nil {
		view = origin.parameter.GoName
		if view == "" {
			view = origin.parameter.SourceName
		}
		if visibleTypeParameterDeclarationForJavaType(view, ctx) != origin.parameter {
			return "", "", false
		}
	} else {
		view = origin.externalName
	}
	if view == "" {
		return "", "", false
	}
	return view, "java.lang.Object", true
}

// The public Go facade retains its declared element result; Java calls select
// the raw hidden entry and check only at a concrete consumer boundary.
func sourceArrayListScalarGetDecls(method *symbol.Definition, ctx Ctx) []ast.Decl {
	plan := sourceArrayListScalarDefaultGet(ctx.currentClass, method, ctx)
	if plan == nil {
		return nil
	}
	parameter := method.Parameters[0]
	params := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: ast.NewIdent("int32")}}}
	receiver := abstractCollectionReceiverName(ctx, params)
	execution := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
	results := &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}
	raw := stdjavaCall(ctx, "ArrayListBaseGetExecution", ast.NewIdent(execution), sourceArrayListNativeViewExpr(ctx.currentClass, ast.NewIdent(receiver), ctx), ast.NewIdent(parameter.Name))
	declaration := &ast.FuncDecl{Name: ast.NewIdent(method.Name), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), &ast.ReturnStmt{Results: []ast.Expr{raw}}}}}
	declarations := buildExecutionAwareFuncDecls(declaration, executionImplementationName(method, ctx.currentClass, ctx), execution, ctx)
	if len(declarations) < 2 {
		return declarations
	}
	facade := declarations[0].(*ast.FuncDecl)
	typed := sourceArrayListStorageTypeArgument(ctx.currentClass, ctx)
	facade.Type.Results = &ast.FieldList{List: []*ast.Field{{Type: typed}}}
	call := methodCall(ast.NewIdent(receiver), executionImplementationName(method, ctx.currentClass, ctx), newExecutionExpr(ctx), ast.NewIdent(parameter.Name))
	descriptor := stdjavaQualifiedExpr("ObjectTypeID", ctx)
	if plan.parameter == nil {
		descriptor = stdjavaQualifiedExpr("StringTypeID", ctx)
	}
	facade.Body = &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{typed}, []ast.Expr{call, descriptor})}}}}
	return declarations
}
