package transpiler

import (
	"bytes"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// This certificate only refines an already-applicable strict invocation score.
// Its capture remains existential: the readable upper origin is used to check
// a bound, never substituted into the invariant formal or published as a type.
func declaredMethodWildcardStrictWidening(method *symbol.Definition, owner *symbol.ClassScope, arguments []*sitter.Node, actual, formal inferredJavaTypeOrigin, ctx Ctx, source []byte) bool {
	if method == nil || len(arguments) != 1 || len(method.Parameters) != 1 || len(method.TypeParameters) != 1 || method.Parameters[0] == nil || executionParameterIsVariadic(method, 0) || method.IsVariadic || actual.arrayRank != 0 || formal.arrayRank != 0 || actual.wildcard || formal.wildcard || actual.nominalScope != nil || formal.nominalScope != nil || actual.parameter != nil || formal.parameter != nil || actual.unresolved || formal.unresolved || actual.externalName == "" || formal.externalName == "" || actual.externalName == formal.externalName || len(actual.arguments) != 1 || len(formal.arguments) != 1 {
		return false
	}
	parent, proved := projectJDK21GenericParentOrigin(actual, formal.externalName)
	if !proved || len(parent.arguments) != 1 || !sameJavaTypeOrigin(parent.arguments[0], actual.arguments[0]) {
		return false
	}
	declaring, owned := genericMethodOwnerContext(method)
	if !owned || declaring.currentClass != owner || !declaredWildcardSourceMethodOwned(method, declaring) || !genericMethodClassHeaderOwned(method, declaring.currentFile.Source) {
		return false
	}
	declaring.localScope = method
	declaring.syntheticTypeParameters = nil
	parameter := method.TypeParameters[0]
	if parameter.Declaration == nil || parameter.Declaration.SourceName != parameter.Name || visibleTypeParameterDeclarationForJavaType(parameter.Name, declaring) != parameter.Declaration {
		return false
	}
	for scope := owner; scope != nil; scope = scope.Enclosing {
		for _, carried := range scope.TypeParameters {
			if carried.Declaration == parameter.Declaration {
				return false
			}
		}
	}
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring))
	if !declaredWildcardBindingsOwned(method.TypeParameterBindings, bindings) || !declaredWildcardBindingsOwned(method.Parameters[0].TypeParameterBindings, bindings) || method.Parameters[0].TypeParameterBindings[parameter.Name] != parameter.Declaration {
		return false
	}
	formals := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
	if len(formals) != 1 || formals[0].Type() != "formal_parameter" {
		return false
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(formals[0]) {
		if child.Type() == "dimensions" {
			return false
		}
	}
	typ, name := nodeutil.JavaParameterNodes(formals[0])
	if typ == nil || name == nil || typ.Content(declaring.currentFile.Source) != method.Parameters[0].OriginalType || name.Content(declaring.currentFile.Source) != method.Parameters[0].OriginalName || !sameJavaTypeOrigin(formal, declaredJavaTypeOrigin(definitionDeclaredJavaType(method.Parameters[0]), declaring)) {
		return false
	}
	formalArgument := declaredWildcardSingleTypeArgument(typ)
	if formalArgument == nil || !declaredWildcardTypeNamespaceOwned(typ, declaring) {
		return false
	}
	actualType, local := declaredWildcardActualLocalType(arguments[0], method, ctx, source)
	if actualType == nil || local == nil || !sameJavaTypeOrigin(actual, declaredJavaTypeOrigin(definitionDeclaredJavaType(local), ctx)) {
		return false
	}
	actualArgument := declaredWildcardSingleTypeArgument(actualType)
	if actualArgument == nil || !declaredWildcardTypeNamespaceOwned(actualType, ctx) {
		return false
	}
	upper := formalArgument.Type() == "type_identifier" && formalArgument.Content(declaring.currentFile.Source) == parameter.Name && formal.arguments[0].parameter == parameter.Declaration && !formal.arguments[0].wildcard
	lowerNode := declaredWildcardBoundNode(formalArgument, "super")
	lower := lowerNode != nil && lowerNode.Type() == "type_identifier" && lowerNode.Content(declaring.currentFile.Source) == parameter.Name && formal.arguments[0].wildcard && formal.arguments[0].parameter == nil && formal.arguments[0].externalName == "java.lang.Object"
	var witness inferredJavaTypeOrigin
	if upper {
		boundNode := declaredWildcardBoundNode(actualArgument, "extends")
		if boundNode == nil || !actual.arguments[0].wildcard {
			return false
		}
		witness = declaredJavaTypeOrigin(symbol.JavaType{Original: boundNode.Content(source), TypeParameterBindings: local.TypeParameterBindings}, ctx)
		readable := actual.arguments[0]
		readable.wildcard = false
		if !declaredWildcardCanonicalScalar(witness) || !sameJavaTypeOrigin(readable, witness) {
			return false
		}
	} else if lower {
		if actualArgument.Type() != "type_identifier" && actualArgument.Type() != "scoped_type_identifier" {
			return false
		}
		witness = declaredJavaTypeOrigin(symbol.JavaType{Original: actualArgument.Content(source), TypeParameterBindings: local.TypeParameterBindings}, ctx)
		if !declaredWildcardCanonicalScalar(witness) || !sameJavaTypeOrigin(actual.arguments[0], witness) {
			return false
		}
	} else {
		return false
	}
	constraints, applicable := methodInvocationConstraintBindings(method, owner, arguments, false, ctx, source)
	if !applicable || len(constraints) != 1 || constraints[parameter.Name] == "" {
		return false
	}
	inferred := declaredJavaTypeOrigin(symbol.JavaType{Original: constraints[parameter.Name], TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
	if !declaredWildcardCanonicalScalar(inferred) {
		return false
	}
	if upper {
		// Equality with this upper view is a bound witness, not capture equality.
		if !sameJavaTypeOrigin(inferred, witness) {
			return false
		}
	} else {
		_, _, applies, handled := declaredOriginConversion(inferred, witness, ctx)
		if !handled || !applies {
			return false
		}
	}
	for _, parameterNode := range nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("type_parameters")) {
		for _, child := range nodeutil.SemanticNamedChildrenOf(parameterNode) {
			if child.Type() == "type_bound" && !declaredWildcardTypeNamespaceOwned(child, declaring) {
				return false
			}
		}
	}
	for _, bound := range parameter.Bounds {
		target := declaredJavaTypeOrigin(bound, declaring)
		if !declaredWildcardCanonicalScalar(target) {
			return false
		}
		_, _, applies, handled := declaredOriginConversion(inferred, target, declaring)
		if !handled || !applies {
			return false
		}
	}
	return true
}

func declaredWildcardCanonicalScalar(origin inferredJavaTypeOrigin) bool {
	return !origin.unresolved && !origin.wildcard && origin.nominalScope == nil && origin.parameter == nil && origin.externalName != "" && origin.arrayRank == 0 && len(origin.arguments) == 0
}

func declaredWildcardBindingsOwned(actual, expected map[string]*symbol.TypeParamDeclaration) bool {
	if len(actual) != len(expected) {
		return false
	}
	for name, declaration := range expected {
		if declaration == nil || declaration.SourceName != name || actual[name] != declaration {
			return false
		}
	}
	return true
}

func declaredWildcardSingleTypeArgument(typ *sitter.Node) *sitter.Node {
	if typ == nil || typ.Type() != "generic_type" {
		return nil
	}
	children := nodeutil.SemanticNamedChildrenOf(typ)
	if len(children) != 2 || children[1].Type() != "type_arguments" {
		return nil
	}
	arguments := nodeutil.SemanticNamedChildrenOf(children[1])
	if len(arguments) != 1 {
		return nil
	}
	return arguments[0]
}

func declaredWildcardBoundNode(wildcard *sitter.Node, direction string) *sitter.Node {
	if wildcard == nil || wildcard.Type() != "wildcard" {
		return nil
	}
	var bound *sitter.Node
	for _, child := range nodeutil.SemanticNamedChildrenOf(wildcard) {
		// Java's super token is named; it is the direction, never the TYPE bound.
		if child.Type() == "extends" || child.Type() == "super" {
			continue
		}
		if bound != nil || child.Type() != "type_identifier" && child.Type() != "scoped_type_identifier" {
			return nil
		}
		bound = child
	}
	if bound == nil {
		return nil
	}
	found := 0
	for index := 0; index < int(wildcard.ChildCount()); index++ {
		child := wildcard.Child(index)
		if child.Type() == "extends" || child.Type() == "super" {
			if child.Type() != direction {
				return nil
			}
			found++
		}
	}
	if found != 1 {
		return nil
	}
	return bound
}

// Reconcile the lookup table with the retained source namespace. A modified
// import map or foreign same-span tree cannot donate a canonical declaration.
func declaredWildcardFileOwned(file *symbol.FileScope) bool {
	if file == nil {
		return false
	}
	// This existing attachment is input-only. AST roles, namespace and binders
	// are independently checked below; no successful validation is cached.
	input := file.SourceInputSnapshot()
	if input == nil || input.Validate() != nil {
		return false
	}
	if file.ParsedRoot() == nil || file.ParsedRoot().HasError() || file.ParsedRoot().EndByte() > uint32(len(file.Source)) {
		return false
	}
	imports := map[string]string{}
	pkg := ""
	packages := 0
	for _, node := range nodeutil.SemanticNamedChildrenOf(file.ParsedRoot()) {
		switch node.Type() {
		case "package_declaration":
			names := nodeutil.SemanticNamedChildrenOf(node)
			if len(names) != 1 {
				return false
			}
			pkg = names[0].Content(file.Source)
			packages++
		case "import_declaration":
			ignored := false
			for index := 0; index < int(node.ChildCount()); index++ {
				child := node.Child(index)
				ignored = ignored || child.Content(file.Source) == "*" || child.Type() == "static"
			}
			if ignored {
				continue
			}
			names := nodeutil.SemanticNamedChildrenOf(node)
			if len(names) != 1 {
				return false
			}
			name, scope := names[0].ChildByFieldName("name"), names[0].ChildByFieldName("scope")
			if name == nil || scope == nil {
				return false
			}
			if _, duplicate := imports[name.Content(file.Source)]; duplicate {
				return false
			}
			imports[name.Content(file.Source)] = scope.Content(file.Source)
		}
	}
	if packages > 1 || pkg != file.Package || len(imports) != len(file.Imports) {
		return false
	}
	for name, path := range imports {
		if file.Imports[name] != path {
			return false
		}
	}
	return true
}

func declaredWildcardSourceMethodOwned(method *symbol.Definition, ctx Ctx) bool {
	if method == nil || ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentFile == nil || !declaredWildcardFileOwned(ctx.currentFile) || findFileScopeForClassScope(ctx.currentClass, ctx) != ctx.currentFile || !sourceReflectionDeclaredMethod(ctx.currentClass, method) || !threadSourceNodeWithin(ctx.currentClass.Class.DeclarationNode, ctx.currentFile.ParsedRoot()) {
		return false
	}
	className := ctx.currentClass.Class.DeclarationNode.ChildByFieldName("name")
	methodName := method.DeclarationNode.ChildByFieldName("name")
	return declaredWildcardNoWrittenParents(method.DeclarationNode) && className != nil && methodName != nil && className.Content(ctx.currentFile.Source) == ctx.currentClass.Class.OriginalName && methodName.Content(ctx.currentFile.Source) == method.OriginalName
}

// Locals have no independently retained DeclarationNode. Match their unique
// original TYPE occurrence in the registered caller's tree and lexical block.
func declaredWildcardActualLocalType(argument *sitter.Node, method *symbol.Definition, ctx Ctx, source []byte) (*sitter.Node, *symbol.Definition) {
	if argument == nil || argument.Type() != "identifier" || ctx.localScope == nil || ctx.currentFile == nil || len(ctx.syntheticTypeParameters) != 0 || len(ctx.localClasses) != 0 || !bytes.Equal(source, ctx.currentFile.Source) || !declaredWildcardSourceMethodOwned(ctx.localScope, ctx) || !genericMethodClassHeaderOwned(ctx.localScope, source) {
		return nil, nil
	}
	caller, registered := genericMethodOwnerContext(ctx.localScope)
	if !registered || caller.currentClass != ctx.currentClass || caller.currentFile != ctx.currentFile || !threadSourceNodeWithin(argument, ctx.localScope.DeclarationNode) {
		return nil, nil
	}
	list := argument.Parent()
	if list == nil || list.Type() != "argument_list" {
		return nil, nil
	}
	arguments := nodeutil.SemanticNamedChildrenOf(list)
	if len(arguments) != 1 || !arguments[0].Equal(argument) {
		return nil, nil
	}
	call := list.Parent()
	if call == nil || call.Type() != "method_invocation" || call.ChildByFieldName("type_arguments") != nil {
		return nil, nil
	}
	name := call.ChildByFieldName("name")
	if name == nil || name.Content(source) != method.OriginalName {
		return nil, nil
	}
	local := ctx.localScope.FindVariable(argument.Content(source))
	expected := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	if local == nil || local.DirectTypeParameter != nil || !declaredWildcardBindingsOwned(local.TypeParameterBindings, expected) {
		return nil, nil
	}
	var declaration, typ *sitter.Node
	matches := 0
	scalar := true
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		switch node.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "object_creation_expression", "lambda_expression":
			return
		}
		if node.Type() == "local_variable_declaration" {
			for _, declarator := range nodeutil.VariableDeclarators(node) {
				n := declarator.ChildByFieldName("name")
				if n != nil && n.Content(source) == local.OriginalName {
					matches++
					declaration = node
					typ = node.ChildByFieldName("type")
					for _, child := range nodeutil.SemanticNamedChildrenOf(declarator) {
						if child.Type() == "dimensions" {
							scalar = false
						}
					}
				}
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(ctx.localScope.DeclarationNode.ChildByFieldName("body"))
	if matches != 1 || !scalar || typ == nil || declaration.EndByte() > argument.StartByte() || !threadSourceNodeWithin(argument, declaration.Parent()) || typ.Content(source) != local.OriginalType {
		return nil, nil
	}
	return typ, local
}

// Written ancestry may contribute inherited TYPE members independently of a
// mutable ClassScope's parent inventory. This bounded proof does not certify
// such a namespace, including enclosing declarations' parents.
func declaredWildcardNoWrittenParents(occurrence *sitter.Node) bool {
	for ancestor := occurrence.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		if !sourceGenericViewTypeDeclaration(ancestor) {
			continue
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(ancestor) {
			switch child.Type() {
			case "superclass", "super_interfaces", "interfaces", "extends_interfaces":
				return false
			}
		}
		for _, field := range []string{"superclass", "interfaces", "extends_interfaces"} {
			if ancestor.ChildByFieldName(field) != nil {
				return false
			}
		}
	}
	return true
}

// Each original TYPE occurrence supplies its own lexical namespace. Imported
// canonical names cannot be donated through a local/member shadow or an
// unresolved demand/static TYPE import; a disjoint static value import is inert.
func declaredWildcardTypeNamespaceOwned(typ *sitter.Node, ctx Ctx) bool {
	if typ == nil || ctx.currentFile == nil || !threadSourceNodeWithin(typ, ctx.currentFile.ParsedRoot()) {
		return false
	}
	source := ctx.currentFile.Source
	if typ.Type() == "type_identifier" || typ.Type() == "scoped_type_identifier" {
		base := typ.Content(source)
		prefix := strings.SplitN(base, ".", 2)[0]
		if sourceGenericViewLocalTypeShadowed(typ, base, source) {
			return false
		}
		lookup := ctx.Clone()
		lookup.localClasses = nil
		lookup.syntheticTypeParameters = nil
		if classServiceASTTypePrefix(typ, prefix, source, lookup) {
			return false
		}
		// The formal method binder is certified separately from nominal prefixes.
		if visibleTypeParameterDeclarationForJavaType(base, lookup) == nil && classServiceSourceTypePrefix(prefix, lookup) {
			return false
		}
		for _, node := range nodeutil.SemanticNamedChildrenOf(ctx.currentFile.ParsedRoot()) {
			if node.Type() != "import_declaration" {
				continue
			}
			static, demand := false, false
			for index := 0; index < int(node.ChildCount()); index++ {
				child := node.Child(index)
				static = static || child.Type() == "static"
				demand = demand || child.Content(source) == "*"
			}
			if demand {
				return false
			}
			if static {
				names := nodeutil.SemanticNamedChildrenOf(node)
				if len(names) != 1 {
					return false
				}
				name := names[0].ChildByFieldName("name")
				if name == nil || name.Content(source) == prefix {
					return false
				}
			}
		}
		return true
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(typ) {
		if !declaredWildcardTypeNamespaceOwned(child, ctx) {
			return false
		}
	}
	return true
}
