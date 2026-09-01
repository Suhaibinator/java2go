package symbol_test

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func parseClassScope(t *testing.T, src, className string) *symbol.ClassScope {
	t.Helper()
	file := parsing.SourceFile{Name: className + ".java", Source: []byte(src)}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed to parse Java source: %v", err)
	}
	scope := file.ParseSymbols().FindClassScope(className)
	if scope == nil {
		t.Fatalf("class scope %q was not found", className)
	}
	return scope
}

func TestParseSymbols_DiscoversFinalClassAffineArrayAccessors(t *testing.T) {
	scope := parseClassScope(t, `
final class Matrix {
    private final double[] values;
    private final int size;

    Matrix(int size) {
        this.size = size;
        this.values = new double[size * size];
    }

    double read(int row, int column) {
        return this.values[row * this.size + column];
    }

    void write(int row, int column, double value) {
        this.values[row * this.size + column] = value;
    }

	final void add(int row, int column, double value) {
        this.values[row * this.size + column] =
            this.values[row * this.size + column] + value;
    }

    void addIndexed(int row, int column, double value) {
        int index = row * this.size + column;
        this.values[index] = this.values[index] + value;
    }
}
`, "Matrix")

	if !scope.Class.IsFinal {
		t.Fatal("final class modifier was not retained")
	}
	values := scope.FindFieldByName("values")
	size := scope.FindFieldByName("size")
	if values == nil || !values.IsPrivate || !values.IsFinal {
		t.Fatalf("values metadata = %#v, want private final field", values)
	}
	if size == nil || !size.IsPrivate || !size.IsFinal {
		t.Fatalf("size metadata = %#v, want private final field", size)
	}
	if len(scope.AffineArrayViews) != 1 {
		t.Fatalf("affine views = %d, want 1", len(scope.AffineArrayViews))
	}
	view := scope.AffineArrayViews[0]
	if view.ArrayField != values || view.SizeField != size {
		t.Fatalf("view fields = (%v, %v), want parsed values and size definitions", view.ArrayField, view.SizeField)
	}

	wants := map[string]symbol.TrivialArrayAccessorKind{
		"read":       symbol.TrivialArrayAccessorGet,
		"write":      symbol.TrivialArrayAccessorSet,
		"add":        symbol.TrivialArrayAccessorAdd,
		"addIndexed": symbol.TrivialArrayAccessorAdd,
	}
	for name, wantKind := range wants {
		method := scope.FindMethodByName(name, nil)
		if method == nil || method.DeclarationNode == nil {
			t.Fatalf("method %q did not retain its declaration", name)
		}
		if name == "add" && !method.IsFinal {
			t.Fatal("final method modifier was not retained")
		}
		accessor := method.TrivialArrayAccessor
		if accessor == nil {
			t.Fatalf("method %q was not recognized as a trivial accessor", name)
		}
		if accessor.Kind != wantKind || accessor.View != view {
			t.Fatalf("method %q accessor = %#v, want kind %d and shared view", name, accessor, wantKind)
		}
		if accessor.RowParameter != 0 || accessor.ColumnParameter != 1 {
			t.Fatalf("method %q coordinates = (%d, %d), want (0, 1)", name, accessor.RowParameter, accessor.ColumnParameter)
		}
		if wantKind == symbol.TrivialArrayAccessorGet {
			if accessor.ValueParameter != -1 {
				t.Fatalf("getter value parameter = %d, want -1", accessor.ValueParameter)
			}
		} else if accessor.ValueParameter != 2 {
			t.Fatalf("method %q value parameter = %d, want 2", name, accessor.ValueParameter)
		}
	}
}

func TestParseSymbols_AffineArrayAddPreservesOperandOrder(t *testing.T) {
	scope := parseClassScope(t, `
final class Matrix {
    private final double[] values = new double[4];
    private final int size = 2;
    void arrayFirst(int row, int column, double value) {
        this.values[row * this.size + column] = this.values[row * this.size + column] + value;
    }
    void valueFirst(int row, int column, double value) {
        this.values[row * this.size + column] = value + this.values[row * this.size + column];
    }
}
`, "Matrix")

	arrayFirst := scope.FindMethodByName("arrayFirst", nil).TrivialArrayAccessor
	valueFirst := scope.FindMethodByName("valueFirst", nil).TrivialArrayAccessor
	if arrayFirst == nil || valueFirst == nil {
		t.Fatal("both structurally trivial add accessors should be discovered")
	}
	if arrayFirst.ValueFirst {
		t.Fatal("array-first addition was recorded in the wrong order")
	}
	if !valueFirst.ValueFirst {
		t.Fatal("value-first addition was recorded in the wrong order")
	}
}

func TestParseSymbols_AffineArrayAccessorDiscoveryFallsBackUnlessProven(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "class can be subclassed",
			src: `class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "backing array can be reassigned",
			src: `final class Matrix {
                private double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "stride can be reassigned",
			src: `final class Matrix {
                private final double[] values = new double[4];
                private int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "backing array is visible",
			src: `final class Matrix {
                final double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "reference element",
			src: `final class Matrix {
                private final Object[] values = new Object[4];
                private final int size = 2;
                Object read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "body performs extra work",
			src: `final class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column] + 1.0; }
            }`,
		},
		{
			name: "index is not exact affine form",
			src: `final class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * (this.size + 1) + column]; }
            }`,
		},
		{
			name: "generic class",
			src: `final class Matrix<T> {
                private final int[] values = new int[4];
                private final int size = 2;
                int read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "synchronized accessor",
			src: `final class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                synchronized double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "strict floating point accessor",
			src: `final class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                strictfp double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
		{
			name: "strict floating point class",
			src: `strictfp final class Matrix {
                private final double[] values = new double[4];
                private final int size = 2;
                double read(int row, int column) { return this.values[row * this.size + column]; }
            }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scope := parseClassScope(t, test.src, "Matrix")
			if len(scope.AffineArrayViews) != 0 {
				t.Fatalf("discovered %d affine views; unsafe case must use normal method dispatch", len(scope.AffineArrayViews))
			}
			method := scope.FindMethodByName("read", nil)
			if method == nil {
				t.Fatal("read method was not parsed")
			}
			if method.TrivialArrayAccessor != nil {
				t.Fatalf("unsafe method was marked as trivial: %#v", method.TrivialArrayAccessor)
			}
		})
	}
}
