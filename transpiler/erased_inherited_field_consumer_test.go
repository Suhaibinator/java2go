package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// The original independently JVM-validated bridge_dispatch fixture returns an
// inherited Base<T> field with an implicit receiver from Text extends Base<String>.
// Its physical storage remains Object; the consuming String result must perform
// javac's delayed checkcast. These focused controls retain declaration identity
// and the existing method/local/hidden-field lookup precedence.
func TestErasedInheritedFieldImplicitConsumer(t *testing.T) {
	for _, tc := range []struct {
		name, owner, method, extra string
		wantCast                   bool
	}{
		{name: "implicit inherited return", owner: "Text", method: `String read(){return value;}`, wantCast: true},
		{name: "explicit inherited return", owner: "Text", method: `String read(){return this.value;}`, wantCast: true},
		{name: "super inherited return", owner: "Text", method: `String read(){return super.value;}`, wantCast: true},
		{name: "parenthesized inherited return", owner: "Text", method: `String read(){return (value);}`, wantCast: true},
		{name: "broad consumer", owner: "Text", method: `Object read(){return value;}`},
		{name: "parameter shadows field", owner: "Text", method: `String read(String value){return value;}`},
		{name: "local shadows field", owner: "Text", method: `String read(){String value="local";return value;}`},
		{name: "own field hides inherited", owner: "Text", extra: `String value;`, method: `String read(){return value;}`},
		{name: "static field hides inherited", owner: "Text", extra: `static String value;`, method: `static String read(){return value;}`},
		{name: "generic owner return remains erased", owner: "Base", method: `T read(){return value;}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseMethod, textMethod := "", tc.method
			if tc.owner == "Base" {
				baseMethod, textMethod = tc.method, ""
			}
			h := setupParseHelper(t, `class Base<T>{T value;`+baseMethod+`} class Text extends Base<String>{`+tc.extra+textMethod+`} class Use{Base<?> demand;}`)
			owner := h.File.Symbols.FindClassScope(tc.owner)
			if owner == nil {
				t.Fatal("missing source owner")
			}
			if canonicalGenericFamily(h.File.Symbols.FindClassScope("Base"), h.Ctx) == nil {
				t.Fatal("control lost complete audited canonical family")
			}
			var method *symbol.Definition
			for _, candidate := range owner.Methods {
				if candidate.OriginalName == "read" {
					method = candidate
					break
				}
			}
			if method == nil {
				t.Fatal("missing source method")
			}
			statement := findNode(method.DeclarationNode, "return_statement")
			if statement == nil {
				t.Fatal("missing return")
			}
			ctx := classScopeCtx(owner, h.Ctx)
			ctx.className = owner.Class.Name
			ctx.localScope = method
			expr := statement.NamedChild(0)
			ctx.expectedType = method.OriginalType
			ctx.expectedTypeRoot = expr
			rendered := symbol.NodeToStr(parseReturnValue(expr, h.File.Source, ctx))
			got := strings.Contains(rendered, "ObjectView[")
			if got != tc.wantCast {
				t.Fatalf("delayed cast=%t want %t: %s", got, tc.wantCast, rendered)
			}
			if tc.wantCast && (!strings.Contains(rendered, "JavaString") || !strings.Contains(rendered, "StringTypeID")) {
				t.Fatalf("wrong physical or nominal cast: %s", rendered)
			}
		})
	}
}
