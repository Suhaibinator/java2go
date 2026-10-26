package transpiler

import "testing"

func TestSourceGenericViewDemandLocalTypeScope(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       bool
	}{
		{name: "local generic shadow", body: `void f(){class Cell<T>{} Cell<?> local=null;}`},
		{name: "local raw cast shadow", body: `void f(){class Cell{} Object local=(Cell)null;}`},
		{name: "qualified demand beside local shadow", body: `void f(){class Cell<T>{} Cell<?> local=null;p.Cell<?> qualified=null;}`, want: true},
		{name: "source use before local declaration", body: `void f(){Cell<?> prior=null;class Cell<T>{} Cell<?> local=null;}`, want: true},
		{name: "sibling block source use", body: `void f(){{class Cell<T>{} Cell<?> local=null;}Cell<?> source=null;}`, want: true},
		{name: "different method source use", body: `void f(){class Cell<T>{} Cell<?> local=null;}Cell<?> g(){return null;}`, want: true},
		{name: "constructor local shadow", body: `Use(){class Cell<T>{} Cell<?> local=null;}`},
		{name: "ancestor block local shadow", body: `void f(){class Cell<T>{}{Cell<?> local=null;}}`},
		{name: "nested record binder shadow", body: `record Holder<Cell>(Cell value){}`},
		{name: "local record name shadow", body: `void f(){record Cell<T>(T value){} Cell<?> local=null;}`},
		{name: "local record qualified demand", body: `void f(){record Cell<T>(T value){} Cell<?> local=null;p.Cell<?> qualified=null;}`, want: true},
		{name: "nested annotation name shadow", body: `@interface Cell{} Object f(Object value){return (Cell)value;}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{
				"p/Cell.java":  `package p; public class Cell<T>{T value;}`,
				"app/Use.java": `package app; import p.Cell; public class Use{` + test.body + `}`,
			})
			member := findQualifiedSourceClass("p.Cell")
			if _, err := planGenericFamily(member, ctx); err != nil {
				t.Fatalf("control's source class fails unchanged audit: %v", err)
			}
			if got := canonicalGenericFamily(member, ctx) != nil; got != test.want {
				t.Errorf("imported p.Cell admitted = %t, want %t", got, test.want)
			}
		})
	}
}
