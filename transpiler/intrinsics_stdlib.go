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
	// length() -> int32(len(s))
	// length() counts characters. Go's len() counts UTF-8 bytes, which differs
	// from Java's UTF-16 code-unit count for non-ASCII text, so use the
	// rune-based stdjava helper (matches Java for BMP characters).
	registerInstanceIntrinsic("String", "length", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "StringLength", recv)
	})

	// isEmpty() -> len(s) == 0
	registerInstanceIntrinsic("String", "isEmpty", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return &ast.BinaryExpr{X: callIdent("len", recv), Op: token.EQL, Y: &ast.BasicLit{Kind: token.INT, Value: "0"}}
	})

	// isBlank() -> stdjava.StringIsBlank(s)  (rune/whitespace aware, Java 11+)
	registerInstanceIntrinsic("String", "isBlank", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "StringIsBlank", recv)
	})

	// charAt(i) -> stdjava.StringCharAt(s, i)  (returns rune; rune-indexed)
	registerInstanceIntrinsic("String", "charAt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringCharAt", recv, args[0])
	})

	// substring(begin) / substring(begin, end) -> rune-indexed helpers.
	registerInstanceIntrinsic("String", "substring", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		switch len(args) {
		case 1:
			return stdjavaCall(ctx, "StringSubstring", recv, args[0])
		case 2:
			return stdjavaCall(ctx, "StringSubstringRange", recv, args[0], args[1])
		}
		return nil
	})

	// indexOf / lastIndexOf -> rune-index helpers (only the (String) overload;
	// the (int ch) overload falls through for now).
	registerInstanceIntrinsic("String", "indexOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringIndexOf", recv, args[0])
	})
	registerInstanceIntrinsic("String", "lastIndexOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringLastIndexOf", recv, args[0])
	})

	// contains(s) -> strings.Contains(s, sub)
	registerInstanceIntrinsic("String", "contains", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "strings", "Contains", recv, args[0])
	})

	// startsWith / endsWith -> strings.HasPrefix / HasSuffix
	registerInstanceIntrinsic("String", "startsWith", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "strings", "HasPrefix", recv, args[0])
	})
	registerInstanceIntrinsic("String", "endsWith", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "strings", "HasSuffix", recv, args[0])
	})

	// equals(o) -> s == o  (Java String.equals on two strings is value equality).
	registerInstanceIntrinsic("String", "equals", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringEquals", recv, args[0])
	})

	// concat(s) -> recv + s. Both operands are Java references, so normalize the
	// argument as well as the receiver and retain concat(null)'s NPE behavior.
	registerInstanceIntrinsic("String", "concat", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return &ast.BinaryExpr{X: recv, Op: token.ADD, Y: stdjavaCall(ctx, "StringRequireNonNull", args[0])}
	})

	// equalsIgnoreCase(o) -> strings.EqualFold via stdjava wrapper.
	registerInstanceIntrinsic("String", "equalsIgnoreCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringEqualsIgnoreCase", recv, args[0])
	})

	// compareTo(o) -> int32(strings.Compare(...)) via stdjava wrapper.
	registerInstanceIntrinsic("String", "compareTo", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringCompareTo", recv, args[0])
	})

	// toUpperCase / toLowerCase -> strings.ToUpper / ToLower
	registerInstanceIntrinsic("String", "toUpperCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return pkgCall(ctx, "strings", "ToUpper", recv)
	})
	registerInstanceIntrinsic("String", "toLowerCase", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return pkgCall(ctx, "strings", "ToLower", recv)
	})

	// trim / strip -> strings.TrimSpace. Java's trim strips <= U+0020 while strip
	// is Unicode-whitespace aware; strings.TrimSpace matches strip and is a close
	// approximation of trim for the common ASCII case.
	registerInstanceIntrinsic("String", "trim", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return pkgCall(ctx, "strings", "TrimSpace", recv)
	})
	registerInstanceIntrinsic("String", "strip", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return pkgCall(ctx, "strings", "TrimSpace", recv)
	})

	// replace(old, new) -> stdjava.StringReplace (strings.ReplaceAll). Java's
	// replace replaces all literal occurrences.
	registerInstanceIntrinsic("String", "replace", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return stdjavaCall(ctx, "StringReplace", recv, args[0], args[1])
	})

	// split(regex) -> stdjava.StringSplitArray, which treats the separator as a Java
	// regex (RE2 approximation), splits literally when the pattern has no regex
	// metacharacters, and removes trailing empty strings like Java's one-arg split.
	registerInstanceIntrinsic("String", "split", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringSplitArray", recv, args[0])
	})

	// chars() -> stdjava.StringCharsStream(s). Java's String.chars returns an
	// IntStream, so it must be a stream for the pipeline operations chained onto
	// it to resolve; the previous []rune result only worked when the caller
	// immediately ranged over it.
	registerInstanceIntrinsic("String", "chars", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "StringCharsStream", recv)
	})
	registerInstanceIntrinsicResultType("String", "chars", "IntStream")

	// --- static String methods ---

	// String.valueOf(x) uses the stdjava conversion bridge so floating-point
	// values, null, and generated fmt.Stringer implementations (notably enums)
	// retain Java's textual form.
	registerStaticIntrinsic("String", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "StringValueOf", args[0])
	})

	// String.format(fmt, args...) -> fmt.Sprintf(convertedFmt, args...). The Java
	// format string is passed through unchanged, which works for the %s/%d/%f
	// conversions shared with Go; locale and Java-specific conversions are not
	// translated.
	registerStaticIntrinsic("String", "format", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) == 0 {
			return nil
		}
		return pkgCall(ctx, "fmt", "Sprintf", args...)
	})

	// String.join(sep, elements) -> strings.Join(elements, sep). Only the
	// (CharSequence, Iterable/array) overload with two arguments is handled.
	registerStaticIntrinsic("String", "join", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return pkgCall(ctx, "strings", "Join", args[1], args[0])
	})
	for _, method := range []string{"length", "indexOf", "lastIndexOf", "compareTo"} {
		registerInstanceIntrinsicResultType("String", method, "int")
	}
	registerInstanceIntrinsicResultType("String", "charAt", "char")
	for _, method := range []string{"isEmpty", "isBlank", "contains", "startsWith", "endsWith", "equals", "equalsIgnoreCase"} {
		registerInstanceIntrinsicResultType("String", method, "boolean")
	}
	for _, method := range []string{"substring", "concat", "toUpperCase", "toLowerCase", "trim", "strip", "replace"} {
		registerInstanceIntrinsicResultType("String", method, "String")
	}
	registerInstanceIntrinsicResultType("String", "split", "String[]")
	for _, method := range []string{"valueOf", "format", "join"} {
		registerStaticIntrinsicResultType("String", method, "String")
	}
}

// --- java.lang.StringBuilder / StringBuffer ---------------------------------

func registerStringBuilderIntrinsics() {
	for _, typeName := range []string{"StringBuilder", "StringBuffer"} {
		// new StringBuilder() / new StringBuilder(String)
		registerConstructorIntrinsic(typeName, func(typeArgs, args []ast.Expr, ctx Ctx) ast.Expr {
			switch len(args) {
			case 0:
				return stdjavaCall(ctx, "NewStringBuilder")
			case 1:
				return stdjavaCall(ctx, "NewStringBuilderString", args[0])
			}
			return nil
		})
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
			return methodCall(recv, "String")
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
		converted := javaStringConversionExpr(valueNode, args[valueIndex], ctx, source)
		switch javaType {
		case "byte", "short", "int", "long", "boolean":
			// Concatenation can hand these values directly to fmt, but builders
			// need a string to distinguish a numeric int32 from a character.
			converted = javaStringValueOfForType(javaType, converted, ctx)
		}
		args[valueIndex] = converted
		return methodCall(recv, goMethod, args...)
	}
}

// --- java.lang.Math ---------------------------------------------------------

func registerMathIntrinsics() {
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
	for _, method := range []string{"abs", "min", "max", "round"} {
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
	registerStaticIntrinsic("Integer", "parseInt", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return stdjavaCall(ctx, "ParseInt", args[0])
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
		return pkgCall(ctx, "fmt", "Sprint", args[0])
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
		return stdjavaCall(ctx, "ParseLong", args[0])
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
	for _, spec := range []struct{ wrapper, primitive, parser string }{
		{"Boolean", "boolean", "ParseBoolean"}, {"Byte", "byte", "ParseByte"},
		{"Short", "short", "ParseShort"}, {"Character", "char", ""},
		{"Integer", "int", "ParseInt"}, {"Long", "long", "ParseLong"},
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
	for _, receiverType := range []string{"Object", "Number"} {
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
			return stdjavaCall(ctx, "StringValueOfExecution", intrinsicExecutionExpr(ctx), stdjavaCall(ctx, "ReferenceRequireNonNull", recv))
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
