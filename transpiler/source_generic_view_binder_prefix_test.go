package transpiler

import "testing"

// The three invalid qualified names were independently rejected by JDK21:
// the captured p binder denotes Object, not package p.
func TestSourceGenericViewDemandBinderPrefix(t *testing.T) {
	for _, test := range []struct {
		name, use           string
		method, constructor bool
	}{
		{name: "method binder prefix", use: `package app;public class Use{<p> void f(){p.Cell<?> invalid=null;}}`, method: true},
		{name: "constructor binder prefix", use: `package app;public class Use{<p> Use(){p.Cell<?> invalid=null;}}`, constructor: true},
		{name: "class binder prefix", use: `package app;public class Use<p>{p.Cell<?> invalid;}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": test.use})
			owner := findQualifiedSourceClass("app.Use")
			ctx = classScopeCtx(owner, ctx)
			if test.method || test.constructor {
				for _, method := range owner.Methods {
					if method.Constructor == test.constructor && len(method.TypeParameters) > 0 {
						ctx.localScope = method
						break
					}
				}
			}
			if visibleTypeParameterDeclarationForJavaType("p", ctx) == nil {
				t.Fatal("fixture lost declaration-bound p")
			}
			if got := resolveClassScopeByQualifiedName(ctx, "p.Cell"); got != nil {
				t.Errorf("binder prefix selected package declaration %s", qualifiedSourceClassName(got))
			}
			if canonicalGenericFamily(findQualifiedSourceClass("p.Cell"), ctx) != nil {
				t.Error("binder-qualified miss spuriously activated package p.Cell")
			}
		})
	}
}

func TestSourceGenericViewDemandRecordHeader(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Cell.java":           `package p;public class Cell<T>{T value;}`,
		"app/RecordHeader.java": `package app;public record RecordHeader(p.Cell<?> cell){public static class p{public static class Cell<T>{T value;}}}`,
	})
	want := findQualifiedSourceClass("app.RecordHeader.p.Cell")
	found := false
	for _, seed := range sourceGenericViewDemandSeeds(ctx) {
		if seed == findQualifiedSourceClass("p.Cell") {
			t.Error("record header selected package p.Cell")
		}
		found = found || seed == want
	}
	if !found {
		t.Error("record header did not retain its own member type demand")
	}
}
