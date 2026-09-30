package transpiler

import (
	"go/ast"
)

var dateTimePackages = map[string]string{"Date": "java.util", "TimeZone": "java.util", "Calendar": "java.util", "GregorianCalendar": "java.util", "ParsePosition": "java.text", "DateFormat": "java.text", "SimpleDateFormat": "java.text", "Locale": "java.util"}

func dateTimeRuntimeTypeID(javaType string, ctx Ctx) (string, bool) {
	owner, registered := canonicalIntrinsicOwner(javaType, ctx)
	if !registered || !intrinsicOwnerSupported(owner) {
		return "", false
	}
	name := stripJavaQualifier(owner)
	pkg, ok := dateTimePackages[name]
	return owner, ok && owner == pkg+"."+name
}
func dateTimeRuntimeTypeExpr(base string, ctx Ctx) (ast.Expr, bool) {
	if owner, registered := canonicalIntrinsicOwner(base, ctx); registered && !intrinsicOwnerSupported(owner) {
		reportUnsupported("JDK owner "+owner, nil, nil, ctx)
		return nil, false
	}
	if owner, registered := canonicalIntrinsicOwner(base, ctx); registered {
		if name := sqlDateRuntimeNames[owner]; name != "" {
			return &ast.StarExpr{X: stdjavaQualifiedExpr(name, ctx)}, true
		}
		if owner == "java.util.Date" {
			return stdjavaQualifiedExpr("DateValue", ctx), true
		}
	}
	if owner, registered := canonicalIntrinsicOwner(base, ctx); registered && owner == "java.util.Locale.Category" {
		return &ast.StarExpr{X: stdjavaQualifiedExpr("LocaleCategory", ctx)}, true
	}
	if _, ok := dateTimeRuntimeTypeID(base, ctx); !ok {
		return nil, false
	}
	name := stripJavaQualifier(base)
	value := stdjavaQualifiedExpr(name, ctx)
	if name == "DateFormat" {
		return stdjavaQualifiedExpr("JavaDateFormat", ctx), true
	}
	if name == "Calendar" || name == "DateFormat" {
		return value, true
	}
	return &ast.StarExpr{X: value}, true
}
func dateTimeReferenceAssignable(actual, expected string, ctx Ctx) bool {
	if left, lok := canonicalIntrinsicOwner(actual, ctx); lok && sqlDateRuntimeNames[left] != "" {
		if right, rok := canonicalIntrinsicOwner(expected, ctx); rok && right == "java.util.Date" {
			return true
		}
	}
	left, lok := dateTimeRuntimeTypeID(actual, ctx)
	right, rok := dateTimeRuntimeTypeID(expected, ctx)
	return lok && rok && ((left == "java.util.GregorianCalendar" && right == "java.util.Calendar") || (left == "java.text.SimpleDateFormat" && right == "java.text.DateFormat"))
}
func init() {
	for name, pkg := range dateTimePackages {
		registerIntrinsicOwner(pkg+"."+name, true)
		intrinsicDefaultOwners[name] = pkg + "." + name
	}
	for _, owner := range []string{"java.sql.Date", "java.sql.Time", "java.sql.Timestamp"} {
		registerIntrinsicOwner(owner, false)
	}
	registerConstructorNodeIntrinsic("Date", dateConstructorNode("Date", true))
	for _, name := range []string{"Date", "ParsePosition"} {
		registerConstructorIntrinsic(name, func(types, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) > 1 {
				return nil
			}
			return stdjavaCall(ctx, "New"+name, args...)
		})
	}
	for _, entry := range []struct {
		owner, java, goName, result string
		arity                       int
	}{
		{"Date", "getTime", "GetTime", "long", 0}, {"Date", "setTime", "SetTime", "void", 1},
		{"Date", "equals", "Equals", "boolean", 1}, {"Date", "hashCode", "HashCode", "int", 0}, {"Date", "compareTo", "CompareTo", "int", 1},
		{"ParsePosition", "getIndex", "GetIndex", "int", 0}, {"ParsePosition", "setIndex", "SetIndex", "void", 1},
		{"ParsePosition", "getErrorIndex", "GetErrorIndex", "int", 0}, {"ParsePosition", "setErrorIndex", "SetErrorIndex", "void", 1},
	} {
		registerInstanceIntrinsic(entry.owner, entry.java, ioMethod(entry.goName, entry.arity))
		registerInstanceIntrinsicResultType(entry.owner, entry.java, entry.result)
	}
	registerInstanceIntrinsic("ParseException", "getErrorOffset", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "ParseExceptionErrorOffset", recv)
	})
	registerInstanceIntrinsicResultType("ParseException", "getErrorOffset", "int")
}

func init() {
	for _, entry := range []struct {
		java, goName, result string
		arity                int
	}{
		{"getTimeZone", "TimeZoneGetTimeZoneJavaString", "TimeZone", 1}, {"getDefault", "TimeZoneGetDefault", "TimeZone", 0}, {"setDefault", "TimeZoneSetDefault", "void", 1},
	} {
		registerStaticIntrinsic("TimeZone", entry.java, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != entry.arity {
				return nil
			}
			return stdjavaCall(ctx, entry.goName, args...)
		})
		registerStaticIntrinsicResultType("TimeZone", entry.java, entry.result)
	}
	for _, entry := range []struct {
		java, goName, result string
		arity                int
	}{
		{"setRawOffset", "SetRawOffset", "void", 1}, {"hasSameRules", "HasSameRules", "boolean", 1}, {"getID", "GetIDJavaString", "String", 0}, {"setID", "SetIDJavaString", "void", 1}, {"getRawOffset", "GetRawOffset", "int", 0}, {"getOffset", "GetOffset", "int", 1}, {"clone", "Clone", "Object", 0},
	} {
		registerInstanceIntrinsic("TimeZone", entry.java, ioMethod(entry.goName, entry.arity))
		registerInstanceIntrinsicResultType("TimeZone", entry.java, entry.result)
	}
}

func init() {
	for name := range builtinExceptionTypes {
		registerInstanceIntrinsic(name, "initCause", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "ThrowableInitCauseExecution", intrinsicExecutionExpr(ctx), recv, args[0])
		})
		registerInstanceIntrinsicResultType(name, "initCause", "Throwable")
	}
}

func init() {
	registerConstructorIntrinsic("GregorianCalendar", func(types, args []ast.Expr, ctx Ctx) ast.Expr {
		return stdjavaCall(ctx, "NewGregorianCalendar", args...)
	})
	for _, owner := range []string{"Calendar", "GregorianCalendar"} {
		for _, name := range []string{"ERA", "YEAR", "MONTH", "WEEK_OF_YEAR", "WEEK_OF_MONTH", "DAY_OF_MONTH", "DAY_OF_YEAR", "DAY_OF_WEEK", "DAY_OF_WEEK_IN_MONTH", "AM_PM", "HOUR", "HOUR_OF_DAY", "MINUTE", "SECOND", "MILLISECOND", "ZONE_OFFSET", "DST_OFFSET"} {
			registerStaticFieldIntrinsic(owner, name, func(ctx Ctx) ast.Expr { return stdjavaQualifiedExpr("Calendar"+name, ctx) })
			registerStaticFieldIntrinsicResultType(owner, name, "int")
		}
		for _, entry := range []struct {
			java, goName, result string
			arity                int
		}{
			{"get", "Get", "int", 1}, {"getTime", "GetTime", "java.util.Date", 0}, {"setTime", "SetTime", "void", 1},
			{"getTimeInMillis", "GetTimeInMillis", "long", 0}, {"setTimeInMillis", "SetTimeInMillis", "void", 1},
			{"setLenient", "SetLenient", "void", 1}, {"isLenient", "IsLenient", "boolean", 0},
			{"getTimeZone", "GetTimeZone", "TimeZone", 0}, {"setTimeZone", "SetTimeZone", "void", 1},
		} {
			registerInstanceIntrinsic(owner, entry.java, ioMethod(entry.goName, entry.arity))
			registerInstanceIntrinsicResultType(owner, entry.java, entry.result)
		}
		registerInstanceIntrinsic(owner, "set", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 2 && len(args) != 3 && len(args) != 5 && len(args) != 6 {
				return nil
			}
			return selectorCall(recv, "Set", args)
		})
		registerInstanceIntrinsicResultType(owner, "set", "void")
		registerInstanceIntrinsic(owner, "clear", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) > 1 {
				return nil
			}
			return selectorCall(recv, "Clear", args)
		})
		registerInstanceIntrinsicResultType(owner, "clear", "void")
	}
}
