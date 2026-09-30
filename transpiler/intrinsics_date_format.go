package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	registerConstructorNodeIntrinsic("SimpleDateFormat", func(_ []ast.Expr, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if len(args) != 1 && len(args) != 2 {
			return nil
		}
		expected := []string{"java.lang.String", "java.util.Locale"}
		// Applicability includes reference identity/widening. convertJavaValue
		// handles primitive/boxing conversions and cannot validate these reference
		// parameters. Parse with their targets to preserve null String storage and
		// the shared left-to-right argument evaluation rules.
		for i := range args {
			if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, i), expected[i], ctx, source) {
				return nil
			}
		}
		converted := parseArgumentListWithExpectedTypes(node.ChildByFieldName("arguments"), source, ctx, expected[:len(args)])
		for i, value := range converted {
			converted[i] = convertIntrinsicStringBoundArgument(value, invocationArgumentNode(node, i), expected[i], ctx, source)
		}
		return stdjavaCall(ctx, "NewSimpleDateFormatJavaString", converted...)
	})
	registerStaticIntrinsic("DateFormat", "getDateTimeInstance", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 2 && len(args) != 3 {
			return nil
		}
		return stdjavaCall(ctx, "DateFormatGetDateTimeInstanceJavaString", args...)
	})
	registerStaticIntrinsicResultType("DateFormat", "getDateTimeInstance", "java.text.DateFormat")
	registerStaticIntrinsicImportSignature("DateFormat", "getDateTimeInstance", "int", "int")
	registerStaticIntrinsicImportSignature("DateFormat", "getDateTimeInstance", "int", "int", "java.util.Locale")
	for _, name := range []string{"FULL", "LONG", "MEDIUM", "SHORT", "DEFAULT"} {
		registerStaticFieldIntrinsic("DateFormat", name, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("DateFormat"+name, ctx) })
		registerStaticFieldIntrinsicResultType("DateFormat", name, "int")
	}
	for _, owner := range []string{"DateFormat", "SimpleDateFormat"} {
		for _, entry := range []struct {
			java, goName, result string
			min, max             int
		}{
			{"format", "DateFormatFormatJavaString", "java.lang.String", 1, 1},
			{"parse", "DateFormatParseJavaString", "java.util.Date", 1, 2},
			{"getTimeZone", "DateFormatGetTimeZone", "java.util.TimeZone", 0, 0},
			{"setTimeZone", "DateFormatSetTimeZone", "void", 1, 1},
			{"setLenient", "DateFormatSetLenient", "void", 1, 1},
			{"isLenient", "DateFormatIsLenient", "boolean", 0, 0},
		} {
			registerInstanceIntrinsic(owner, entry.java, func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
				if len(args) < entry.min || len(args) > entry.max {
					return nil
				}
				return stdjavaCall(ctx, entry.goName, append([]ast.Expr{recv}, args...)...)
			})
			registerInstanceIntrinsicResultType(owner, entry.java, entry.result)
		}
	}
	for _, entry := range []struct {
		java, goName, result string
		arity                int
	}{
		{"toPattern", "ToPatternJavaString", "java.lang.String", 0}, {"applyPattern", "ApplyPatternJavaString", "void", 1},
		{"set2DigitYearStart", "Set2DigitYearStart", "void", 1}, {"get2DigitYearStart", "Get2DigitYearStart", "java.util.Date", 0},
	} {
		registerInstanceIntrinsic("SimpleDateFormat", entry.java, ioMethod(entry.goName, entry.arity))
		registerInstanceIntrinsicResultType("SimpleDateFormat", entry.java, entry.result)
	}
	for _, entry := range []struct {
		java, goName, result string
		arity                int
	}{
		{"getDefault", "LocaleGetDefault", "java.util.Locale", 0}, {"setDefault", "LocaleSetDefault", "void", 1},
	} {
		registerStaticIntrinsic("Locale", entry.java, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != entry.arity {
				return nil
			}
			return stdjavaCall(ctx, entry.goName, args...)
		})
		registerStaticIntrinsicResultType("Locale", entry.java, entry.result)
	}
	registerInstanceIntrinsic("Locale", "equals", ioMethod("Equals", 1))
	registerInstanceIntrinsicResultType("Locale", "equals", "boolean")
}
func dateFormatExpectedArgumentTypes(owner, method string, count int) ([]string, bool) {
	if owner == "DateFormat" || owner == "SimpleDateFormat" {
		switch method {
		case "getDateTimeInstance":
			if count == 2 {
				return []string{"int", "int"}, true
			}
			if count == 3 {
				return []string{"int", "int", "java.util.Locale"}, true
			}
		case "format", "set2DigitYearStart":
			if count == 1 {
				return []string{"java.util.Date"}, true
			}
		case "parse":
			if count == 1 {
				return []string{"java.lang.String"}, true
			}
			if count == 2 {
				return []string{"java.lang.String", "java.text.ParsePosition"}, true
			}
		case "applyPattern":
			if count == 1 {
				return []string{"java.lang.String"}, true
			}
		case "setTimeZone":
			if count == 1 {
				return []string{"java.util.TimeZone"}, true
			}
		case "setLenient":
			if count == 1 {
				return []string{"boolean"}, true
			}
		}
	}
	if owner == "Locale" && count == 1 {
		if method == "setDefault" {
			return []string{"java.util.Locale"}, true
		}
		if method == "equals" {
			return []string{"java.lang.Object"}, true
		}
	}
	return nil, false
}
