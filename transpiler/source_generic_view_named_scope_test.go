package transpiler

import "testing"

func TestSourceGenericViewDemandNamedPrefixShadow(t *testing.T) {
	for _, test := range []struct {
		name, use string
		extra     map[string]string
		want      string
	}{
		{name: "lexical member prefix", use: `package app;public class Use{static class p{static class Cell<T>{T value;}}p.Cell<?> value;}`, want: "app.Use.p.Cell"},
		{name: "imported type prefix", use: `package app;import origin.p;public class Use{p.Cell<?> value;}`, extra: map[string]string{"origin/p.java": `package origin;public class p{public static class Cell<T>{T value;}}`}, want: "origin.p.Cell"},
		{name: "same file type prefix", use: `package app;class p{static class Cell<T>{T value;}}public class Use{p.Cell<?> value;}`, want: "app.p.Cell"},
		{name: "package prefix unshadowed", use: `package app;public class Use{p.Cell<?> value;}`, want: "p.Cell"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := map[string]string{
				"p/Cell.java":  `package p;public class Cell<T>{T value;}`,
				"app/Use.java": test.use,
			}
			for name, source := range test.extra {
				sources[name] = source
			}
			ctx := sourceGenericViewDemandTestContext(t, sources)
			ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
			want := findQualifiedSourceClass(test.want)
			if want == nil {
				t.Fatalf("missing expected resolved declaration %s", test.want)
			}
			if got := resolveClassScopeByQualifiedName(ctx, "p.Cell"); got != want {
				t.Errorf("physical mapper resolver selects %q, want %q", qualifiedSourceClassName(got), test.want)
			}
			if _, err := planGenericFamily(want, ctx); err != nil {
				t.Fatalf("expected declaration fails unchanged complete audit: %v", err)
			}
			if canonicalGenericFamily(want, ctx) == nil {
				t.Errorf("wildcard demand did not activate its actual declaration %s", test.want)
			}
			if test.want != "p.Cell" && canonicalGenericFamily(findQualifiedSourceClass("p.Cell"), ctx) != nil {
				t.Error("member/import prefix spuriously activated package p.Cell")
			}
		})
	}
}
