package transpiler

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
	// The production entry point resets the project-wide symbol graph for each
	// conversion. Mirror that boundary here so package-level collision and
	// hierarchy analyses cannot see classes parsed by an earlier unit test.
	previousGlobal := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() {
		symbol.GlobalScope = previousGlobal
	})

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

	// The unused Java import is omitted; only the execution-ABI runtime import remains.
	if len(file.Imports) != 1 || file.Imports[0].Name == nil || file.Imports[0].Name.Name != "stdjava" ||
		file.Imports[0].Path.Value != `"github.com/NickyBoy89/java2go/stdjava"` {
		t.Errorf("Expected only the stdjava execution-ABI import, got %#v", file.Imports)
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

	// Interface methods are implicitly public in Java, so generated names are exported.
	if len(field.Names) != 1 || field.Names[0].Name != "MyMethod" {
		t.Errorf("Expected method name 'MyMethod', got %v", field.Names)
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
	if len(stmts) < 3 {
		t.Errorf("Expected lowered try/catch/finally statements, got %d", len(stmts))
	}

	var rendered bytes.Buffer
	for _, stmt := range stmts {
		if err := printer.Fprint(&rendered, token.NewFileSet(), stmt); err != nil {
			t.Fatalf("Failed to print lowered try statement: %v", err)
		}
	}
	output := strings.Join(strings.Fields(rendered.String()), " ")
	if !strings.Contains(output, "recover()") {
		t.Fatalf("Expected lowered try to include recover(), got: %s", output)
	}
	if !strings.Contains(output, "__java2goCatchHandled") {
		t.Fatalf("Expected lowered catch dispatch state, got: %s", output)
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
	var rendered bytes.Buffer
	if err := printer.Fprint(&rendered, token.NewFileSet(), spreadParam.Type); err != nil {
		t.Fatal(err)
	}
	if got := rendered.String(); got != "*stdjava.ReferenceArray" {
		t.Errorf("Expected Java reference-array type for spread param, got %s", got)
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

	if importSpec.Path == nil || importSpec.Path.Value != "\"java/util\"" {
		t.Errorf("Expected import path '\"java/util\"', got %v", importSpec.Path)
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

		// Diamond operator should result in type inference - the call should include the canonical String type argument
		if !strings.Contains(output, "NewBoxJava2goExecution[*stdjava.JavaString]") {
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

		// Explicit Java wrapper arguments retain their nullable object type.
		if !strings.Contains(output, "NewBoxJava2goExecution[*stdjava.Integer]") {
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

		if !strings.Contains(output, "NewPairJava2goExecution[*stdjava.JavaString, *stdjava.Integer]") {
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
	if !strings.Contains(output, "NewBoxIdentityHelper") || !strings.Contains(output, ".IdentityJava2goExecution(") {
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

	owner := helper.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("original Box owner lost")
	}
	identity := owner.FindMethodByName("identity", nil)
	if identity == nil || identity.IsPrivate || identity.IsStatic || !identity.RequiresHelper || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 {
		t.Fatal("original public instance identity declaration lost")
	}
	ownerParameter := owner.TypeParameters[0]
	if ownerParameter.Name != "T" || ownerParameter.Declaration == nil || len(ownerParameter.Bounds) != 0 || identity.OriginalType != "R" || identity.Parameters[0].OriginalType != "R" {
		t.Fatal("original logical owner T or method R signature lost")
	}
	r := identity.TypeParameters[0].Declaration
	if r == nil || r == owner.TypeParameters[0].Declaration || definitionDeclaredJavaType(identity).TypeParameterBindings["R"] != r || identity.Parameters[0].TypeParameterBindings["R"] != r {
		t.Fatal("method-owned R declaration identity lost")
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		id, ok := expr.(*ast.Ident)
		if !ok || id.Name != name {
			t.Fatalf("expected %s identifier, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime %s selector, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	ownerCtx := helper.Ctx.Clone()
	ownerCtx.currentClass = owner
	family := canonicalGenericFamily(owner, ownerCtx)
	if family == nil || !canonicalGenericClass(owner, ownerCtx) {
		t.Fatal("original generic-X demand lost its completed physical family")
	}
	if _, member := family.members[owner]; !member {
		t.Fatal("physical family lost actual source owner membership")
	}
	if _, binder := family.binders[ownerParameter.Declaration]; !binder {
		t.Fatal("physical family lost original owner T declaration")
	}
	ownerRepresentation, represented := family.representations[ownerParameter.Declaration]
	if !represented || ownerRepresentation.erasure != "java.lang.Object" || ownerRepresentation.goName != "any" {
		t.Fatal("owner T lost independent Object physical representation")
	}
	assertIdent(genericFamilyBinderGoType(ownerParameter, ownerCtx), "any")
	rawName := availableCanonicalGenericName(owner.Class.Name + "Java2goErased")
	check := func(callerName string, foo bool) {
		t.Helper()
		caller := owner.FindMethodByName(callerName, nil)
		if caller == nil {
			t.Fatal("original caller lost")
		}
		ctx := helper.Ctx.Clone()
		ctx.localScope = caller
		ctx.expectedType, ctx.expectedTypeRoot = "", nil
		invocation := findNode(caller.DeclarationNode, "method_invocation")
		if invocation == nil || invocation.ChildByFieldName("object").Content(helper.File.Source) != "box" || invocation.ChildByFieldName("arguments").NamedChildCount() != 1 || invocation.ChildByFieldName("arguments").NamedChild(0).Content(helper.File.Source) != "value" {
			t.Fatal("original invocation nodes changed")
		}
		assertLogical := func(expr ast.Expr) {
			t.Helper()
			if foo {
				p, ok := expr.(*ast.StarExpr)
				if !ok {
					t.Fatalf("Foo lost pointer: %T", expr)
				}
				assertIdent(p.X, "Foo")
				return
			}
			if len(caller.TypeParameters) != 1 || caller.TypeParameters[0].Name != "X" || caller.TypeParameters[0].Declaration == nil || caller.TypeParameters[0].Declaration == r || caller.Parameters[1].TypeParameterBindings["X"] != caller.TypeParameters[0].Declaration {
				t.Fatal("caller-owned X lost its declaration identity")
			}
			assertIdent(expr, caller.TypeParameters[0].EmittedName())
		}
		target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, helper.File.Source)
		resolved, selected := findBestMethodForInvocationTarget(target, "identity", invocation.ChildByFieldName("arguments"), true, false, ctx, helper.File.Source)
		if target == nil || selected == nil || resolved == nil || resolved.def != identity || resolved.owner != owner || target.classScope != owner || len(target.classTypeArgs) != 1 {
			t.Fatal("actual selected owner/method declaration lost")
		}
		assertLogical(target.classTypeArgs[0])
		bindings := genericJavaInvocationTypeBindings(resolved.def, invocation, ctx, helper.File.Source)
		if len(bindings) != 1 || bindings["R"] == "" {
			t.Fatalf("actual no-target R inference lost: %v", bindings)
		}
		assertLogical(javaTypeStringToGoTypeExpr(bindings["R"], inScopeTypeParameters(ctx), ctx))
		logical := inferMethodTypeArguments(resolved.def, invocation, ctx, helper.File.Source)
		if len(logical) != 1 {
			t.Fatal("logical R arity changed")
		}
		assertLogical(logical[0])
		plan := planGenericMethodEntryABI(owner, identity, ctx)
		if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != r || len(plan.parameterTypes) != 1 || plan.resultJavaType != "java.lang.Object" {
			t.Fatal("actual declaration-owned erased entry lost")
		}
		assertIdent(plan.parameterTypes[0], "any")
		assertIdent(plan.resultType, "any")
		physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
		if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
			t.Fatal("helper physical R first-bound slot changed")
		}
		assertIdent(physical[0], "any")
		verifyExpression := func(expr ast.Expr, execution string) {
			t.Helper()
			view, ok := expr.(*ast.CallExpr)
			if !ok || len(view.Args) != 2 {
				t.Fatalf("one typed consumer view required: %#v", expr)
			}
			indexed, ok := view.Fun.(*ast.IndexExpr)
			if !ok {
				t.Fatal("consumer type argument lost")
			}
			assertRuntime(indexed.X, "ObjectView")
			assertLogical(indexed.Index)
			if foo {
				if reflectCoreTypeID(t, view.Args[1]) != "Foo" {
					t.Fatal("original synthetic Foo nominal descriptor changed")
				}
			} else {
				assertRuntime(view.Args[1], "ObjectTypeID")
			}
			invoke, ok := view.Args[0].(*ast.CallExpr)
			if !ok {
				t.Fatal("consumer no longer encloses one call")
			}
			selector, ok := invoke.Fun.(*ast.SelectorExpr)
			if !ok {
				t.Fatal("instance selector lost")
			}
			assertIdent(selector.X, "box")
			if execution == "" {
				if selector.Sel.Name != identity.Name || len(invoke.Args) != 1 {
					t.Fatal("original no-Execution parse context changed")
				}
				assertIdent(invoke.Args[0], "value")
			} else {
				if selector.Sel.Name != executionImplementationName(identity, owner, ctx) || len(invoke.Args) != 2 {
					t.Fatal("Execution forwarding changed")
				}
				assertIdent(invoke.Args[0], execution)
				assertIdent(invoke.Args[1], "value")
			}
		}
		verifyExpression(ParseExpr(invocation, helper.File.Source, ctx), "")
		withExecution := ctx.Clone()
		withExecution.executionContextName = "unitExecution"
		verifyExpression(ParseExpr(invocation, helper.File.Source, withExecution), "unitExecution")
	}
	check("callIdentity", false)

	node := ParseNode(helper.File.Ast, helper.File.Source, helper.Ctx)
	file, ok := node.(*ast.File)
	if !ok {
		t.Fatalf("expected complete file AST: %T", node)
	}
	var logicalAlias, physicalLayout *ast.TypeSpec
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typ, ok := specification.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if typ.Name.Name == owner.Class.Name {
				if logicalAlias != nil {
					t.Fatal("duplicate logical owner alias")
				}
				logicalAlias = typ
			}
			if typ.Name.Name == rawName {
				if physicalLayout != nil {
					t.Fatal("duplicate physical owner layout")
				}
				physicalLayout = typ
			}
		}
	}
	if logicalAlias == nil || !logicalAlias.Assign.IsValid() || logicalAlias.TypeParams == nil || len(logicalAlias.TypeParams.List) != 1 || len(logicalAlias.TypeParams.List[0].Names) != 1 || physicalLayout == nil || physicalLayout.TypeParams != nil {
		t.Fatal("logical owner T alias and non-generic physical class layout lost")
	}
	assertIdent(logicalAlias.Type, rawName)
	assertIdent(logicalAlias.TypeParams.List[0].Type, "any")
	if logicalAlias.TypeParams.List[0].Names[0].Name != ownerParameter.EmittedName() {
		t.Fatal("logical owner T declaration alias renamed")
	}
	if _, ok := physicalLayout.Type.(*ast.StructType); !ok {
		t.Fatal("physical class layout stopped being a shared struct")
	}
	var entry, body *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "IdentityJava2goExecution" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		p, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			t.Fatal("instance receiver pointer lost")
		}
		switch x := p.X.(type) {
		case *ast.Ident:
			if x.Name == rawName {
				if entry != nil || fn.Type.TypeParams != nil {
					t.Fatal("non-generic physical entry duplicated or regained owner T")
				}
				entry = fn
			}
		case *ast.IndexListExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == identity.HelperName {
				if body != nil || len(x.Indices) != 2 {
					t.Fatal("typed body slot count changed")
				}
				assertIdent(x.Indices[0], "T")
				assertIdent(x.Indices[1], "R")
				body = fn
			}
		}
	}
	if entry == nil || body == nil || len(entry.Type.Params.List) != 2 || entry.Type.Results == nil || len(entry.Type.Results.List) != 1 || len(entry.Body.List) != 1 {
		t.Fatal("exact erased entry/typed helper body lost")
	}
	assertIdent(entry.Type.Params.List[1].Type, "any")
	assertIdent(entry.Type.Results.List[0].Type, "any")
	returned, ok := entry.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		t.Fatal("one bridge return required")
	}
	call, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 2 {
		t.Fatal("one original body execution required")
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != body.Name.Name {
		t.Fatal("bridge original body lost")
	}
	constructor, ok := selector.X.(*ast.CallExpr)
	if !ok || len(constructor.Args) != 1 {
		t.Fatal("one helper receiver required")
	}
	indexed, ok := constructor.Fun.(*ast.IndexListExpr)
	if !ok || len(indexed.Indices) != 2 {
		t.Fatal("helper T/physical-R slots lost")
	}
	assertIdent(indexed.X, "New"+identity.HelperName)
	assertIdent(indexed.Indices[0], "any")
	assertIdent(indexed.Indices[1], "any")
	assertIdent(constructor.Args[0], entry.Recv.List[0].Names[0].Name)
	assertIdent(call.Args[0], entry.Type.Params.List[0].Names[0].Name)
	assertIdent(call.Args[1], "value")
	if len(body.Type.Params.List) != 2 || body.Type.Results == nil || len(body.Type.Results.List) != 1 || len(body.Body.List) == 0 {
		t.Fatal("original typed implementation lost")
	}
	assertIdent(body.Type.Params.List[1].Type, "R")
	assertIdent(body.Type.Results.List[0].Type, "R")
	last, ok := body.Body.List[len(body.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(last.Results) != 1 {
		t.Fatal("identity return lost")
	}
	assertIdent(last.Results[0], "value")
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
	// Inference is a logical declaration-owned contract. Do not infer it from
	// the consuming return type or require a physical helper specialization.
	owner := helper.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("original Box owner binder lost")
	}
	identity := owner.FindMethodByName("identity", nil)
	caller := owner.FindMethodByName("call", nil)
	if identity == nil || caller == nil || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 || !identity.RequiresHelper {
		t.Fatal("original generic method declarations lost")
	}
	ownerParameter := owner.TypeParameters[0]
	if ownerParameter.Name != "T" || ownerParameter.Declaration == nil || len(ownerParameter.Bounds) != 0 || identity.OriginalType != "R" || identity.Parameters[0].OriginalType != "R" {
		t.Fatal("original logical owner T or method R signature lost")
	}
	r := identity.TypeParameters[0].Declaration
	if r == nil || r == owner.TypeParameters[0].Declaration || identity.TypeParameters[0].Name != "R" || definitionDeclaredJavaType(identity).TypeParameterBindings["R"] != r || identity.Parameters[0].TypeParameterBindings["R"] != r {
		t.Fatal("method-owned R lost its declaration identity")
	}
	ctx := helper.Ctx.Clone()
	ctx.localScope = caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	if canonicalGenericClass(owner, ctx) || canonicalGenericFamily(owner, ctx) != nil {
		t.Fatal("original ordinary pointer-only owner unexpectedly acquired an erased class family")
	}
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil || invocation.ChildByFieldName("object").Content(helper.File.Source) != "box" || invocation.ChildByFieldName("arguments").NamedChildCount() != 1 || invocation.ChildByFieldName("arguments").NamedChild(0).Content(helper.File.Source) != "value" {
		t.Fatal("original receiver or value argument changed")
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name != name {
			t.Fatalf("expected exact identifier %s, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime selector %s, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	assertFoo := func(expr ast.Expr) {
		t.Helper()
		pointer, ok := expr.(*ast.StarExpr)
		if !ok {
			t.Fatalf("Foo lost its reference pointer: %T", expr)
		}
		assertIdent(pointer.X, "Foo")
	}
	target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, helper.File.Source)
	resolved, selected := findBestMethodForInvocationTarget(target, "identity", invocation.ChildByFieldName("arguments"), true, false, ctx, helper.File.Source)
	if target == nil || selected == nil || resolved == nil || resolved.owner != owner || resolved.def != identity || target.classScope != owner || len(target.classTypeArgs) != 1 {
		t.Fatal("original invocation no longer resolves the exact owner/method declaration")
	}
	assertFoo(target.classTypeArgs[0])
	bindings := genericJavaInvocationTypeBindings(resolved.def, invocation, ctx, helper.File.Source)
	if len(bindings) != 1 || bindings["R"] == "" {
		t.Fatalf("missing actual method R inference: %v", bindings)
	}
	assertFoo(javaTypeStringToGoTypeExpr(bindings["R"], inScopeTypeParameters(ctx), ctx))
	logical := inferMethodTypeArguments(resolved.def, invocation, ctx, helper.File.Source)
	if len(logical) != 1 {
		t.Fatalf("logical R argument count=%d", len(logical))
	}
	assertFoo(logical[0])
	plan := planGenericMethodEntryABI(owner, identity, ctx)
	if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != r || len(plan.parameterTypes) != 1 || plan.resultJavaType != "java.lang.Object" {
		t.Fatal("physical ABI lost the admitted declaration-owned Object slot")
	}
	assertIdent(plan.parameterTypes[0], "any")
	assertIdent(plan.resultType, "any")
	physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
	if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
		t.Fatal("physical helper R slot is not its declared Object erasure")
	}
	assertIdent(physical[0], "any")
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

	// Inspect the emitted AST: the typed reference view belongs to the consumer,
	// while the virtual entry executes the same method body with physical any.
	var call, entry, implementation *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "CallJava2goExecution" && fn.Recv == nil {
			if call != nil {
				t.Fatal("duplicate typed call declaration")
			}
			call = fn
		}
		if fn.Name.Name != "IdentityJava2goExecution" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		pointer, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			t.Fatal("generic method receiver lost pointer")
		}
		switch receiver := pointer.X.(type) {
		case *ast.IndexExpr:
			if name, ok := receiver.X.(*ast.Ident); ok && name.Name == "Box" {
				assertIdent(receiver.Index, "T")
				if entry != nil {
					t.Fatal("duplicate erased instance entry")
				}
				entry = fn
			}
		case *ast.IndexListExpr:
			if name, ok := receiver.X.(*ast.Ident); ok && name.Name == identity.HelperName {
				if len(receiver.Indices) != 2 {
					t.Fatal("typed helper lost T/R slots")
				}
				assertIdent(receiver.Indices[0], "T")
				assertIdent(receiver.Indices[1], "R")
				if implementation != nil {
					t.Fatal("duplicate typed helper body")
				}
				implementation = fn
			}
		}
	}
	if call == nil || entry == nil || implementation == nil {
		t.Fatalf("typed consumer, erased entry or typed implementation missing:\n%s", output)
	}
	if len(call.Type.Params.List) != 3 || call.Type.Results == nil || len(call.Type.Results.List) != 1 || len(call.Body.List) != 1 {
		t.Fatal("typed consumer signature/body changed")
	}
	execution, ok := call.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		t.Fatal("caller Execution lost pointer")
	}
	assertRuntime(execution.X, "Execution")
	box, ok := call.Type.Params.List[1].Type.(*ast.StarExpr)
	if !ok {
		t.Fatal("Box receiver parameter lost pointer")
	}
	boxed, ok := box.X.(*ast.IndexExpr)
	if !ok {
		t.Fatal("Box receiver lost owner instantiation")
	}
	assertIdent(boxed.X, "Box")
	assertFoo(boxed.Index)
	assertFoo(call.Type.Params.List[2].Type)
	assertFoo(call.Type.Results.List[0].Type)
	returned, ok := call.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		t.Fatal("typed result return changed")
	}
	view, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(view.Args) != 2 {
		t.Fatal("expected one consumer projection")
	}
	indexed, ok := view.Fun.(*ast.IndexExpr)
	if !ok {
		t.Fatal("consumer projection lost type argument")
	}
	assertRuntime(indexed.X, "ObjectView")
	assertFoo(indexed.Index)
	if reflectCoreTypeID(t, view.Args[1]) != "Foo" {
		t.Fatal("consumer lost original synthetic Foo descriptor")
	}
	invoke, ok := view.Args[0].(*ast.CallExpr)
	if !ok || len(invoke.Args) != 2 {
		t.Fatal("projection must enclose one instance invocation")
	}
	selector, ok := invoke.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != entry.Name.Name {
		t.Fatal("consumer bypassed erased virtual entry")
	}
	assertIdent(selector.X, "box")
	assertIdent(invoke.Args[0], call.Type.Params.List[0].Names[0].Name)
	assertIdent(invoke.Args[1], "value")
	if len(entry.Type.Params.List) != 2 || entry.Type.Results == nil || len(entry.Type.Results.List) != 1 || len(entry.Body.List) != 1 {
		t.Fatal("erased entry signature or bridge body changed")
	}
	assertIdent(entry.Type.Params.List[1].Type, "any")
	assertIdent(entry.Type.Results.List[0].Type, "any")
	bridgeReturn, ok := entry.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(bridgeReturn.Results) != 1 {
		t.Fatal("erased bridge return changed")
	}
	bridge, ok := bridgeReturn.Results[0].(*ast.CallExpr)
	if !ok || len(bridge.Args) != 2 {
		t.Fatal("erased bridge must execute body once")
	}
	bridgeSelector, ok := bridge.Fun.(*ast.SelectorExpr)
	if !ok || bridgeSelector.Sel.Name != implementation.Name.Name {
		t.Fatal("erased entry lost existing implementation")
	}
	constructor, ok := bridgeSelector.X.(*ast.CallExpr)
	if !ok || len(constructor.Args) != 1 {
		t.Fatal("typed body must retain one receiver")
	}
	constructorType, ok := constructor.Fun.(*ast.IndexListExpr)
	if !ok || len(constructorType.Indices) != 2 {
		t.Fatal("physical helper lost owner/method slots")
	}
	assertIdent(constructorType.X, "New"+identity.HelperName)
	assertIdent(constructorType.Indices[0], "T")
	assertIdent(constructorType.Indices[1], "any")
	assertIdent(constructor.Args[0], entry.Recv.List[0].Names[0].Name)
	assertIdent(bridge.Args[0], entry.Type.Params.List[0].Names[0].Name)
	assertIdent(bridge.Args[1], "value")
	if len(implementation.Type.Params.List) != 2 || implementation.Type.Results == nil || len(implementation.Type.Results.List) != 1 || len(implementation.Body.List) == 0 {
		t.Fatal("original typed identity implementation lost")
	}
	assertIdent(implementation.Type.Params.List[1].Type, "R")
	assertIdent(implementation.Type.Results.List[0].Type, "R")
	bodyReturn, ok := implementation.Body.List[len(implementation.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(bodyReturn.Results) != 1 {
		t.Fatal("identity body stopped returning its original value")
	}
	assertIdent(bodyReturn.Results[0], "value")
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
	// Inference is a logical declaration-owned contract. Do not infer it from
	// the consuming return type or require a physical helper specialization.
	owner := helper.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("original Box owner binder lost")
	}
	identity := owner.FindMethodByName("identity", nil)
	caller := owner.FindMethodByName("call", nil)
	if identity == nil || caller == nil || len(identity.TypeParameters) != 1 || len(identity.Parameters) != 1 || !identity.RequiresHelper {
		t.Fatal("original generic method declarations lost")
	}
	r := identity.TypeParameters[0].Declaration
	if r == nil || r == owner.TypeParameters[0].Declaration || identity.TypeParameters[0].Name != "R" || definitionDeclaredJavaType(identity).TypeParameterBindings["R"] != r || identity.Parameters[0].TypeParameterBindings["R"] != r {
		t.Fatal("method-owned R lost its declaration identity")
	}
	ctx := helper.Ctx.Clone()
	ctx.localScope = caller
	ctx.expectedType, ctx.expectedTypeRoot = "", nil
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil || invocation.ChildByFieldName("object").Content(helper.File.Source) != "box" || invocation.ChildByFieldName("arguments").NamedChildCount() != 1 || invocation.ChildByFieldName("arguments").NamedChild(0).Content(helper.File.Source) != "value" {
		t.Fatal("original receiver or value argument changed")
	}
	assertIdent := func(expr ast.Expr, name string) {
		t.Helper()
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name != name {
			t.Fatalf("expected exact identifier %s, got %#v", name, expr)
		}
	}
	assertRuntime := func(expr ast.Expr, name string) {
		t.Helper()
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			t.Fatalf("expected runtime selector %s, got %#v", name, expr)
		}
		assertIdent(selector.X, "stdjava")
	}
	assertNestedList := func(expr ast.Expr) {
		t.Helper()
		pointer, ok := expr.(*ast.StarExpr)
		if !ok {
			t.Fatalf("nested List lost reference pointer: %T", expr)
		}
		indexed, ok := pointer.X.(*ast.IndexExpr)
		if !ok {
			t.Fatalf("nested List lost one exact type argument: %T", pointer.X)
		}
		assertRuntime(indexed.X, "List")
		foo, ok := indexed.Index.(*ast.StarExpr)
		if !ok {
			t.Fatalf("nested Foo lost reference pointer: %T", indexed.Index)
		}
		assertIdent(foo.X, "Foo")
	}
	target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, helper.File.Source)
	resolved, selected := findBestMethodForInvocationTarget(target, "identity", invocation.ChildByFieldName("arguments"), true, false, ctx, helper.File.Source)
	if target == nil || selected == nil || resolved == nil || resolved.owner != owner || resolved.def != identity || target.classScope != owner || len(target.classTypeArgs) != 1 {
		t.Fatal("original invocation no longer resolves the exact owner/method declaration")
	}
	assertNestedList(target.classTypeArgs[0])
	bindings := genericJavaInvocationTypeBindings(resolved.def, invocation, ctx, helper.File.Source)
	if len(bindings) != 1 || bindings["R"] == "" {
		t.Fatalf("missing actual method R inference: %v", bindings)
	}
	assertNestedList(javaTypeStringToGoTypeExpr(bindings["R"], inScopeTypeParameters(ctx), ctx))
	logical := inferMethodTypeArguments(resolved.def, invocation, ctx, helper.File.Source)
	if len(logical) != 1 {
		t.Fatalf("logical R argument count=%d", len(logical))
	}
	assertNestedList(logical[0])
	plan := planGenericMethodEntryABI(owner, identity, ctx)
	if plan == nil || plan.owner != owner || plan.method != identity || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != r || len(plan.parameterTypes) != 1 || plan.resultJavaType != "java.lang.Object" {
		t.Fatal("physical ABI lost the admitted declaration-owned Object slot")
	}
	assertIdent(plan.parameterTypes[0], "any")
	assertIdent(plan.resultType, "any")
	physical, javaTypes, known := genericMethodPhysicalTypeArguments(identity, ctx)
	if !known || len(physical) != 1 || len(javaTypes) != 1 || javaTypes[0] != "java.lang.Object" {
		t.Fatal("physical helper R slot is not its declared Object erasure")
	}
	assertIdent(physical[0], "any")
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

	// Inspect the emitted AST: the typed nested view belongs to the consumer,
	// while the virtual entry executes the same method body with physical any.
	var call, entry, implementation *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "CallJava2goExecution" && fn.Recv == nil {
			if call != nil { t.Fatal("duplicate typed call declaration") }
			call = fn
		}
		if fn.Name.Name != "IdentityJava2goExecution" || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		pointer, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
		if !ok { t.Fatal("generic method receiver lost pointer") }
		switch receiver := pointer.X.(type) {
		case *ast.IndexExpr:
			if name, ok := receiver.X.(*ast.Ident); ok && name.Name == "Box" {
				assertIdent(receiver.Index, "T")
				if entry != nil { t.Fatal("duplicate erased instance entry") }
				entry = fn
			}
		case *ast.IndexListExpr:
			if name, ok := receiver.X.(*ast.Ident); ok && name.Name == identity.HelperName {
				if len(receiver.Indices) != 2 { t.Fatal("typed helper lost T/R slots") }
				assertIdent(receiver.Indices[0], "T")
				assertIdent(receiver.Indices[1], "R")
				if implementation != nil { t.Fatal("duplicate typed helper body") }
				implementation = fn
			}
		}
	}
	if call == nil || entry == nil || implementation == nil {
		t.Fatalf("typed consumer, erased entry or typed implementation missing:\n%s", output)
	}
	if len(call.Type.Params.List) != 3 || call.Type.Results == nil || len(call.Type.Results.List) != 1 || len(call.Body.List) != 1 {
		t.Fatal("typed consumer signature/body changed")
	}
	execution, ok := call.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok { t.Fatal("caller Execution lost pointer") }
	assertRuntime(execution.X, "Execution")
	box, ok := call.Type.Params.List[1].Type.(*ast.StarExpr)
	if !ok { t.Fatal("Box receiver parameter lost pointer") }
	boxed, ok := box.X.(*ast.IndexExpr)
	if !ok { t.Fatal("Box receiver lost owner instantiation") }
	assertIdent(boxed.X, "Box")
	assertNestedList(boxed.Index)
	assertNestedList(call.Type.Params.List[2].Type)
	assertNestedList(call.Type.Results.List[0].Type)
	returned, ok := call.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 { t.Fatal("typed result return changed") }
	view, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(view.Args) != 2 { t.Fatal("expected one consumer projection") }
	indexed, ok := view.Fun.(*ast.IndexExpr)
	if !ok { t.Fatal("consumer projection lost type argument") }
	assertRuntime(indexed.X, "ObjectView")
	assertNestedList(indexed.Index)
	if reflectCoreTypeID(t, view.Args[1]) != "java.util.List" { t.Fatal("consumer lost canonical List descriptor") }
	invoke, ok := view.Args[0].(*ast.CallExpr)
	if !ok || len(invoke.Args) != 2 { t.Fatal("projection must enclose one instance invocation") }
	selector, ok := invoke.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != entry.Name.Name { t.Fatal("consumer bypassed erased virtual entry") }
	assertIdent(selector.X, "box")
	assertIdent(invoke.Args[0], call.Type.Params.List[0].Names[0].Name)
	assertIdent(invoke.Args[1], "value")
	if len(entry.Type.Params.List) != 2 || entry.Type.Results == nil || len(entry.Type.Results.List) != 1 || len(entry.Body.List) != 1 {
		t.Fatal("erased entry signature or bridge body changed")
	}
	assertIdent(entry.Type.Params.List[1].Type, "any")
	assertIdent(entry.Type.Results.List[0].Type, "any")
	bridgeReturn, ok := entry.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(bridgeReturn.Results) != 1 { t.Fatal("erased bridge return changed") }
	bridge, ok := bridgeReturn.Results[0].(*ast.CallExpr)
	if !ok || len(bridge.Args) != 2 { t.Fatal("erased bridge must execute body once") }
	bridgeSelector, ok := bridge.Fun.(*ast.SelectorExpr)
	if !ok || bridgeSelector.Sel.Name != implementation.Name.Name { t.Fatal("erased entry lost existing implementation") }
	constructor, ok := bridgeSelector.X.(*ast.CallExpr)
	if !ok || len(constructor.Args) != 1 { t.Fatal("typed body must retain one receiver") }
	constructorType, ok := constructor.Fun.(*ast.IndexListExpr)
	if !ok || len(constructorType.Indices) != 2 { t.Fatal("physical helper lost owner/method slots") }
	assertIdent(constructorType.X, "New"+identity.HelperName)
	assertIdent(constructorType.Indices[0], "T")
	assertIdent(constructorType.Indices[1], "any")
	assertIdent(constructor.Args[0], entry.Recv.List[0].Names[0].Name)
	assertIdent(bridge.Args[0], entry.Type.Params.List[0].Names[0].Name)
	assertIdent(bridge.Args[1], "value")
	if len(implementation.Type.Params.List) != 2 || implementation.Type.Results == nil || len(implementation.Type.Results.List) != 1 || len(implementation.Body.List) == 0 {
		t.Fatal("original typed identity implementation lost")
	}
	assertIdent(implementation.Type.Params.List[1].Type, "R")
	assertIdent(implementation.Type.Results.List[0].Type, "R")
	bodyReturn, ok := implementation.Body.List[len(implementation.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(bodyReturn.Results) != 1 { t.Fatal("identity body stopped returning its original value") }
	assertIdent(bodyReturn.Results[0], "value")
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

	// The inner class constructor call should inherit parent type params.
	// new Node(e) inside a generic class becomes a generic constructor call using
	// the parent type parameter. The nested class Node is renamed to the
	// collision-safe LinkedListnode, and its constructor follows that name so the
	// call binds to the actual generated struct, e.g. newLinkedListnode[E](e).
	if !strings.Contains(output, "[E]") ||
		(!strings.Contains(output, "newLinkedListnodeJava2goExecution[E]") &&
			!strings.Contains(output, "NewLinkedListnodeJava2goExecution[E]") &&
			!strings.Contains(output, "ConstructNodeJava2goExecution[E]")) {
		t.Errorf("Inner class constructor should use parent type param [E] and the renamed nested-class name, got:\n%s", output)
	}
}

// TestVariadicParameter_WithTypeParameter tests that variadic parameters with
// type parameters use the reference-array ABI, preserving the Java array object.
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

	arrayPointer, ok := field.Type.(*ast.StarExpr)
	if !ok {
		t.Fatalf("Expected variadic parameter to have array pointer type, got %T", field.Type)
	}
	arrayType, ok := arrayPointer.X.(*ast.SelectorExpr)
	if !ok || arrayType.Sel.Name != "ReferenceArray" {
		t.Fatalf("Expected variadic type parameter to use stdjava.ReferenceArray, got %#v", arrayPointer.X)
	}
	if pkg, ok := arrayType.X.(*ast.Ident); !ok || pkg.Name != "stdjava" {
		t.Fatalf("Expected stdjava array type, got %#v", arrayType.X)
	}
}

func findTypeParam(t *testing.T, params []symbol.TypeParam, name string) symbol.TypeParam {
	t.Helper()
	for _, p := range params {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("Could not find type parameter %q", name)
	return symbol.TypeParam{}
}

func TestSymbolParsing_TypeParameterBounds_ClassAndMethod(t *testing.T) {
	src := `
package com.example;
public class Bounds<T extends Number & Comparable<T>, U> {
    public <R extends Foo & Bar> R id(R value) {
        return value;
    }
}
`
	helper := setupParseHelper(t, src)

	classParams := helper.Ctx.currentClass.TypeParameters
	if len(classParams) != 2 {
		t.Fatalf("Expected 2 class type parameters, got %d", len(classParams))
	}

	tpT := findTypeParam(t, classParams, "T")
	if len(tpT.Bounds) != 2 {
		t.Fatalf("Expected 2 bounds for T, got %d", len(tpT.Bounds))
	}
	if tpT.Bounds[0].Original != "Number" || tpT.Bounds[1].Original != "Comparable<T>" {
		t.Fatalf("Unexpected bounds for T: %#v", tpT.Bounds)
	}

	tpU := findTypeParam(t, classParams, "U")
	if len(tpU.Bounds) != 0 {
		t.Fatalf("Expected no bounds for U, got %#v", tpU.Bounds)
	}

	methodDefs := helper.Ctx.currentClass.FindMethod().ByOriginalName("id")
	if len(methodDefs) != 1 {
		t.Fatalf("Expected to find method 'id', got %d matches", len(methodDefs))
	}
	methodParams := methodDefs[0].TypeParameters
	if len(methodParams) != 1 {
		t.Fatalf("Expected 1 method type parameter, got %d", len(methodParams))
	}

	tpR := findTypeParam(t, methodParams, "R")
	if len(tpR.Bounds) != 2 {
		t.Fatalf("Expected 2 bounds for R, got %d", len(tpR.Bounds))
	}
	if tpR.Bounds[0].Original != "Foo" || tpR.Bounds[1].Original != "Bar" {
		t.Fatalf("Unexpected bounds for R: %#v", tpR.Bounds)
	}
}
