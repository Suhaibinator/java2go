package transpiler

import "go/ast"

var atomicUpdaterOperators = map[string]struct {
	primitive, method string
	arity             int
}{
	"IntUnaryOperator": {"int", "applyAsInt", 1}, "IntBinaryOperator": {"int", "applyAsInt", 2},
	"LongUnaryOperator": {"long", "applyAsLong", 1}, "LongBinaryOperator": {"long", "applyAsLong", 2},
	"UnaryOperator": {"T", "apply", 1}, "BinaryOperator": {"T", "apply", 2},
}

func atomicUpdaterOperatorName(javaType string, ctx Ctx) string {
	owner, known := canonicalIntrinsicOwner(javaType, ctx)
	if !known {
		return ""
	}
	for name := range atomicUpdaterOperators {
		if owner == "java.util.function."+name {
			return name
		}
	}
	return ""
}
func isExternalAtomicUpdaterOperator(javaType string, ctx Ctx) bool {
	return atomicUpdaterOperatorName(javaType, ctx) != ""
}
func registerAtomicUpdaterOperators() {
	registerAdditionalNativeFunctionalViews()
	registerNativeFunctionalDefaults()
	for name, spec := range atomicUpdaterOperators {
		name, spec := name, spec
		registerIntrinsicOwner("java.util.function."+name, true)
		parameters := make([]string, spec.arity)
		for i := range parameters {
			parameters[i] = spec.primitive
		}
		sam := builtinFunctionalInterface{parameterTypes: parameters, resultType: spec.primitive}
		if spec.primitive == "T" {
			sam.typeParameters = []string{"T"}
		}
		builtinFunctionalInterfaces[name] = sam
		intrinsicFunctionalMethodNames[name] = spec.method
		registerInstanceIntrinsic(name, spec.method, func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != spec.arity {
				return nil
			}
			return stdjavaCall(ctx, "Call"+name+"Execution", append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
		})
	}
}
func atomicUpdaterOperatorAdapter(value ast.Expr, javaType string, executionAware bool, ctx Ctx) ast.Expr {
	name := atomicUpdaterOperatorName(javaType, ctx)
	if name == "" {
		return nil
	}
	constructor := "New" + name + "FuncAdapter"
	if !executionAware {
		constructor = "NewPlain" + name + "FuncAdapter"
	}
	_, args := parseJavaTypeString(javaType)
	if name == "UnaryOperator" || name == "BinaryOperator" {
		if len(args) != 1 {
			return nil
		}
		return stdjavaGenericCall(ctx, constructor, []ast.Expr{javaTypeStringToGoTypeExpr(args[0], inScopeTypeParameters(ctx), ctx)}, []ast.Expr{value})
	}
	return stdjavaCall(ctx, constructor, value)
}
func atomicUpdaterReferenceCallback(value ast.Expr, operator, javaType string, ctx Ctx) ast.Expr {
	descriptor, known := javaSourceTypeDescriptorExpr(javaType, ctx)
	if !known {
		return nil
	}
	target := abstractClassToInterface(javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx), javaType, ctx)
	return stdjavaGenericCall(ctx, "Reference"+operator+"CallbackExecution", []ast.Expr{target}, []ast.Expr{value, descriptor})
}
