package transpiler

import (
	"encoding/json"
	"os"
	"testing"
)

func TestJDK21NominalActualOriginFrontiers(t *testing.T) {
	for _, row := range []struct{ header, written, canonical string }{
		{"import java.nio.channels.*;", "java.nio.ByteBuffer", "java.nio.ByteBuffer"},
		{"import java.nio.channels.ReadableByteChannel;", "ReadableByteChannel", "java.nio.channels.ReadableByteChannel"},
		{"import java.nio.channels.*;", "ReadableByteChannel", "java.nio.channels.ReadableByteChannel"},
		{"import java.nio.channels.*;", "WritableByteChannel", "java.nio.channels.WritableByteChannel"},
		{"import java.nio.charset.Charset;", "Charset", "java.nio.charset.Charset"},
		{"import java.nio.charset.*;", "Charset", "java.nio.charset.Charset"},
		{"import java.nio.channels.*;", "java.nio.charset.Charset", "java.nio.charset.Charset"},
		{"", "java.util.concurrent.ConcurrentMap", "java.util.concurrent.ConcurrentMap"},
		{"import java.util.concurrent.ConcurrentMap;", "ConcurrentMap", "java.util.concurrent.ConcurrentMap"},
		{"import java.util.concurrent.*;", "ConcurrentMap", "java.util.concurrent.ConcurrentMap"},
	} {
		t.Run(row.header+row.written, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, row.header+"class Probe{Object run("+row.written+" value){return value;}}", "Probe")
			got := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if got.unresolved || got.externalName != row.canonical || got.nominalScope != nil || got.parameter != nil {
				t.Fatalf("actual declaration origin=%+v; want %s", got, row.canonical)
			}
		})
	}
}
func TestJDK21NominalNamespaceShields(t *testing.T) {
	for _, src := range []string{
		"import mystery.*;import java.nio.channels.*;class Probe{Object run(java.nio.ByteBuffer value){return value;}}",
		"import mystery.*;import java.nio.charset.*;class Probe{Object run(Charset value){return value;}}",
		"import java.nio.charset.Charset;class Probe<Charset>{Object run(Charset value){return value;}}",
		"import java.nio.charset.Charset;class Probe{static class Charset{}Object run(Charset value){return value;}}",
		"class java{static class nio{static class charset{static class Charset{}}}}class Probe{Object run(java.nio.charset.Charset value){return value;}}",
		"import java.util.*;import java.sql.*;class Probe{Object run(Date value){return value;}}",
		"import java.util.*;import java.awt.*;class Probe{Object run(List value){return value;}}",
	} {
		t.Run(src, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, src, "Probe")
			got := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			if got.externalName != "" {
				t.Fatalf("namespace/binder authority lost: %+v", got)
			}
		})
	}
}

func TestJDK21NominalPresenceAndEligibility(t *testing.T) {
	for _, row := range []struct {
		pkg, name string
		presence  jdk21NominalPresence
		eligible  bool
	}{
		{"java.nio.channels", "ReadableByteChannel", jdk21NominalPresent, true},
		{"java.nio.charset", "Charset", jdk21NominalPresent, true},
		{"java.util.concurrent", "ConcurrentMap", jdk21NominalPresent, true},
		{"java.nio.channels", "java", jdk21NominalAbsentCertified, false},
		{"java.nio.charset", "CharsetEncoderExtra", jdk21NominalAbsentCertified, false},
		{"java.lang", "StringTemplate", jdk21NominalPresent, false},
		{"java.util", "FormatProcessor", jdk21NominalPresent, false},
		{"java.io", "ClassCache", jdk21NominalPresent, false},
		{"jdk.internal.misc", "Unsafe", jdk21NominalUnknown, false},
		{"java.sql", "DriverInfo", jdk21NominalUnknown, false},
		{"java.util.Map", "Entry", jdk21NominalUnknown, false},
		{"mystery", "Charset", jdk21NominalUnknown, false},
		{"java.nio.charset", "spi.CharsetProvider", jdk21NominalUnknown, false},
	} {
		t.Run(row.pkg+"/"+row.name, func(t *testing.T) {
			got, presence := lookupJDK21TopLevel(row.pkg, row.name)
			if presence != row.presence || jdk21NominalEligible(got) != row.eligible {
				t.Fatalf("lookup=%+v/%d want presence%d eligible%t", got, presence, row.presence, row.eligible)
			}
		})
	}
}
func TestJDK21NominalDeclaringOwnership(t *testing.T) {
	for _, missing := range []string{"file", "class", "stale"} {
		t.Run(missing, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, "class Probe{Object run(Object value){return value;}}", "Probe")
			switch missing {
			case "file":
				ctx.currentFile = nil
			case "class":
				ctx.currentClass = nil
			case "stale":
				ctx.currentFile = ordinaryDemandTestContext(t, map[string]string{"Other.java": "class Other{}"}, "Other.java").currentFile
			}
			if got := declaredExternalNominalOwner("java.nio.charset.Charset", nil, ctx); got != "" {
				t.Fatalf("missing declaring owner/file certified %s", got)
			}
		})
	}
}
func TestJDK21NominalPrefixAbsenceAndUnknown(t *testing.T) {
	for _, row := range []struct {
		header string
		want   bool
	}{
		{"import java.nio.channels.*;", true},
		{"import java.nio.charset.*;import java.util.concurrent.*;", true},
		{"import java.nio.channels.*;import mystery.*;", false},
		{"import java.nio.channels.*;import static mystery.Owner.*;", false},
	} {
		t.Run(row.header, func(t *testing.T) {
			ctx := nativeChannelOriginContext(t, row.header+"class Probe{Object run(Object value){return value;}}", "Probe")
			if got := declaredInterfaceDemandPrefixAbsent("java", ctx); got != row.want {
				t.Fatalf("prefix absence=%t want%t", got, row.want)
			}
		})
	}
}

func TestJDK21NominalGeneratedTableMatchesCertificate(t *testing.T) {
	data, err := os.ReadFile("testdata/jdk21nominal/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		Profile  string `json:"profile"`
		Packages map[string]struct {
			Certificate  string `json:"certificateID"`
			Declarations map[string]struct {
				BinaryName                            string `json:"binaryName"`
				Public, Interface, Preview, Synthetic bool
			} `json:"declarations"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.Profile != jdk21NominalProfile || len(inventory.Packages) != len(jdk21NominalPackages) {
		t.Fatal("profile/package mismatch")
	}
	count := 0
	for pkg, certificate := range inventory.Packages {
		got := jdk21NominalPackages[pkg]
		if got.certificate != certificate.Certificate || len(got.declarations) != len(certificate.Declarations) {
			t.Fatalf("package certificate mismatch %s", pkg)
		}
		for name, want := range certificate.Declarations {
			decl, state := lookupJDK21TopLevel(pkg, name)
			if state != jdk21NominalPresent || decl.binaryName != want.BinaryName || decl.public != want.Public || decl.isInterface != want.Interface || decl.preview != want.Preview || decl.synthetic != want.Synthetic {
				t.Fatalf("record mismatch %s.%s: %+v versus %+v", pkg, name, decl, want)
			}
			count++
		}
	}
	if count != 1561 {
		t.Fatalf("complete pinned record count=%d", count)
	}
}
