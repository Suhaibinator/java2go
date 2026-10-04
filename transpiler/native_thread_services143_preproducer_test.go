package transpiler

import (
	"strings"
	"testing"
)

func nativeThreadServices143Contract(t *testing.T, sources []string, result, runtime string) {
	t.Helper()
	for _, source := range sources {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			actual, known := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source)
			if !known || actual != result {
				t.Errorf("canonical declaration result=%q/%t want=%s/true", actual, known, result)
			}
			generated := renderGoFileFromJava(t, source)
			if !strings.Contains(generated, "stdjava."+runtime+"(") {
				t.Errorf("canonical Execution service absent: %s", generated)
			}
		})
	}
}

func TestNativeThreadServices143PreProducerObserverContract(t *testing.T) {
	nativeThreadServices143Contract(t, []string{`class Probe{boolean run(java.lang.Thread t){return t.isInterrupted();}}`}, "boolean", "ThreadIsInterruptedExecution")
}
func TestNativeThreadServices143PreProducerClearingContract(t *testing.T) {
	nativeThreadServices143Contract(t, []string{`class Probe{boolean run(){return java.lang.Thread.interrupted();}}`, `class Probe{boolean run(java.lang.Thread t){return t.interrupted();}}`}, "boolean", "ThreadInterruptedExecution")
}
func TestNativeThreadServices143PreProducerHintContract(t *testing.T) {
	nativeThreadServices143Contract(t, []string{`class Probe{void run(){Thread.onSpinWait();}}`, `class Probe{void run(java.lang.Thread t){t.onSpinWait();}}`}, "void", "ThreadOnSpinWaitExecution")
}
func TestNativeThreadServices143PreProducerStaticImportContract(t *testing.T) {
	nativeThreadServices143Contract(t, []string{`import static java.lang.Thread.interrupted;class Probe{boolean run(){return interrupted();}}`, `import static java.lang.Thread.*;class Probe{boolean run(){return interrupted();}}`}, "boolean", "ThreadInterruptedExecution")
	nativeThreadServices143Contract(t, []string{`import static java.lang.Thread.onSpinWait;class Probe{void run(){onSpinWait();}}`, `import static java.lang.Thread.*;class Probe{void run(){onSpinWait();}}`}, "void", "ThreadOnSpinWaitExecution")
}
func TestNativeThreadServices143PreProducerNegativeShield(t *testing.T) {
	for _, test := range []struct {
		source        string
		strictRefusal bool
	}{
		{`class Thread{int isInterrupted(){return 31;}}class Probe{int run(Thread t){return t.isInterrupted();}}`, false},
		{`class Thread{static int interrupted(){return 37;}}class Probe{int run(){return Thread.interrupted();}}`, false},
		{`class Thread{static int onSpinWait(){return 41;}}class Probe{int run(){return Thread.onSpinWait();}}`, false},
		{`class Child extends java.lang.Thread{public boolean isInterrupted(){return true;}}class Probe{boolean run(Child t){return t.isInterrupted();}}`, false},
		{`class Child extends java.lang.Thread{public boolean isInterrupted(){return true;}int isInterrupted(int x){return x;}}class Probe{int run(Child t){return t.isInterrupted(73);}}`, false},
		{`import foreign.Thread;class Probe{int run(Thread t){return t.isInterrupted();}}`, false},
		{`class Probe<Thread extends java.lang.Thread>{boolean run(Thread t){return t.isInterrupted();}}`, true},
		{`class Probe{<Thread extends java.lang.Thread>boolean run(Thread t){return t.isInterrupted();}}`, true},
		{`class Probe{boolean run(Thread[] t){return t.isInterrupted();}}`, false},
		{`class Probe{boolean run(Thread[][] t){return t.interrupted();}}`, false},
		{`class Probe{void run(Thread[] t){t.onSpinWait();}}`, false},
		{`class Probe{boolean run(Thread t){return t.isInterrupted(7);}}`, true},
		{`class Probe{boolean run(){return Thread.interrupted(7);}}`, true},
		{`class Probe{void run(){Thread.onSpinWait(7);}}`, true},
		{`class Probe{boolean run(){return Thread.isInterrupted();}}`, true},
		{`import static java.lang.Thread.interrupted;class Probe{int interrupted(){return 43;}int run(){return interrupted();}}`, false},
		{`import static java.lang.Thread.onSpinWait;class Probe{int onSpinWait(){return 47;}int run(){return onSpinWait();}}`, false},
		{`import static java.lang.Thread.*;class Probe{int interrupted(){return 43;}int run(){return interrupted();}}`, false},
		{`import static java.lang.Thread.interrupted;class Probe{boolean run(){return interrupted(7);}}`, true},
		{`import static java.lang.Thread.onSpinWait;class Probe{void run(){onSpinWait(7);}}`, true},
	} {
		t.Run(test.source, func(t *testing.T) {
			for _, mode := range []struct {
				name   string
				strict bool
			}{{"permissive", false}, {"strict", true}} {
				t.Run(mode.name, func(t *testing.T) {
					strictRoutingState(t)
					setStrictMode(mode.strict)
					helper := setupParseHelper(t, test.source)
					ctx := helper.Ctx.Clone()
					ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
					ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
					invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
					if result, known := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source); known {
						t.Errorf("noncanonical/invalid signature borrowed result metadata %q", result)
					}
					want := Diagnostic{Kind: "intrinsic invocation", NodeType: "method_invocation", Message: invocation.Content(helper.File.Source), ClassName: ctx.currentClass.Class.Name, Line: invocation.StartPoint().Row + 1}
					var generated string
					var observed any
					func() { defer func() { observed = recover() }(); generated = renderGoFileFromJava(t, test.source) }()
					items := Diagnostics()
					if test.strictRefusal {
						if len(items) != 1 || items[0] != want {
							t.Fatalf("refusal diagnostic=%+v want exact %+v", items, want)
						}
						if mode.strict {
							failure, ok := observed.(strictModeError)
							if !ok || failure.diagnostic != want || generated != "" {
								t.Fatalf("strict refusal=%T/%v generated=%q", observed, observed, generated)
							}
							return
						}
						quote := "\"" + strings.ReplaceAll(want.String(), "\"", "\\\"") + "\""
						if !strings.Contains(generated, "panic("+quote+")") {
							t.Fatalf("permissive refusal placeholder absent: %s", generated)
						}
					} else if len(items) != 0 {
						t.Fatalf("ordinary/source/array-shield rendering gained diagnostic: %+v", items)
					}
					if observed != nil || generated == "" {
						t.Fatalf("non-refusal rendering panic=%T/%v generated=%q", observed, observed, generated)
					}
					for _, name := range []string{"ThreadIsInterruptedExecution", "ThreadInterruptedExecution", "ThreadOnSpinWaitExecution"} {
						if strings.Contains(generated, "stdjava."+name+"(") {
							t.Errorf("noncanonical/invalid signature borrowed %s: %s", name, generated)
						}
					}
				})
			}
		})
	}
}
