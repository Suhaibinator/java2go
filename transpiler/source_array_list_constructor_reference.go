package transpiler

import (
	"go/ast"
	"maps"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A constructor reference contributes a lazy function value, not a constructed
// object. This certificate is bounded to the already supported scalar source
// ArrayList ABI and a declaration-owned, invariant Supplier<C> target.
func lowerSourceArrayListConstructorReference(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	if !sourceArrayListNativeSourceContext(node, ctx, source) || !sourceArrayListConstructorFileNamespace(ctx.currentFile) || node.Type() != "method_reference" || node.NamedChildCount() != 1 || node.ChildCount() == 0 || node.Child(int(node.ChildCount())-1).Type() != "new" {
		return nil, false
	}
	// SourceContext establishes a body/initializer occurrence. A caller's
	// class-header lookup flag cannot erase inherited member TYPE declarations.
	ctx = ctx.Clone()
	ctx.memberTypeHeaderOwner = nil
	ctx.localClasses = nil
	ctx.anonymousClasses = nil
	target := node.NamedChild(0)
	if target.Type() != "generic_type" || !sourceArrayListConstructorTypeNamespace(target, source) {
		return nil, false
	}
	clean := ctx.Clone()
	clean.expectedType = ""
	clean.expectedTypeRoot = nil
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	result := declaredJavaTypeOrigin(symbol.JavaType{Original: target.Content(source), TypeParameterBindings: bindings}, clean)
	owner := result.nominalScope
	if result.unresolved || result.wildcard || result.parameter != nil || result.arrayRank != 0 || owner == nil || owner.Class == nil || owner.IsAbstract || owner.IsInterface || owner.IsEnum || owner.IsInner || len(owner.TypeParameters) != 1 || len(result.arguments) != 1 || !sourceArrayListNativeScalarOrigin(result.arguments[0]) {
		return nil, false
	}
	declaring, attached := supplierSourceDeclarationCtx(owner, clean)
	if !attached || !sourceArrayListConstructorFileNamespace(declaring.currentFile) {
		return nil, false
	}
	// A descendant's separate constructor/outer ABI is not proved by the root.
	if sourceArrayListRoot(owner, clean) != owner {
		return nil, false
	}
	scalar := sourceArrayListScalarOwner(owner, clean)
	if scalar == nil || scalar.parameter == nil || scalar.parameter != owner.TypeParameters[0].Declaration {
		return nil, false
	}
	declaration := owner.Class.DeclarationNode
	if declaration.Type() != "class_declaration" || declaration.Parent() == nil || (declaration.Parent().Type() != "program" && declaration.Parent().Type() != "class_body") {
		return nil, false
	}
	wanted, known := sourceArrayListConstructorSupplierResult(node, ctx, source)
	if !known || !sameJavaTypeOrigin(wanted, result) {
		return nil, false
	}
	// The actual boundary is authority; an inconsistent renderer sidecar cannot
	// replace its declared C. The canonical spelling retains binder identity.
	if strings.TrimSpace(ctx.expectedType) != "" {
		expected := declaredJavaTypeOrigin(symbol.JavaType{Original: ctx.expectedType, TypeParameterBindings: bindings}, clean)
		if expected.unresolved || expected.wildcard || expected.parameter != nil || expected.nominalScope != nil || expected.arrayRank != 0 || expected.externalName != "java.util.function.Supplier" || len(expected.arguments) != 1 || !sameJavaTypeOrigin(expected.arguments[0], result) {
			return nil, false
		}
	}
	name, known := sourceArrayListConstructorFunction(owner)
	physical, rendered := deferredToCollectionResultType(result, clean)
	element, elementKnown := sourceArrayListNativeElementType(result.arguments[0], clean)
	if !known || !rendered || !elementKnown {
		return nil, false
	}
	function := applyTypeArguments(qualifiedNameExpr(name, findJavaPackageForClassScope(owner), ctx), []ast.Expr{element})
	return stdjavaGenericCall(ctx, "NewSupplierFuncAdapter", []ast.Expr{physical}, []ast.Expr{function}), true
}

// Check lexical TYPE prefixes before any spelling is re-resolved in a cleaned
// context. A scoped type is atomic: its first segment owns the lookup.
func sourceArrayListConstructorTypeNamespace(typ *sitter.Node, source []byte) bool {
	if typ == nil {
		return false
	}
	switch typ.Type() {
	case "type_identifier", "scoped_type_identifier":
		return !sourceGenericViewLocalTypeShadowed(typ, typ.Content(source), source)
	case "generic_type":
		found := false
		for _, child := range nodeutil.NamedChildrenOf(typ) {
			if child.Type() == "type_arguments" {
				for _, argument := range nodeutil.NamedChildrenOf(child) {
					if !sourceArrayListConstructorTypeNamespace(argument, source) {
						return false
					}
				}
			} else {
				if found || !sourceArrayListConstructorTypeNamespace(child, source) {
					return false
				}
				found = true
			}
		}
		return found
	}
	return false
}

func sourceArrayListConstructorSupplierResult(node *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	var empty inferredJavaTypeOrigin
	if _, attached := sourceArrayListNativeBoundaryOrigin(node, ctx, source); attached {
		typ, bytes := sourceArrayListConstructorBoundaryType(node, ctx, source)
		boundary := node
		for boundary.Parent() != nil && boundary.Parent().Type() == "parenthesized_expression" {
			boundary = boundary.Parent()
		}
		var operand *sitter.Node
		if parent := boundary.Parent(); parent != nil && parent.Type() == "assignment_expression" {
			operand, _, _, _ = assignmentExpressionNodes(parent, source)
		}
		target, known := sourceArrayListConstructorActualTypeOrigin(typ, bytes, operand, ctx, source)
		if !known || target.unresolved || target.wildcard || target.parameter != nil || target.nominalScope != nil || target.arrayRank != 0 || target.externalName != "java.util.function.Supplier" || len(target.arguments) != 1 {
			return empty, false
		}
		return target.arguments[0], true
	}
	// A direct canonical collector owns its JDK Supplier<C> target. Querying this
	// origin certificate never lowers the factory or recursively calls ParseExpr.
	arguments := node.Parent()
	if arguments == nil || arguments.Type() != "argument_list" || arguments.NamedChildCount() != 1 || !arguments.NamedChild(0).Equal(node) {
		return empty, false
	}
	collector := arguments.Parent()
	if collector == nil || collector.Type() != "method_invocation" {
		return empty, false
	}
	object := collector.ChildByFieldName("object")
	if object == nil || sourceGenericViewLocalTypeShadowed(object, object.Content(source), source) {
		return empty, false
	}
	proof, known := deferredToCollectionCertificate(collector, ctx, source)
	if !known || proof.factory == nil || !proof.factory.Equal(node) {
		return empty, false
	}
	receiver := collector.Parent().Parent().ChildByFieldName("object")
	if _, _, ownedFactory := explicitStreamFactoryElement(receiver, ctx, source); !ownedFactory {
		typ, bytes := sourceArrayListConstructorOperandType(receiver, ctx, source)
		actual, owned := sourceArrayListConstructorActualTypeOrigin(typ, bytes, receiver, ctx, source)
		if !owned || actual.unresolved || actual.wildcard || actual.parameter != nil || actual.nominalScope != nil || actual.arrayRank != 0 || actual.externalName != "java.util.stream.Stream" || len(actual.arguments) != 1 || !sameJavaTypeOrigin(actual.arguments[0], proof.element) {
			return empty, false
		}
	}
	return proof.result, true
}

// Locate the actual boundary TYPE for reject-only local namespace checks. Its
// lexical location, rather than the RHS location, controls visible local types.
func sourceArrayListConstructorBoundaryType(node *sitter.Node, ctx Ctx, source []byte) (*sitter.Node, []byte) {
	boundary := node
	for boundary.Parent() != nil && boundary.Parent().Type() == "parenthesized_expression" {
		boundary = boundary.Parent()
	}
	parent := boundary.Parent()
	if parent == nil {
		return nil, nil
	}
	switch parent.Type() {
	case "variable_declarator":
		return parent.Parent().ChildByFieldName("type"), source
	case "return_statement":
		if ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
			return ctx.localScope.DeclarationNode.ChildByFieldName("type"), source
		}
	case "assignment_expression":
		left, _, _, known := assignmentExpressionNodes(parent, source)
		if known {
			return sourceArrayListConstructorOperandType(left, ctx, source)
		}
	}
	return nil, nil
}

func sourceArrayListConstructorOperandType(node *sitter.Node, ctx Ctx, source []byte) (*sitter.Node, []byte) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return nil, nil
	}
	var definition *symbol.Definition
	var owner *symbol.ClassScope
	if node.Type() == "identifier" {
		name := node.Content(source)
		if ctx.localScope != nil && ctx.localScope.FindVariable(name) != nil {
			if !sourceArrayListNativeLocalScalar(name, ctx, source) {
				return nil, nil
			}
			var typ *sitter.Node
			found := 0
			var walk func(*sitter.Node)
			walk = func(current *sitter.Node) {
				if current == nil {
					return
				}
				if !current.Equal(ctx.localScope.DeclarationNode) {
					switch current.Type() {
					case "lambda_expression", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "method_declaration", "constructor_declaration":
						return
					case "object_creation_expression":
						if objectCreationClassBody(current) != nil {
							return
						}
					}
				}
				var candidate, identifier *sitter.Node
				switch current.Type() {
				case "formal_parameter":
					candidate, identifier = nodeutil.JavaParameterNodes(current)
				case "variable_declarator":
					if parent := current.Parent(); parent != nil && parent.Type() == "local_variable_declaration" {
						candidate, identifier = parent.ChildByFieldName("type"), current.ChildByFieldName("name")
					}
				}
				if identifier != nil && identifier.Content(source) == name {
					found++
					typ = candidate
				}
				for _, child := range nodeutil.NamedChildrenOf(current) {
					walk(child)
				}
			}
			walk(ctx.localScope.DeclarationNode)
			if found != 1 {
				return nil, nil
			}
			return typ, source
		}
		if field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx); field != nil {
			definition, owner = field.def, field.owner
		} else if field := resolveUnqualifiedStaticField(name, ctx); field != nil {
			definition, owner = field.def, field.owner
		} else if imported := resolveStaticImportedField(name, ctx); imported.problem == "" && imported.source != nil {
			definition, owner = imported.source.def, imported.source.owner
		}
	} else if node.Type() == "field_access" {
		if access, known := resolveStaticFieldAccess(node, source, ctx); known {
			definition, owner = access.resolution.def, access.resolution.owner
		} else {
			object, field := node.ChildByFieldName("object"), node.ChildByFieldName("field")
			if object == nil || field == nil {
				return nil, nil
			}
			selectedOwner := ctx.currentClass
			switch object.Type() {
			case "this":
			case "super":
				selectedOwner = resolveSuperclassScopeInDeclaringContext(ctx, ctx.currentClass)
			default:
				target := resolveInvocationTarget(object, ctx, source)
				if target == nil {
					return nil, nil
				}
				selectedOwner = target.classScope
			}
			if selected := findFieldResolutionInHierarchy(selectedOwner, field.Content(source), ctx); selected != nil {
				definition, owner = selected.def, selected.owner
			}
		}
	}
	if node.Type() == "method_invocation" {
		name := node.ChildByFieldName("name")
		if name == nil {
			return nil, nil
		}
		var selected *methodResolution
		if object := node.ChildByFieldName("object"); object != nil {
			if scope := resolveClassScopeByIdentifier(ctx, source, object); scope != nil {
				selected = findBestMethodInHierarchy(scope, name.Content(source), node.ChildByFieldName("arguments"), false, true, ctx, source)
			} else {
				if !sourceArrayListNativeScalarExpression(object, ctx, source) {
					return nil, nil
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
		if selected != nil {
			definition, owner = selected.def, selected.owner
		}
	}
	if definition == nil || owner == nil || !sourceArrayListNativeDeclaredScalar(definition, owner, ctx) {
		return nil, nil
	}
	file := findFileScopeForClassScope(owner, ctx)
	if file == nil {
		return nil, nil
	}
	return definition.DeclarationNode.ChildByFieldName("type"), file.Source
}

// Re-derive the TYPE in its actual declaration's member/body namespace. The
// older boundary helper supplies attachment/rank, but intentionally uses header
// context for a return. That header lookup cannot certify a Supplier body role.
func sourceArrayListConstructorActualTypeOrigin(typ *sitter.Node, bytes []byte, operand *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	var empty inferredJavaTypeOrigin
	if typ == nil {
		return empty, false
	}
	var declaration, executable *sitter.Node
	for current := typ.Parent(); current != nil; current = current.Parent() {
		switch current.Type() {
		case "lambda_expression":
			return empty, false
		case "method_declaration", "constructor_declaration":
			executable = current
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			declaration = current
		}
		if declaration != nil {
			break
		}
	}
	if declaration == nil {
		return empty, false
	}
	var owner *symbol.ClassScope
	for _, candidate := range allSourceClassScopes() {
		if candidate != nil && candidate.Class != nil && candidate.Class.DeclarationNode != nil && candidate.Class.DeclarationNode.Equal(declaration) {
			if owner != nil {
				return empty, false
			}
			owner = candidate
		}
	}
	declaring, attached := supplierSourceDeclarationCtx(owner, ctx)
	if !attached || !sourceArrayListConstructorFileNamespace(declaring.currentFile) {
		return empty, false
	}
	file := declaring.currentFile
	if len(bytes) == 0 || len(bytes) != len(file.Source) || &bytes[0] != &file.Source[0] || typ.EndByte() > uint32(len(bytes)) || !threadSourceNodeWithin(typ, file.ParsedRoot()) || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil {
		return empty, false
	}
	if !sourceArrayListConstructorTypeNamespace(typ, bytes) {
		return empty, false
	}
	declaring = classScopeCtx(owner, ctx)
	declaring.localClasses = nil
	declaring.anonymousClasses = nil
	declaring.syntheticTypeParameters = nil
	if executable != nil {
		attached := 0
		for _, method := range owner.Methods {
			if method != nil && method.DeclarationNode != nil && method.DeclarationNode.Equal(executable) {
				declaring.localScope = method
				attached++
			}
		}
		if attached != 1 {
			return empty, false
		}
	}
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: typ.Content(bytes), TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring))}, declaring)
	start, arguments := ctx.currentClass, classParameterTypeOrigins(ctx.currentClass)
	if operand != nil {
		object := operand.ChildByFieldName("object")
		if object != nil && object.Type() != "this" && object.Type() != "super" {
			if scope := resolveClassScopeByIdentifier(ctx, source, object); scope != nil {
				start, arguments = scope, classParameterTypeOrigins(scope)
			} else {
				var receiver inferredJavaTypeOrigin
				clean := ctx.Clone()
				clean.expectedType = ""
				clean.expectedTypeRoot = nil
				if _, known := inferExprJavaType(object, clean, source, &receiver); !known || receiver.nominalScope == nil || receiver.unresolved || receiver.arrayRank != 0 {
					return empty, false
				}
				start, arguments = receiver.nominalScope, receiver.arguments
			}
		}
	}
	origin = substituteJavaTypeOrigin(origin, sourceClassOriginBindings(start, arguments, owner, ctx))
	return origin, completeJavaTypeOrigin(origin)
}

// Reconstruct only parser-owned package/import namespace facts. FileScope's
// mutable lookup table is useful for resolution, but cannot attest its own
// correspondence to the original source that authorizes this new capability.
func sourceArrayListConstructorFileNamespace(file *symbol.FileScope) bool {
	if file == nil || file.ParsedRoot() == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil {
		return false
	}
	// Legacy wildcard/static import queries start at BaseClass's ancestor.
	// Attest that second entry point against the independently retained root.
	if file.BaseClass == nil || file.BaseClass.Class == nil || file.BaseClass.Class.DeclarationNode == nil {
		return false
	}
	registered, member := false, 0
	for _, candidate := range allSourceClassScopes() {
		if candidate == file.BaseClass {
			registered = true
		}
	}
	for _, candidate := range file.TopLevelClasses {
		if candidate == file.BaseClass {
			member++
		}
	}
	base := file.BaseClass.Class.DeclarationNode
	if !registered || member != 1 || !threadSourceNodeWithin(base, file.ParsedRoot()) {
		return false
	}
	topRoot := base
	for topRoot.Parent() != nil {
		topRoot = topRoot.Parent()
	}
	if !topRoot.Equal(file.ParsedRoot()) {
		return false
	}
	var first *sitter.Node
	for _, node := range nodeutil.NamedChildrenOf(file.ParsedRoot()) {
		switch node.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
			first = node
		}
		if first != nil {
			break
		}
	}
	if first == nil || !base.Equal(first) {
		return false
	}
	imports := make(map[string]string)
	pkg := ""
	for _, node := range nodeutil.NamedChildrenOf(file.ParsedRoot()) {
		switch node.Type() {
		case "package_declaration":
			if node.NamedChildCount() == 0 {
				return false
			}
			pkg = node.NamedChild(0).Content(file.Source)
		case "import_declaration":
			declaration, ok := nodeutil.ReadJavaImportSyntax(node, file.Source)
			if !ok {
				return false
			}
			if declaration.Kind != nodeutil.JavaImportSingleType {
				continue
			}
			imports[declaration.Name] = declaration.Scope
		}
	}
	return pkg == file.Package && maps.Equal(imports, file.Imports)
}

func sourceArrayListConstructorFunction(owner *symbol.ClassScope) (string, bool) {
	// The current scalar source ABI admits only the implicit empty constructor.
	// Explicit zero-argument constructors still require their own body ABI audit;
	// a method-shaped metadata entry cannot donate that missing capability.
	if !constructorHasExecutionImplementation(nil, owner) {
		return "", false
	}
	for _, method := range owner.Methods {
		if method != nil && method.Constructor {
			return "", false
		}
	}
	return executionConstructorImplementationName(defaultConstructorName(owner.Class.Name), owner), true
}
