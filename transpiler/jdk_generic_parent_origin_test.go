package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func jdkParentExistingAPIContext(t *testing.T) Ctx {
	ctx := ordinaryDemandTestContext(t, map[string]string{"p/Use.java": `package p;
 import java.util.List; import java.util.Collection; import java.util.ArrayDeque;
 class Value {} class Other {}
 public class Use<T> {
   static <E> Collection<E> retain(Collection<E> input) { return input; }
   <T> void caller(List<T> input) {}
 }`}, "p/Use.java")
 ctx.currentClass=ctx.currentFile.FindClassScope("Use")
 if ctx.currentClass==nil {t.Fatal("actual Use declaration missing")}
 return ctx
}

func TestJDKGenericParentExistingAPIConversion(t *testing.T) {
	ctx := jdkParentExistingAPIContext(t)
	origin := func(text string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: text}, ctx)
	}
	for _, actual := range []string{"List<Value>", "java.util.SequencedCollection<Value>", "ArrayDeque<Value>", "java.util.Deque<Value>", "java.util.Queue<Value>"} {
		t.Run(actual, func(t *testing.T) {
			value, want := origin(actual), origin("Collection<Value>")
			if value.unresolved || want.unresolved || value.externalName == "" || want.externalName == "" || len(value.arguments) != 1 || value.arguments[0].nominalScope == nil {
				t.Fatal("actual declaration origins unavailable")
			}
			cost, exact, applies, handled := declaredOriginConversion(value, want, ctx)
			t.Logf("actual=%s formal=%s conversion=(%d,%t,%t,%t)", actual, want.javaType, cost, exact, applies, handled)
			if !handled || !applies || exact || cost < 1 {
				t.Errorf("certified generic parent upcast must apply at non-exact reference cost")
			}
		})
	}
	for _, tc := range []struct{ actual, formal string }{
		{"List<Other>", "Collection<Value>"}, {"Collection<Value>", "List<Value>"}, {"List<Value>", "Collection<Other>"},
	} {
		_, _, applies, handled := declaredOriginConversion(origin(tc.actual), origin(tc.formal), ctx)
		if !handled || applies {
			t.Errorf("invariant/reverse conversion %s -> %s must remain rejected", tc.actual, tc.formal)
		}
	}
	raw := origin("List")
	formal := origin("Collection")
	cost, exact, applies, handled := declaredOriginConversion(raw, formal, ctx)
	if !handled || !applies || exact || cost < 1 {
		t.Errorf("raw certified upcast must preserve existing unchecked path, got %d/%t/%t/%t", cost, exact, applies, handled)
	}
	unresolved := origin("List<Value>")
	unresolved.unresolved = true
	if _, _, applies, handled := declaredOriginConversion(unresolved, origin("Collection<Value>"), ctx); applies || !handled {
		t.Fatal("unknown rejection changed")
	}
}

func TestJDKGenericParentExistingAPIGather(t *testing.T) {
	ctx := jdkParentExistingAPIContext(t)
	var helper, caller *symbol.Definition
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "retain" {
			helper = method
		}
		if method.OriginalName == "caller" {
			caller = method
		}
	}
	if helper == nil || caller == nil || len(helper.TypeParameters) != 1 || len(caller.TypeParameters) != 1 {
		t.Fatal("original method declaration missing")
	}
	e, actualT := helper.TypeParameters[0].Declaration, caller.TypeParameters[0].Declaration
	if e == nil || actualT == nil || e == actualT || actualT == ctx.currentClass.TypeParameters[0].Declaration {
		t.Fatal("method/class binder identities lost")
	}
	formal := methodFormalOrigins(helper, ctx.currentClass, nil, ctx)[0]
	for _, kind := range []string{"java.util.List", "java.util.ArrayDeque"} {
		for _, tc := range []struct {
			name, text string
			bindings   map[string]*symbol.TypeParamDeclaration
		}{
			{"source", kind + "<Value>", nil},
			{"caller-binder", kind + "<T>", map[string]*symbol.TypeParamDeclaration{"T": actualT}},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				actual := declaredJavaTypeOrigin(symbol.JavaType{Original: tc.text, TypeParameterBindings: tc.bindings}, ctx)
				candidates := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
				gatherMethodTypeOriginBindings(formal, actual, map[*symbol.TypeParamDeclaration]bool{e: true}, candidates)
				if len(candidates[e]) != 1 {
					t.Fatalf("actual method E received %d constraints across certified parent; want1", len(candidates[e]))
				}
				if !sameJavaTypeOrigin(candidates[e][0], actual.arguments[0]) {
					t.Fatal("gather must preserve exact supplied declaration pointer and nested origin")
				}
			})
		}
	}
}
