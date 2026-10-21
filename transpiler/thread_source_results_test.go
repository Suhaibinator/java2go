package transpiler

import (
	"strings"
	"testing"
)

func TestThreadSourceInheritedResultEmitterAgreement(t *testing.T) {
	for _, test := range []struct{ call, typ, emitter string }{
		{"value.isInterrupted()", "boolean", "ThreadIsInterruptedExecution("},
		{"value.isAlive()", "boolean", "ReferenceRequireNonNull("},
		{"value.getName()", "java.lang.String", "GetNameReference("},
		{"value.currentThread()", "java.lang.Thread", "ThreadCurrentThread("},
		{"value.interrupted()", "boolean", "ThreadInterruptedExecution("},
	} {
		t.Run(test.call, func(t *testing.T) {
			source := `class Child extends java.lang.Thread {} class Probe {Child value;void read(){` + test.call + `;}}`
			h := setupParseHelper(t, source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("read", nil)
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			var origin inferredJavaTypeOrigin
			typ, ok := threadSourceInheritedResultType(node, ctx, h.File.Source, &origin)
			if !ok || typ != test.typ {
				t.Fatalf("result%q/%t want%s", typ, ok, test.typ)
			}
			if strings.HasPrefix(typ, "java.lang.") && origin.externalName != typ {
				t.Fatal("canonical origin missing")
			}
			out := renderGoFileFromJava(t, source)
			if !strings.Contains(out, test.emitter) {
				t.Fatal("emitter disagrees", out)
			}
		})
	}
	for _, source := range []string{
		`class Child extends java.lang.Thread {public boolean isInterrupted(){return false;}}class Probe{Child value;void read(){value.isInterrupted();}}`,
		`class Thread {boolean isInterrupted(){return false;}}class Child extends Thread {}class Probe{Child value;void read(){value.isInterrupted();}}`,
		`import foreign.Thread;class Child extends Thread {}class Probe{Child value;void read(){value.isInterrupted();}}`,
		`class Child extends java.lang.Thread {}class Probe<Child extends java.lang.Thread>{Child value;void read(){value.isInterrupted();}}`,
		`class Child extends java.lang.Thread {}class Probe{Child[] value;void read(){value.isInterrupted();}}`,
		`class Child extends java.lang.Thread {}class Probe{Child value;void read(){value.isInterrupted(7);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			h := setupParseHelper(t, source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("read", nil)
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if typ, ok := threadSourceInheritedResultType(node, ctx, h.File.Source); ok {
				t.Fatalf("shield admitted %q", typ)
			}
		})
	}
}
