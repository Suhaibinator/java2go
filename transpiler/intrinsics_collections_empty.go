package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	registerIntrinsicOwner("java.util.Collections", true)
	registerStaticNodeIntrinsic("Collections", "emptySet", lowerCollectionsEmptySet)
	registerStaticIntrinsicDerivedResultType("Collections", "emptySet", collectionsEmptySetResult)
	parameter := symbol.NewTypeParam("T", []symbol.JavaType{{Original: "java.lang.Object"}})
	definition := &symbol.Definition{OriginalName: "emptySet", OriginalType: "java.util.Set<T>", IsStatic: true,
		TypeParameters: []symbol.TypeParam{parameter}, TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"T": parameter.Declaration}}
	key := intrinsicKey{"Collections", "emptySet"}
	staticIntrinsicImportSignatures[key] = append(staticIntrinsicImportSignatures[key], definition)
}

func lowerCollectionsEmptySet(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if len(args) != 0 {
		return nil
	}
	if _, known := collectionsEmptySetResult(invocation, ctx, source); !known {
		return nil
	}
	return stdjavaCall(ctx, "EmptySet")
}

func collectionsEmptySetResult(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if invocation == nil || invocationArgumentCount(invocation) != 0 {
		return "", false
	}
	if object := invocation.ChildByFieldName("object"); object != nil {
		key, known := intrinsicStaticClassName(object, ctx, source)
		owner, resolved := canonicalIntrinsicOwner(object.Content(source), ctx)
		if !known || key != "Collections" || !resolved || owner != "java.util.Collections" {
			return "", false
		}
		proven := object.Content(source) == owner
		if ctx.currentFile != nil {
			proven = proven || ctx.currentFile.Imports["Collections"] == "java.util"
		}
		for _, pkg := range intrinsicOnDemandImports(ctx) {
			proven = proven || pkg == "java.util"
		}
		if !proven {
			return "", false
		}
	} else {
		resolution := resolveStaticImportedMethod(invocation, ctx, source)
		if resolution.intrinsic != "Collections" || resolution.source != nil || resolution.problem != "" {
			return "", false
		}
		proven := false
		for _, entry := range staticMethodImports(ctx) {
			proven = proven || entry.owner == "java.util.Collections" && (entry.wildcard || entry.member == "emptySet")
		}
		if !proven {
			return "", false
		}
	}
	var element string
	if explicit := invocation.ChildByFieldName("type_arguments"); explicit != nil {
		if explicit.NamedChildCount() != 1 {
			return "", false
		}
		syntax := strings.TrimSpace(explicit.NamedChild(0).Content(source))
		base, rank := javaArrayTypeParts(syntax)
		if base == "void" || strings.HasPrefix(base, "?") || base == "" {
			return "", false
		}
		if _, primitive := javaPrimitiveType(base); primitive && rank == 0 {
			return "", false
		}
		element = qualifyDeclaredReferenceType(symbol.JavaType{Original: syntax,
			TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
	} else {
		var known bool
		element, known = collectionsEmptySetExpectedElement(invocation, ctx, source)
		if !known {
			return "", false
		}
	}
	return "java.util.Set<" + element + ">", true
}

func collectionsEmptySetExpectedElement(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if ctx.expectedType == "" || !expectedTypeTargetsExpression(ctx, invocation) {
		return "java.lang.Object", true
	}
	_, rank := javaArrayTypeParts(ctx.expectedType)
	if rank != 0 {
		return "java.lang.Object", true
	}
	target, declaring, owned := collectionsEmptySetDeclaredTarget(invocation, ctx, source)
	if !owned {
		// Formal types have already crossed their declaration boundary. Only
		// allocated aliases may be substituted here; source names can shadow them.
		bindings := map[string]*symbol.TypeParamDeclaration{}
		for _, binding := range referenceTypeParameterBindings(ctx) {
			name := binding.parameter.EmittedName()
			if declaration, exists := bindings[name]; exists && declaration != binding.parameter.Declaration {
				return "", false
			}
			bindings[name] = binding.parameter.Declaration
		}
		target = symbol.JavaType{Original: ctx.expectedType, TypeParameterBindings: bindings}
		declaring = ctx
	}
	qualified := qualifyDeclaredReferenceType(target, declaring)
	base, arguments := parseJavaTypeString(qualified)
	if base != "java.util.Set" && base != "java.util.Collection" && base != "java.lang.Iterable" {
		return "java.lang.Object", true
	}
	if len(arguments) == 0 {
		return "java.lang.Object", true
	}
	if len(arguments) != 1 {
		return "", false
	}
	// The factory is empty; wildcard consumers retain their ordinary read
	// projection without introducing a wildcard Go storage argument.
	return readableWildcardProjection(arguments[0]), true
}

func collectionsEmptySetDeclaredTarget(invocation *sitter.Node, ctx Ctx, source []byte) (symbol.JavaType, Ctx, bool) {
	accept := func(definition *symbol.Definition, declaring Ctx) (symbol.JavaType, Ctx, bool) {
		target := definitionDeclaredJavaType(definition)
		if definition == nil || target.Original == "" {
			return symbol.JavaType{}, ctx, false
		}
		if target.Original != ctx.expectedType && qualifyDeclaredReferenceType(target, declaring) != ctx.expectedType {
			return symbol.JavaType{}, ctx, false
		}
		// An explicitly empty binding map marks nominal syntax as declaration-owned.
		if target.TypeParameterBindings == nil {
			target.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{}
		}
		return target, declaring, true
	}
	root := invocation
	for root.Parent() != nil && root.Parent().Type() == "parenthesized_expression" {
		root = root.Parent()
	}
	parent := root.Parent()
	if parent == nil {
		return symbol.JavaType{}, ctx, false
	}
	switch parent.Type() {
	case "return_statement":
		if ctx.localScope != nil {
			declaring, owner := invocationMethodDeclarationContext(ctx.localScope, ctx)
			if owner != nil {
				declaring = classHeaderTypeCtx(owner, declaring)
				declaring.localScope = ctx.localScope
			}
			return accept(ctx.localScope, declaring)
		}
	case "variable_declarator":
		if !sameSourceNode(parent.ChildByFieldName("value"), root) {
			break
		}
		name := parent.ChildByFieldName("name")
		if name == nil {
			break
		}
		if declaration := parent.Parent(); declaration != nil && declaration.Type() == "local_variable_declaration" && ctx.localScope != nil {
			return accept(ctx.localScope.FindVariable(name.Content(source)), ctx)
		}
		if ctx.currentClass != nil && parent.Parent() != nil && parent.Parent().Type() == "field_declaration" {
			field := ctx.currentClass.FindFieldByName(name.Content(source))
			if field != nil && sameSourceNode(field.DeclarationNode, parent.Parent()) {
				return accept(field, classHeaderTypeCtx(ctx.currentClass, ctx))
			}
		}
	}
	return symbol.JavaType{}, ctx, false
}
