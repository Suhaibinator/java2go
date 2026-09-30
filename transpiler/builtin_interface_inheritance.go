package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

func builtinInterfaceParentMethod(javaType string, ctx Ctx) *symbol.Definition {
	base, arguments := parseJavaTypeString(javaType)
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return nil
	}
	name := stripJavaQualifier(base)
	methodName := intrinsicFunctionalMethodNames[name]
	if name == "Callable" {
		methodName = "call"
	}
	if name == "Comparator" {
		methodName = "compare"
	}
	if methodName == "" {
		return nil
	}
	if len(arguments) == 0 {
		for range builtinFunctionalInterfaces[name].typeParameters {
			arguments = append(arguments, "Object")
		}
	}
	method, bindings, ok := builtinFunctionalInterfaceMethod(name, arguments)
	if !ok {
		return nil
	}
	method.OriginalName, method.Name = methodName, symbol.Uppercase(methodName)
	method.OriginalType = substituteJavaTypeParams(method.OriginalType, bindings)
	for _, parameter := range method.Parameters {
		parameter.OriginalType = substituteJavaTypeParams(parameter.OriginalType, bindings)
	}
	return method
}

// Source interfaces inherit the Java method contract of external functional
// interfaces. Their Go interface cannot embed the runtime's function value
// representation, so retain the abstract method in the source symbol hierarchy.
func prepareBuiltinInterfaceMethods() {
	for _, scope := range allSourceClassScopes() {
		if !scope.IsInterface {
			continue
		}
		ctx := Ctx{currentClass: scope, currentFile: findFileScopeForClassScope(scope)}
		for _, parent := range scope.ImplementedInterfaces {
			method := builtinInterfaceParentMethod(parent, ctx)
			if method == nil {
				continue
			}
			exists := false
			for _, own := range scope.Methods {
				if interfaceMethodSignature(own) == interfaceMethodSignature(method) {
					exists = true
					break
				}
			}
			if !exists {
				scope.Methods = append(scope.Methods, method)
			}
		}
	}
}

func builtinInheritedMethodField(method *symbol.Definition, typeParams []string, ctx Ctx) *ast.Field {
	params := &ast.FieldList{}
	for _, parameter := range method.Parameters {
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: javaTypeStringToGoTypeExpr(parameter.OriginalType, typeParams, ctx)})
	}
	function := &ast.FuncType{Params: params}
	if result := strings.TrimSpace(method.OriginalType); result != "" && result != "void" {
		function.Results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(result, typeParams, ctx)}}}
	}
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(method.Name)}, Type: function}
}
