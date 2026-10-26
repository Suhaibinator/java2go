package transpiler

import (
	"strings"
	"testing"
)

func TestBoundedSourceClassGoArgumentSecondaryProducers(t *testing.T) {
	t.Run("raw_ancestor", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;}`,
			"app/Use.java": `package app;public class Use extends p.Holder{}`,
		})
		child, parent := findQualifiedSourceClass("app.Use"), findQualifiedSourceClass("p.Holder")
		ctx = classScopeCtx(child, ctx)
		args := mapClassTypeArgsToAncestor(child, nil, parent, ctx)
		if len(args) != 1 { t.Fatalf("ancestor argument count = %d, want 1", len(args)) }
		if got := boundedArgumentTestGoType(t, args[0]); got != "*stdjava.List[*stdjava.JavaString]" {
			t.Errorf("Go raw ancestor argument representation = %s, want preserved List<String>", got)
		}
	})
	t.Run("raw_receiver", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;public int size(){return 0;}}`,
			"app/Use.java": `package app;public class Use{p.Holder holder;public int size(){return holder.size();}}`,
		})
		ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
		for _, method := range ctx.currentClass.Methods { if method.OriginalName == "size" { ctx.localScope = method } }
		node := findNode(ctx.currentClass.Class.DeclarationNode, "method_invocation")
		if node == nil { t.Fatal("missing source invocation") }
		target := resolveInvocationTarget(node.ChildByFieldName("object"), ctx, ctx.currentFile.Source)
		if target == nil || len(target.classTypeArgs) != 1 { t.Fatal("missing receiver class argument") }
		if got := boundedArgumentTestGoType(t, target.classTypeArgs[0]); got != "*stdjava.List[*stdjava.JavaString]" {
			t.Errorf("Go raw receiver argument representation = %s, want preserved List<String>", got)
		}
	})
	t.Run("raw_anonymous_constructor", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Holder.java": `package p;public class Holder<T extends java.util.List<String>>{public T value;public Holder(){}}`,
			"app/Use.java": `package app;public class Use{public Object make(){return new p.Holder(){};}}`,
		})
		ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
		for _, method := range ctx.currentClass.Methods { if method.OriginalName == "make" { ctx.localScope = method } }
		node := findNode(ctx.currentClass.Class.DeclarationNode, "object_creation_expression")
		if node == nil { t.Fatal("missing anonymous source creation") }
		expr := anonymousSuperclassConstructorExpr(node, node.ChildByFieldName("type"), findQualifiedSourceClass("p.Holder"), nil, ctx.currentFile.Source, ctx)
		got := boundedArgumentTestGoType(t, expr)
		want := "p.NewHolder[*stdjava.List[*stdjava.JavaString]]"
		if !strings.Contains(got, want) {
			t.Errorf("Go raw anonymous constructor representation = %s, want %s", got, want)
		}
	})
}
