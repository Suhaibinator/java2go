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
	for _, source := range []string{
		`class Thread{int isInterrupted(){return 31;}}class Probe{int run(Thread t){return t.isInterrupted();}}`,
		`class Thread{static int interrupted(){return 37;}}class Probe{int run(){return Thread.interrupted();}}`,
		`class Thread{static int onSpinWait(){return 41;}}class Probe{int run(){return Thread.onSpinWait();}}`,
		`class Child extends java.lang.Thread{public boolean isInterrupted(){return true;}}class Probe{boolean run(Child t){return t.isInterrupted();}}`,
		`class Child extends java.lang.Thread{public boolean isInterrupted(){return true;}int isInterrupted(int x){return x;}}class Probe{int run(Child t){return t.isInterrupted(73);}}`,
		`import foreign.Thread;class Probe{int run(Thread t){return t.isInterrupted();}}`,
		`class Probe<Thread extends java.lang.Thread>{boolean run(Thread t){return t.isInterrupted();}}`,
		`class Probe{<Thread extends java.lang.Thread>boolean run(Thread t){return t.isInterrupted();}}`,
		`class Probe{boolean run(Thread[] t){return t.isInterrupted();}}`,
		`class Probe{boolean run(Thread[][] t){return t.interrupted();}}`,
		`class Probe{void run(Thread[] t){t.onSpinWait();}}`,
		`class Probe{boolean run(Thread t){return t.isInterrupted(7);}}`,
		`class Probe{boolean run(){return Thread.interrupted(7);}}`,
		`class Probe{void run(){Thread.onSpinWait(7);}}`,
		`class Probe{boolean run(){return Thread.isInterrupted();}}`,
		`import static java.lang.Thread.interrupted;class Probe{int interrupted(){return 43;}int run(){return interrupted();}}`,
		`import static java.lang.Thread.onSpinWait;class Probe{int onSpinWait(){return 47;}int run(){return onSpinWait();}}`,
		`import static java.lang.Thread.*;class Probe{int interrupted(){return 43;}int run(){return interrupted();}}`,
		`import static java.lang.Thread.interrupted;class Probe{boolean run(){return interrupted(7);}}`,
		`import static java.lang.Thread.onSpinWait;class Probe{void run(){onSpinWait(7);}}`,
	} {
		t.Run(source, func(t *testing.T) {
			helper := setupParseHelper(t, source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if result, known := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source); known {
				t.Errorf("noncanonical/invalid signature borrowed result metadata %q", result)
			}
			generated := renderGoFileFromJava(t, source)
			for _, name := range []string{"ThreadIsInterruptedExecution", "ThreadInterruptedExecution", "ThreadOnSpinWaitExecution"} {
				if strings.Contains(generated, "stdjava."+name+"(") {
					t.Errorf("noncanonical/invalid signature borrowed %s: %s", name, generated)
				}
			}
		})
	}
}
