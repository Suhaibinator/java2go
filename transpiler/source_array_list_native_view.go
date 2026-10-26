package transpiler

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func sourceArrayListNativeSourceContext(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || ctx.currentFile == nil || ctx.currentClass == nil || ctx.currentFile.ParsedRoot() == nil || !threadSourceNodeWithin(node, ctx.currentFile.ParsedRoot()) || node.EndByte() > uint32(len(ctx.currentFile.Source)) {
		return false
	}
	// The parser snapshot owns the bytes. Equal offsets or a copied caller slice
	// cannot certify that a token belongs to this source occurrence.
	if len(source) == 0 || len(source) != len(ctx.currentFile.Source) || &source[0] != &ctx.currentFile.Source[0] {
		return false
	}
	snapshot := ctx.currentFile.SourceInputSnapshot()
	if snapshot == nil || snapshot.Validate() != nil {
		return false
	}
	registered := false
	for _, scope := range allSourceClassScopes() {
		if scope == ctx.currentClass {
			registered = true
			break
		}
	}
	if !registered {
		return false
	}
	declaring, owned := supplierSourceDeclarationCtx(ctx.currentClass, ctx)
	if !owned || declaring.currentFile != ctx.currentFile {
		return false
	}
	executable := false
	fieldInitializer := false
	for current := node; current != nil; current = current.Parent() {
		switch current.Type() {
		case "lambda_expression":
			return false
		case "method_declaration", "constructor_declaration", "compact_constructor_declaration":
			if executable || ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || !current.Equal(ctx.localScope.DeclarationNode) || !threadSourceNodeWithin(ctx.localScope.DeclarationNode, ctx.currentFile.ParsedRoot()) {
				return false
			}
			attached := 0
			for _, method := range ctx.currentClass.Methods {
				if method == ctx.localScope {
					attached++
				}
			}
			if attached != 1 {
				return false
			}
			executable = true
		case "field_declaration":
			fieldInitializer = sourceArrayListNativeFieldInitializerContext(node, current, ctx, source)
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			return current.Equal(ctx.currentClass.Class.DeclarationNode) && (executable || fieldInitializer)
		case "object_creation_expression":
			if objectCreationClassBody(current) != nil {
				return false
			}
		}
	}
	return false
}

// Field initializers are emitted through two existing synthetic producers:
// static initialization uses only IsStatic; the instance initializer uses only
// OriginalName. Those names are never source-method authority: the actual
// field/value ancestry and owned static metadata must agree as well.
func sourceArrayListNativeFieldInitializerContext(node, declaration *sitter.Node, ctx Ctx, source []byte) bool {
	if ctx.localScope == nil || declaration == nil || ctx.currentClass == nil {
		return false
	}
	body := ctx.currentClass.Class.DeclarationNode.ChildByFieldName("body")
	if declaration.Parent() == nil || body == nil || !declaration.Parent().Equal(body) {
		return false
	}
	var variable *sitter.Node
	for current := node; current != nil && !current.Equal(declaration); current = current.Parent() {
		if current.Type() == "variable_declarator" && current.Parent() != nil && current.Parent().Equal(declaration) {
			variable = current
			break
		}
	}
	if variable == nil || variable.ChildByFieldName("value") == nil || !threadSourceNodeWithin(node, variable.ChildByFieldName("value")) {
		return false
	}
	name, typ := variable.ChildByFieldName("name"), declaration.ChildByFieldName("type")
	if name == nil || typ == nil {
		return false
	}
	var actual *symbol.Definition
	for _, field := range ctx.currentClass.Fields {
		if field != nil && field.OriginalName == name.Content(source) && field.DeclarationNode != nil && field.DeclarationNode.Equal(declaration) && field.OriginalType == typ.Content(source) {
			if actual != nil {
				return false
			}
			actual = field
		}
	}
	if actual == nil || actual.IsStatic != declarationHasModifier(declaration, "static") {
		return false
	}
	expected := &symbol.Definition{OriginalName: fieldInitMethodName}
	if actual.IsStatic {
		expected = &symbol.Definition{IsStatic: true}
	}
	return reflect.DeepEqual(ctx.localScope, expected)
}

// Symbol OriginalType intentionally retains the prefix type spelling; Java
// also permits dimensions after a variable name or method parameter list.
// This scalar transport checks that actual suffix, independently of sidecars.
func sourceArrayListNativeNoDimensions(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if child.Type() == "dimensions" || child.Type() == "dimensions_expr" {
			return false
		}
	}
	return true
}

func sourceArrayListNativeDeclaredScalar(def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if def == nil || owner == nil {
		return false
	}
	declaring, owned := supplierSourceDeclarationCtx(owner, ctx)
	if !owned || declaring.currentFile.SourceInputSnapshot() == nil || declaring.currentFile.SourceInputSnapshot().Validate() != nil {
		return false
	}
	declaration := def.DeclarationNode
	if declaration == nil || !threadSourceNodeWithin(declaration, owner.Class.DeclarationNode) {
		return false
	}
	typ := declaration.ChildByFieldName("type")
	if typ == nil || typ.Content(declaring.currentFile.Source) != def.OriginalType || !sourceArrayListNativeNoDimensions(declaration) {
		return false
	}
	switch declaration.Type() {
	case "method_declaration":
		name := declaration.ChildByFieldName("name")
		if name == nil || name.Content(declaring.currentFile.Source) != def.OriginalName {
			return false
		}
		count := 0
		for _, method := range owner.Methods {
			if method == def {
				count++
			}
		}
		return count == 1
	case "field_declaration":
		count := 0
		for _, field := range owner.Fields {
			if field == def {
				count++
			}
		}
		if count != 1 {
			return false
		}
		found := 0
		for _, variable := range nodeutil.VariableDeclarators(declaration) {
			name := variable.ChildByFieldName("name")
			if name != nil && name.Content(declaring.currentFile.Source) == def.OriginalName {
				if !sourceArrayListNativeNoDimensions(variable) {
					return false
				}
				found++
			}
		}
		return found == 1
	}
	return false
}

// Locals and parameters have no DeclarationNode in the symbol model. Require
// their unique actual declaration in this executable, with identical metadata;
// never borrow a same-named declaration from a nested class or lambda.
func sourceArrayListNativeLocalScalar(name string, ctx Ctx, source []byte) bool {
	if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil {
		return false
	}
	def := ctx.localScope.FindVariable(name)
	if def == nil {
		return false
	}
	found := 0
	valid := true
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if !node.Equal(ctx.localScope.DeclarationNode) {
			switch node.Type() {
			case "lambda_expression", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "method_declaration", "constructor_declaration":
				return
			case "object_creation_expression":
				if objectCreationClassBody(node) != nil {
					return
				}
			}
		}
		var typ, identifier *sitter.Node
		switch node.Type() {
		case "formal_parameter", "spread_parameter":
			typ, identifier = nodeutil.JavaParameterNodes(node)
		case "variable_declarator":
			if parent := node.Parent(); parent != nil && parent.Type() == "local_variable_declaration" {
				typ, identifier = parent.ChildByFieldName("type"), node.ChildByFieldName("name")
			}
		}
		if identifier != nil && identifier.Content(source) == name {
			found++
			if typ == nil || typ.Content(source) != def.OriginalType || !sourceArrayListNativeNoDimensions(node) || node.Type() == "spread_parameter" {
				valid = false
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(ctx.localScope.DeclarationNode)
	return valid && found == 1
}

// Inference supplies origin substitution, while the selected declaration below
// supplies scalar rank. Neither a spelling nor OriginalType alone proves rank.
func sourceArrayListNativeScalarExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "parenthesized_expression" {
		return sourceArrayListNativeScalarExpression(nodeutil.SemanticNamedChild(node, 0), ctx, source)
	}
	switch node.Type() {
	case "this", "super":
		return true
	case "identifier":
		name := node.Content(source)
		if ctx.localScope != nil && ctx.localScope.FindVariable(name) != nil {
			return sourceArrayListNativeLocalScalar(name, ctx, source)
		}
		if field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx); field != nil {
			return sourceArrayListNativeDeclaredScalar(field.def, field.owner, ctx)
		}
		if field := resolveUnqualifiedStaticField(name, ctx); field != nil {
			return sourceArrayListNativeDeclaredScalar(field.def, field.owner, ctx)
		}
		if imported := resolveStaticImportedField(name, ctx); imported.problem == "" && imported.source != nil {
			return sourceArrayListNativeDeclaredScalar(imported.source.def, imported.source.owner, ctx)
		}
	case "field_access":
		if access, ok := resolveStaticFieldAccess(node, source, ctx); ok {
			return sourceArrayListNativeDeclaredScalar(access.resolution.def, access.resolution.owner, ctx)
		}
		object, field := node.ChildByFieldName("object"), node.ChildByFieldName("field")
		if object == nil || field == nil {
			return false
		}
		owner := ctx.currentClass
		switch object.Type() {
		case "this":
		case "super":
			owner = resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass)
		default:
			if !sourceArrayListNativeScalarExpression(object, ctx, source) {
				return false
			}
			target := resolveInvocationTarget(object, ctx, source)
			if target == nil {
				return false
			}
			owner = target.classScope
		}
		if selected := findFieldResolutionInHierarchy(owner, field.Content(source), ctx); selected != nil {
			return sourceArrayListNativeDeclaredScalar(selected.def, selected.owner, ctx)
		}
	case "method_invocation":
		name := node.ChildByFieldName("name")
		if name == nil {
			return false
		}
		var selected *methodResolution
		if object := node.ChildByFieldName("object"); object != nil {
			if owner := resolveClassScopeByIdentifier(ctx, source, object); owner != nil {
				selected = findBestMethodInHierarchy(owner, name.Content(source), node.ChildByFieldName("arguments"), false, true, ctx, source)
			} else {
				if !sourceArrayListNativeScalarExpression(object, ctx, source) {
					return false
				}
				selected, _ = findBestMethodForInvocationTarget(resolveInvocationTarget(object, ctx, source), name.Content(source), node.ChildByFieldName("arguments"), true, true, ctx, source)
			}
		} else {
			selected = findBestMethodInHierarchy(ctx.currentClass, name.Content(source), node.ChildByFieldName("arguments"), ctx.localScope == nil || !ctx.localScope.IsStatic, true, ctx, source)
			if selected == nil {
				selected = findEnclosingStaticMethod(name.Content(source), node.ChildByFieldName("arguments"), ctx, source)
			}
			if selected == nil {
				selected = resolveStaticImportedMethod(node, ctx, source).source
			}
		}
		return selected != nil && sourceArrayListNativeDeclaredScalar(selected.def, selected.owner, ctx)
	case "object_creation_expression":
		return objectCreationClassBody(node) == nil && node.ChildByFieldName("type") != nil
	}
	return false
}

// Boundary authority comes from the written target declaration or the actual
// assignment target's origin, not its spelling re-resolved in the RHS caller.
// Formal arguments and casts have separate declaration transport requirements.
func sourceArrayListNativeBoundaryOrigin(node *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	if !sourceArrayListNativeSourceContext(node, ctx, source) {
		return inferredJavaTypeOrigin{}, false
	}
	boundary := node
	for boundary.Parent() != nil && boundary.Parent().Type() == "parenthesized_expression" {
		boundary = boundary.Parent()
	}
	parent := boundary.Parent()
	if parent == nil {
		return inferredJavaTypeOrigin{}, false
	}
	clean := ctx.Clone()
	clean.expectedType = ""
	clean.expectedTypeRoot = nil
	switch parent.Type() {
	case "variable_declarator":
		if !sameSourceNode(parent.ChildByFieldName("value"), boundary) {
			return inferredJavaTypeOrigin{}, false
		}
		declaration := parent.Parent()
		if declaration == nil {
			return inferredJavaTypeOrigin{}, false
		}
		typ := declaration.ChildByFieldName("type")
		if typ == nil || !sourceArrayListNativeNoDimensions(parent) {
			return inferredJavaTypeOrigin{}, false
		}
		if declaration.Type() == "local_variable_declaration" {
			if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || !threadSourceNodeWithin(declaration, ctx.localScope.DeclarationNode) {
				return inferredJavaTypeOrigin{}, false
			}
			attached := 0
			for _, method := range ctx.currentClass.Methods {
				if method == ctx.localScope {
					attached++
				}
			}
			if attached != 1 {
				return inferredJavaTypeOrigin{}, false
			}
			return declaredJavaTypeOrigin(symbol.JavaType{Original: typ.Content(source), TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, clean), true
		}
		if declaration.Type() == "field_declaration" {
			name := parent.ChildByFieldName("name")
			if name == nil {
				return inferredJavaTypeOrigin{}, false
			}
			declaring, _ := supplierSourceDeclarationCtx(ctx.currentClass, ctx)
			var field *symbol.Definition
			for _, candidate := range ctx.currentClass.Fields {
				if candidate != nil && candidate.OriginalName == name.Content(source) && candidate.DeclarationNode != nil && candidate.DeclarationNode.Equal(declaration) && candidate.OriginalType == typ.Content(source) {
					if field != nil {
						return inferredJavaTypeOrigin{}, false
					}
					field = candidate
				}
			}
			if field != nil {
				return declaredJavaTypeOrigin(definitionDeclaredJavaType(field), declaring), true
			}
		}
	case "assignment_expression":
		left, operator, right, valid := assignmentExpressionNodes(parent, source)
		if !valid || operator.Content(source) != "=" || !sameSourceNode(right, boundary) || !sourceArrayListNativeScalarExpression(left, clean, source) {
			return inferredJavaTypeOrigin{}, false
		}
		var origin inferredJavaTypeOrigin
		_, known := inferExprJavaType(left, clean, source, &origin)
		return origin, known
	case "return_statement":
		method := ctx.localScope
		if method == nil || method.DeclarationNode == nil || !threadSourceNodeWithin(parent, method.DeclarationNode) {
			return inferredJavaTypeOrigin{}, false
		}
		for current := parent.Parent(); current != nil && !current.Equal(method.DeclarationNode); current = current.Parent() {
			if current.Type() == "lambda_expression" || current.Type() == "class_declaration" {
				return inferredJavaTypeOrigin{}, false
			}
		}
		attached := 0
		for _, actual := range ctx.currentClass.Methods {
			if actual == method {
				attached++
			}
		}
		typ := method.DeclarationNode.ChildByFieldName("type")
		if attached != 1 || typ == nil || typ.Content(source) != method.OriginalType || !sourceArrayListNativeNoDimensions(method.DeclarationNode) {
			return inferredJavaTypeOrigin{}, false
		}
		declaring, _ := supplierSourceDeclarationCtx(ctx.currentClass, ctx)
		declaring.localScope = method
		return declaredJavaTypeOrigin(definitionDeclaredJavaType(method), declaring), true
	}
	return inferredJavaTypeOrigin{}, false
}

func sourceArrayListNativeRole(origin inferredJavaTypeOrigin) bool {
	return !origin.unresolved && !origin.wildcard && origin.parameter == nil && origin.nominalScope == nil && origin.arrayRank == 0 && (origin.externalName == "java.util.ArrayList" || origin.externalName == "java.util.List") && len(origin.arguments) <= 1
}
func sourceArrayListNativeScalarOrigin(origin inferredJavaTypeOrigin) bool {
	return !origin.unresolved && !origin.wildcard && origin.arrayRank == 0 && len(origin.arguments) == 0 && (origin.parameter != nil || origin.externalName != "" || origin.nominalScope != nil && len(origin.nominalScope.TypeParameters) == 0)
}

func sourceArrayListNativeElement(origin inferredJavaTypeOrigin, ctx Ctx) (inferredJavaTypeOrigin, bool) {
	if origin.unresolved || origin.wildcard || origin.parameter != nil || origin.nominalScope == nil || origin.arrayRank != 0 {
		return inferredJavaTypeOrigin{}, false
	}
	root := sourceArrayListRoot(origin.nominalScope, ctx)
	if root == nil {
		return inferredJavaTypeOrigin{}, false
	}
	if scalar := sourceArrayListScalarOwner(root, ctx); scalar != nil {
		if scalar.parameter == nil {
			return inferredJavaTypeOrigin{externalName: "java.lang.String", javaType: "java.lang.String"}, true
		}
		bindings := sourceClassOriginBindings(origin.nominalScope, origin.arguments, root, ctx)
		element, known := bindings[scalar.parameter]
		return element, known && sourceArrayListNativeScalarOrigin(element)
	}
	// P1 roots and their admitted descendants have one canonical Object slot.
	return inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}, true
}

func sourceArrayListNativeElementType(origin inferredJavaTypeOrigin, ctx Ctx) (ast.Expr, bool) {
	if !sourceArrayListNativeScalarOrigin(origin) {
		return nil, false
	}
	name := origin.externalName
	if origin.nominalScope != nil {
		name = javaInferenceTypeName(origin.nominalScope)
	}
	if origin.parameter != nil {
		name = origin.parameter.GoName
		if name == "" {
			name = origin.parameter.SourceName
		}
		if visibleTypeParameterDeclarationForJavaType(name, ctx) != origin.parameter {
			return nil, false
		}
	}
	if strings.TrimSpace(name) == "" {
		return nil, false
	}
	render := ctx.Clone()
	render.sourceNativeVirtualBody = nil
	return javaTypeStringToGoTypeExpr(name, inScopeTypeParameters(render), render), true
}

func sourceArrayListNativeBoundaryConversion(value ast.Expr, node *sitter.Node, expected string, ctx Ctx, source []byte) (ast.Expr, bool) {
	if value == nil || strings.TrimSpace(expected) == "" {
		return nil, false
	}
	target, owned := sourceArrayListNativeBoundaryOrigin(node, ctx, source)
	if !owned || !sourceArrayListNativeRole(target) || !sourceArrayListNativeScalarExpression(node, ctx, source) {
		return nil, false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(node, inference, source, &origin)
	if !known {
		return nil, false
	}
	element, known := sourceArrayListNativeElement(origin, ctx)
	if !known {
		return nil, false
	}
	wanted := inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}
	if len(target.arguments) == 1 {
		wanted = target.arguments[0]
	}
	if !sourceArrayListNativeScalarOrigin(wanted) || !sameJavaTypeOrigin(element, wanted) {
		return nil, false
	}
	physical, known := sourceArrayListNativeElementType(wanted, ctx)
	if !known {
		return nil, false
	}
	native := &ast.StarExpr{X: applyTypeArguments(stdjavaQualifiedExpr("List", ctx), []ast.Expr{physical})}
	// ObjectView asks the source provider for this exact typed native parent.
	// It evaluates value once, preserves Java null, and never copies storage.
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{native}, []ast.Expr{value, javaTypeIDLiteral(target.externalName, ctx)}), true
}

// A source object and its invariant native parent are distinct Go pointers for
// one Java allocation. Certify both declaration-owned scalar origins before
// choosing JavaReferenceEqual; its existing ObjectInfo identity is shared by
// the parent and the most-derived source object. This query evaluates neither
// operand and introduces no broader native or generic equality family.
func sourceArrayListNativeEquality(left, right *sitter.Node, ctx Ctx, source []byte) bool {
	if !sourceArrayListNativeSourceContext(left, ctx, source) || !sourceArrayListNativeSourceContext(right, ctx, source) {
		return false
	}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	operand := func(node *sitter.Node) (inferredJavaTypeOrigin, *symbol.ClassScope, bool) {
		if !sourceArrayListNativeScalarExpression(node, inference, source) {
			return inferredJavaTypeOrigin{}, nil, false
		}
		var origin inferredJavaTypeOrigin
		if _, known := inferExprJavaType(node, inference, source, &origin); !known {
			return inferredJavaTypeOrigin{}, nil, false
		}
		if element, known := sourceArrayListNativeElement(origin, inference); known {
			return element, origin.nominalScope, true
		}
		if !sourceArrayListNativeRole(origin) {
			return inferredJavaTypeOrigin{}, nil, false
		}
		element := inferredJavaTypeOrigin{externalName: "java.lang.Object", javaType: "java.lang.Object"}
		if len(origin.arguments) == 1 {
			element = origin.arguments[0]
		}
		return element, nil, sourceArrayListNativeScalarOrigin(element)
	}
	leftElement, leftOwner, leftKnown := operand(left)
	rightElement, rightOwner, rightKnown := operand(right)
	if !leftKnown || !rightKnown || leftOwner == nil && rightOwner == nil || !sameJavaTypeOrigin(leftElement, rightElement) {
		return false
	}
	if leftOwner != nil && rightOwner != nil && leftOwner != rightOwner && !javaReferenceTypeAssignable(leftOwner, rightOwner, inference) && !javaReferenceTypeAssignable(rightOwner, leftOwner, inference) {
		return false
	}
	return true
}
