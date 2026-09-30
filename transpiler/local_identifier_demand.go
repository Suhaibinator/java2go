package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// A Go declaration in a method signature is outside the parameter's scope.
// Merely sharing a name with a file-level type, function or import therefore
// does not require changing that parameter. Only uses in its generated body
// need protection; optimizations retain their existing collision fallbacks.
func localIdentifierRequiredByBody(name string, ctx Ctx) bool {
	body := ctx.localBindingBody
	if ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
		body = ctx.localScope.DeclarationNode.ChildByFieldName("body")
	}
	if body == nil || ctx.currentFile == nil {
		return false
	}
	source := ctx.currentFile.Source
	if name == "nil" && ctx.localScope != nil && !ctx.localScope.IsStatic {
		return true
	}
	calledNames := map[string]bool{}
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			if method.IsStatic && (name == method.Name || name == method.Name+executionMethodSuffix) || method.RequiresHelper && (name == method.HelperName || name == "New"+method.HelperName) {
				calledNames[method.OriginalName] = true
			}
		}
	}
	var walk func(*sitter.Node) bool
	walk = func(node *sitter.Node) bool {
		if node == nil {
			return false
		}
		switch node.Type() {
		case "type_identifier", "integral_type", "floating_point_type", "boolean_type":
			if localTypeRequiresIdentifier(node.Content(source), name, ctx) {
				return true
			}
		case "return_statement":
			if ctx.localScope != nil && localTypeRequiresIdentifier(ctx.localScope.OriginalType, name, ctx) {
				return true
			}
		case "null_literal":
			if name == "nil" {
				return true
			}
		case "true", "false":
			if name == node.Type() {
				return true
			}
		case "throw_statement", "assert_statement", "try_statement", "try_with_resources_statement", "synchronized_statement":
			if name == "panic" || name == "recover" || name == "nil" || name == "bool" || name == "any" {
				return true
			}
		case "method_invocation":
			if called := node.ChildByFieldName("name"); called != nil && calledNames[called.Content(source)] {
				return true
			}
			if localImplicitTypeRequiresIdentifier(node, name, ctx, source) {
				return true
			}
		case "field_access":
			if field := node.ChildByFieldName("field"); field != nil && field.Content(source) == "length" && (name == "len" || name == "int32") {
				return true
			}
			if localImplicitTypeRequiresIdentifier(node, name, ctx, source) {
				return true
			}
		case "object_creation_expression":
			if typ := node.ChildByFieldName("type"); typ != nil {
				base, _ := parseJavaTypeString(typ.Content(source))
				if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil {
					if name == "New"+scope.Class.Name || name == "new"+scope.Class.Name || name == classDispatchTypeName(scope) {
						return true
					}
				}
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			if walk(child) {
				return true
			}
		}
		return false
	}
	return walk(body)
}

func localTypeRequiresIdentifier(javaType, name string, ctx Ctx) bool {
	if scope := resolveClassScopeByQualifiedName(ctx, javaType); scope != nil && scope.Class != nil {
		if name == scope.Class.Name || name == scope.Class.Name+"I" {
			return true
		}
		pkg := findJavaPackageForClassScope(scope)
		return pkg != "" && ctx.importAliases[pkg] == name
	}
	for _, parameter := range visibleTypeParameterDeclarations(ctx) {
		if parameter.Name == javaType || parameter.EmittedName() == javaType {
			return name == parameter.EmittedName() || name == "any"
		}
	}
	builtin := map[string]string{"String": "string", "java.lang.String": "string", "Object": "any", "java.lang.Object": "any", "boolean": "bool", "byte": "int8", "short": "int16", "char": "rune", "int": "int32", "long": "int64", "float": "float32", "double": "float64"}
	return builtin[strings.TrimSpace(javaType)] == name
}

// Erased generic reads and numeric unboxing can introduce a Go type argument
// or conversion without spelling that type in Java's body syntax.
func localImplicitTypeRequiresIdentifier(node *sitter.Node, name string, ctx Ctx, source []byte) bool {
	switch name {
	case "string", "any", "bool", "int8", "int16", "rune", "int32", "int64", "float32", "float64":
		if typ, ok := inferExprJavaType(node, ctx, source); ok {
			return localTypeRequiresIdentifier(typ, name, ctx)
		}
	}
	return false
}
