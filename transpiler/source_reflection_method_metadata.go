package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/printer"
	"go/token"
	"strconv"
)

// Only genuine Java method declarations enter reflection. Hidden execution
// selectors and generated helper functions remain implementation details.
func sourceReflectionDeclaredMethod(scope *symbol.ClassScope, method *symbol.Definition) bool {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil || method == nil || method.Constructor || method.DeclarationNode == nil || method.DeclarationNode.Type() != "method_declaration" {
		return false
	}
	// Byte ranges alone cannot identify the declaration's tree: inherited
	// methods from another file can happen to occupy the same offsets.
	owner := scope.Class.DeclarationNode
	for parent := method.DeclarationNode.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Equal(owner) {
			return true
		}
		switch parent.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "object_creation_expression":
			return false
		}
	}
	return false
}
func sourceReflectionMethodSignature(scope *symbol.ClassScope, method *symbol.Definition, ctx Ctx) ([]ast.Expr, ast.Expr, bool) {
	declaring := classScopeCtx(sourceReflectionJavaDeclarationScope(scope), ctx).Clone()
	declaring.localScope = method
	descriptor := func(original string, bindings map[string]*symbol.TypeParamDeclaration) (ast.Expr, bool) {
		return javaTypeDescriptorExpr(qualifyDeclaredReferenceType(symbol.JavaType{Original: original, TypeParameterBindings: bindings}, declaring), declaring)
	}
	parameters := make([]ast.Expr, len(method.Parameters))
	for index, parameter := range method.Parameters {
		value, ok := descriptor(definitionParameterJavaSignatureType(method, index), parameter.TypeParameterBindings)
		if !ok {
			return nil, nil, false
		}
		parameters[index] = value
	}
	if javaMethodResultIsVoid(method) {
		return parameters, stdjavaCall(ctx, "PrimitiveTypeID", metadataString("void")), true
	}
	result, ok := descriptor(method.OriginalType, method.TypeParameterBindings)
	return parameters, result, ok
}
func sourceReflectionDescriptorKey(parameters []ast.Expr, result ast.Expr) string {
	var buffer bytes.Buffer
	for _, parameter := range parameters {
		if err := printer.Fprint(&buffer, token.NewFileSet(), parameter); err != nil {
			panic(err)
		}
		buffer.WriteByte(0)
	}
	buffer.WriteByte(1)
	if err := printer.Fprint(&buffer, token.NewFileSet(), result); err != nil {
		panic(err)
	}
	return buffer.String()
}
func sourceReflectionTypeIDs(values []ast.Expr, ctx Ctx) ast.Expr {
	return &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("TypeID", ctx)}, Elts: values}
}

// Source modifiers describe the real declaration. javac's synthetic bridge
// keeps its access visibility, independently of final/synchronized/abstract.
func sourceReflectionMethodModifiers(scope *symbol.ClassScope, method *symbol.Definition, bridge bool) int32 {
	modifiers := sourceReflectionModifiers(method, scope)
	if scope.IsInterface {
		if method.IsPrivate {
			modifiers &^= 1
		}
		if method.HasBody || method.IsStatic || method.IsPrivate {
			modifiers &^= 1024
		}
	}
	if len(method.Parameters) > 0 && executionParameterIsVariadic(method, len(method.Parameters)-1) {
		modifiers |= 128
	}
	if bridge {
		modifiers = modifiers&7 | 64 | 4096
	}
	return modifiers
}

func sourceReflectionMethodDescriptor(scope *symbol.ClassScope, method *symbol.Definition, parameters []ast.Expr, result ast.Expr, bridge bool, invocationParameters []ast.Expr, ctx Ctx) ast.Expr {
	modifiers := sourceReflectionMethodModifiers(scope, method, bridge)
	values := []ast.Expr{metadataKey("Name", metadataString(method.OriginalName)), metadataKey("GoName", metadataGoName(sourceReflectionMethodExecutionName(scope, method, ctx))), metadataKey("ParameterTypes", sourceReflectionTypeIDs(parameters, ctx)), metadataKey("Return", result), metadataKey("Modifiers", reflectionInteger(modifiers)), metadataKey("HasModifiers", ast.NewIdent("true")), metadataKey("Bridge", ast.NewIdent(strconv.FormatBool(bridge))), metadataKey("Synthetic", ast.NewIdent(strconv.FormatBool(bridge)))}
	if bridge {
		values = append(values, metadataKey("InvocationParameterTypes", sourceReflectionTypeIDs(invocationParameters, ctx)))
	}
	if method.IsStatic && !method.RequiresHelper && len(scope.TypeParameters) == 0 {
		values = append(values, metadataKey("StaticFunction", ast.NewIdent(executionImplementationName(method, scope, ctx))))
	}
	return &ast.CompositeLit{Elts: values}
}
func sourceReflectionMethods(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	var out []ast.Expr
	var ancestors []*symbol.ClassScope
	seen := map[*symbol.ClassScope]bool{scope: true}
	queue := []*symbol.ClassScope{scope}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		parents := resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)
		if parent := resolveSuperclassScopeInDeclaringContext(ctx, current); parent != nil {
			parents = append(parents, parent)
		}
		for _, parent := range parents {
			if parent != nil && !seen[parent] {
				seen[parent] = true
				ancestors = append(ancestors, parent)
				queue = append(queue, parent)
			}
		}
	}
	for _, method := range scope.Methods {
		if !sourceReflectionDeclaredMethod(scope, method) {
			continue
		}
		parameters, result, ok := sourceReflectionMethodSignature(scope, method, ctx)
		if !ok {
			continue
		}
		out = append(out, sourceReflectionMethodDescriptor(scope, method, parameters, result, false, nil, ctx))
		if method.IsStatic || method.IsPrivate || len(method.TypeParameters) != 0 {
			continue
		}
		descriptors := map[string]bool{sourceReflectionDescriptorKey(parameters, result): true}
		for _, ancestor := range ancestors {
			args := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), ancestor, ctx)
			for _, candidate := range ancestor.Methods {
				if !sourceReflectionDeclaredMethod(ancestor, candidate) || candidate.IsStatic || candidate.IsPrivate || candidate.OriginalName != method.OriginalName {
					continue
				}
				mappedParameters, mappedResult := mappedOverrideSourceSignature(ancestor, candidate, args)
				matched := false
				for _, override := range directDeclaredOverrides(ancestor, scope, candidate, mappedParameters, mappedResult, ctx) {
					if override == method {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
				erasedParameters, erasedResult, supported := sourceReflectionMethodSignature(ancestor, candidate, ctx)
				if !supported {
					continue
				}
				key := sourceReflectionDescriptorKey(erasedParameters, erasedResult)
				if descriptors[key] {
					continue
				}
				descriptors[key] = true
				out = append(out, sourceReflectionMethodDescriptor(scope, method, erasedParameters, erasedResult, true, parameters, ctx))
			}
		}
	}
	return append(out, sourceReflectionInheritedInterfaceBridges(scope, ctx, out)...)
}
