package transpiler

import (
	"os"
	"strings"
	"testing"
)

func url180StaticOwnerSource(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/url180_contracts/static-owner/src/main/java/urlowner/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestURL180StaticSourceFieldPrefix(t *testing.T) {
	strictRoutingState(t)
	var observed any
	var output string
	func() {
		defer func() { observed = recover() }()
		output = renderGoFileFromJava(t, url180StaticOwnerSource(t))
	}()
	if observed != nil {
		t.Fatalf("source-owned static-field prefix must not acquire JDK URL refusal: %T %v", observed, observed)
	}
	if strings.Contains(output, "java.net.URL static service") {
		t.Fatalf("source-owned endpoint was rewritten as native service: %s", output)
	}
	if items := Diagnostics(); len(items) != 0 {
		t.Fatalf("source-owned endpoint diagnostics=%v", items)
	}
}

func TestURL180StaticSourceFieldPrefixJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml":                          "<project><groupId>urlowner</groupId><artifactId>source-static-prefix</artifactId><version>1</version></project>\n",
		"src/main/java/urlowner/Main.java": url180StaticOwnerSource(t),
	}, "urlowner.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "11:true\n"}, {name: "second", stdout: "11:true\n"}, {name: "third", stdout: "11:true\n"}})
}

func TestURL180StaticDeclarationOwnership(t *testing.T) {
	for _, entry := range []struct {
		name   string
		files  map[string]string
		refuse bool
	}{
		{"canonical_factory", map[string]string{"Probe.java": `class Probe{void value(){java.net.URL.setURLStreamHandlerFactory(null);}}`}, true},
		{"canonical_of", map[string]string{"Probe.java": `class Probe{void value(){java.net.URL.of(null,null);}}`}, true},
		{"explicit_import", map[string]string{"Probe.java": `import java.net.URL;class Probe{void value(){URL.of(null,null);}}`}, true},
		{"static_explicit", map[string]string{"Probe.java": `import static java.net.URL.of;class Probe{void value(){of(null,null);}}`}, true},
		{"static_wildcard", map[string]string{"Probe.java": `import static java.net.URL.*;class Probe{void value(){setURLStreamHandlerFactory(null);}}`}, true},
		{"static_import_source_homonym", map[string]string{"Probe.java": `import static java.net.URL.of;class URL{}class Probe{void value(){of(null,null);}}`}, true},
		{"source_simple", map[string]string{"Probe.java": `class URL{static Object of(Object a,Object b){return a;}}class Probe{void value(){URL.of(null,null);}}`}, false},
		{"source_type_prefix", map[string]string{"Probe.java": `class java{static class net{static class URL{static Object of(Object a,Object b){return a;}}}}class Probe{void value(){java.net.URL.of(null,null);}}`}, false},
		{"source_field_prefix", map[string]string{"Probe.java": `class Endpoint{void setURLStreamHandlerFactory(Object x){}Object of(Object a,Object b){return a;}}class Net{Endpoint URL=new Endpoint();}class java{static Net net=new Net();}class Probe{void value(){java.net.URL.setURLStreamHandlerFactory(null);java.net.URL.of(null,null);}}`}, false},
		{"source_value", map[string]string{"Probe.java": `class Endpoint{Object of(Object a,Object b){return a;}}class Probe{void value(Endpoint URL){URL.of(null,null);}}`}, false},
		{"class_binder", map[string]string{"Probe.java": `class Endpoint{Object of(Object a,Object b){return a;}}class Probe<URL extends Endpoint>{void value(URL endpoint){endpoint.of(null,null);}}`}, false},
		{"method_binder", map[string]string{"Probe.java": `class Endpoint{Object of(Object a,Object b){return a;}}class Probe{<URL extends Endpoint>void value(URL endpoint){endpoint.of(null,null);}}`}, false},
		{"foreign_import", map[string]string{"foreign/URL.java": `package foreign;public class URL{public static Object of(Object a,Object b){return a;}}`, "Probe.java": `import foreign.URL;class Probe{void value(){URL.of(null,null);}}`}, false},
		{"imported_prefix", map[string]string{"foreign/java.java": `package foreign;public class java{public static class net{public static class URL{public static Object of(Object a,Object b){return a;}}}}`, "Probe.java": `import foreign.java;class Probe{void value(){java.net.URL.of(null,null);}}`}, false},
		{"source_static_vs_jdk_wildcard", map[string]string{"foreign/Endpoint.java": `package foreign;public class Endpoint{public static Object of(Object a,Object b){return a;}}`, "Probe.java": `import static foreign.Endpoint.of;import static java.net.URL.*;class Probe{void value(){of(null,null);}}`}, false},
		{"source_lexical_vs_jdk_import", map[string]string{"Probe.java": `import static java.net.URL.*;class Probe{static Object of(Object a,Object b){return a;}void value(){of(null,null);}}`}, false},
	} {
		t.Run(entry.name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, entry.files, "Probe.java")
			strictRoutingState(t)
			var observed any
			func() {
				defer func() { observed = recover() }()
				node := ctx.currentClass.Class.DeclarationNode
				for node.Parent() != nil {
					node = node.Parent()
				}
				ParseNode(node, ctx.currentFile.Source, ctx)
			}()
			if entry.refuse {
				failure, ok := observed.(strictModeError)
				if !ok || failure.diagnostic.Kind != "intrinsic invocation" {
					t.Fatalf("canonical URL refusal=%T %v", observed, observed)
				}
			} else if observed != nil || len(Diagnostics()) != 0 {
				t.Fatalf("source/import/binder stolen: %T %v diagnostics=%v", observed, observed, Diagnostics())
			}
		})
	}
}
