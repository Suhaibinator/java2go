package transpiler

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const sourceArrayListNativeGuardRoot = `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} `

// Exercise the existing conversion API at its real parsed source boundary.
// No future planner helper or metadata manufactured by the test is required.
func sourceArrayListNativeGuardCoerce(t *testing.T, source, ownerName, methodName, targetType string) (string, string) {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope(ownerName)
	method := owner.FindMethodByName(methodName, nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	var operand *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil || operand != nil {
			return
		}
		if node.Type() == "identifier" && node.Content(h.File.Source) == "operand" {
			operand = node
			return
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(method.DeclarationNode.ChildByFieldName("body"))
	if operand == nil {
		t.Fatal("actual operand boundary missing")
	}
	parsed := ast.NewIdent("actualSource")
	converted := coerceArgumentToExpectedType(parsed, operand, targetType, ctx, h.File.Source)
	required := javaTypeStringToGoTypeExpr(targetType, inScopeTypeParameters(ctx), ctx)
	return sourceArrayListP2GuardText(converted), sourceArrayListP2GuardText(required)
}

func TestSourceArrayListP2DeclaredNativeAliasView(t *testing.T) {
	for _, row := range []struct{ name, body, target string }{
		{"ArrayList_local", `java.util.ArrayList<java.lang.String> target=operand;`, "java.util.ArrayList<java.lang.String>"},
		{"List_local", `java.util.List<java.lang.String> target=operand;`, "java.util.List<java.lang.String>"},
		{"assignment", `java.util.List<java.lang.String> target=null; target=operand;`, "java.util.List<java.lang.String>"},
		{"return", `return operand;`, "java.util.List<java.lang.String>"},
	} {
		t.Run(row.name, func(t *testing.T) {
			result := "void"
			if row.name == "return" {
				result = row.target
			}
			source := sourceArrayListNativeGuardRoot + `class Use{` + result + ` run(Values<java.lang.String> operand){` + row.body + `}}`
			got, required := sourceArrayListNativeGuardCoerce(t, source, "Use", "run", row.target)
			if !strings.Contains(got, "ObjectView["+required+"]") || !strings.Contains(got, "java.util.") {
				t.Fatalf("declared canonical native alias retains source pointer instead of exact parent provider: got=%s required=%s", got, required)
			}
		})
	}
}

func TestSourceArrayListP2DeclaredNativeAliasElementOrigins(t *testing.T) {
	for _, row := range []struct{ name, source, target string }{
		{"inherited_Item", sourceArrayListNativeGuardRoot + `class Parent{static class Item{} Values<Item> operand;java.util.List<Item> target;} class Child extends Parent{static class Item{} void run(){target=operand;}}`, "java.util.List<Parent.Item>"},
		{"reordered_binder", sourceArrayListNativeGuardRoot + `class Parent<A,B>{Values<B> operand;java.util.List<B> target;} class Child<X,Y> extends Parent<Y,X>{void run(){target=operand;}}`, "java.util.List<X>"},
	} {
		t.Run(row.name, func(t *testing.T) {
			got, required := sourceArrayListNativeGuardCoerce(t, row.source, "Child", "run", row.target)
			if !strings.Contains(got, "ObjectView["+required+"]") {
				t.Fatalf("declared source/target element origin lost native view: got=%s required=%s", got, required)
			}
		})
	}
}

func TestSourceArrayListP2NativeAliasHomonymAndInvariantGuards(t *testing.T) {
	for _, row := range []struct{ prefix, target string }{
		{`class List<T>{} `, "List<java.lang.String>"},
		{`class ArrayList<T>{} `, "ArrayList<java.lang.String>"},
		{"", "java.util.List<java.lang.Object>"},
		{"", "java.util.ArrayList"},
	} {
		source := sourceArrayListNativeGuardRoot + row.prefix + `class Use{void run(Values<java.lang.String> operand){` + row.target + ` target=operand;}}`
		got, _ := sourceArrayListNativeGuardCoerce(t, source, "Use", "run", row.target)
		if strings.Contains(got, "ObjectView[*stdjava.List[") {
			t.Fatalf("foreign/raw/invariant-mismatched target acquired native provider projection: %s", got)
		}
	}
}

func TestSourceArrayListP2NativeAliasProjectionOnceAndFieldOrigin(t *testing.T) {
	source := sourceArrayListNativeGuardRoot + `class Use{Values<java.lang.String> make(){return null;}void run(){java.util.List<java.lang.String> target=make();}}`
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	declaration := findNode(method.DeclarationNode, "variable_declarator")
	node := declaration.ChildByFieldName("value")
	value := &ast.CallExpr{Fun: ast.NewIdent("produceSourceOnce")}
	got := sourceArrayListP2GuardText(coerceArgumentToExpectedType(value, node, "java.util.List<java.lang.String>", ctx, h.File.Source))
	if strings.Count(got, "produceSourceOnce(") != 1 || !strings.Contains(got, "ObjectView[*stdjava.List[") {
		t.Fatalf("native parent projection copied or reevaluated source: %s", got)
	}
	h = setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use<E>{Values<E> operand;java.util.List<E> target=operand;}`)
	owner = h.Ctx.currentFile.FindClassScope("Use")
	ctx = classScopeCtx(owner, h.Ctx)
	ctx.localScope = &symbol.Definition{OriginalName: fieldInitMethodName}
	var field *symbol.Definition
	for _, definition := range owner.Fields {
		if definition.OriginalName == "target" {
			field = definition
		}
	}
	if field == nil {
		t.Fatal("actual field target missing")
	}
	node = findNode(field.DeclarationNode, "variable_declarator").ChildByFieldName("value")
	converted := coerceArgumentToExpectedType(ast.NewIdent("actualFieldSource"), node, field.OriginalType, ctx, h.File.Source)
	if got := sourceArrayListP2GuardText(converted); !strings.Contains(got, "ObjectView[*stdjava.List[E]]") {
		t.Fatalf("field declaration lost its owned E native parent: %s", got)
	}
}

func TestSourceArrayListP2NativeAliasSuffixArrayDimensions(t *testing.T) {
	for _, row := range []struct{ name, members, body string }{
		{"parameter_and_local", "", `void run(Values<java.lang.String> operand[]){java.util.List<java.lang.String> target[]=operand;}`},
		{"local_source_assignment", "", `void run(){Values<java.lang.String> operand[]=null;java.util.List<java.lang.String> target[]=null;target=operand;}`},
		{"field_source_assignment", `Values<java.lang.String> operand[];java.util.List<java.lang.String> target[];`, `void run(){target=operand;}`},
		{"return_suffix", "", `java.util.List<java.lang.String> run(Values<java.lang.String> operand[])[]{return operand;}`},
		{"source_return_suffix", `Values<java.lang.String> make()[]{return null;}`, `void run(){java.util.List<java.lang.String> target[]=make();}`},
		{"prefix_arrays", "", `void run(Values<java.lang.String>[] operand){java.util.List<java.lang.String>[] target=operand;}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use{`+row.members+row.body+`}`)
			owner := h.Ctx.currentFile.FindClassScope("Use")
			method := owner.FindMethodByName("run", nil)
			ctx := classScopeCtx(owner, h.Ctx)
			ctx.localScope = method
			var boundary *sitter.Node
			var visit func(*sitter.Node)
			visit = func(node *sitter.Node) {
				if node == nil {
					return
				}
				switch node.Type() {
				case "variable_declarator":
					name := node.ChildByFieldName("name")
					value := node.ChildByFieldName("value")
					if name != nil && name.Content(h.File.Source) == "target" && value != nil && value.Type() != "null_literal" {
						boundary = value
					}
				case "assignment_expression":
					boundary = node.ChildByFieldName("right")
				case "return_statement":
					boundary = nodeutil.SemanticNamedChild(node, 0)
				}
				for _, child := range nodeutil.NamedChildrenOf(node) {
					visit(child)
				}
			}
			visit(method.DeclarationNode.ChildByFieldName("body"))
			if boundary == nil {
				t.Fatal("actual array boundary absent")
			}
			if row.name != "prefix_arrays" && sourceArrayListNativeScalarExpression(boundary, ctx, h.File.Source) {
				t.Fatal("actual source suffix array was certified scalar")
			}
			got := coerceArgumentToExpectedType(ast.NewIdent("actualArray"), boundary, "java.util.List<java.lang.String>", ctx, h.File.Source)
			if strings.Contains(sourceArrayListP2GuardText(got), "ObjectView[*stdjava.List[") {
				t.Fatalf("suffix array acquired scalar native parent: %s", sourceArrayListP2GuardText(got))
			}
		})
	}
	for _, members := range []string{
		`Values<java.lang.String> operand[];java.util.List<java.lang.String> target[]=operand;`,
		`Values<java.lang.String> operand;java.util.List<java.lang.String> target[]=operand;`,
	} {
		h := setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use{`+members+`}`)
		owner := h.Ctx.currentFile.FindClassScope("Use")
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = &symbol.Definition{OriginalName: fieldInitMethodName}
		field := owner.FindFieldByName("target")
		node := findNode(field.DeclarationNode, "variable_declarator").ChildByFieldName("value")
		got := coerceArgumentToExpectedType(ast.NewIdent("actualField"), node, field.OriginalType, ctx, h.File.Source)
		if strings.Contains(sourceArrayListP2GuardText(got), "ObjectView[*stdjava.List[") {
			t.Fatal("field suffix array acquired scalar native projection")
		}
	}
	// A suffix on a different variable in the same declaration does not change
	// this scalar field's own rank.
	h := setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use{Values<java.lang.String> array[],operand;java.util.List<java.lang.String> target=operand;}`)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = &symbol.Definition{OriginalName: fieldInitMethodName}
	field := owner.FindFieldByName("target")
	node := findNode(field.DeclarationNode, "variable_declarator").ChildByFieldName("value")
	got := coerceArgumentToExpectedType(ast.NewIdent("scalarField"), node, field.OriginalType, ctx, h.File.Source)
	if !strings.Contains(sourceArrayListP2GuardText(got), "ObjectView[*stdjava.List[") {
		t.Fatal("unrelated declarator suffix poisoned scalar field")
	}
}

func TestSourceArrayListP2NativeNestedExecutableRefusal(t *testing.T) {
	for _, body := range []string{
		`void run(Values<java.lang.String> operand){Runnable task=()->{java.util.List<java.lang.String> target=operand;};}`,
		`void run(Values<java.lang.String> operand){class Inner{void nested(){java.util.List<java.lang.String> target=operand;}}}`,
	} {
		h := setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use{`+body+`}`)
		owner := h.Ctx.currentFile.FindClassScope("Use")
		method := owner.FindMethodByName("run", nil)
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = method
		var boundary *sitter.Node
		var visit func(*sitter.Node)
		visit = func(node *sitter.Node) {
			if node == nil {
				return
			}
			if node.Type() == "variable_declarator" {
				name := node.ChildByFieldName("name")
				if name != nil && name.Content(h.File.Source) == "target" {
					boundary = node.ChildByFieldName("value")
				}
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				visit(child)
			}
		}
		visit(method.DeclarationNode)
		if boundary == nil {
			t.Fatal("nested boundary absent")
		}
		got := coerceArgumentToExpectedType(ast.NewIdent("nestedSource"), boundary, "java.util.List<java.lang.String>", ctx, h.File.Source)
		if strings.Contains(sourceArrayListP2GuardText(got), "ObjectView[*stdjava.List[") {
			t.Fatal("nested executable borrowed outer method authority")
		}
	}
}

func TestSourceArrayListP2NativeFieldInitializerActualRendering(t *testing.T) {
	for _, row := range []struct {
		name, modifier, prefix, target string
		want                           bool
	}{
		{"instance", "", "", "java.util.List<java.lang.String>", true},
		{"static", "static ", "", "java.util.List<java.lang.String>", true},
		{"homonym", "", `class List<T>{} `, "List<java.lang.String>", false},
		{"target_suffix", "", "", "java.util.List<java.lang.String>", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			suffix := ""
			if row.name == "target_suffix" {
				suffix = "[]"
			}
			source := sourceArrayListNativeGuardRoot + row.prefix + `class Use{` + row.modifier + `Values<java.lang.String> operand;` + row.modifier + row.target + ` target` + suffix + `=operand;}`
			h := setupParseHelper(t, source)
			owner := h.Ctx.currentFile.FindClassScope("Use")
			ctx := classScopeCtx(owner, h.Ctx)
			// This existing emission API constructs the real synthetic initializer
			// scopes; a manually nil localScope cannot substitute for this regression.
			declarations := parseDecls(owner.Class.DeclarationNode, h.File.Source, ctx)
			rendered := sourceArrayListP2GuardText(&ast.File{Name: ast.NewIdent("example"), Decls: declarations})
			found := strings.Contains(rendered, "ObjectView[*stdjava.List[")
			if found != row.want {
				t.Fatalf("real field initializer projection=%v want=%v:\n%s", found, row.want, rendered)
			}
		})
	}
}

func TestSourceArrayListP2NativeFieldInitializerContextGuards(t *testing.T) {
	for _, modifier := range []string{"", "static "} {
		h := setupParseHelper(t, sourceArrayListNativeGuardRoot+`class Use{`+modifier+`Values<java.lang.String> operand;`+modifier+`java.util.List<java.lang.String> target=operand;}`)
		owner := h.Ctx.currentFile.FindClassScope("Use")
		ctx := classScopeCtx(owner, h.Ctx)
		field := owner.FindFieldByName("target")
		node := findNode(field.DeclarationNode, "variable_declarator").ChildByFieldName("value")
		expected := &symbol.Definition{OriginalName: fieldInitMethodName}
		wrong := &symbol.Definition{IsStatic: true}
		if modifier != "" {
			expected, wrong = wrong, expected
		}
		for _, local := range []*symbol.Definition{nil, wrong, {OriginalName: fieldInitMethodName, Name: "borrowed"}, {OriginalName: "sourceMethod"}} {
			ctx.localScope = local
			if _, ok := sourceArrayListNativeBoundaryConversion(ast.NewIdent("sourceField"), node, field.OriginalType, ctx, h.File.Source); ok {
				t.Fatal("unowned synthetic field context acquired projection")
			}
		}
		ctx.localScope = expected
		if _, ok := sourceArrayListNativeBoundaryConversion(ast.NewIdent("sourceField"), node, field.OriginalType, ctx, h.File.Source); !ok {
			t.Fatal("actual producer shape lost owned field projection")
		}
		field.IsStatic = !field.IsStatic
		if _, ok := sourceArrayListNativeBoundaryConversion(ast.NewIdent("sourceField"), node, field.OriginalType, ctx, h.File.Source); ok {
			t.Fatal("stale field static metadata acquired initializer authority")
		}
	}
}
