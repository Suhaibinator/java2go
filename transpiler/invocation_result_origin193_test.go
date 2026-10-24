package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
	"testing"
)

func stagedResult193Context(t *testing.T, producer, extra string) (Ctx, *sitter.Node, []byte) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, map[string]string{
		"p/Box.java":          producer,
		"app/Main.java":       `package app; import p.Box; public class Main {static <URI> Object read(Box box){return box.get();}}`,
		"app/java.java":       `package app; public class java {}`,
		"model/Priority.java": `package model; public class Priority {}`,
		"app/Extra.java":      `package app; ` + extra,
	}, "app/Main.java")
	ctx = classScopeCtx(ctx.currentClass, ctx)
	ctx.localScope = ctx.currentClass.FindMethodByName("read", nil)
	ctx.executionContextName = "__java2goExecution"
	var call *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || call != nil {
			return
		}
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "get" {
			call = n
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if call == nil {
		t.Fatal("missing actual parsed invocation")
	}
	return ctx, call, ctx.currentFile.Source
}

func TestInvocationClosureResult193ImportedNativeOrigin(t *testing.T) {
	ctx, n, s := stagedResult193Context(t, `package p; import java.net.URI; public class Box {public URI value; public URI get(){return value;}}`, `class Extra {}`)
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(n, ctx, s, &origin)
	if !known || origin.externalName != "java.net.URI" || origin.parameter != nil || origin.nominalScope != nil {
		t.Fatalf("actual imported result origin lost: %+v", origin)
	}
	r := helpfulNPEResolution(t, ctx, n, s)
	if helpfulSourceInvocationPlan(n, n.ChildByFieldName("object"), r, ctx, s) == nil {
		t.Fatal("actual registered source call not admitted")
	}
	out := helpfulNPEText(t, ParseExpr(n, s, ctx))
	if !strings.Contains(out, "func() *stdjava.JavaURI") {
		t.Fatalf("closure dropped declared native return origin into caller URI binder: %s", out)
	}
	if !strings.Contains(out, "RequireInvocationReceiver(") || !strings.Contains(out, "GetJava2goExecution(__java2goExecution)") {
		t.Fatalf("staging/check/execution changed: %s", out)
	}
}
func TestInvocationClosureResult193SourceOrigin(t *testing.T) {
	ctx, n, s := stagedResult193Context(t, `package p; import model.Priority; public class Box {public Priority get(){return null;}}`, `class Priority {}`)
	out := helpfulNPEText(t, ParseExpr(n, s, ctx))
	if !strings.Contains(out, "func() *model.Priority") {
		t.Fatalf("declared source return misrendered in caller namespace: %s", out)
	}
}
func TestInvocationClosureResult193SourceHomonym(t *testing.T) {
	ctx, n, s := stagedResult193Context(t, `package p; public class Box {public static class URI {} public URI get(){return null;}}`, `class URI {}`)
	var origin inferredJavaTypeOrigin
	if _, known := inferExprJavaType(n, ctx, s, &origin); !known || origin.nominalScope == nil || origin.externalName != "" {
		t.Fatalf("source result recaptured as native: %+v", origin)
	}
	out := helpfulNPEText(t, ParseExpr(n, s, ctx))
	if strings.Contains(out, "JavaURI") || strings.Contains(out, "func() URI") {
		t.Fatalf("source return recaptured by native/caller binder: %s", out)
	}
}
