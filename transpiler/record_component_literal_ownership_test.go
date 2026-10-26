package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestRecordComponentLiteralCanonicalDescriptors(t *testing.T) {
	for _, row := range []struct {
		name, source, typ string
		rank              int
	}{
		{"imported", `import java.lang.reflect.RecordComponent;class Probe{}`, "RecordComponent", 0},
		{"qualified", `class Probe{}`, "java.lang.reflect.RecordComponent", 0},
		{"demand", `import java.lang.reflect.*;class Probe{}`, "RecordComponent", 0},
		{"array", `import java.lang.reflect.RecordComponent;class Probe{}`, "RecordComponent[]", 1},
		{"array2", `import java.lang.reflect.RecordComponent;class Probe{}`, "RecordComponent[][]", 2},
		{"value-name", `import java.lang.reflect.RecordComponent;class Probe{int RecordComponent;}`, "RecordComponent", 0},
		{"value-root", `class Probe{int java;}`, "java.lang.reflect.RecordComponent", 0},
		{"go-alias", `import java.lang.reflect.RecordComponent;class Probe{int stdjava;}`, "RecordComponent", 0},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := classScopeCtx(h.File.Symbols.FindClassScope("Probe"), h.Ctx)
			expr, known := javaTypeDescriptorExpr(row.typ, ctx)
			if !known {
				t.Fatal("canonical descriptor rejected")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "RecordComponentTypeID") || strings.Count(out.String(), "ArrayTypeID(") != row.rank {
				t.Fatalf("wrong canonical/rank descriptor: %s", out.String())
			}
		})
	}
}

func TestRecordComponentLiteralOwnershipShields(t *testing.T) {
	for _, row := range []struct{ name, source, typ string }{
		{"member", `import java.lang.reflect.RecordComponent;class Probe{static class RecordComponent{}}`, "RecordComponent"},
		{"source", `class RecordComponent{}class Probe{}`, "RecordComponent"},
		{"binder", `import java.lang.reflect.RecordComponent;class Probe<RecordComponent>{}`, "RecordComponent"},
		{"method-binder", `import java.lang.reflect.RecordComponent;class Probe{<RecordComponent> void body(){}}`, "RecordComponent"},
		{"foreign-import", `import foreign.RecordComponent;class Probe{}`, "RecordComponent"},
		{"unproved-demand", `import java.lang.reflect.*;import foreign.*;class Probe{}`, "RecordComponent"},
		{"no-import", `class Probe{}`, "RecordComponent"},
		{"source-root", `class Probe{static class java{static class lang{static class reflect{static class RecordComponent{}}}}}`, "java.lang.reflect.RecordComponent"},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := classScopeCtx(h.File.Symbols.FindClassScope("Probe"), h.Ctx)
			if row.name == "method-binder" {
				ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("body")[0]
			}
			if _, proved := sourceReflectionRecordComponentDescriptor(row.typ, ctx); proved {
				t.Fatal("source/binder/foreign name borrowed native identity")
			}
			expr, known := javaTypeDescriptorExpr(row.typ, ctx)
			if !known {
				t.Fatal("legacy source/binder descriptor disappeared")
			}
			var out bytes.Buffer
			if err := printer.Fprint(&out, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "RecordComponentTypeID") {
				t.Fatalf("native descriptor leaked through shield: %s", out.String())
			}
		})
	}
}

func TestRecordComponentLiteralRequiresOwnedDeclaration(t *testing.T) {
	h := setupParseHelper(t, `import java.lang.reflect.RecordComponent;class Probe{}`)
	ctx := classScopeCtx(h.File.Symbols.FindClassScope("Probe"), h.Ctx)
	if _, proved := sourceReflectionRecordComponentDescriptor("RecordComponent", ctx); !proved {
		t.Fatal("genuine owner rejected")
	}
	missingFile, missingOwner, copiedFile := ctx.Clone(), ctx.Clone(), ctx.Clone()
	missingFile.currentFile = nil
	missingOwner.currentClass = nil
	copied := *ctx.currentFile
	copiedFile.currentFile = &copied
	for _, row := range []struct {
		name string
		ctx  Ctx
	}{{"missing-file", missingFile}, {"missing-owner", missingOwner}, {"copied-file", copiedFile}} {
		t.Run(row.name, func(t *testing.T) {
			if _, proved := sourceReflectionRecordComponentDescriptor("RecordComponent", row.ctx); proved {
				t.Fatal("detached origin acquired native descriptor")
			}
		})
	}
}

func TestRecordComponentLiteralCurrentSourceNamespace(t *testing.T) {
	for _, mutation := range []string{"rebound-import", "package", "source", "genuine"} {
		t.Run(mutation, func(t *testing.T) {
			source := `import java.lang.reflect.RecordComponent;class Probe{}`
			if mutation == "rebound-import" {
				source = `import foreign.RecordComponent;class Probe{}`
			}
			h := setupParseHelper(t, source)
			ctx := classScopeCtx(h.File.Symbols.FindClassScope("Probe"), h.Ctx)
			if mutation == "rebound-import" {
				ctx.currentFile.Imports["RecordComponent"] = "java.lang.reflect"
			}
			if mutation == "package" {
				ctx.currentFile.Package = "foreign"
			}
			if mutation == "source" {
				ctx.currentFile.Source[0] ^= 1
			}
			_, proved := sourceReflectionRecordComponentDescriptor("RecordComponent", ctx)
			if proved != (mutation == "genuine") {
				t.Fatalf("native literal proved=%v mutation=%s", proved, mutation)
			}
		})
	}
}
