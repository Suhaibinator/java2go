package transpiler

import (
	"go/ast"
	"go/token"

	sitter "github.com/smacker/go-tree-sitter"
)

// This file registers the concrete java.lang intrinsics: String, StringBuilder /
// StringBuffer, Math, and the boxed numeric/character types. The table machinery
// lives in intrinsics.go.

func init() {
	registerStringIntrinsics()
	registerLocaleIntrinsics()
	registerStringBuilderIntrinsics()
	registerMathIntrinsics()
	registerNumberIntrinsics()
	registerBoxedTypeIntrinsics()
	registerBoxedObjectIntrinsics()
}

// expectArgs returns true when args has exactly n elements.
func expectArgs(args []ast.Expr, n int) bool {
	return len(args) == n
}

// --- java.lang.String -------------------------------------------------------

func registerStringIntrinsics() {
	for _, spec := range []struct {
		java, goName, result string
		arguments            int
	}{
		{"length", "Length", "int", 0}, {"charAt", "CharAt", "char", 1},
		{"equals", "Equals", "boolean", 1}, {"compareTo", "CompareTo", "int", 1},
		{"concat", "Concat", "java.lang.String", 1},
	} {
		spec := spec
		registerInstanceIntrinsic("String", spec.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != spec.arguments {
				return nil
			}
			return methodCall(recv, spec.goName, args...)
		})
		registerInstanceIntrinsicResultType("String", spec.java, spec.result)
	}
	registerInstanceIntrinsic("String", "isEmpty", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return &ast.BinaryExpr{X: methodCall(recv, "Length"), Op: token.EQL, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
	})
	registerInstanceIntrinsicResultType("String", "isEmpty", "boolean")
	registerInstanceIntrinsic("String", "substring", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		switch len(args) {
		case 1:
			return stdjavaCall(ctx, "JavaStringSubstringFrom", recv, args[0])
		case 2:
			return methodCall(recv, "Substring", args...)
		}
		return nil
	})
	registerInstanceIntrinsicResultType("String", "substring", "java.lang.String")
	for _, spec := range []struct {
		java, helper, result string
		min, max             int
	}{
		{"isBlank", "JavaStringIsBlank", "boolean", 0, 0},
		{"indexOf", "JavaStringIndexOf", "int", 1, 2},
		{"lastIndexOf", "JavaStringLastIndexOf", "int", 1, 2},
		{"contains", "JavaStringContains", "boolean", 1, 1},
		{"startsWith", "JavaStringStartsWith", "boolean", 1, 2},
		{"endsWith", "JavaStringEndsWith", "boolean", 1, 1},
		{"equalsIgnoreCase", "JavaStringEqualsIgnoreCase", "boolean", 1, 1},
		{"toUpperCase", "JavaStringToUpperCase", "java.lang.String", 0, 1},
		{"toLowerCase", "JavaStringToLowerCase", "java.lang.String", 0, 1},
		{"trim", "JavaStringTrim", "java.lang.String", 0, 0},
		{"strip", "JavaStringStrip", "java.lang.String", 0, 0},
		{"split", "JavaStringSplitArray", "java.lang.String[]", 1, 2},
		{"chars", "JavaStringCharsStream", "IntStream", 0, 0},
	} {
		spec := spec
		registerInstanceIntrinsic("String", spec.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) < spec.min || len(args) > spec.max {
				return nil
			}
			return stdjavaCall(ctx, spec.helper, append([]ast.Expr{recv}, args...)...)
		})
		registerInstanceIntrinsicResultType("String", spec.java, spec.result)
	}
	registerInstanceIntrinsic("String", "replace", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "JavaStringReplaceExecution", intrinsicExecutionExpr(ctx), recv, args[0], args[1])
	})
	registerInstanceIntrinsicResultType("String", "replace", "java.lang.String")
	registerInstanceIntrinsic("String", "intern", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "InternJavaString", recv)
	})
	registerInstanceIntrinsicResultType("String", "intern", "java.lang.String")
	registerInstanceIntrinsic("String", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "RequireJavaString", recv)
	})
	registerInstanceIntrinsicResultType("String", "toString", "java.lang.String")
	// Static callsite/method-reference lowering supplies the selected overload.
	registerStaticIntrinsic("String", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "JavaStringValueOfExecution", intrinsicExecutionExpr(ctx), args[0])
	})
	registerStaticIntrinsic("String", "format", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return nil
		}
		return lowerStringFormatReference(args, ctx)
	})
	registerStaticIntrinsic("String", "join", func(_ ast.Expr, _ []ast.Expr, _ Ctx) ast.Expr { return nil })
	for _, method := range []string{"valueOf", "format", "join"} {
		registerStaticIntrinsicResultType("String", method, "java.lang.String")
	}
}

// --- java.lang.StringBuilder / StringBuffer ---------------------------------

func registerStringBuilderIntrinsics() {
	for _, typeName := range []string{"StringBuilder", "StringBuffer"} {
		for _, method := range []string{"append", "insert", "deleteCharAt", "reverse"} {
			registerInstanceIntrinsicResultType(typeName, method, typeName)
		}
		registerInstanceIntrinsicResultType(typeName, "toString", "String")
		registerInstanceIntrinsicResultType(typeName, "length", "int")
		registerInstanceIntrinsicResultType(typeName, "charAt", "char")
		registerConstructorNodeIntrinsic(typeName, lowerStringBuilderConstructor)
		registerInstanceIntrinsic(typeName, "append", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return methodCall(recv, "Append", args[0])
		})
		registerInstanceNodeIntrinsic(typeName, "append", lowerStringBuilderTextCall("append", "Append", 0))
		registerInstanceIntrinsic(typeName, "insert", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 2) {
				return nil
			}
			return methodCall(recv, "Insert", args[0], args[1])
		})
		registerInstanceNodeIntrinsic(typeName, "insert", lowerStringBuilderTextCall("insert", "Insert", 1))
		registerInstanceIntrinsic(typeName, "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return methodCall(recv, "ToJavaString")
		})
		registerInstanceIntrinsic(typeName, "length", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return methodCall(recv, "Length")
		})
		registerInstanceIntrinsic(typeName, "charAt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return methodCall(recv, "CharAt", args[0])
		})
		registerInstanceIntrinsic(typeName, "deleteCharAt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 1) {
				return nil
			}
			return methodCall(recv, "DeleteCharAt", args[0])
		})
		registerInstanceIntrinsic(typeName, "reverse", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 0) {
				return nil
			}
			return methodCall(recv, "Reverse")
		})
	}
}

// Java selects append/insert overloads from the argument's static type. Go's
// rune alias is indistinguishable from int32 at runtime, so pass the Java text
// representation explicitly instead of letting the builder guess the overload.
func lowerStringBuilderTextCall(javaMethod, goMethod string, valueIndex int) nodeIntrinsicGenerator {
	return func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if invocationArgumentCount(invocation) != valueIndex+1 {
			return nil
		}
		object := invocation.ChildByFieldName("object")
		if object == nil {
			return nil
		}
		args := intrinsicArgs(object, javaMethod, source, ctx)
		valueNode := invocationArgumentNode(invocation, valueIndex)
		javaType, _ := inferExprJavaType(valueNode, ctx, source)
		if javaType == "char" {
			return methodCall(recv, goMethod+"Char", args...)
		}
		if javaType == "char[]" {
			return methodCall(recv, goMethod+"Chars", args...)
		}
		converted := javaStringConversionExpr(valueNode, args[valueIndex], ctx, source)
		args[valueIndex] = converted
		return methodCall(recv, goMethod, args...)
	}
}

// --- java.lang.Math ---------------------------------------------------------

func registerMathIntrinsics() {
	registerStaticIntrinsic("Math", "multiplyExact", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "MathMultiplyExact", args...)
	})
	registerStaticIntrinsic("Math", "addExact", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 {
			return nil
		}
		return stdjavaCall(ctx, "MathAddExact", args...)
	})

	// abs is type-preserving in Java. Go's math.Abs is float64-only, so emit a
	// stdjava generic helper that keeps the operand's numeric type.
	registerStaticIntrinsic("Math", "abs", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "MathAbs", args[0])
	})
	registerStaticIntrinsic("Math", "max", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "MathMax", args[0], args[1])
	})
	registerStaticIntrinsic("Math", "min", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "MathMin", args[0], args[1])
	})

	// Trigonometric functions use the fdlibm-compatible runtime so accumulated
	// results retain Java StrictMath's last-bit behavior.
	registerStaticIntrinsic("Math", "pow", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return pkgCall(ctx, "math", "Pow", args[0], args[1])
	})
	registerStaticIntrinsic("Math", "sin", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaMathSin", args[0])
	})
	registerStaticIntrinsic("Math", "cos", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaMathCos", args[0])
	})
	registerStaticIntrinsic("Math", "sqrt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "math", "Sqrt", args[0])
	})
	registerStaticIntrinsic("Math", "floor", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "math", "Floor", args[0])
	})
	registerStaticIntrinsic("Math", "ceil", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "math", "Ceil", args[0])
	})

	// Java's Math.round(double) returns long (int64) using round-half-up. The
	// stdjava helper reproduces that contract.
	registerStaticIntrinsic("Math", "round", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "MathRound", args[0])
	})

	// Math.random() -> rand.Float64()
	registerStaticIntrinsic("Math", "random", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return pkgCall(ctx, "math/rand", "Float64")
	})

	registerStaticFieldIntrinsic("Math", "PI", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("Pi", "math", ctx)
	})
	registerStaticFieldIntrinsic("Math", "E", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("E", "math", ctx)
	})
	for _, method := range []string{"sin", "cos", "pow", "sqrt", "floor", "ceil", "random"} {
		registerStaticIntrinsicResultType("Math", method, "double")
	}
	registerStaticIntrinsicResultType("Math", "round", "long")
	for _, method := range []string{"abs", "min", "max", "round", "addExact", "multiplyExact"} {
		registerStaticIntrinsicDerivedResultType("Math", method, func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			parameter := intrinsicMathParameterJavaType(invocation, ctx, source)
			if method == "round" {
				if parameter == "float" {
					return "int", true
				}
				return "long", true
			}
			return parameter, true
		})
	}
	registerStaticFieldIntrinsicResultType("Math", "PI", "double")
	registerStaticFieldIntrinsicResultType("Math", "E", "double")
}

func registerNumberIntrinsics() {
	accessors := map[string]string{
		"byteValue": "NumberByteValue", "shortValue": "NumberShortValue",
		"intValue": "NumberIntValue", "longValue": "NumberLongValue",
		"floatValue": "NumberFloatValue", "doubleValue": "NumberDoubleValue",
	}
	results := map[string]string{
		"byteValue": "byte", "shortValue": "short", "intValue": "int",
		"longValue": "long", "floatValue": "float", "doubleValue": "double",
	}
	for _, typeName := range []string{"Number", "Byte", "Short", "Integer", "Long", "Float", "Double"} {
		for methodName, runtimeName := range accessors {
			runtimeName := runtimeName
			registerInstanceIntrinsic(typeName, methodName, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if !expectArgs(args, 0) {
					return nil
				}
				return stdjavaCall(ctx, runtimeName, recv)
			})
			registerInstanceIntrinsicResultType(typeName, methodName, results[methodName])
		}
	}
}

// --- boxed types: Integer / Long / Double / Boolean / Character -------------
func registerBoxedTypeIntrinsics() {
	// Integer
	registerStaticIntrinsic("Integer", "toHexString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaIntegerToHexString", args[0])
	})
	registerStaticIntrinsicResultType("Integer", "toHexString", "java.lang.String")
	registerStaticIntrinsic("Integer", "parseInt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaIntegerParseInt", args[0])
	})
	registerStaticIntrinsic("Integer", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseInt", args[0])
	})
	registerStaticIntrinsic("Integer", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaStringValueOfInt", args[0])
	})
	registerStaticFieldIntrinsic("Integer", "MAX_VALUE", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("MaxInt32", "math", ctx)
	})
	registerStaticFieldIntrinsic("Integer", "MIN_VALUE", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("MinInt32", "math", ctx)
	})

	// Long
	registerStaticIntrinsic("Long", "parseLong", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "JavaLongParseLong", args[0])
	})
	registerStaticIntrinsic("Long", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseLong", args[0])
	})
	registerStaticIntrinsic("Long", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "fmt", "Sprint", args[0])
	})
	registerStaticFieldIntrinsic("Long", "MAX_VALUE", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("MaxInt64", "math", ctx)
	})
	registerStaticFieldIntrinsic("Long", "MIN_VALUE", func(ctx Ctx) ast.Expr {
		return qualifiedNameExpr("MinInt64", "math", ctx)
	})

	// Double
	registerStaticIntrinsic("Double", "parseDouble", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseDouble", args[0])
	})
	registerStaticIntrinsic("Double", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "DoubleValueOf", args[0])
	})
	registerStaticIntrinsicResultType("Double", "valueOf", "Double")

	// The floating-point comparison statics and the special-value constants.
	// Double.compare is a total order (NaN last, -0.0 before 0.0) that differs
	// from Go's `<`, so it maps to the runtime's Java-compatible comparison
	// rather than to a subtraction.
	registerStaticIntrinsic("Double", "compare", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "DoubleCompare", args[0], args[1])
	})
	registerStaticIntrinsicResultType("Double", "compare", "int")
	registerStaticIntrinsic("Float", "compare", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "FloatCompare", args[0], args[1])
	})
	registerStaticIntrinsicResultType("Float", "compare", "int")
	registerStaticIntrinsic("Double", "isNaN", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "DoubleIsNaN", args[0])
	})
	registerStaticIntrinsicResultType("Double", "isNaN", "boolean")

	for _, spec := range []struct{ field, runtime, resultType string }{
		{"NaN", "DoubleNaN", "double"},
		{"POSITIVE_INFINITY", "DoublePositiveInfinity", "double"},
		{"NEGATIVE_INFINITY", "DoubleNegativeInfinity", "double"},
	} {
		registerStaticFieldIntrinsic("Double", spec.field, func(ctx Ctx) ast.Expr {
			return stdjavaCall(ctx, spec.runtime)
		})
		registerStaticFieldIntrinsicResultType("Double", spec.field, spec.resultType)
	}
	registerStaticFieldIntrinsic("Float", "NaN", func(ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "FloatNaN")
	})
	registerStaticFieldIntrinsicResultType("Float", "NaN", "float")
	registerStaticIntrinsic("Double", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "DoubleToString", args[0])
	})

	// Float
	registerStaticIntrinsic("Float", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "FloatToString", args[0])
	})

	// Boolean
	registerStaticIntrinsic("Boolean", "parseBoolean", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseBoolean", args[0])
	})
	registerStaticIntrinsic("Boolean", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseBoolean", args[0])
	})
	registerStaticIntrinsic("Boolean", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "fmt", "Sprint", args[0])
	})

	// Character (static predicates and conversions operate on a rune)
	registerStaticIntrinsic("Character", "isDigit", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsDigit", args[0])
	})
	registerStaticIntrinsic("Character", "isLetter", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsLetter", args[0])
	})
	registerStaticIntrinsic("Character", "isLetterOrDigit", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsLetterOrDigit", args[0])
	})
	registerStaticIntrinsic("Character", "isWhitespace", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsWhitespace", args[0])
	})
	registerStaticIntrinsic("Character", "isUpperCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsUpperCase", args[0])
	})
	registerStaticIntrinsic("Character", "isLowerCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharIsLowerCase", args[0])
	})
	registerStaticIntrinsic("Character", "toUpperCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharToUpperCase", args[0])
	})
	registerStaticIntrinsic("Character", "toLowerCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "CharToLowerCase", args[0])
	})
}

// All wrappers are references. Factories and constructors must remain distinct:
// valueOf may return a cached object, whereas new always preserves fresh identity.
func registerBoxedObjectIntrinsics() {
	registerIntrinsicOwner("java.lang.Integer", true)
	registerIntrinsicOwner("java.lang.Long", true)
	// Imported calls use the same lowering only after declared overload applicability.
	registerStaticIntrinsicImportSignature("Integer", "parseInt", "java.lang.String")
	registerStaticIntrinsicImportSignature("Integer", "parseInt", "java.lang.String", "int")
	registerStaticIntrinsicImportSignature("Long", "parseLong", "java.lang.String")
	registerStaticIntrinsicImportSignature("Long", "parseLong", "java.lang.String", "int")
	for _, spec := range []struct{ wrapper, primitive, parser string }{
		{"Boolean", "boolean", "ParseBoolean"}, {"Byte", "byte", "ParseByte"},
		{"Short", "short", "ParseShort"}, {"Character", "char", ""},
		{"Integer", "int", "JavaIntegerParseInt"}, {"Long", "long", "JavaLongParseLong"},
		{"Float", "float", "ParseFloat"}, {"Double", "double", "ParseDouble"},
	} {
		registerStaticIntrinsic(spec.wrapper, "valueOf", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) < 1 || len(args) > 2 {
				return nil
			}
			return stdjavaCall(ctx, spec.wrapper+"ValueOf", args...)
		})
		registerStaticIntrinsicResultType(spec.wrapper, "valueOf", "java.lang."+spec.wrapper)
		if spec.parser != "" {
			method := "parse" + spec.wrapper
			if spec.wrapper == "Integer" {
				method = "parseInt"
			}
			registerStaticIntrinsic(spec.wrapper, method, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) < 1 || len(args) > 2 {
					return nil
				}
				return stdjavaCall(ctx, spec.parser, args...)
			})
			registerStaticIntrinsicResultType(spec.wrapper, method, spec.primitive)
		}
		for _, method := range []struct {
			java, goName, result string
			count                int
		}{
			{"equals", "Equals", "boolean", 1}, {"hashCode", "HashCode", "int", 0},
			{"compareTo", "CompareTo", "int", 1}, {"toString", "String", "String", 0},
		} {
			registerInstanceIntrinsic(spec.wrapper, method.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != method.count {
					return nil
				}
				return methodCall(recv, method.goName, args...)
			})
			registerInstanceIntrinsicResultType(spec.wrapper, method.java, method.result)
		}
		registerInstanceIntrinsic(spec.wrapper, "getClass", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectGetClass", recv)
		})
		registerInstanceIntrinsicResultType(spec.wrapper, "getClass", "Class")
		registerStaticFieldIntrinsic(spec.wrapper, "TYPE", func(ctx Ctx) ast.Expr {
			descriptor, ok := javaTypeDescriptorExpr(spec.primitive, ctx)
			if !ok {
				return nil
			}
			return stdjavaCall(ctx, "ClassLiteral", descriptor)
		})
		registerStaticFieldIntrinsicResultType(spec.wrapper, "TYPE", "Class")
		registerStaticIntrinsic(spec.wrapper, "hashCode", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return methodCall(stdjavaCall(ctx, "Box"+spec.wrapper, args[0]), "HashCode")
		})
		registerStaticIntrinsicResultType(spec.wrapper, "hashCode", "int")
		registerStaticIntrinsicResultType(spec.wrapper, "toString", "String")
		if spec.wrapper == "Byte" || spec.wrapper == "Short" || spec.wrapper == "Character" {
			registerStaticIntrinsic(spec.wrapper, "toString", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != 1 {
					return nil
				}
				return methodCall(stdjavaCall(ctx, "Box"+spec.wrapper, args[0]), "String")
			})
		}
		if spec.wrapper != "Float" && spec.wrapper != "Double" {
			registerStaticIntrinsic(spec.wrapper, "compare", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != 2 {
					return nil
				}
				return methodCall(stdjavaCall(ctx, "Box"+spec.wrapper, args[0]), "CompareTo", stdjavaCall(ctx, "Box"+spec.wrapper, args[1]))
			})
			registerStaticIntrinsicResultType(spec.wrapper, "compare", "int")
		}
		if spec.wrapper == "Boolean" || spec.wrapper == "Character" {
			method := "booleanValue"
			if spec.wrapper == "Character" {
				method = "charValue"
			}
			registerInstanceIntrinsic(spec.wrapper, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != 0 {
					return nil
				}
				return stdjavaCall(ctx, "Unbox"+spec.wrapper, recv)
			})
			registerInstanceIntrinsicResultType(spec.wrapper, method, spec.primitive)
		}
	}
	for _, constant := range []struct{ field, value string }{{"TRUE", "true"}, {"FALSE", "false"}} {
		registerStaticFieldIntrinsic("Boolean", constant.field, func(ctx Ctx) ast.Expr {
			return stdjavaCall(ctx, "BoxBoolean", &ast.Ident{Name: constant.value})
		})
		registerStaticFieldIntrinsicResultType("Boolean", constant.field, "java.lang.Boolean")
	}
	for _, spec := range []struct{ wrapper, primitive, min, max string }{
		{"Byte", "byte", "-128", "127"}, {"Short", "short", "-32768", "32767"},
		{"Character", "char", "0", "65535"},
	} {
		for _, limit := range []struct{ field, value string }{{"MIN_VALUE", spec.min}, {"MAX_VALUE", spec.max}} {
			registerStaticFieldIntrinsic(spec.wrapper, limit.field, func(ctx Ctx) ast.Expr {
				return &ast.CallExpr{Fun: javaTypeStringToGoTypeExpr(spec.primitive, nil, ctx), Args: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: limit.value}}}
			})
			registerStaticFieldIntrinsicResultType(spec.wrapper, limit.field, spec.primitive)
		}
	}
	for _, wrapper := range []string{"Integer", "Long"} {
		primitive := "int"
		if wrapper == "Long" {
			primitive = "long"
		}
		registerStaticFieldIntrinsicResultType(wrapper, "MIN_VALUE", primitive)
		registerStaticFieldIntrinsicResultType(wrapper, "MAX_VALUE", primitive)
	}
	for _, spec := range []struct{ wrapper, primitive, min, max, normal string }{
		{"Float", "float", "SmallestNonzeroFloat32", "MaxFloat32", "0x1p-126"},
		{"Double", "double", "SmallestNonzeroFloat64", "MaxFloat64", "0x1p-1022"},
	} {
		for _, limit := range []struct{ field, name string }{{"MIN_VALUE", spec.min}, {"MAX_VALUE", spec.max}} {
			registerStaticFieldIntrinsic(spec.wrapper, limit.field, func(ctx Ctx) ast.Expr {
				return &ast.CallExpr{Fun: javaTypeStringToGoTypeExpr(spec.primitive, nil, ctx), Args: []ast.Expr{qualifiedNameExpr(limit.name, "math", ctx)}}
			})
			registerStaticFieldIntrinsicResultType(spec.wrapper, limit.field, spec.primitive)
		}
		registerStaticFieldIntrinsic(spec.wrapper, "MIN_NORMAL", func(ctx Ctx) ast.Expr {
			return &ast.CallExpr{Fun: javaTypeStringToGoTypeExpr(spec.primitive, nil, ctx), Args: []ast.Expr{&ast.BasicLit{Kind: token.FLOAT, Value: spec.normal}}}
		})
		registerStaticFieldIntrinsicResultType(spec.wrapper, "MIN_NORMAL", spec.primitive)
		for _, limit := range []struct{ field, runtime string }{{"POSITIVE_INFINITY", "DoublePositiveInfinity"}, {"NEGATIVE_INFINITY", "DoubleNegativeInfinity"}} {
			registerStaticFieldIntrinsic(spec.wrapper, limit.field, func(ctx Ctx) ast.Expr {
				value := ast.Expr(stdjavaCall(ctx, limit.runtime))
				if spec.wrapper == "Float" {
					value = callIdent("float32", value)
				}
				return value
			})
			registerStaticFieldIntrinsicResultType(spec.wrapper, limit.field, spec.primitive)
		}
		for _, method := range []string{"isNaN", "isInfinite"} {
			predicate := func(value ast.Expr, ctx Ctx) ast.Expr {
				if method == "isInfinite" {
					return pkgCall(ctx, "math", "IsInf", callIdent("float64", value), &ast.BasicLit{Kind: token.INT, Value: "0"})
				}
				return pkgCall(ctx, "math", "IsNaN", callIdent("float64", value))
			}
			if spec.wrapper != "Double" || method != "isNaN" {
				registerStaticIntrinsic(spec.wrapper, method, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
					if len(args) != 1 {
						return nil
					}
					return predicate(args[0], ctx)
				})
			}
			registerStaticIntrinsicResultType(spec.wrapper, method, "boolean")
			registerInstanceIntrinsic(spec.wrapper, method, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) != 0 {
					return nil
				}
				return predicate(stdjavaCall(ctx, "Unbox"+spec.wrapper, recv), ctx)
			})
			registerInstanceIntrinsicResultType(spec.wrapper, method, "boolean")
		}
	}
	for _, method := range []string{"isDigit", "isLetter", "isLetterOrDigit", "isWhitespace", "isUpperCase", "isLowerCase"} {
		registerStaticIntrinsicResultType("Character", method, "boolean")
	}
	for _, method := range []string{"toUpperCase", "toLowerCase"} {
		registerStaticIntrinsicResultType("Character", method, "char")
	}
	registerInstanceIntrinsic("Comparable", "compareTo", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		execution := executionExpr(ctx)
		if execution == nil {
			execution = &ast.Ident{Name: "nil"}
		}
		return stdjavaCall(ctx, "ComparableCompareToExecution", execution, recv, args[0])
	})
	registerInstanceIntrinsicResultType("Comparable", "compareTo", "int")
	for receiverType := range builtinExceptionTypes {
		registerInstanceIntrinsic(receiverType, "getClass", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectGetClass", recv)
		})
		registerInstanceIntrinsicResultType(receiverType, "getClass", "Class")
	}
	for _, receiverType := range []string{"Object", "Number", "CharSequence"} {
		registerInstanceIntrinsic(receiverType, "equals", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectEqualsExecution", intrinsicExecutionExpr(ctx), recv, args[0])
		})
		registerInstanceIntrinsicResultType(receiverType, "equals", "boolean")
		registerInstanceIntrinsic(receiverType, "hashCode", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), recv)
		})
		registerInstanceIntrinsicResultType(receiverType, "hashCode", "int")
		registerInstanceIntrinsic(receiverType, "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "JavaStringValueOfExecution", intrinsicExecutionExpr(ctx), stdjavaCall(ctx, "ReferenceRequireNonNull", recv))
		})
		registerInstanceIntrinsicResultType(receiverType, "toString", "String")
		registerInstanceIntrinsic(receiverType, "getClass", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectGetClass", recv)
		})
		registerInstanceIntrinsicResultType(receiverType, "getClass", "Class")
	}
}

func intrinsicExecutionExpr(ctx Ctx) ast.Expr {
	if execution := executionExpr(ctx); execution != nil {
		return execution
	}
	return &ast.Ident{Name: "nil"}
}

func registerLocaleIntrinsics() {
	for _, name := range []string{"ROOT", "ENGLISH", "US"} {
		name := name
		registerStaticFieldIntrinsic("Locale", name, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("Locale"+name, ctx) })
		registerStaticFieldIntrinsicResultType("Locale", name, "Locale")
	}
	registerStaticIntrinsic("Locale", "forLanguageTag", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 1 {
			return nil
		}
		return stdjavaCall(ctx, "LocaleForLanguageTagJavaString", args[0])
	})
	registerStaticIntrinsicResultType("Locale", "forLanguageTag", "Locale")
}
