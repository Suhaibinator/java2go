package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

var sqlDateRuntimeNames = map[string]string{"java.sql.Date": "SQLDate", "java.sql.Time": "SQLTime", "java.sql.Timestamp": "SQLTimestamp"}

func init() {
	for owner, runtimeName := range sqlDateRuntimeNames {
		registerIntrinsicOwner(owner, true)
		intrinsicOwnerKeys[owner] = owner
		constructor := func(_ []ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "New"+runtimeName, args...)
		}
		registerConstructorNodeIntrinsic(owner, dateConstructorNode(runtimeName, false))
		registerConstructorIntrinsic(owner, constructor)
		if stripJavaQualifier(owner) != "Date" {
			registerConstructorIntrinsic(stripJavaQualifier(owner), constructor)
		}
		registerStaticIntrinsic(owner, "valueOf", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, runtimeName+"ValueOf", args...)
		})
		registerStaticIntrinsicResultType(owner, "valueOf", owner)
		registerInstanceIntrinsic(owner, "toString", ioMethod("String", 0))
		registerInstanceIntrinsicResultType(owner, "toString", "String")
		for _, method := range []struct {
			java, goName, result string
			arity                int
		}{
			{"getTime", "GetTime", "long", 0}, {"setTime", "SetTime", "void", 1}, {"equals", "Equals", "boolean", 1}, {"hashCode", "HashCode", "int", 0}, {"compareTo", "CompareTo", "int", 1},
		} {
			registerInstanceIntrinsic(owner, method.java, ioMethod(method.goName, method.arity))
			registerInstanceIntrinsicResultType(owner, method.java, method.result)
		}
	}
	registerInstanceIntrinsic("java.sql.Timestamp", "getNanos", ioMethod("GetNanos", 0))
	registerInstanceIntrinsicResultType("java.sql.Timestamp", "getNanos", "int")
	registerInstanceIntrinsic("java.sql.Timestamp", "setNanos", ioMethod("SetNanos", 1))
	registerInstanceIntrinsicResultType("java.sql.Timestamp", "setNanos", "void")
}

func dateConstructorNode(runtimeName string, noArgument bool) constructorNodeGenerator {
	return func(_ []ast.Expr, args []ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if len(args) == 0 && noArgument {
			return stdjavaCall(ctx, "New"+runtimeName)
		}
		if len(args) != 1 {
			return nil
		}
		actual, known := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, source)
		if !known {
			return nil
		}
		converted, ok := convertJavaValue(args[0], actual, "long", ctx)
		if !ok {
			return nil
		}
		return stdjavaCall(ctx, "New"+runtimeName, converted)
	}
}

func datetimeExpectedArgumentTypes(owner, method string, count int) ([]string, bool) {
	sql := sqlDateRuntimeNames[owner] != ""
	if sql && method == "valueOf" && count == 1 {
		return []string{"String"}, true
	}
	if (sql || owner == "Date") && count == 1 {
		switch method {
		case "setTime":
			return []string{"long"}, true
		case "compareTo":
			return []string{"java.util.Date"}, true
		case "equals":
			return []string{"Object"}, true
		}
	}
	if owner == "java.sql.Timestamp" && method == "setNanos" && count == 1 {
		return []string{"int"}, true
	}
	if (owner == "Calendar" || owner == "GregorianCalendar") && method == "setTime" && count == 1 {
		return []string{"java.util.Date"}, true
	}
	return nil, false
}
