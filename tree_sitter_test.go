package main

import (
	"go/ast"
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
