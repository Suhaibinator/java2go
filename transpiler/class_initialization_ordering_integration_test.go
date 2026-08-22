package transpiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func assertGeneratedClassInitializationResult(t *testing.T, source, want string) {
	t.Helper()
	out := renderGoFileFromJava(t, source)
	runGeneratedWithStdjava(t, out, fmt.Sprintf(`
package main

import "testing"

func TestClassInitializationResult(t *testing.T) {
	if got := Run(); got != %q {
		t.Fatalf("Run() = %%q, want %%q", got, %q)
	}
}
`, want, want))
}

func TestClassInitialization_DormantClassStaysUninitialized(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class DormantInitializationTarget {
    static int ignored = ClassInitializationProgram.mark("D");
}

public class ClassInitializationProgram {
    static String trace = "";
    static int initialized = mark("M");

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        return trace + ":" + initialized;
    }
}
`, "M:1")
}

func TestClassInitialization_StaticMethodRunsAfterQualifierAndArguments(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticMethodInitializationTarget {
    static int initialized = ClassInitializationProgram.mark("I");

    static void call(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static StaticMethodInitializationTarget qualifier() {
        mark("Q");
        return null;
    }

    public static String run() {
        trace = "";
        qualifier().call(mark("A"));
        return trace;
    }
}
`, "QAIC")
}

func TestClassInitialization_DirectNewRunsBeforeArguments(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class ConstructorInitializationTarget {
    static int initialized = ClassInitializationProgram.mark("I");

    ConstructorInitializationTarget(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        trace = "";
        new ConstructorInitializationTarget(mark("A"));
        return trace;
    }
}
`, "IAC")
}

func TestClassInitialization_StaticFieldReadRunsAfterQualifier(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticReadInitializationTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value = 7;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static StaticReadInitializationTarget qualifier() {
        mark("Q");
        return null;
    }

    public static String run() {
        trace = "";
        int value = qualifier().value;
        return trace + ":" + value;
    }
}
`, "QI:7")
}

func TestClassInitialization_StaticFieldWriteRunsAfterQualifierAndRHS(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticWriteInitializationTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static StaticWriteInitializationTarget qualifier() {
        mark("Q");
        return null;
    }

    public static String run() {
        trace = "";
        qualifier().value = mark("R");
        return trace + ":" + StaticWriteInitializationTarget.value;
    }
}
`, "QRI:2")
}

func TestClassInitialization_ThrowingStaticMethodArgumentLeavesTargetDormant(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticMethodAbruptTarget {
    static int initialized = ClassInitializationProgram.mark("I");

    static void call(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static int fail(String marker) {
        mark(marker);
        throw new RuntimeException(marker);
    }

    public static String run() {
        trace = "";
        try {
            StaticMethodAbruptTarget.call(fail("X"));
        } catch (RuntimeException expected) {
        }
        String before = trace;
        StaticMethodAbruptTarget.call(0);
        return before + ":" + trace;
    }
}
`, "X:XIC")
}

func TestClassInitialization_ThrowingDirectNewArgumentLeavesTargetInitialized(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class ConstructorAbruptTarget {
    static int initialized = ClassInitializationProgram.mark("I");

    ConstructorAbruptTarget(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static int fail(String marker) {
        mark(marker);
        throw new RuntimeException(marker);
    }

    public static String run() {
        trace = "";
        try {
            new ConstructorAbruptTarget(fail("X"));
        } catch (RuntimeException expected) {
        }
        String before = trace;
        new ConstructorAbruptTarget(0);
        return before + ":" + trace;
    }
}
`, "IX:IXC")
}

func TestClassInitialization_ThrowingStaticFieldQualifierLeavesTargetDormant(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticReadAbruptTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value = 7;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static StaticReadAbruptTarget failQualifier() {
        mark("Q");
        throw new RuntimeException("Q");
    }

    public static String run() {
        trace = "";
        try {
            int ignored = failQualifier().value;
        } catch (RuntimeException expected) {
        }
        String before = trace;
        int value = StaticReadAbruptTarget.value;
        return before + ":" + trace + ":" + value;
    }
}
`, "Q:QI:7")
}

func TestClassInitialization_ThrowingStaticFieldRHSLeavesTargetDormant(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class StaticWriteAbruptTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static int fail(String marker) {
        mark(marker);
        throw new RuntimeException(marker);
    }

    public static String run() {
        trace = "";
        try {
            StaticWriteAbruptTarget.value = fail("R");
        } catch (RuntimeException expected) {
        }
        String before = trace;
        int value = StaticWriteAbruptTarget.value;
        return before + ":" + trace + ":" + value;
    }
}
`, "R:RI:0")
}

func TestClassInitialization_InheritedStaticFieldInitializesDeclaringClassOnly(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class InheritedStaticFieldBase {
    static int value = ClassInitializationProgram.mark("B");
}

class InheritedStaticFieldSub extends InheritedStaticFieldBase {
    static int initialized = ClassInitializationProgram.mark("S");
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        trace = "";
        int value = InheritedStaticFieldSub.value;
        return trace + ":" + value;
    }
}
`, "B:1")
}

func TestClassInitialization_CompoundStaticFieldAssignmentInitializesBeforeRHS(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class CompoundStaticFieldTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value = 10;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static CompoundStaticFieldTarget qualifier() {
        mark("Q");
        return null;
    }

    public static String run() {
        trace = "";
        qualifier().value += mark("R");
        return trace + ":" + CompoundStaticFieldTarget.value;
    }
}
`, "QIR:13")
}

func TestClassInitialization_ThrowingCompoundStaticFieldRHSLeavesOldValue(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class CompoundStaticFieldAbruptTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value = 10;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static int fail(String marker) {
        mark(marker);
        throw new RuntimeException(marker);
    }

    static CompoundStaticFieldAbruptTarget qualifier() {
        mark("Q");
        return null;
    }

    public static String run() {
        trace = "";
        try {
            qualifier().value += fail("X");
        } catch (RuntimeException expected) {
        }
        return trace + ":" + CompoundStaticFieldAbruptTarget.value;
    }
}
`, "QIX:10")
}

func TestClassInitialization_PreAndPostStaticFieldUpdatesInitializeBeforeMutation(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class PostUpdateStaticFieldTarget {
    static int initialized = ClassInitializationProgram.mark("I");
    static int value = 10;
}

class PreUpdateStaticFieldTarget {
    static int initialized = ClassInitializationProgram.mark("J");
    static int value = 20;
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    static PostUpdateStaticFieldTarget postQualifier() {
        mark("Q");
        return null;
    }

    static PreUpdateStaticFieldTarget preQualifier() {
        mark("P");
        return null;
    }

    public static String run() {
        trace = "";
        int post = postQualifier().value++;
        int pre = ++preQualifier().value;
        return trace + ":" + post + ":" + pre + ":"
            + PostUpdateStaticFieldTarget.value + ":" + PreUpdateStaticFieldTarget.value;
    }
}
`, "QIPJ:10:21:11:21")
}

func TestClassInitialization_CompileTimeConstantsDoNotInitializeDeclaringClass(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class ConstantInitializationTarget {
    static final int BASE = 3;
    static final int NUMBER = BASE * 2 + 1;
    static final String TEXT = "rea" + "dy";
    static int initialized = ClassInitializationProgram.mark("I");
}

class ConstantInitializationSub extends ConstantInitializationTarget {
    static int initialized = ClassInitializationProgram.mark("S");
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        trace = "";
        int number = ConstantInitializationSub.NUMBER;
        String text = ConstantInitializationSub.TEXT;
        String before = trace;
        int initialized = ConstantInitializationTarget.initialized;
        return before + ":" + trace + ":" + number + ":" + text + ":" + initialized;
    }
}
`, ":I:7:ready:1")
}

func TestClassInitialization_SuperclassPrecedesSubclassForNewAndStaticMethod(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class NewInitializationBase {
    static int initialized = ClassInitializationProgram.mark("B");
}

class NewInitializationSub extends NewInitializationBase {
    static int initialized = ClassInitializationProgram.mark("S");

    NewInitializationSub(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

class MethodInitializationBase {
    static int initialized = ClassInitializationProgram.mark("B");
}

class MethodInitializationSub extends MethodInitializationBase {
    static int initialized = ClassInitializationProgram.mark("S");

    static void call(int ignored) {
        ClassInitializationProgram.mark("C");
    }
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        trace = "";
        new NewInitializationSub(mark("A"));
        String newOrder = trace;

        trace = "";
        MethodInitializationSub.call(mark("A"));
        return newOrder + ":" + trace;
    }
}
`, "BSAC:ABSC")
}

func TestClassInitialization_EmptySubclassRetainsItsOwnErroneousState(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class FailingInitializationBase {
    static int initialized = fail();

    static int fail() {
        throw new RuntimeException("boom");
    }
}

class EmptyInitializationSub extends FailingInitializationBase {
}

public class ClassInitializationProgram {
    public static String run() {
        String first = "none";
        try {
            new EmptyInitializationSub();
        } catch (ExceptionInInitializerError expected) {
            first = "ExceptionInInitializerError";
        }

        String sub = "none";
        try {
            new EmptyInitializationSub();
        } catch (NoClassDefFoundError expected) {
            sub = expected.getMessage();
        }

        String base = "none";
        try {
            new FailingInitializationBase();
        } catch (NoClassDefFoundError expected) {
            base = expected.getMessage();
        }
        return first + "|" + sub + "|" + base;
    }
}
`, "ExceptionInInitializerError|Could not initialize class EmptyInitializationSub|Could not initialize class FailingInitializationBase")
}

func TestClassInitialization_CrossClassConstantInitializerDoesNotTriggerEitherClass(t *testing.T) {
	assertGeneratedClassInitializationResult(t, `
class B {
    static final int Y = 7;
    static int initialized = ClassInitializationProgram.mark("B");
}

class A {
    static final int X = B.Y;
    static int initialized = ClassInitializationProgram.mark("A");
}

public class ClassInitializationProgram {
    static String trace = "";

    static int mark(String marker) {
        trace = trace + marker;
        return trace.length();
    }

    public static String run() {
        trace = "";
        int value = A.X;
        String before = trace;
        int initializedA = A.initialized;
        String afterA = trace;
        int initializedB = B.initialized;
        return before + ":" + afterA + ":" + trace + ":"
            + value + ":" + initializedA + ":" + initializedB;
    }
}
`, ":A:AB:7:1:2")
}

func TestClassInitialization_CrossPackageEmptyParentFindsInitializedGrandparent(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "classinit/trace/Trace.java", `
package classinit.trace;

public final class Trace {
    public static String value = "";

    public static int mark(String marker) {
        value = value + marker;
        return value.length();
    }

    public static void reset() {
        value = "";
    }
}
`)
	writeJavaTestSource(t, sourceRoot, "classinit/grand/Grandparent.java", `
package classinit.grand;

import classinit.trace.Trace;

public class Grandparent {
    static int initialized = Trace.mark("G");
}
`)
	writeJavaTestSource(t, sourceRoot, "classinit/parent/Parent.java", `
package classinit.parent;

import classinit.grand.Grandparent;

public class Parent extends Grandparent {
}
`)
	writeJavaTestSource(t, sourceRoot, "classinit/app/Application.java", `
package classinit.app;

import classinit.parent.Parent;
import classinit.trace.Trace;

public class Application {
    public static String run() {
        Trace.reset();
        new Parent();
        return Trace.value;
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	moduleRoot := t.TempDir()
	goMod := "module classinit\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write generated module go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "classinit/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated package directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated source: %v", err)
		}
	}

	testPath := filepath.Join(moduleRoot, "app", "class_initialization_test.go")
	if err := os.WriteFile(testPath, []byte(`package app

import "testing"

func TestCrossPackageClassInitialization(t *testing.T) {
    if got := Run(); got != "G" {
        t.Fatalf("Run() = %q, want %q", got, "G")
    }
}
`), 0o644); err != nil {
		t.Fatalf("write generated runtime test: %v", err)
	}

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = moduleRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cross-package class initialization failed:\n%s", output)
	}
}
