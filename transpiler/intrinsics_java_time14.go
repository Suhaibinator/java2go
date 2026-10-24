package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strings"
)

var javaTime14Names = map[string]bool{"Duration": true, "Instant": true, "LocalDate": true, "LocalTime": true, "LocalDateTime": true, "MonthDay": true, "Period": true, "Year": true, "YearMonth": true, "OffsetDateTime": true, "OffsetTime": true, "ZoneId": true, "ZoneOffset": true, "ZonedDateTime": true}

func javaTime14Owner(javaType string, ctx Ctx) (string, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if _, binder := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); binder {
		return "", false
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	name := stripJavaQualifier(owner)
	return owner, known && javaTime14Names[name] && owner == "java.time."+name
}
func javaTimeRuntimeTypeExpr(javaType string, ctx Ctx) (ast.Expr, bool) {
	owner, known := javaTime14Owner(javaType, ctx)
	if !known {
		return nil, false
	}
	name := stripJavaQualifier(owner)
	expression := stdjavaQualifiedExpr(name, ctx)
	if name == "ZoneId" {
		return expression, true
	}
	return &ast.StarExpr{X: expression}, true
}
func javaTimeReferenceAssignable(actual, expected string, ctx Ctx) bool {
	left, lok := javaTime14Owner(actual, ctx)
	right, rok := javaTime14Owner(expected, ctx)
	return lok && rok && left == "java.time.ZoneOffset" && right == "java.time.ZoneId"
}
func javaTimeExpectedArgumentTypes(owner, method string, count int, ctx Ctx) ([]string, bool) {
	canonical, known := javaTime14Owner(owner, ctx)
	name := stripJavaQualifier(canonical)
	if !known {
		return nil, false
	}
	if method == "parse" && count == 1 && (name == "Duration" || name == "Instant" || name == "LocalDate" || name == "LocalTime" || name == "LocalDateTime") {
		return []string{"java.lang.CharSequence"}, true
	}
	if name == "ZoneId" && method == "of" && count == 1 {
		return []string{"java.lang.String"}, true
	}
	switch name + "." + method {
	case "Duration.ofSeconds", "Instant.ofEpochSecond":
		if count == 2 {
			return []string{"long", "long"}, true
		}
	case "LocalDate.of", "Period.of":
		if count == 3 {
			return []string{"int", "int", "int"}, true
		}
	case "LocalTime.of":
		if count == 4 {
			return []string{"int", "int", "int", "int"}, true
		}
	case "MonthDay.of", "YearMonth.of":
		if count == 2 {
			return []string{"int", "int"}, true
		}
	case "Year.of", "ZoneOffset.ofTotalSeconds":
		if count == 1 {
			return []string{"int"}, true
		}
	case "LocalDateTime.of":
		if count == 2 {
			return []string{"java.time.LocalDate", "java.time.LocalTime"}, true
		}
	case "OffsetDateTime.of":
		if count == 2 {
			return []string{"java.time.LocalDateTime", "java.time.ZoneOffset"}, true
		}
	case "OffsetTime.of":
		if count == 2 {
			return []string{"java.time.LocalTime", "java.time.ZoneOffset"}, true
		}
	case "ZonedDateTime.ofInstant":
		if count == 3 {
			return []string{"java.time.LocalDateTime", "java.time.ZoneOffset", "java.time.ZoneId"}, true
		}
	}
	return nil, false
}
func init() {
	for name := range javaTime14Names {
		registerIntrinsicOwner("java.time."+name, true)
	}
	for _, entry := range []struct {
		owner, java, goName string
		arity               int
	}{
		{"Duration", "ofSeconds", "DurationOfSeconds", 2}, {"Instant", "ofEpochSecond", "InstantOfEpochSecond", 2},
		{"LocalDate", "of", "LocalDateOf", 3}, {"LocalTime", "of", "LocalTimeOf", 4}, {"LocalDateTime", "of", "LocalDateTimeOf", 2},
		{"MonthDay", "of", "MonthDayOf", 2}, {"Period", "of", "PeriodOf", 3}, {"Year", "of", "YearOf", 1}, {"YearMonth", "of", "YearMonthOf", 2},
		{"OffsetDateTime", "of", "OffsetDateTimeOf", 2}, {"OffsetTime", "of", "OffsetTimeOf", 2}, {"ZoneId", "of", "ZoneIdOf", 1},
		{"ZoneOffset", "ofTotalSeconds", "ZoneOffsetOfTotalSeconds", 1}, {"ZonedDateTime", "ofInstant", "ZonedDateTimeOfInstant", 3},
	} {
		entry := entry
		registerStaticIntrinsic(entry.owner, entry.java, func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != entry.arity {
				return nil
			}
			if entry.owner == "Duration" || entry.owner == "Instant" {
				converted := make([]ast.Expr, len(args))
				for i, arg := range args {
					converted[i] = &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{arg}}
				}
				args = converted
			}
			return stdjavaCall(ctx, entry.goName, args...)
		})
		registerStaticIntrinsicResultType(entry.owner, entry.java, "java.time."+entry.owner)
	}
	for _, owner := range []string{"Duration", "Instant", "LocalDate", "LocalTime", "LocalDateTime"} {
		owner := owner
		registerStaticIntrinsic(owner, "parse", func(_ ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, owner+"ParseExecution", intrinsicExecutionExpr(ctx), args[0])
		})
		registerStaticIntrinsicResultType(owner, "parse", "java.time."+owner)
	}
	for _, entry := range []struct{ owner, java, goName, result string }{
		{"Duration", "getSeconds", "GetSeconds", "long"}, {"Duration", "getNano", "GetNano", "int"}, {"Instant", "getEpochSecond", "GetEpochSecond", "long"}, {"Instant", "getNano", "GetNano", "int"},
		{"LocalDate", "getYear", "GetYear", "int"}, {"LocalDate", "getMonthValue", "GetMonthValue", "int"}, {"LocalDate", "getDayOfMonth", "GetDayOfMonth", "int"},
		{"LocalTime", "getHour", "GetHour", "int"}, {"LocalTime", "getMinute", "GetMinute", "int"}, {"LocalTime", "getSecond", "GetSecond", "int"}, {"LocalTime", "getNano", "GetNano", "int"},
		{"LocalDateTime", "toLocalDate", "ToLocalDate", "java.time.LocalDate"}, {"LocalDateTime", "toLocalTime", "ToLocalTime", "java.time.LocalTime"},
		{"MonthDay", "getMonthValue", "GetMonthValue", "int"}, {"MonthDay", "getDayOfMonth", "GetDayOfMonth", "int"},
		{"Period", "getYears", "GetYears", "int"}, {"Period", "getMonths", "GetMonths", "int"}, {"Period", "getDays", "GetDays", "int"},
		{"Year", "getValue", "GetValue", "int"}, {"YearMonth", "getYear", "GetYear", "int"}, {"YearMonth", "getMonthValue", "GetMonthValue", "int"},
		{"OffsetDateTime", "toLocalDateTime", "ToLocalDateTime", "java.time.LocalDateTime"}, {"OffsetDateTime", "getOffset", "GetOffset", "java.time.ZoneOffset"},
		{"OffsetTime", "toLocalTime", "ToLocalTime", "java.time.LocalTime"}, {"OffsetTime", "getOffset", "GetOffset", "java.time.ZoneOffset"},
		{"ZoneId", "getId", "GetID", "java.lang.String"}, {"ZoneOffset", "getId", "GetID", "java.lang.String"}, {"ZoneOffset", "getTotalSeconds", "GetTotalSeconds", "int"},
		{"ZonedDateTime", "toLocalDateTime", "ToLocalDateTime", "java.time.LocalDateTime"}, {"ZonedDateTime", "getOffset", "GetOffset", "java.time.ZoneOffset"}, {"ZonedDateTime", "getZone", "GetZone", "java.time.ZoneId"},
	} {
		registerInstanceIntrinsic(entry.owner, entry.java, ioMethod(entry.goName, 0))
		registerInstanceIntrinsicResultType(entry.owner, entry.java, entry.result)
	}
	for owner := range javaTime14Names {
		registerInstanceIntrinsic(owner, "toString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "JavaTimeStringExecution", intrinsicExecutionExpr(ctx), recv)
		})
		registerInstanceIntrinsicResultType(owner, "toString", "java.lang.String")
		registerInstanceIntrinsic(owner, "equals", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectEqualsExecution", intrinsicExecutionExpr(ctx), recv, args[0])
		})
		registerInstanceIntrinsicResultType(owner, "equals", "boolean")
		registerInstanceIntrinsic(owner, "hashCode", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), recv)
		})
		registerInstanceIntrinsicResultType(owner, "hashCode", "int")
	}
	for _, entry := range []struct{ name, owner string }{{"DateTimeException", "java.time.DateTimeException"}, {"DateTimeParseException", "java.time.format.DateTimeParseException"}, {"ZoneRulesException", "java.time.zone.ZoneRulesException"}} {
		registerIntrinsicOwner(entry.owner, true)
	}
	registerInstanceIntrinsic("DateTimeParseException", "getErrorIndex", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "DateTimeParseExceptionErrorIndex", recv)
	})
	registerInstanceIntrinsicResultType("DateTimeParseException", "getErrorIndex", "int")
	registerInstanceIntrinsic("DateTimeParseException", "getParsedString", func(recv ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "DateTimeParseExceptionParsedString", recv)
	})
	registerInstanceIntrinsicResultType("DateTimeParseException", "getParsedString", "java.lang.String")
}
