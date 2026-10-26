package transpiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

type syntaxBounds214Fixture struct {
	scope  *symbol.FileScope
	root   *sitter.Node
	tree   *sitter.Tree
	source []byte
}

func syntaxBounds214Parse(t *testing.T, source string) syntaxBounds214Fixture {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	data := []byte(source)
	tree, err := parser.ParseCtx(context.Background(), nil, data)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close(); parser.Close() })
	root := tree.RootNode()
	return syntaxBounds214Fixture{symbol.ParseSymbols(root, data), root, tree, data}
}

func syntaxBounds214Equal(t *testing.T, file *symbol.FileScope) *syntaxIdentifierFile {
	t.Helper()
	want := syntaxBounds214ReferenceBuild(file)
	got := buildSyntaxIdentifierFile(file)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("identifier index differs: anchor, copied bytes, bodies, ordinals, raw/method/prefix/declaration maps")
	}
	return got
}

func TestSyntaxIdentifierBounds214IndexEquality(t *testing.T) {
	sources := []string{
		"package bounds; class Plain { int value; public int read(int input) { return input + value; } }",
		"package bounds; @interface Mark { String value() default \"literal\"; } @Mark(\"Δ\") class Fixture<T> { public <U> U echo(U input) { return input; } public int[] pack(int... values) { return values; } private int dollar$name(int type) { return type; } void use() { class Local { void local() {} } Object x = new Object() { public void anonymous() {} }; java.util.function.IntUnaryOperator lambda = n -> n + 1; } class Nested { int member; } enum Choice { A { public void action() {} }; public void action() {} } record Pair(int component) { int read() { return component; } } }",
		"package bounds; class Δ { int café; public int read(int input) { /* commentName */ String text = \"stringName\"; return café + input; } }",
		"@Deprecated package bounds; import java.util.List;",
		"package bounds; class Empty {}",
	}
	for index, source := range sources {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			f := syntaxBounds214Parse(t, source)
			if f.root.HasError() {
				t.Fatal("valid fixed fixture grammar premise")
			}
			before := append([]byte(nil), f.scope.Source...)
			treeBefore := f.root.String()
			got := syntaxBounds214Equal(t, f.scope)
			if got == nil || got.anchor != f.scope.ParsedRoot() || !bytes.Equal(got.source, before) {
				t.Fatal("fixed fixture index premise")
			}
			if !bytes.Equal(before, f.scope.Source) || treeBefore != f.root.String() {
				t.Fatal("index construction mutated input")
			}
			// Snapshot bytes are copied at the original construction boundary.
			if len(f.scope.Source) > 0 {
				saved := f.scope.Source[0]
				f.scope.Source[0] ^= 1
				if !bytes.Equal(got.source, before) {
					t.Fatal("entry source aliases input")
				}
				f.scope.Source[0] = saved
			}
			runtime.KeepAlive(f)
		})
	}
}

func TestSyntaxIdentifierBounds214Refusals(t *testing.T) {
	for _, row := range []struct {
		name  string
		scope *symbol.FileScope
	}{{"nil_scope", nil}, {"nil_root", &symbol.FileScope{}}} {
		t.Run(row.name, func(t *testing.T) {
			if syntaxBounds214Equal(t, row.scope) != nil {
				t.Fatal("missing root admitted")
			}
		})
	}
	t.Run("parser_error", func(t *testing.T) {
		f := syntaxBounds214Parse(t, "class Broken { void method( {")
		if !f.root.HasError() || syntaxBounds214Equal(t, f.scope) != nil {
			t.Fatal("parser error refusal premise")
		}
	})
	t.Run("short_source", func(t *testing.T) {
		f := syntaxBounds214Parse(t, "class Fixture { int value; }")
		f.scope.Source = f.scope.Source[:1]
		if syntaxBounds214Equal(t, f.scope) != nil {
			t.Fatal("out of range identifier admitted")
		}
	})
	t.Run("detached_body", func(t *testing.T) {
		f := syntaxBounds214Parse(t, "class Fixture { int value; }")
		twin := syntaxBounds214Parse(t, "class Fixture { int value; }")
		if len(f.scope.TopLevelClasses) != 1 || len(twin.scope.TopLevelClasses) != 1 {
			t.Fatal("one body premise")
		}
		saved := f.scope.TopLevelClasses[0].Class.DeclarationNode
		f.scope.TopLevelClasses[0].Class.DeclarationNode = twin.scope.TopLevelClasses[0].Class.DeclarationNode
		defer func() { f.scope.TopLevelClasses[0].Class.DeclarationNode = saved }()
		if syntaxBounds214Equal(t, f.scope) != nil {
			t.Fatal("detached target admitted")
		}
	})
	t.Run("edited_tree", func(t *testing.T) {
		f := syntaxBounds214Parse(t, "class Fixture { int value; }")
		at := uint32(bytes.Index(f.source, []byte("value")))
		f.tree.Edit(sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}})
		if !f.root.HasChanges() || syntaxBounds214Equal(t, f.scope) != nil {
			t.Fatal("edited tree refusal premise")
		}
	})
}

func syntaxBounds214BudgetSource(width int) string {
	return "package bounds; class Fixture { int a,b,c; int fold(int value) { " + strings.Repeat("value = value + a + b + c; ", width) + "return value; } }"
}

func syntaxBounds214Window(build func(*symbol.FileScope) *syntaxIdentifierFile, file *symbol.FileScope) (int64, []*syntaxIdentifierFile) {
	// The parsed file and interned node wrappers remain fixed. Parse, symbol setup,
	// warmup, validation, logging, and garbage collection requests are outside.
	results := make([]*syntaxIdentifierFile, 2)
	before := runtime.NumCgoCall()
	for index := range results {
		results[index] = build(file)
	}
	count := runtime.NumCgoCall() - before
	runtime.KeepAlive(file)
	return count, results
}

func TestSyntaxIdentifierBounds214NativeBudget(t *testing.T) {
	widths := [2]int{64, 256}
	expectedSHA := [2]string{"a437871b55b763710811ba1c982a97ed1bb64be15c67f2c2ad9a846df4722194", "bf5ad57aaa0b7b146e5ff3ed7d8766e48d348b5fcd6bbfb38ac68161219ce619"}
	var fixtures [2]syntaxBounds214Fixture
	var expected [2]*syntaxIdentifierFile
	for index, width := range widths {
		fixtures[index] = syntaxBounds214Parse(t, syntaxBounds214BudgetSource(width))
		if fmt.Sprintf("%x", sha256.Sum256(fixtures[index].source)) != expectedSHA[index] || fixtures[index].root.HasError() {
			t.Fatal("fixed parsed input premise")
		}
		expected[index] = syntaxBounds214Equal(t, fixtures[index].scope)
		if expected[index] == nil {
			t.Fatal("fixed parsed input index refused")
		}
	}
	arms := [2]func(*symbol.FileScope) *syntaxIdentifierFile{syntaxBounds214ReferenceBuild, buildSyntaxIdentifierFile}
	// Counter RED is expected on unchanged production: the actual current builder
	// and frozen old builder perform the same work; only the fixed budget may fail.
	for sample := 1; sample <= 3; sample++ {
		var counts [2][2]int64
		var orders [2]string
		for widthIndex := range widths {
			order := [4]int{0, 1, 1, 0}
			orders[widthIndex] = "DOOD"
			if (sample+widthIndex)%2 == 1 {
				order = [4]int{1, 0, 0, 1}
				orders[widthIndex] = "ODDO"
			}
			for _, arm := range order {
				count, results := syntaxBounds214Window(arms[arm], fixtures[widthIndex].scope)
				counts[widthIndex][arm] += count
				for _, entry := range results {
					if !reflect.DeepEqual(entry, expected[widthIndex]) {
						t.Fatal("counter arm index changed")
					}
				}
			}
		}
		baseline, current := counts[1][0], counts[1][1]
		ceiling := 9 * baseline / 10
		t.Logf("SYNTAX_BOUNDS214_NATIVE sample=%d small_width=64 large_width=256 builds_per_arm=4 small_order=%s large_order=%s small_reference_cgo=%d small_current_cgo=%d large_reference_cgo=%d large_current_cgo=%d ceiling=%d", sample, orders[0], orders[1], counts[0][0], counts[0][1], baseline, current, ceiling)
		if baseline <= 0 || current <= 0 {
			t.Fatal("missing absolute native counter")
		}
		if current > ceiling {
			t.Errorf("SYNTAX_BOUNDS214_NATIVE_BUDGET_RED sample=%d current=%d exceeds fixed90percent=%d", sample, current, ceiling)
		}
	}
	runtime.KeepAlive(fixtures)
}
