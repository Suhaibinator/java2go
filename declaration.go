package main

import (
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
			case "constructor_declaration", "method_declaration", "static_initializer":
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
					case "constructor_declaration", "method_declaration", "static_initializer":
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
		// An enum is treated as a type alias (int) and a list of constants
		// that define the possible values the enum can have

		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name
		ctx.currentClass = ctx.currentFile.BaseClass

		declarations := []ast.Decl{}

		// Generate type declaration: type EnumName int
		declarations = append(declarations, &ast.GenDecl{
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: &ast.Ident{Name: ctx.className},
					Type: &ast.Ident{Name: "int"},
				},
			},
		})

		// Generate constants using iota
		if len(ctx.currentClass.EnumConstants) > 0 {
			constSpecs := []ast.Spec{}
			for i, constName := range ctx.currentClass.EnumConstants {
				spec := &ast.ValueSpec{
					Names: []*ast.Ident{{Name: constName}},
					Type:  &ast.Ident{Name: ctx.className},
				}
				if i == 0 {
					spec.Values = []ast.Expr{&ast.Ident{Name: "iota"}}
				}
				constSpecs = append(constSpecs, spec)
			}
			declarations = append(declarations, &ast.GenDecl{
				Tok:   token.CONST,
				Specs: constSpecs,
			})

			// Generate a values variable: var _enumNameValues = []EnumName{CONST1, CONST2, ...}
			valuesVarName := "_" + symbol.Lowercase(ctx.className) + "Values"
			constExprs := []ast.Expr{}
			for _, constName := range ctx.currentClass.EnumConstants {
				constExprs = append(constExprs, &ast.Ident{Name: constName})
			}
			declarations = append(declarations, &ast.GenDecl{
				Tok: token.VAR,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{{Name: valuesVarName}},
						Values: []ast.Expr{
							&ast.CompositeLit{
								Type: &ast.ArrayType{Elt: &ast.Ident{Name: ctx.className}},
								Elts: constExprs,
							},
						},
					},
				},
			})

			// Generate Values() function: func EnumNameValues() []EnumName { return _enumNameValues }
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: ctx.className + "Values"},
				Type: &ast.FuncType{
					Params: &ast.FieldList{},
					Results: &ast.FieldList{
						List: []*ast.Field{
							{Type: &ast.ArrayType{Elt: &ast.Ident{Name: ctx.className}}},
						},
					},
				},
				Body: &ast.BlockStmt{
					List: []ast.Stmt{
						&ast.ReturnStmt{
							Results: []ast.Expr{&ast.Ident{Name: valuesVarName}},
						},
					},
				},
			})
		}

		// Parse the enum body declarations (methods, constructors, etc.)
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		return declarations
	}
	panic("Unknown type to parse for decls: " + node.Type())
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

func genInstanceGenericHelperDecls(ctx Ctx, def *symbol.Definition, doc *ast.CommentGroup, params, results *ast.FieldList, body *ast.BlockStmt, receiverBaseType ast.Expr) []ast.Decl {
	classTypeParams := ctx.currentClass.TypeParameters
	combinedTypeParams := append([]string{}, classTypeParams...)
	combinedTypeParams = append(combinedTypeParams, def.TypeParameters...)

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

	helperTypeArgs := typeParamExprs(combinedTypeParams)
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
			// Create ClassName[T, U, ...] for generic structs
			typeParamExprs := make([]ast.Expr, len(ctx.currentClass.TypeParameters))
			for i, tp := range ctx.currentClass.TypeParameters {
				typeParamExprs[i] = &ast.Ident{Name: tp}
			}
			if len(typeParamExprs) == 1 {
				structType = &ast.IndexExpr{
					X:     &ast.Ident{Name: ctx.className},
					Index: typeParamExprs[0],
				}
			} else {
				structType = &ast.IndexListExpr{
					X:       &ast.Ident{Name: ctx.className},
					Indices: typeParamExprs,
				}
			}
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

		constructorTypeParams := append([]string{}, ctx.currentClass.TypeParameters...)
		if ctx.localScope != nil && len(ctx.localScope.TypeParameters) > 0 {
			constructorTypeParams = append(constructorTypeParams, ctx.localScope.TypeParameters...)
		}

		return []ast.Decl{GenFuncDeclWithTypeParams(
			ctx.localScope.Name,
			constructorTypeParams,
			ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList),
			&ast.FieldList{List: []*ast.Field{{Type: returnType}}},
			body,
		)}
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
					return []ast.Decl{&ast.BadDecl{}}
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
			receiverBaseType = instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.TypeParameters))
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

		body := ParseStmt(node.ChildByFieldName("body"), source, ctx).(*ast.BlockStmt)
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
				typeParamFields := make([]*ast.Field, len(ctx.localScope.TypeParameters))
				for i, tp := range ctx.localScope.TypeParameters {
					typeParamFields[i] = &ast.Field{
						Names: []*ast.Ident{{Name: tp}},
						Type:  &ast.Ident{Name: "any"},
					}
				}
				funcDecl.Type.TypeParams = &ast.FieldList{List: typeParamFields}
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
