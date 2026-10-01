package transpiler

import "go/ast"

// Primitive operator SAMs use the existing native function representation. Their
// signature supplies Java widths at lambda, method-reference and call boundaries.
var primitiveOperatorSAMs = map[string]struct {
	name      string
	method    string
	signature builtinFunctionalInterface
}{
	"java.util.function.IntBinaryOperator": {
		name: "IntBinaryOperator", method: "applyAsInt",
		signature: builtinFunctionalInterface{parameterTypes: []string{"int", "int"}, resultType: "int"},
	},
}

func init() {
	for owner, spec := range primitiveOperatorSAMs {
		registerIntrinsicOwner(owner, true)
		builtinFunctionalInterfaces[spec.name] = spec.signature
		intrinsicFunctionalMethodNames[spec.name] = spec.method
		registerInstanceIntrinsicResultType(spec.name, spec.method, spec.signature.resultType)
		// Preserve the published primitive signature/result metadata; only its
		// legacy function invocation loses to a declared native object contract.
		if _, native := nativeFunctionalContract(spec.name, nil); native {
			continue
		}
		registerInstanceIntrinsic(spec.name, spec.method, func(receiver ast.Expr, arguments []ast.Expr, ctx Ctx) ast.Expr {
			if len(arguments) != len(spec.signature.parameterTypes) {
				return nil
			}
			return &ast.CallExpr{Fun: receiver, Args: arguments}
		})
	}
}

func primitiveOperatorSAMSignature(javaType string, ctx Ctx) (builtinFunctionalInterface, bool) {
	base, arguments := parseJavaTypeString(javaType)
	if _, supported := primitiveOperatorSAMs["java.util.function."+stripJavaQualifier(base)]; !supported {
		return builtinFunctionalInterface{}, false
	}
	if _, binder := lexicalTypeParameterTypeExpr(base, arguments, inScopeTypeParameters(ctx), ctx); binder {
		return builtinFunctionalInterface{}, false
	}
	owner, registered := canonicalIntrinsicOwner(base, ctx)
	spec, known := primitiveOperatorSAMs[owner]
	return spec.signature, registered && known && len(arguments) == 0
}

func primitiveOperatorTypeExpr(javaType string, ctx Ctx) ast.Expr {
	// Keep nominal object views and consuming Execution at a native SAM
	// boundary; only families without a native model use the func fallback.
	if nativeFunctionalFamily(javaType, ctx) != "" {
		return nil
	}
	spec, known := primitiveOperatorSAMSignature(javaType, ctx)
	if !known {
		return nil
	}
	parameters := &ast.FieldList{}
	for _, javaType := range spec.parameterTypes {
		parameters.List = append(parameters.List, &ast.Field{Type: ast.NewIdent(goPrimitiveConversionName(javaType))})
	}
	return &ast.FuncType{
		Params: parameters,
		Results: &ast.FieldList{List: []*ast.Field{
			{Type: ast.NewIdent(goPrimitiveConversionName(spec.resultType))},
		}},
	}
}
