package main

import (
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

var javaTypeNodeKinds = map[string]struct{}{
	"integral_type":          {},
	"floating_point_type":    {},
	"void_type":              {},
	"boolean_type":           {},
	"generic_type":           {},
	"array_type":             {},
	"type_identifier":        {},
	"scoped_type_identifier": {},
	"annotated_type":         {},
}

func collectTypeNodes(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}

	if _, ok := javaTypeNodeKinds[node.Type()]; ok {
		return []*sitter.Node{node}
	}

	var types []*sitter.Node
	for _, child := range nodeutil.NamedChildrenOf(node) {
		types = append(types, collectTypeNodes(child)...)
	}

	return types
}

// ParseDecls represents any type that returns a list of top-level declarations,
// this is any class, interface, or enum declaration
func ParseDecls(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	switch node.Type() {
	case "class_declaration":
		// The declarations and fields for the class
		declarations := []ast.Decl{}
		fields := &ast.FieldList{}

		// Handle inheritance: embed superclass and implemented interfaces
		typeParams := ctx.currentClass.TypeParameterNames()

		if superNode := node.ChildByFieldName("superclass"); superNode != nil {
			for _, t := range collectTypeNodes(superNode) {
				fields.List = append(fields.List, &ast.Field{Type: astutil.ParseTypeWithTypeParams(t, source, typeParams)})
			}
		}

		if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := astutil.ParseTypeWithTypeParams(t, source, typeParams)
				if star, ok := embedType.(*ast.StarExpr); ok {
					embedType = star.X
				}
				fields.List = append(fields.List, &ast.Field{Type: embedType})
			}
		}

		// Global variables
		globalVariables := &ast.GenDecl{Tok: token.VAR}

		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name

		// First, look through the class's body for field declarations
		for _, child := range nodeutil.NamedChildrenOf(node.ChildByFieldName("body")) {
			if child.Type() == "field_declaration" {

				var staticField bool

				comments := []*ast.Comment{}

				// Handle any modifiers that the field might have
				if child.NamedChild(0).Type() == "modifiers" {
					for _, modifier := range nodeutil.UnnamedChildrenOf(child.NamedChild(0)) {
						switch modifier.Type() {
						case "static":
							staticField = true
						case "marker_annotation", "annotation":
							modContent := modifier.Content(source)
							comments = append(comments, &ast.Comment{Text: "//" + modContent})
							if excludedAnnotations[modContent] {
								// Skip this field if there is an ignored annotation
								continue
							}
						}
					}
				}

				// TODO: If a field is initialized to a value, that value is discarded

				field := &ast.Field{}
				if len(comments) > 0 {
					field.Doc = &ast.CommentGroup{List: comments}
				}

				fieldName := child.ChildByFieldName("declarator").ChildByFieldName("name").Content(source)

				fieldDef := ctx.currentClass.FindField().ByOriginalName(fieldName)[0]

				field.Names, field.Type = []*ast.Ident{{Name: fieldDef.Name}}, &ast.Ident{Name: fieldDef.Type}

				if staticField {
					globalVariables.Specs = append(globalVariables.Specs, &ast.ValueSpec{Names: field.Names, Type: field.Type})
				} else {
					fields.List = append(fields.List, field)
				}
			}
		}

		// Add the global variables
		if len(globalVariables.Specs) > 0 {
			declarations = append(declarations, globalVariables)
		}

		// Add the struct for the class (with type parameters if present)
		declarations = append(declarations, GenStructWithTypeParams(ctx.className, fields, ctx.currentClass.TypeParameters))

		// Add all the declarations that appear in the class
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		return declarations
	case "class_body", "enum_body": // The body of the currently parsed class or enum
		decls := []ast.Decl{}

		// To switch to parsing the subclasses of a class, since we assume that
		// all the class's subclass definitions are in-order, if we find some number
		// of subclasses in a class, we can refer to them by index
		var subclassIndex int

		for _, child := range nodeutil.NamedChildrenOf(node) {
			switch child.Type() {
			// Skip fields, comments, and enum constants (already processed)
			case "field_declaration", "comment", "enum_constant":
			case "constructor_declaration", "method_declaration", "abstract_method_declaration", "static_initializer":
				for _, d := range ParseDecl(child, source, ctx) {
					// If the declaration is bad, skip it
					_, bad := d.(*ast.BadDecl)
					if !bad {
						decls = append(decls, d)
					}
				}
			case "enum_body_declarations":
				// Process methods and constructors inside enum body declarations
				for _, declChild := range nodeutil.NamedChildrenOf(child) {
					switch declChild.Type() {
					case "constructor_declaration", "method_declaration", "abstract_method_declaration", "static_initializer":
						for _, d := range ParseDecl(declChild, source, ctx) {
							_, bad := d.(*ast.BadDecl)
							if !bad {
								decls = append(decls, d)
							}
						}
					}
				}
			// Subclasses
			case "class_declaration", "interface_declaration", "enum_declaration":
				newCtx := ctx.Clone()
				newCtx.currentClass = ctx.currentClass.Subclasses[subclassIndex]
				subclassIndex++
				decls = append(decls, ParseDecls(child, source, newCtx)...)
			}
		}

		return decls
	case "interface_declaration":
		nameNode := node.ChildByFieldName("name")
		interfaceName := nameNode.Content(source)

		// Prefer the resolved, exported name from symbols when available
		if ctx.currentClass != nil && ctx.currentClass.Class != nil {
			interfaceName = ctx.currentClass.Class.Name
		} else if ctx.currentFile != nil {
			if def := ctx.currentFile.FindClass(interfaceName); def != nil {
				interfaceName = def.Name
			}
		}

		ctx.className = interfaceName

		var typeParams []string
		if ctx.currentClass != nil {
			typeParams = ctx.currentClass.TypeParameterNames()
		}

		interfacesNode := node.ChildByFieldName("extends_interfaces")
		if interfacesNode == nil {
			interfacesNode = node.ChildByFieldName("interfaces")
		}
		if interfacesNode == nil {
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if child.Type() == "extends_interfaces" || child.Type() == "interfaces" {
					interfacesNode = child
					break
				}
			}
		}

		methods := &ast.FieldList{}

		// Embed any extended interfaces directly into the generated interface
		if interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := astutil.ParseTypeWithTypeParams(t, source, typeParams)
				if star, ok := embedType.(*ast.StarExpr); ok {
					embedType = star.X
				}
				methods.List = append(methods.List, &ast.Field{Type: embedType})
			}
		}

		// Add the interface's declared methods
		if body := node.ChildByFieldName("body"); body != nil {
			for _, c := range nodeutil.NamedChildrenOf(body) {
				if c.Type() == "method_declaration" {
					parsedMethod := ParseNode(c, source, ctx).(*ast.Field)
					// If the method was ignored with an annotation, it will return a blank
					// field, so ignore that
					if parsedMethod.Type != nil {
						methods.List = append(methods.List, parsedMethod)
					}
				}
			}
		}

		var classTypeParams []symbol.TypeParam
		if ctx.currentClass != nil {
			classTypeParams = ctx.currentClass.TypeParameters
		}

		return []ast.Decl{GenInterface(interfaceName, methods, classTypeParams)}
	case "enum_declaration":
		// Enums are modeled as structs with named singleton instances rather than integer constants.

		enumName := node.ChildByFieldName("name").Content(source)
		if ctx.currentClass != nil {
			ctx.className = ctx.currentClass.Class.Name
		} else {
			ctx.className = ctx.currentFile.FindClass(enumName).Name
			ctx.currentClass = ctx.currentFile.FindClassScope(enumName)
		}

		declarations := []ast.Decl{}

		// Build struct fields for enum instances. Always include Name and Ordinal
		// to mirror Java's Enum metadata.
		fields := &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{{Name: "Name"}}, Type: &ast.Ident{Name: "string"}},
				{Names: []*ast.Ident{{Name: "Ordinal"}}, Type: &ast.Ident{Name: "int"}},
			},
		}

		// Embed implemented interfaces
		typeParams := ctx.currentClass.TypeParameterNames()
		if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := astutil.ParseTypeWithTypeParams(t, source, typeParams)
				if star, ok := embedType.(*ast.StarExpr); ok {
					embedType = star.X
				}
				fields.List = append(fields.List, &ast.Field{Type: embedType})
			}
		}
		globalVariables := &ast.GenDecl{Tok: token.VAR}

		// Add declared fields from the enum body
		for _, fieldDef := range ctx.currentClass.Fields {
			field := &ast.Field{}
			field.Names, field.Type = []*ast.Ident{{Name: fieldDef.Name}}, &ast.Ident{Name: fieldDef.Type}

			if fieldDef.IsStatic {
				globalVariables.Specs = append(globalVariables.Specs, &ast.ValueSpec{Names: field.Names, Type: field.Type})
			} else {
				fields.List = append(fields.List, field)
			}
		}

		if len(globalVariables.Specs) > 0 {
			declarations = append(declarations, globalVariables)
		}

		// Declare the enum struct type
		declarations = append(declarations, GenStructWithTypeParams(ctx.className, fields, ctx.currentClass.TypeParameters))

		// Generate ordinal constants to preserve declaration order
		if len(ctx.currentClass.EnumConstants) > 0 {
			ordinalSpecs := []ast.Spec{}
			ordinalPrefix := "_" + symbol.Lowercase(ctx.className) + "_ordinal_"
			for i, enumConst := range ctx.currentClass.EnumConstants {
				spec := &ast.ValueSpec{Names: []*ast.Ident{{Name: ordinalPrefix + enumConst.Name}}}
				if i == 0 {
					spec.Values = []ast.Expr{&ast.Ident{Name: "iota"}}
				}
				ordinalSpecs = append(ordinalSpecs, spec)
			}

			declarations = append(declarations, &ast.GenDecl{Tok: token.CONST, Specs: ordinalSpecs})

			// Build enum instances
			valueSpecs := []ast.Spec{}
			valuesVarName := "_" + symbol.Lowercase(ctx.className) + "Values"
			valuesSlice := []ast.Expr{}
			for _, enumConst := range ctx.currentClass.EnumConstants {
				ordinalIdent := &ast.Ident{Name: ordinalPrefix + enumConst.Name}
				initializer := buildEnumConstantInitializer(enumConst, ordinalIdent, ctx, source)

				valueSpecs = append(valueSpecs, &ast.ValueSpec{
					Names:  []*ast.Ident{{Name: enumConst.Name}},
					Values: []ast.Expr{initializer},
				})
				valuesSlice = append(valuesSlice, &ast.Ident{Name: enumConst.Name})
			}

			declarations = append(declarations, &ast.GenDecl{Tok: token.VAR, Specs: valueSpecs})

			declarations = append(declarations, &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{{Name: valuesVarName}},
						Values: []ast.Expr{
							&ast.CompositeLit{
								Type: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}},
								Elts: valuesSlice,
							},
						},
					},
				},
			})

			// Generate Values() function: func EnumNameValues() []*EnumName { return _enumNameValues }
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: ctx.className + "Values"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
					Results: &ast.FieldList{
						List: []*ast.Field{{Type: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}}}},
					},
				},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: valuesVarName}}}}},
			})

			// Generate valueOf(String) method
			valueOfCases := []ast.Stmt{}
			for _, enumConst := range ctx.currentClass.EnumConstants {
				valueOfCases = append(valueOfCases, &ast.CaseClause{
					List: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"" + enumConst.Name + "\""}},
					Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: enumConst.Name}}}},
				})
			}
			valueOfCases = append(valueOfCases, &ast.CaseClause{
				List: nil,
				Body: []ast.Stmt{&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.Ident{Name: "panic"}, Args: []ast.Expr{&ast.BinaryExpr{X: &ast.BasicLit{Kind: token.STRING, Value: "\"No enum constant \""}, Op: token.ADD, Y: &ast.Ident{Name: "name"}}}}}, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "nil"}}}},
			})

			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: ctx.className + "ValueOf"},
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "name"}}, Type: &ast.Ident{Name: "string"}}}},
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}}}},
				},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.SwitchStmt{
						Tag:  &ast.Ident{Name: "name"},
						Body: &ast.BlockStmt{List: valueOfCases},
					},
				}},
			})

			receiverBase := instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.TypeParameterNames()))
			receiver := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: ShortName(ctx.className)}}, Type: &ast.StarExpr{X: receiverBase}}}}

			// name() accessor
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "name")},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: "Name"}}}}}},
			})

			// ordinal() accessor
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "ordinal")},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: "Ordinal"}}}}}},
			})

			// compareTo(E)
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "compareTo")},
				Recv: receiver,
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "other"}}, Type: &ast.StarExpr{X: receiverBase}}}},
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}},
				},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.BinaryExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: "Ordinal"}}, Op: token.SUB, Y: &ast.SelectorExpr{X: &ast.Ident{Name: "other"}, Sel: &ast.Ident{Name: "Ordinal"}}}}}}},
			})
		}

		// Parse the enum body declarations (methods, constructors, etc.)
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		return declarations
	}
	panic("Unknown type to parse for decls: " + node.Type())
}

func zeroValueForType(expr ast.Expr) ast.Expr {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "", "void":
			return nil
		case "string":
			return &ast.BasicLit{Kind: token.STRING, Value: "\"\""}
		case "bool":
			return &ast.Ident{Name: "false"}
		case "int", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "byte", "rune":
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		default:
			return &ast.Ident{Name: "nil"}
		}
	case *ast.StarExpr, *ast.ArrayType, *ast.MapType, *ast.InterfaceType, *ast.FuncType, *ast.SliceExpr, *ast.ChanType:
		return &ast.Ident{Name: "nil"}
	default:
		return &ast.CompositeLit{Type: expr}
	}
}

func methodNodeMatchesDefinition(node *sitter.Node, def *symbol.Definition, source []byte) bool {
	if def == nil || node == nil {
		return false
	}
	if node.ChildByFieldName("name").Content(source) != def.OriginalName {
		return false
	}

	paramsNode := node.ChildByFieldName("parameters")
	if def.Parameters == nil {
		return paramsNode.NamedChildCount() == 0
	}
	if len(def.Parameters) != int(paramsNode.NamedChildCount()) {
		return false
	}

	for index, param := range nodeutil.NamedChildrenOf(paramsNode) {
		var paramType string
		if param.Type() == "spread_parameter" {
			paramType = param.NamedChild(0).Content(source)
		} else {
			paramType = param.ChildByFieldName("type").Content(source)
		}
		if def.Parameters[index].OriginalType != paramType {
			return false
		}
	}
	return true
}

func enumConstantMethodDeclarations(body *sitter.Node) []*sitter.Node {
	methods := []*sitter.Node{}
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_declaration" {
			methods = append(methods, node)
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(body)
	return methods
}

func buildEnumMethodImplementation(funcName string, node *sitter.Node, def *symbol.Definition, ctx Ctx, source []byte, receiverBaseType ast.Expr) *ast.FuncDecl {
	ctx.localScope = def
	params := ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList)
	params.List = append([]*ast.Field{{Names: []*ast.Ident{{Name: ShortName(ctx.className)}}, Type: &ast.StarExpr{X: receiverBaseType}}}, params.List...)

	body := ParseStmt(node.ChildByFieldName("body"), source, ctx).(*ast.BlockStmt)

	var results *ast.FieldList
	if def.Type != "" {
		results = &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: def.Type}}}}
	}

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: funcName},
		Type: &ast.FuncType{Params: params, Results: results},
		Body: body,
	}
}

func buildEnumMethodWrapper(def *symbol.Definition, overrides map[string]string, defaultImpl string, params *ast.FieldList, results *ast.FieldList, receiver *ast.FieldList, ctx Ctx) *ast.FuncDecl {
	recvName := ShortName(ctx.className)
	args := []ast.Expr{&ast.Ident{Name: recvName}}
	if params != nil {
		for _, field := range params.List {
			for _, name := range field.Names {
				args = append(args, &ast.Ident{Name: name.Name})
			}
		}
	}

	clauses := []ast.Stmt{}
	for constName, implName := range overrides {
		clauses = append(clauses, &ast.CaseClause{
			List: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"" + constName + "\""}},
			Body: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: implName}, Args: args}}}},
		})
	}

	defaultBody := []ast.Stmt{}
	if defaultImpl != "" {
		defaultBody = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: defaultImpl}, Args: args}}}}
	} else {
		panicStmt := &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.Ident{Name: "panic"}, Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"abstract enum method not implemented\""}}}}
		defaultBody = append(defaultBody, panicStmt)
		if results != nil && len(results.List) > 0 {
			defaultBody = append(defaultBody, &ast.ReturnStmt{Results: []ast.Expr{zeroValueForType(results.List[0].Type)}})
		}
	}
	clauses = append(clauses, &ast.CaseClause{Body: defaultBody})

	wrapperBody := &ast.BlockStmt{List: []ast.Stmt{
		&ast.SwitchStmt{
			Tag:  &ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: "Name"}},
			Body: &ast.BlockStmt{List: clauses},
		},
	}}

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: def.Name},
		Recv: receiver,
		Type: &ast.FuncType{Params: params, Results: results},
		Body: wrapperBody,
	}
}

func typeParamExprs(params []string) []ast.Expr {
	if len(params) == 0 {
		return nil
	}
	result := make([]ast.Expr, len(params))
	for i, tp := range params {
		result[i] = &ast.Ident{Name: tp}
	}
	return result
}

func instantiateGenericType(name string, args []ast.Expr) ast.Expr {
	if len(args) == 0 {
		return &ast.Ident{Name: name}
	}
	if len(args) == 1 {
		return &ast.IndexExpr{
			X:     &ast.Ident{Name: name},
			Index: args[0],
		}
	}
	return &ast.IndexListExpr{
		X:       &ast.Ident{Name: name},
		Indices: args,
	}
}

// buildEnumConstantInitializer constructs the Go expression used to initialize a single enum constant.
// It invokes a matching constructor if one exists, then injects the synthetic Name and Ordinal fields
// to mirror Java enum metadata.
func buildEnumConstantInitializer(enumConst symbol.EnumConstant, ordinal ast.Expr, ctx Ctx, source []byte) ast.Expr {
	args := parseEnumConstantArguments(enumConst, ctx, source)

	var baseInit ast.Expr = &ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: &ast.Ident{Name: ctx.className}}}
	if ctor := findEnumConstructor(ctx, len(args)); ctor != nil {
		baseInit = &ast.CallExpr{Fun: &ast.Ident{Name: ctor.Name}, Args: args}
	}

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: "inst"}}, Tok: token.DEFINE, Rhs: []ast.Expr{baseInit}},
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "inst"}, Sel: &ast.Ident{Name: "Name"}}}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"" + enumConst.Name + "\""}}},
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "inst"}, Sel: &ast.Ident{Name: "Ordinal"}}}, Tok: token.ASSIGN, Rhs: []ast.Expr{ordinal}},
				&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "inst"}}},
			}},
		},
	}
}

func parseEnumConstantArguments(enumConst symbol.EnumConstant, ctx Ctx, source []byte) []ast.Expr {
	args := []ast.Expr{}
	for _, arg := range enumConst.Arguments {
		args = append(args, ParseExpr(arg, source, ctx))
	}
	return args
}

func findEnumConstructor(ctx Ctx, argumentCount int) *symbol.Definition {
	for _, def := range ctx.currentClass.Methods {
		if def.Constructor && len(def.Parameters) == argumentCount {
			return def
		}
	}
	return nil
}

func genInstanceGenericHelperDecls(ctx Ctx, def *symbol.Definition, doc *ast.CommentGroup, params, results *ast.FieldList, body *ast.BlockStmt, receiverBaseType ast.Expr) []ast.Decl {
	combinedTypeParams := symbol.MergeTypeParams(ctx.currentClass.TypeParameters, def.TypeParameters)
	combinedTypeParamNames := symbol.TypeParamNames(combinedTypeParams)

	helperName := def.HelperName
	helperFields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "recv"}},
				Type:  &ast.StarExpr{X: receiverBaseType},
			},
		},
	}
	helperStruct := GenStructWithTypeParams(helperName, helperFields, combinedTypeParams)

	helperTypeArgs := typeParamExprs(combinedTypeParamNames)
	helperTypeExpr := instantiateGenericType(helperName, helperTypeArgs)

	receiverShortName := ShortName(ctx.className)
	constructorName := "New" + helperName
	constructorParams := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: receiverShortName}},
				Type:  &ast.StarExpr{X: receiverBaseType},
			},
		},
	}
	returnType := &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: helperTypeExpr}}}}
	constructorBody := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.UnaryExpr{
						Op: token.AND,
						X: &ast.CompositeLit{
							Type: helperTypeExpr,
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   &ast.Ident{Name: "recv"},
									Value: &ast.Ident{Name: receiverShortName},
								},
							},
						},
					},
				},
			},
		},
	}
	constructor := GenFuncDeclWithTypeParams(constructorName, combinedTypeParams, constructorParams, returnType, constructorBody)

	helperRecvName := receiverShortName + "Helper"
	helperReceiver := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: helperRecvName}},
				Type:  &ast.StarExpr{X: helperTypeExpr},
			},
		},
	}

	assignOriginalReceiver := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: receiverShortName}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.SelectorExpr{
				X:   &ast.Ident{Name: helperRecvName},
				Sel: &ast.Ident{Name: "recv"},
			},
		},
	}
	modifiedBody := &ast.BlockStmt{
		List: append([]ast.Stmt{assignOriginalReceiver}, body.List...),
	}

	funcDecl := &ast.FuncDecl{
		Doc:  doc,
		Name: &ast.Ident{Name: def.Name},
		Recv: helperReceiver,
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: modifiedBody,
	}

	return []ast.Decl{helperStruct, constructor, funcDecl}
}

// ParseDecl parses a top-level declaration within a source file, including
// but not limited to fields and methods
func ParseDecl(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	switch node.Type() {
	case "constructor_declaration":
		paramNode := node.ChildByFieldName("parameters")

		constructorName := node.ChildByFieldName("name").Content(source)

		comparison := func(d *symbol.Definition) bool {
			// The names must match
			if constructorName != d.OriginalName {
				return false
			}

			// Size of parameters must match
			if int(paramNode.NamedChildCount()) != len(d.Parameters) {
				return false
			}

			// Go through the types and check to see if they differ
			for index, param := range nodeutil.NamedChildrenOf(paramNode) {
				var paramType string
				if param.Type() == "spread_parameter" {
					paramType = param.NamedChild(0).Content(source)
				} else {
					paramType = param.ChildByFieldName("type").Content(source)
				}
				if paramType != d.Parameters[index].OriginalType {
					return false
				}
			}

			return true
		}

		// Search through the current class for the constructor, which is simply labeled as a method
		ctx.localScope = ctx.currentClass.FindMethod().By(comparison)[0]

		body := ParseStmt(node.ChildByFieldName("body"), source, ctx).(*ast.BlockStmt)

		// Generate the struct type for `new` call - if generic, include type params
		var structType ast.Expr = &ast.Ident{Name: ctx.className}
		if len(ctx.currentClass.TypeParameters) > 0 {
			structType = instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.TypeParameterNames()))
		}

		body.List = append([]ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{structType}}},
			},
		}, body.List...)

		body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}}})

		// Build the return type: *ClassName or *ClassName[T, U, ...]
		returnType := &ast.StarExpr{X: structType}

		constructorTypeParams := symbol.MergeTypeParams(ctx.currentClass.TypeParameters, ctx.localScope.TypeParameters)

		return []ast.Decl{GenFuncDeclWithTypeParams(
			ctx.localScope.Name,
			constructorTypeParams,
			ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList),
			&ast.FieldList{List: []*ast.Field{{Type: returnType}}},
			body,
		)}
	case "method_declaration", "abstract_method_declaration":
		var static bool

		// Store the annotations as comments on the method
		comments := []*ast.Comment{}

		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				switch modifier.Type() {
				case "static":
					static = true
				case "marker_annotation", "annotation":
					comments = append(comments, &ast.Comment{Text: "//" + modifier.Content(source)})
					// If the annotation was on the list of ignored annotations, don't
					// parse the method
					if _, in := excludedAnnotations[modifier.Content(source)]; in {
						return []ast.Decl{&ast.BadDecl{}}
					}
				}
			}
		}

		var receiver *ast.FieldList
		var receiverBaseType ast.Expr

		// If a function is non-static, it has a method receiver
		if !static {
			receiverBaseType = instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.TypeParameterNames()))
			receiver = &ast.FieldList{
				List: []*ast.Field{
					{
						Names: []*ast.Ident{{Name: ShortName(ctx.className)}},
						Type:  &ast.StarExpr{X: receiverBaseType},
					},
				},
			}
		}

		methodName := ParseExpr(node.ChildByFieldName("name"), source, ctx).(*ast.Ident)
		methodParameters := node.ChildByFieldName("parameters")

		comparison := func(d *symbol.Definition) bool {
			if d.OriginalName != methodName.Name {
				return false
			}
			if len(d.Parameters) != int(methodParameters.NamedChildCount()) {
				return false
			}
			for index, param := range nodeutil.NamedChildrenOf(methodParameters) {
				var paramType string
				if param.Type() == "spread_parameter" {
					paramType = param.NamedChild(0).Content(source)
				} else {
					paramType = param.ChildByFieldName("type").Content(source)
				}
				if d.Parameters[index].OriginalType != paramType {
					return false
				}
			}
			return true
		}

		methodDefinition := ctx.currentClass.FindMethod().By(comparison)

		if len(methodDefinition) == 0 {
			log.WithFields(log.Fields{
				"methodName": methodName.Name,
			}).Panic("No matching definition found for method")
		}

		ctx.localScope = methodDefinition[0]

		if ctx.currentClass.IsEnum && !static {
			params := ParseNode(methodParameters, source, ctx).(*ast.FieldList)
			var results *ast.FieldList
			if ctx.localScope.Type != "" {
				results = &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: ctx.localScope.Type}}}}
			}

			implDecls := []ast.Decl{}
			defaultImpl := ""
			if node.ChildByFieldName("body") != nil {
				defaultImpl = "_" + ctx.className + "_" + ctx.localScope.Name + "_default"
				implDecls = append(implDecls, buildEnumMethodImplementation(defaultImpl, node, ctx.localScope, ctx, source, receiverBaseType))
			}

			overrides := map[string]string{}
			for _, enumConst := range ctx.currentClass.EnumConstants {
				if enumConst.Body == nil {
					continue
				}
				for _, child := range enumConstantMethodDeclarations(enumConst.Body) {
					if !methodNodeMatchesDefinition(child, ctx.localScope, source) {
						continue
					}
					implName := "_" + ctx.className + "_" + enumConst.Name + "_" + ctx.localScope.Name
					implDecls = append(implDecls, buildEnumMethodImplementation(implName, child, ctx.localScope, ctx, source, receiverBaseType))
					overrides[enumConst.Name] = implName
					break
				}
			}

			wrapper := buildEnumMethodWrapper(ctx.localScope, overrides, defaultImpl, params, results, receiver, ctx)
			return append(implDecls, wrapper)
		}

		bodyNode := node.ChildByFieldName("body")
		if bodyNode == nil {
			return nil
		}

		body := ParseStmt(bodyNode, source, ctx).(*ast.BlockStmt)
		params := ParseNode(methodParameters, source, ctx).(*ast.FieldList)

		if methodName.Name == "main" {
			params = nil
			body.List = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "args"}},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.SelectorExpr{
							X:   &ast.Ident{Name: "os"},
							Sel: &ast.Ident{Name: "Args"},
						},
					},
				},
			}, body.List...)
		}

		var docGroup *ast.CommentGroup
		if len(comments) > 0 {
			docGroup = &ast.CommentGroup{List: comments}
		}

		results := &ast.FieldList{
			List: []*ast.Field{
				{Type: &ast.Ident{Name: ctx.localScope.Type}},
			},
		}

		if ctx.localScope.RequiresHelper {
			if receiverBaseType == nil {
				log.WithFields(log.Fields{
					"class":  ctx.className,
					"method": ctx.localScope.Name,
				}).Error("Receiver type missing for helper generation")
				return []ast.Decl{&ast.BadDecl{}}
			}
			return genInstanceGenericHelperDecls(ctx, ctx.localScope, docGroup, params, results, body, receiverBaseType)
		}

		funcDecl := &ast.FuncDecl{
			Doc:  docGroup,
			Name: &ast.Ident{Name: ctx.localScope.Name},
			Recv: receiver,
			Type: &ast.FuncType{
				Params:  params,
				Results: results,
			},
			Body: body,
		}
		if static {
			if len(ctx.localScope.TypeParameters) > 0 {
				funcDecl.Type.TypeParams = &ast.FieldList{List: makeTypeParamFields(ctx.localScope.TypeParameters)}
			}
		} else if len(ctx.localScope.TypeParameters) > 0 {
			log.WithFields(log.Fields{
				"class":  ctx.className,
				"method": ctx.localScope.Name,
			}).Warn("Instance methods with type parameters are not supported in Go; type parameters ignored")
		}
		return []ast.Decl{funcDecl}
	case "static_initializer":

		ctx.localScope = &symbol.Definition{}

		// A block of `static`, which is run before the main function
		return []ast.Decl{&ast.FuncDecl{
			Name: &ast.Ident{Name: "init"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{}},
			},
			Body: ParseStmt(node.NamedChild(0), source, ctx).(*ast.BlockStmt),
		}}
	}

	panic("Unknown node type for declaration: " + node.Type())
}
