package symbol

import (
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
		params = append(params, TypeParam{
			Name:   nameNode.Content(source),
			Bounds: extractTypeParameterBounds(param, source),
		})
	}
	return params
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
		case "class_declaration", "interface_declaration", "enum_declaration", "annotation_type_declaration":
			topLevelNodes = append(topLevelNodes, node)
		}
	}

	classScopes := make([]*ClassScope, 0, len(topLevelNodes))
	for _, decl := range topLevelNodes {
		classScopes = append(classScopes, parseClassScope(decl, source))
	}

	var baseClass *ClassScope
	if len(classScopes) > 0 {
		baseClass = classScopes[0]
	}

	return &FileScope{
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
	// Rename the type based on the public/static rules
	if root.NamedChild(0).Type() == "modifiers" {
		for _, node := range nodeutil.UnnamedChildrenOf(root.NamedChild(0)) {
			if node.Type() == "public" {
				public = true
			}
		}
	}

	nodeutil.AssertTypeIs(root.ChildByFieldName("name"), "identifier")

	// Parse the main class in the file

	className := root.ChildByFieldName("name").Content(source)
	scope := &ClassScope{
		Class: &Definition{
			OriginalName: className,
			Name:         HandleExportStatus(public, className),
		},
		IsEnum: root.Type() == "enum_declaration",
	}

	// Extract this class's own type parameters first (e.g., class Foo<T, U>)
	ownTypeParams := extractTypeParameters(root.ChildByFieldName("type_parameters"), source)

	// Merge parent type parameters (for nested classes), applying shadowing:
	// class Outer<T> { class Inner<T> { } } where Inner's T shadows Outer's T.
	scope.TypeParameters = MergeTypeParams(parentTypeParams, ownTypeParams)

	// Parse the body of the class (or enum)

	for _, node := range nodeutil.NamedChildrenOf(root.ChildByFieldName("body")) {

		switch node.Type() {
		case "enum_constant":
			// Parse enum constants
			constName := node.ChildByFieldName("name").Content(source)
			scope.EnumConstants = append(scope.EnumConstants, constName)
		case "enum_body_declarations":
			// Parse the methods and constructors inside the enum
			for _, declNode := range nodeutil.NamedChildrenOf(node) {
				parseClassMember(scope, declNode, source)
			}
		default:
			parseClassMember(scope, node, source)
		}
	}

	return scope
}

// parseClassMember parses a single class member (field, method, constructor, or nested class)
func parseClassMember(scope *ClassScope, node *sitter.Node, source []byte) {
	switch node.Type() {
	case "field_declaration":
		var public bool
		// Rename the type based on the public/static rules
		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				if modifier.Type() == "public" {
					public = true
				}
			}
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
		fieldType := nodeToStr(astutil.ParseTypeWithTypeParams(typeNode, source, scope.TypeParameterNames()))

		scope.Fields = append(scope.Fields, &Definition{
			Name:         HandleExportStatus(public, fieldName),
			OriginalName: fieldName,
			Type:         fieldType,
			OriginalType: typeNode.Content(source),
		})
	case "method_declaration", "constructor_declaration":
		var public bool
		var isStatic bool
		// Rename the type based on the public/static rules
		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				if modifier.Type() == "public" {
					public = true
				}
				if modifier.Type() == "static" {
					isStatic = true
				}
			}
		}

		nodeutil.AssertTypeIs(node.ChildByFieldName("name"), "identifier")

		name := node.ChildByFieldName("name").Content(source)
		methodTypeParams := extractTypeParameters(node.ChildByFieldName("type_parameters"), source)
		combinedTypeParams := MergeTypeParams(scope.TypeParameters, methodTypeParams)
		combinedTypeParamNames := TypeParamNames(combinedTypeParams)

		declaration := &Definition{
			Name:           HandleExportStatus(public, name),
			OriginalName:   name,
			Parameters:     []*Definition{},
			TypeParameters: methodTypeParams,
			IsStatic:       isStatic,
		}

		if node.Type() == "method_declaration" {
			declaration.Type = nodeToStr(astutil.ParseTypeWithTypeParams(node.ChildByFieldName("type"), source, combinedTypeParamNames))
			declaration.OriginalType = node.ChildByFieldName("type").Content(source)
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
			})
		}

		if node.ChildByFieldName("body") != nil {
			methodScope := parseScope(node.ChildByFieldName("body"), source)
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
	case "class_declaration", "interface_declaration", "enum_declaration":
		other := parseClassScopeWithParentTypeParams(node, source, scope.TypeParameters)
		// Any subclasses will be renamed to part of their parent class
		other.Class.Rename(scope.Class.Name + other.Class.Name)
		scope.Subclasses = append(scope.Subclasses, other)
	}
}

func parseScope(root *sitter.Node, source []byte) *Definition {
	def := &Definition{}
	for _, node := range nodeutil.NamedChildrenOf(root) {
		switch node.Type() {
		case "local_variable_declaration":
			/*
				name := nodeToStr(ParseExpr(node.ChildByFieldName("declarator").ChildByFieldName("name"), source, Ctx{}))
				def.Children = append(def.Children, &symbol.Definition{
					OriginalName: name,
					OriginalType: node.ChildByFieldName("type").Content(source),
					Type:         nodeToStr(ParseExpr(node.ChildByFieldName("type"), source, Ctx{})),
					Name:         name,
				})
			*/
		case "for_statement", "enhanced_for_statement", "while_statement", "if_statement":
			def.Children = append(def.Children, parseScope(node, source))
		}
	}
	return def
}
