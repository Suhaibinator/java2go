package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"reflect"
	"testing"
)

func TestThreadSourceCanonicalOwnerRetainsNonvisibleDeclarationBindings(t *testing.T) {
	for _, test := range []struct{ name, typ, owner string }{
		{"thread", "java.lang.Thread", "java.lang.Thread"},
		{"runnable", "java.lang.Runnable", "java.lang.Runnable"},
		{"object", "java.lang.Object", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			h := setupParseHelper(t, `class Probe{static<T>`+test.typ+` select(T marker,`+test.typ+` value){return value;}}`)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = nil
			method := ctx.currentClass.FindMethodByName("select", nil)
			def := method.ParameterByName("value")
			if symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx)) != nil {
				t.Fatal("fixture unexpectedly has visible binders")
			}
			if def == nil || len(def.TypeParameterBindings) == 0 || def.TypeParameterBindings["T"] != method.TypeParameters[0].Declaration {
				t.Fatal("actual parameter declaration bindings missing")
			}
			saved := make(map[string]*symbol.TypeParamDeclaration, len(def.TypeParameterBindings))
			for name, parameter := range def.TypeParameterBindings {
				saved[name] = parameter
			}
			if got := threadSourceCanonicalOwner(def.OriginalType, def, ctx); got != test.owner {
				t.Fatalf("owner=%q want%q", got, test.owner)
			}
			if !reflect.DeepEqual(saved, def.TypeParameterBindings) {
				t.Fatal("declaration bindings mutated or dropped")
			}
		})
	}
}

func TestThreadSourceCanonicalOwnerBindingNamespaceAndConstructorControls(t *testing.T) {
	for _, source := range []string{
		`class Probe{static<Thread>Thread select(Thread value){return value;}}`,
		`class Probe{static<Runnable>Runnable select(Runnable value){return value;}}`,
		`import foreign.Thread;class Probe{static<T>Thread select(T marker,Thread value){return value;}}`,
		`class Thread{}class Probe{static<T>Thread select(T marker,Thread value){return value;}}`,
	} {
		t.Run(source, func(t *testing.T) {
			h := setupParseHelper(t, source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			method := ctx.currentClass.FindMethodByName("select", nil)
			def := method.ParameterByName("value")
			if got := threadSourceCanonicalOwner(def.OriginalType, def, ctx); got != "" {
				t.Fatalf("namespace shield admitted %q", got)
			}
		})
	}
	h := setupParseHelper(t, `class Probe{<T>Probe(T marker,java.lang.Runnable value){}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
	ctx.localScope = nil
	var constructor *symbol.Definition
	for _, method := range ctx.currentClass.Methods {
		if method.Constructor {
			constructor = method
		}
	}
	if constructor == nil || len(constructor.TypeParameters) != 1 {
		t.Fatal("registered constructor binder missing")
	}
	def := constructor.ParameterByName("value")
	if def == nil || def.TypeParameterBindings["T"] != constructor.TypeParameters[0].Declaration {
		t.Fatal("constructor parameter binding missing")
	}
	if got := threadSourceCanonicalOwner(def.OriginalType, def, ctx); got != "java.lang.Runnable" {
		t.Fatalf("constructor owner=%q", got)
	}
}
