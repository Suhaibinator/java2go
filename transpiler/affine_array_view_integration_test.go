package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const affineArrayViewJavaSource = `
class Prelude {}

public final class Matrix {
    public int Java2goAffineView0Values;
    private final double[] values;
    private final int size;

    public Matrix(int size) {
        this.size = size;
        this.values = new double[size * size];
    }

    public int Java2goAffineView1Values() {
        return 17;
    }

    public double read(int row, int column) {
        return this.values[row * this.size + column];
    }
}
`

func TestResolveAffineArrayViewHelperNames_AfterDeepOrdinaryResolution(t *testing.T) {
	helper := setupParseHelper(t, `
class Prelude {}
class Outer {
    static class Middle {
        static final class DeepMatrix {
            private final double[] map = new double[4];
            private final int size = 2;

            public int Java2goAffineView0Map0() { return 1; }
            double read(int row, int column) {
                return this.map[row * this.size + column];
            }
        }
    }
}
`)
	deep := helper.File.Symbols.FindClassScope("DeepMatrix")
	if deep == nil {
		t.Fatal("deeply nested class scope was not parsed")
	}
	backing := deep.FindFieldByName("map")
	if backing == nil || backing.Name != "map0" {
		t.Fatalf("deep backing field resolved name = %#v, want map0", backing)
	}
	if len(deep.AffineArrayViews) != 1 {
		t.Fatalf("deep affine views = %d, want 1", len(deep.AffineArrayViews))
	}
	if got, want := deep.AffineArrayViews[0].HelperName, "Java2goAffineView1Map0"; got != want {
		t.Fatalf("deep helper name = %q, want post-resolution collision-free %q", got, want)
	}
}

func TestResolveAffineArrayViewHelperNames_CrossPackageProvenanceAndFutureRenames(t *testing.T) {
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previousGlobal })

	parse := func(name, source string) parsing.SourceFile {
		t.Helper()
		file := parsing.SourceFile{Name: name, Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		return file
	}

	parentFile := parse("Parent.java", `
package parent;
public interface Java2goAffineView0Values {
    default int Java2goAffineView0Values() { return 1; }
    default int Java2goAffineView0Values(int ignored) { return ignored; }
}
class Grandparent implements Java2goAffineView0Values {}
public class Parent extends Grandparent {}
`)
	childFile := parse("Matrix.java", `
package child;
import parent.Parent;
public final class Matrix extends Parent {
    private final double[] values0 = new double[4];
    private final double[] valuesJava2goDefaults = new double[4];
    private final int size = 2;

    double readFutureRename(int row, int column) {
        return this.values0[row * this.size + column];
    }
    double readDefaultCarrier(int row, int column) {
        return this.valuesJava2goDefaults[row * this.size + column];
    }
}
`)

	// Deliberately resolve the child first. Its naming pass must account for
	// ancestor overload names that the parent file will disambiguate later.
	ResolveFile(childFile)
	matrix := childFile.Symbols.FindClassScope("Matrix")
	if matrix == nil || len(matrix.AffineArrayViews) != 2 {
		t.Fatalf("Matrix affine views = %#v, want two", matrix)
	}
	helperByField := make(map[string]string)
	for _, view := range matrix.AffineArrayViews {
		helperByField[view.ArrayField.OriginalName] = view.HelperName
	}
	if got, want := helperByField["values0"], "Java2goAffineView1Values0"; got != want {
		t.Fatalf("future-overload-safe helper = %q, want %q", got, want)
	}
	if got, want := helperByField["valuesJava2goDefaults"], "Java2goAffineView1ValuesJava2goDefaults"; got != want {
		t.Fatalf("default-carrier-safe helper = %q, want %q", got, want)
	}

	ResolveFile(parentFile)
	iface := parentFile.Symbols.FindClassScope("Java2goAffineView0Values")
	if iface == nil {
		t.Fatal("parent interface scope was not parsed")
	}
	futureCollisionMaterialized := false
	for _, method := range iface.Methods {
		if method.Name == "Java2goAffineView0Values0" {
			futureCollisionMaterialized = true
		}
	}
	if !futureCollisionMaterialized {
		t.Fatalf("test precondition failed: parent overload names after resolution = %#v", iface.Methods)
	}
	if helperByField["values0"] == "Java2goAffineView0Values0" {
		t.Fatal("child helper collided with a parent overload renamed later")
	}
}

func TestResolveAffineArrayViewHelperNames_AllTopLevelClassesAndCollisions(t *testing.T) {
	helper := setupParseHelper(t, affineArrayViewJavaSource)
	matrix := helper.File.Symbols.FindClassScope("Matrix")
	if matrix == nil {
		t.Fatal("Matrix class scope was not parsed")
	}
	if matrix == helper.File.Symbols.BaseClass {
		t.Fatal("test precondition failed: Matrix must not be the legacy base class")
	}
	if len(matrix.AffineArrayViews) != 1 {
		t.Fatalf("affine views = %d, want 1", len(matrix.AffineArrayViews))
	}
	if got, want := matrix.AffineArrayViews[0].HelperName, "Java2goAffineView2Values"; got != want {
		t.Fatalf("helper name = %q, want collision-free %q", got, want)
	}
}

func TestGenerateAffineArrayViewHelper_IsNilSafeAndAliasesBackingSlice(t *testing.T) {
	out := renderGoFileFromJava(t, affineArrayViewJavaSource)
	flat := normalizeSpaces(out)
	for _, fragment := range []string{
		"func (mx *Matrix) Java2goAffineView2Values() ([]float64, int32)",
		"if mx == nil { return nil, 0 }",
		"return stdjava.PrimitiveArrayElements(mx.values), mx.size",
	} {
		if !strings.Contains(flat, fragment) {
			t.Fatalf("generated affine view helper is missing %q:\n%s", fragment, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestAffineViewContract(t *testing.T) {
    var nilMatrix *Matrix
    values, size := nilMatrix.Java2goAffineView2Values()
    if values != nil || size != 0 {
        t.Fatalf("nil receiver view = (%v, %d), want (nil, 0)", values, size)
    }

    matrix := NewMatrix(2)
    values, size = matrix.Java2goAffineView2Values()
    if len(values) != 4 || size != 2 {
        t.Fatalf("view = (len %d, size %d), want (4, 2)", len(values), size)
    }
    values[3] = 7.5
    if got := matrix.Read(1, 1); got != 7.5 {
        t.Fatalf("view does not alias backing storage: Read(1, 1) = %v", got)
    }
}
`)
}
