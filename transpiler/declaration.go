package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

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

const (
	enumMetaNameField    = "enumName"
	enumMetaOrdinalField = "enumOrdinal"
	fieldInitMethodName  = "__java2goInitFields"
)

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
				// A class extending a built-in exception embeds the stdjava runtime
				// type so it inherits the Throwable method set and message storage.
				if builtin := stripJavaQualifier(t.Content(source)); isBuiltinExceptionType(builtin) && resolveClassScopeByQualifiedName(ctx, builtin) == nil {
					fields.List = append(fields.List, &ast.Field{Type: stdjavaQualifiedExpr(builtin, ctx)})
					continue
				}
				// A class extending java.lang.Thread embeds *stdjava.Thread so it
				// inherits Start()/Join(); the constructor wires the embedded Thread
				// to dispatch to this struct's Run() override.
				if super := stripJavaQualifier(t.Content(source)); super == "Thread" && resolveClassScopeByQualifiedName(ctx, super) == nil {
					fields.List = append(fields.List, &ast.Field{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Thread", ctx)}})
					continue
				}
				// When the superclass is a user-defined class (possibly a nested class
				// or a package-private one whose Go struct name was lowercased), embed
				// its resolved Go struct name so the field reference matches the struct
				// type and what the super() constructor call assigns. Without this the
				// embed would use the verbatim Java name (e.g. *Animal) while the type
				// is generated as `animal`.
				if base, _ := parseJavaTypeString(t.Content(source)); base != "" {
					if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil && scope.Class.Name != "" {
						fields.List = append(fields.List, &ast.Field{Type: &ast.StarExpr{X: &ast.Ident{Name: scope.Class.Name}}})
						continue
					}
				}
				fields.List = append(fields.List, &ast.Field{Type: astutil.ParseTypeWithTypeParams(t, source, typeParams)})
			}
		}

		if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := astutil.ParseTypeWithTypeParams(t, source, typeParams)
				if star, ok := embedType.(*ast.StarExpr); ok {
					embedType = star.X
				}
				if _, ok := embedType.(*ast.InterfaceType); ok {
					// Interface literals are not valid anonymous struct embeds in Go.
					// For well-known external interfaces (e.g. AutoCloseable), skip embed.
					continue
				}
				fields.List = append(fields.List, &ast.Field{Type: embedType})
			}
		}

		// Global variables
		globalVariables := &ast.GenDecl{Tok: token.VAR}
		instanceFieldInitializers := []ast.Stmt{}

		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name

		// Inner (non-static nested) classes hold an implicit reference to an
		// instance of their enclosing class. Model that as a leading struct field
		// so unqualified outer-member access and `outer.new Inner()` resolve.
		if enclField := buildEnclosingInstanceField(ctx); enclField != nil {
			fields.List = append([]*ast.Field{enclField}, fields.List...)
		}

		// First, look through the class's body for field declarations
		for _, child := range nodeutil.NamedChildrenOf(node.ChildByFieldName("body")) {
			if child.Type() == "field_declaration" {

				var staticField bool
				var skipField bool

				comments := []*ast.Comment{}

				// Handle any modifiers that the field might have
				if child.NamedChild(0).Type() == "modifiers" {
					for _, modifier := range nodeutil.UnnamedChildrenOf(child.NamedChild(0)) {
						switch modifier.Type() {
						case "static":
							staticField = true
						case "volatile":
							// Go has no field-level volatile. The visibility/ordering
							// guarantee is documented rather than enforced; callers
							// needing atomicity should use the sync/atomic helpers or a
							// mutex. Full atomic-field lowering would have to rewrite
							// every read/write site and is out of scope for this task.
							comments = append(comments, &ast.Comment{
								Text: "// volatile: Java visibility/ordering not enforced in Go; guard with sync/atomic or a mutex if shared across goroutines",
							})
						case "marker_annotation", "annotation":
							modContent := modifier.Content(source)
							comments = append(comments, &ast.Comment{Text: "//" + modContent})
							if excludedAnnotations[modContent] {
								// Skip this field if there is an ignored annotation
								skipField = true
							}
						}
					}
				}
				if skipField {
					continue
				}

				field := &ast.Field{}
				if len(comments) > 0 {
					field.Doc = &ast.CommentGroup{List: comments}
				}

				declarator := child.ChildByFieldName("declarator")
				if declarator == nil {
					continue
				}

				fieldName := declarator.ChildByFieldName("name").Content(source)
				fieldValueNode := declarator.ChildByFieldName("value")

				fieldDef := ctx.currentClass.FindField().ByOriginalName(fieldName)[0]

				field.Names = []*ast.Ident{{Name: fieldDef.Name}}
				field.Type = javaTypeStringToGoTypeExpr(fieldDef.OriginalType, typeParams, ctx)

				if staticField {
					spec := &ast.ValueSpec{Names: field.Names, Type: field.Type}
					if fieldValueNode != nil {
						valueCtx := ctx.Clone()
						valueCtx.localScope = &symbol.Definition{IsStatic: true}
						valueCtx.expectedType = fieldDef.OriginalType
						spec.Values = []ast.Expr{ParseExpr(fieldValueNode, source, valueCtx)}
					}
					globalVariables.Specs = append(globalVariables.Specs, spec)
				} else {
					fields.List = append(fields.List, field)
					if fieldValueNode != nil {
						valueCtx := ctx.Clone()
						valueCtx.expectedType = fieldDef.OriginalType
						instanceFieldInitializers = append(instanceFieldInitializers, &ast.AssignStmt{
							Lhs: []ast.Expr{
								&ast.SelectorExpr{
									X:   &ast.Ident{Name: ShortName(ctx.className)},
									Sel: &ast.Ident{Name: fieldDef.Name},
								},
							},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{
								ParseExpr(fieldValueNode, source, valueCtx),
							},
						})
					}
				}
			}
		}

		ctx.currentClass.HasInstanceFieldInitializers = len(instanceFieldInitializers) > 0

		// Add the global variables
		if len(globalVariables.Specs) > 0 {
			declarations = append(declarations, globalVariables)
		}

		// Add the struct for the class (with type parameters if present)
		declarations = append(declarations, GenStructWithTypeParams(ctx.className, fields, ctx.currentClass.TypeParameters))

		if helperDecl := buildInstanceFieldInitializerMethodDecl(ctx, instanceFieldInitializers); helperDecl != nil {
			declarations = append(declarations, helperDecl)
		}

		// Java provides an implicit no-arg constructor for any class that does not
		// declare one of its own. Without an explicit `New<Class>` function, calls
		// to `new Class()` have nothing to bind to, so synthesize one. This is
		// especially important for nested classes, which frequently omit
		// constructors.
		if ctorDecl := buildDefaultConstructorDecl(ctx); ctorDecl != nil {
			declarations = append(declarations, ctorDecl)
		}

		// Generate companion interface for abstract classes so that method
		// parameters typed as the abstract class preserve runtime type identity.
		if ctx.currentClass.IsAbstract {
			if ifaceDecl := generateAbstractClassInterface(ctx); ifaceDecl != nil {
				declarations = append(declarations, ifaceDecl)
			}
		}

		// Add all the declarations that appear in the class
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		// User-defined exception classes are wired into the stdjava hierarchy: a
		// ThrowableTypeName() override reports the class's own Java name (the
		// embedded runtime type would otherwise report the parent's), and an
		// init() registers the parent link so catch-by-supertype dispatch works.
		if parentName := exceptionSuperclassName(ctx, ctx.currentClass); parentName != "" {
			javaName := node.ChildByFieldName("name").Content(source)
			declarations = append(declarations,
				buildThrowableTypeNameMethod(ctx, javaName),
				buildExceptionRegistrationDecl(javaName, parentName, ctx),
			)
		}

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
				if _, ok := embedType.(*ast.InterfaceType); ok {
					continue
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

		declarations := []ast.Decl{GenInterface(interfaceName, methods, classTypeParams)}
		declarations = append(declarations, genFunctionalInterfaceAdapterDecls(interfaceName, methods, classTypeParams, ctx.currentClass)...)
		return declarations
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

		// Build struct fields for enum instances. Always include enum metadata.
		// to mirror Java's Enum metadata.
		fields := &ast.FieldList{
			List: []*ast.Field{
				{Names: []*ast.Ident{{Name: enumMetaNameField}}, Type: &ast.Ident{Name: "string"}},
				{Names: []*ast.Ident{{Name: enumMetaOrdinalField}}, Type: &ast.Ident{Name: "int"}},
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
				if _, ok := embedType.(*ast.InterfaceType); ok {
					continue
				}
				fields.List = append(fields.List, &ast.Field{Type: embedType})
			}
		}
		globalVariables := &ast.GenDecl{Tok: token.VAR}

		// Add declared fields from the enum body
		for _, fieldDef := range ctx.currentClass.Fields {
			field := &ast.Field{}
			field.Names = []*ast.Ident{{Name: fieldDef.Name}}
			field.Type = javaTypeStringToGoTypeExpr(fieldDef.OriginalType, typeParams, ctx)

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
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enumMetaNameField}}}}}},
			})

			// ordinal() accessor
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "ordinal")},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enumMetaOrdinalField}}}}}},
			})

			// compareTo(E)
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "compareTo")},
				Recv: receiver,
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "other"}}, Type: &ast.StarExpr{X: receiverBase}}}},
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int"}}}},
				},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.BinaryExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enumMetaOrdinalField}}, Op: token.SUB, Y: &ast.SelectorExpr{X: &ast.Ident{Name: "other"}, Sel: &ast.Ident{Name: enumMetaOrdinalField}}}}}}},
			})
		}

		// Parse the enum body declarations (methods, constructors, etc.)
		declarations = append(declarations, ParseDecls(node.ChildByFieldName("body"), source, ctx)...)

		return declarations
	}
	panic("Unknown type to parse for decls: " + node.Type())
}

func buildInstanceFieldInitializerMethodDecl(ctx Ctx, initializers []ast.Stmt) ast.Decl {
	if len(initializers) == 0 || ctx.currentClass == nil {
		return nil
	}

	receiverBaseType := instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.TypeParameterNames()))

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: fieldInitMethodName},
		Recv: &ast.FieldList{
			List: []*ast.Field{
				{
					Names: []*ast.Ident{{Name: ShortName(ctx.className)}},
					Type:  &ast.StarExpr{X: receiverBaseType},
				},
			},
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
		},
		Body: &ast.BlockStmt{List: initializers},
	}
}

// enclosingInstanceType returns the Go type expression (*Outer or *Outer[T,...])
// used for an inner class's enclosing-instance field/parameter, or nil if the
// scope is not an inner class with a resolvable enclosing class.
func enclosingInstanceType(scope *symbol.ClassScope) ast.Expr {
	if scope == nil || !scope.IsInner || scope.Enclosing == nil || scope.Enclosing.Class == nil {
		return nil
	}
	encl := scope.Enclosing
	base := instantiateGenericType(encl.Class.Name, typeParamExprs(encl.TypeParameterNames()))
	return &ast.StarExpr{X: base}
}

// buildEnclosingInstanceField returns the synthesized struct field that holds an
// inner class's enclosing instance, or nil for non-inner classes.
func buildEnclosingInstanceField(ctx Ctx) *ast.Field {
	scope := ctx.currentClass
	enclType := enclosingInstanceType(scope)
	if enclType == nil {
		return nil
	}
	return &ast.Field{
		Names: []*ast.Ident{{Name: scope.EnclosingFieldName()}},
		Type:  enclType,
	}
}

// classHasExplicitConstructor reports whether the class scope declares at least
// one constructor of its own.
func classHasExplicitConstructor(scope *symbol.ClassScope) bool {
	if scope == nil {
		return false
	}
	for _, method := range scope.Methods {
		if method != nil && method.Constructor {
			return true
		}
	}
	return false
}

// buildDefaultConstructorDecl synthesizes an implicit no-arg constructor for a
// class that declares none. The generated function mirrors the shape produced
// for explicit constructors: allocate the struct, run instance-field
// initializers, and return the pointer. Interfaces and enums are skipped since
// they are never instantiated through a `New<Class>` function.
func buildDefaultConstructorDecl(ctx Ctx) ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.IsInterface || scope.IsEnum {
		return nil
	}
	if classHasExplicitConstructor(scope) {
		return nil
	}

	typeParams := scope.TypeParameters
	var structType ast.Expr = &ast.Ident{Name: ctx.className}
	if len(typeParams) > 0 {
		structType = instantiateGenericType(ctx.className, typeParamExprs(scope.TypeParameterNames()))
	}

	recvName := ShortName(ctx.className)
	body := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: recvName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{structType}}},
		},
	}

	// A Thread subclass with no explicit constructor still needs its embedded
	// *stdjava.Thread wired so Start() dispatches to this instance's Run().
	if stmt := threadBaseWiringStmt(ctx); stmt != nil {
		body = append(body, stmt)
	}

	// Inner classes capture their enclosing instance through a synthesized
	// leading parameter that is stored into the enclosing-instance field.
	params := &ast.FieldList{}
	if enclType := enclosingInstanceType(scope); enclType != nil {
		enclFieldName := scope.EnclosingFieldName()
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{{Name: enclFieldName}},
			Type:  enclType,
		})
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: enclFieldName}}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: enclFieldName}},
		})
	}

	if scope.HasInstanceFieldInitializers {
		body = append(body, &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{Name: recvName},
					Sel: &ast.Ident{Name: fieldInitMethodName},
				},
			},
		})
	}

	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: recvName}}})

	constructorName := "New" + ctx.className
	returnType := &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: structType}}}}

	return GenFuncDeclWithTypeParams(constructorName, typeParams, params, returnType, &ast.BlockStmt{List: body})
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

// generateAbstractClassInterface creates a Go interface declaration from an
// abstract class's public methods.  The interface name is ClassName + "I".
// When used as a method parameter type this preserves the caller's runtime type
// so that instanceof / type-assertion checks work correctly.
func generateAbstractClassInterface(ctx Ctx) ast.Decl {
	scope := ctx.currentClass
	if scope == nil || !scope.IsAbstract {
		return nil
	}

	typeParams := scope.TypeParameterNames()
	methods := &ast.FieldList{}

	for _, method := range scope.Methods {
		if method.Constructor {
			continue
		}

		// Build parameter list
		params := &ast.FieldList{}
		for _, param := range method.Parameters {
			params.List = append(params.List, &ast.Field{
				Names: []*ast.Ident{{Name: param.Name}},
				Type:  javaTypeStringToGoTypeExpr(param.OriginalType, typeParams, ctx),
			})
		}

		// Build return type
		var results *ast.FieldList
		if method.OriginalType != "" && method.OriginalType != "void" {
			results = &ast.FieldList{
				List: []*ast.Field{{
					Type: javaTypeStringToGoTypeExpr(method.OriginalType, typeParams, ctx),
				}},
			}
		}

		methods.List = append(methods.List, &ast.Field{
			Names: []*ast.Ident{{Name: method.Name}},
			Type: &ast.FuncType{
				Params:  params,
				Results: results,
			},
		})
	}

	return GenInterface(ctx.className+"I", methods, scope.TypeParameters)
}

func buildAbstractMethodBody(methodName string, results *ast.FieldList) *ast.BlockStmt {
	panicMsg := "\"abstract method " + methodName + " not implemented\""
	stmts := []ast.Stmt{
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "panic"},
				Args: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: panicMsg}},
			},
		},
	}
	if results != nil && len(results.List) > 0 {
		stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{zeroValueForType(results.List[0].Type)}})
	}
	return &ast.BlockStmt{List: stmts}
}

func stmtAlwaysTerminates(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BlockStmt:
		if len(s.List) == 0 {
			return false
		}
		return stmtAlwaysTerminates(s.List[len(s.List)-1])
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		ident, ok := call.Fun.(*ast.Ident)
		return ok && ident.Name == "panic"
	case *ast.IfStmt:
		if s.Else == nil {
			return false
		}
		return stmtAlwaysTerminates(s.Body) && stmtAlwaysTerminates(s.Else)
	default:
		return false
	}
}

func bodyNeedsFallbackReturn(body *ast.BlockStmt) bool {
	if body == nil || len(body.List) == 0 {
		return true
	}
	return !stmtAlwaysTerminates(body.List[len(body.List)-1])
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
	if def != nil && strings.TrimSpace(def.OriginalType) != "" && strings.TrimSpace(def.OriginalType) != "void" {
		results = &ast.FieldList{
			List: []*ast.Field{
				{Type: javaTypeStringToGoTypeExpr(def.OriginalType, inScopeTypeParameters(ctx), ctx)},
			},
		}
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
			Tag:  &ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: enumMetaNameField}},
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

func cloneFieldList(fields *ast.FieldList) *ast.FieldList {
	if fields == nil {
		return nil
	}

	cloned := &ast.FieldList{
		Opening: fields.Opening,
		Closing: fields.Closing,
		List:    make([]*ast.Field, 0, len(fields.List)),
	}

	for _, field := range fields.List {
		if field == nil {
			continue
		}
		next := &ast.Field{
			Type: field.Type,
			Tag:  field.Tag,
		}
		if len(field.Names) > 0 {
			next.Names = make([]*ast.Ident, len(field.Names))
			for ind, name := range field.Names {
				if name == nil {
					continue
				}
				next.Names[ind] = &ast.Ident{Name: name.Name}
			}
		}
		cloned.List = append(cloned.List, next)
	}

	return cloned
}

func cloneFuncType(ft *ast.FuncType) *ast.FuncType {
	if ft == nil {
		return nil
	}
	return &ast.FuncType{
		Params:  cloneFieldList(ft.Params),
		Results: cloneFieldList(ft.Results),
	}
}

func methodCallArgs(params *ast.FieldList) []ast.Expr {
	if params == nil || len(params.List) == 0 {
		return nil
	}

	args := []ast.Expr{}
	nextUnnamed := 0

	for _, field := range params.List {
		if field == nil {
			continue
		}

		if len(field.Names) == 0 {
			generatedName := fmt.Sprintf("arg%d", nextUnnamed)
			nextUnnamed++
			field.Names = []*ast.Ident{{Name: generatedName}}
		}

		for _, name := range field.Names {
			if name == nil || name.Name == "" {
				continue
			}
			args = append(args, &ast.Ident{Name: name.Name})
		}
	}

	return args
}

func genFunctionalInterfaceAdapterDecls(interfaceName string, methods *ast.FieldList, typeParams []symbol.TypeParam, scope *symbol.ClassScope) []ast.Decl {
	if scope == nil {
		return nil
	}
	for _, field := range methods.List {
		if field == nil {
			continue
		}
		if len(field.Names) == 0 {
			// Extended/embedded interfaces can contribute additional abstract methods.
			// Restrict adapters to interfaces with exactly one explicitly declared method.
			return nil
		}
	}

	var samDef *symbol.Definition
	for _, method := range scope.Methods {
		if method == nil || method.IsStatic || method.Constructor {
			continue
		}
		if samDef != nil {
			return nil
		}
		samDef = method
	}
	if samDef == nil {
		return nil
	}

	var methodField *ast.Field
	for _, field := range methods.List {
		if field == nil || len(field.Names) == 0 {
			continue
		}
		if field.Names[0].Name == samDef.Name {
			methodField = field
			break
		}
	}
	if methodField == nil {
		return nil
	}

	methodType, ok := methodField.Type.(*ast.FuncType)
	if !ok || methodType == nil {
		return nil
	}

	adapterName := interfaceName + "FuncAdapter"
	typeParamNames := symbol.TypeParamNames(typeParams)
	typeArgs := typeParamExprs(typeParamNames)

	structFields := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "fn"}},
				Type:  cloneFuncType(methodType),
			},
		},
	}

	adapterStruct := GenStructWithTypeParams(adapterName, structFields, typeParams)

	adapterTypeExpr := instantiateGenericType(adapterName, typeArgs)
	interfaceTypeExpr := instantiateGenericType(interfaceName, typeArgs)

	receiverName := "fa"
	receiver := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: receiverName}},
				Type:  &ast.StarExpr{X: adapterTypeExpr},
			},
		},
	}

	methodParams := cloneFieldList(methodType.Params)
	methodResults := cloneFieldList(methodType.Results)
	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: "fn"},
		},
		Args: methodCallArgs(methodParams),
	}

	body := &ast.BlockStmt{}
	if methodResults != nil && len(methodResults.List) > 0 {
		body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{call}})
	} else {
		body.List = append(body.List, &ast.ExprStmt{X: call})
	}

	implMethod := &ast.FuncDecl{
		Name: &ast.Ident{Name: methodField.Names[0].Name},
		Recv: receiver,
		Type: &ast.FuncType{
			Params:  methodParams,
			Results: methodResults,
		},
		Body: body,
	}

	constructorName := "New" + adapterName
	constructorParams := &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "fn"}},
				Type:  cloneFuncType(methodType),
			},
		},
	}

	constructorResults := &ast.FieldList{
		List: []*ast.Field{
			{Type: interfaceTypeExpr},
		},
	}

	constructorBody := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					&ast.UnaryExpr{
						Op: token.AND,
						X: &ast.CompositeLit{
							Type: adapterTypeExpr,
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   &ast.Ident{Name: "fn"},
									Value: &ast.Ident{Name: "fn"},
								},
							},
						},
					},
				},
			},
		},
	}

	constructor := GenFuncDeclWithTypeParams(constructorName, typeParams, constructorParams, constructorResults, constructorBody)

	return []ast.Decl{adapterStruct, implMethod, constructor}
}

// buildEnumConstantInitializer constructs the Go expression used to initialize a single enum constant.
// It invokes a matching constructor if one exists, then injects the synthetic enum metadata fields
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
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "inst"}, Sel: &ast.Ident{Name: enumMetaNameField}}}, Tok: token.ASSIGN, Rhs: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: "\"" + enumConst.Name + "\""}}},
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: "inst"}, Sel: &ast.Ident{Name: enumMetaOrdinalField}}}, Tok: token.ASSIGN, Rhs: []ast.Expr{ordinal}},
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

func shouldCallFieldInitializerMethod(constructorBody []ast.Stmt, ctx Ctx) bool {
	if ctx.currentClass == nil || !ctx.currentClass.HasInstanceFieldInitializers {
		return false
	}
	return !startsWithThisConstructorInvocation(constructorBody, ctx.className)
}

// shouldCallFieldInitializerMethodForBody is like shouldCallFieldInitializerMethod
// but operates on the raw, parsed constructor body (before the synthesized
// allocation prelude is prepended), where a delegating this(...) call appears as
// the first statement.
func shouldCallFieldInitializerMethodForBody(userBody []ast.Stmt, ctx Ctx) bool {
	if ctx.currentClass == nil || !ctx.currentClass.HasInstanceFieldInitializers {
		return false
	}
	if len(userBody) == 0 {
		return true
	}
	return !isThisConstructorInvocation(userBody[0], ctx.className)
}

// isThisConstructorInvocation reports whether a statement is a delegating
// this(...) constructor call lowered to New<Class>(...).
func isThisConstructorInvocation(stmt ast.Stmt, className string) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	callExpr, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	funIdent, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return funIdent.Name == "New"+className
}

func startsWithThisConstructorInvocation(stmts []ast.Stmt, className string) bool {
	if len(stmts) < 2 {
		return false
	}

	exprStmt, ok := stmts[1].(*ast.ExprStmt)
	if !ok {
		return false
	}
	callExpr, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	funIdent, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	return funIdent.Name == "New"+className
}

func isExplicitSuperConstructorAssignment(stmt ast.Stmt, ctx Ctx) bool {
	if ctx.currentClass == nil {
		return false
	}
	assignStmt, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assignStmt.Lhs) != 1 {
		return false
	}

	lhsSelector, ok := assignStmt.Lhs[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	recvIdent, ok := lhsSelector.X.(*ast.Ident)
	if !ok || recvIdent.Name != ShortName(ctx.className) {
		return false
	}

	superType := strings.TrimSpace(ctx.currentClass.Superclass)
	if superType == "" {
		return false
	}
	base, _ := parseJavaTypeString(superType)
	superName := stripJavaQualifier(base)
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil && scope.Class.Name != "" {
		superName = scope.Class.Name
	}

	return lhsSelector.Sel.Name == superName
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

		prelude := []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{structType}}},
			},
		}
		// For an inner class, store the captured enclosing instance into its field
		// immediately after allocation so the rest of the constructor body can use
		// it.
		if enclosingInstanceType(ctx.currentClass) != nil {
			enclFieldName := ctx.currentClass.EnclosingFieldName()
			prelude = append(prelude, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enclFieldName}}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: enclFieldName}},
			})
		}
		// For a `class X extends Thread` subclass, wire the embedded *stdjava.Thread
		// to dispatch Start() to this instance's Run() override.
		if stmt := threadBaseWiringStmt(ctx); stmt != nil {
			prelude = append(prelude, stmt)
		}
		preludeLen := len(prelude)
		userBody := body.List
		body.List = append(prelude, body.List...)

		if shouldCallFieldInitializerMethodForBody(userBody, ctx) {
			initCall := &ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: ShortName(ctx.className)},
						Sel: &ast.Ident{Name: fieldInitMethodName},
					},
				},
			}

			// Insert right after the prelude (allocation + enclosing-instance
			// capture), unless the first user statement is an explicit super(...)
			// call, which must run before instance-field initializers.
			insertIndex := preludeLen
			if insertIndex < len(body.List) && isExplicitSuperConstructorAssignment(body.List[insertIndex], ctx) {
				insertIndex++
			}
			body.List = append(body.List[:insertIndex], append([]ast.Stmt{initCall}, body.List[insertIndex:]...)...)
		}

		body.List = append(body.List, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: ShortName(ctx.className)}}})

		// Build the return type: *ClassName or *ClassName[T, U, ...]
		returnType := &ast.StarExpr{X: structType}

		constructorTypeParams := symbol.MergeTypeParams(ctx.currentClass.TypeParameters, ctx.localScope.TypeParameters)

		constructorParams := ParseNode(node.ChildByFieldName("parameters"), source, ctx).(*ast.FieldList)
		// Inner-class constructors take the captured enclosing instance as a
		// leading parameter.
		if enclType := enclosingInstanceType(ctx.currentClass); enclType != nil {
			enclParam := &ast.Field{
				Names: []*ast.Ident{{Name: ctx.currentClass.EnclosingFieldName()}},
				Type:  enclType,
			}
			constructorParams.List = append([]*ast.Field{enclParam}, constructorParams.List...)
		}

		return []ast.Decl{GenFuncDeclWithTypeParams(
			ctx.localScope.Name,
			constructorTypeParams,
			constructorParams,
			&ast.FieldList{List: []*ast.Field{{Type: returnType}}},
			body,
		)}
	case "method_declaration", "abstract_method_declaration":
		var static bool
		var synchronizedMethod bool

		// Store the annotations as comments on the method
		comments := []*ast.Comment{}

		if node.NamedChild(0).Type() == "modifiers" {
			for _, modifier := range nodeutil.UnnamedChildrenOf(node.NamedChild(0)) {
				switch modifier.Type() {
				case "static":
					static = true
				case "synchronized":
					synchronizedMethod = true
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

		// Preserve a `throws` clause as documentation. Full error-return
		// translation is out of scope; the clause records the original contract.
		if throwsComment := throwsClauseComment(node, source); throwsComment != "" {
			comments = append(comments, &ast.Comment{Text: throwsComment})
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

		methodName := identFromNode(node.ChildByFieldName("name"), source)
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
			if strings.TrimSpace(ctx.localScope.OriginalType) != "" && strings.TrimSpace(ctx.localScope.OriginalType) != "void" {
				results = &ast.FieldList{
					List: []*ast.Field{
						{Type: javaTypeStringToGoTypeExpr(ctx.localScope.OriginalType, inScopeTypeParameters(ctx), ctx)},
					},
				}
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
		params := ParseNode(methodParameters, source, ctx).(*ast.FieldList)

		var results *ast.FieldList
		if strings.TrimSpace(ctx.localScope.OriginalType) != "" && strings.TrimSpace(ctx.localScope.OriginalType) != "void" {
			results = &ast.FieldList{
				List: []*ast.Field{
					{Type: javaTypeStringToGoTypeExpr(ctx.localScope.OriginalType, inScopeTypeParameters(ctx), ctx)},
				},
			}
		}

		var body *ast.BlockStmt
		if bodyNode != nil {
			body = ParseStmt(bodyNode, source, ctx).(*ast.BlockStmt)
		} else {
			body = buildAbstractMethodBody(ctx.localScope.OriginalName, results)
		}

		if methodName.Name == "main" && bodyNode != nil {
			params = nil
			argsAccess := qualifiedNameExpr("Args", "os", ctx)
			body.List = append([]ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "args"}},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{argsAccess},
				},
				&ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{&ast.Ident{Name: "args"}},
				},
			}, body.List...)
		}

		// A synchronized method holds its monitor for the whole body: the
		// receiver instance for an instance method, or a class-level token for a
		// static method. Prepend monitor enter + deferred exit.
		if synchronizedMethod && bodyNode != nil {
			body.List = append(synchronizedMethodPrologue(ctx, static), body.List...)
		}

		if results != nil && bodyNeedsFallbackReturn(body) {
			body.List = append(body.List, &ast.ReturnStmt{
				Results: []ast.Expr{zeroValueForType(results.List[0].Type)},
			})
		}

		var docGroup *ast.CommentGroup
		if len(comments) > 0 {
			docGroup = &ast.CommentGroup{List: comments}
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

	diag := reportUnsupported("declaration", node, source, ctx)
	// Emit a placeholder declaration carrying the diagnostic as a comment, so the
	// rest of the file can be converted.
	return []ast.Decl{unsupportedDeclStub(diag)}
}

// unsupportedDeclStub builds a declaration-level placeholder that records an
// unsupported construct via a leading `// UNSUPPORTED:` comment while remaining
// valid Go.
func unsupportedDeclStub(diag Diagnostic) ast.Decl {
	return &ast.GenDecl{
		Doc: &ast.CommentGroup{
			List: []*ast.Comment{{Text: unsupportedComment(diag)}},
		},
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{{Name: "_"}},
				Type:  &ast.Ident{Name: "struct{}"},
			},
		},
	}
}
