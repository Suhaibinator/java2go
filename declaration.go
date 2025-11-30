package main

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

// ParseDecls represents any type that returns a list of top-level declarations,
// this is any class, interface, or enum declaration
func ParseDecls(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	switch node.Type() {
	case "class_declaration":
		// TODO: Currently ignores implements and extends with the following tags:
		//"superclass"
		//"interfaces"

		// The declarations and fields for the class
		declarations := []ast.Decl{}
		fields := &ast.FieldList{}

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
							comments = append(comments, &ast.Comment{Text: "//" + modifier.Content(source)})
							if _, in := excludedAnnotations[modifier.Content(source)]; in {
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

				field.Names, field.Type = []*ast.Ident{&ast.Ident{Name: fieldDef.Name}}, &ast.Ident{Name: fieldDef.Type}

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

		// Add any type paramters defined in the class
		if node.ChildByFieldName("type_parameters") != nil {
			declarations = append(declarations, ParseDecls(node.ChildByFieldName("type_parameters"), source, ctx)...)
		}

		// Add the struct for the class
		declarations = append(declarations, GenStruct(ctx.className, fields))

		// Add all the declarations that appear in the class
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		return declarations
	case "class_body", "enum_body_declarations": // The body of the currently parsed class or enum
		decls := []ast.Decl{}

		// To switch to parsing the subclasses of a class, since we assume that
		// all the class's subclass definitions are in-order, if we find some number
		// of subclasses in a class, we can refer to them by index
		var subclassIndex int

		for _, child := range nodeutil.NamedChildrenOf(node) {
			switch child.Type() {
			// Skip fields and comments
			case "field_declaration", "comment", "line_comment", "block_comment":
			case "constructor_declaration", "method_declaration", "static_initializer":
				d := ParseDecl(child, source, ctx)
				// If the declaration is bad, skip it
				_, bad := d.(*ast.BadDecl)
				if !bad {
					decls = append(decls, d)
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
	case "interface_body":
		methods := &ast.FieldList{}

		for _, c := range nodeutil.NamedChildrenOf(node) {
			if c.Type() == "method_declaration" {
				parsedMethod := ParseNode(c, source, ctx).(*ast.Field)
				// If the method was ignored with an annotation, it will return a blank
				// field, so ignore that
				if parsedMethod.Type != nil {
					methods.List = append(methods.List, parsedMethod)
				}
			}
		}

		return []ast.Decl{GenInterface(ctx.className, methods)}
	case "interface_declaration":
		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name

		return ParseDecls(node.ChildByFieldName("body"), source, ctx)
	case "enum_declaration":
		// An enum is treated as both a struct, and a list of values that define
		// the states that the enum can be in
		// We parse it by creating a struct, then creating global variables for each constant

		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name
		declarations := []ast.Decl{}

		// Fields of the enum struct
		fields := &ast.FieldList{}

		// Add default fields (name, ordinal)
		// name string
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: "name"}},
			Type:  &ast.Ident{Name: "string"},
		})
		// ordinal int
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: "ordinal"}},
			Type:  &ast.Ident{Name: "int"},
		})

		enumBody := node.ChildByFieldName("body")
		var bodyDeclarations *sitter.Node

		// Parse user defined fields first
		for _, child := range nodeutil.NamedChildrenOf(enumBody) {
			if child.Type() == "enum_body_declarations" {
				bodyDeclarations = child
				for _, subChild := range nodeutil.NamedChildrenOf(child) {
					if subChild.Type() == "field_declaration" {
						// Logic similar to class_declaration field parsing
						// We can't reuse it easily because it's embedded in the big switch
						// But for enums we can assume similar handling.

						// Simplified field handling for Enums
						fieldType := ParseExpr(subChild.ChildByFieldName("type"), source, ctx)
						fieldName := subChild.ChildByFieldName("declarator").ChildByFieldName("name").Content(source)

						// Use symbol table to find correct name (resolved)
						fieldDef := ctx.currentClass.FindField().ByOriginalName(fieldName)[0]

						fields.List = append(fields.List, &ast.Field{
							Names: []*ast.Ident{{Name: fieldDef.Name}},
							Type:  fieldType,
						})
					}
				}
			}
		}

		// Emit Struct
		declarations = append(declarations, GenStruct(ctx.className, fields))

		// Add String() method
		declarations = append(declarations, &ast.FuncDecl{
			Recv: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: "c"}},
				Type:  &ast.StarExpr{X: &ast.Ident{Name: ctx.className}},
			}}},
			Name: &ast.Ident{Name: "String"},
			Type: &ast.FuncType{
				Results: &ast.FieldList{List: []*ast.Field{{
					Type: &ast.Ident{Name: "string"},
				}}},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "c"}, Sel: &ast.Ident{Name: "name"}}}},
				},
			},
		})

		// Parse Constants
		// var CompassValues []*Compass
		valuesVarName := fmt.Sprintf("_%sValues", ctx.className)
		valuesElements := []ast.Expr{}

		ordinal := 0
		for _, child := range nodeutil.NamedChildrenOf(enumBody) {
			if child.Type() == "enum_constant" {
				name := child.ChildByFieldName("name").Content(source)
				globalName := fmt.Sprintf("%s%s", ctx.className, name)

				// Find constructor
				args := []ast.Expr{}
				argTypes := []string{}

				argumentsNode := child.ChildByFieldName("arguments")
				if argumentsNode != nil {
					// In tree-sitter, arguments might be wrapped in argument_list
					if argumentsNode.Type() == "argument_list" {
						for _, arg := range nodeutil.NamedChildrenOf(argumentsNode) {
							args = append(args, ParseExpr(arg, source, ctx))
							argTypes = append(argTypes, symbol.TypeOfLiteral(arg, source)) // Approx type
						}
					}
				}

				// Resolve constructor name
				// Default to New + className
				constructorName := fmt.Sprintf("New%s", ctx.className)

				// Find correct constructor if possible
				classOriginalName := ctx.currentClass.Class.OriginalName
				comparison := func(d *symbol.Definition) bool {
					// Constructor must match class name
					if d.OriginalName != classOriginalName {
						return false
					}
					// Match parameter count
					if len(d.Parameters) != len(args) {
						return false
					}
					return true
				}
				defs := ctx.currentClass.FindMethod().By(comparison)
				if len(defs) > 0 {
					constructorName = defs[0].Name
				}

				initFunc := &ast.FuncLit{
					Type: &ast.FuncType{
						Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}}}},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.Ident{Name: "c"}},
								Tok: token.DEFINE,
								Rhs: []ast.Expr{&ast.CallExpr{
									Fun:  &ast.Ident{Name: constructorName},
									Args: args,
								}},
							},
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "c"}, Sel: &ast.Ident{Name: "name"}}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("\"%s\"", name)}},
							},
							&ast.AssignStmt{
								Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "c"}, Sel: &ast.Ident{Name: "ordinal"}}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", ordinal)}},
							},
							&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "c"}}},
						},
					},
				}

				declarations = append(declarations, &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{{Name: globalName}},
							Values: []ast.Expr{
								&ast.CallExpr{Fun: initFunc},
							},
						},
					},
				})

				valuesElements = append(valuesElements, &ast.Ident{Name: globalName})
				ordinal++
			}
		}

		// Values list
		declarations = append(declarations, &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{{Name: valuesVarName}},
					Values: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}},
							Elts: valuesElements,
						},
					},
				},
			},
		})

		// values() function
		// func CompassValuesFunc() []*Compass { return CompassValues }
		// Naming: "values" is a static method in Java. Compass.values().
		// In Go: CompassValues(). (Since we can't have static method on struct).
		declarations = append(declarations, &ast.FuncDecl{
			Name: &ast.Ident{Name: fmt.Sprintf("%sValues", ctx.className)},
			Type: &ast.FuncType{
				Results: &ast.FieldList{List: []*ast.Field{{
					Type: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}}},
				}}},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: valuesVarName}}},
				},
			},
		})

		// valueOf(name string) function
		// func CompassValueOf(name string) *Compass { ... }
		declarations = append(declarations, &ast.FuncDecl{
			Name: &ast.Ident{Name: fmt.Sprintf("%sValueOf", ctx.className)},
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Names: []*ast.Ident{{Name: "name"}},
					Type: &ast.Ident{Name: "string"},
				}}},
				Results: &ast.FieldList{List: []*ast.Field{{
					Type: &ast.StarExpr{X: &ast.Ident{Name: ctx.className}},
				}}},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.RangeStmt{
						Key: &ast.Ident{Name: "_"},
						Value: &ast.Ident{Name: "v"},
						Tok: token.DEFINE,
						X: &ast.Ident{Name: valuesVarName},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.IfStmt{
									Cond: &ast.BinaryExpr{
										X: &ast.SelectorExpr{X: &ast.Ident{Name: "v"}, Sel: &ast.Ident{Name: "name"}},
										Op: token.EQL,
										Y: &ast.Ident{Name: "name"},
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "v"}}},
										},
									},
								},
							},
						},
					},
					&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "nil"}}}, // Or panic? Java throws exception.
				},
			},
		})

		// Parse body declarations (methods etc)
		if bodyDeclarations != nil {
			declarations = append(declarations, ParseDecls(bodyDeclarations, source, ctx)...)
		}

		return declarations

	case "type_parameters":
		var declarations []ast.Decl

		// A list of generic type parameters
		for _, param := range nodeutil.NamedChildrenOf(node) {
			switch param.Type() {
			case "type_parameter":
				declarations = append(declarations, GenTypeInterface(param.NamedChild(0).Content(source), []string{"any"}))
			}
		}

		return declarations
	}
	panic("Unknown type to parse for decls: " + node.Type())
}

// ParseDecl parses a top-level declaration within a source file, including
// but not limited to fields and methods
func ParseDecl(node *sitter.Node, source []byte, ctx Ctx) ast.Decl {
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

		body.List = append([]ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{&ast.Ident{Name: ctx.className}}}},
			},
		}, body.List...)

		body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}}})

		return &ast.FuncDecl{
			Name: &ast.Ident{Name: ctx.localScope.Name},
			Type: &ast.FuncType{
				Params: ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList),
				Results: &ast.FieldList{List: []*ast.Field{&ast.Field{
					Type: &ast.Ident{Name: ctx.localScope.Type},
				}}},
			},
			Body: body,
		}
	case "method_declaration":
		var static bool

		// Store the annotations as comments on the method
		comments := []*ast.Comment{}

		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				switch modifier.Type() {
				case "static":
					static = true
				case "abstract":
					log.Warn("Unhandled abstract class")
					// TODO: Handle abstract methods correctly
					return &ast.BadDecl{}
				case "marker_annotation", "annotation":
					comments = append(comments, &ast.Comment{Text: "//" + modifier.Content(source)})
					// If the annotation was on the list of ignored annotations, don't
					// parse the method
					if _, in := excludedAnnotations[modifier.Content(source)]; in {
						return &ast.BadDecl{}
					}
				}
			}
		}

		var receiver *ast.FieldList

		// If a function is non-static, it has a method receiver
		if !static {
			receiver = &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{&ast.Ident{Name: ShortName(ctx.className)}},
						Type:  &ast.StarExpr{X: &ast.Ident{Name: ctx.className}},
					},
				},
			}
		}

		methodName := ParseExpr(node.ChildByFieldName("name"), source, ctx).(*ast.Ident)

		methodParameters := node.ChildByFieldName("parameters")

		// Find the declaration for the method that we are defining

		// Find a method that is more or less exactly the same
		comparison := func(d *symbol.Definition) bool {
			// Throw out any methods that aren't named the same
			if d.OriginalName != methodName.Name {
				return false
			}

			// Now, even though the method might have the same name, it could be overloaded,
			// so we have to check the parameters as well

			// Number of parameters are not the same, invalid
			if len(d.Parameters) != int(methodParameters.NamedChildCount()) {
				return false
			}

			// Go through the types and check to see if they differ
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

			// We found the correct method
			return true
		}

		methodDefinition := ctx.currentClass.FindMethod().By(comparison)

		// No definition was found
		if len(methodDefinition) == 0 {
			log.WithFields(log.Fields{
				"methodName": methodName.Name,
			}).Panic("No matching definition found for method")
		}

		ctx.localScope = methodDefinition[0]

		body := ParseStmt(node.ChildByFieldName("body"), source, ctx).(*ast.BlockStmt)

		params := ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList)

		// Special case for the main method, because in Java, this method has the
		// command line args passed in as a parameter
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

		return &ast.FuncDecl{
			Doc:  &ast.CommentGroup{List: comments},
			Name: &ast.Ident{Name: ctx.localScope.Name},
			Recv: receiver,
			Type: &ast.FuncType{
				Params: params,
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{Type: &ast.Ident{Name: ctx.localScope.Type}},
					},
				},
			},
			Body: body,
		}
	case "static_initializer":

		ctx.localScope = &symbol.Definition{}

		// A block of `static`, which is run before the main function
		return &ast.FuncDecl{
			Name: &ast.Ident{Name: "init"},
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{}},
			},
			Body: ParseStmt(node.NamedChild(0), source, ctx).(*ast.BlockStmt),
		}
	}

	panic("Unknown node type for declaration: " + node.Type())
}
