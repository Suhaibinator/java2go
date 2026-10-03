package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func bigMathOwner(javaType string, ctx Ctx) (string, bool) {
	owner, known := canonicalIntrinsicOwner(javaType, ctx)
	return owner, known && (owner == "java.math.BigInteger" || owner == "java.math.BigDecimal")
}
func bigMathRuntimeTypeExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	owner, known := canonicalIntrinsicOwner(javaType, ctx)
	if !known {
		return nil, false
	}
	if owner != "java.math.BigInteger" && owner != "java.math.BigDecimal" {
		return nil, false
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr(stripJavaQualifier(owner), ctx)}, true
}
func bigMathExpectedArgumentTypes(owner, method string, count int) ([]string, bool) {
	if count != 1 {
		return nil, false
	}
	switch owner {
	case "BigInteger", "BigDecimal":
		if method == "equals" {
			return []string{"Object"}, true
		}
		if method == "compareTo" {
			return []string{"java.math." + owner}, true
		}
		if owner == "BigInteger" && method == "valueOf" {
			return []string{"long"}, true
		}
		if owner == "BigInteger" {
			switch method {
			case "add", "subtract", "multiply", "divide", "mod":
				return []string{"java.math.BigInteger"}, true
			}
		}
	case "Float":
		if method == "floatToRawIntBits" || method == "floatToIntBits" {
			return []string{"float"}, true
		}
	case "Double":
		if method == "doubleToRawLongBits" || method == "doubleToLongBits" {
			return []string{"double"}, true
		}
	}
	return nil, false
}
func init() {
	for _, name := range []string{"BigInteger", "BigDecimal"} {
		registerIntrinsicOwner("java.math."+name, true)
		registerConstructorNodeIntrinsic(name, func(_ []ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			actual, known := inferExprJavaType(invocationArgumentNode(node, 0), ctx, source)
			if !known || (!isJavaStringType(actual) && actual != "null") {
				return nil
			}
			// String reference identity/null conversions use the reference coercion
			// path; convertJavaValue only reports the conversions it actively emits.
			converted := coerceArgumentToExpectedType(args[0], invocationArgumentNode(node, 0), "String", ctx, source)
			return stdjavaCall(ctx, "New"+name+"JavaString", converted)
		})
		registerInstanceIntrinsic(name, "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return methodCall(recv, "StringJava2goExecution", intrinsicExecutionExpr(ctx))
		})
		registerInstanceIntrinsicResultType(name, "toString", "String")
		for _, method := range []struct {
			java, goName, result string
			arity                int
		}{
			{"equals", "Equals", "boolean", 1}, {"compareTo", "CompareTo", "int", 1}, {"hashCode", "HashCode", "int", 0},
			{"byteValue", "ByteValue", "byte", 0}, {"shortValue", "ShortValue", "short", 0}, {"intValue", "IntValue", "int", 0}, {"longValue", "LongValue", "long", 0}, {"floatValue", "FloatValue", "float", 0}, {"doubleValue", "DoubleValue", "double", 0},
		} {
			registerInstanceIntrinsic(name, method.java, ioMethod(method.goName, method.arity))
			registerInstanceIntrinsicResultType(name, method.java, method.result)
		}
	}
	registerStaticIntrinsic("BigInteger", "valueOf", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "BigIntegerValueOf", args...)
	})
	registerStaticIntrinsicResultType("BigInteger", "valueOf", "java.math.BigInteger")
	for _, method := range []struct{ java, goName string }{
		{"add", "Add"}, {"subtract", "Subtract"}, {"multiply", "Multiply"}, {"divide", "Divide"}, {"mod", "Mod"},
	} {
		registerInstanceIntrinsic("BigInteger", method.java, ioMethod(method.goName, 1))
		registerInstanceIntrinsicResultType("BigInteger", method.java, "java.math.BigInteger")
	}
	registerInstanceIntrinsic("BigInteger", "bitLength", ioMethod("BitLength", 0))
	registerInstanceIntrinsicResultType("BigInteger", "bitLength", "int")
	registerInstanceIntrinsic("BigDecimal", "scale", ioMethod("Scale", 0))
	registerInstanceIntrinsicResultType("BigDecimal", "scale", "int")
	for _, method := range []struct{ owner, java, goName, result string }{
		{"Float", "floatToRawIntBits", "FloatToRawIntBits", "int"}, {"Float", "floatToIntBits", "FloatToIntBits", "int"},
		{"Double", "doubleToRawLongBits", "DoubleToRawLongBits", "long"}, {"Double", "doubleToLongBits", "DoubleToLongBits", "long"},
	} {
		registerStaticIntrinsic(method.owner, method.java, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, method.goName, args...)
		})
		registerStaticIntrinsicResultType(method.owner, method.java, method.result)
	}
}
