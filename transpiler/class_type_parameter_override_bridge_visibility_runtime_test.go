package transpiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func overrideBridgeVisibilityProject(
	t *testing.T,
	ancestorModifier string,
	overrideAnnotation string,
	nextClass string,
) map[string]string {
	t.Helper()
	root := t.TempDir()
	baseDir := filepath.Join(root, "audit", "a")
	childDir := filepath.Join(root, "audit", "b")
	for _, directory := range []string{baseDir, childDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create source directory %s: %v", directory, err)
		}
	}

	baseSource := fmt.Sprintf(`
package audit.a;

public class Base<T extends Numbered> {
    public static int baseBodies;
    T value;

    public Base(T value) { this.value = value; }

    %s T exchange(T next) {
        baseBodies++;
        T previous = value;
        value = next;
        return previous;
    }

    @SuppressWarnings({"rawtypes", "unchecked"})
    public static Numbered invokeRaw(Base receiver, Numbered next) {
        return receiver.exchange(next);
    }
}
`, ancestorModifier)
	childSource := fmt.Sprintf(`
package audit.b;

import audit.a.Base;
import audit.a.First;
import audit.a.Numbered;
import audit.a.Second;

public final class Child extends Base<First> {
    public static int childBodies;

    public Child(First value) { super(value); }

    %s
    public First exchange(First next) {
        childBodies++;
        return next;
    }

    public static String run() {
        Base.baseBodies = 0;
        childBodies = 0;
        Child child = new Child(new First(1));
        Numbered result = Base.invokeRaw(child, new %s(2));
        return result.number() + ":" + Base.baseBodies + ":" + childBodies;
    }
}
`, overrideAnnotation, nextClass)
	for path, source := range map[string]string{
		filepath.Join(baseDir, "Numbered.java"): `
package audit.a;
public interface Numbered { int number(); }
`,
		filepath.Join(baseDir, "First.java"): `
package audit.a;
public final class First implements Numbered {
    private final int value;
    public First(int value) { this.value = value; }
    public int number() { return value; }
}
`,
		filepath.Join(baseDir, "Second.java"): `
package audit.a;
public final class Second implements Numbered {
    private final int value;
    public Second(int value) { this.value = value; }
    public int number() { return value; }
}
`,
		filepath.Join(baseDir, "Base.java"):   baseSource,
		filepath.Join(childDir, "Child.java"): childSource,
	} {
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return convertJavaProjectDir(t, root)
}

func runOverrideBridgeVisibilityProject(t *testing.T, outputs map[string]string, want string) {
	t.Helper()
	moduleDir := t.TempDir()
	goMod := "module audit\n\ngo 1.26.0\n\n" +
		"require github.com/NickyBoy89/java2go v0.0.0\n\n" +
		"replace github.com/NickyBoy89/java2go => " + repoRoot(t) + "\n"
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600); err != nil {
		t.Fatalf("write generated go.mod: %v", err)
	}
	for relative, source := range outputs {
		relative = strings.TrimPrefix(filepath.ToSlash(relative), "audit/")
		path := filepath.Join(moduleDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create generated directory for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			t.Fatalf("write generated source %s: %v", path, err)
		}
	}
	testPath := filepath.Join(moduleDir, "b", "visibility_behavior_test.go")
	testSource := fmt.Sprintf(`
package b

import "testing"

func TestVisibilityBehavior(t *testing.T) {
    if got := Run(); got != %q {
        t.Fatalf("Run() = %%q, want %%q", got, %q)
    }
}
`, want, want)
	if err := os.WriteFile(testPath, []byte(testSource), 0o600); err != nil {
		t.Fatalf("write generated behavior test: %v", err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = moduleDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("generated go mod tidy failed:\n%s", out)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = moduleDir
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated cross-package behavior failed:\n%s", out)
	}
}

// A same-signature declaration in another package does not override a
// package-private ancestor method. The Base invokevirtual must continue to
// enter Base even though Child has a public source method with the same shape.
func TestDirectOwnerOverrideBridgeVisibility_PackagePrivateCrossPackageIsNotOverride(t *testing.T) {
	outputs := overrideBridgeVisibilityProject(t, "", "", "Second")
	runOverrideBridgeVisibilityProject(t, outputs, "1:1:0")

	child := outputs["audit/b/Child.go"]
	if strings.Contains(child, "Java2goExactExecution") {
		t.Fatalf("package-private non-override received a synthetic exact/bridge split:\n%s", child)
	}
}

// Public methods are inherited across packages. This is the supported
// cross-package bridge path: Base dispatch must enter Child's cast bridge and
// then its exact override body.
func TestDirectOwnerOverrideBridgeVisibility_PublicCrossPackageDispatchesOverride(t *testing.T) {
	outputs := overrideBridgeVisibilityProject(t, "public", "@Override", "First")
	runOverrideBridgeVisibilityProject(t, outputs, "2:0:1")

	child := outputs["audit/b/Child.go"]
	if !strings.Contains(child, "Java2goExactExecution") {
		t.Fatalf("public cross-package override omitted its exact bridge body:\n%s", child)
	}
}

// A protected override is Java-legal across packages, but this bridge slice
// deliberately keeps it on the established ABI until hidden family selectors
// are exported. Silently planning it would emit two package-distinct lowercase
// Go methods and route Base dispatch to the wrong body.
func TestDirectOwnerOverrideBridgeVisibility_ProtectedCrossPackageIsConservativelyUnplanned(t *testing.T) {
	outputs := overrideBridgeVisibilityProject(t, "protected", "@Override", "First")
	basePackage := symbol.GlobalScope.FindPackage("audit.a")
	if basePackage == nil {
		t.Fatal("missing audit.a package")
	}
	base := basePackage.FindClassScope("Base")
	method := overrideBridgeTestMethod(t, base, "exchange")
	if plan, ok := planDirectOwnerCallableOverrideBridgeFamily(base, method, classScopeCtx(base, Ctx{})); ok || plan != nil {
		t.Fatalf("protected cross-package override received unsupported bridge plan: %#v", plan)
	}

	if child := outputs["audit/b/Child.go"]; strings.Contains(child, "Java2goExactExecution") {
		t.Fatalf("protected cross-package override silently emitted a broken bridge:\n%s", child)
	}
}
