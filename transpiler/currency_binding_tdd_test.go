package transpiler

import (
	"strings"
	"testing"
)

func TestCurrencyCanonicalSourceForeignBinderBindings(t *testing.T) {
	for _, source := range []string{
		`class Currency{static Currency getInstance(String s){return new Currency();}String getCurrencyCode(){return "source";}}class Probe{static Currency run(String s){return Currency.getInstance(s);}}`,
		`class Probe<Currency>{Currency value;Currency get(){return value;}}`,
		`class Probe{<Currency>Currency get(Currency value){return value;}}`,
		`import foreign.Currency;class Probe{Currency value;static Currency run(String s){return Currency.getInstance(s);}}`,
	} {
		generated := renderGoFileFromJava(t, source)
		for _, forbidden := range []string{"stdjava.JavaCurrency", "stdjava.CurrencyGetInstanceJavaString"} {
			if strings.Contains(generated, forbidden) {
				t.Fatalf("source/foreign/binder borrowed %s\n%s", forbidden, generated)
			}
		}
	}
	for _, owner := range []string{"foreign.Currency", "other.Currency"} {
		if _, mapped := currencyRuntimeTypeExpr(owner, nil, nil, Ctx{}); mapped {
			t.Fatalf("foreign currency mapped: %s", owner)
		}
	}
}

func TestCurrencyUnavailableAndAmbiguousInvocationsStrictlyRefused(t *testing.T) {
	for _, source := range []string{
		`import java.util.Currency;class Probe{static Currency run(){return Currency.getInstance(null);}}`,
		`import java.util.Currency;class Probe{static Currency run(){return Currency.getInstance((null));}}`,
		`import java.util.Currency;class Probe{static Currency run(java.util.Locale l){return Currency.getInstance(l);}}`,
		`import static java.util.Currency.getInstance;class Probe{static java.util.Currency run(){return getInstance(null);}}`,
		`import java.util.Currency;class Probe{static Currency run(){return new Currency();}}`,
		`import java.util.Currency;class Probe{static Currency run(){return Currency.getInstance(7);}}`,
		`import java.util.Currency;class Probe{static int run(Currency c){return c.getDefaultFractionDigits();}}`,
		`import java.util.Currency;class Probe{static int run(Currency c){return c.getNumericCode();}}`,
		`import java.util.Currency;class Probe{static String run(Currency c){return c.getSymbol();}}`,
		`import java.util.Currency;class Probe{static String run(Currency c){return c.getDisplayName();}}`,
		`import java.util.Currency;class Probe{static Object run(){return Currency.getAvailableCurrencies();}}`,
	} {
		func() {
			strictRoutingState(t)
			defer func() {
				if failure := recover(); failure == nil || len(Diagnostics()) == 0 {
					t.Fatalf("Currency unavailable/ambiguous call escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		}()
	}
}

func TestCurrencyNominalReferenceOwnership(t *testing.T) {
	cases := []struct {
		name, source, actual, expected string
		binders                        []string
		want                           bool
	}{
		{name: "canonical-object", actual: "java.util.Currency", expected: "java.lang.Object", want: true},
		{name: "canonical-serializable", actual: "java.util.Currency", expected: "java.io.Serializable", want: true},
		{name: "canonical-self", actual: "java.util.Currency", expected: "java.util.Currency", want: true},
		{name: "imported", source: "import java.util.Currency;import java.io.Serializable;class Probe{}", actual: "Currency", expected: "Serializable", want: true},
		{name: "wildcard", source: "import java.util.*;import java.io.*;class Probe{}", actual: "Currency", expected: "Serializable", want: true},
		{name: "source-actual", source: "class Currency{}class Probe{}", actual: "Currency", expected: "java.io.Serializable"},
		{name: "foreign-actual", source: "import foreign.Currency;class Probe{}", actual: "Currency", expected: "java.io.Serializable"},
		{name: "foreign-qualified-actual", actual: "foreign.Currency", expected: "java.io.Serializable"},
		{name: "source-target-object", source: "class Object{}class Probe{}", actual: "java.util.Currency", expected: "Object"},
		{name: "source-target-serializable", source: "interface Serializable{}class Probe{}", actual: "java.util.Currency", expected: "Serializable"},
		{name: "foreign-target-object", source: "import foreign.Object;class Probe{}", actual: "java.util.Currency", expected: "Object"},
		{name: "foreign-target-serializable", source: "import foreign.Serializable;class Probe{}", actual: "java.util.Currency", expected: "Serializable"},
		{name: "foreign-qualified-object", actual: "java.util.Currency", expected: "foreign.Object"},
		{name: "foreign-qualified-serializable", actual: "java.util.Currency", expected: "foreign.Serializable"},
		{name: "foreign-qualified-currency", actual: "java.util.Currency", expected: "foreign.Currency"},
		{name: "class-actual-binder", source: "class Probe<Currency>{}", actual: "Currency", expected: "java.io.Serializable"},
		{name: "class-object-binder", source: "class Probe<Object>{}", actual: "java.util.Currency", expected: "Object"},
		{name: "class-serializable-binder", source: "class Probe<Serializable>{}", actual: "java.util.Currency", expected: "Serializable"},
		{name: "candidate-actual-binder", actual: "Currency", expected: "java.io.Serializable", binders: []string{"Currency"}},
		{name: "candidate-object-binder", actual: "java.util.Currency", expected: "Object", binders: []string{"Object"}},
		{name: "candidate-serializable-binder", actual: "java.util.Currency", expected: "Serializable", binders: []string{"Serializable"}},
		{name: "arguments-actual", actual: "java.util.Currency<String>", expected: "java.io.Serializable"},
		{name: "arguments-object", actual: "java.util.Currency", expected: "java.lang.Object<String>"},
		{name: "arguments-serializable", actual: "java.util.Currency", expected: "java.io.Serializable<String>"},
		{name: "actual-array", actual: "java.util.Currency[]", expected: "java.io.Serializable"},
		{name: "target-array", actual: "java.util.Currency", expected: "java.io.Serializable[]"},
		{name: "non-currency", actual: "java.util.UUID", expected: "java.io.Serializable"},
		{name: "unsupported-interface", actual: "java.util.Currency", expected: "java.lang.Comparable"},
		{name: "source-shadow-qualified-owner", source: "class Currency{}class Object{}interface Serializable{}class Probe{}", actual: "java.util.Currency", expected: "java.io.Serializable", want: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			source := item.source
			if source == "" {
				source = "class Probe{}"
			}
			ctx := ordinaryDemandTestContext(t, map[string]string{"Probe.java": source}, "Probe.java")
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			if got := currencyReferenceAssignable(item.actual, item.expected, item.binders, ctx); got != item.want {
				t.Fatalf("%s -> %s: got %v, want %v", item.actual, item.expected, got, item.want)
			}
		})
	}
}

func TestCurrencyNominalMethodBinders(t *testing.T) {
	helper := setupParseHelper(t, `class Probe { <Currency> void actual(Currency value){} <Serializable> void serializable(Serializable value){} <Object> void object(Object value){} }`)
	ctx := helper.Ctx
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
	for _, item := range []struct{ method, actual, expected string }{{"actual", "Currency", "java.io.Serializable"}, {"serializable", "java.util.Currency", "Serializable"}, {"object", "java.util.Currency", "Object"}} {
		ctx.localScope = ctx.currentClass.FindMethodByName(item.method, nil)
		if ctx.localScope == nil {
			t.Fatalf("missing method %s", item.method)
		}
		if currencyReferenceAssignable(item.actual, item.expected, nil, ctx) {
			t.Fatalf("method binder borrowed Currency hierarchy: %s", item.method)
		}
	}
}
