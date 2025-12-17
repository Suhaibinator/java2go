package symbol

import (
	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func extractTypeParameterNames(node *sitter.Node, source []byte) []string {
	if node == nil {
		return nil
	}

	var params []string
	for _, param := range nodeutil.NamedChildrenOf(node) {
		if param.Type() == "type_parameter" {
			if nameNode := param.NamedChild(0); nameNode != nil {
				params = append(params, nameNode.Content(source))
			}
		}
	}
	return params
}

// ParseSymbols generates a symbol table for a single class file.
func ParseSymbols(root *sitter.Node, source []byte) *FileScope {
	var filePackage string

	var baseClass *sitter.Node

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
			baseClass = node
		}
	}

	return &FileScope{
		Imports:   imports,
		Package:   filePackage,
		BaseClass: parseClassScope(baseClass, source),
	}
}

func parseClassScope(root *sitter.Node, source []byte) *ClassScope {
	return parseClassScopeWithParentTypeParams(root, source, nil)
}

func parseClassScopeWithParentTypeParams(root *sitter.Node, source []byte, parentTypeParams []string) *ClassScope {
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
	ownTypeParams := extractTypeParameterNames(root.ChildByFieldName("type_parameters"), source)

	// Build the type parameters list:
	// 1. Start with parent type parameters (for nested classes)
	// 2. Add own type parameters, but if a name matches a parent's, the inner one shadows it
	// This handles cases like: class Outer<T> { class Inner<T> { } } where Inner's T shadows Outer's T
	for _, parentTP := range parentTypeParams {
		shadowed := false
		for _, ownTP := range ownTypeParams {
			if parentTP == ownTP {
				shadowed = true
				break
			}
		}
		if !shadowed {
			scope.TypeParameters = append(scope.TypeParameters, parentTP)
		}
	}
	scope.TypeParameters = append(scope.TypeParameters, ownTypeParams...)

	// Parse the body of the class (or enum)

	for _, node := range nodeutil.NamedChildrenOf(root.ChildByFieldName("body")) {

		switch node.Type() {
		case "enum_constant":
			// Parse enum constants with their arguments
			constName := node.ChildByFieldName("name").Content(source)
			scope.EnumConstants = append(scope.EnumConstants, constName)

			// Parse constructor arguments if present
			enumConstant := &EnumConstant{Name: constName}
			argsNode := node.ChildByFieldName("arguments")
			if argsNode != nil {
				for _, argNode := range nodeutil.NamedChildrenOf(argsNode) {
					enumConstant.Arguments = append(enumConstant.Arguments, argNode.Content(source))
				}
			}
			scope.EnumConstantList = append(scope.EnumConstantList, enumConstant)
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
		fieldType := nodeToStr(astutil.ParseTypeWithTypeParams(typeNode, source, scope.TypeParameters))

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
		methodTypeParams := extractTypeParameterNames(node.ChildByFieldName("type_parameters"), source)
		combinedTypeParams := append([]string{}, scope.TypeParameters...)
		combinedTypeParams = append(combinedTypeParams, methodTypeParams...)

		declaration := &Definition{
			Name:           HandleExportStatus(public, name),
			OriginalName:   name,
			Parameters:     []*Definition{},
			TypeParameters: methodTypeParams,
			IsStatic:       isStatic,
		}

		if node.Type() == "method_declaration" {
			declaration.Type = nodeToStr(astutil.ParseTypeWithTypeParams(node.ChildByFieldName("type"), source, combinedTypeParams))
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
				Type:         nodeToStr(astutil.ParseTypeWithTypeParams(paramType, source, combinedTypeParams)),
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
