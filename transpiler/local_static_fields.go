package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Retain the existing instance-field ABI. Only actual static declarations gain
// ordinary field metadata; anonymous classes keep their own lowering path.
func localClassDeclaredFields(body *sitter.Node, source []byte, ctx Ctx) ([]*symbol.Definition, []*ast.Field) {
	definitions, fields := anonymousClassDeclaredFields(body, source, ctx)
	indices := map[string]int{}
	for i, definition := range definitions {
		indices[definition.OriginalName] = i
	}
	// The local declaration is a class even when its method is declared in an
	// interface. Do not inherit that enclosing owner's implicit field modifiers.
	declaring := &symbol.ClassScope{Class: &symbol.Definition{}}
	if declaration := body.Parent(); declaration != nil && declaration.Type() == "class_declaration" {
		declaring.Class.DeclarationNode = declaration
		if name := declaration.ChildByFieldName("name"); name != nil {
			declaring.Class.OriginalName = name.Content(source)
		}
	}
	if ctx.currentClass != nil {
		declaring.TypeParameters = ctx.currentClass.TypeParameters
	}
	for _, member := range nodeutil.NamedChildrenOf(body) {
		for _, definition := range symbol.ParseFieldDefinitions(member, source, declaring) {
			if index, exists := indices[definition.OriginalName]; exists && definition.IsStatic {
				definitions[index] = definition
			}
		}
	}
	return definitions, fields
}

func hoistedLocalClassScope(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "class_declaration" {
		return false
	}
	for _, local := range ctx.localClasses {
		if local != nil && local.scope == scope {
			return true
		}
	}
	return false
}

func localStaticNameOccupied(name string, scope *symbol.ClassScope, ctx Ctx) bool {
	if generatedIdentifierExists(name, scope) {
		return true
	}
	if ctx.hoistedDecls != nil {
		for _, declaration := range *ctx.hoistedDecls {
			occupied := false
			ast.Inspect(declaration, func(node ast.Node) bool {
				if identifier, ok := node.(*ast.Ident); ok && identifier.Name == name {
					occupied = true
				}
				return !occupied
			})
			if occupied {
				return true
			}
		}
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			if strings.Contains(string(file.Source), name) {
				return true
			}
		}
	}
	return false
}

// A static slot belongs to the source declaration, independent of instances,
// captures, enclosing invocations, and generic instantiations of that class.
func allocateLocalStaticFieldNames(scope *symbol.ClassScope, ctx Ctx) {
	for _, field := range scope.Fields {
		if !field.IsStatic {
			continue
		}
		base := scope.Class.Name + "_" + sanitizeGoIdent(field.OriginalName) + "Java2goStatic"
		for suffix := 0; ; suffix++ {
			name := base
			if suffix != 0 {
				name += "_" + strconv.Itoa(suffix)
			}
			if !localStaticNameOccupied(name, scope, ctx) {
				field.Name = name
				break
			}
		}
	}
}

func localStaticInitializerCtx(scope *symbol.ClassScope, ctx Ctx) Ctx {
	ctx = ctx.Clone()
	ctx.currentClass = scope
	ctx.className = scope.Class.Name
	ctx.localScope = &symbol.Definition{IsStatic: true}
	ctx.executionContextName = ""
	ctx.expectedType = ""
	ctx.expectedTypeRoot = nil
	ctx.tryReturnTarget = nil
	ctx.tryControlBoundary = nil
	ctx.doWhileContinueTargets = nil
	ctx.javaLabelTargets = nil
	return ctx
}

func localStaticFieldDecls(body *sitter.Node, source []byte, scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	staticCtx := localStaticInitializerCtx(scope, ctx)
	resolveCompileTimeConstantsForClass(scope, source, staticCtx)
	variables := &ast.GenDecl{Tok: token.VAR}
	for _, field := range scope.Fields {
		if !field.IsStatic {
			continue
		}
		typeExpr := javaTypeStringToGoTypeExpr(field.OriginalType, inScopeTypeParameters(staticCtx), staticCtx)
		typeExpr = abstractClassToInterface(typeExpr, field.OriginalType, staticCtx)
		typeExpr = volatileFieldBackingType(field, typeExpr, staticCtx)
		spec := &ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(field.Name)}, Type: typeExpr}
		if isBuiltinJavaString(field.OriginalType, staticCtx) && !volatileFieldDefinition(field) {
			spec.Values = []ast.Expr{javaStringNullExpr(staticCtx)}
		}
		if field.IsCompileTimeConstant {
			declarator := declaratorForField(field, source)
			valueNode := declarator.ChildByFieldName("value")
			valueCtx := staticCtx.Clone()
			valueCtx.expectedType = field.OriginalType
			valueCtx.expectedTypeRoot = valueNode
			value := ParseExpr(valueNode, source, valueCtx)
			spec.Values = []ast.Expr{coerceArgumentToExpectedType(value, valueNode, field.OriginalType, valueCtx, source)}
		}
		variables.Specs = append(variables.Specs, spec)
	}
	var declarations []ast.Decl
	if len(variables.Specs) != 0 {
		declarations = append(declarations, variables)
		declarations = append(declarations, staticFieldStorageHelperDecls(variables, staticCtx)...)
	}
	return append(declarations, buildLazyClassInitializationDecls(body, source, staticCtx)...)
}
