package transpiler

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func systemPropertyReferenceHome(t *testing.T) string {
	t.Helper()
	release, err := os.ReadFile("../stdjava/testdata/system_properties/reference-jdk21.release")
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "release"), release, 0600); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestSystemPropertyCanonicalCalls(t *testing.T) {
	for _, row := range []struct{ name, source, helper string }{
		{"get", `class Probe {String value(String key){return System.getProperty(key);}}`, "SystemGetProperty"},
		{"get-default", `class Probe {String value(String key,String fallback){return java.lang.System.getProperty(key,fallback);}}`, "SystemGetPropertyDefault"},
		{"set", `class Probe {String value(String key,String value){return System.setProperty(key,value);}}`, "SystemSetProperty"},
		{"clear", `class Probe {String value(String key){return System.clearProperty(key);}}`, "SystemClearProperty"},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Setenv("JAVA_HOME", systemPropertyReferenceHome(t))
			strictRoutingState(t)
			var failure any
			var output string
			func() { defer func() { failure = recover() }(); output = renderGoFileFromJava(t, row.source) }()
			if failure != nil || !strings.Contains(output, "stdjava."+row.helper+"(") || !strings.Contains(output, "stdjava.SystemPropertyProfile{") {
				t.Fatalf("canonical System property service/profile missing: %T %v\n%s", failure, failure, output)
			}
			if len(Diagnostics()) != 0 {
				t.Fatalf("canonical property diagnostics: %v", Diagnostics())
			}
		})
	}
}

func TestSystemPropertyMissingProfileRefusal(t *testing.T) {
	t.Setenv("JAVA_HOME", "")
	strictRoutingState(t)
	var failure any
	func() {
		defer func() { failure = recover() }()
		renderGoFileFromJava(t, `class Probe {String value(){return System.getProperty("java.version");}}`)
	}()
	refused, ok := failure.(strictModeError)
	if !ok || !strings.Contains(strings.ToLower(refused.diagnostic.Message), "profile") {
		t.Fatalf("missing selected JDK profile must refuse explicitly: %T %v", failure, failure)
	}
}

func systemPropertyAdapterCount(program ast.Node) (count int, valid bool) {
	valid = true
	if program == nil {
		return
	}
	ast.Inspect(program, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if !ok || owner.Name != "stdjava" {
			return true
		}
		arguments := 0
		switch selector.Sel.Name {
		case "SystemGetProperty", "SystemClearProperty":
			arguments = 2
		case "SystemGetPropertyDefault", "SystemSetProperty":
			arguments = 3
		default:
			return true
		}
		count++
		if len(call.Args) != arguments {
			valid = false
			return true
		}
		profile, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			valid = false
			return true
		}
		typ, ok := profile.Type.(*ast.SelectorExpr)
		if !ok || typ.Sel.Name != "SystemPropertyProfile" {
			valid = false
		}
		return true
	})
	return
}

func TestSystemPropertyStaticImportAndOwnerGuards(t *testing.T) {
	for _, row := range []struct {
		name  string
		files map[string]string
		admit bool
	}{
		{"static-explicit", map[string]string{"Probe.java": `import static java.lang.System.getProperty;class Probe{String value(String key){return getProperty(key);}}`}, true},
		{"static-wildcard", map[string]string{"Probe.java": `import static java.lang.System.*;class Probe{String value(String key){return getProperty(key,"fallback");}}`}, true},
		{"source-value", map[string]string{"Probe.java": `class Configuration{String getProperty(String key){return key;}}class Probe{String value(Configuration System){return System.getProperty("source");}}`}, false},
		{"generic-bound-value", map[string]string{"Probe.java": `class Configuration{String getProperty(String key){return key;}}class Probe{<T extends Configuration>String value(T System){return System.getProperty("source");}}`}, false},
		{"source-simple", map[string]string{"Probe.java": `class System{static String getProperty(String key){return key;}}class Probe{String value(String key){return System.getProperty(key);}}`}, false},
		{"source-prefix", map[string]string{"Probe.java": `class java{static class lang{static class System{static String getProperty(String key){return key;}}}}class Probe{String value(String key){return java.lang.System.getProperty(key);}}`}, false},
		{"source-lexical", map[string]string{"Probe.java": `import static java.lang.System.*;class Probe{static String getProperty(String key){return key;}String value(){return getProperty("source");}}`}, false},
		{"foreign-import", map[string]string{"foreign/System.java": `package foreign;public class System{public static String getProperty(String key){return key;}}`, "Probe.java": `import foreign.System;class Probe{String value(){return System.getProperty("source");}}`}, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			if row.admit {
				t.Setenv("JAVA_HOME", systemPropertyReferenceHome(t))
			} else {
				t.Setenv("JAVA_HOME", "")
			}
			ctx := ordinaryDemandTestContext(t, row.files, "Probe.java")
			strictRoutingState(t)
			node := ctx.currentClass.Class.DeclarationNode
			for node.Parent() != nil {
				node = node.Parent()
			}
			program, ok := ParseNode(node, ctx.currentFile.Source, ctx).(ast.Node)
			if !ok {
				t.Fatal("actual program AST absent")
			}
			count, valid := systemPropertyAdapterCount(program)
			if len(Diagnostics()) != 0 || !valid || (row.admit && count != 1) || (!row.admit && count != 0) {
				t.Fatalf("declaration ownership admit=%v count=%d valid=%v diagnostics=%v", row.admit, count, valid, Diagnostics())
			}
		})
	}
}
