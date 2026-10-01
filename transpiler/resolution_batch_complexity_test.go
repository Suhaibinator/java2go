package transpiler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

// File partitioning must not multiply whole-program hierarchy analysis. Both
// inputs below contain the same declaration and inheritance graph; only the
// Java compilation-unit boundaries differ. Allocation counts measure actual
// work without depending on machine speed or a wall-clock timeout.
func TestResolutionBatch_FilePartitionDoesNotMultiplyHierarchyWork(t *testing.T) {
	root := t.TempDir()
	joined, split := filepath.Join(root, "joined"), filepath.Join(root, "split")
	for _, dir := range []string{joined, split} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	declarations := []string{
		`class Value {}`,
		`class Base {
		    Value choose(Value value) { return value; }
		    int choose(int value) { return value; }
		    Class<?> descriptor() { return getClass(); }
		}`,
	}
	for index := 0; index < 24; index++ {
		declarations = append(declarations, fmt.Sprintf(`class Child%d extends Base {
		    Value choose(Value value) { return value; }
		    String choose(String value) { return value; }
		}`, index))
	}
	if err := os.WriteFile(filepath.Join(joined, "All.java"), []byte("package resolver.fixture;\n"+strings.Join(declarations, "\n")), 0644); err != nil {
		t.Fatal(err)
	}
	for index, declaration := range declarations {
		if err := os.WriteFile(filepath.Join(split, fmt.Sprintf("Unit%02d.java", index)), []byte("package resolver.fixture;\n"+declaration), 0644); err != nil {
			t.Fatal(err)
		}
	}
	previous := symbol.GlobalScope
	previousOutput := log.StandardLogger().Out
	log.SetOutput(io.Discard)
	t.Cleanup(func() { symbol.GlobalScope = previous; log.SetOutput(previousOutput) })
	measure := func(dir string) float64 {
		return testing.AllocsPerRun(1, func() {
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
			if err := runInternal([]string{"-strict", "-sync", "-q", dir}, io.Discard, false); err != nil {
				t.Fatal(err)
			}
		})
	}
	joinedWork := measure(joined)
	assertResolutionBatchFamilies(t)
	splitWork := measure(split)
	assertResolutionBatchFamilies(t)
	t.Logf("same 26-class graph: joined allocations=%.0f split allocations=%.0f ratio=%.2f", joinedWork, splitWork, splitWork/joinedWork)
	if splitWork > 5*joinedWork {
		t.Fatalf("compilation-unit boundaries multiplied hierarchy analysis: split %.0f > 5 * joined %.0f", splitWork, joinedWork)
	}
}

func assertResolutionBatchFamilies(t *testing.T) {
	t.Helper()
	var base, child *symbol.ClassScope
	pkg := symbol.GlobalScope.FindPackage("resolver.fixture")
	for _, file := range pkg.Files {
		for _, scope := range file.TopLevelClasses {
			switch scope.Class.OriginalName {
			case "Base":
				base = scope
			case "Child0":
				child = scope
			}
		}
	}
	if base == nil || child == nil {
		t.Fatal("resolution lost fixture class")
	}
	name := func(owner *symbol.ClassScope, parameter string) string {
		for _, method := range owner.Methods {
			if method.OriginalName == "choose" && len(method.Parameters) == 1 && method.Parameters[0].OriginalType == parameter {
				return method.Name
			}
		}
		t.Fatalf("missing %s.choose(%s)", owner.Class.OriginalName, parameter)
		return ""
	}
	if name(base, "Value") != name(child, "Value") {
		t.Fatal("genuine override was split into different selectors")
	}
	if name(base, "int") == name(child, "String") || name(base, "Value") == name(child, "String") {
		t.Fatal("child overload hides a distinct inherited method family")
	}
}
