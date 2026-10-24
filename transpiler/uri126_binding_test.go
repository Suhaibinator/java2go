package transpiler

import (
	"strings"
	"testing"
)

func TestURI126DeclaredOwnerAndComparableGuards(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"foreign/URI.java": "package foreign;public class URI {}",
		"app/Probe.java":   "package app;import java.net.URI;public class Probe {}",
	}, "app/Probe.java")
	for _, entry := range []struct {
		actual, expected string
		want             bool
	}{
		{"java.net.URI", "java.lang.Object", true}, {"URI", "java.io.Serializable", true}, {"URI", "Comparable<URI>", true},
		{"URI", "Comparable<foreign.URI>", false}, {"URI", "Comparable<java.lang.String>", false}, {"foreign.URI", "java.io.Serializable", false}, {"URI[]", "java.io.Serializable", false},
	} {
		if got := uriReferenceAssignable(entry.actual, entry.expected, nil, ctx); got != entry.want {
			t.Errorf("%s -> %s = %t, want %t", entry.actual, entry.expected, got, entry.want)
		}
	}
}

func TestURI126SourceBinderAndForeignDecline(t *testing.T) {
	for _, source := range []string{
		`class URI {String toASCIIString(){return "source";}} public class Probe {URI value;}`,
		`public class Probe<URI> {URI value;URI identity(URI value){return value;}}`,
		`import foreign.URI;public class Probe {URI value;}`,
	} {
		generated := renderGoFileFromJava(t, source)
		for _, native := range []string{"stdjava.JavaURI", "NewJavaURIStringExecution", "ToASCIIStringJava2goExecution"} {
			if strings.Contains(generated, native) {
				t.Fatalf("source/binder/foreign borrowed URI service %s: %s", native, generated)
			}
		}
	}
}

func TestURI126ConstructorAndInvocationStrictRefusals(t *testing.T) {
	for _, source := range []string{
		`class Probe {java.net.URI value() throws Exception{return new java.net.URI("http",null,"host",-1,"/a",null,null);}}`,
		`class Probe {String value(java.net.URI value){return value.toASCIIString(1);}}`,
		`class Probe {java.net.URISyntaxException value(){return new java.net.URISyntaxException("only-one");}}`,
	} {
		func() {
			strictRoutingState(t)
			defer func() {
				if observed := recover(); observed == nil || len(Diagnostics()) == 0 {
					t.Fatalf("unadmitted declaration escaped strict refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		}()
	}
}
