package transpiler

import "testing"

func TestSourceGenericViewDemandContextBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, use, want string
		extra           map[string]string
		header, demand  bool
	}{
		{name: "missing lexical prefix member", use: `package app;public class Use{static class p{}p.Cell<?> invalid;}`},
		{name: "missing imported prefix member", use: `package app;import origin.p;public class Use{p.Cell<?> invalid;}`, extra: map[string]string{"origin/p.java": `package origin;public class p{}`}},
		{name: "missing same file prefix member", use: `package app;class p{}public class Use{p.Cell<?> invalid;}`},
		{name: "class header package prefix", use: `package app;public class Use extends p.Cell<String>{static class p{static class Cell<T>{T value;}}}`, want: "p.Cell", header: true},
		{name: "class header imported prefix", use: `package app;import origin.p;public class Use extends p.Cell<String>{static class p{static class Cell<T>{T value;}}}`, extra: map[string]string{"origin/p.java": `package origin;public class p{public static class Cell<T>{T value;}}`}, want: "origin.p.Cell", header: true},
		{name: "switch same group local shadow", use: `package app;import p.Cell;public class Use{void f(int value){switch(value){case 0:class Cell<T>{}Cell<?> local=null;break;default:break;}}}`},
		{name: "switch later group imported demand", use: `package app;import p.Cell;public class Use{void f(int value){switch(value){case 0:class Cell<T>{}Cell<?> local=null;break;case 1:Cell<?> source=null;break;}}}`, demand: true},
		{name: "switch prior group imported demand", use: `package app;import p.Cell;public class Use{void f(int value){switch(value){case 0:Cell<?> source=null;break;case 1:class Cell<T>{}Cell<?> local=null;break;}}}`, demand: true},
		{name: "switch qualified demand", use: `package app;import p.Cell;public class Use{void f(int value){switch(value){case 0:class Cell<T>{}Cell<?> local=null;break;case 1:p.Cell<?> source=null;break;}}}`, demand: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": test.use}
			for name, source := range test.extra {
				sources[name] = source
			}
			ctx := sourceGenericViewDemandTestContext(t, sources)
			use := findQualifiedSourceClass("app.Use")
			ctx = classScopeCtx(use, ctx)
			if test.header {
				if got := resolveSuperclassScopeInDeclaringContext(ctx, use); qualifiedSourceClassName(got) != test.want {
					t.Fatalf("superclass header resolved to %q, want %q", qualifiedSourceClassName(got), test.want)
				}
				return
			}
			if test.name == "missing lexical prefix member" || test.name == "missing imported prefix member" || test.name == "missing same file prefix member" {
				if got := resolveClassScopeByQualifiedName(ctx, "p.Cell"); got != nil {
					t.Errorf("missing member in bound prefix selected %s", qualifiedSourceClassName(got))
				}
			}
			if got := canonicalGenericFamily(findQualifiedSourceClass("p.Cell"), ctx) != nil; got != test.demand {
				t.Errorf("package p.Cell demand = %t, want %t", got, test.demand)
			}
		})
	}
}
