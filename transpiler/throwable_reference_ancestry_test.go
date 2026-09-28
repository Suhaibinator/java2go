package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestThrowableReferenceAncestryCanonicalOwners(t *testing.T) {
	old := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = old })
	parse := func(name, source string) parsing.SourceFile {
		t.Helper()
		file := parsing.SourceFile{Name: name, Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		return file
	}
	parse("Shadows.java", `package shadows; class Exception {} class Throwable {} class ClassNotFoundException {}`)
	file := parse("Use.java", `package app; import shadows.Exception; import shadows.Throwable; import shadows.ClassNotFoundException;
 class Use {} class Child extends java.lang.ReflectiveOperationException {Child(){super();}}`)
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass}
	for _, actual := range []string{"java.lang.ClassNotFoundException", "java.lang.NoSuchMethodException", "java.lang.NoSuchFieldException", "java.lang.IllegalAccessException", "java.lang.reflect.InvocationTargetException", "java.lang.ReflectiveOperationException", "Child"} {
		for _, expected := range []string{"java.lang.Throwable", "java.lang.Exception", "java.lang.ReflectiveOperationException"} {
			if !javaExceptionReferenceAssignable(actual, expected, ctx) {
				t.Errorf("%s not assignable to %s under source shadow imports", actual, expected)
			}
		}
	}
	for _, pair := range [][2]string{
		{"ClassNotFoundException", "java.lang.Throwable"},
		{"shadows.ClassNotFoundException", "java.lang.Throwable"},
		{"other.ClassNotFoundException", "java.lang.Throwable"},
		{"java.lang.reflect.ClassNotFoundException", "java.lang.Throwable"},
		{"java.lang.InvocationTargetException", "java.lang.Throwable"},
		{"java.lang.ClassNotFoundException", "Throwable"},
		{"java.lang.ClassNotFoundException", "Exception"},
		{"java.lang.ClassNotFoundException", "java.lang.RuntimeException"},
	} {
		if javaExceptionReferenceAssignable(pair[0], pair[1], ctx) {
			t.Errorf("incorrect ancestry %s -> %s", pair[0], pair[1])
		}
	}
	external := Ctx{currentFile: &symbol.FileScope{Imports: map[string]string{"InvocationTargetException": "java.lang.reflect"}}}
	if !javaExceptionReferenceAssignable("InvocationTargetException", "java.lang.Throwable", external) {
		t.Fatal("canonical explicit import rejected")
	}
	external.currentFile.Imports["InvocationTargetException"] = "foreign"
	if javaExceptionReferenceAssignable("InvocationTargetException", "java.lang.Throwable", external) {
		t.Fatal("foreign explicit import accepted")
	}
	// Reference support must not enable nonexistent named Go constructors.
	for canonical := range reflectiveThrowableReferenceParents {
		if isBuiltinExceptionType(canonical) {
			t.Errorf("reference-only type enabled constructor lowering: %s", canonical)
		}
	}
}

func TestThrowableReferenceAncestryTypeBinders(t *testing.T) {
	for _, name := range []string{"ClassNotFoundException", "Throwable", "Exception", "InvocationTargetException"} {
		for _, methodOwned := range []bool{false, true} {
			parameter := symbol.TypeParam{Name: name}
			ctx := Ctx{currentClass: &symbol.ClassScope{TypeParameters: []symbol.TypeParam{parameter}}}
			if methodOwned {
				ctx.localScope = &symbol.Definition{IsStatic: true, TypeParameters: []symbol.TypeParam{parameter}}
			}
			if _, known := builtinThrowableReferenceName(name, ctx); known {
				t.Errorf("binder %s classified as builtin (method=%v)", name, methodOwned)
			}
			if javaExceptionReferenceAssignable(name, "java.lang.Throwable", ctx) {
				t.Errorf("actual binder %s classified by name", name)
			}
			if javaExceptionReferenceAssignable("java.lang.ClassNotFoundException", name, ctx) {
				t.Errorf("expected binder %s classified by name", name)
			}
			if !javaExceptionReferenceAssignable("java.lang.ClassNotFoundException", "java.lang.Throwable", ctx) {
				t.Errorf("qualified builtin blocked by binder %s", name)
			}
		}
	}
}
