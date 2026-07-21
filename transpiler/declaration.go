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
	enumMetaNameField       = "enumName"
	enumMetaOrdinalField    = "enumOrdinal"
	fieldInitMethodName     = "__java2goInitFields"
	interfaceDefaultsSuffix = "Java2goDefaults"
	classDispatchSuffix     = "Java2goDispatch"
	constructorSelfSuffix   = "Java2goWithSelf"
	constructorSelfParam    = "__java2goMostDerived"
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
	case "record_declaration":
		return parseRecordDecls(node, source, ctx)
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
						fields.List = append(fields.List, &ast.Field{Type: javaTypeStringToGoTypeExpr(
							t.Content(source),
							typeParams,
							ctx,
						)})
						continue
					}
				}
				fields.List = append(fields.List, &ast.Field{Type: astutil.ParseTypeWithTypeParams(t, source, typeParams)})
			}
		}

		if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := implementedInterfaceTypeExpr(t.Content(source), typeParams, ctx)
				if embedType == nil {
					// Interface literals are not valid anonymous struct embeds in Go.
					// For well-known external interfaces (e.g. AutoCloseable), skip embed.
					continue
				}
				fields.List = append(fields.List, &ast.Field{Type: embedType})
			}
		}

		if classNeedsVirtualDispatch(ctx.currentClass, ctx) {
			fields.List = append(fields.List, &ast.Field{
				Names: []*ast.Ident{{Name: classDispatchFieldName(ctx.currentClass)}},
				Type:  classDispatchTypeExpr(ctx.currentClass),
			})
		}

		// Global variables
		globalVariables := &ast.GenDecl{Tok: token.VAR}
		instanceFieldInitializers := []ast.Stmt{}
		classBody := node.ChildByFieldName("body")
		consolidateStaticInitialization := classBodyHasStaticInitializer(classBody)

		ctx.className = ctx.currentFile.FindClass(node.ChildByFieldName("name").Content(source)).Name

		// Inner (non-static nested) classes hold an implicit reference to an
		// instance of their enclosing class. Model that as a leading struct field
		// so unqualified outer-member access and `outer.new Inner()` resolve.
		if enclField := buildEnclosingInstanceField(ctx); enclField != nil {
			fields.List = append([]*ast.Field{enclField}, fields.List...)
		}

		// First, look through the class's body for field declarations
		for _, child := range nodeutil.NamedChildrenOf(classBody) {
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
				field.Type = abstractClassToInterface(
					javaTypeStringToGoTypeExpr(fieldDef.OriginalType, typeParams, ctx),
					fieldDef.OriginalType,
					ctx,
				)

				if staticField {
					spec := &ast.ValueSpec{Names: field.Names, Type: field.Type}
					if fieldValueNode != nil && !consolidateStaticInitialization {
						valueCtx := ctx.Clone()
						valueCtx.localScope = &symbol.Definition{IsStatic: true}
						valueCtx.expectedType = fieldDef.OriginalType
						valueCtx.expectedTypeRoot = fieldValueNode
						spec.Values = []ast.Expr{ParseExpr(fieldValueNode, source, valueCtx)}
					}
					globalVariables.Specs = append(globalVariables.Specs, spec)
				} else {
					fields.List = append(fields.List, field)
					if fieldValueNode != nil {
						valueCtx := ctx.Clone()
						valueCtx.expectedType = fieldDef.OriginalType
						valueCtx.expectedTypeRoot = fieldValueNode
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

		// Add the class's virtual-dispatch contract before the struct. Go permits
		// either declaration order, but keeping the pair adjacent makes generated
		// inheritance machinery easier to inspect.
		if dispatchDecl := generateClassDispatchInterface(ctx); dispatchDecl != nil {
			declarations = append(declarations, dispatchDecl)
		}

		// Add the struct for the class (with type parameters if present)
		declarations = append(declarations, genStructWithTypeParamsInContext(ctx.className, fields, ctx.currentClass.TypeParameters, ctx))
		if setterDecl := generateClassSelfSetter(ctx); setterDecl != nil {
			declarations = append(declarations, setterDecl)
		}
		declarations = append(declarations, generateAffineArrayViewDecls(ctx)...)

		if helperDecl := buildInstanceFieldInitializerMethodDecl(ctx, instanceFieldInitializers); helperDecl != nil {
			declarations = append(declarations, helperDecl)
		}

		// Java provides an implicit no-arg constructor for any class that does not
		// declare one of its own. Without an explicit `New<Class>` function, calls
		// to `new Class()` have nothing to bind to, so synthesize one. This is
		// especially important for nested classes, which frequently omit
		// constructors.
		declarations = append(declarations, buildDefaultConstructorDecls(ctx)...)

		// Generate companion interface for abstract classes so that method
		// parameters typed as the abstract class preserve runtime type identity.
		if ctx.currentClass.IsAbstract {
			if ifaceDecl := generateAbstractClassInterface(ctx); ifaceDecl != nil {
				declarations = append(declarations, ifaceDecl)
			}
		}

		// Go evaluates every package variable before any init function. Java instead
		// executes static field initializers and static blocks in their exact source
		// order. When a class contains a static block, move all of that class's
		// explicit static field values into one ordered init function. Classes with
		// fields only retain direct Go initializers, preserving compact output and
		// compile-time dependency handling.
		if consolidateStaticInitialization {
			if initDecl := buildOrderedStaticInitializationDecl(classBody, source, ctx); initDecl != nil {
				declarations = append(declarations, initDecl)
			}
		}

		// Add all the declarations that appear in the class
		declarations = append(declarations, parseClassBodyDeclarations(classBody, source, ctx, consolidateStaticInitialization)...)

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
		return parseClassBodyDeclarations(node, source, ctx, false)
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
				embedType := javaTypeStringToGoTypeExpr(t.Content(source), typeParams, ctx)
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

		declarations := []ast.Decl{genInterfaceInContext(interfaceName, methods, classTypeParams, ctx)}
		declarations = append(declarations, generateInterfaceDefaultMethodDecls(node, source, ctx)...)
		declarations = append(declarations, genFunctionalInterfaceAdapterDecls(interfaceName, methods, classTypeParams, ctx.currentClass, ctx)...)
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
				{Names: []*ast.Ident{{Name: enumMetaOrdinalField}}, Type: &ast.Ident{Name: "int32"}},
			},
		}

		// Embed implemented interfaces
		typeParams := ctx.currentClass.TypeParameterNames()
		if interfacesNode := node.ChildByFieldName("interfaces"); interfacesNode != nil {
			for _, t := range collectTypeNodes(interfacesNode) {
				embedType := implementedInterfaceTypeExpr(t.Content(source), typeParams, ctx)
				if embedType == nil {
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
		declarations = append(declarations, genStructWithTypeParamsInContext(ctx.className, fields, ctx.currentClass.TypeParameters, ctx))

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
			receiverName := ShortName(ctx.className)
			enumStringResult := ast.Expr(&ast.SelectorExpr{X: &ast.Ident{Name: receiverName}, Sel: &ast.Ident{Name: enumMetaNameField}})
			if toString := findEnumToStringMethod(ctx.currentClass); toString != nil {
				enumStringResult = &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: receiverName},
						Sel: &ast.Ident{Name: toString.Name},
					},
				}
			}

			// name() accessor
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "name")},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enumMetaNameField}}}}}},
			})

			// String implements fmt.Stringer so every Go formatting path used for
			// println, string concatenation, and String.valueOf observes Java's
			// default Enum.toString() result instead of the backing Go struct.
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: "String"},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.IfStmt{
						Cond: &ast.BinaryExpr{X: &ast.Ident{Name: receiverName}, Op: token.EQL, Y: &ast.Ident{Name: "nil"}},
						Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.BasicLit{Kind: token.STRING, Value: `"null"`}}}}},
					},
					&ast.ReturnStmt{Results: []ast.Expr{enumStringResult}},
				}},
			})

			// ordinal() accessor
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "ordinal")},
				Recv: receiver,
				Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int32"}}}}},
				Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: ShortName(ctx.className)}, Sel: &ast.Ident{Name: enumMetaOrdinalField}}}}}},
			})

			// compareTo(E)
			declarations = append(declarations, &ast.FuncDecl{
				Name: &ast.Ident{Name: symbol.HandleExportStatus(true, "compareTo")},
				Recv: receiver,
				Type: &ast.FuncType{
					Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: "other"}}, Type: &ast.StarExpr{X: receiverBase}}}},
					Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int32"}}}},
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

func parseClassBodyDeclarations(node *sitter.Node, source []byte, ctx Ctx, skipStaticInitializers bool) []ast.Decl {
	decls := []ast.Decl{}

	// Nested class scopes are stored in the same source order as their syntax.
	var subclassIndex int
	for _, child := range nodeutil.NamedChildrenOf(node) {
		switch child.Type() {
		// Fields and enum constants are handled by their enclosing declaration.
		case "field_declaration", "comment", "enum_constant":
		case "constructor_declaration", "method_declaration", "abstract_method_declaration":
			decls = appendValidDeclarations(decls, ParseDecl(child, source, ctx))
		case "static_initializer":
			if !skipStaticInitializers {
				decls = appendValidDeclarations(decls, ParseDecl(child, source, ctx))
			}
		case "enum_body_declarations":
			for _, declChild := range nodeutil.NamedChildrenOf(child) {
				switch declChild.Type() {
				case "constructor_declaration", "method_declaration", "abstract_method_declaration":
					decls = appendValidDeclarations(decls, ParseDecl(declChild, source, ctx))
				case "static_initializer":
					if !skipStaticInitializers {
						decls = appendValidDeclarations(decls, ParseDecl(declChild, source, ctx))
					}
				}
			}
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			newCtx := ctx.Clone()
			newCtx.currentClass = ctx.currentClass.Subclasses[subclassIndex]
			subclassIndex++
			decls = append(decls, ParseDecls(child, source, newCtx)...)
		}
	}
	return decls
}

func appendValidDeclarations(dst []ast.Decl, declarations []ast.Decl) []ast.Decl {
	for _, declaration := range declarations {
		if _, bad := declaration.(*ast.BadDecl); !bad {
			dst = append(dst, declaration)
		}
	}
	return dst
}

func classBodyHasStaticInitializer(body *sitter.Node) bool {
	if body == nil {
		return false
	}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		if child.Type() == "static_initializer" {
			return true
		}
	}
	return false
}

func fieldDeclarationIsStatic(node *sitter.Node) bool {
	if node == nil || node.Type() != "field_declaration" || node.NamedChildCount() == 0 {
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

func buildOrderedStaticInitializationDecl(body *sitter.Node, source []byte, ctx Ctx) ast.Decl {
	if body == nil || ctx.currentClass == nil {
		return nil
	}

	statements := []ast.Stmt{}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		switch child.Type() {
		case "field_declaration":
			if !fieldDeclarationIsStatic(child) {
				continue
			}
			declarator := child.ChildByFieldName("declarator")
			if declarator == nil {
				continue
			}
			valueNode := declarator.ChildByFieldName("value")
			nameNode := declarator.ChildByFieldName("name")
			if valueNode == nil || nameNode == nil {
				continue
			}
			fieldDefinitions := ctx.currentClass.FindField().ByOriginalName(nameNode.Content(source))
			if len(fieldDefinitions) == 0 {
				continue
			}
			fieldDefinition := fieldDefinitions[0]
			valueCtx := ctx.Clone()
			valueCtx.localScope = &symbol.Definition{IsStatic: true}
			valueCtx.expectedType = fieldDefinition.OriginalType
			valueCtx.expectedTypeRoot = valueNode
			statements = append(statements, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: fieldDefinition.Name}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{ParseExpr(valueNode, source, valueCtx)},
			})
		case "static_initializer":
			staticCtx := ctx.Clone()
			staticCtx.localScope = &symbol.Definition{IsStatic: true}
			block, ok := ParseStmt(child.NamedChild(0), source, staticCtx).(*ast.BlockStmt)
			if ok && block != nil {
				statements = append(statements, block.List...)
			}
		}
	}

	if len(statements) == 0 {
		return nil
	}
	return &ast.FuncDecl{
		Name: &ast.Ident{Name: "init"},
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: &ast.BlockStmt{List: statements},
	}
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

// implementedInterfaceTypeExpr lowers the type named in an implements clause
// to an embeddable Go interface type. Using the string-based, symbol-aware type
// converter is important here: it preserves generic arguments, recognizes type
// parameters from the implementing class, applies generated name casing, and
// qualifies types that live in another generated package.
func implementedInterfaceTypeExpr(javaType string, typeParams []string, ctx Ctx) ast.Expr {
	base, _ := parseJavaTypeString(javaType)
	// Runtime interfaces are satisfied structurally by the generated methods.
	// Embedding stdjava.Runnable would add an unnecessary interface field to every
	// anonymous Runnable implementation and changes its zero-value behavior.
	if stripJavaQualifier(base) == "Runnable" && resolveClassScopeByQualifiedName(ctx, base) == nil {
		return nil
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); interfaceHasDefaultMethods(scope, ctx) {
		return interfaceDefaultCarrierTypeExpr(javaType, typeParams, ctx)
	}

	embedType := javaTypeStringToGoTypeExpr(javaType, typeParams, ctx)
	if star, ok := embedType.(*ast.StarExpr); ok {
		// Interfaces are embedded by value. The converter only leaves a pointer
		// here when the interface could not be resolved, matching the historical
		// fallback for an implements clause.
		embedType = star.X
	}
	if _, ok := embedType.(*ast.InterfaceType); ok {
		// Interface literals cannot be anonymous struct fields. This is how
		// well-known external interfaces such as AutoCloseable are represented.
		return nil
	}
	return embedType
}

// interfaceHasDefaultMethods reports whether an interface contributes concrete
// instance methods. Go interfaces only describe a method set, so Java default
// bodies are carried by a generated embedded implementation instead of a nil
// interface field.
func interfaceHasDefaultMethods(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]struct{}{}
	var hasDefaults func(*symbol.ClassScope, Ctx) bool
	hasDefaults = func(current *symbol.ClassScope, currentCtx Ctx) bool {
		if current == nil || !current.IsInterface {
			return false
		}
		if _, duplicate := seen[current]; duplicate {
			return false
		}
		seen[current] = struct{}{}
		for _, method := range current.Methods {
			if method != nil && !method.IsStatic && !method.IsPrivate && !method.Constructor && method.HasBody {
				return true
			}
		}
		if ownerFile := findFileScopeForClassScope(current); ownerFile != nil {
			currentCtx.currentFile = ownerFile
			currentCtx.currentClass = current
		}
		for _, parentType := range current.ImplementedInterfaces {
			base, _ := parseJavaTypeString(parentType)
			if hasDefaults(resolveClassScopeByQualifiedName(currentCtx, base), currentCtx) {
				return true
			}
		}
		return false
	}
	return hasDefaults(scope, ctx)
}

func interfaceDefaultCarrierName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	return scope.Class.Name + interfaceDefaultsSuffix
}

// interfaceDefaultCarrierTypeExpr preserves the interface's generic arguments
// while replacing its name with the generated default-method carrier type.
func interfaceDefaultCarrierTypeExpr(javaType string, typeParams []string, ctx Ctx) ast.Expr {
	base, typeArgs := parseJavaTypeString(javaType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if !interfaceHasDefaultMethods(scope, ctx) {
		return nil
	}
	carrier := qualifiedNameExpr(
		interfaceDefaultCarrierName(scope),
		resolveJavaPackageForType(ctx, base, scope),
		ctx,
	)
	if len(typeArgs) > 0 {
		args := make([]ast.Expr, 0, len(typeArgs))
		for _, arg := range typeArgs {
			args = append(args, javaTypeStringToGoTypeExpr(arg, typeParams, ctx))
		}
		carrier = applyTypeArguments(carrier, args)
	}
	return &ast.StarExpr{X: carrier}
}

func interfaceMethodSignature(def *symbol.Definition) string {
	if def == nil {
		return ""
	}
	parts := make([]string, 0, len(def.Parameters)+1)
	parts = append(parts, def.OriginalName)
	for _, param := range def.Parameters {
		parts = append(parts, param.OriginalType)
	}
	return strings.Join(parts, "\x00")
}

func collectInterfaceDefaultMethods(scope *symbol.ClassScope, ctx Ctx, seen map[*symbol.ClassScope]struct{}) []*symbol.Definition {
	if scope == nil || !scope.IsInterface {
		return nil
	}
	if _, duplicate := seen[scope]; duplicate {
		return nil
	}
	seen[scope] = struct{}{}
	methods := []*symbol.Definition{}
	known := map[string]struct{}{}
	for _, method := range scope.Methods {
		if method == nil || method.IsStatic || method.IsPrivate || method.Constructor || !method.HasBody {
			continue
		}
		methods = append(methods, method)
		known[interfaceMethodSignature(method)] = struct{}{}
	}
	if ownerFile := findFileScopeForClassScope(scope); ownerFile != nil {
		ctx.currentFile = ownerFile
		ctx.currentClass = scope
	}
	for _, parentType := range scope.ImplementedInterfaces {
		base, _ := parseJavaTypeString(parentType)
		for _, inherited := range collectInterfaceDefaultMethods(resolveClassScopeByQualifiedName(ctx, base), ctx, seen) {
			key := interfaceMethodSignature(inherited)
			if _, exists := known[key]; exists {
				continue
			}
			known[key] = struct{}{}
			methods = append(methods, inherited)
		}
	}
	return methods
}

func buildInheritedInterfaceDefaultForwarder(
	carrierName string,
	parentType string,
	parentScope *symbol.ClassScope,
	def *symbol.Definition,
	childScope *symbol.ClassScope,
	ctx Ctx,
) ast.Decl {
	if def == nil || parentScope == nil || childScope == nil || def.RequiresHelper {
		return nil
	}
	typeBindings := map[string]string{}
	_, parentArgs := parseJavaTypeString(parentType)
	for i, paramName := range parentScope.TypeParameterNames() {
		if i < len(parentArgs) {
			typeBindings[paramName] = parentArgs[i]
		}
	}
	mapType := func(javaType string) ast.Expr {
		javaType = substituteJavaTypeParameters(javaType, typeBindings)
		return javaTypeStringToGoTypeExpr(javaType, childScope.TypeParameterNames(), ctx)
	}

	params := &ast.FieldList{}
	args := []ast.Expr{}
	for _, param := range def.Parameters {
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{{Name: param.Name}},
			Type:  mapType(param.OriginalType),
		})
		args = append(args, &ast.Ident{Name: param.Name})
	}
	var results *ast.FieldList
	if strings.TrimSpace(def.OriginalType) != "" && strings.TrimSpace(def.OriginalType) != "void" {
		results = &ast.FieldList{List: []*ast.Field{{Type: mapType(def.OriginalType)}}}
	}
	recvName := ShortName(carrierName)
	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.SelectorExpr{
				X:   &ast.Ident{Name: recvName},
				Sel: &ast.Ident{Name: interfaceDefaultCarrierName(parentScope)},
			},
			Sel: &ast.Ident{Name: def.Name},
		},
		Args: args,
	}
	body := &ast.BlockStmt{}
	if results != nil {
		body.List = []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}
	} else {
		body.List = []ast.Stmt{&ast.ExprStmt{X: call}}
	}
	recvType := instantiateGenericType(carrierName, typeParamExprs(childScope.TypeParameterNames()))
	return &ast.FuncDecl{
		Name: &ast.Ident{Name: def.Name},
		Recv: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: recvName}},
			Type:  &ast.StarExpr{X: recvType},
		}}},
		Type: &ast.FuncType{Params: params, Results: results},
		Body: body,
	}
}

// generateInterfaceDefaultMethodDecls materializes Java interface default
// methods in a small carrier struct. Implementing classes embed an initialized
// carrier, while its embedded interface value points back at the concrete Java
// object. Calls made by a default body therefore retain normal virtual dispatch
// (for example, default shout() calling an overridden greet()).
func generateInterfaceDefaultMethodDecls(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if node == nil || !interfaceHasDefaultMethods(scope, ctx) || scope.Class == nil {
		return nil
	}

	carrierName := interfaceDefaultCarrierName(scope)
	typeParamNames := scope.TypeParameterNames()
	interfaceType := instantiateGenericType(scope.Class.Name, typeParamExprs(typeParamNames))
	carrierType := instantiateGenericType(carrierName, typeParamExprs(typeParamNames))

	fields := &ast.FieldList{List: []*ast.Field{{Type: interfaceType}}}
	parentDefaultTypes := defaultInterfaceTypes(scope, ctx)
	for _, parentType := range parentDefaultTypes {
		if carrierType := interfaceDefaultCarrierTypeExpr(parentType, typeParamNames, ctx); carrierType != nil {
			fields.List = append(fields.List, &ast.Field{Type: carrierType})
		}
	}
	decls := []ast.Decl{genStructWithTypeParamsInContext(carrierName, fields, scope.TypeParameters, ctx)}

	selfName := "self"
	constructorValues := []ast.Expr{&ast.Ident{Name: selfName}}
	for _, parentType := range parentDefaultTypes {
		if constructor := interfaceDefaultCarrierConstructorExpr(parentType, typeParamNames, ctx); constructor != nil {
			constructorValues = append(constructorValues, &ast.CallExpr{
				Fun:  constructor,
				Args: []ast.Expr{&ast.Ident{Name: selfName}},
			})
		}
	}
	constructorBody := &ast.BlockStmt{List: []ast.Stmt{
		&ast.ReturnStmt{Results: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X: &ast.CompositeLit{
					Type: carrierType,
					Elts: constructorValues,
				},
			},
		}},
	}}
	decls = append(decls, genFuncDeclWithTypeParamsInContext(
		defaultConstructorName(carrierName),
		scope.TypeParameters,
		&ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: selfName}}, Type: interfaceType}}},
		&ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: carrierType}}}},
		constructorBody,
		ctx,
	))

	body := node.ChildByFieldName("body")
	directMethods := map[string]struct{}{}
	for _, method := range scope.Methods {
		directMethods[interfaceMethodSignature(method)] = struct{}{}
	}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		if child.Type() != "method_declaration" || child.ChildByFieldName("body") == nil {
			continue
		}
		methodCtx := ctx.Clone()
		methodCtx.className = carrierName
		decls = append(decls, ParseDecl(child, source, methodCtx)...)
	}
	forwarded := map[string]struct{}{}
	for _, parentType := range parentDefaultTypes {
		base, _ := parseJavaTypeString(parentType)
		parentScope := resolveClassScopeByQualifiedName(ctx, base)
		for _, method := range collectInterfaceDefaultMethods(parentScope, ctx, map[*symbol.ClassScope]struct{}{}) {
			key := interfaceMethodSignature(method)
			if _, overridden := directMethods[key]; overridden {
				continue
			}
			if _, duplicate := forwarded[key]; duplicate {
				continue
			}
			forwarded[key] = struct{}{}
			if forwarder := buildInheritedInterfaceDefaultForwarder(carrierName, parentType, parentScope, method, scope, ctx); forwarder != nil {
				decls = append(decls, forwarder)
			}
		}
	}
	return decls
}

func classDispatchTypeName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	return scope.Class.Name + classDispatchSuffix
}

func classDispatchFieldName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	name := "java2go" + symbol.Uppercase(scope.Class.Name) + "Self"
	if scope.Class.Name == symbol.Uppercase(scope.Class.Name) {
		return symbol.Uppercase(name)
	}
	return "__" + name
}

func classSelfSetterName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	name := "java2goSet" + symbol.Uppercase(scope.Class.Name) + "Self"
	if scope.Class.Name == symbol.Uppercase(scope.Class.Name) {
		name = symbol.Uppercase(name)
	}
	return name
}

func classDispatchTypeExpr(scope *symbol.ClassScope) ast.Expr {
	if scope == nil {
		return &ast.InterfaceType{Methods: &ast.FieldList{}}
	}
	return instantiateGenericType(classDispatchTypeName(scope), typeParamExprs(scope.TypeParameterNames()))
}

func visitClassScopes(scope *symbol.ClassScope, visit func(*symbol.ClassScope) bool) bool {
	if scope == nil {
		return false
	}
	if visit(scope) {
		return true
	}
	for _, nested := range scope.Subclasses {
		if visitClassScopes(nested, visit) {
			return true
		}
	}
	return false
}

// classHasKnownSubclass uses the complete symbol graph for the conversion run,
// rather than just lexical nested classes. Java subclasses are normally sibling
// or cross-file declarations, and each one must be able to replace the dynamic
// receiver stored by its ancestors.
func classHasKnownSubclass(target *symbol.ClassScope) bool {
	if target == nil {
		return false
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				found := visitClassScopes(top, func(candidate *symbol.ClassScope) bool {
					if candidate == nil || candidate == target {
						return false
					}
					candidateCtx := Ctx{currentFile: file, currentClass: candidate}
					seen := map[*symbol.ClassScope]struct{}{}
					for parent := resolveSuperclassScope(candidateCtx, candidate); parent != nil; parent = resolveSuperclassScope(candidateCtx, parent) {
						if _, duplicate := seen[parent]; duplicate {
							break
						}
						seen[parent] = struct{}{}
						if parent == target {
							return true
						}
					}
					return false
				})
				if found {
					return true
				}
			}
		}
	}
	return false
}

func classNeedsVirtualDispatch(scope *symbol.ClassScope, _ Ctx) bool {
	if scope == nil || scope.IsInterface || scope.IsEnum {
		return false
	}
	hasDispatchableMethod := false
	for _, method := range scope.Methods {
		if method != nil && !method.Constructor && !method.IsStatic && !method.IsPrivate && !method.RequiresHelper {
			hasDispatchableMethod = true
			break
		}
	}
	return hasDispatchableMethod && (scope.IsAbstract || classHasKnownSubclass(scope))
}

func generateClassDispatchInterface(ctx Ctx) ast.Decl {
	scope := ctx.currentClass
	if !classNeedsVirtualDispatch(scope, ctx) {
		return nil
	}

	methods := &ast.FieldList{}
	typeParams := scope.TypeParameterNames()
	for _, method := range scope.Methods {
		if method == nil || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper {
			continue
		}
		params := &ast.FieldList{}
		for _, param := range method.Parameters {
			params.List = append(params.List, &ast.Field{
				Names: []*ast.Ident{{Name: param.Name}},
				Type:  javaTypeStringToGoTypeExpr(param.OriginalType, typeParams, ctx),
			})
		}
		var results *ast.FieldList
		if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
			results = &ast.FieldList{List: []*ast.Field{{
				Type: javaTypeStringToGoTypeExpr(method.OriginalType, typeParams, ctx),
			}}}
		}
		methods.List = append(methods.List, &ast.Field{
			Names: []*ast.Ident{{Name: method.Name}},
			Type:  &ast.FuncType{Params: params, Results: results},
		})
	}
	return genInterfaceInContext(classDispatchTypeName(scope), methods, scope.TypeParameters, ctx)
}

func defaultInterfaceTypes(scope *symbol.ClassScope, ctx Ctx) []string {
	if scope == nil {
		return nil
	}
	var defaults []string
	for _, implemented := range scope.ImplementedInterfaces {
		base, _ := parseJavaTypeString(implemented)
		if interfaceHasDefaultMethods(resolveClassScopeByQualifiedName(ctx, base), ctx) {
			defaults = append(defaults, implemented)
		}
	}
	return defaults
}

func classHasSelfSetter(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]struct{}{}
	for current := scope; current != nil; current = resolveSuperclassScope(ctx, current) {
		if _, duplicate := seen[current]; duplicate {
			return false
		}
		seen[current] = struct{}{}
		if classNeedsVirtualDispatch(current, ctx) || len(defaultInterfaceTypes(current, ctx)) > 0 {
			return true
		}
	}
	return false
}

func interfaceDefaultCarrierConstructorExpr(javaType string, typeParams []string, ctx Ctx) ast.Expr {
	base, typeArgs := parseJavaTypeString(javaType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if !interfaceHasDefaultMethods(scope, ctx) {
		return nil
	}
	constructor := qualifiedNameExpr(
		defaultConstructorName(interfaceDefaultCarrierName(scope)),
		resolveJavaPackageForType(ctx, base, scope),
		ctx,
	)
	if len(typeArgs) > 0 {
		args := make([]ast.Expr, 0, len(typeArgs))
		for _, arg := range typeArgs {
			args = append(args, javaTypeStringToGoTypeExpr(arg, typeParams, ctx))
		}
		constructor = applyTypeArguments(constructor, args)
	}
	return constructor
}

// generateClassSelfSetter wires all dispatch layers after construction. A
// subclass replaces each ancestor's dynamic receiver with itself, and inherited
// interface-default carriers are rebuilt around that same concrete value.
func generateClassSelfSetter(ctx Ctx) ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.IsInterface || scope.IsEnum || !classHasSelfSetter(scope, ctx) {
		return nil
	}

	recvName := ShortName(scope.Class.Name)
	recvType := instantiateGenericType(scope.Class.Name, typeParamExprs(scope.TypeParameterNames()))
	selfName := "self"
	body := []ast.Stmt{}

	if classNeedsVirtualDispatch(scope, ctx) {
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: classDispatchFieldName(scope)}}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.TypeAssertExpr{X: &ast.Ident{Name: selfName}, Type: classDispatchTypeExpr(scope)}},
		})
	}

	for _, implemented := range defaultInterfaceTypes(scope, ctx) {
		base, _ := parseJavaTypeString(implemented)
		interfaceScope := resolveClassScopeByQualifiedName(ctx, base)
		carrierConstructor := interfaceDefaultCarrierConstructorExpr(implemented, scope.TypeParameterNames(), ctx)
		interfaceType := javaTypeStringToGoTypeExpr(implemented, scope.TypeParameterNames(), ctx)
		body = append(body, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{
				X:   &ast.Ident{Name: recvName},
				Sel: &ast.Ident{Name: interfaceDefaultCarrierName(interfaceScope)},
			}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.CallExpr{
				Fun: carrierConstructor,
				Args: []ast.Expr{&ast.TypeAssertExpr{
					X:    &ast.Ident{Name: selfName},
					Type: interfaceType,
				}},
			}},
		})
	}

	if parent := resolveSuperclassScope(ctx, scope); parent != nil && classHasSelfSetter(parent, ctx) {
		body = append(body, &ast.ExprStmt{X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.SelectorExpr{
					X:   &ast.Ident{Name: recvName},
					Sel: &ast.Ident{Name: parent.Class.Name},
				},
				Sel: &ast.Ident{Name: classSelfSetterName(parent)},
			},
			Args: []ast.Expr{&ast.Ident{Name: selfName}},
		}})
	}

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: classSelfSetterName(scope)},
		Recv: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: recvName}},
			Type:  &ast.StarExpr{X: recvType},
		}}},
		Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: selfName}},
			Type:  &ast.Ident{Name: "any"},
		}}}},
		Body: &ast.BlockStmt{List: body},
	}
}

func classSelfSetterCallStmtWithValue(ctx Ctx, receiverName string, self ast.Expr) ast.Stmt {
	if ctx.currentClass == nil || !classHasSelfSetter(ctx.currentClass, ctx) {
		return nil
	}
	if self == nil {
		self = &ast.Ident{Name: receiverName}
	}
	return &ast.ExprStmt{X: &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: classSelfSetterName(ctx.currentClass)},
		},
		Args: []ast.Expr{self},
	}}
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

// parseRecordDecls lowers a Java record into a Go struct with the components as
// fields, a canonical constructor, an accessor method per component (named after
// the component, e.g. X() not GetX()), a value-equality method, and any
// user-declared body methods.
func parseRecordDecls(node *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil {
		return nil
	}
	ctx.className = scope.Class.Name
	typeParams := scope.TypeParameterNames()

	declarations := []ast.Decl{}

	// Struct fields from the record components (recorded in scope.Fields).
	fields := &ast.FieldList{}
	for _, f := range scope.Fields {
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: f.Name}},
			Type:  javaTypeStringToGoTypeExpr(f.OriginalType, typeParams, ctx),
		})
	}
	declarations = append(declarations, genStructWithTypeParamsInContext(ctx.className, fields, scope.TypeParameters, ctx))

	recvBase := instantiateGenericType(ctx.className, typeParamExprs(typeParams))
	recvName := ShortName(ctx.className)

	// Canonical constructor: New<Name>(c1, c2, ...) *Name.
	ctorParams := &ast.FieldList{}
	ctorBody := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: recvName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{recvBase}}},
		},
	}
	for _, f := range scope.Fields {
		ctorParams.List = append(ctorParams.List, &ast.Field{
			Names: []*ast.Ident{{Name: f.OriginalName}},
			Type:  javaTypeStringToGoTypeExpr(f.OriginalType, typeParams, ctx),
		})
		ctorBody = append(ctorBody, &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: f.Name}}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: f.OriginalName}},
		})
	}
	ctorBody = append(ctorBody, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: recvName}}})
	declarations = append(declarations, genFuncDeclWithTypeParamsInContext(
		"New"+ctx.className,
		scope.TypeParameters,
		ctorParams,
		&ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: recvBase}}}},
		&ast.BlockStmt{List: ctorBody},
		ctx,
	))

	// Accessor method per component: func (r *Name) X() T { return r.x }. The
	// method is exported (matching the resolved accessor Definition); the field it
	// returns is unexported, so they do not collide.
	bodyMethodNames := recordBodyMethodNames(node, source)
	for _, f := range scope.Fields {
		if _, declared := bodyMethodNames[f.OriginalName]; declared {
			continue // user provided an explicit accessor; emitted from the body below
		}
		accessorName := symbol.HandleExportStatus(true, f.OriginalName)
		declarations = append(declarations, &ast.FuncDecl{
			Name: &ast.Ident{Name: accessorName},
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: recvName}}, Type: &ast.StarExpr{X: recvBase}}}},
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(f.OriginalType, typeParams, ctx)}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: f.Name}}}},
			}},
		})
	}

	// Value-equality method mirroring Java record equals(): compares all fields.
	declarations = append(declarations, buildRecordEqualsDecl(scope, recvBase, recvName))

	// User-declared methods in the record body.
	if body := node.ChildByFieldName("body"); body != nil {
		declarations = append(declarations, ParseDecls(body, source, ctx)...)
	}

	return declarations
}

// recordBodyMethodNames returns the set of original method names declared in a
// record body, so synthesized accessors don't collide with explicit ones.
func recordBodyMethodNames(node *sitter.Node, source []byte) map[string]struct{} {
	names := map[string]struct{}{}
	body := node.ChildByFieldName("body")
	if body == nil {
		return names
	}
	for _, child := range nodeutil.NamedChildrenOf(body) {
		if child.Type() == "method_declaration" {
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				names[nameNode.Content(source)] = struct{}{}
			}
		}
	}
	return names
}

// buildRecordEqualsDecl synthesizes an Equals method comparing every field,
// mirroring the value semantics of a Java record's equals().
func buildRecordEqualsDecl(scope *symbol.ClassScope, recvBase ast.Expr, recvName string) ast.Decl {
	otherName := "other"
	var cond ast.Expr
	for _, f := range scope.Fields {
		cmp := ast.Expr(&ast.BinaryExpr{
			X:  &ast.SelectorExpr{X: &ast.Ident{Name: recvName}, Sel: &ast.Ident{Name: f.Name}},
			Op: token.EQL,
			Y:  &ast.SelectorExpr{X: &ast.Ident{Name: otherName}, Sel: &ast.Ident{Name: f.Name}},
		})
		if cond == nil {
			cond = cmp
		} else {
			cond = &ast.BinaryExpr{X: cond, Op: token.LAND, Y: cmp}
		}
	}
	if cond == nil {
		cond = &ast.Ident{Name: "true"}
	}
	return &ast.FuncDecl{
		Name: &ast.Ident{Name: "Equals"},
		Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: recvName}}, Type: &ast.StarExpr{X: recvBase}}}},
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: otherName}}, Type: &ast.StarExpr{X: recvBase}}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "bool"}}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{cond}}}},
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

func noArgConstructorName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	foundExplicit := false
	for _, method := range scope.Methods {
		if method == nil || !method.Constructor {
			continue
		}
		foundExplicit = true
		if len(method.Parameters) == 0 {
			return method.Name
		}
	}
	if foundExplicit {
		return ""
	}
	return defaultConstructorName(scope.Class.Name)
}

func constructorWithSelfName(name string) string {
	return name + constructorSelfSuffix
}

func constructorMostDerivedInitStmt(receiverName string) ast.Stmt {
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.Ident{Name: constructorSelfParam},
			Op: token.EQL,
			Y:  &ast.Ident{Name: "nil"},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: constructorSelfParam}},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{&ast.Ident{Name: receiverName}},
		}}},
	}
}

func renameConstructorCallForSelf(fun ast.Expr) ast.Expr {
	switch typed := fun.(type) {
	case *ast.Ident:
		return &ast.Ident{Name: constructorWithSelfName(typed.Name)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: typed.X, Sel: &ast.Ident{Name: constructorWithSelfName(typed.Sel.Name)}}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: renameConstructorCallForSelf(typed.X), Index: typed.Index}
	case *ast.IndexListExpr:
		return &ast.IndexListExpr{X: renameConstructorCallForSelf(typed.X), Indices: typed.Indices}
	default:
		return fun
	}
}

// rewriteConstructorChainForSelf threads the most-derived receiver through an
// explicit super(...) or this(...) constructor invocation. The callee installs
// that receiver before executing any of its Java constructor body.
func rewriteConstructorChainForSelf(stmt ast.Stmt, ctx Ctx) ast.Stmt {
	if stmt == nil || ctx.currentClass == nil {
		return stmt
	}
	var call *ast.CallExpr
	if isExplicitSuperConstructorAssignment(stmt, ctx) {
		parent := resolveSuperclassScope(ctx, ctx.currentClass)
		if parent == nil || !classHasSelfSetter(parent, ctx) {
			return stmt
		}
		assignment := stmt.(*ast.AssignStmt)
		if len(assignment.Rhs) == 1 {
			call, _ = assignment.Rhs[0].(*ast.CallExpr)
		}
	} else if isThisConstructorInvocation(stmt, ctx.className) {
		exprStmt := stmt.(*ast.ExprStmt)
		call, _ = exprStmt.X.(*ast.CallExpr)
	}
	if call == nil {
		return stmt
	}
	call.Fun = renameConstructorCallForSelf(call.Fun)
	call.Args = append([]ast.Expr{&ast.Ident{Name: constructorSelfParam}}, call.Args...)
	return stmt
}

// implicitSuperConstructorAssignmentWithSelf emits Java's implicit leading
// super() call. It is used both by synthesized default constructors and by
// explicit constructors whose source body omits a this(...)/super(...) call.
func implicitSuperConstructorAssignmentWithSelf(ctx Ctx, receiverName string, mostDerived ast.Expr) ast.Stmt {
	scope := ctx.currentClass
	parent := resolveSuperclassScope(ctx, scope)
	if scope == nil || parent == nil || parent.Class == nil {
		return nil
	}
	constructorName := noArgConstructorName(parent)
	if constructorName == "" {
		return nil
	}
	args := []ast.Expr{}
	if mostDerived != nil && classHasSelfSetter(parent, ctx) {
		constructorName = constructorWithSelfName(constructorName)
		args = append(args, mostDerived)
	}

	base, typeArgs := parseJavaTypeString(scope.Superclass)
	constructor := qualifiedNameExpr(
		constructorName,
		resolveJavaPackageForType(ctx, base, parent),
		ctx,
	)
	if len(typeArgs) > 0 {
		goTypeArgs := make([]ast.Expr, 0, len(typeArgs))
		for _, typeArg := range typeArgs {
			goTypeArgs = append(goTypeArgs, javaTypeStringToGoTypeExpr(typeArg, scope.TypeParameterNames(), ctx))
		}
		constructor = applyTypeArguments(constructor, goTypeArgs)
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.SelectorExpr{
			X:   &ast.Ident{Name: receiverName},
			Sel: &ast.Ident{Name: parent.Class.Name},
		}},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.CallExpr{Fun: constructor, Args: args}},
	}
}

func buildConstructorDeclarations(
	constructorName string,
	typeParams []symbol.TypeParam,
	params *ast.FieldList,
	returnType *ast.FieldList,
	body *ast.BlockStmt,
	usesMostDerived bool,
	ctx Ctx,
) []ast.Decl {
	if !usesMostDerived {
		return []ast.Decl{genFuncDeclWithTypeParamsInContext(
			constructorName, typeParams, params, returnType, body, ctx,
		)}
	}

	wrapperParams := cloneFieldList(params)
	internalParams := cloneFieldList(params)
	internalParams.List = append([]*ast.Field{{
		Names: []*ast.Ident{{Name: constructorSelfParam}},
		Type:  &ast.Ident{Name: "any"},
	}}, internalParams.List...)

	internalName := constructorWithSelfName(constructorName)
	internalFun := ast.Expr(&ast.Ident{Name: internalName})
	if len(typeParams) > 0 {
		internalFun = applyTypeArguments(internalFun, typeParamExprs(symbol.TypeParamNames(typeParams)))
	}
	args := append([]ast.Expr{&ast.Ident{Name: "nil"}}, methodCallArgs(wrapperParams)...)
	call := &ast.CallExpr{Fun: internalFun, Args: args}
	if wrapperParams != nil && len(wrapperParams.List) > 0 {
		if _, variadic := wrapperParams.List[len(wrapperParams.List)-1].Type.(*ast.Ellipsis); variadic {
			call.Ellipsis = token.Pos(1)
		}
	}
	wrapperBody := &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}

	return []ast.Decl{
		genFuncDeclWithTypeParamsInContext(constructorName, typeParams, wrapperParams, returnType, wrapperBody, ctx),
		genFuncDeclWithTypeParamsInContext(internalName, typeParams, internalParams, returnType, body, ctx),
	}
}

// buildDefaultConstructorDecls synthesizes an implicit no-arg constructor for a
// class that declares none. The generated function mirrors the shape produced
// for explicit constructors: allocate the struct, run instance-field
// initializers, and return the pointer. Interfaces and enums are skipped since
// they are never instantiated through a `New<Class>` function.
func buildDefaultConstructorDecls(ctx Ctx) []ast.Decl {
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
	usesMostDerived := classHasSelfSetter(scope, ctx)
	body := []ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: recvName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{structType}}},
		},
	}
	if usesMostDerived {
		body = append(body, constructorMostDerivedInitStmt(recvName))
	}
	var mostDerived ast.Expr
	if usesMostDerived {
		mostDerived = &ast.Ident{Name: constructorSelfParam}
	}
	if superInit := implicitSuperConstructorAssignmentWithSelf(ctx, recvName, mostDerived); superInit != nil {
		body = append(body, superInit)
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

	if setterCall := classSelfSetterCallStmtWithValue(ctx, recvName, mostDerived); setterCall != nil {
		body = append(body, setterCall)
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

	// Match the casing of the call-site reference (defaultConstructorName): a
	// package-private class gets `new<Name>`, not the miscased `New<name>`.
	constructorName := defaultConstructorName(ctx.className)
	returnType := &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: structType}}}}

	return buildConstructorDeclarations(constructorName, typeParams, params, returnType, &ast.BlockStmt{List: body}, usesMostDerived, ctx)
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
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "byte", "rune":
			return &ast.BasicLit{Kind: token.INT, Value: "0"}
		default:
			// Named interfaces and type parameters cannot be distinguished from the
			// identifier alone. Dereferencing a freshly allocated value yields the
			// valid zero for both (nil for an interface, concrete zero for T), unlike
			// a literal nil which does not compile when T is instantiated by a value
			// type such as the project's Integer -> int32 representation.
			return &ast.StarExpr{X: &ast.CallExpr{Fun: &ast.Ident{Name: "new"}, Args: []ast.Expr{t}}}
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
		if method.Constructor || method.IsStatic || method.IsPrivate {
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

	return genInterfaceInContext(ctx.className+"I", methods, scope.TypeParameters, ctx)
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

func genFunctionalInterfaceAdapterDecls(interfaceName string, methods *ast.FieldList, typeParams []symbol.TypeParam, scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
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

	adapterStruct := genStructWithTypeParamsInContext(adapterName, structFields, typeParams, ctx)

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

	constructor := genFuncDeclWithTypeParamsInContext(constructorName, typeParams, constructorParams, constructorResults, constructorBody, ctx)

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

func findEnumToStringMethod(scope *symbol.ClassScope) *symbol.Definition {
	if scope == nil {
		return nil
	}
	for _, def := range scope.Methods {
		if def == nil || def.IsStatic || def.Constructor || len(def.Parameters) != 0 {
			continue
		}
		if def.OriginalName == "toString" && stripJavaQualifier(def.OriginalType) == "String" {
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
	helperStruct := genStructWithTypeParamsInContext(helperName, helperFields, combinedTypeParams, ctx)

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
	constructor := genFuncDeclWithTypeParamsInContext(constructorName, combinedTypeParams, constructorParams, returnType, constructorBody, ctx)

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

// synthesizeRawGenericFunctionParameters models Java raw generic parameters on
// static methods. A raw `Box box` accepts Box<String>, Box<Integer>, and every
// other instantiation, but Go requires an explicit argument and its generic
// types are invariant. Emitting a fresh function type parameter per raw formal
// (`func Use[BoxT any](box *Box[BoxT])`) preserves that call-site flexibility
// and lets Go infer the concrete argument without changing the Java symbols.
func synthesizeRawGenericFunctionParameters(def *symbol.Definition, ctx Ctx) ([]symbol.TypeParam, map[string]string) {
	if def == nil || !def.IsStatic {
		return nil, nil
	}

	usedNames := make(map[string]struct{})
	for _, typeParam := range def.TypeParameters {
		usedNames[typeParam.Name] = struct{}{}
	}

	var synthetic []symbol.TypeParam
	rewrittenTypes := make(map[string]string)
	for _, param := range def.Parameters {
		if param == nil {
			continue
		}

		javaType := strings.TrimSpace(param.OriginalType)
		arraySuffix := ""
		for strings.HasSuffix(javaType, "[]") {
			arraySuffix += "[]"
			javaType = strings.TrimSpace(javaType[:len(javaType)-2])
		}
		base, explicitArgs := parseJavaTypeString(javaType)
		if base == "" || len(explicitArgs) > 0 {
			continue
		}
		target := resolveClassScopeByQualifiedName(ctx, base)
		if target == nil || len(target.TypeParameters) == 0 {
			continue
		}

		stem := symbol.Uppercase(sanitizeGoIdent(param.Name))
		if stem == "" {
			stem = "Raw"
		}
		bindings := make(map[string]string, len(target.TypeParameters))
		generatedNames := make([]string, 0, len(target.TypeParameters))
		for _, targetParam := range target.TypeParameters {
			candidate := stem + targetParam.Name
			if candidate == "" {
				candidate = "RawType"
			}
			baseCandidate := candidate
			for suffix := 2; ; suffix++ {
				if _, exists := usedNames[candidate]; !exists {
					break
				}
				candidate = fmt.Sprintf("%s%d", baseCandidate, suffix)
			}
			usedNames[candidate] = struct{}{}
			bindings[targetParam.Name] = candidate
			generatedNames = append(generatedNames, candidate)
		}

		for index, targetParam := range target.TypeParameters {
			generated := symbol.TypeParam{Name: generatedNames[index]}
			for _, bound := range targetParam.Bounds {
				generated.Bounds = append(generated.Bounds, symbol.JavaType{
					Original: substituteJavaTypeParameters(bound.Original, bindings),
				})
			}
			synthetic = append(synthetic, generated)
		}

		rewritten := base + "<" + strings.Join(generatedNames, ", ") + ">" + arraySuffix
		rewrittenTypes[param.OriginalName] = rewritten
		rewrittenTypes[param.Name] = rewritten
	}

	if len(rewrittenTypes) == 0 {
		return nil, nil
	}
	return synthetic, rewrittenTypes
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
		usesMostDerived := classHasSelfSetter(ctx.currentClass, ctx)
		if usesMostDerived {
			prelude = append(prelude, constructorMostDerivedInitStmt(ShortName(ctx.className)))
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
		userBody := body.List
		remainingBody := userBody
		var constructorChain ast.Stmt
		if len(userBody) > 0 && (isExplicitSuperConstructorAssignment(userBody[0], ctx) || isThisConstructorInvocation(userBody[0], ctx.className)) {
			constructorChain = userBody[0]
			remainingBody = userBody[1:]
			if usesMostDerived {
				constructorChain = rewriteConstructorChainForSelf(constructorChain, ctx)
			}
		} else {
			var mostDerived ast.Expr
			if usesMostDerived {
				mostDerived = &ast.Ident{Name: constructorSelfParam}
			}
			constructorChain = implicitSuperConstructorAssignmentWithSelf(ctx, ShortName(ctx.className), mostDerived)
		}

		body.List = prelude
		if constructorChain != nil {
			body.List = append(body.List, constructorChain)
		}
		var mostDerived ast.Expr
		if usesMostDerived {
			mostDerived = &ast.Ident{Name: constructorSelfParam}
		}
		if setterCall := classSelfSetterCallStmtWithValue(ctx, ShortName(ctx.className), mostDerived); setterCall != nil {
			body.List = append(body.List, setterCall)
		}
		if shouldCallFieldInitializerMethodForBody(userBody, ctx) {
			body.List = append(body.List, &ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: ShortName(ctx.className)},
				Sel: &ast.Ident{Name: fieldInitMethodName},
			}}})
		}
		body.List = append(body.List, remainingBody...)
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

		return buildConstructorDeclarations(
			ctx.localScope.Name,
			constructorTypeParams,
			constructorParams,
			&ast.FieldList{List: []*ast.Field{{Type: returnType}}},
			body,
			usesMostDerived,
			ctx,
		)
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

		// Symbol lookup must use the original Java spelling. identFromNode applies
		// Go-keyword sanitization (`range` -> `range_`), but symbol definitions keep
		// OriginalName as `range` and carry their resolved Go name separately.
		methodName := node.ChildByFieldName("name").Content(source)
		methodParameters := node.ChildByFieldName("parameters")

		comparison := func(d *symbol.Definition) bool {
			if d.OriginalName != methodName {
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
				"methodName": methodName,
			}).Panic("No matching definition found for method")
		}

		ctx.localScope = methodDefinition[0]
		if static {
			ctx.syntheticTypeParameters, ctx.rawGenericParameterTypes = synthesizeRawGenericFunctionParameters(ctx.localScope, ctx)
		}

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

		if methodName == "main" && bodyNode != nil {
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
			effectiveTypeParameters := symbol.MergeTypeParams(ctx.localScope.TypeParameters, ctx.syntheticTypeParameters)
			if len(effectiveTypeParameters) > 0 {
				funcDecl.Type.TypeParams = &ast.FieldList{List: makeTypeParamFieldsInContext(effectiveTypeParameters, ctx)}
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
