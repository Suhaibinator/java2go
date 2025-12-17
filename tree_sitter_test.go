package main

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// --- Unit Tests for Helper Functions ---

func TestCapitalizeIdent(t *testing.T) {
	in := &ast.Ident{Name: "test"}
	out := CapitalizeIdent(in)
	if out.Name != "Test" {
		t.Errorf("Expected Test, got %s", out.Name)
	}
}

func TestLowercaseIdent(t *testing.T) {
	in := &ast.Ident{Name: "Test"}
	out := LowercaseIdent(in)
	if out.Name != "test" {
		t.Errorf("Expected test, got %s", out.Name)
	}
}

func TestCtxClone(t *testing.T) {
	original := Ctx{
		className:    "TestClass",
		expectedType: "int",
		currentFile:  &symbol.FileScope{},
	}

	clone := original.Clone()

	if clone.className != original.className {
		t.Errorf("Expected className %s, got %s", original.className, clone.className)
	}
	if clone.expectedType != original.expectedType {
		t.Errorf("Expected expectedType %s, got %s", original.expectedType, clone.expectedType)
	}
	if clone.currentFile != original.currentFile {
		t.Error("Expected currentFile pointers to be identical")
	}

	clone.className = "NewClass"
	if original.className == "NewClass" {
		t.Error("Modifying clone's className affected original")
	}
}

// --- Integration Tests for ParseNode ---

// Helper to parse Java code and return the helper struct containing AST and Context
type ParseHelper struct {
	File parsing.SourceFile
	Ctx  Ctx
}

func setupParseHelper(t *testing.T, source string) *ParseHelper {
	file := parsing.SourceFile{
		Name:   "Test.java",
		Source: []byte(source),
	}
	if err := file.ParseAST(); err != nil {
		t.Fatalf("Failed to parse AST: %v", err)
	}

	symbols := file.ParseSymbols()
	symbol.AddSymbolsToPackage(symbols)

	ResolveFile(file)

	ctx := Ctx{
		currentFile:  file.Symbols,
		currentClass: file.Symbols.BaseClass,
	}

	return &ParseHelper{
		File: file,
		Ctx:  ctx,
	}
}

// Helper to find the first node of a given type
func findNode(node *sitter.Node, typeName string) *sitter.Node {
	if node.Type() == typeName {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		found := findNode(node.Child(i), typeName)
		if found != nil {
			return found
		}
	}
	return nil
}

func TestParseNode_Program(t *testing.T) {
	src := `
package com.example;
import java.util.List;
public class TestProgram {}
`
	helper := setupParseHelper(t, src)
	// Program node is the root
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)

	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	if file.Name.Name != "example" {
		t.Errorf("Expected package name 'example', got '%s'", file.Name.Name)
	}

	if len(file.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(file.Imports))
	}
	// Note: Current behavior of ParseExpr for scoped_identifier (java.util.List) returns the root (java).
	// This might be unintended behavior in the codebase, but the test reflects current state.
	if file.Imports[0].Name.Name != "java" {
		t.Errorf("Expected import name 'java', got '%s'", file.Imports[0].Name.Name)
	}
}

func TestParseNode_MethodDeclaration_Interface(t *testing.T) {
	// ParseNode returns *ast.Field for method declarations (used in interfaces)
	src := `
package com.example;
public interface TestInterface {
    void myMethod(int a);
}
`
	helper := setupParseHelper(t, src)
	methodNode := findNode(helper.File.Ast, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration")
	}

	res := ParseNode(methodNode, helper.File.Source, helper.Ctx)
	field, ok := res.(*ast.Field)
	if !ok {
		t.Fatalf("Expected *ast.Field, got %T", res)
	}

	// Interface methods are not automatically capitalized in the current implementation
	if len(field.Names) != 1 || field.Names[0].Name != "myMethod" {
		t.Errorf("Expected method name 'myMethod', got %v", field.Names)
	}
}

func TestParseNode_MethodDeclaration_InterfaceGenericParam(t *testing.T) {
	src := `
package com.example;
public interface Holder<T> {
    void put(T value);
}
`
	helper := setupParseHelper(t, src)
	methodNode := findNode(helper.File.Ast, "method_declaration")
	if methodNode == nil {
		t.Fatal("Could not find method_declaration")
	}

	res := ParseNode(methodNode, helper.File.Source, helper.Ctx)
	field, ok := res.(*ast.Field)
	if !ok {
		t.Fatalf("Expected *ast.Field, got %T", res)
	}

	funcType, ok := field.Type.(*ast.FuncType)
	if !ok {
		t.Fatalf("Expected FuncType, got %T", field.Type)
	}
	if len(funcType.Params.List) != 1 {
		t.Fatalf("Expected 1 parameter, got %d", len(funcType.Params.List))
	}
	paramType, ok := funcType.Params.List[0].Type.(*ast.Ident)
	if !ok {
		t.Fatalf("Expected parameter type to be *ast.Ident, got %T", funcType.Params.List[0].Type)
	}
	if paramType.Name != "T" {
		t.Errorf("Expected parameter type 'T', got '%s'", paramType.Name)
	}
}

func TestParseNode_FieldDeclaration(t *testing.T) {
	src := `
package com.example;
public class TestField {
    public int myField;
    private String val = "abc";
}
`
	helper := setupParseHelper(t, src)

	// Test uninitialized field
	fieldNode := findNode(helper.File.Ast, "field_declaration") // Finds the first one (myField)
	if fieldNode == nil {
		t.Fatal("Could not find field_declaration")
	}

	res := ParseNode(fieldNode, helper.File.Source, helper.Ctx)
	field, ok := res.(*ast.Field)
	if !ok {
		t.Fatalf("Expected *ast.Field for uninitialized field, got %T", res)
	}
	// Check name (should be Capitalized if public)
	if len(field.Names) != 1 || field.Names[0].Name != "MyField" {
		t.Errorf("Expected field name 'MyField', got %v", field.Names)
	}

	// Test initialized field
	// Find the second field declaration
	var initializedFieldNode *sitter.Node
	foundCount := 0
	checkNode := func(n *sitter.Node) {
		if n.Type() == "field_declaration" {
			foundCount++
			if foundCount == 2 {
				initializedFieldNode = n
			}
		}
	}
	// Simple traversal to find 2nd field
	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		checkNode(n)
		if initializedFieldNode != nil {
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(i))
		}
	}
	traverse(helper.File.Ast)

	if initializedFieldNode == nil {
		t.Fatal("Could not find second field_declaration")
	}

	res = ParseNode(initializedFieldNode, helper.File.Source, helper.Ctx)
	valSpec, ok := res.(*ast.ValueSpec)
	if !ok {
		t.Fatalf("Expected *ast.ValueSpec for initialized field, got %T", res)
	}
	if len(valSpec.Names) != 1 || valSpec.Names[0].Name != "val" {
		// Private field -> lowercase
		t.Errorf("Expected field name 'val', got %v", valSpec.Names)
	}
}

func TestParseNode_FieldDeclaration_GenericTypeParameter(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    public T value;
    private T cached = null;
}
`
	helper := setupParseHelper(t, src)

	var fieldNodes []*sitter.Node
	var collect func(*sitter.Node)
	collect = func(n *sitter.Node) {
		if n.Type() == "field_declaration" {
			fieldNodes = append(fieldNodes, n)
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			collect(n.Child(i))
		}
	}
	collect(helper.File.Ast)

	if len(fieldNodes) < 2 {
		t.Fatalf("Expected at least 2 field declarations, got %d", len(fieldNodes))
	}

	// Uninitialized generic field should keep the bare type parameter (not *T)
	fieldRes := ParseNode(fieldNodes[0], helper.File.Source, helper.Ctx)
	field, ok := fieldRes.(*ast.Field)
	if !ok {
		t.Fatalf("Expected *ast.Field, got %T", fieldRes)
	}
	fieldType, ok := field.Type.(*ast.Ident)
	if !ok {
		t.Fatalf("Expected field type to be *ast.Ident, got %T", field.Type)
	}
	if fieldType.Name != "T" {
		t.Errorf("Expected field type 'T', got '%s'", fieldType.Name)
	}

	// Initialized generic field should also preserve the bare type parameter.
	valueRes := ParseNode(fieldNodes[1], helper.File.Source, helper.Ctx)
	valSpec, ok := valueRes.(*ast.ValueSpec)
	if !ok {
		t.Fatalf("Expected *ast.ValueSpec, got %T", valueRes)
	}
	valType, ok := valSpec.Type.(*ast.Ident)
	if !ok {
		t.Fatalf("Expected value spec type to be *ast.Ident, got %T", valSpec.Type)
	}
	if valType.Name != "T" {
		t.Errorf("Expected value spec type 'T', got '%s'", valType.Name)
	}
}

func TestParseNode_TryStatement(t *testing.T) {
	src := `
package com.example;
public class TestTry {
    public void test() {
        try {
            int x = 1;
        } catch (Exception e) {
        }
    }
}
`
	helper := setupParseHelper(t, src)
	tryNode := findNode(helper.File.Ast, "try_statement")
	if tryNode == nil {
		t.Fatal("Could not find try_statement")
	}

	// We need to set localScope to the method for context (though try_statement might not strictly need it if variable types are explicit)
	// But let's try calling it.
	res := ParseNode(tryNode, helper.File.Source, helper.Ctx)
	stmts, ok := res.([]ast.Stmt)
	if !ok {
		t.Fatalf("Expected []ast.Stmt, got %T", res)
	}

	// Should contain the body of the try block (int x = 1)
	if len(stmts) != 1 {
		t.Errorf("Expected 1 statement, got %d", len(stmts))
	}
}

func TestParseNode_Parameters(t *testing.T) {
	src := `
package com.example;
public class TestParams {
    public void method(int a, String... args) {}
}
`
	helper := setupParseHelper(t, src)

	// Need to find the method scope to populate ctx.localScope if we want accurate parameter parsing
	methodDef := helper.Ctx.currentClass.FindMethod().ByName("Method")[0] // Renamed to Method
	helper.Ctx.localScope = methodDef

	// Test formal_parameters list
	paramsNode := findNode(helper.File.Ast, "formal_parameters")
	if paramsNode == nil {
		t.Fatal("Could not find formal_parameters")
	}

	res := ParseNode(paramsNode, helper.File.Source, helper.Ctx)
	fieldList, ok := res.(*ast.FieldList)
	if !ok {
		t.Fatalf("Expected *ast.FieldList, got %T", res)
	}

	if len(fieldList.List) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(fieldList.List))
	}

	// Check first param
	if fieldList.List[0].Names[0].Name != "a" {
		t.Errorf("Expected first param 'a', got %s", fieldList.List[0].Names[0].Name)
	}

	// Check spread parameter
	spreadParam := fieldList.List[1]
	if spreadParam.Names[0].Name != "args" {
		t.Errorf("Expected second param 'args', got %s", spreadParam.Names[0].Name)
	}
	if _, ok := spreadParam.Type.(*ast.Ellipsis); !ok {
		t.Errorf("Expected Ellipsis type for spread param, got %T", spreadParam.Type)
	}
}

func TestParseNode_SwitchLabel(t *testing.T) {
	src := `
package com.example;
public class TestSwitch {
    public void test(int x) {
        switch (x) {
            case 1: break;
        }
    }
}
`
	helper := setupParseHelper(t, src)
	labelNode := findNode(helper.File.Ast, "switch_label")
	if labelNode == nil {
		t.Fatal("Could not find switch_label")
	}

	res := ParseNode(labelNode, helper.File.Source, helper.Ctx)
	caseClause, ok := res.(*ast.CaseClause)
	if !ok {
		t.Fatalf("Expected *ast.CaseClause, got %T", res)
	}

	if len(caseClause.List) != 1 {
		t.Errorf("Expected 1 expression in case list, got %d", len(caseClause.List))
	}
}

func TestParseNode_ImportDeclaration(t *testing.T) {
	src := `
package com.example;
import java.util.List;
public class TestImport {}
`
	helper := setupParseHelper(t, src)
	importNode := findNode(helper.File.Ast, "import_declaration")
	if importNode == nil {
		t.Fatal("Could not find import_declaration")
	}

	res := ParseNode(importNode, helper.File.Source, helper.Ctx)
	importSpec, ok := res.(*ast.ImportSpec)
	if !ok {
		t.Fatalf("Expected *ast.ImportSpec, got %T", res)
	}

	// ParseNode implementation: returns ImportSpec with Name set to the last identifier part?
	// case "import_declaration": return &ast.ImportSpec{Name: ParseExpr(node.NamedChild(0), source, ctx).(*ast.Ident)}
	// The named child 0 is the scoped_identifier (java.util.List).
	// ParseExpr on scoped_identifier returns an *ast.SelectorExpr or *ast.Ident depending on implementation.
	// Wait, let's check ParseExpr implementation or rely on what ParseNode returns.

	if importSpec.Name.Name != "java" {
		t.Errorf("Expected import name 'java', got '%s'", importSpec.Name.Name)
	}
}

// TestDiamondOperatorDetection tests that diamond operator (<>) is correctly
// distinguished from raw type usage (no type parameters)
func TestDiamondOperatorDetection(t *testing.T) {
	// Test diamond operator: new Box<>() - should infer type from declaration
	t.Run("DiamondOperator", func(t *testing.T) {
		src := `
package com.example;
public class Box<T> {
    T value;
    public Box() {}
    public static void test() {
        Box<String> box = new Box<>();
    }
}
`
		helper := setupParseHelper(t, src)
		node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
		file, ok := node.(*ast.File)
		if !ok {
			t.Fatalf("Expected *ast.File, got %T", node)
		}

		// Find the generated code and verify it includes type parameters
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
			t.Fatalf("Failed to print AST: %v", err)
		}
		output := buf.String()

		// Diamond operator should result in type inference - the call should include [String]
		if !strings.Contains(output, "[String]") && !strings.Contains(output, "[string]") {
			t.Errorf("Diamond operator should infer type arguments, got:\n%s", output)
		}
	})

	// Test raw type: new Box() - should NOT infer type (deprecated but valid Java)
	t.Run("RawType", func(t *testing.T) {
		src := `
package com.example;
public class Box<T> {
    T value;
    public Box() {}
    public static void test() {
        Box raw = new Box();
    }
}
`
		helper := setupParseHelper(t, src)
		node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
		file, ok := node.(*ast.File)
		if !ok {
			t.Fatalf("Expected *ast.File, got %T", node)
		}

		// Find the object_creation_expression node and check isDiamond behavior
		objectCreationNode := findNode(helper.File.Ast, "object_creation_expression")
		if objectCreationNode == nil {
			t.Fatal("Could not find object_creation_expression")
		}

		// Get the type node - for raw type it should be type_identifier, not generic_type
		typeNode := objectCreationNode.ChildByFieldName("type")
		if typeNode == nil {
			t.Fatal("Could not find type node in object_creation_expression")
		}

		// Raw type should NOT be parsed as generic_type
		if typeNode.Type() == "generic_type" {
			t.Error("Raw type 'new Box()' should not be parsed as generic_type")
		}

		// Verify the generated code - raw type should result in ConstructBox() without type args
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
			t.Fatalf("Failed to print AST: %v", err)
		}
		output := buf.String()
		t.Logf("Generated output:\n%s", output)
	})

	// Test explicit type: new Box<String>() - should use explicit type args
	t.Run("ExplicitType", func(t *testing.T) {
		src := `
package com.example;
public class Box<T> {
    T value;
    public Box() {}
    public static void test() {
        Box<Integer> box = new Box<Integer>();
    }
}
`
		helper := setupParseHelper(t, src)
		node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
		file, ok := node.(*ast.File)
		if !ok {
			t.Fatalf("Expected *ast.File, got %T", node)
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
			t.Fatalf("Failed to print AST: %v", err)
		}
		output := buf.String()

		// Explicit type args should be preserved
		if !strings.Contains(output, "[Integer]") && !strings.Contains(output, "[*Integer]") && !strings.Contains(output, "[int]") {
			t.Errorf("Explicit type arguments should be preserved, got:\n%s", output)
		}
	})

	// Test diamond operator with multiple type arguments inferred from expected type
	t.Run("DiamondOperatorMultipleTypeArgs", func(t *testing.T) {
		src := `
package com.example;
public class Pair<K, V> {
    K key;
    V value;

    public Pair(K k, V v) {
        this.key = k;
        this.value = v;
    }

    public static Pair<String, Integer> create(String k, Integer v) {
        Pair<String, Integer> pair = new Pair<>(k, v);
        return pair;
    }
}
`
		helper := setupParseHelper(t, src)
		node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
		file, ok := node.(*ast.File)
		if !ok {
			t.Fatalf("Expected *ast.File, got %T", node)
		}

		var buf bytes.Buffer
		if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
			t.Fatalf("Failed to print AST: %v", err)
		}
		output := buf.String()

		if !strings.Contains(output, "NewPair[String, Integer]") && !strings.Contains(output, "NewPair[string, *Integer]") && !strings.Contains(output, "NewPair[string,*Integer]") {
			t.Errorf("Diamond operator should infer multiple type arguments, got:\n%s", output)
		}
	})
}

// TestGenericClass_ConstructorAndReceiver tests that generic class constructors
// return *ClassName[T] and methods have receivers with type parameters.
func TestGenericClass_ConstructorAndReceiver(t *testing.T) {
	// Test with two type parameters to ensure multi-param generics work
	src := `
package com.example;
public class Pair<K, V> {
    K key;
    V value;

    public Pair(K k, V v) {
        this.key = k;
        this.value = v;
    }

    public K getKey() {
        return this.key;
    }

    public V getValue() {
        return this.value;
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	// Constructor should return *Pair[K, V]
	if !strings.Contains(output, "*Pair[K, V]") {
		t.Errorf("Constructor should return *Pair[K, V], got:\n%s", output)
	}

	// Methods should have receiver (pr *Pair[K, V]) - ShortName("Pair") = "pr"
	if !strings.Contains(output, "(pr *Pair[K, V])") {
		t.Errorf("Methods should have receiver (pr *Pair[K, V]), got:\n%s", output)
	}

	// Struct should be defined with type parameters: type Pair[K any, V any] struct
	if !strings.Contains(output, "Pair[K any, V any]") {
		t.Errorf("Struct should have type parameters [K any, V any], got:\n%s", output)
	}
}

// TestGenericClass_SingleTypeParam tests the simpler single type parameter case
func TestGenericClass_SingleTypeParam(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    T value;

    public Box(T v) {
        this.value = v;
    }

    public T get() {
        return this.value;
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	// Constructor should return *Box[T]
	if !strings.Contains(output, "*Box[T]") {
		t.Errorf("Constructor should return *Box[T], got:\n%s", output)
	}

	// Methods should have receiver (bx *Box[T]) - ShortName("Box") = "bx"
	if !strings.Contains(output, "(bx *Box[T])") {
		t.Errorf("Methods should have receiver (bx *Box[T]), got:\n%s", output)
	}

	// Struct definition: type Box[T any] struct
	if !strings.Contains(output, "Box[T any]") {
		t.Errorf("Struct should have type parameter [T any], got:\n%s", output)
	}
}

func TestStaticGenericMethod(t *testing.T) {
	src := `
package com.example;
public class Utils {
    public static <R> R identity(R value) {
        return value;
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "func Identity[R any]") {
		t.Errorf("Static generic method should include type parameters, got:\n%s", output)
	}
	if !strings.Contains(output, "value R") {
		t.Errorf("Parameter should remain type R, got:\n%s", output)
	}
	if !strings.Contains(output, ") R {") {
		t.Errorf("Return type should be R, got:\n%s", output)
	}
}

func TestInstanceGenericMethodHelperRequired(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    public <R> R identity(R value) {
        return value;
    }

    public static <X> X callIdentity(Box<X> box, X value) {
        return box.identity(value);
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "type BoxIdentityHelper") {
		t.Errorf("Expected helper type for instance generic method, got:\n%s", output)
	}
	if !strings.Contains(output, "func NewBoxIdentityHelper") {
		t.Errorf("Expected helper constructor for instance generic method, got:\n%s", output)
	}
	if !strings.Contains(output, "NewBoxIdentityHelper") || !strings.Contains(output, ".Identity(") {
		t.Errorf("Expected call sites to use helper, got:\n%s", output)
	}
}

func TestParseExpr_InstanceGenericMethodInvocationUsesHelper(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    public <R> R identity(R value) {
        return value;
    }

    public static <X> X callIdentity(Box<X> box, X value) {
        return box.identity(value);
    }
}
`
	helper := setupParseHelper(t, src)

	invocationNode := findNode(helper.File.Ast, "method_invocation")
	if invocationNode == nil {
		t.Fatal("Could not find method_invocation node")
	}

	methodDefs := helper.Ctx.currentClass.FindMethod().ByName("CallIdentity")
	if len(methodDefs) == 0 {
		t.Fatal("Could not find CallIdentity definition")
	}

	ctx := helper.Ctx
	ctx.localScope = methodDefs[0]

	expr := ParseExpr(invocationNode, helper.File.Source, ctx)
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		t.Fatalf("Expected *ast.CallExpr, got %T", expr)
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("Expected call fun to be SelectorExpr, got %T", call.Fun)
	}

	helperCall, ok := sel.X.(*ast.CallExpr)
	if !ok {
		t.Fatalf("Expected selector receiver to be CallExpr (helper constructor), got %T", sel.X)
	}

	var constructorName string
	switch fun := helperCall.Fun.(type) {
	case *ast.Ident:
		constructorName = fun.Name
	case *ast.IndexExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			constructorName = ident.Name
		}
	case *ast.IndexListExpr:
		if ident, ok := fun.X.(*ast.Ident); ok {
			constructorName = ident.Name
		}
	default:
		t.Fatalf("Expected helper constructor to be Ident or indexed generic call, got %T", helperCall.Fun)
	}

	if constructorName != "NewBoxIdentityHelper" {
		t.Errorf("Expected helper constructor 'NewBoxIdentityHelper', got '%s'", constructorName)
	}
}

func TestInstanceGenericMethodHelper_InfersPointerTypeArgs(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    public <R> R identity(R value) {
        return value;
    }

    public static Foo call(Box<Foo> box, Foo value) {
        return box.identity(value);
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	// Foo is a Java reference type; the generator represents it as *Foo, so the
	// helper invocation needs to pass *Foo as a type argument (not Foo).
	if !strings.Contains(output, "NewBoxIdentityHelper[*Foo, *Foo]") && !strings.Contains(output, "NewBoxIdentityHelper[*Foo,*Foo]") {
		t.Errorf("Expected helper invocation to use pointer type args for Foo, got:\n%s", output)
	}
}

func TestInstanceGenericMethodHelper_InfersNestedGenericTypeArgs(t *testing.T) {
	src := `
package com.example;
public class Box<T> {
    public <R> R identity(R value) {
        return value;
    }

    public static List<Foo> call(Box<List<Foo>> box, List<Foo> value) {
        return box.identity(value);
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	// Nested type args should be converted to Go's indexed generic form and keep
	// pointer semantics for reference types.
	if !strings.Contains(output, "NewBoxIdentityHelper[*List[*Foo], *List[*Foo]]") && !strings.Contains(output, "NewBoxIdentityHelper[*List[*Foo],*List[*Foo]]") {
		t.Errorf("Expected helper invocation to use nested generic type args, got:\n%s", output)
	}
}

// TestInnerClass_ParentTypeParameterReuse tests that inner class constructors
// inherit the parent class's type parameters (the third fallback path in expression.go)
func TestInnerClass_ParentTypeParameterReuse(t *testing.T) {
	src := `
package com.example;
public class LinkedList<E> {
    E value;
    Node head;

    public LinkedList() {}

    class Node {
        E element;
        Node next;

        Node(E e) {
            this.element = e;
        }
    }

    public void addFirst(E e) {
        Node newNode = new Node(e);
        this.head = newNode;
    }
}
`
	helper := setupParseHelper(t, src)
	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("Expected *ast.File, got %T", node)
	}

	var buf bytes.Buffer
	if err := printer.Fprint(&buf, token.NewFileSet(), file); err != nil {
		t.Fatalf("Failed to print AST: %v", err)
	}
	output := buf.String()

	// The inner class constructor call should inherit parent type params
	// new Node(e) inside a generic class should become a generic constructor call
	// using the parent type parameter, e.g. ConstructNode[E](e) or newNode[E](e).
	if !strings.Contains(output, "ConstructNode[E]") && !strings.Contains(output, "newNode[E]") && !strings.Contains(output, "NewNode[E]") {
		t.Errorf("Inner class constructor should use parent type param [E], got:\n%s", output)
	}
}

// TestVariadicParameter_WithTypeParameter tests that variadic parameters with
// type parameters produce *ast.Ellipsis with the correct element type (not *ast.StarExpr)
func TestVariadicParameter_WithTypeParameter(t *testing.T) {
	src := `
package com.example;
public class Utils<T> {
    public void process(T... values) {
    }

    public static <E> void collect(E... items) {
    }
}
`
	helper := setupParseHelper(t, src)

	// Find the process method parameters
	methodDefs := helper.Ctx.currentClass.FindMethod().ByName("Process")
	if len(methodDefs) == 0 {
		t.Fatal("Could not find Process method")
	}
	helper.Ctx.localScope = methodDefs[0]

	// Find spread_parameter node
	var spreadNode *sitter.Node
	var findSpread func(*sitter.Node)
	findSpread = func(n *sitter.Node) {
		if n.Type() == "spread_parameter" {
			spreadNode = n
			return
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			findSpread(n.Child(i))
			if spreadNode != nil {
				return
			}
		}
	}
	findSpread(helper.File.Ast)

	if spreadNode == nil {
		t.Fatal("Could not find spread_parameter node")
	}

	res := ParseNode(spreadNode, helper.File.Source, helper.Ctx)
	field, ok := res.(*ast.Field)
	if !ok {
		t.Fatalf("Expected *ast.Field, got %T", res)
	}

	// Type should be *ast.Ellipsis, NOT *ast.StarExpr
	ellipsis, ok := field.Type.(*ast.Ellipsis)
	if !ok {
		t.Fatalf("Expected variadic parameter to have *ast.Ellipsis type, got %T", field.Type)
	}

	// The element type should be T (an identifier), not *T (a StarExpr)
	// Since T is a type parameter, it should not be wrapped in a pointer
	elt, ok := ellipsis.Elt.(*ast.Ident)
	if !ok {
		// If it's a StarExpr with T inside, that's the bug we're testing for
		if star, isStar := ellipsis.Elt.(*ast.StarExpr); isStar {
			if ident, isIdent := star.X.(*ast.Ident); isIdent && ident.Name == "T" {
				t.Errorf("Type parameter T in variadic should not be wrapped in *ast.StarExpr, got *T")
			}
		}
		t.Errorf("Expected ellipsis element to be *ast.Ident for type parameter T, got %T", ellipsis.Elt)
	} else if elt.Name != "T" {
		t.Errorf("Expected ellipsis element name 'T', got '%s'", elt.Name)
	}
}
