package transpiler

import (
	"go/ast"
	"go/token"
)

// This file registers the concrete java.lang intrinsics: String, StringBuilder /
// StringBuffer, Math, and the boxed numeric/character types. The table machinery
// lives in intrinsics.go.

func init() {
	registerStringIntrinsics()
	registerStringBuilderIntrinsics()
	registerMathIntrinsics()
	registerBoxedTypeIntrinsics()
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
		return &ast.BinaryExpr{X: recv, Op: token.EQL, Y: args[0]}
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

	// split(regex) -> strings.Split. NOTE: Java's split takes a regex; this maps
	// to a literal split. Patterns with regex metacharacters are not handled.
	registerInstanceIntrinsic("String", "split", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "strings", "Split", recv, args[0])
	})

	// chars() -> stdjava.StringChars(s) (returns []rune)
	registerInstanceIntrinsic("String", "chars", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 0) {
			return nil
		}
		return stdjavaCall(ctx, "StringChars", recv)
	})

	// --- static String methods ---

	// String.valueOf(x) -> fmt.Sprint(x). For the common boolean/number/char
	// cases this matches Java's textual form.
	registerStaticIntrinsic("String", "valueOf", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "fmt", "Sprint", args[0])
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
}

// --- java.lang.StringBuilder / StringBuffer ---------------------------------

func registerStringBuilderIntrinsics() {
	for _, typeName := range []string{"StringBuilder", "StringBuffer"} {
		// new StringBuilder() / new StringBuilder(String)
		registerConstructorIntrinsic(typeName, func(args []ast.Expr, ctx Ctx) ast.Expr {
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
		registerInstanceIntrinsic(typeName, "insert", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if !expectArgs(args, 2) {
				return nil
			}
			return methodCall(recv, "Insert", args[0], args[1])
		})
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

	// The floating-point functions map directly to the math package.
	registerStaticIntrinsic("Math", "pow", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 2) {
			return nil
		}
		return pkgCall(ctx, "math", "Pow", args[0], args[1])
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
		return stdjavaCall(ctx, "ParseDouble", args[0])
	})
	registerStaticIntrinsic("Double", "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if !expectArgs(args, 1) {
			return nil
		}
		return pkgCall(ctx, "fmt", "Sprint", args[0])
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
