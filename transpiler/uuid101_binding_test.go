package transpiler

import (
	"strings"
	"testing"
)

func TestUUID101CanonicalBindingsAndRefusals(t *testing.T) {
	for _, source := range []string{`import java.util.*;class Probe {static UUID use(String s){return UUID.fromString(s);}}`, `class Probe {static java.util.UUID use(String s){return java.util.UUID.fromString(s);}}`, `import static java.util.UUID.fromString;class Probe {static java.util.UUID use(String s){return fromString(s);}}`, `import static java.util.UUID.*;class Probe {static java.util.UUID use(byte[] b){return nameUUIDFromBytes(b);}}`} {
		g := renderGoFileFromJava(t, source)
		if !strings.Contains(g, "UUIDFromStringJavaString(") && !strings.Contains(g, "UUIDNameFromBytes(") {
			t.Fatalf("canonical UUID factory missing\n%s", g)
		}
	}
	for _, source := range []string{`import java.util.UUID;class Probe {static UUID use(){return UUID.fromString();}}`, `import java.util.UUID;class Probe {static UUID use(int n){return UUID.fromString(n);}}`, `import java.util.UUID;class Probe {static UUID use(String s){return UUID.nameUUIDFromBytes(s);}}`, `import java.util.UUID;class Probe {static UUID use(double d){return new UUID(d,1L);}}`, `import java.util.UUID;class Probe {static int use(UUID u,String s){return u.compareTo(s);}}`, `import java.util.UUID;class Probe {static int use(UUID u){return u.hashCode(1);}}`, `import java.util.UUID;class Probe {static UUID use(){return UUID.randomUUID();}}`, `import java.util.UUID;class Probe {static int use(UUID u){return u.version();}}`} {
		func() {
			strictRoutingState(t)
			defer func() {
				if p := recover(); p == nil || len(Diagnostics()) == 0 {
					t.Fatalf("unmodeled UUID invocation escaped refusal: %s", source)
				}
			}()
			renderGoFileFromJava(t, source)
		}()
	}
	// Foreign owners decline JDK routing; this differs from a malformed invocation
	// on the canonical supported owner, which must produce a strict diagnostic.
	strictRoutingState(t)
	foreignSource := `import foreign.UUID;class Probe {static UUID use(String s){return UUID.fromString(s);}}`
	helper := setupParseHelper(t, foreignSource)
	owner, known := canonicalIntrinsicOwner("UUID", helper.Ctx)
	if !known || owner != "foreign.UUID" || intrinsicOwnerSupported(owner) {
		t.Fatalf("foreign owner was borrowed or incorrectly supported: %q %v", owner, known)
	}
	if expression, mapped := uuidRuntimeTypeExpr("UUID", nil, helper.Ctx); mapped || expression != nil {
		t.Fatal("foreign UUID received a runtime type")
	}
	invocation := findNode(helper.File.Ast, "method_invocation")
	if invocation == nil {
		t.Fatal("missing foreign invocation fixture")
	}
	if expression, mapped := tryStaticIntrinsic(invocation.ChildByFieldName("object"), "fromString", helper.File.Source, helper.Ctx); mapped || expression != nil {
		t.Fatal("foreign UUID received a static JDK intrinsic")
	}
	generated := renderGoFileFromJava(t, foreignSource)
	for _, forbidden := range []string{"stdjava.JavaUUID", "stdjava.UUIDFromStringJavaString", "stdjava.UUIDNameFromBytes", "stdjava.NewJavaUUID"} {
		if strings.Contains(generated, forbidden) {
			t.Fatalf("foreign UUID borrowed %s\n%s", forbidden, generated)
		}
	}
	if len(Diagnostics()) != 0 {
		t.Fatalf("foreign decline diagnosed an unrelated JDK owner: %v", Diagnostics())
	}
}
