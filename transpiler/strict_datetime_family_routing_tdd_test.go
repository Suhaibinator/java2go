package transpiler

import (
	"go/ast"
	"testing"
)

// Each control owns and restores its diagnostic state. These tests do not fix
// the separate production strict-state lifetime issue or disable strict mode.
func strictRoutingState(t *testing.T) {
	t.Helper()
	diagnostics.mu.Lock()
	previousStrict, previousItems := diagnostics.strict, diagnostics.items
	diagnostics.strict, diagnostics.items = true, nil
	diagnostics.mu.Unlock()
	t.Cleanup(func() {
		diagnostics.mu.Lock()
		diagnostics.strict, diagnostics.items = previousStrict, previousItems
		diagnostics.mu.Unlock()
	})
}

// A panic is observed only at this NEW regression's boundary. Foreign-owner
// diagnostics are converted into a failing record, never accepted as decline.
// This lets all frozen causal controls run without modifying original tests.
func strictRoutingInvoke(route func() (ast.Expr, bool)) (expression ast.Expr, mapped bool, observedPanic any) {
	defer func() { observedPanic = recover() }()
	expression, mapped = route()
	return
}

func strictForeignRoutingDeclines(t *testing.T, javaType string) {
	t.Helper()
	strictRoutingState(t)
	owner, known := canonicalIntrinsicOwner(javaType, Ctx{})
	if !known || owner != javaType || intrinsicOwnerSupported(owner) {
		t.Fatalf("fixture must select a registered foreign family: %s", javaType)
	}
	routes := []struct {
		name string
		call func() (ast.Expr, bool)
	}{
		{"date/time family", func() (ast.Expr, bool) { return dateTimeRuntimeTypeExpr(javaType, Ctx{}) }},
		{"complete runtime router", func() (ast.Expr, bool) {
			return stdjavaRuntimeTypeExpr(javaType, []string{"java.lang.Object"}, nil, Ctx{})
		}},
	}
	for _, route := range routes {
		expression, mapped, observed := strictRoutingInvoke(route.call)
		if observed != nil {
			if failure, ok := observed.(strictModeError); ok && failure.diagnostic.Kind == "JDK owner "+javaType {
				t.Fatalf("strict routing foreign diagnostic: %s via %s", javaType, route.name)
			}
			t.Fatalf("unexpected strict routing panic for %s via %s: %v", javaType, route.name, observed)
		}
		if mapped || expression != nil {
			t.Fatalf("foreign owner acquired native mapping: %s via %s", javaType, route.name)
		}
		if items := Diagnostics(); len(items) != 0 {
			t.Fatalf("foreign owner emitted a diagnostic instead of declining: %s: %v", javaType, items)
		}
	}
}

func TestStrictDateTimeForeignAtomicDeclinesTDD(t *testing.T) {
	for _, name := range []string{"foreign.AtomicIntegerFieldUpdater", "foreign.AtomicLongFieldUpdater", "foreign.AtomicReferenceFieldUpdater"} {
		t.Run(name, func(t *testing.T) { strictForeignRoutingDeclines(t, name) })
	}
}

func TestStrictDateTimeForeignFunctionalDeclinesTDD(t *testing.T) {
	for _, name := range []string{"foreign.Function", "foreign.UnaryOperator", "foreign.IntUnaryOperator"} {
		t.Run(name, func(t *testing.T) { strictForeignRoutingDeclines(t, name) })
	}
}

func TestStrictDateTimeForeignBigMathDeclinesTDD(t *testing.T) {
	for _, name := range []string{"foreign.BigInteger", "foreign.BigDecimal"} {
		t.Run(name, func(t *testing.T) { strictForeignRoutingDeclines(t, name) })
	}
}

func TestStrictDateTimeSupportedSQLMappingTDD(t *testing.T) {
	for _, entry := range []struct{ javaType, runtimeName string }{
		{"java.sql.Date", "SQLDate"}, {"java.sql.Time", "SQLTime"}, {"java.sql.Timestamp", "SQLTimestamp"},
	} {
		t.Run(entry.javaType, func(t *testing.T) {
			strictRoutingState(t)
			if !intrinsicOwnerSupported(entry.javaType) {
				t.Fatalf("actual frozen registry must support this SQL runtime: %s", entry.javaType)
			}
			expression, mapped, observed := strictRoutingInvoke(func() (ast.Expr, bool) {
				return dateTimeRuntimeTypeExpr(entry.javaType, Ctx{})
			})
			if observed != nil || !mapped {
				t.Fatalf("supported SQL owner failed exact mapping: %s: %v", entry.javaType, observed)
			}
			pointer, ok := expression.(*ast.StarExpr)
			if !ok {
				t.Fatalf("SQL owner lost reference ABI: %s", entry.javaType)
			}
			selector, ok := pointer.X.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != entry.runtimeName || len(Diagnostics()) != 0 {
				t.Fatalf("SQL owner borrowed another mapping or emitted diagnostics: %s", entry.javaType)
			}
		})
	}
}

// SQL Date/Time/Timestamp are supported in the actual frozen registry. This
// positive refusal control explicitly models their declared unsupported state,
// restores the exact flag, and does not claim that current SQL is unsupported.
func TestStrictDateTimeUnsupportedSQLStateRefusesTDD(t *testing.T) {
	for _, owner := range []string{"java.sql.Date", "java.sql.Time", "java.sql.Timestamp"} {
		t.Run(owner, func(t *testing.T) {
			strictRoutingState(t)
			owners := intrinsicOwners[stripJavaQualifier(owner)]
			previous, found := owners[owner]
			if !found {
				t.Fatalf("SQL declaration must already be registered: %s", owner)
			}
			owners[owner] = false
			t.Cleanup(func() { owners[owner] = previous })
			expression, mapped, observed := strictRoutingInvoke(func() (ast.Expr, bool) {
				return dateTimeRuntimeTypeExpr(owner, Ctx{})
			})
			failure, ok := observed.(strictModeError)
			items := Diagnostics()
			if !ok || failure.diagnostic.Kind != "JDK owner "+owner || expression != nil || mapped || len(items) != 1 || items[0].Kind != "JDK owner "+owner {
				t.Fatalf("unsupported-state SQL owner escaped strict refusal: %s; panic=%v; mapped=%t; diagnostics=%v", owner, observed, mapped, items)
			}
		})
	}
}
