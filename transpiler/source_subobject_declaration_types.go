package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
)

// Declaration receivers already carry allocated Go binder identities. They
// must not cross the Java-spelling converter, which resolves lexical names.
func classSubobjectDeclarationPointerType(scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	if scope == nil || scope.Class == nil {
		return classSubobjectPointerTypeWithGoArguments(scope, nil, ctx)
	}
	typeArgs := typeParamExprs(scope.GoTypeParameterNames())
	if canonicalGenericClass(scope, ctx) {
		// Canonical receivers have no owner binders. Use the physical raw
		// representation shared with their erased method entry points.
		for index, parameter := range scope.TypeParameters {
			typeArgs[index] = javaTypeStringToGoTypeExpr(rawTypeParameterErasure(parameter, scope.TypeParameters), nil, ctx)
		}
	}
	return classSubobjectPointerTypeWithGoArguments(scope, typeArgs, ctx)
}
