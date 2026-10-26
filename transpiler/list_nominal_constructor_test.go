package transpiler

import (
	"bytes"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestListCanonicalConstructorProducer(t *testing.T) {
	for _, row := range []struct{ expression, want string }{
		{`new java.util.ArrayList<Object>()`, "NewArrayList"},
		{`new java.util.ArrayList<Object>(3)`, "NewArrayListWithArgument"},
		{`new java.util.LinkedList<Object>()`, "NewLinkedList"},
		{`new java.util.LinkedList<Object>(source)`, "NewLinkedListFromCollection"},
	} {
		t.Run(row.expression, func(t *testing.T) {
			code := `class Use{java.util.List<Object> source;Object f(){return ` + row.expression + `;}}`
			h := setupParseHelper(t, code)
			node := findNode(h.File.Ast, "object_creation_expression")
			if node == nil {
				t.Fatal("missing actual Java constructor")
			}
			got := ParseExpr(node, h.File.Source, h.Ctx)
			var text bytes.Buffer
			if err := format.Node(&text, token.NewFileSet(), got); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text.String(), "stdjava."+row.want+"[") {
				t.Fatalf("canonical declaration lost at allocation: %s want %s", text.String(), row.want)
			}
		})
	}
}

func TestListCanonicalConstructorOwnerShields(t *testing.T) {
	for _, row := range []struct {
		source string
		want   bool
	}{
		{`import java.util.ArrayList;class Use{Object f(){return new ArrayList<Object>();}}`, true},
		{`import java.util.*;class Use{Object f(){return new LinkedList<Object>();}}`, true},
		{`class ArrayList<T>{} class Use{Object f(){return new ArrayList<Object>();}}`, false},
		{`class LinkedList<T>{} class Use{Object f(){return new LinkedList<Object>();}}`, false},
	} {
		h := setupParseHelper(t, row.source)
		node := findNode(h.File.Ast, "object_creation_expression")
		got := ParseExpr(node, h.File.Source, h.Ctx)
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), got); err != nil {
			t.Fatal(err)
		}
		canonical := strings.Contains(out.String(), "stdjava.NewArrayList[") || strings.Contains(out.String(), "stdjava.NewLinkedList[")
		if canonical != row.want {
			t.Fatalf("owner allocation %s canonical=%v want %v", out.String(), canonical, row.want)
		}
	}
	for _, name := range []string{"ArrayList", "LinkedList"} {
		h := setupParseHelper(t, "class Use {}")
		owner, recognized := canonicalIntrinsicOwner("foreign."+name, h.Ctx)
		if !recognized || owner != "foreign."+name || intrinsicOwnerSupported(owner) {
			t.Fatalf("foreign %s borrowed JDK declaration: %s/%v", name, owner, recognized)
		}
	}
}

func TestListLinkedListRejectsCapacityConstructor(t *testing.T) {
	diagnostics.mu.Lock()
	previousStrict, previousItems := diagnostics.strict, diagnostics.items
	diagnostics.strict, diagnostics.items = true, nil
	diagnostics.mu.Unlock()
	defer func() {
		diagnostics.mu.Lock()
		diagnostics.strict, diagnostics.items = previousStrict, previousItems
		diagnostics.mu.Unlock()
	}()
	var observed any
	func() {
		defer func() { observed = recover() }()
		h := setupParseHelper(t, `class Use {Object f(){return new java.util.LinkedList<Object>(3);}}`)
		node := findNode(h.File.Ast, "object_creation_expression")
		ParseExpr(node, h.File.Source, h.Ctx)
	}()
	failure, ok := observed.(strictModeError)
	if !ok || !strings.Contains(failure.diagnostic.Kind, "LinkedList capacity overload") {
		t.Fatalf("nonexistent LinkedList capacity constructor was accepted: %v", observed)
	}
	if len(Diagnostics()) != 1 {
		t.Fatalf("wrong rejection diagnostics: %v", Diagnostics())
	}
}
