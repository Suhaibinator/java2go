package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func init() {
	registerIntrinsicOwner("java.util.AbstractList", true)
	registerIntrinsicOwner("java.util.RandomAccess", true)
	registerIntrinsicOwner("java.util.ListIterator", true)
}
func sourceAbstractListDeclaration(scope *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "class_declaration" {
		return Ctx{}, false
	}
	registered := false
	for _, actual := range allSourceClassScopes() {
		if actual == scope {
			registered = true
			break
		}
	}
	if !registered {
		return sourceAbstractListLocalDeclaration(scope, ctx)
	}
	declaring, owned := supplierSourceDeclarationCtx(scope, ctx)
	if !owned || declaring.currentFile == nil || declaring.currentFile.SourceInputSnapshot() == nil || declaring.currentFile.SourceInputSnapshot().Validate() != nil {
		return Ctx{}, false
	}
	return declaring, true
}
func sourceAbstractListDirectRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	// A negative spelling filter avoids attesting unrelated complete headers.
	// Ambiguous syntax still enters every declaration and nominal-edge proof.
	if scope == nil || !sourceNativeSuperclassCandidate(scope.Superclass, "AbstractList") {
		return false
	}
	declaring, owned := sourceAbstractListDeclaration(scope, ctx)
	if !owned {
		return false
	}
	edge := declaredJavaTypeOrigin(symbol.JavaType{Original: scope.Superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring))}, declaring)
	return !edge.unresolved && !edge.wildcard && edge.nominalScope == nil && edge.parameter == nil && edge.arrayRank == 0 && edge.externalName == "java.util.AbstractList" && len(edge.arguments) <= 1 && sourceArrayListNativeConstructors(scope, declaring)
}
func sourceAbstractListSuperclassCandidate(scope *symbol.ClassScope, ctx Ctx, seen map[*symbol.ClassScope]bool) bool {
	if scope == nil || seen[scope] {
		return false
	}
	seen[scope] = true
	base, _ := parseJavaTypeString(scope.Superclass)
	if stripJavaQualifier(base) == "AbstractList" {
		return true
	}
	if base == "" {
		return false
	}
	return sourceAbstractListSuperclassCandidate(resolveSuperclassScopeInDeclaringContext(ctx, scope), ctx, seen)
}
func sourceAbstractListRoot(scope *symbol.ClassScope, ctx Ctx) *symbol.ClassScope {
	if !sourceAbstractListSuperclassCandidate(scope, ctx, map[*symbol.ClassScope]bool{}) {
		return nil
	}
	seen := map[*symbol.ClassScope]bool{}
	for scope != nil && !seen[scope] {
		seen[scope] = true
		declaring, owned := sourceAbstractListDeclaration(scope, ctx)
		if !owned {
			return nil
		}
		if sourceAbstractListDirectRoot(scope, declaring) {
			return scope
		}
		parent := resolveSuperclassScopeInDeclaringContext(declaring, scope)
		if parent == nil {
			return nil
		}
		edge := declaredJavaTypeOrigin(symbol.JavaType{Original: scope.Superclass}, declaring)
		if edge.unresolved || edge.wildcard || edge.parameter != nil || edge.nominalScope != parent || edge.arrayRank != 0 {
			return nil
		}
		scope = parent
	}
	return nil
}
func sourceAbstractListElement(scope *symbol.ClassScope, ctx Ctx) string {
	root := sourceAbstractListRoot(scope, ctx)
	if root == nil {
		return "java.lang.Object"
	}
	_, args := parseJavaTypeString(root.Superclass)
	if len(args) == 1 {
		return qualifyJavaTypeInDeclaringContext(args[0], root)
	}
	return "java.lang.Object"
}
func sourceAbstractListStorageName(scope *symbol.ClassScope, ctx Ctx) string {
	stem := "Java2goAbstractListProjectionFor" + fmt.Sprintf("%X", []byte(javaClassBinaryName(scope)))
	for i := 0; ; i++ {
		name := stem
		if i != 0 {
			name += "_" + strconv.Itoa(i)
		}
		if !generatedIdentifierExists(name, scope) && !superclassEmbeddingMemberExists(name, scope, ctx) && !superclassEmbeddingSourceIdentifierExists(name, ctx) {
			return name
		}
	}
}
func sourceAbstractListStorageField(scope *symbol.ClassScope, ctx Ctx) *ast.Field {
	if !sourceAbstractListDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(sourceAbstractListStorageName(scope, ctx))}, Type: &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("List", ctx), []ast.Expr{javaTypeStringToGoTypeExpr(sourceAbstractListElement(scope, ctx), inScopeTypeParameters(ctx), ctx)})}}
}
func sourceAbstractListNativeViewExpr(scope *symbol.ClassScope, receiver ast.Expr, ctx Ctx) ast.Expr {
	root := sourceAbstractListRoot(scope, ctx)
	if root == nil {
		return nil
	}
	view := sourceClassViewExpr(scope, root, receiver, ctx)
	if view == nil {
		return nil
	}
	return &ast.SelectorExpr{X: view, Sel: ast.NewIdent(sourceAbstractListStorageName(root, ctx))}
}
func sourceAbstractListSuperConstructor(args []ast.Expr, scope *symbol.ClassScope, receiver string, ctx Ctx) ast.Stmt {
	if !sourceAbstractListDirectRoot(scope, ctx) || len(args) != 0 {
		return nil
	}
	return &ast.AssignStmt{Lhs: []ast.Expr{sourceAbstractListNativeViewExpr(scope, ast.NewIdent(receiver), ctx)}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaGenericCall(ctx, "NewAbstractListProjection", []ast.Expr{javaTypeStringToGoTypeExpr(sourceAbstractListElement(scope, ctx), inScopeTypeParameters(ctx), ctx)}, nil)}}
}
func sourceAbstractListCarrierInitStmt(scope *symbol.ClassScope, receiver string, self ast.Expr, ctx Ctx) ast.Stmt {
	if !sourceAbstractListDirectRoot(scope, ctx) || self == nil {
		return nil
	}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "BindAbstractListSourceReference", sourceAbstractListNativeViewExpr(scope, ast.NewIdent(receiver), ctx), &ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent("ObjectInfo")})}
}
func sourceAbstractListStateDecl(ctx Ctx) ast.Decl {
	if !sourceAbstractListDirectRoot(ctx.currentClass, ctx) {
		return nil
	}
	receiver := abstractCollectionReceiverName(ctx, nil)
	return &ast.FuncDecl{Name: ast.NewIdent("AbstractListStateJava2go"), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("AbstractListState", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), &ast.ReturnStmt{Results: []ast.Expr{methodCall(sourceAbstractListNativeViewExpr(ctx.currentClass, ast.NewIdent(receiver), ctx), "AbstractListStateJava2go")}}}}}
}
func sourceAbstractListObjectInfoDecl(scope *symbol.ClassScope, receiver *ast.FieldList, name string, ctx Ctx) ast.Decl {
	if !sourceAbstractListDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.FuncDecl{Name: ast.NewIdent("JavaObjectInfo"), Recv: receiver, Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ObjectInfo", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent(name), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}}, &ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(name), Sel: ast.NewIdent("ObjectInfo")}}}}}}
}
func sourceAbstractListDefaults(scope *symbol.ClassScope, ctx Ctx) []abstractCollectionDefault {
	if !sourceAbstractListDirectRoot(scope, ctx) {
		return nil
	}
	element := sourceAbstractListElement(scope, ctx)
	return []abstractCollectionDefault{
		{"add", "boolean", "AbstractListAddExecution", []string{element}},
		{"set", element, "AbstractListSetExecution", []string{"int", element}},
		{"add", "void", "AbstractListAddAtExecution", []string{"int", element}},
		{"remove", element, "AbstractListRemoveAtExecution", []string{"int"}},
		{"isEmpty", "boolean", "AbstractCollectionIsEmptyExecution", nil},
		{"contains", "boolean", "AbstractCollectionContainsExecution", []string{"java.lang.Object"}},
		{"remove", "boolean", "AbstractCollectionRemoveExecution", []string{"java.lang.Object"}},
		{"clear", "void", "AbstractListClearExecution", nil},
		{"containsAll", "boolean", "AbstractCollectionContainsAllExecution", []string{"java.util.Collection<?>"}},
		{"indexOf", "int", "AbstractListIndexOfExecution", []string{"java.lang.Object"}},
		{"lastIndexOf", "int", "AbstractListLastIndexOfExecution", []string{"java.lang.Object"}},
		{"iterator", "java.util.Iterator<" + element + ">", "AbstractListIteratorExecution", nil},
		{"listIterator", "java.util.ListIterator<" + element + ">", "AbstractListListIteratorDefaultExecution", nil},
		{"listIterator", "java.util.ListIterator<" + element + ">", "AbstractListListIteratorExecution", []string{"int"}},
		{"subList", "java.util.List<" + element + ">", "AbstractListSubListTypedExecution", []string{"int", "int"}},
		{"equals", "boolean", "AbstractListEqualsExecution", []string{"java.lang.Object"}},
		{"hashCode", "int", "AbstractListHashCodeExecution", nil},
		{"toString", "java.lang.String", "AbstractListStringExecution", nil},
		{"addAll", "boolean", "AbstractListAddAllExecution", []string{"java.util.Collection<?>"}},
		{"addAll", "boolean", "AbstractListAddAllAtExecution", []string{"int", "java.util.Collection<?>"}},
		{"removeAll", "boolean", "AbstractListRemoveAllExecution", []string{"java.util.Collection<?>"}},
		{"retainAll", "boolean", "AbstractListRetainAllExecution", []string{"java.util.Collection<?>"}},
		{"toArray", "java.lang.Object[]", "AbstractListToArrayExecution", nil},
		{"toArray", "java.lang.Object[]", "AbstractListToArrayIntoExecution", []string{"java.lang.Object[]"}},
		{"removeRange", "void", "AbstractListRemoveRangeExecution", []string{"int", "int"}},
	}
}
func sourceAbstractListDefaultSpec(method *symbol.Definition, ctx Ctx) (abstractCollectionDefault, bool) {
	for _, spec := range sourceAbstractListDefaults(ctx.currentClass, ctx) {
		if method.OriginalName != spec.java || len(method.Parameters) != len(spec.parameters) {
			continue
		}
		match := true
		for i, p := range spec.parameters {
			match = match && javaInferenceSameType(method.Parameters[i].OriginalType, p, ctx)
		}
		if match {
			return spec, true
		}
	}
	return abstractCollectionDefault{}, false
}
func sourceAbstractListDefaultDecls(ctx Ctx) []ast.Decl {
	var declarations []ast.Decl
	if state := sourceAbstractListStateDecl(ctx); state != nil {
		declarations = append(declarations, state)
	}
	for _, method := range ctx.currentClass.Methods {
		if !method.RuntimeDefault {
			continue
		}
		spec, known := sourceAbstractListDefaultSpec(method, ctx)
		if !known {
			continue
		}
		params := &ast.FieldList{}
		var args []ast.Expr
		for _, p := range method.Parameters {
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(p.Name)}, Type: javaTypeStringToGoTypeExpr(p.OriginalType, inScopeTypeParameters(ctx), ctx)})
			args = append(args, ast.NewIdent(p.Name))
		}
		sourceParams := cloneFieldList(params)
		sourceCtx := ctx.Clone()
		sourceCtx.localScope = method
		sourceCtx.sourceNativeVirtualBody = sourceAbstractListVirtualBody(ctx.currentClass, method, sourceCtx)
		if plan := sourceCtx.sourceNativeVirtualBody; plan != nil {
			for i, p := range method.Parameters {
				if sourceNativeVirtualScalarDefinition(p, plan.declarations) {
					params.List[i].Type = ast.NewIdent("any")
				}
			}
		}
		receiver := abstractCollectionReceiverName(ctx, params)
		execution := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
		args = append([]ast.Expr{ast.NewIdent(execution), abstractCollectionDynamicReference(ctx, receiver)}, args...)
		var result *ast.FieldList
		if spec.result != "void" {
			result = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(spec.result, inScopeTypeParameters(ctx), sourceCtx)}}}
		}
		sourceResults := cloneFieldList(result)
		if sourceResults != nil {
			sourceResults.List[0].Type = javaTypeStringToGoTypeExpr(spec.result, inScopeTypeParameters(ctx), ctx)
		}
		var call ast.Expr = stdjavaCall(ctx, spec.runtime, args...)
		if spec.java == "subList" {
			call = stdjavaGenericCall(ctx, spec.runtime, []ast.Expr{javaTypeStringToGoTypeExpr(sourceAbstractListElement(ctx.currentClass, ctx), inScopeTypeParameters(ctx), ctx)}, args)
		}
		if spec.java == "set" || spec.java == "remove" && len(spec.parameters) == 1 && spec.parameters[0] == "int" {
			call = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(spec.result, inScopeTypeParameters(ctx), sourceCtx)}, []ast.Expr{call, stdjavaQualifiedExpr("ObjectTypeID", ctx)})
		}
		declaration := &ast.FuncDecl{Name: ast.NewIdent(method.Name), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: result}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), invocationClosureCallStatement(call, result)}}}
		if native, ok := buildSourceNativeVirtualBodyDecls(declaration, sourceParams, sourceResults, execution, sourceCtx); ok {
			declarations = append(declarations, native...)
		} else {
			declarations = append(declarations, buildExecutionAwareFuncDecls(declaration, executionImplementationName(method, ctx.currentClass, ctx), execution, ctx)...)
		}
	}
	return declarations
}

func sourceAbstractListSelectedMethod(scope *symbol.ClassScope, name string, arity int, indexed bool, ctx Ctx) (*symbol.Definition, *symbol.ClassScope) {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		seen[current] = true
		declaring := classScopeCtx(current, ctx)
		for _, m := range current.Methods {
			if m == nil || m.OriginalName != name || m.IsStatic || m.IsPrivate || !m.HasBody || len(m.Parameters) != arity {
				continue
			}
			if arity > 0 && (name == "get" || name == "set" || name == "add" && arity == 2 || name == "remove" || name == "listIterator") {
				first := m.Parameters[0].OriginalType
				isInt := javaInferenceSameType(first, "int", declaring)
				if isInt != indexed {
					continue
				}
			}
			if sourceAbstractListSlotMatches(m, current, scope, ctx) {
				return m, current
			}
		}
	}
	return nil, nil
}
func sourceAbstractListBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if sourceAbstractListRoot(scope, ctx) == nil {
		return nil
	}
	var declarations []ast.Decl
	for _, spec := range []struct {
		name, bridge, result string
		parameters           []string
		indexed              bool
	}{
		{"get", "CollectionListGetJava2goExecution", "java.lang.Object", []string{"int"}, true},
		{"set", "CollectionListSetJava2goExecution", "java.lang.Object", []string{"int", "java.lang.Object"}, true},
		{"add", "CollectionListAddAtJava2goExecution", "void", []string{"int", "java.lang.Object"}, true},
		{"remove", "CollectionListRemoveAtJava2goExecution", "java.lang.Object", []string{"int"}, true},
		{"add", "CollectionAddJava2goExecution", "boolean", []string{"java.lang.Object"}, false},
		{"indexOf", "CollectionListIndexOfJava2goExecution", "int", []string{"java.lang.Object"}, false},
		{"lastIndexOf", "CollectionListLastIndexOfJava2goExecution", "int", []string{"java.lang.Object"}, false},
		{"listIterator", "CollectionListIteratorJava2goExecution", "java.util.ListIterator", []string{"int"}, true},
		{"subList", "CollectionListSubListJava2goExecution", "java.lang.Iterable", []string{"int", "int"}, true},
		{"removeRange", "CollectionListRemoveRangeJava2goExecution", "void", []string{"int", "int"}, true},
		{"addAll", "CollectionListAddAllAtJava2goExecution", "boolean", []string{"int", "java.util.Collection<?>"}, true},
		{"addAll", "CollectionAddAllJava2goExecution", "boolean", []string{"java.util.Collection<?>"}, false},
		{"removeAll", "CollectionRemoveAllJava2goExecution", "boolean", []string{"java.util.Collection<?>"}, false},
		{"retainAll", "CollectionRetainAllJava2goExecution", "boolean", []string{"java.util.Collection<?>"}, false},
		{"toArray", "CollectionToArrayJava2goExecution", "java.lang.Object[]", nil, false},
		{"toArray", "CollectionToArrayIntoJava2goExecution", "java.lang.Object[]", []string{"java.lang.Object[]"}, false},
	} {
		method, owner := sourceAbstractListSelectedMethod(scope, spec.name, len(spec.parameters), spec.indexed, ctx)
		if method == nil {
			continue
		}
		params := &ast.FieldList{}
		execution := "__java2goAbstractListExecution"
		receiver := abstractCollectionReceiverName(ctx, params)
		var args []ast.Expr
		for i, p := range spec.parameters {
			name := fmt.Sprintf("__java2goAbstractListArgument%d", i)
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: javaTypeStringToGoTypeExpr(p, inScopeTypeParameters(ctx), ctx)})
			value := ast.Expr(ast.NewIdent(name))
			if p == "java.lang.Object" && (spec.name == "add" || spec.name == "set") {
				target := sourceAbstractListInstantiatedMethodType(method.Parameters[i].OriginalType, owner, scope, ctx)
				if erased, known := sourceNativeVirtualParameterErasure(owner, method, i, ctx); known {
					target = erased
				}
				descriptor, known := javaTypeDescriptorExpr(target, ctx)
				if !known {
					descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
				}
				value = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(target, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{value, descriptor})
			}
			args = append(args, value)
		}
		params.List = append([]*ast.Field{executionParameterField(execution, ctx)}, params.List...)
		args = append([]ast.Expr{ast.NewIdent(execution)}, args...)
		var selected ast.Expr = ast.NewIdent(receiver)
		if owner != scope {
			selected = sourceClassViewExpr(scope, owner, selected, ctx)
		}
		if method.RequiresHelper {
			logicalTypes := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), owner, ctx)
			logical := []ast.Expr{}
			for _, typ := range logicalTypes {
				logical = append(logical, javaTypeStringToGoTypeExpr(typ, inScopeTypeParameters(ctx), ctx))
			}
			logical = genericPrivateHelperOwnerTypeArguments(owner, method, logical, ctx)
			for range method.TypeParameters {
				logical = append(logical, ast.NewIdent("any"))
			}
			selected = &ast.CallExpr{Fun: applyTypeArguments(qualifiedNameExpr("New"+method.HelperName, findJavaPackageForClassScope(owner), ctx), logical), Args: []ast.Expr{selected}}
		} else if classNeedsVirtualDispatch(owner, ctx) {
			selected = &ast.SelectorExpr{X: selected, Sel: ast.NewIdent(classDispatchFieldName(owner))}
		}
		call := methodCall(selected, executionImplementationName(method, owner, ctx), args...)
		var result *ast.FieldList
		if spec.result != "void" {
			result = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(spec.result, inScopeTypeParameters(ctx), ctx)}}}
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(spec.bridge), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: result}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), invocationClosureCallStatement(call, result)}}})
	}
	return declarations
}
func sourceAbstractListBoundaryConversion(value ast.Expr, node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if value == nil || node == nil {
		return nil, false
	}
	base, _ := parseJavaTypeString(expected)
	owner := platformTypeOwner(base, nil, ctx, nil)
	if owner != "java.util.List" && owner != "java.util.AbstractList" {
		return nil, false
	}
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(node, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.arrayRank != 0 || origin.nominalScope == nil || sourceAbstractListRoot(origin.nominalScope, ctx) == nil {
		return nil, false
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{javaTypeStringToGoTypeExpr(expected, inScopeTypeParameters(ctx), ctx)}, []ast.Expr{value, javaTypeIDLiteral(owner, ctx)}), true
}
func sourceAbstractListModCountExpr(node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	if node == nil || ctx.currentClass == nil || ctx.localScope == nil || ctx.localScope.IsStatic || sourceAbstractListRoot(ctx.currentClass, ctx) == nil || !sourceArrayListNativeSourceContext(node, ctx, source) {
		return nil
	}
	explicitSuper := false
	switch node.Type() {
	case "identifier":
		if node.Content(source) != "modCount" || ctx.localScope.ParameterByName("modCount") != nil || ctx.localScope.FindVariable("modCount") != nil {
			return nil
		}
	case "field_access":
		field, object := node.ChildByFieldName("field"), node.ChildByFieldName("object")
		if field == nil || field.Content(source) != "modCount" || object == nil || (object.Type() != "this" && object.Type() != "super") {
			return nil
		}
		explicitSuper = object.Type() == "super"
	default:
		return nil
	}
	if !explicitSuper && findFieldInHierarchy(ctx.currentClass, "modCount", ctx) != nil {
		return nil
	}
	if explicitSuper && !sourceAbstractListDirectRoot(ctx.currentClass, ctx) {
		return nil
	}
	return &ast.SelectorExpr{X: methodCall(sourceAbstractListNativeViewExpr(ctx.currentClass, ast.NewIdent(sourceInstanceReceiverName(ctx)), ctx), "AbstractListStateJava2go"), Sel: ast.NewIdent("ModCount")}
}
func abstractListIteratorRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	base, args := parseJavaTypeString(strings.TrimSpace(javaType))
	if containsString(typeParams, base) || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return nil, false
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return nil, false
	}
	switch owner {
	case "java.util.ListIterator":
		if len(args) > 1 || len(typeArgs) > 1 {
			return nil, false
		}
		return stdjavaQualifiedExpr("JavaListIterator", ctx), true
	case "java.util.RandomAccess":
		if len(args) != 0 || len(typeArgs) != 0 {
			return nil, false
		}
		return ast.NewIdent("any"), true
	}
	return nil, false
}

// A local declaration may donate a platform view only through the active
// synthesis witness. A copied scope or caller-created lookup map is insufficient.
func sourceAbstractListLocalDeclaration(scope *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	frame := ctx.lateSourceOwnership
	if frame == nil || frame != activeLateSourceOwnership || ctx.currentFile != frame.file || ctx.syntaxInvocation != frame.invocation || !frame.current() || !frame.ownerCurrent(scope) {
		return Ctx{}, false
	}
	registration, ok := frame.late[scope]
	if !ok || registration.local == nil || ctx.localClasses[registration.name] != registration.local || !frame.registrationCurrent(scope, registration) {
		return Ctx{}, false
	}
	declaring, ok := threadSourceDeclarationCtx(scope, nil, ctx)
	if !ok || declaring.currentFile != frame.file || frame.snapshot == nil || frame.snapshot.Validate() != nil {
		return Ctx{}, false
	}
	node := scope.Class.DeclarationNode
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(frame.file.Source) != scope.Class.OriginalName {
		return Ctx{}, false
	}
	var written []string
	if superclass := node.ChildByFieldName("superclass"); superclass != nil {
		types := supplierHeaderTypeNodes(superclass)
		if len(types) != 1 {
			return Ctx{}, false
		}
		written = append(written, types[0].Content(frame.file.Source))
	} else {
		written = append(written, "")
	}
	if interfaces := node.ChildByFieldName("interfaces"); interfaces != nil {
		for _, typ := range supplierHeaderTypeNodes(interfaces) {
			written = append(written, typ.Content(frame.file.Source))
		}
	}
	metadata := append([]string{scope.Superclass}, scope.ImplementedInterfaces...)
	if len(written) != len(metadata) {
		return Ctx{}, false
	}
	for i := range written {
		if written[i] != metadata[i] {
			return Ctx{}, false
		}
	}
	return declaring, true
}

func sourceAbstractListSyntheticIdentityDecls(name, id string, scope, parent *symbol.ClassScope, interfaces []*symbol.ClassScope, ctx Ctx) []ast.Decl {
	declarations := syntheticHierarchicalReferenceIdentityDecls(name, id, scope, parent, interfaces, scope.GoTypeParameterNames(), ctx)
	view := declarations[1].(*ast.FuncDecl)
	receiver := view.Recv.List[0].Names[0].Name
	cases := view.Body.List[0].(*ast.SwitchStmt).Body
	native := sourceAbstractListNativeViewExpr(scope, ast.NewIdent(receiver), ctx)
	for _, name := range []string{"java.util.AbstractList", "java.util.List", "java.util.Collection", "java.lang.Iterable"} {
		value := native
		if name == "java.util.Collection" || name == "java.lang.Iterable" {
			value = ast.NewIdent(receiver)
		}
		clause := &ast.CaseClause{List: []ast.Expr{javaTypeIDLiteral(name, ctx)}, Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{value}}}}
		cases.List = append(cases.List[:len(cases.List)-1], clause, cases.List[len(cases.List)-1])
	}
	if sourceAbstractListDirectRoot(scope, ctx) {
		interfaceIDs := sourceDeclaredExternalInterfaceIDs(scope, ctx)
		for _, implemented := range interfaces {
			if implemented != nil {
				interfaceIDs = append(interfaceIDs, javaTypeIDLiteral(javaClassBinaryName(implemented), ctx))
			}
		}
		declarations[2] = syntheticReferenceRegistrationDecl(name, id, javaTypeIDLiteral("java.util.AbstractList", ctx), interfaceIDs, ctx)
		declarations = append(declarations, sourceAbstractListObjectInfoDecl(scope, view.Recv, receiver, ctx))
	}
	return declarations
}

// Late platform defaults enter after the normal overload allocation pass.
// Preserve written source selectors and allocate each new default in their
// shared Go namespace before bridges or method bodies use its definition.
func sourceAbstractListAllocateLocalDefaults(scope *symbol.ClassScope, ctx Ctx) {
	if sourceAbstractListRoot(scope, ctx) == nil {
		return
	}
	used := map[string]struct{}{}
	for _, field := range scope.Fields {
		if field != nil {
			used[field.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method != nil && !method.RuntimeDefault {
			used[method.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method == nil || !method.RuntimeDefault {
			continue
		}
		name := method.Name
		if _, occupied := used[name]; occupied {
			name = nextSyntheticMemberName(name, used)
		}
		method.Rename(name)
		used[name] = struct{}{}
	}
}
