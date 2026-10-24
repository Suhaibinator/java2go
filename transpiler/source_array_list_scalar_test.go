package transpiler

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func sourceArrayListP2GuardFile(t *testing.T, source string) (file *ast.File) {
	t.Helper()
	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("P2 actual AST emission remains refused: %T %v", failure, failure)
		}
	}()
	strictRoutingState(t)
	output := renderGoFileFromJava(t, source)
	var err error
	file, err = parser.ParseFile(token.NewFileSet(), "p2.go", output, 0)
	if err != nil {
		t.Fatalf("invalid emitted Go AST: %v\n%s", err, output)
	}
	return file
}
func sourceArrayListP2GuardText(node ast.Node) string {
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), node); err != nil {
		panic(err)
	}
	return out.String()
}
func sourceArrayListP2GuardMethod(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && strings.EqualFold(fn.Name.Name, name) {
			return fn
		}
	}
	t.Fatalf("missing generated method %s", name)
	return nil
}

func TestSourceArrayListP2CanonicalAdmission(t *testing.T) {
	for _, row := range []struct{ name, source string }{
		{"generic", `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`},
		{"renamed_binder", `class Bag<X> extends java.util.ArrayList<X>{int count;public boolean add(X item){count++;return super.add(item);}}`},
		{"String", `final class Values extends java.util.ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			owner := h.Ctx.currentFile.TopLevelClasses[0]
			if sourceArrayListRoot(owner, h.Ctx) == nil {
				t.Fatal("actual canonical scalar ArrayList declaration remains refused")
			}
			detached := *owner
			if sourceArrayListRoot(&detached, h.Ctx) != nil {
				t.Fatal("detached source metadata acquired scalar ArrayList authority")
			}
		})
	}
}

func TestSourceArrayListP2ErasedBodyAndNativeSlot(t *testing.T) {
	file := sourceArrayListP2GuardFile(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`)
	output := sourceArrayListP2GuardText(file)
	if !strings.Contains(output, "*stdjava.List[E]") || !strings.Contains(output, "stdjava.NewArrayList[E]") || strings.Contains(output, "stdjava.NewList") {
		t.Fatalf("logical E lost invariant actual native parent/allocation\n%s", output)
	}
	found := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		body := sourceArrayListP2GuardText(fn.Body)
		if !strings.Contains(body, "ArrayListBaseAddExecution(") || !strings.Contains(strings.ToLower(body), "calls++") {
			continue
		}
		found++
		if len(fn.Type.Params.List) < 2 || sourceArrayListP2GuardText(fn.Type.Params.List[len(fn.Type.Params.List)-1].Type) != "any" {
			t.Fatalf("generic original add body narrows E before effects: %s", sourceArrayListP2GuardText(fn))
		}
		if strings.Contains(body, "ObjectView[") || strings.Contains(body, ".(E)") {
			t.Fatalf("generic add body casts before raw super storage: %s", body)
		}
	}
	if found != 1 {
		t.Fatalf("expected exactly one original erased add body with counter and nonvirtual native store, got %d", found)
	}
}

func TestSourceArrayListP2LateReadConsumer(t *testing.T) {
	file := sourceArrayListP2GuardFile(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} class Reads{Object raw(Values<String> values){return values.get(0);} String typed(Values<String> values){return values.get(0);}}`)
	raw := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "RawJava2goExecution").Body)
	typed := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "TypedJava2goExecution").Body)
	if strings.Contains(raw, "ObjectView[") || strings.Contains(raw, "StringTypeID") {
		t.Fatalf("Object consumer narrows erased get before Java checkcast boundary: %s", raw)
	}
	if !strings.Contains(typed, "ObjectView[") || !strings.Contains(typed, "StringTypeID") {
		t.Fatalf("typed String consumer lost its delayed checkcast: %s", typed)
	}
}

func TestSourceArrayListP2StringPreBodyBridge(t *testing.T) {
	file := sourceArrayListP2GuardFile(t, `final class Values extends java.util.ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`)
	bridge := sourceArrayListP2GuardMethod(t, file, "CollectionAddJava2goExecution")
	body := sourceArrayListP2GuardText(bridge.Body)
	cast := strings.Index(body, "ObjectView[")
	if cast < 0 || !strings.Contains(body, "StringTypeID") || strings.Count(body, "ObjectView[") != 1 {
		t.Fatalf("raw String alias bridge lacks exactly one pre-body String checkcast: %s", body)
	}
	call := strings.LastIndex(body, ".Add")
	if call < 0 || cast > call && !strings.Contains(body[call:], "ObjectView[") {
		t.Fatalf("String bridge checkcast does not precede source body entry: %s", body)
	}
	if strings.Contains(strings.ToLower(body), "calls++") || strings.Contains(body, "ArrayListBaseAddExecution(") {
		t.Fatalf("raw bridge bypasses/nests source counter body: %s", body)
	}
}

func TestSourceArrayListP2KeepsTreeMapGate(t *testing.T) {
	h := setupParseHelper(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`)
	owner := h.Ctx.currentFile.FindClassScope("Values")
	method := owner.FindMethodByName("add", nil)
	if sourceNativeVirtualBodyForMethod(owner, method, h.Ctx) == nil {
		t.Fatal("owned ArrayList scalar body positive missing")
	}
	other := setupParseHelper(t, `final class Values<E> extends java.util.TreeMap<E,E>{}`)
	held := other.Ctx.currentFile.FindClassScope("Values")
	if sourceArrayListRoot(held, other.Ctx) != nil {
		t.Fatal("ArrayList source admission donated its parent to TreeMap")
	}
	for _, method := range held.Methods {
		if sourceNativeVirtualBodyForMethod(held, method, other.Ctx) != nil {
			t.Fatal("ArrayList-only body capability admitted TreeMap")
		}
	}
}

func TestSourceArrayListP2ScalarRefusalFrontier(t *testing.T) {
	for _, source := range []string{
		`class Values<E extends Number> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`,
		`class Values<E> extends java.util.ArrayList<E>{E held;public boolean add(E value){return super.add(value);}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){java.util.List<E> nested=null;return super.add(value);}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);} E other(E value){return value;}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;public <E> boolean add(E value){calls++;return super.add(value);}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return true;}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;Values(){super(3);}public boolean add(E value){calls++;return super.add(value);}}`,
		`class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} class Child extends Values<String>{}`,
		`class String{} final class Values extends java.util.ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`,
		`class ArrayList<E>{} class Values<E> extends ArrayList<E>{int calls;public boolean add(E value){calls++;return true;}}`,
	} {
		h := setupParseHelper(t, source)
		owner := h.Ctx.currentFile.FindClassScope("Values")
		if sourceArrayListRoot(owner, h.Ctx) != nil {
			t.Fatalf("unsupported scalar declaration acquired source/native authority: %s", source)
		}
	}
}

func TestSourceArrayListP2ScalarDefaultGetAuthority(t *testing.T) {
	h := setupParseHelper(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`)
	owner := h.Ctx.currentFile.FindClassScope("Values")
	prepareAbstractCollectionDefaultsForScope(owner, h.Ctx)
	method := owner.FindMethodByName("get", nil)
	if method == nil {
		t.Fatal("actual inherited scalar get missing")
	}
	if _, ok := sourceArrayListScalarGetErasure(owner, method, h.Ctx); !ok {
		t.Fatal("actual generated scalar get refused")
	}
	copied := *method
	if _, ok := sourceArrayListScalarGetErasure(owner, &copied, h.Ctx); ok {
		t.Fatal("detached get metadata acquired erasure authority")
	}
	before := len(owner.Methods)
	owner.Methods = append(owner.Methods, &copied)
	defer func() { owner.Methods = owner.Methods[:before] }()
	if _, ok := sourceArrayListScalarGetErasure(owner, method, h.Ctx); ok {
		t.Fatal("duplicate attached get declarations retained erasure authority")
	}
	if _, ok := sourceArrayListScalarGetErasure(owner, &copied, h.Ctx); ok {
		t.Fatal("copied duplicate get metadata acquired erasure authority")
	}

}

func TestSourceArrayListP2ScalarBodyCertificate(t *testing.T) {
	h := setupParseHelper(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`)
	owner := h.Ctx.currentFile.FindClassScope("Values")
	method := owner.FindMethodByName("add", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	plan := sourceArrayListScalarBody(owner, method, ctx)
	if plan == nil {
		t.Fatal("actual scalar source body certificate missing")
	}
	ctx.sourceNativeVirtualBody = plan
	if got := sourceArrayListP2GuardText(javaTypeStringToGoTypeExpr("E", inScopeTypeParameters(ctx), ctx)); got != "any" {
		t.Fatalf("owned scalar body E=%s", got)
	}
	if got := sourceArrayListP2GuardText(sourceArrayListStorageTypeArgument(owner, ctx)); got != "E" {
		t.Fatalf("body erasure leaked into invariant native storage E=%s", got)
	}
	copied := *method
	if sourceArrayListScalarBody(owner, &copied, ctx) != nil {
		t.Fatal("copied add metadata acquired scalar body authority")
	}
	old := method.Parameters[0].DirectTypeParameter
	method.Parameters[0].DirectTypeParameter = nil
	if sourceArrayListScalarBody(owner, method, ctx) != nil {
		t.Fatal("wrong attached formal binder retained scalar body authority")
	}
	method.Parameters[0].DirectTypeParameter = old
}

func TestSourceArrayListP2CallerHomonymsKeepCanonicalString(t *testing.T) {
	file := sourceArrayListP2GuardFile(t, `class String{} class Object{} class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} class Reads{java.lang.Object raw(Values<java.lang.String> values){return values.get(0);} java.lang.String typed(Values<java.lang.String> values){return values.get(0);}}`)
	raw := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "RawJava2goExecution").Body)
	typed := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "TypedJava2goExecution").Body)
	if strings.Contains(raw, "ObjectView[") || !strings.Contains(typed, "StringTypeID") || strings.Contains(typed, `TypeID("String")`) {
		t.Fatalf("caller homonyms changed canonical get consumer: raw=%s typed=%s", raw, typed)
	}
}

func TestSourceArrayListP2InheritedFieldElementOrigin(t *testing.T) {
	file := sourceArrayListP2GuardFile(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} class Parent{static class Item{} Values<Item> values;} class Child extends Parent{static class Item{} Parent.Item read(){return values.get(0);}}`)
	body := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "ReadJava2goExecution").Body)
	if !strings.Contains(body, "Parent$Item") || strings.Contains(body, "Child$Item") {
		t.Fatalf("inherited field element was recaptured by caller homonym: %s", body)
	}
}

func TestSourceArrayListP2ReorderedReceiverBinderOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Values<E> extends java.util.ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}} class Parent<A,B>{Values<B> values;} class Child<X,Y> extends Parent<Y,X>{Object read(){return values.get(0);}}`)
	values := h.Ctx.currentFile.FindClassScope("Values")
	prepareAbstractCollectionDefaultsForScope(values, h.Ctx)
	child := h.Ctx.currentFile.FindClassScope("Child")
	method := child.FindMethodByName("read", nil)
	ctx := classScopeCtx(child, h.Ctx)
	ctx.localScope = method
	invocation := findNode(method.DeclarationNode, "method_invocation")
	target, resolution := resolvedInstanceInvocation(invocation, ctx, h.File.Source)
	view, erasure, ok := sourceArrayListScalarResultView(target, resolution, ctx)
	if !ok || erasure != "java.lang.Object" {
		t.Fatalf("reordered receiver origin refused: view=%q erasure=%q", view, erasure)
	}
	actual := visibleTypeParameterDeclarationForJavaType(view, ctx)
	if actual != child.TypeParameters[0].Declaration {
		t.Fatalf("receiver B -> Child.X substitution lost binder identity: view=%q", view)
	}
}
