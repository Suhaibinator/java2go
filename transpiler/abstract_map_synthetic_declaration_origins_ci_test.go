package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// This test asks the existing declaration-origin APIs about a synthesized Java
// platform signature. Expectations come from unchanged parsed source formals,
// not from the synthesizer's strings or a guessed emitted binder spelling.
func TestAbstractMapSyntheticDeclarationOriginsCI(t *testing.T) {
	for _, test := range []struct {
		name, source string
		renamed      bool
		fresh        bool
	}{
		{"ordinary", `package synthetic;
abstract class Base<K,V> extends java.util.AbstractMap<K,V> {
 V relay(K key,V value) { return put(key,value); }
 <K> V shadow(K ignored,V value) { return null; }
}`, false, false},
		{"renamed-and-namespace", `package synthetic;
class K {} class V {} class K2 {} class V2 {}
abstract class Base<K,V> extends java.util.AbstractMap<K,V> {
 V relay(K key,V value) { return put(key,value); }
 <K> V shadow(K ignored,V value) { return null; }
}`, true, false},
		{"captured-enclosing", `package synthetic;
class Outer<K> {
 abstract class Base<V> extends java.util.AbstractMap<K,V> {
  V relay(K key,V value) { return put(key,value); }
  <K> V shadow(K ignored,V value) { return null; }
 }
}`, false, false},
		{"inner-owner-shadows-enclosing", `package synthetic;
class Outer<K> {
 abstract class Base<K,V> extends java.util.AbstractMap<K,V> {
  V relay(K key,V value) { return put(key,value); }
  <K> V shadow(K ignored,V value) { return null; }
 }
}`, false, false},
		{"fresh-synthesis-after-renaming", `package synthetic;
class K {} class V {} class K2 {} class V2 {}
abstract class Base<K,V> extends java.util.AbstractMap<K,V> {
 V relay(K key,V value) { return put(key,value); }
 <K> V shadow(K ignored,V value) { return null; }
}`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			var owner *symbol.ClassScope
			var visit func(*symbol.ClassScope)
			visit = func(scope *symbol.ClassScope) {
				if scope == nil {
					return
				}
				if scope.Class != nil && scope.Class.OriginalName == "Base" {
					owner = scope
				}
				for _, nested := range scope.Subclasses {
					visit(nested)
				}
			}
			for _, top := range helper.File.Symbols.TopLevelClasses {
				visit(top)
			}
			if owner == nil {
				t.Fatal("parsed Base owner missing")
			}
			relay := owner.FindMethodByName("relay", nil)
			shadow := owner.FindMethodByName("shadow", nil)
			if relay == nil || len(relay.Parameters) != 2 || shadow == nil || len(shadow.TypeParameters) != 1 {
				t.Fatal("parsed owner/relay/shadow declarations missing")
			}
			key := relay.Parameters[0].DirectTypeParameter
			value := relay.Parameters[1].DirectTypeParameter
			if key == nil || value == nil || key == value || shadow.TypeParameters[0].Declaration == key {
				t.Fatal("source declarations do not retain distinct owner K/V and method K identities")
			}
			if test.renamed && (key.GoName == key.SourceName || value.GoName == value.SourceName) {
				t.Fatal("source nominal K/V fixture must reserve both binder spellings")
			}
			ctx := classScopeCtx(owner, helper.Ctx)
			// A donor method and synthetic binder must never capture the
			// platform declaration's K; preparation uses the owner context.
			ctx.localScope = shadow
			ctx.syntheticTypeParameters = []symbol.TypeParam{symbol.NewTypeParam("K", nil)}
			if test.fresh {
				var authored []*symbol.Definition
				for _, method := range owner.Methods {
					if !method.RuntimeDefault {
						authored = append(authored, method)
					}
				}
				owner.Methods = authored
				prepareAbstractMapDefaultsForScope(owner, ctx)
			}
			before := append([]*symbol.Definition(nil), owner.Methods...)
			prepareAbstractMapDefaultsForScope(owner, ctx)
			prepareAbstractMapDefaultsForScope(owner, ctx)
			if len(owner.Methods) != len(before) {
				t.Fatalf("preparation is not idempotent: %d methods became %d", len(before), len(owner.Methods))
			}
			for i, method := range before {
				if owner.Methods[i] != method {
					t.Fatal("preparation replaced an existing declaration")
				}
			}
			var put, keySet *symbol.Definition
			for _, method := range owner.Methods {
				if method.RuntimeDefault && method.OriginalName == "put" {
					if put != nil {
						t.Fatal("duplicate synthesized put")
					}
					put = method
				}
				if method.RuntimeDefault && method.OriginalName == "keySet" {
					keySet = method
				}
			}
			if put == nil || len(put.Parameters) != 2 || keySet == nil {
				t.Fatal("required inherited platform declarations missing")
			}
			formals := methodFormalOrigins(put, owner, nil, ctx)
			if len(formals) != 2 || formals[0].unresolved || formals[1].unresolved || formals[0].parameter != key || formals[1].parameter != value {
				t.Fatalf("synthetic put formals must retain parsed owner declarations: %#v", formals)
			}
			if put.Parameters[0].DirectTypeParameter != key || put.Parameters[1].DirectTypeParameter != value || put.DirectTypeParameter != value {
				t.Fatal("synthetic direct-parameter identities differ from source owner K/V")
			}
			declaring := classScopeCtx(owner, ctx)
			declaring.localScope = put
			declaring.syntheticTypeParameters = nil
			result := declaredJavaTypeOrigin(definitionDeclaredJavaType(put), declaring)
			if result.unresolved || result.parameter != value {
				t.Fatalf("synthetic put return lost source V: %#v", result)
			}
			keys := declaredJavaTypeOrigin(definitionDeclaredJavaType(keySet), declaring)
			if keys.unresolved || len(keys.arguments) != 1 || keys.arguments[0].unresolved || keys.arguments[0].parameter != key {
				t.Fatalf("synthetic parameterized return lost source K: %#v", keys)
			}
			for _, method := range owner.Methods {
				if !method.RuntimeDefault {
					continue
				}
				for _, definition := range append([]*symbol.Definition{method}, method.Parameters...) {
					component, _ := javaArrayTypeParts(definition.OriginalType)
					typ := definitionDeclaredJavaType(definition)
					typ.Original = component
					origin := declaredJavaTypeOrigin(typ, declaring)
					if origin.parameter != nil && definition.DirectTypeParameter != origin.parameter {
						t.Fatalf("synthetic direct origin is absent or rebound for %s %s", method.OriginalName, definition.OriginalType)
					}
				}
			}
			if test.renamed {
				nominal := declaredJavaTypeOrigin(symbol.JavaType{Original: "synthetic.K2"}, declaring)
				if nominal.nominalScope == nil || nominal.parameter != nil || nominal.unresolved {
					t.Fatal("qualified source homonym was authorized as an emitted binder alias")
				}
			}
			// Preserve the original invocation and exact selected definition.
			caller := classScopeCtx(owner, helper.Ctx)
			caller.localScope = relay
			invocation := findNode(relay.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("original put invocation missing")
			}
			selected := findBestMethodInHierarchy(owner, "put", invocation.ChildByFieldName("arguments"), true, false, caller, helper.File.Source)
			if selected == nil || selected.owner != owner || selected.def != put {
				t.Fatal("original implicit call must select the exact synthesized owner declaration")
			}
		})
	}
}
