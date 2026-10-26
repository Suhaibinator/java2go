package transpiler

import (
	"strings"
	"testing"
)

func TestInetAddressNativeDemand(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.net.InetAddress;import java.net.UnknownHostException;class Probe {InetAddress address(String text)throws Exception{return InetAddress.getByName(text);}String show(InetAddress value){return value.getHostAddress();} UnknownHostException error(String text){return new UnknownHostException(text);}}`)
	for _, want := range []string{"*stdjava.JavaInetAddress", "JavaInetAddressGetByNameExecution", "GetHostAddressJava2goExecution", "NewJavaUnknownHostExceptionMessage"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s: %s", want, out)
		}
	}
}
func TestInetAddressSourceHomonyms(t *testing.T) {
	for _, source := range []string{`class InetAddress{String getHostAddress(){return "own";}}class Probe{InetAddress address;}`, `class Probe<InetAddress>{InetAddress address;}`, `import foreign.InetAddress;class Probe{InetAddress address;}`} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "stdjava.JavaInetAddress") {
			t.Fatalf("homonym acquired platform carrier: %s", out)
		}
	}
}
func TestInetAddressUnimplementedMembersRefuse(t *testing.T) {
	for _, source := range []string{`class Probe {Object value()throws Exception{return java.net.InetAddress.getLocalHost();}}`, `class Probe {String value(java.net.InetAddress a){return a.getCanonicalHostName();}}`, `class Probe {Object value()throws Exception{return java.net.InetAddress.getAllByName("host");}}`} {
		func() {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatalf("unsupported service escaped: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		}()
	}
}

func TestInetAddressDeclaredOwnership(t *testing.T) {
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"foreign/InetAddress.java": `package foreign;public class InetAddress {}`,
		"app/Probe.java":           `package app;import java.net.InetAddress;class Probe {}`,
	}, "app/Probe.java")
	for _, row := range []struct {
		actual, expected string
		want             bool
	}{
		{"InetAddress", "java.lang.Object", true}, {"java.net.InetAddress", "java.io.Serializable", true}, {"InetAddress", "java.net.InetAddress", true}, {"foreign.InetAddress", "java.lang.Object", false}, {"InetAddress[]", "java.lang.Object", false}, {"InetAddress", "java.lang.Comparable", false},
	} {
		if got := inetAddressReferenceAssignable(row.actual, row.expected, nil, ctx); got != row.want {
			t.Errorf("%s -> %s=%t", row.actual, row.expected, got)
		}
	}
	for _, source := range []string{
		`import foreign.InetAddress;class Probe {Object value(String text){return InetAddress.getByName(text);}}`,
		`class Probe<InetAddress>{String value(InetAddress address){return address.getHostAddress();}}`,
		`class java {static class net {static class InetAddress {}}}class Probe{java.net.InetAddress address;}`,
	} {
		out := renderGoFileFromJava(t, source)
		for _, native := range []string{"*stdjava.JavaInetAddress", "JavaInetAddressGetByNameExecution", "GetHostAddressJava2goExecution"} {
			if strings.Contains(out, native) {
				t.Fatalf("unowned type borrowed %s: %s", native, out)
			}
		}
	}
}

func TestUnknownHostExceptionOwnedConstructor(t *testing.T) {
	for _, source := range []string{
		`class UnknownHostException {UnknownHostException(String text){}}class Probe {UnknownHostException value(){return new UnknownHostException("own");}}`,
		`import foreign.UnknownHostException;class Probe {UnknownHostException value;}`,
		`class Probe<UnknownHostException> {UnknownHostException value;}`,
	} {
		out := renderGoFileFromJava(t, source)
		if strings.Contains(out, "stdjava.UnknownHostException") || strings.Contains(out, "NewJavaUnknownHostExceptionMessage") {
			t.Fatalf("unowned checked exception borrowed builtin: %s", out)
		}
	}
	for _, source := range []string{`class Probe {Object value(){return new java.net.UnknownHostException(new Object());}}`, `class Probe {Object value(){return new java.net.UnknownHostException("message",new Exception());}}`} {
		func() {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatalf("unadmitted constructor escaped: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		}()
	}
}
