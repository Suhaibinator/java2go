package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"strings"
	"testing"
)

func TestIntPredicateCanonicalPrimitiveSAM(t *testing.T) {
	for _, java := range []string{
		`import java.util.function.IntPredicate;public class Canonical {IntPredicate use(){return n->n<0;} boolean run(IntPredicate value){return value.test(3);}}`,
		`public class Canonical {java.util.function.IntPredicate use(){return n->n<0;} boolean run(java.util.function.IntPredicate value){return value.test(3);}}`,
	} {
		_, generated := updaterTDDParse(t, java)
		for _, required := range []string{"NewIntPredicateFuncAdapter", "CallIntPredicateExecution", "stdjava.IntPredicate", "int32", "bool"} {
			if !strings.Contains(generated, required) {
				t.Fatalf("missing %s\n%s", required, generated)
			}
		}
		if strings.Contains(generated, ".test(") {
			t.Fatalf("unlowered primitive SAM invocation\n%s", generated)
		}
	}
}
func TestIntPredicateNativeNamespaceDeclines(t *testing.T) {
	for _, java := range []string{
		`public class SourceShadow {interface IntPredicate {boolean test(int value);} static class Check implements IntPredicate{public boolean test(int value){return value>0;}} boolean run(IntPredicate value){return value.test(2);}}`,
		`public class BinderShadow<IntPredicate>{IntPredicate field;IntPredicate read(){return field;}}`,
		`public class MethodBinder {<IntPredicate> IntPredicate read(IntPredicate value){return value;}}`,
		`public class ValueShadow {static class Other {boolean test(int value){return value>0;}} boolean run(Other IntPredicate){return IntPredicate.test(2);}}`,
	} {
		_, generated := updaterTDDParse(t, java)
		if strings.Contains(generated, "stdjava.IntPredicate") || strings.Contains(generated, "CallIntPredicateExecution") || strings.Contains(generated, "NewIntPredicateSourceView") {
			t.Fatalf("source/type/value namespace borrowed JDK family\n%s", generated)
		}
	}
	for _, javaType := range []string{"foreign.IntPredicate", "java.util.function.IntPredicate<String>"} {
		if _, ok := stdjavaRuntimeTypeExpr(javaType, func() []string { _, args := parseJavaTypeString(javaType); return args }(), nil, Ctx{}); ok {
			t.Fatalf("invalid nominal/arity native type admitted: %s", javaType)
		}
	}
	if family := nativeFunctionalFamily("foreign.IntPredicate", Ctx{}); family != "" {
		t.Fatal("foreign family admitted")
	}
}
func TestIntPredicateSourceViewUsesExactExecutionEntry(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.IntPredicate;public class Collision implements IntPredicate{public boolean testJava2goExecution(int value){return false;}public boolean test(int value){return value>0;}}`)
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || updaterTDDCallName(call) != "NewIntPredicateSourceViewWithDefaults" {
			return true
		}
		found = true
		if len(call.Args) != 3 {
			t.Fatal("source view must capture info, callback and source defaults")
		}
		fn, ok := call.Args[1].(*ast.FuncLit)
		if !ok || fn.Type.Params.NumFields() != 2 {
			t.Fatal("source view callback lost Execution/int formal")
		}
		selected := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			invoke, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := invoke.Fun.(*ast.SelectorExpr)
			if ok && strings.HasPrefix(selector.Sel.Name, "TestJava2goExecution") && selector.Sel.Name != "TestJava2goExecution" {
				selected = true
			}
			return true
		})
		if !selected {
			t.Fatalf("native view borrowed public collision\n%s", generated)
		}
		return false
	})
	if !found {
		t.Fatalf("missing declared source native view\n%s", generated)
	}
}
func TestIntPredicateDefaultsKeepReferenceABI(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.IntPredicate;public class Defaults{IntPredicate run(IntPredicate value){return value.and(n->n>0).or(n->n<0).negate();}}`)
	calls := updaterTDDCalls(file)
	for _, required := range []string{"IntPredicateAndExecution", "IntPredicateOrExecution", "IntPredicateNegateExecution"} {
		if calls[required] != 1 {
			t.Fatalf("missing reference default %s\n%s", required, generated)
		}
	}
	if strings.Contains(generated, "IntUnaryOperator") {
		t.Fatalf("predicate borrowed int-return operator ABI\n%s", generated)
	}
}

func TestIntPredicateSourceSAMFlagGuards(t *testing.T) {
	for _, row := range []struct {
		name   string
		change func(*symbol.Definition)
	}{
		{"helper", func(m *symbol.Definition) { m.RequiresHelper = true }},
		{"helperName", func(m *symbol.Definition) { m.HelperName = "foreign" }},
		{"runtimeDefault", func(m *symbol.Definition) { m.RuntimeDefault = true }},
		{"variadic", func(m *symbol.Definition) { m.IsVariadic = true }},
		{"wrongResult", func(m *symbol.Definition) { m.OriginalType = "java.lang.Boolean" }},
		{"wrongParameter", func(m *symbol.Definition) { m.Parameters[0].OriginalType = "long" }},
		{"missingDeclaration", func(m *symbol.Definition) { m.DeclarationNode = nil }},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, `import java.util.function.IntPredicate;public class Guard implements IntPredicate{public boolean test(int value){return value>0;}}`)
			scope := helper.File.Symbols.FindClassScope("Guard")
			if scope == nil {
				t.Fatal("source missing")
			}
			ctx := classScopeCtx(scope, helper.Ctx)
			contract, _ := nativeFunctionalContract("IntPredicate", nil)
			method, _, _ := resolveSourceNativeSAM(scope, contract, ctx)
			if !intPredicateSourceMethodOwned(method, scope, contract, ctx) {
				t.Fatal("positive source proof missing")
			}
			row.change(method)
			if view := sourceNativeFunctionalViewExpr(scope, contract, ast.NewIdent("source"), ctx); view != nil {
				t.Fatal("invalid selected method donated native source entry")
			}
		})
	}
}

func TestIntPredicateSourceDefaultOverrideGuards(t *testing.T) {
	for _, row := range []struct {
		name   string
		change func(*symbol.Definition)
	}{
		{"helper", func(m *symbol.Definition) { m.RequiresHelper = true }},
		{"helperName", func(m *symbol.Definition) { m.HelperName = "foreign" }},
		{"runtimeDefault", func(m *symbol.Definition) { m.RuntimeDefault = true }},
		{"variadic", func(m *symbol.Definition) { m.IsVariadic = true }},
		{"missingDeclaration", func(m *symbol.Definition) { m.DeclarationNode = nil }},
		{"wrongReturn", func(m *symbol.Definition) { m.OriginalType = "boolean" }},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, `import java.util.function.IntPredicate;public class Guard implements IntPredicate{public boolean test(int n){return false;}public IntPredicate negate(){return n->false;}}`)
			scope := helper.File.Symbols.FindClassScope("Guard")
			ctx := classScopeCtx(scope, helper.Ctx)
			method, _, valid := intPredicateDefaultOverride(scope, "negate", ctx)
			if !valid || method == nil {
				t.Fatal("source override positive missing")
			}
			if defaults, valid := intPredicateSourceDefaults(scope, ast.NewIdent("source"), ctx); !valid || defaults == nil {
				t.Fatal("source override body positive missing")
			}
			row.change(method)
			if _, valid := intPredicateSourceDefaults(scope, ast.NewIdent("source"), ctx); valid {
				t.Fatal("invalid source override accepted")
			}
		})
	}
}

func TestIntPredicateInheritedDefaultStaticProjection(t *testing.T) {
	file, generated := updaterTDDParse(t, `import java.util.function.IntPredicate;interface Derived extends IntPredicate{} abstract class Base implements IntPredicate{} class Child extends Base implements Derived{public boolean test(int n){return n>0;}} public class Use{IntPredicate use(Derived d,Base b){return d.negate().and(b.negate());}}`)
	calls := updaterTDDCalls(file)
	if calls["IntPredicateNegateExecution"] != 2 || calls["IntPredicateAndExecution"] != 1 || calls["ObjectView"] < 2 {
		t.Fatalf("inherited static edge did not use checked concrete native view\n%s", generated)
	}
}
