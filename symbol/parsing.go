package symbol

import (
	"strconv"
	"unicode"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func isJavaTypeNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "integral_type", "floating_point_type", "void_type", "boolean_type",
		"generic_type", "array_type", "type_identifier", "scoped_type_identifier",
		"annotated_type":
		return true
	default:
		return false
	}
}

func collectTypeNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}

	if isJavaTypeNode(node) {
		return []*sitter.Node{node}
	}

	var types []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		types = append(types, collectTypeNodes(child)...)
	}
	return types
}

func extractTypeParameterBounds(param *sitter.Node, source []byte) []JavaType {
	if param == nil {
		return nil
	}

	// Prefer field-based access when available.
	boundsNode := param.ChildByFieldName("bounds")
	if boundsNode == nil {
		boundsNode = param.ChildByFieldName("bound")
	}

	var boundTypeNodes []*sitter.Node
	var collectFrom func(n *sitter.Node)
	collectFrom = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if isJavaTypeNode(n) {
			boundTypeNodes = append(boundTypeNodes, n)
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(n) {
			// If the child is a type node at this level, keep it as a whole bound.
			if isJavaTypeNode(child) {
				boundTypeNodes = append(boundTypeNodes, child)
				continue
			}
			// Otherwise recurse; this covers containers like type_bound/type_bounds.
			collectFrom(child)
		}
	}

	if boundsNode != nil {
		collectFrom(boundsNode)
	} else {
		// Fall back to scanning named children after the parameter name.
		// (tree-sitter grammars can differ in whether bounds are exposed via fields).
		for i := 1; i < int(param.NamedChildCount()); i++ {
			collectFrom(param.NamedChild(i))
		}
	}

	if len(boundTypeNodes) == 0 {
		return nil
	}

	// De-duplicate by node range (same node can be reached via recursion).
	seen := make(map[[2]uint32]struct{}, len(boundTypeNodes))
	bounds := make([]JavaType, 0, len(boundTypeNodes))
	for _, n := range boundTypeNodes {
		if n == nil {
			continue
		}
		key := [2]uint32{n.StartByte(), n.EndByte()}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		bounds = append(bounds, JavaType{Original: n.Content(source)})
	}
	return bounds
}

func extractTypeParameters(node *sitter.Node, source []byte) []TypeParam {
	if node == nil {
		return nil
	}

	var params []TypeParam
	for _, param := range nodeutil.NamedChildrenOf(node) {
		if param.Type() != "type_parameter" {
			continue
		}
		nameNode := param.NamedChild(0)
		if nameNode == nil {
			continue
		}
		params = append(params, NewTypeParam(nameNode.Content(source), extractTypeParameterBounds(param, source)))
	}
	return params
}

// ExtractTypeParameters exposes the parser's declaration-identity preserving
// type-parameter extraction for method-local class lowering.
func ExtractTypeParameters(node *sitter.Node, source []byte) []TypeParam {
	return extractTypeParameters(node, source)
}

// ParseSymbols generates a symbol table for a single class file.
func ParseSymbols(root *sitter.Node, source []byte) *FileScope {
	var filePackage string

	var topLevelNodes []*sitter.Node

	imports := make(map[string]string)
	for _, node := range nodeutil.NamedChildrenOf(root) {
		switch node.Type() {
		case "package_declaration":
			filePackage = node.NamedChild(0).Content(source)
		case "import_declaration":
			importedItem := node.NamedChild(0).ChildByFieldName("name").Content(source)
			importPath := node.NamedChild(0).ChildByFieldName("scope").Content(source)

			imports[importedItem] = importPath
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration", "record_declaration":
			topLevelNodes = append(topLevelNodes, node)
		}
	}

	classScopes := make([]*ClassScope, 0, len(topLevelNodes))
	for _, decl := range topLevelNodes {
		classScopes = append(classScopes, parseClassScope(decl, source))
	}

	// The nested-class naming scheme concatenates the enclosing and nested names
	// (Outer + Inner = OuterInner), which can collide with a top-level class of
	// the same name. Disambiguate any duplicates so the generated Go has unique
	// type and constructor names.
	resolveClassNameCollisions(classScopes)

	var baseClass *ClassScope
	if len(classScopes) > 0 {
		baseClass = classScopes[0]
	}

	return &FileScope{
		Source:          source,
		Imports:         imports,
		Package:         filePackage,
		TopLevelClasses: classScopes,
		BaseClass:       baseClass,
	}
}

func parseClassScope(root *sitter.Node, source []byte) *ClassScope {
	return parseClassScopeWithParentTypeParams(root, source, nil)
}

func parseClassScopeWithParentTypeParams(root *sitter.Node, source []byte, parentTypeParams []TypeParam) *ClassScope {
	var public bool
	var isAbstract bool
	var isFinal bool
	// Rename the type based on the public/static rules
	if root.NamedChild(0).Type() == "modifiers" {
		for _, node := range nodeutil.UnnamedChildrenOf(root.NamedChild(0)) {
			if node.Type() == "public" {
				public = true
			}
			if node.Type() == "abstract" {
				isAbstract = true
			}
			if node.Type() == "final" {
				isFinal = true
			}
		}
	}

	nodeutil.AssertTypeIs(root.ChildByFieldName("name"), "identifier")

	// Parse the main class in the file

	className := root.ChildByFieldName("name").Content(source)
	scope := &ClassScope{
		Class: &Definition{
			OriginalName:    className,
			Name:            HandleExportStatus(public, className),
			IsFinal:         isFinal,
			DeclarationNode: root,
		},
		IsEnum:      root.Type() == "enum_declaration",
		IsInterface: root.Type() == "interface_declaration",
		IsAbstract:  isAbstract,
	}

	// Track superclass (for classes/enums). Tree-sitter represents the superclass
	// clause as a container that can include keywords like "extends", so extract
	// the underlying type node (e.g., "Animal" or "Base<T>").
	if superNode := root.ChildByFieldName("superclass"); superNode != nil {
		if types := collectTypeNodes(superNode); len(types) > 0 {
			scope.Superclass = types[0].Content(source)
		} else {
			scope.Superclass = superNode.Content(source)
		}
	}

	// Extract this class's own type parameters first (e.g., class Foo<T, U>)
	ownTypeParams := extractTypeParameters(root.ChildByFieldName("type_parameters"), source)

	// Generated member-class ABIs carry enclosing declarations even when an own
	// parameter shadows the same Java source spelling. Lexical lookup still uses
	// MergeTypeParams at use sites; ABI carriage is declaration-identity based.
	carriedTypeParams := AppendTypeParamsByDeclaration(parentTypeParams, ownTypeParams)
	DisambiguateTypeParamGoNames(carriedTypeParams)
	BindTypeParameterBounds(ownTypeParams, MergeTypeParams(parentTypeParams, ownTypeParams))
	carriedTypeParams = AppendTypeParamsByDeclaration(parentTypeParams, ownTypeParams)
	// Preserve a non-nil empty slice: it distinguishes a parsed non-generic
	// member class that only carries enclosing parameters from a synthetic scope
	// which has not supplied declared-arity metadata yet.
	scope.DeclaredTypeParameters = append([]TypeParam{}, ownTypeParams...)
	scope.TypeParameters = carriedTypeParams

	// Track implemented or extended interfaces. Tree-sitter uses
	// `extends_interfaces` for interface inheritance and `interfaces` for class /
	// enum implements clauses; both feed the same Java superinterface graph.
	interfacesNode := root.ChildByFieldName("interfaces")
	if interfacesNode == nil {
		interfacesNode = root.ChildByFieldName("extends_interfaces")
	}
	if interfacesNode == nil {
		for _, child := range nodeutil.NamedChildrenOf(root) {
			if child.Type() == "interfaces" || child.Type() == "extends_interfaces" {
				interfacesNode = child
				break
			}
		}
	}
	if interfacesNode != nil {
		for _, t := range collectTypeNodes(interfacesNode) {
			scope.ImplementedInterfaces = append(scope.ImplementedInterfaces, t.Content(source))
		}
	}

	// Parse the body of the class (or enum)

	for _, node := range nodeutil.NamedChildrenOf(root.ChildByFieldName("body")) {

		switch node.Type() {
		case "enum_constant":
			// Parse enum constants and capture their constructor arguments for later codegen
			constName := node.ChildByFieldName("name").Content(source)

			var args []*sitter.Node
			if argsNode := node.ChildByFieldName("arguments"); argsNode != nil {
				args = nodeutil.NamedChildrenOf(argsNode)
			}

			scope.EnumConstants = append(scope.EnumConstants, EnumConstant{
				Name:      constName,
				Arguments: args,
				Body:      node.ChildByFieldName("body"),
			})
		case "enum_body_declarations":
			// Parse the methods and constructors inside the enum
			for _, declNode := range nodeutil.NamedChildrenOf(node) {
				parseClassMember(scope, declNode, source)
			}
		default:
			parseClassMember(scope, node, source)
		}
	}

	// A record's components (its header parameters) become fields plus implicit
	// accessor methods named after each component, and a canonical constructor.
	if root.Type() == "record_declaration" {
		injectRecordMembers(scope, root, source)
	}

	// Inject standard enum methods that Java provides implicitly.
	if scope.IsEnum {
		baseType := "*" + scope.Class.Name
		scope.Methods = append(scope.Methods,
			&Definition{
				Name:         HandleExportStatus(true, "name"),
				OriginalName: "name",
				OriginalType: "String",
				Type:         "string",
			},
			&Definition{
				Name:         HandleExportStatus(true, "ordinal"),
				OriginalName: "ordinal",
				OriginalType: "int",
				Type:         "int32",
			},
			&Definition{
				Name:         HandleExportStatus(true, "compareTo"),
				OriginalName: "compareTo",
				OriginalType: "int",
				Type:         "int32",
				Parameters: []*Definition{{
					Name:         "other",
					OriginalName: "other",
					Type:         baseType,
					OriginalType: scope.Class.OriginalName,
				}},
			},
			&Definition{
				Name:         scope.Class.Name + "ValueOf",
				OriginalName: "valueOf",
				Type:         baseType,
				IsStatic:     true,
				Parameters: []*Definition{{
					Name:         "name",
					OriginalName: "name",
					Type:         "string",
					OriginalType: "String",
				}},
			},
		)
	}

	discoverTrivialArrayAccessors(scope, source)

	return scope
}

// parseClassMember parses a single class member (field, method, constructor, or nested class)
func parseClassMember(scope *ClassScope, node *sitter.Node, source []byte) {
	switch node.Type() {
	case "field_declaration":
		var public bool
		var isStatic bool
		var isFinal bool
		var isPrivate bool
		// Rename the type based on the public/static rules
		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				if modifier.Type() == "public" {
					public = true
				}
				if modifier.Type() == "static" {
					isStatic = true
				}
				if modifier.Type() == "final" {
					isFinal = true
				}
				if modifier.Type() == "private" {
					isPrivate = true
				}
			}
		}
		if scope.IsInterface {
			public = true
			isStatic = true
			isFinal = true
		}

		fieldNameNode := node.ChildByFieldName("declarator").ChildByFieldName("name")

		nodeutil.AssertTypeIs(fieldNameNode, "identifier")

		// TODO: Scoped type identifiers are in a format such as RemotePackage.ClassName
		// To handle this, we remove the RemotePackage part, and depend on the later
		// type resolution to figure things out

		// The node that the field's type comes from
		typeNode := node.ChildByFieldName("type")

		// If the field is being assigned to a value
		if typeNode.Type() == "scoped_type_identifier" {
			typeNode = typeNode.NamedChild(int(typeNode.NamedChildCount()) - 1)
		}

		// The converted name and type of the field
		fieldName := fieldNameNode.Content(source)
		fieldType := nodeToStr(astutil.ParseTypeWithTypeParams(typeNode, source, TypeParamNames(scope.TypeParameters)))

		initializer := node.ChildByFieldName("declarator").ChildByFieldName("value")
		scope.Fields = append(scope.Fields, &Definition{
			Name:         HandleExportStatus(public, fieldName),
			OriginalName: fieldName,
			Type:         fieldType,
			OriginalType: typeNode.Content(source),
			DirectTypeParameter: DirectTypeParamForJavaType(
				typeNode.Content(source),
				scope.TypeParameters,
			),
			TypeParameterBindings: VisibleTypeParamBindings(scope.TypeParameters),
			IsStatic:              isStatic,
			IsFinal:               isFinal,
			IsPrivate:             isPrivate,
			IsCompileTimeConstant: isStatic && isFinal &&
				javaConstantVariableType(typeNode.Content(source)) &&
				javaConstantExpression(initializer, source, scope),
			DeclarationNode: node,
		})
	case "method_declaration", "abstract_method_declaration", "constructor_declaration":
		var public bool
		var isStatic bool
		var isPrivate bool
		var isFinal bool
		// Java interface methods are implicitly public.
		if scope.IsInterface && node.Type() != "constructor_declaration" {
			public = true
		}
		// Rename the type based on the public/static rules
		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				if modifier.Type() == "public" {
					public = true
				}
				if modifier.Type() == "static" {
					isStatic = true
				}
				if modifier.Type() == "private" {
					isPrivate = true
				}
				if modifier.Type() == "final" {
					isFinal = true
				}
			}
		}

		nodeutil.AssertTypeIs(node.ChildByFieldName("name"), "identifier")

		name := node.ChildByFieldName("name").Content(source)
		methodTypeParams := extractTypeParameters(node.ChildByFieldName("type_parameters"), source)
		carriedMethodParams := AppendTypeParamsByDeclaration(scope.TypeParameters, methodTypeParams)
		DisambiguateTypeParamGoNames(carriedMethodParams)
		combinedTypeParams := MergeTypeParams(scope.TypeParameters, methodTypeParams)
		BindTypeParameterBounds(methodTypeParams, combinedTypeParams)
		combinedTypeParamNames := TypeParamNames(combinedTypeParams)

		declaration := &Definition{
			Name:            HandleExportStatus(public, name),
			OriginalName:    name,
			Parameters:      []*Definition{},
			TypeParameters:  methodTypeParams,
			IsStatic:        isStatic,
			IsPrivate:       isPrivate,
			IsFinal:         isFinal,
			HasBody:         node.ChildByFieldName("body") != nil,
			DeclarationNode: node,
		}

		if node.Type() == "method_declaration" {
			declaration.Type = nodeToStr(astutil.ParseTypeWithTypeParams(node.ChildByFieldName("type"), source, combinedTypeParamNames))
			declaration.OriginalType = node.ChildByFieldName("type").Content(source)
			declaration.DirectTypeParameter = DirectTypeParamForJavaType(declaration.OriginalType, combinedTypeParams)
			declaration.TypeParameterBindings = VisibleTypeParamBindings(combinedTypeParams)
		} else {
			// A constructor declaration returns the type being constructed

			// Rename the constructor with "New" + name of type
			declaration.Rename(HandleExportStatus(public, "New") + name)
			declaration.Constructor = true

			// There is no original type, and the constructor returns the name of
			// the new type
			declaration.Type = scope.Class.OriginalName
		}

		// Parse the parameters

		for _, parameter := range nodeutil.NamedChildrenOf(node.ChildByFieldName("parameters")) {

			var paramName string
			var paramType *sitter.Node

			// If this is a spread parameter, then it will be in the format:
			// (type) (variable_declarator name: (name))
			if parameter.Type() == "spread_parameter" {
				paramName = parameter.NamedChild(1).ChildByFieldName("name").Content(source)
				paramType = parameter.NamedChild(0)
			} else {
				paramName = parameter.ChildByFieldName("name").Content(source)
				paramType = parameter.ChildByFieldName("type")
			}

			declaration.Parameters = append(declaration.Parameters, &Definition{
				Name:         paramName,
				OriginalName: paramName,
				Type:         nodeToStr(astutil.ParseTypeWithTypeParams(paramType, source, combinedTypeParamNames)),
				OriginalType: paramType.Content(source),
				DirectTypeParameter: DirectTypeParamForJavaType(
					paramType.Content(source),
					combinedTypeParams,
				),
				TypeParameterBindings: VisibleTypeParamBindings(combinedTypeParams),
			})
		}

		if node.ChildByFieldName("body") != nil {
			methodScope := parseScope(node.ChildByFieldName("body"), source, combinedTypeParams)
			if !methodScope.IsEmpty() {
				declaration.Children = append(declaration.Children, methodScope.Children...)
			}
		}

		// Go doesn't support method type parameters on methods, so instance generic
		// methods are modeled via helper types. Constructors are plain functions in
		// the generated Go, so they don't need helpers even if they declare type
		// parameters.
		if node.Type() == "method_declaration" && len(methodTypeParams) > 0 && !isStatic {
			declaration.RequiresHelper = true
			declaration.HelperName = scope.Class.Name + declaration.Name + "Helper"
		}

		scope.Methods = append(scope.Methods, declaration)
	case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
		implicitlyStatic := scope.IsInterface
		parentTypeParams := scope.TypeParameters
		if implicitlyStatic || node.Type() != "class_declaration" || nestedClassIsStatic(node) {
			parentTypeParams = nil
		}
		other := parseClassScopeWithParentTypeParams(node, source, parentTypeParams)
		// Any subclasses will be renamed to part of their parent class
		other.Class.Rename(scope.Class.Name + other.Class.Name)
		// Constructors carry the (now stale) original short name in their "New" +
		// Name form, so rebuild them against the renamed nested-class name.
		retargetConstructorNames(other)
		other.Enclosing = scope
		// A non-static nested class (only plain classes can be "inner"; nested
		// interfaces and enums are implicitly static) holds an enclosing instance.
		if node.Type() == "class_declaration" && !implicitlyStatic && !nestedClassIsStatic(node) {
			other.IsInner = true
		}
		scope.Subclasses = append(scope.Subclasses, other)
	}
}

// injectRecordMembers adds a record's components as fields, an accessor method
// per component (named after the component, e.g. x() not getX()), and a
// canonical constructor. Components are exported since records expose them
// publicly. Explicit accessors/constructors already declared in the body are not
// duplicated.
func injectRecordMembers(scope *ClassScope, root *sitter.Node, source []byte) {
	paramsNode := root.ChildByFieldName("parameters")
	if paramsNode == nil {
		return
	}

	type component struct {
		name     string
		origType string
		goType   string
	}
	var components []component
	for _, param := range nodeutil.NamedChildrenOf(paramsNode) {
		if param.Type() != "formal_parameter" {
			continue
		}
		nameNode := param.ChildByFieldName("name")
		typeNode := param.ChildByFieldName("type")
		if nameNode == nil || typeNode == nil {
			continue
		}
		comp := component{
			name:     nameNode.Content(source),
			origType: typeNode.Content(source),
			goType:   nodeToStr(astutil.ParseTypeWithTypeParams(typeNode, source, TypeParamNames(scope.TypeParameters))),
		}
		components = append(components, comp)

		// The component field is unexported so it does not collide with the
		// exported accessor METHOD of the same Java name (Go shares the field/method
		// namespace; Java does not).
		scope.Fields = append(scope.Fields, &Definition{
			Name:         Lowercase(comp.name),
			OriginalName: comp.name,
			Type:         comp.goType,
			OriginalType: comp.origType,
			DirectTypeParameter: DirectTypeParamForJavaType(
				comp.origType,
				scope.TypeParameters,
			),
			TypeParameterBindings: VisibleTypeParamBindings(scope.TypeParameters),
		})

		// Accessor method named exactly after the component (exported), unless the
		// body already declares one.
		if scope.FindMethodByName(comp.name, nil) == nil {
			scope.Methods = append(scope.Methods, &Definition{
				Name:         HandleExportStatus(true, comp.name),
				OriginalName: comp.name,
				Type:         comp.goType,
				OriginalType: comp.origType,
			})
		}
	}

	// Implicit value-equality method, so `a.equals(b)` resolves to the synthesized
	// Equals, unless the body declares its own.
	if scope.FindMethodByName("equals", nil) == nil {
		scope.Methods = append(scope.Methods, &Definition{
			Name:         "Equals",
			OriginalName: "equals",
			Type:         "boolean",
			OriginalType: "boolean",
			Parameters: []*Definition{{
				Name:         "other",
				OriginalName: "other",
				Type:         "*" + scope.Class.Name,
				OriginalType: scope.Class.OriginalName,
			}},
		})
	}

	// Canonical constructor, unless one is already declared.
	hasCanonical := false
	for _, method := range scope.Methods {
		if method != nil && method.Constructor && len(method.Parameters) == len(components) {
			hasCanonical = true
			break
		}
	}
	if !hasCanonical {
		ctor := &Definition{
			Name:         "New" + scope.Class.Name,
			OriginalName: scope.Class.OriginalName,
			Constructor:  true,
			Type:         scope.Class.OriginalName,
		}
		for _, comp := range components {
			ctor.Parameters = append(ctor.Parameters, &Definition{
				Name:         comp.name,
				OriginalName: comp.name,
				Type:         comp.goType,
				OriginalType: comp.origType,
			})
		}
		scope.Methods = append(scope.Methods, ctor)
	}
}

// nestedClassIsStatic reports whether a nested class declaration carries the
// `static` modifier.
func nestedClassIsStatic(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	modifiers := node.NamedChild(0)
	if modifiers == nil || modifiers.Type() != "modifiers" {
		return false
	}
	for _, modifier := range nodeutil.UnnamedChildrenOf(modifiers) {
		if modifier.Type() == "static" {
			return true
		}
	}
	return false
}

// retargetConstructorNames rebuilds the "New<Class>" constructor names for a
// class scope after the class itself has been renamed (e.g. when a nested class
// Inner is renamed to OuterInner). It preserves each constructor's export
// status, which is encoded in the first character of its existing name.
func retargetConstructorNames(scope *ClassScope) {
	if scope == nil || scope.Class == nil {
		return
	}
	className := scope.Class.Name
	for _, method := range scope.Methods {
		if method == nil || !method.Constructor {
			continue
		}
		exported := false
		if len(method.Name) > 0 {
			exported = unicode.IsUpper(rune(method.Name[0]))
		}
		method.Rename(HandleExportStatus(exported, "New") + className)
	}
}

// resolveClassNameCollisions walks every class scope in the file (top-level and
// nested, depth-first) and ensures each generated Go type name is unique. The
// first scope to claim a name keeps it; any later scope with the same name is
// renamed with a numeric suffix and its constructors retargeted. This guards the
// Outer+Inner concatenation scheme against colliding with a same-named top-level
// class.
func resolveClassNameCollisions(scopes []*ClassScope) {
	used := make(map[string]struct{})
	var visit func(scope *ClassScope)
	visit = func(scope *ClassScope) {
		if scope == nil || scope.Class == nil {
			return
		}
		name := scope.Class.Name
		if _, taken := used[name]; taken {
			// Find the next free suffixed name (Name2, Name3, ...).
			for i := 2; ; i++ {
				candidate := name + strconv.Itoa(i)
				if _, clash := used[candidate]; !clash {
					name = candidate
					break
				}
			}
			scope.Class.Rename(name)
			retargetConstructorNames(scope)
		}
		used[name] = struct{}{}
		for _, sub := range scope.Subclasses {
			visit(sub)
		}
	}
	for _, scope := range scopes {
		visit(scope)
	}
}

func parseScope(root *sitter.Node, source []byte, typeParams []TypeParam) *Definition {
	def := &Definition{}
	if root == nil {
		return def
	}
	typeParamNames := TypeParamNames(typeParams)
	typeParamBindings := VisibleTypeParamBindings(typeParams)
	for _, node := range nodeutil.NamedChildrenOf(root) {
		switch node.Type() {
		case "local_variable_declaration":
			typeNode := node.ChildByFieldName("type")
			declarator := node.ChildByFieldName("declarator")
			if typeNode == nil || declarator == nil {
				continue
			}

			typeStr := nodeToStr(astutil.ParseTypeWithTypeParams(typeNode, source, typeParamNames))
			originalType := typeNode.Content(source)

			if declarator.NamedChildCount() == 1 {
				nameNode := declarator.NamedChild(0)
				def.Children = append(def.Children, &Definition{
					OriginalName:          nameNode.Content(source),
					Name:                  nameNode.Content(source),
					OriginalType:          originalType,
					DirectTypeParameter:   DirectTypeParamForJavaType(originalType, typeParams),
					TypeParameterBindings: typeParamBindings,
					Type:                  typeStr,
				})
				continue
			}

			for ind := 0; ind < int(declarator.NamedChildCount())-1; ind += 2 {
				nameNode := declarator.NamedChild(ind)
				def.Children = append(def.Children, &Definition{
					OriginalName:          nameNode.Content(source),
					Name:                  nameNode.Content(source),
					OriginalType:          originalType,
					DirectTypeParameter:   DirectTypeParamForJavaType(originalType, typeParams),
					TypeParameterBindings: typeParamBindings,
					Type:                  typeStr,
				})
			}
		default:
			inner := parseScope(node, source, typeParams)
			if len(inner.Children) > 0 {
				def.Children = append(def.Children, inner.Children...)
			}
		}
	}
	return def
}
