package transpiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolymorphismRuntime_InterfaceDefaultMethodUsesConcreteReceiver(t *testing.T) {
	src := `
interface DefaultGreeter {
    String greet();
    default String decorated() { return "[" + greet() + "]"; }
}
class DefaultBase implements DefaultGreeter {
    public String greet() { return "base"; }
}
class DefaultChild extends DefaultBase {
    public String greet() { return "child"; }
}
public class DefaultDispatchProgram {
    public static String run() {
        DefaultGreeter[] values = new DefaultGreeter[] {
            new DefaultBase(), new DefaultChild()
        };
        return values[0].decorated() + "|" + values[1].decorated();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestInterfaceDefaultDispatch(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("[base]|[child]"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "[base]|[child]")
	}
}
`)
}

func TestPolymorphismRuntime_BaseReferencesDispatchOverrides(t *testing.T) {
	src := `
class PolyBase {
    String value() { return "base"; }
    String render() { return "[" + value() + "]"; }
}
class PolyChild extends PolyBase {
    String value() { return "child"; }
}
class PolyGrandchild extends PolyChild {
    String render() { return "grand:" + super.render(); }
}
public class VirtualDispatchProgram {
    public static String run() {
        PolyBase[] values = new PolyBase[] {
            new PolyBase(), new PolyChild(), new PolyGrandchild()
        };
        return values[0].render() + "|" + values[1].render() + "|" + values[2].render();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestClassVirtualDispatch(t *testing.T) {
	const want = "[base]|[child]|grand:[child]"
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, want)
	}
}
`)
}

func TestPolymorphismRuntime_AnonymousAbstractSubclassDispatch(t *testing.T) {
	src := `
abstract class AnonymousBase {
    abstract String value();
    String render() { return value() + "!"; }
}
public class AnonymousDispatchProgram {
    public static String run() {
        AnonymousBase subject = new AnonymousBase() {
            String value() { return "anonymous"; }
        };
        return subject.value() + "|" + subject.render();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestAnonymousAbstractDispatch(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("anonymous|anonymous!"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "anonymous|anonymous!")
	}
}
`)
}

func TestPolymorphismRuntime_ConstructorCallsMostDerivedOverride(t *testing.T) {
	src := `
class ConstructorBase {
    String observed;
    ConstructorBase() { observed = label(); }
    String label() { return "base"; }
    String result() { return observed; }
}
class ConstructorMiddle extends ConstructorBase {
    ConstructorMiddle() { super(); }
}
class ConstructorLeaf extends ConstructorMiddle {
    ConstructorLeaf() { super(); }
    String label() { return "leaf"; }
}
public class ConstructorDispatchProgram {
    public static String run() {
        return new ConstructorLeaf().result();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestConstructorVirtualDispatch(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("leaf"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "leaf")
	}
}
`)
}

func TestPolymorphismRuntime_InstanceFieldInitializerCallsMostDerivedOverride(t *testing.T) {
	src := `
class InitializerDispatchBase {
    int observed = read();
    InitializerDispatchBase() {}
    int read() { return -1; }
}
class InitializerDispatchChild extends InitializerDispatchBase {
    int ready = 41;
    InitializerDispatchChild() { super(); }
    int read() { return ready; }
}
public class InitializerDispatchProgram {
    public static String run() {
        InitializerDispatchChild value = new InitializerDispatchChild();
        return value.observed + ":" + value.ready;
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestInstanceFieldInitializerDispatch(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("0:41"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "0:41")
	}
}
`)
}

func TestPolymorphismRuntime_InheritedInterfaceDefaultIsInitialized(t *testing.T) {
	src := `
interface ParentDefault {
    default String greet() { return "inherited"; }
}
interface ChildDefault extends ParentDefault {}
class InheritedDefaultImpl implements ChildDefault {}
public class InheritedDefaultProgram {
	static String invoke(ParentDefault value) { return value.greet(); }
    public static String run() {
		return invoke(new InheritedDefaultImpl());
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestInheritedInterfaceDefault(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("inherited"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "inherited")
	}
}
`)
}

func TestPolymorphismRuntime_ConcreteClassSelectsInheritedInterfaceDefault(t *testing.T) {
	src := `
interface ConcreteDefaultGreeting {
    String label();
    default String greet(int repeat) {
        return label() + ":" + repeat;
    }
}
class ConcreteDefaultGreeter implements ConcreteDefaultGreeting {
    public String label() { return "concrete"; }
}
public class ConcreteDefaultProgram {
    public static String run() {
        ConcreteDefaultGreeter value = new ConcreteDefaultGreeter();
        return value.greet(3);
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestConcreteDefaultSelection(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("concrete:3"))) {
		t.Fatalf("Run() UTF16 = %x", units)
	}
}
`)
}

func TestPolymorphismRuntime_PrivateBaseMethodIsNotOverridden(t *testing.T) {
	src := `
class PrivateDispatchBase {
    private String label() { return "base"; }
    String callLabel() { return label(); }
}
class PrivateDispatchChild extends PrivateDispatchBase {
    String label() { return "child"; }
}
public class PrivateDispatchProgram {
    public static String run() {
        return new PrivateDispatchChild().callLabel();
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestPrivateMethodBinding(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("base"))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, "base")
	}
}
`)
}

func TestPolymorphismRuntime_CrossPackageDispatchUsesExportedSlot(t *testing.T) {
	sourceRoot := t.TempDir()
	writeJavaTestSource(t, sourceRoot, "cross/base/Base.java", `
package cross.base;
public class Base {
    public String value() { return "base"; }
    public String render() { return "[" + value() + "]"; }
}
`)
	writeJavaTestSource(t, sourceRoot, "cross/child/Child.java", `
package cross.child;
import cross.base.Base;
public class Child extends Base {
    public String value() { return "child"; }
}
`)
	writeJavaTestSource(t, sourceRoot, "cross/app/App.java", `
package cross.app;
import cross.base.Base;
import cross.child.Child;
public class App {
    public static String run() {
        Base value = new Child();
        return value.render();
    }
}
`)

	outputs := convertJavaProjectDir(t, sourceRoot)
	moduleRoot := t.TempDir()
	goMod := "module cross\n\ngo 1.27.0\n\nrequire github.com/NickyBoy89/java2go v0.0.0\n\nreplace github.com/NickyBoy89/java2go => " + repoRootDir(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleRoot, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for relative, generated := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "cross/")
		path := filepath.Join(moduleRoot, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated package: %v", err)
		}
		if err := os.WriteFile(path, []byte(generated), 0o644); err != nil {
			t.Fatalf("write generated source: %v", err)
		}
	}
	testPath := filepath.Join(moduleRoot, "app", "dispatch_test.go")
	if err := os.WriteFile(testPath, []byte(`package app

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestCrossPackageDispatch(t *testing.T) {
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune("[child]"))) {
		t.Fatalf("Run() UTF16 = %x", units)
	}
}
`), 0o644); err != nil {
		t.Fatalf("write generated runtime test: %v", err)
	}
	cmd := exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = moduleRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cross-package generated code failed:\n%s", output)
	}
}

func TestPolymorphismRuntime_BaseArrayPreservesSubtypeIdentity(t *testing.T) {
	src := `
class IdentityBase {
    String value() { return "base"; }
}
class IdentityChild extends IdentityBase {
    String value() { return "child"; }
}
public class IdentityProgram {
    static boolean isChild(IdentityBase value) {
        return value instanceof IdentityChild;
    }
    static String classify(IdentityBase value) {
        if (value instanceof IdentityChild child) {
            return child.value();
        }
        return value.value();
    }
    public static String run() {
        IdentityBase[] values = new IdentityBase[] {
            new IdentityChild(), new IdentityBase()
        };
        return isChild(values[0]) + ":" + classify(values[0]) + "|"
            + isChild(values[1]) + ":" + classify(values[1]);
    }
}
`

	out := renderGoFileFromJava(t, src)
	runGoTestInTempModule(t, out, `
package main

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestBaseArrayIdentity(t *testing.T) {
	const want = "true:child|false:base"
	got := Run()
	if got == nil {
		t.Fatal("Run() returned null")
	}
	if units := got.UTF16Copy(); !slices.Equal(units, utf16.Encode([]rune(want))) {
		t.Fatalf("Run() UTF16 = %x, want %q", units, want)
	}
}
`)
}
