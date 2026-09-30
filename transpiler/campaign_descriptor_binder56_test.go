package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// The parser's declaration pointers distinguish the outer T captured by U's
// bound from the method T. The emitted names model the actual generic helper
// context in the unchanged expanded workflow, where inferred array types have
// already been qualified by declaration identity.
func descriptorBinderContext56(t *testing.T) Ctx {
	t.Helper()
	helper := setupParseHelper(t, `package descriptor;
class Scope<T extends java.lang.CharSequence, U extends T> {
 <T extends java.lang.Number> void nested() {}
}
class T {}
`)
	ctx := helper.Ctx
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Scope")
	if ctx.currentClass == nil {
		t.Fatal("missing Scope")
	}
	ctx.localScope = ctx.currentClass.FindMethodByName("nested", nil)
	if ctx.localScope == nil {
		t.Fatal("missing nested method")
	}
	combined := symbol.AppendTypeParamsByDeclaration(ctx.currentClass.TypeParameters, ctx.localScope.TypeParameters)
	symbol.DisambiguateTypeParamGoNames(combined)
	if got := ctx.localScope.TypeParameters[0].EmittedName(); got != "T2" {
		t.Fatalf("fixture did not create distinct emitted identities: %q", got)
	}
	return ctx
}

func TestCampaignDescriptorBinderErasure56(t *testing.T) {
	ctx := descriptorBinderContext56(t)
	for _, test := range []struct {
		name, written, want string
		bound               bool
	}{
		{"emitted-method", "T2", "java.lang.Number", true},
		{"emitted-outer", "T", "java.lang.CharSequence", true},
		{"captured-outer-first-bound", "U", "java.lang.CharSequence", true},
		{"qualified-source-class", "descriptor.T", "", false},
		{"qualified-external-class", "other.T", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, bound := javaTypeParameterErasure(test.written, ctx)
			if got != test.want || bound != test.bound {
				t.Fatalf("erasure(%q) = (%q,%v), want (%q,%v)", test.written, got, bound, test.want, test.bound)
			}
		})
	}
}

func TestCampaignDescriptorBinderArrayRank56(t *testing.T) {
	ctx := descriptorBinderContext56(t)
	for _, test := range []struct{ written, want string }{
		{"T[]", "stdjava.ArrayTypeID(stdjava.CharSequenceTypeID)"},
		{"T2[]", "stdjava.ArrayTypeID(stdjava.NumberTypeID)"},
		{"T2[][]", "stdjava.ArrayTypeID(stdjava.ArrayTypeID(stdjava.NumberTypeID))"},
		{"U[][]", "stdjava.ArrayTypeID(stdjava.ArrayTypeID(stdjava.CharSequenceTypeID))"},
		{"descriptor.T[]", `stdjava.ArrayTypeID(stdjava.TypeID("descriptor.T"))`},
	} {
		t.Run(test.written, func(t *testing.T) {
			expr, ok := javaTypeDescriptorExpr(test.written, ctx)
			if !ok {
				t.Fatal("missing descriptor")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != test.want {
				t.Fatalf("descriptor(%q) = %s, want %s", test.written, got, test.want)
			}
		})
	}
}

// A source-written T in a cast denotes the method binder, while an inferred T
// naming the outer declaration denotes the outer binder. Both paths must keep
// working; changing one global string-name precedence cannot establish this.
func TestCampaignDescriptorSourceCastShadow56(t *testing.T) {
	out := renderGoFileFromJava(t, `
class DescriptorSourceCast<T extends java.lang.CharSequence> {
 <T extends java.lang.Number> T[] cast(Object value) { return (T[]) value; }
}
`)
	if !strings.Contains(out, "stdjava.JavaArrayCast[*stdjava.ReferenceArray](value, stdjava.ArrayTypeID(stdjava.NumberTypeID))") {
		t.Fatalf("source cast must use the method's Number first bound:\n%s", out)
	}
}
