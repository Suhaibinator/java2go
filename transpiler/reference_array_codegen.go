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

const (
	generatedDynamicTypeMethod = "Java2goReferenceDynamicType"
	generatedObjectViewMethod  = "Java2goReferenceView"
)

func referenceIdentityReservedSelector(name string) bool {
	switch name {
	case "ObjectInfo", "JavaObjectInfo", "JavaDynamicTypeID", generatedDynamicTypeMethod, generatedObjectViewMethod:
		return true
	default:
		return false
	}
}

// javaArrayTypeParts separates the immediate source component from its array
// rank. The retained Java spelling, rather than the generated Go type, is the
// authority for choosing the reified reference-array ABI.
func javaArrayTypeParts(javaType string) (string, int) {
	component := strings.TrimSpace(javaType)
	rank := 0
	for strings.HasSuffix(component, "[]") {
		rank++
		component = strings.TrimSpace(strings.TrimSuffix(component, "[]"))
	}
	return component, rank
}

func javaPrimitiveArrayComponent(javaType string) (string, bool) {
	base, rank := javaArrayTypeParts(javaType)
	if rank != 1 {
		return "", false
	}
	switch stripJavaQualifier(base) {
	case "boolean", "byte", "short", "char", "int", "long", "float", "double":
		return base, true
	default:
		return "", false
	}
}

// reifiedSourceReferenceArrayComponent recognizes every array whose immediate
// component is a Java reference. This includes every rank above a primitive
// leaf: int[][] has a reference-array outer ABI and a PrimitiveArray[int32]
// leaf. A single common pointer ABI is essential for covariant aliases and
// erased Object views; the runtime descriptor, not the Go shape, distinguishes
// Child[] from Base[].
func reifiedSourceReferenceArrayComponent(javaType string, ctx Ctx) (string, *symbol.ClassScope, bool) {
	base, rank := javaArrayTypeParts(javaType)
	if rank < 1 || base == "" {
		return "", nil, false
	}
	if rank == 1 {
		if _, primitive := javaPrimitiveArrayComponent(javaType); primitive {
			return "", nil, false
		}
	}
	immediate := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(javaType), "[]"))
	resolvedBase, _ := parseJavaTypeString(base)
	scope := resolveClassScopeByQualifiedName(ctx, resolvedBase)
	return immediate, scope, true
}

func primitiveArrayTypeExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	component, ok := javaPrimitiveArrayComponent(javaType)
	if !ok {
		return nil, false
	}
	elementType := javaTypeStringToGoTypeExpr(component, inScopeTypeParameters(ctx), ctx)
	return &ast.StarExpr{X: &ast.IndexExpr{
		X:     stdjavaQualifiedExpr("PrimitiveArray", ctx),
		Index: elementType,
	}}, true
}

func reifiedReferenceArrayTypeExpr(ctx Ctx) ast.Expr {
	return &ast.StarExpr{X: stdjavaQualifiedExpr("ReferenceArray", ctx)}
}

func javaTypeIDLiteral(id string, ctx Ctx) ast.Expr {
	return &ast.CallExpr{
		Fun: stdjavaQualifiedExpr("TypeID", ctx),
		Args: []ast.Expr{&ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(id),
		}},
	}
}

func sourceReferenceTypeIDExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if scope == nil || scope.Class == nil {
		return nil, false
	}
	id := javaClassBinaryName(scope)
	for _, local := range ctx.localClasses {
		if local != nil && local.scope == scope && local.dynamicTypeID != "" {
			id = local.dynamicTypeID
			break
		}
	}
	if id == "" {
		return nil, false
	}
	return javaTypeIDLiteral(id, ctx), true
}

func javaPrimitiveTypeIDExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	switch stripJavaQualifier(strings.TrimSpace(javaType)) {
	case "boolean":
		return stdjavaQualifiedExpr("PrimitiveBooleanTypeID", ctx), true
	case "byte":
		return stdjavaQualifiedExpr("PrimitiveByteTypeID", ctx), true
	case "short":
		return stdjavaQualifiedExpr("PrimitiveShortTypeID", ctx), true
	case "char":
		return stdjavaQualifiedExpr("PrimitiveCharTypeID", ctx), true
	case "int":
		return stdjavaQualifiedExpr("PrimitiveIntTypeID", ctx), true
	case "long":
		return stdjavaQualifiedExpr("PrimitiveLongTypeID", ctx), true
	case "float":
		return stdjavaQualifiedExpr("PrimitiveFloatTypeID", ctx), true
	case "double":
		return stdjavaQualifiedExpr("PrimitiveDoubleTypeID", ctx), true
	default:
		return nil, false
	}
}

// javaTypeParameterErasure returns the Java erasure of an in-scope type
// parameter. Generic code is generated once, so a use of T in T[] cannot name
// the caller's inferred instantiation in its static descriptor. Java instead
// uses the first declared bound, or Object when T is unbounded.
func javaTypeParameterErasure(javaType string, ctx Ctx) (string, bool) {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	name := stripJavaQualifier(base)
	if name == "" || len(arguments) != 0 {
		return "", false
	}

	find := func(parameters []symbol.TypeParam) (string, bool) {
		for _, parameter := range parameters {
			if parameter.Name != name {
				continue
			}
			if len(parameter.Bounds) == 0 || strings.TrimSpace(parameter.Bounds[0].Original) == "" {
				return "Object", true
			}
			return strings.TrimSpace(parameter.Bounds[0].Original), true
		}
		return "", false
	}

	// Method parameters shadow synthetic/raw parameters and class parameters.
	if ctx.localScope != nil {
		if erased, ok := find(ctx.localScope.TypeParameters); ok {
			return erased, true
		}
	}
	if erased, ok := find(ctx.syntheticTypeParameters); ok {
		return erased, true
	}
	if ctx.currentClass != nil && (ctx.localScope == nil || !ctx.localScope.IsStatic) {
		if erased, ok := find(ctx.currentClass.TypeParameters); ok {
			return erased, true
		}
	}
	return "", false
}

// javaTypeDescriptorExpr returns the nominal runtime descriptor for a Java
// type. Array descriptors nest around the immediate component descriptor.
func javaTypeDescriptorExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	javaType = strings.TrimSpace(javaType)
	if javaType == "" {
		return nil, false
	}
	if strings.HasSuffix(javaType, "[]") {
		component := strings.TrimSpace(strings.TrimSuffix(javaType, "[]"))
		descriptor, ok := javaTypeDescriptorExpr(component, ctx)
		if !ok {
			return nil, false
		}
		return stdjavaCall(ctx, "ArrayTypeID", descriptor), true
	}
	if primitive, ok := javaPrimitiveTypeIDExpr(javaType, ctx); ok {
		return primitive, true
	}
	if erased, ok := javaTypeParameterErasure(javaType, ctx); ok {
		// A legal Java bound cannot form a cycle, but guard malformed/incomplete
		// symbols so descriptor lowering remains finite.
		if strings.TrimSpace(erased) == strings.TrimSpace(javaType) {
			return stdjavaQualifiedExpr("ObjectTypeID", ctx), true
		}
		return javaTypeDescriptorExpr(erased, ctx)
	}
	base, _ := parseJavaTypeString(javaType)
	baseName := stripJavaQualifier(base)
	builtin := map[string]string{
		"Object": "ObjectTypeID", "String": "StringTypeID",
		"Throwable": "ThrowableTypeID",
		"Thread":    "ThreadTypeID", "Runnable": "RunnableTypeID",
		"Cloneable": "CloneableTypeID", "Serializable": "SerializableTypeID",
		"Boolean": "BooleanTypeID", "Byte": "ByteTypeID", "Short": "ShortTypeID",
		"Character": "CharacterTypeID", "Integer": "IntegerTypeID", "Long": "LongTypeID",
		"Float": "FloatTypeID", "Double": "DoubleTypeID",
	}
	if constant, ok := builtin[baseName]; ok {
		return stdjavaQualifiedExpr(constant, ctx), true
	}
	if source, ok := sourceReferenceTypeIDExpr(base, ctx); ok {
		return source, true
	}
	// External reference types still need stable exact descriptors. Hierarchy
	// edges can be registered incrementally by their runtime adapters.
	return javaTypeIDLiteral(base, ctx), true
}

func reifiedReferenceArrayComponentInfo(javaType string, ctx Ctx) (string, ast.Expr, ast.Expr, bool) {
	component, _, ok := reifiedSourceReferenceArrayComponent(javaType, ctx)
	if !ok {
		return "", nil, nil, false
	}
	componentType := javaTypeStringToGoTypeExpr(component, inScopeTypeParameters(ctx), ctx)
	componentType = abstractClassToInterface(componentType, component, ctx)
	componentID, ok := javaTypeDescriptorExpr(component, ctx)
	if !ok {
		return "", nil, nil, false
	}
	return component, componentType, componentID, true
}

func allSourceClassScopes() []*symbol.ClassScope {
	var scopes []*symbol.ClassScope
	for _, pkg := range symbol.GlobalScope.Packages {
		if pkg == nil {
			continue
		}
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			var visit func(*symbol.ClassScope)
			visit = func(scope *symbol.ClassScope) {
				if scope == nil {
					return
				}
				scopes = append(scopes, scope)
				for _, nested := range scope.Subclasses {
					visit(nested)
				}
			}
			for _, top := range file.TopLevelClasses {
				visit(top)
			}
		}
	}
	return scopes
}

// classHasSyntheticSubclass finds subclass relationships represented only in
// source syntax. Anonymous and method-local classes are hoisted during lowering
// and therefore are absent from the ordinary global ClassScope hierarchy.
func classHasSyntheticSubclass(target *symbol.ClassScope, ctx Ctx) bool {
	if target == nil {
		return false
	}
	activeLocalTarget := false
	for _, info := range ctx.localClasses {
		if info != nil && info.scope == target {
			activeLocalTarget = true
			break
		}
	}
	for _, owner := range allSourceClassScopes() {
		if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
			continue
		}
		file := findFileScopeForClassScope(owner)
		if file == nil {
			continue
		}
		ownerCtx := Ctx{currentFile: file, currentClass: owner}
		if activeLocalTarget {
			// A method-local target exists only in the active lowering registry.
			// Preserve that map solely while scanning the same source file; carrying
			// it into another file could hijack an unrelated same-simple-name type.
			if file != ctx.currentFile {
				continue
			}
			ownerCtx = ctx.Clone()
			ownerCtx.currentFile = file
		}
		var visit func(*sitter.Node) bool
		visit = func(node *sitter.Node) bool {
			if node == nil {
				return false
			}
			var supertype *sitter.Node
			switch node.Type() {
			case "object_creation_expression":
				for _, child := range nodeutil.NamedChildrenOf(node) {
					if child.Type() == "class_body" {
						supertype = node.ChildByFieldName("type")
						break
					}
				}
			case "class_declaration":
				if superclass := node.ChildByFieldName("superclass"); superclass != nil {
					types := collectTypeNodes(superclass)
					if len(types) > 0 {
						supertype = types[0]
					}
				}
			}
			if supertype != nil {
				base, _ := parseJavaTypeString(supertype.Content(file.Source))
				if resolveClassScopeByQualifiedName(ownerCtx, base) == target {
					return true
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if visit(child) {
					return true
				}
			}
			return false
		}
		if visit(owner.Class.DeclarationNode) {
			return true
		}
	}
	return false
}

func addReferenceArrayTypeSeed(javaType string, owner *symbol.ClassScope, ctx Ctx, seeds map[*symbol.ClassScope]struct{}, objectComponent *bool) {
	declarationCtx := ctx.Clone()
	declarationCtx.currentClass = owner
	if file := findFileScopeForClassScope(owner); file != nil {
		declarationCtx.currentFile = file
	}
	_, component, ok := reifiedSourceReferenceArrayComponent(javaType, declarationCtx)
	if ok && component != nil {
		seeds[component] = struct{}{}
		return
	}
	if !ok {
		return
	}

	base, _ := javaArrayTypeParts(javaType)
	if erased, typeParameter := javaTypeParameterErasure(base, declarationCtx); typeParameter {
		base = erased
	}
	erasedBase, _ := parseJavaTypeString(base)
	if stripJavaQualifier(erasedBase) == "Object" {
		*objectComponent = true
		return
	}
	if erasedScope := resolveClassScopeByQualifiedName(declarationCtx, erasedBase); erasedScope != nil {
		seeds[erasedScope] = struct{}{}
	}
}

func addReferenceArrayDefinitionSeeds(definition *symbol.Definition, owner *symbol.ClassScope, ctx Ctx, seeds map[*symbol.ClassScope]struct{}, objectComponent *bool) {
	if definition == nil {
		return
	}
	addReferenceArrayTypeSeed(definition.OriginalType, owner, ctx, seeds, objectComponent)
	for _, parameter := range definition.Parameters {
		addReferenceArrayDefinitionSeeds(parameter, owner, ctx, seeds, objectComponent)
	}
	for _, child := range definition.Children {
		addReferenceArrayDefinitionSeeds(child, owner, ctx, seeds, objectComponent)
	}
}

func addReferenceArrayCreationSeeds(node *sitter.Node, source []byte, owner *symbol.ClassScope, ctx Ctx, seeds map[*symbol.ClassScope]struct{}, objectComponent *bool) {
	if node == nil {
		return
	}
	if node.Type() == "array_creation_expression" {
		if javaType, rank := javaArrayCreationJavaType(node, source); rank > 0 {
			addReferenceArrayTypeSeed(javaType, owner, ctx, seeds, objectComponent)
		}
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		addReferenceArrayCreationSeeds(child, source, owner, ctx, seeds, objectComponent)
	}
}

func classDirectlyReferencesRelevantInterface(scope *symbol.ClassScope, relevant map[*symbol.ClassScope]struct{}, ctx Ctx) bool {
	for _, implemented := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope) {
		if _, ok := relevant[implemented]; ok {
			return true
		}
	}
	return false
}

// referenceIdentityScopes computes the smallest source hierarchy set that can
// contribute objects to a reified rank-one reference array. Unrelated classes
// retain their historical structs and constructors; this avoids imposing
// runtime metadata or imports on programs that never use reference arrays.
func referenceIdentityScopes(ctx Ctx) map[*symbol.ClassScope]struct{} {
	allScopes := allSourceClassScopes()
	relevant := map[*symbol.ClassScope]struct{}{}
	objectComponent := false
	for _, scope := range allScopes {
		for _, field := range scope.Fields {
			fieldCtx := ctx.Clone()
			fieldCtx.currentClass = scope
			fieldCtx.localScope = nil
			addReferenceArrayDefinitionSeeds(field, scope, fieldCtx, relevant, &objectComponent)
		}
		for _, method := range scope.Methods {
			methodCtx := ctx.Clone()
			methodCtx.currentClass = scope
			methodCtx.localScope = method
			addReferenceArrayDefinitionSeeds(method, scope, methodCtx, relevant, &objectComponent)
		}
		if scope.Class != nil && scope.Class.DeclarationNode != nil {
			if file := findFileScopeForClassScope(scope); file != nil {
				addReferenceArrayCreationSeeds(scope.Class.DeclarationNode, file.Source, scope, ctx, relevant, &objectComponent)
			}
		}
	}
	if objectComponent {
		// Object[] (and an unbounded T[]) may receive an instance of any concrete
		// source class even when no corresponding Foo[] declaration exists. Keep
		// this conservative expansion conditional on such an erased component so
		// programs without reference arrays retain their historical Go shape.
		for _, scope := range allScopes {
			if scope != nil && !scope.IsInterface && !scope.IsAbstract {
				relevant[scope] = struct{}{}
			}
		}
	}

	changed := true
	for changed {
		changed = false
		add := func(scope *symbol.ClassScope) {
			if scope == nil {
				return
			}
			if _, exists := relevant[scope]; exists {
				return
			}
			relevant[scope] = struct{}{}
			changed = true
		}
		for scope := range relevant {
			add(resolveSuperclassScopeInDeclaringContext(ctx, scope))
			for _, implemented := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope) {
				add(implemented)
			}
		}
		for _, candidate := range allScopes {
			if _, exists := relevant[candidate]; exists {
				continue
			}
			if parent := resolveSuperclassScopeInDeclaringContext(ctx, candidate); parent != nil {
				if _, parentRelevant := relevant[parent]; parentRelevant {
					add(candidate)
					continue
				}
			}
			if classDirectlyReferencesRelevantInterface(candidate, relevant, ctx) {
				add(candidate)
			}
		}
	}
	return relevant
}

func classNeedsReferenceIdentity(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	_, ok := referenceIdentityScopes(ctx)[scope]
	return ok
}

func expressionUsesReifiedReferenceArray(node *sitter.Node, ctx Ctx, source []byte) (string, ast.Expr, ast.Expr, bool) {
	if node == nil {
		return "", nil, nil, false
	}
	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return "", nil, nil, false
	}
	return reifiedReferenceArrayComponentInfo(javaType, ctx)
}

func expressionUsesPrimitiveArray(node *sitter.Node, ctx Ctx, source []byte) (string, ast.Expr, ast.Expr, bool) {
	if node == nil {
		return "", nil, nil, false
	}
	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		return "", nil, nil, false
	}
	component, ok := javaPrimitiveArrayComponent(javaType)
	if !ok {
		return "", nil, nil, false
	}
	componentType := javaTypeStringToGoTypeExpr(component, inScopeTypeParameters(ctx), ctx)
	componentID, _ := javaPrimitiveTypeIDExpr(component, ctx)
	return component, componentType, componentID, true
}

func sourceHierarchyUsesMostDerived(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.IsInterface || scope.IsEnum {
		return false
	}
	return resolveSuperclassScopeInDeclaringContext(ctx, scope) != nil || classHasKnownSubclass(scope, ctx)
}

func classNeedsReferenceObjectInfo(scope *symbol.ClassScope, ctx Ctx) bool {
	return classNeedsReferenceIdentity(scope, ctx) && sourceHierarchyUsesMostDerived(scope, ctx)
}

func constructorUsesMostDerived(scope *symbol.ClassScope, ctx Ctx) bool {
	return classHasSelfSetter(scope, ctx) ||
		classNeedsReferenceObjectInfo(scope, ctx)
}

func sourceHierarchyRoot(scope *symbol.ClassScope, ctx Ctx) bool {
	return scope != nil && !scope.IsInterface && !scope.IsEnum && resolveSuperclassScopeInDeclaringContext(ctx, scope) == nil
}

func generatedObjectInfoField(ctx Ctx) *ast.Field {
	return &ast.Field{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ObjectInfo", ctx)}}
}

func constructorObjectInfoInitStmt(scope *symbol.ClassScope, receiverName string, mostDerived ast.Expr, ctx Ctx) ast.Stmt {
	if !classNeedsReferenceObjectInfo(scope, ctx) || !sourceHierarchyRoot(scope, ctx) || receiverName == "" {
		return nil
	}
	value := mostDerived
	if value == nil {
		value = &ast.Ident{Name: receiverName}
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: "ObjectInfo"},
		}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{stdjavaCall(ctx, "NewGeneratedObjectInfo", value)},
	}
}

func sourceClassRegistrationDecl(scope *symbol.ClassScope, ctx Ctx) ast.Decl {
	if scope == nil || scope.Class == nil || !classNeedsReferenceIdentity(scope, ctx) {
		return nil
	}
	id := javaClassBinaryName(scope)
	if id == "" {
		return nil
	}
	args := []ast.Expr{javaTypeIDLiteral(id, ctx)}
	if parent := resolveSuperclassScopeInDeclaringContext(ctx, scope); parent != nil {
		args = append(args, javaTypeIDLiteral(javaClassBinaryName(parent), ctx))
	} else {
		args = append(args, stdjavaQualifiedExpr("ObjectTypeID", ctx))
	}
	for _, implementedScope := range resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope) {
		args = append(args, javaTypeIDLiteral(javaClassBinaryName(implementedScope), ctx))
	}

	name := collisionSafeExecutionIdentifier("__java2goReferenceTypeRegistration"+scope.Class.Name, scope)
	initializer := &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "bool"}}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: stdjavaCall(ctx, "RegisterJavaType", args...)},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "true"}}},
		}},
	}}
	return &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
		Names:  []*ast.Ident{{Name: name}},
		Values: []ast.Expr{initializer},
	}}}
}

// transitiveImplementedInterfaceScopes returns every direct and inherited
// source interface implemented by scope. Resolving each edge in the interface's
// declaring file avoids losing an unqualified parent when the original class is
// consumed from another package.
func transitiveImplementedInterfaceScopes(scope *symbol.ClassScope, ctx Ctx) []*symbol.ClassScope {
	if scope == nil {
		return nil
	}
	seen := make(map[*symbol.ClassScope]struct{})
	queue := append([]*symbol.ClassScope(nil), resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope)...)
	result := make([]*symbol.ClassScope, 0, len(queue))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil {
			continue
		}
		if _, duplicate := seen[current]; duplicate {
			continue
		}
		seen[current] = struct{}{}
		result = append(result, current)
		queue = append(queue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
	}
	return result
}

func sourceClassViewExpr(scope, requested *symbol.ClassScope, receiver ast.Expr, ctx Ctx) ast.Expr {
	if scope == nil || requested == nil || receiver == nil {
		return nil
	}
	view := receiver
	seen := map[*symbol.ClassScope]struct{}{}
	for current := scope; current != nil && current != requested; {
		if _, duplicate := seen[current]; duplicate {
			return nil
		}
		seen[current] = struct{}{}
		parent := resolveSuperclassScopeInDeclaringContext(ctx, current)
		if parent == nil || parent.Class == nil {
			return nil
		}
		view = &ast.SelectorExpr{X: view, Sel: &ast.Ident{Name: parent.Class.Name}}
		current = parent
	}
	return view
}

func sourceClassReferenceIdentityDecls(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	if scope == nil || scope.Class == nil || scope.IsInterface || !classNeedsReferenceIdentity(scope, ctx) {
		return nil
	}
	if scope.IsEnum {
		if declaration := fixedJavaDynamicTypeDecl(scope.Class.Name, javaClassBinaryName(scope), ctx); declaration != nil {
			return []ast.Decl{declaration}
		}
		return nil
	}
	receiverName := ShortName(scope.Class.Name)
	receiverType := &ast.StarExpr{X: instantiateGenericType(scope.Class.Name, typeParamExprs(scope.TypeParameterNames()))}
	receiver := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: receiverName}}, Type: receiverType}}}
	if !classNeedsReferenceObjectInfo(scope, ctx) {
		return []ast.Decl{&ast.FuncDecl{
			Name: &ast.Ident{Name: "JavaDynamicTypeID"},
			Recv: receiver,
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{
				Type: stdjavaQualifiedExpr("TypeID", ctx),
			}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
				javaTypeIDLiteral(javaClassBinaryName(scope), ctx),
			}}}},
		}}
	}

	dynamicType := &ast.FuncDecl{
		Name: &ast.Ident{Name: generatedDynamicTypeMethod},
		Recv: receiver,
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("TypeID", ctx)}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
			javaTypeIDLiteral(javaClassBinaryName(scope), ctx),
		}}}},
	}

	requestedName := "__java2goRequestedType"
	cases := []ast.Stmt{}
	seenIDs := map[string]struct{}{}
	appendCase := func(id string, value ast.Expr) {
		if id == "" || value == nil {
			return
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return
		}
		seenIDs[id] = struct{}{}
		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{javaTypeIDLiteral(id, ctx)},
			Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{value}}},
		})
	}

	receiverExpr := ast.Expr(&ast.Ident{Name: receiverName})
	seenScopes := map[*symbol.ClassScope]struct{}{}
	for current := scope; current != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		if _, duplicate := seenScopes[current]; duplicate {
			break
		}
		seenScopes[current] = struct{}{}
		appendCase(javaClassBinaryName(current), sourceClassViewExpr(scope, current, receiverExpr, ctx))
		for _, interfaceScope := range transitiveImplementedInterfaceScopes(current, ctx) {
			appendCase(javaClassBinaryName(interfaceScope), receiverExpr)
		}
	}
	cases = append(cases, &ast.CaseClause{Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "nil"}}}}})
	viewMethod := &ast.FuncDecl{
		Name: &ast.Ident{Name: generatedObjectViewMethod},
		Recv: receiver,
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: requestedName}},
				Type:  stdjavaQualifiedExpr("TypeID", ctx),
			}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "any"}}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.SwitchStmt{
			Tag:  &ast.Ident{Name: requestedName},
			Body: &ast.BlockStmt{List: cases},
		}}},
	}
	return []ast.Decl{dynamicType, viewMethod}
}

// fixedJavaDynamicTypeDecl gives a value whose Go pointer already is every
// required static view (enums and synthetic interface implementations) a
// lightweight nominal Java identity. These values do not need ObjectInfo's
// class-subobject view provider.
func fixedJavaDynamicTypeDecl(structName, dynamicID string, ctx Ctx) ast.Decl {
	return fixedJavaDynamicTypeDeclWithTypeParams(structName, dynamicID, nil, ctx)
}

func fixedJavaDynamicTypeDeclWithTypeParams(structName, dynamicID string, typeParams []string, ctx Ctx) ast.Decl {
	if structName == "" || dynamicID == "" {
		return nil
	}
	receiverName := "synthetic"
	receiverType := instantiateGenericType(structName, typeParamExprs(typeParams))
	return &ast.FuncDecl{
		Name: &ast.Ident{Name: "JavaDynamicTypeID"},
		Recv: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: receiverName}},
			Type:  &ast.StarExpr{X: receiverType},
		}}},
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: stdjavaQualifiedExpr("TypeID", ctx)}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
			javaTypeIDLiteral(dynamicID, ctx),
		}}}},
	}
}

func syntheticReferenceRegistrationDecl(
	structName string,
	dynamicID string,
	superID ast.Expr,
	interfaceIDs []ast.Expr,
	ctx Ctx,
) ast.Decl {
	if structName == "" || dynamicID == "" {
		return nil
	}
	if superID == nil {
		superID = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	args := []ast.Expr{javaTypeIDLiteral(dynamicID, ctx), superID}
	args = append(args, interfaceIDs...)
	registrationName := "__java2goSyntheticTypeRegistration" + sanitizeGoIdent(structName)
	initializer := &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "bool"}}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ExprStmt{X: stdjavaCall(ctx, "RegisterJavaType", args...)},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "true"}}},
		}},
	}}
	return &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{
		Names:  []*ast.Ident{{Name: registrationName}},
		Values: []ast.Expr{initializer},
	}}}
}

// syntheticHierarchicalReferenceIdentityDecls gives a hoisted local concrete
// subclass its own nominal runtime identity while exposing each embedded source
// superclass view. The hierarchy root owns ObjectInfo; the local child supplies
// the most-derived descriptor and view provider captured by that root.
func syntheticHierarchicalReferenceIdentityDecls(
	structName string,
	dynamicID string,
	superScope *symbol.ClassScope,
	directInterfaces []*symbol.ClassScope,
	typeParams []string,
	ctx Ctx,
) []ast.Decl {
	if structName == "" || dynamicID == "" || superScope == nil || superScope.Class == nil {
		return nil
	}

	receiverName := ShortName(structName)
	receiverExpr := ast.Expr(&ast.Ident{Name: receiverName})
	receiverType := instantiateGenericType(structName, typeParamExprs(typeParams))
	receiver := &ast.FieldList{List: []*ast.Field{{
		Names: []*ast.Ident{{Name: receiverName}},
		Type:  &ast.StarExpr{X: receiverType},
	}}}
	dynamicType := &ast.FuncDecl{
		Name: &ast.Ident{Name: generatedDynamicTypeMethod},
		Recv: receiver,
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{
			Type: stdjavaQualifiedExpr("TypeID", ctx),
		}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
			javaTypeIDLiteral(dynamicID, ctx),
		}}}},
	}

	requestedName := "__java2goRequestedType"
	var cases []ast.Stmt
	seenIDs := map[string]struct{}{}
	appendCase := func(id string, value ast.Expr) {
		if id == "" || value == nil {
			return
		}
		if _, duplicate := seenIDs[id]; duplicate {
			return
		}
		seenIDs[id] = struct{}{}
		cases = append(cases, &ast.CaseClause{
			List: []ast.Expr{javaTypeIDLiteral(id, ctx)},
			Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{value}}},
		})
	}
	appendCase(dynamicID, receiverExpr)
	for _, interfaceScope := range directInterfaces {
		if interfaceScope != nil {
			appendCase(javaClassBinaryName(interfaceScope), receiverExpr)
			for _, inherited := range transitiveImplementedInterfaceScopes(interfaceScope, ctx) {
				appendCase(javaClassBinaryName(inherited), receiverExpr)
			}
		}
	}

	view := ast.Expr(&ast.SelectorExpr{X: receiverExpr, Sel: &ast.Ident{Name: superScope.Class.Name}})
	seenScopes := map[*symbol.ClassScope]struct{}{}
	for current := superScope; current != nil && current.Class != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		if _, duplicate := seenScopes[current]; duplicate {
			break
		}
		seenScopes[current] = struct{}{}
		appendCase(javaClassBinaryName(current), view)
		for _, interfaceScope := range transitiveImplementedInterfaceScopes(current, ctx) {
			appendCase(javaClassBinaryName(interfaceScope), receiverExpr)
		}
		parent := resolveSuperclassScopeInDeclaringContext(ctx, current)
		if parent != nil && parent.Class != nil {
			view = &ast.SelectorExpr{X: view, Sel: &ast.Ident{Name: parent.Class.Name}}
		}
	}
	cases = append(cases, &ast.CaseClause{Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "nil"}}}}})
	viewMethod := &ast.FuncDecl{
		Name: &ast.Ident{Name: generatedObjectViewMethod},
		Recv: receiver,
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: requestedName}},
				Type:  stdjavaQualifiedExpr("TypeID", ctx),
			}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "any"}}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.SwitchStmt{
			Tag:  &ast.Ident{Name: requestedName},
			Body: &ast.BlockStmt{List: cases},
		}}},
	}

	interfaceIDs := make([]ast.Expr, 0, len(directInterfaces))
	for _, interfaceScope := range directInterfaces {
		if interfaceScope != nil && interfaceScope.Class != nil {
			interfaceIDs = append(interfaceIDs, javaTypeIDLiteral(javaClassBinaryName(interfaceScope), ctx))
		}
	}
	registration := syntheticReferenceRegistrationDecl(
		structName,
		dynamicID,
		javaTypeIDLiteral(javaClassBinaryName(superScope), ctx),
		interfaceIDs,
		ctx,
	)
	return []ast.Decl{dynamicType, viewMethod, registration}
}

func syntheticReferenceIdentityDeclsWithTypeParams(
	structName string,
	dynamicID string,
	superID ast.Expr,
	interfaceIDs []ast.Expr,
	typeParams []string,
	ctx Ctx,
) []ast.Decl {
	if structName == "" || dynamicID == "" {
		return nil
	}
	dynamicMethod := fixedJavaDynamicTypeDeclWithTypeParams(structName, dynamicID, typeParams, ctx)
	registration := syntheticReferenceRegistrationDecl(structName, dynamicID, superID, interfaceIDs, ctx)
	return []ast.Decl{dynamicMethod, registration}
}
