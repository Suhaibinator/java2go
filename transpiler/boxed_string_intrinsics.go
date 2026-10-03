package transpiler

import "go/ast"

// canonicalBoxedStringParse lowers a selected java.lang wrapper String overload
// through its existing UTF16 parser. Factories box the result; constructors use
// NewX so parsing cannot accidentally give a fresh object cached identity.
func canonicalBoxedStringParse(javaType string, args []ast.Expr, ctx Ctx) ast.Expr {
	if _, builtin := builtinJavaWrapperPrimitive(javaType, ctx); !builtin {
		return nil
	}
	name := stripJavaQualifier(javaType)
	parser := map[string]string{
		"Boolean": "JavaBooleanParseBoolean", "Byte": "JavaByteParseByte", "Short": "JavaShortParseShort",
		"Integer": "JavaIntegerParseInt", "Long": "JavaLongParseLong", "Float": "JavaFloatParseFloat", "Double": "JavaDoubleParseDouble",
	}[name]
	if parser == "" || len(args) < 1 || len(args) > 2 {
		return nil
	}
	if len(args) == 2 && name != "Byte" && name != "Short" && name != "Integer" && name != "Long" {
		return nil
	}
	return stdjavaCall(ctx, parser, args...)
}
