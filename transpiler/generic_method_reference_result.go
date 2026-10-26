package transpiler

import (
	"go/ast"
	"strings"
)

// A generic instance method's callable entry returns its erased descriptor.
// Restore the consuming SAM's view before its usual boxing/widening conversion,
// just as for an ordinary invocation of that same erased entry.
func genericMethodReferenceResult(value ast.Expr, resolution *methodResolution, target *invocationTargetInfo, unbound bool, ctx Ctx) ast.Expr {
	actual := methodReferenceDeclaredResultType(resolution, target, ctx)
	if genericMethodHasErasedEntry(resolution.def) {
		parameters, result := methodReferenceJavaSignature(ctx)
		if result == "void" {
			return value
		}
		if unbound && len(parameters) > 0 {
			parameters = parameters[1:]
		}
		bindings := methodReferenceInferredBindings(resolution.def, parameters, result, ctx)
		arguments := make([]ast.Expr, len(resolution.def.TypeParameters))
		for i, parameter := range resolution.def.TypeParameters {
			javaType := bindings[parameter.Name]
			if primitive, ok := javaPrimitiveType(javaType); ok {
				javaType = "java.lang." + ternaryBoxedJavaType(primitive)
				bindings[parameter.Name] = javaType
			}
			bindings[parameter.EmittedName()] = javaType
			arguments[i] = javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
		}
		value = genericMethodProjectedResult(value, resolution.def, arguments, bindings, result, ctx)
		actual = substituteJavaTypeParams(actual, bindings)
		for _, parameter := range resolution.def.TypeParameters {
			declared := strings.TrimSpace(resolution.def.OriginalType)
			if declared != parameter.Name && declared != parameter.EmittedName() {
				continue
			}
			// A method binder shadows an owner binder with the same spelling.
			// Recover its inferred view from the method declaration before unboxing.
			actual = bindings[parameter.Name]
			// Projection already returns the consuming reference view; applying
			// inheritance conversion again would project its parent twice.
			if _, primitive := javaPrimitiveType(result); result != "" && !primitive {
				actual = result
			}
			break
		}
	}
	return methodReferenceResultConversion(value, actual, ctx)
}
