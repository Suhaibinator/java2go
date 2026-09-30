package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func parseJavaFileForTypeInfo(t *testing.T, path string) parsing.SourceFile {
	t.Helper()

	files, err := parsing.ReadSourcesInDir(path)
	if err != nil {
		t.Fatalf("failed reading source %s: %v", path, err)
	}
	if len(files) == 0 {
		t.Fatalf("no java files found for %s", path)
	}

	file := files[0]
	if err := file.ParseAST(); err != nil {
		t.Fatalf("failed parsing AST for %s: %v", path, err)
	}
	file.ParseSymbols()
	ResolveFile(file)

	return file
}

func TestSimpleDeclaration(t *testing.T) {
	file := parseJavaFileForTypeInfo(t, "../testfiles/typechecks/SimpleDeclaration.java")
	info, err := ExtractTypeInformation(file)
	if err != nil {
		t.Fatalf("ExtractTypeInformation returned error: %v", err)
	}

	classInfo, ok := info.Classes["SimpleDeclaration"]
	if !ok {
		t.Fatalf("expected class info for SimpleDeclaration, got: %#v", info.Classes)
	}

	mainInfo, ok := classInfo.Methods["main"]
	if !ok {
		t.Fatalf("expected method info for main, got: %#v", classInfo.Methods)
	}

	if got := mainInfo.Parameters["args"]; got != "String[]" {
		t.Fatalf("expected main args parameter type String[], got %q", got)
	}
	if got := mainInfo.Locals["variable"]; got != "int" {
		t.Fatalf("expected local variable type int, got %q", got)
	}
}

func TestMethodDeclaration(t *testing.T) {
	file := parseJavaFileForTypeInfo(t, "../testfiles/typechecks/MethodConstructorDeclaration.java")
	info, err := ExtractTypeInformation(file)
	if err != nil {
		t.Fatalf("ExtractTypeInformation returned error: %v", err)
	}

	classInfo, ok := info.Classes["MethodConstructorDeclaration"]
	if !ok {
		t.Fatalf("expected class info for MethodConstructorDeclaration, got: %#v", info.Classes)
	}

	sayHelloInfo, ok := classInfo.Methods["sayHello"]
	if !ok {
		t.Fatalf("expected method info for sayHello, got: %#v", classInfo.Methods)
	}
	if sayHelloInfo.ReturnType != "String" {
		t.Fatalf("expected return type String for sayHello, got %q", sayHelloInfo.ReturnType)
	}

	squaredInfo, ok := classInfo.Methods["squared"]
	if !ok {
		t.Fatalf("expected method info for squared, got: %#v", classInfo.Methods)
	}
	if squaredInfo.ReturnType != "int" {
		t.Fatalf("expected return type int for squared, got %q", squaredInfo.ReturnType)
	}
	if got := squaredInfo.Parameters["n"]; got != "int" {
		t.Fatalf("expected parameter n type int, got %q", got)
	}
}
