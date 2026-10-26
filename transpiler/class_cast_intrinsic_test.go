package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func classCastTestInvocation(t *testing.T, source string) (*sitter.Node, Ctx) {
	t.Helper()
	h := setupParseHelper(t, source)
	scope := h.File.Symbols.FindClassScope("Probe")
	if scope == nil {
		t.Fatal("Probe declaration missing")
	}
	ctx := classScopeCtx(scope, h.Ctx)
	defs := scope.FindMethod().ByOriginalName("run")
	if len(defs) != 1 {
		t.Fatal("run declaration missing")
	}
	ctx.localScope = defs[0]
	var found *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || found != nil {
			return
		}
		if n.Type() == "method_invocation" {
			name := n.ChildByFieldName("name")
			if name != nil && name.Content([]byte(source)) == "cast" {
				found = n
				return
			}
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(defs[0].DeclarationNode)
	if found == nil {
		t.Fatal("cast invocation missing")
	}
	return found, ctx
}

func TestClassCastIntrinsicTypedRawAndLiteral(t *testing.T) {
	for _, row := range []struct{ source, want string }{
		{`class Node{} class Probe{Object run(Class<Node> token,Object value){return token.cast(value);}}`, "ClassCast[*node]"},
		{`class Node{} class Probe{Object run(Class token,Object value){return token.cast(value);}}`, "ClassCast[any]"},
		{`class Node{} class Probe{Object run(Class<?> token,Object value){return token.cast(value);}}`, "ClassCast[any]"},
		{`class Node{} class Probe{Object run(Class<? extends Node> token,Object value){return token.cast(value);}}`, "ClassCast[*node]"},
		{`class Node{} class Probe{Object run(Class<? super Node> token,Object value){return token.cast(value);}}`, "ClassCast[any]"},
		{`class Node{} class Probe{Object run(Object value){return Node.class.cast(value);}}`, "ClassCast[*node]"},
		{`class Probe{Object run(Object value){return int.class.cast(value);}}`, "ClassCast[*stdjava.Integer]"},
		{`class Probe{<T>T run(Class<T> token,Object value){return token.cast(value);}}`, "ClassCast[T]"},
	} {
		t.Run(row.want+row.source, func(t *testing.T) {
			n, ctx := classCastTestInvocation(t, row.source)
			ctx.expectedType = "StringBuffer"
			var out bytes.Buffer
			err := printer.Fprint(&out, token.NewFileSet(), ParseExpr(n, []byte(row.source), ctx))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), row.want) {
				t.Fatalf("receiver-declared result missing: %s", out.String())
			}
		})
	}
}

func TestClassCastIntrinsicSourceClassShields(t *testing.T) {
	for _, source := range []string{
		`class Class{Object cast(Object v){return v;}}class Probe{Object run(Class token,Object value){return token.cast(value);}}`,
		`class Probe{static class java{static class lang{static class Class{Object cast(Object v){return v;}}}} Object run(java.lang.Class token,Object value){return token.cast(value);}}`,
		`class SourceClass{Object cast(Object v){return v;}}class Probe{<Class extends SourceClass>Object run(Class token,Object value){return token.cast(value);}}`,
	} {
		n, ctx := classCastTestInvocation(t, source)
		var out bytes.Buffer
		if err := printer.Fprint(&out, token.NewFileSet(), ParseExpr(n, []byte(source), ctx)); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "ClassCast[") {
			t.Fatalf("source declaration became JDK Class: %s", out.String())
		}
	}
}

func TestClassLiteralCanonicalDeclarationAndSourceRoots(t *testing.T) {
	for _, row := range []struct{ source, typ, want string }{
		{`class Probe{}`, "StringBuilder", `"java.lang.StringBuilder"`},
		{`class Probe{}`, "StringBuffer[]", `ArrayTypeID(stdjava.TypeID("java.lang.StringBuffer"))`},
		{`import java.sql.*;class Probe{}`, "Timestamp", `"java.sql.Timestamp"`},
		{`import java.io.StringReader;class Probe{}`, "StringReader", `"java.io.StringReader"`},
		{`class Probe{static class java{static class lang{static class StringBuilder{}}}}`, "StringBuilder", `"java.lang.StringBuilder"`},
	} {
		t.Run(row.typ+row.source, func(t *testing.T) {
			ctx := nativeExternalOrigin3930Owner(t, row.source, "Probe")
			expr, ok := classLiteralTypeDescriptorExpr(row.typ, ctx)
			if !ok {
				t.Fatal("literal proof refused")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), row.want) {
				t.Fatalf("canonical declaration lost: %s", out.String())
			}
		})
	}
	for _, row := range []struct{ source, typ string }{
		{`class StringBuilder{}class Probe{}`, "StringBuilder"},
		{`class Probe{static class java{static class lang{static class StringBuilder{}}}}`, "java.lang.StringBuilder"},
	} {
		ctx := nativeExternalOrigin3930Owner(t, row.source, "Probe")
		expr, ok := classLiteralTypeDescriptorExpr(row.typ, ctx)
		if !ok {
			t.Fatal("source literal refused")
		}
		var out bytes.Buffer
		if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Java2goClassLiteralTypeID") {
			t.Fatalf("source literal acquired native identity: %s", out.String())
		}
	}
}

func TestClassCastIntrinsicOriginAndExpectedTypeShields(t *testing.T) {
	source := `class Node{}class Probe{Object run(Class<Node> token,Object value){return token.cast(value);}}`
	n, ctx := classCastTestInvocation(t, source)
	for _, expected := range []string{"java.lang.StringBuffer", "Object", "Node", "Missing"} {
		ctx.expectedType = expected
		var origin inferredJavaTypeOrigin
		got, ok := inferExprJavaType(n, ctx, []byte(source), &origin)
		if !ok || origin.nominalScope == nil || origin.nominalScope.Class.OriginalName != "Node" || origin.parameter != nil || got != "Node" {
			t.Fatalf("consumer recaptured receiver result: expected=%s got=%s origin=%+v", expected, got, origin)
		}
	}
	for _, source := range []string{
		`import foreign.Class;class Probe{Object run(Class token,Object value){return token.cast(value);}}`,
		`class Probe{Object run(foreign.Class token,Object value){return token.cast(value);}}`,
		`class Probe{Object run(Missing token,Object value){return token.cast(value);}}`,
		`class Probe<Class>{Object run(Class token,Object value){return token.cast(value);}}`,
	} {
		n, ctx := classCastTestInvocation(t, source)
		if _, ok := classCastResultOrigin(n, ctx, []byte(source)); ok {
			t.Fatal("unproved receiver gained JDK cast")
		}
	}
}
