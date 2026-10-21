package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestStringReaderReferenceParsedDeclarationABI(t *testing.T) {
	source := `import java.io.StringReader; class NativeOwner { StringReader field; java.io.StringReader same(StringReader value){return value;} static <StringReader> java.io.StringReader qualified(java.io.StringReader value){return value;} }`
	rendered := normalizeSpaces(renderGoFileFromJava(t, source))
	if strings.Count(rendered, "*stdjava.StringReader") < 5 || strings.Contains(rendered, "*StringReader") {
		t.Fatalf("declared pointer ABI missing: %s", rendered)
	}
}

func TestStringReaderReferenceParsedTypeRootsDecline(t *testing.T) {
	sources := []string{
		`class Binder<java>{java.io.StringReader value;}`,
		`class MethodBinder{<java> void use(java.io.StringReader value){}}`,
		`class Captured<java>{class Inner{java.io.StringReader value;}}`,
		`class SourceRoot{static class java{} java.io.StringReader value;}`,
		`class LocalRoot{void use(){class java{} java.io.StringReader value=null;}}`,
	}
	for _, source := range sources {
		rendered := renderGoFileFromJava(t, source)
		if strings.Contains(rendered, "stdjava.StringReader") {
			t.Fatalf("bound TYPE root acquired native pointer: %s", rendered)
		}
	}
}

func TestStringReaderReferenceParsedInvalidGenericStrict(t *testing.T) {
	sources := []string{
		`class InvalidField{java.io.StringReader<Object> value;}`,
		`import java.io.StringReader;class InvalidImport{StringReader<Object> value;}`,
		`class InvalidFormal{void use(java.io.StringReader<Object> value){}}`,
		`class InvalidReturn{java.io.StringReader<Object> value(){return null;}}`,
		`class InvalidArray{java.io.StringReader<Object>[] value;}`,
		`class InvalidArrayFormal{void use(java.io.StringReader<Object>[][] value){}}`,
		`class InvalidArrayReturn{java.io.StringReader<Object>[] use(){return null;}}`,
	}
	for index, source := range sources {
		t.Run(string(rune('A'+index)), func(t *testing.T) {
			withCleanDiagnostics(t)
			previous := symbol.GlobalScope
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
			t.Cleanup(func() { symbol.GlobalScope = previous })
			root := t.TempDir()
			file := filepath.Join(root, "Invalid.java")
			if err := os.WriteFile(file, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			err := run([]string{"-sync", "-strict", file}, &output)
			if err == nil {
				t.Fatalf("actual parsed nongeneric JDK declaration silently accepted: %s", output.String())
			}
			if strings.Contains(output.String(), "BadExpr") {
				t.Fatal("strict published invalid Go")
			}
			found := false
			for _, d := range Diagnostics() {
				if d.NodeType == "ast.BadExpr" {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing terminal parsed type diagnostic: %v %#v", err, Diagnostics())
			}
		})
	}
}

func TestStringReaderReferenceParsedArraysAndSourceGenerics(t *testing.T) {
	native := normalizeSpaces(renderGoFileFromJava(t, `import java.io.StringReader;class ArraysOwner{StringReader[] field;StringReader[][] nested;StringReader[] same(StringReader[] value){return value;}}`))
	if strings.Count(native, "*stdjava.ReferenceArray") < 4 || strings.Contains(native, "BadExpr") {
		t.Fatalf("established common reference-array declaration ABI missing: %s", native)
	}
	source := renderGoFileFromJava(t, `class SourceOwner{static class StringReader<T>{T value;T get(){return value;}}StringReader<Object> field;}`)
	if strings.Contains(source, "stdjava.StringReader") || strings.Contains(source, "BadExpr") {
		t.Fatalf("source generic declaration swallowed by native refusal: %s", source)
	}
}

func TestStringReaderReferenceParsedArrayComponentAndShields(t *testing.T) {
	native := normalizeSpaces(renderGoFileFromJava(t, `import java.io.StringReader;class ArraysOwner{StringReader first(StringReader[] values){return values[0];}}`))
	if !strings.Contains(native, "ReferenceArrayGet[*stdjava.StringReader]") || strings.Contains(native, "BadExpr") {
		t.Fatalf("native scalar component missing: %s", native)
	}
	for _, source := range []string{
		`class Binder<java>{java.io.StringReader<Object>[] value;}`,
		`class MethodBinder{<java> void use(java.io.StringReader<Object>[] value){}}`,
		`class Captured<java>{class Inner{java.io.StringReader<Object>[] value;}}`,
		`class SourceRoot{static class java{} java.io.StringReader<Object>[] value;}`,
		`class SourceOwner{static class StringReader<T>{T value;}StringReader<Object>[] field;}`,
	} {
		rendered := renderGoFileFromJava(t, source)
		if strings.Contains(rendered, "stdjava.StringReader") || strings.Contains(rendered, "BadExpr") {
			t.Fatalf("bound/source generic array took canonical-invalid refusal: %s", rendered)
		}
	}
	valueRoot := renderGoFileFromJava(t, `class ValueRoot{Object java;java.io.StringReader[] field;}`)
	if !strings.Contains(valueRoot, "*stdjava.ReferenceArray") || strings.Contains(valueRoot, "BadExpr") {
		t.Fatalf("VALUE root changed legal array declaration: %s", valueRoot)
	}
}

func TestStringReaderReferenceDeclaredSourceInvocation(t *testing.T) {
	source := `class TypeRoot{static class java{static class io{static class StringReader{int read(){return 79;}void close(){}}}}java.io.StringReader field=new java.io.StringReader();}class Consumer{int read(TypeRoot root){return root.field.read();}void close(TypeRoot root){root.field.close();}}`
	rendered := renderGoFileFromJava(t, source)
	if strings.Contains(rendered, "BadExpr") || strings.Contains(rendered, "stdjava.StringReaderReadCharExecution") || strings.Contains(rendered, "stdjava.StringReaderCloseExecution") {
		t.Fatalf("actual source declaration lost original source invocation: %s", rendered)
	}
}
