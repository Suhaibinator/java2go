package transpiler

import (
	"go/ast"
	"strings"
	"testing"
)

const url180OriginalConstructorSource = `class Probe {Object value() throws Exception{return new java.net.URL("https://example.test/a");}}`

func TestURL180CanonicalStringConstructor(t *testing.T) {
	strictRoutingState(t)
	var observed any
	var output string
	func() {
		defer func() { observed = recover() }()
		output = renderGoFileFromJava(t, url180OriginalConstructorSource)
	}()
	if observed != nil {
		t.Fatalf("valid canonical URL(String) must lower, actual %T: %v", observed, observed)
	}
	if !strings.Contains(output, "NewJavaURLStringExecution") {
		t.Fatalf("canonical service absent: %s", output)
	}
	if items := Diagnostics(); len(items) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", items)
	}
}

func TestURL180ConstructorOwnerShields(t *testing.T) {
	for name, source := range map[string]string{
		"source_simple": `class URL {URL(String s){}}class Probe {URL value(){return new URL("x");}}`,
		"source_prefix": `class java {static class net {static class URL {URL(String s){}}}}class Probe {Object value(){return new java.net.URL("x");}}`,
		"source_local":  `class Probe {Object value(){class URL {URL(String s){}}return new URL("x");}}`,
		"source_nested": `class Probe {static class URL {URL(String s){}}Object value(){return new URL("x");}}`,
		"class_binder":  `class Probe<URL> {String value(URL value){return value.toString();}}`,
		"method_binder": `class Probe {<URL>String value(URL value){return value.toString();}}`,
	} {
		t.Run(name, func(t *testing.T) {
			strictRoutingState(t)
			output := renderGoFileFromJava(t, source)
			if strings.Contains(output, "NewJavaURLStringExecution") || strings.Contains(output, "*stdjava.JavaURL") {
				t.Fatalf("source/binder stolen: %s", output)
			}
		})
	}
}

func TestURL180UnsupportedTypedServices(t *testing.T) {
	t.Setenv("JAVA_HOME", systemPropertyReferenceHome(t))
	for name, source := range map[string]string{
		"context_constructor":    `class Probe {Object value(java.net.URL base) throws Exception{return new java.net.URL(base,"next");}}`,
		"components_constructor": `class Probe {Object value() throws Exception{return new java.net.URL("http","host",80,"/a");}}`,
		"equals":                 `class Probe {boolean value(java.net.URL a,Object b){return a.equals(b);}}`,
		"hashCode":               `class Probe {int value(java.net.URL a){return a.hashCode();}}`,
		"sameFile":               `class Probe {boolean value(java.net.URL a,java.net.URL b){return a.sameFile(b);}}`,
		"getHost":                `class Probe {String value(java.net.URL a){return a.getHost();}}`,
		"toURI":                  `class Probe {Object value(java.net.URL a) throws Exception{return a.toURI();}}`,
		"connection":             `class Probe {Object value(java.net.URL a) throws Exception{return a.openConnection();}}`,
		"stream":                 `class Probe {Object value(java.net.URL a) throws Exception{return a.openStream();}}`,
		"content":                `class Probe {Object value(java.net.URL a) throws Exception{return a.getContent();}}`,
		"factory":                `class Probe {void value(){java.net.URL.setURLStreamHandlerFactory(null);}}`,
		"handler_property":       `class Probe {void value(){System.setProperty("java.protocol.handler.pkgs","mine");}}`,
		"validation_property":    `class Probe {void value(){System.setProperty("jdk.net.url.delayParsing","true");}}`,
	} {
		t.Run(name, func(t *testing.T) {
			strictRoutingState(t)
			var observed any
			var output string
			func() { defer func() { observed = recover() }(); output = renderGoFileFromJava(t, source) }()
			// These exact two former blanket-refusal inputs now exercise the general
			// String property service. Real JDK state and URL boundary tests accompany
			// this deliberate admission; all other unsupported URL rows remain refused.
			if name == "handler_property" || name == "validation_property" {
				if observed != nil || len(Diagnostics()) != 0 || strings.Count(output, "stdjava.SystemSetProperty(") != 1 || !strings.Contains(output, "stdjava.SystemPropertyProfile{") {
					t.Fatalf("canonical property admission/profile missing: %T %v\n%s", observed, observed, output)
				}
				return
			}
			if _, ok := observed.(strictModeError); !ok {
				t.Fatalf("unsupported URL/configuration must be refused, actual %T %v", observed, observed)
			}
		})
	}
}

func TestURL180ConstructorFormatterJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml": "<project><groupId>url180</groupId><artifactId>canonical-url</artifactId><version>1</version></project>\n",
		"src/main/java/url180/Main.java": `package url180;import java.net.URL;import java.net.MalformedURLException;import java.io.IOException;public class Main {
 static int count;static String once(){count++;return "HTTP://Example.invalid:80/a/../b?q=x%20y#piece";}
 public static void main(String[] args)throws Exception {
 URL u=new URL(once());URL alias=u;System.out.println(u.toExternalForm());System.out.println(u.toString());System.out.println((u==alias)+":"+count+":"+(u instanceof java.io.Serializable)+":"+u.getClass().getName());
 System.out.println(new URL("  url:HTTPS://Example.invalid/path?q=1  ").toExternalForm());System.out.println(new URL("file:/var/tmp/catalog%20one.json").toExternalForm());System.out.println(new URL("jar:file:/var/tmp/catalog.jar!/entries/a.json").toExternalForm());
 try{new URL("relative/catalog");}catch(MalformedURLException e){System.out.println(e.getClass().getName()+":"+e.getMessage()+":"+(e.getCause()==null)+":"+(e instanceof IOException));}
 MalformedURLException m=new MalformedURLException("same");System.out.println(m.getMessage()+":"+(m.getCause()==null)+":"+m.getClass().getSuperclass().getName());
 }}`,
	}, "url180.Main", []campaignCompilerProjectObservation{{name: "first", stdout: url180PositiveOutput}, {name: "second", stdout: url180PositiveOutput}, {name: "third", stdout: url180PositiveOutput}})
}

const url180PositiveOutput = "http://Example.invalid:80/a/../b?q=x%20y#piece\nhttp://Example.invalid:80/a/../b?q=x%20y#piece\ntrue:1:true:java.net.URL\nhttps://Example.invalid/path?q=1\nfile:/var/tmp/catalog%20one.json\njar:file:/var/tmp/catalog.jar!/entries/a.json\njava.net.MalformedURLException:no protocol: relative/catalog:true:true\nsame:true:java.io.IOException\n"

func TestURL180ConfigurationDeclarationOwnership(t *testing.T) {
	t.Setenv("JAVA_HOME", systemPropertyReferenceHome(t))
	for _, entry := range []struct {
		name   string
		files  map[string]string
		refuse bool
	}{
		{"qualified", map[string]string{"Probe.java": `class Probe {void value(String key,String value){java.lang.System.setProperty(key,value);}}`}, true},
		{"implicit", map[string]string{"Probe.java": `class Probe {void value(){System.clearProperty("anything");}}`}, true},
		{"all_properties", map[string]string{"Probe.java": `class Probe {void value(){System.setProperties(null);}}`}, true},
		{"explicit_import", map[string]string{"Probe.java": `import java.lang.System;class Probe {void value(){System.setProperty("arbitrary","value");}}`}, true},
		{"static_explicit", map[string]string{"Probe.java": `import static java.lang.System.setProperty;class Probe {void value(){setProperty("arbitrary","value");}}`}, true},
		{"static_wildcard", map[string]string{"Probe.java": `import static java.lang.System.*;class Probe {void value(){clearProperty("arbitrary");}}`}, true},
		{"static_import_with_source_homonym", map[string]string{"Probe.java": `import static java.lang.System.setProperty;class System{}class Probe {void value(){setProperty("arbitrary","value");}}`}, true},
		{"source_simple", map[string]string{"Probe.java": `class System {static String setProperty(String k,String v){return v;}}class Probe {void value(){System.setProperty("x","y");}}`}, false},
		{"source_prefix", map[string]string{"Probe.java": `class java {static class lang {static class System {static String setProperty(String k,String v){return v;}}}}class Probe {void value(){java.lang.System.setProperty("x","y");}}`}, false},
		{"source_local", map[string]string{"Probe.java": `class Probe {void value(){class System{static String setProperty(String k,String v){return v;}}System.setProperty("x","y");}}`}, false},
		{"source_value", map[string]string{"Probe.java": `class Configuration {String setProperty(String k,String v){return v;}}class Probe {void value(Configuration System){System.setProperty("x","y");}}`}, false},
		{"class_binder", map[string]string{"Probe.java": `class Configuration {String setProperty(String k,String v){return v;}}class Probe<System extends Configuration> {void value(System configuration){configuration.setProperty("x","y");}}`}, false},
		{"method_binder", map[string]string{"Probe.java": `class Configuration {String setProperty(String k,String v){return v;}}class Probe {<System extends Configuration>void value(System configuration){configuration.setProperty("x","y");}}`}, false},
		{"foreign_import", map[string]string{"foreign/System.java": `package foreign;public class System{public static String setProperty(String k,String v){return v;}}`, "Probe.java": `import foreign.System;class Probe {void value(){System.setProperty("x","y");}}`}, false},
		{"foreign_static_explicit_vs_jdk_wildcard", map[string]string{"foreign/Configuration.java": `package foreign;public class Configuration{public static String setProperty(String k,String v){return v;}}`, "Probe.java": `import static foreign.Configuration.setProperty;import static java.lang.System.*;class Probe {void value(){setProperty("x","y");}}`}, false},
		{"source_lexical_vs_jdk_import", map[string]string{"Probe.java": `import static java.lang.System.*;class Probe {static String setProperty(String k,String v){return v;}void value(){setProperty("x","y");}}`}, false},
	} {
		t.Run(entry.name, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, entry.files, "Probe.java")
			strictRoutingState(t)
			var observed any
			var program ast.Node
			func() {
				defer func() { observed = recover() }()
				node := ctx.currentClass.Class.DeclarationNode
				for node.Parent() != nil {
					node = node.Parent()
				}
				program, _ = ParseNode(node, ctx.currentFile.Source, ctx).(ast.Node)
			}()
			propertyAdmission := entry.refuse && entry.name != "all_properties"
			if propertyAdmission {
				count, valid := systemPropertyAdapterCount(program)
				if observed != nil || len(Diagnostics()) != 0 || count != 1 || !valid {
					t.Fatalf("owned property call must admit one typed profile adapter: %T %v count=%d valid=%v", observed, observed, count, valid)
				}
			} else if entry.refuse {
				failure, ok := observed.(strictModeError)
				if !ok || failure.diagnostic.Kind != "intrinsic invocation" {
					t.Fatalf("canonical configuration refusal=%T %v", observed, observed)
				}
			} else if observed != nil || len(Diagnostics()) != 0 {
				t.Fatalf("source/import/binder was stolen: %T %v diagnostics=%v", observed, observed, Diagnostics())
			} else if count, _ := systemPropertyAdapterCount(program); count != 0 {
				t.Fatalf("source/import/binder borrowed System property runtime: %d calls", count)
			}
		})
	}
}

func TestURL180FreshReceiverWrittenTypeProof(t *testing.T) {
	for _, source := range []string{
		`import java.net.URL;class Probe {String value() throws Exception{return new URL("http://host/a").toExternalForm();}}`,
		`class Probe {String value() throws Exception{return (new java.net.URL("http://host/a")).toExternalForm();}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			output := renderGoFileFromJava(t, source)
			if !strings.Contains(output, "ToExternalFormJava2goExecution") {
				t.Fatalf("fresh canonical receiver not lowered: %s", output)
			}
		})
	}
	for _, source := range []string{
		`import java.net.URL;class Probe {String value() throws Exception{return new URL("http://host/a").getHost();}}`,
		`class Probe {boolean value() throws Exception{return (new java.net.URL("http://host/a")).equals(null);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			var observed any
			func() { defer func() { observed = recover() }(); renderGoFileFromJava(t, source) }()
			if _, ok := observed.(strictModeError); !ok {
				t.Fatalf("fresh canonical unadmitted service accepted: %T %v", observed, observed)
			}
		})
	}
	for _, source := range []string{
		`class URL {URL(String s){}String toExternalForm(){return "source";}}class Probe {String value(){return new URL("x").toExternalForm();}}`,
		`class java {static class net {static class URL {URL(String s){}String toExternalForm(){return "source";}}}}class Probe {String value(){return (new java.net.URL("x")).toExternalForm();}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			output := renderGoFileFromJava(t, source)
			if strings.Contains(output, "ToExternalFormJava2goExecution") {
				t.Fatalf("source fresh receiver stolen: %s", output)
			}
		})
	}
}

func TestURL180ConfigurationSourceAndMixedImportsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml": "<project><groupId>ownership</groupId><artifactId>property-guards</artifactId><version>1</version></project>\n",
		"src/main/java/foreign/Configuration.java": `package foreign;public class Configuration {public static int calls;public static String setProperty(String key,String value){calls++;return "foreign:"+key+":"+value;}}`,
		"src/main/java/ownership/Main.java": `package ownership;import static foreign.Configuration.setProperty;import static java.lang.System.*;
 class System {static int calls;static String setProperty(String key,String value){calls++;return "source:"+key+":"+value;}}
 public class Main {public static void main(String[] args){java.lang.System.out.println(setProperty("key","value"));java.lang.System.out.println(System.setProperty("key","value"));java.lang.System.out.println(foreign.Configuration.calls+":"+System.calls);}}`,
	}, "ownership.Main", []campaignCompilerProjectObservation{{name: "first", stdout: "foreign:key:value\nsource:key:value\n1:1\n"}, {name: "second", stdout: "foreign:key:value\nsource:key:value\n1:1\n"}, {name: "third", stdout: "foreign:key:value\nsource:key:value\n1:1\n"}})
}

func TestURL180ArgumentEvaluationTypedNullJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml": "<project><groupId>evaluation</groupId><artifactId>url-order</artifactId><version>1</version></project>\n",
		"src/main/java/evaluation/Main.java": `package evaluation;import java.net.URL;import java.net.MalformedURLException;
 public class Main {static int calls;static RuntimeException marker=new RuntimeException("marker");static String abrupt(){calls++;throw marker;}
 public static void main(String[] args)throws Exception {try{new URL(abrupt());}catch(RuntimeException e){System.out.println((e==marker)+":"+calls);}
 String text=null;try{new URL(text);}catch(MalformedURLException e){System.out.println(e.getClass().getName()+":"+e.getCause().getClass().getName()+":"+e.getMessage());}
 MalformedURLException m=new MalformedURLException((String)null);System.out.println((m.getMessage()==null)+":"+(m.getCause()==null));}}
 `,
	}, "evaluation.Main", []campaignCompilerProjectObservation{{name: "first", stdout: url180EvaluationOutput}, {name: "second", stdout: url180EvaluationOutput}, {name: "third", stdout: url180EvaluationOutput}})
}

const url180EvaluationOutput = "true:1\njava.net.MalformedURLException:java.lang.NullPointerException:Cannot invoke \"String.length()\" because \"spec\" is null\ntrue:true\n"

// These exact valid Java bodies previously belonged to the unimplemented URL
// table. Admission now requires real native lowering and no diagnostics; the
// original frozen Java text remains unchanged in uri129's corpus.
func TestURL701PriorCanonicalStringRefusalsNowLower(t *testing.T) {
	for _, source := range []string{
		`class Probe {Object value() throws Exception{return new java.net.URL("https://example.test/a");}}`,
		`import java.net.URL;class Probe {Object value() throws Exception{return new URL("https://example.test/a");}}`,
		`import java.net.*;class Probe {Object value() throws Exception{return new URL("https://example.test/a");}}`,
		`class Probe {Object value(int java) throws Exception{return new java.net.URL("https://example.test/a");}}`,
	} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			output := renderGoFileFromJava(t, source)
			if !strings.Contains(output, "NewJavaURLStringExecution") || len(Diagnostics()) != 0 {
				t.Fatal("canonical URL(String) did not acquire native service", output, Diagnostics())
			}
		})
	}
}

func TestURL701PriorCanonicalStringResultsJDK21(t *testing.T) {
	for _, entry := range []struct{ name, source, invocation string }{
		{"qualified", `class Probe {Object value() throws Exception{return new java.net.URL("https://example.test/a");}}`, "new Probe().value()"},
		{"imported", `import java.net.URL;class Probe {Object value() throws Exception{return new URL("https://example.test/a");}}`, "new Probe().value()"},
		{"wildcard", `import java.net.*;class Probe {Object value() throws Exception{return new URL("https://example.test/a");}}`, "new Probe().value()"},
		{"value_prefix", `class Probe {Object value(int java) throws Exception{return new java.net.URL("https://example.test/a");}}`, "new Probe().value(0)"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			main := "public class Main {public static void main(String[] args)throws Exception {Object value=" + entry.invocation + ";java.net.URL url=(java.net.URL)value;System.out.println(url.toExternalForm());System.out.println(value.getClass().getName());}}\n"
			runCampaignCompilerStrictProjectObservations(t, map[string]string{
				"pom.xml":                  "<project><modelVersion>4.0.0</modelVersion><groupId>url701</groupId><artifactId>migrated</artifactId><version>1</version></project>\n",
				"src/main/java/Probe.java": entry.source,
				"src/main/java/Main.java":  main,
			}, "Main", []campaignCompilerProjectObservation{
				{name: "first", stdout: "https://example.test/a\njava.net.URL\n"},
				{name: "second", stdout: "https://example.test/a\njava.net.URL\n"},
				{name: "third", stdout: "https://example.test/a\njava.net.URL\n"},
			})
		})
	}
}
