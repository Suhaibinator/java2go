package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestThreadSourceRuntimeAncestryInvocation(t *testing.T) {
	for _, test := range []struct {
		source, expected string
		want             bool
	}{
		{`class Child extends java.lang.Thread {} class Probe {}`, "java.lang.Runnable", true},
		{`class Base extends java.lang.Thread {} class Child extends Base {} class Probe {}`, "java.lang.Runnable", true},
		{`class Child extends java.lang.Thread {} class Probe {}`, "java.lang.Thread", true},
		{`class Thread {} class Child extends Thread {} class Probe {}`, "java.lang.Runnable", false},
		{`import foreign.Thread; class Child extends Thread {} class Probe {}`, "java.lang.Runnable", false},
		{`class Child<Thread> extends Thread {} class Probe {}`, "java.lang.Runnable", false},
		{`class Child extends java.lang.Thread {} class Runnable {} class Probe {}`, "Runnable", false},
		{`class Child extends java.lang.Thread {} class Probe<Runnable> {}`, "Runnable", false},
		{`class Child extends java.lang.Thread {} class Probe<java> {}`, "java.lang.Runnable", false},
		{`class Child extends java.lang.Thread {} class Probe {}`, "java.lang.Runnable[]", false},
		{`class Child extends java.lang.Thread {} class Probe {}`, "java.lang.Runnable<String>", false},
	} {
		t.Run(test.source+test.expected, func(t *testing.T) {
			h := setupParseHelper(t, test.source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			child := resolveClassScopeByQualifiedName(ctx, "Child")
			if got := sourceRuntimeReferenceAssignable(child, nil, test.expected, ctx); got != test.want {
				t.Fatalf("declaration ancestry=%v want%v", got, test.want)
			}
		})
	}
}

func TestThreadNativeRunnableDeclaredAncestry(t *testing.T) {
	h := setupParseHelper(t, `class Probe{java.lang.Thread actual;java.lang.Runnable expected;}`)
	actual := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.Thread"}, h.Ctx)
	formal := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.Runnable"}, h.Ctx)
	_, _, applies, handled := declaredOriginConversion(actual, formal, h.Ctx)
	if !handled || !applies {
		t.Errorf("declared native Thread->Runnable=%v/%v", applies, handled)
	}
	if !builtinJavaReferenceAssignable("java.lang.Thread", "java.lang.Runnable", h.Ctx) {
		t.Error("native Thread->Runnable shared builtin false")
	}
}
