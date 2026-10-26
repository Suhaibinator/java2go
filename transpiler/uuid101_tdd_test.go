package transpiler

import (
	"strings"
	"testing"
)

func TestUUID101CanonicalOwnerAndABI(t *testing.T) {
	g := renderGoFileFromJava(t, `import java.util.UUID; public class UUIDProbe {static UUID make(String text){return UUID.fromString(text);}static UUID name(byte[] bytes){return UUID.nameUUIDFromBytes(bytes);}static UUID bits(long a,long b){return new UUID(a,b);}static String text(UUID value){return value.toString();}static int hash(UUID value){return value.hashCode();}static long msb(UUID value){return value.getMostSignificantBits();}static long lsb(UUID value){return value.getLeastSignificantBits();}static boolean equal(UUID value,Object other){return value.equals(other);}static int compare(UUID value,UUID other){return value.compareTo(other);}}`)
	for _, want := range []string{"*stdjava.JavaUUID", "stdjava.UUIDFromStringJavaString(", "stdjava.UUIDNameFromBytes(", "stdjava.NewJavaUUID(", ".StringJava2goExecution(", ".HashCode(", ".GetMostSignificantBits(", ".GetLeastSignificantBits(", ".Equals(", ".CompareTo("} {
		if !strings.Contains(g, want) {
			t.Errorf("missing canonical UUID ABI %q\n%s", want, g)
		}
	}
}
func TestUUID101SourceAndBinderShadows(t *testing.T) {
	for _, src := range []string{`class UUID {static UUID fromString(String s){return new UUID();}public String toString(){return "source";}} public class Probe {static UUID use(String s){return UUID.fromString(s);}static String text(UUID u){return u.toString();}}`, `public class Probe<UUID> {UUID value; UUID identity(UUID value){return value;}}`, `public class Probe {<UUID> UUID identity(UUID value){return value;}}`} {
		g := renderGoFileFromJava(t, src)
		for _, bad := range []string{"stdjava.JavaUUID", "UUIDFromStringJavaString", "UUIDNameFromBytes"} {
			if strings.Contains(g, bad) {
				t.Fatalf("UUID source/binder shadow borrowed JDK ABI %q\n%s", bad, g)
			}
		}
	}
	for _, name := range []string{"foreign.UUID", "other.UUID"} {
		if _, ok := stdjavaRuntimeTypeExpr(name, nil, nil, Ctx{}); ok {
			t.Fatalf("foreign UUID mapped: %s", name)
		}
	}
}
func TestUUID101StaticSourceCallerExecution(t *testing.T) {
	g := renderGoFileFromJava(t, `public class Probe {static class UUID {static synchronized String fromString(String value){return value;}} static String use(String value){return UUID.fromString(value);}}`)
	if strings.Contains(g, "UUIDFromStringJavaString") || strings.Contains(g, "stdjava.JavaUUID") {
		t.Fatalf("static source shadow mapped\n%s", g)
	}
	if !strings.Contains(g, "Java2goExecution") || !strings.Contains(g, "MonitorEnterExecution") {
		t.Fatalf("synchronized source caller lost execution\n%s", g)
	}
}
