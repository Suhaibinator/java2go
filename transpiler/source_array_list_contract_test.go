package transpiler

import (
	"testing"
)

func TestSourceArrayListP1CanonicalDeclaration(t *testing.T) {
	for _, source := range []string{
		`class Values extends java.util.ArrayList<java.lang.Object>{}`,
		`import java.util.ArrayList; class Values extends ArrayList<Object>{}`,
		`class Parent extends java.util.ArrayList{} class Values extends Parent{}`,
	} {
		h := setupParseHelper(t, source)
		scope := h.Ctx.currentFile.FindClassScope("Values")
		if sourceArrayListRoot(scope, h.Ctx) == nil {
			t.Fatalf("canonical source root refused: %s", source)
		}
		detached := *scope
		if sourceArrayListRoot(&detached, h.Ctx) != nil {
			t.Fatal("detached metadata acquired source authority")
		}
	}
}

func TestSourceArrayListP1Refusals(t *testing.T) {
	for _, source := range []string{
		`class ArrayList<T>{} class Values extends ArrayList<Object>{}`,
		`class Object{} class Values extends java.util.ArrayList<Object>{}`,
		`class Values<E> extends java.util.ArrayList<E>{}`,
		`class Values extends java.util.ArrayList<String>{}`,
		`class Parent extends java.util.ArrayList<Object>{} class Values<E> extends Parent{}`,
		`class Values extends java.util.ArrayList<Object>{public int size(){return 1;}}`,
		`class Values extends java.util.ArrayList<Object>{public Object get(int n){return null;}}`,
		`class Values extends java.util.ArrayList<Object>{public java.util.Iterator<Object> iterator(){return null;}}`,
		`class Values extends java.util.ArrayList<Object>{public void clear(){}}`,
		`abstract class Values extends java.util.ArrayList<Object>{public abstract Object get(int n);}`,
		`abstract class Values extends java.util.ArrayList<Object>{public abstract boolean add(Object v);}`,
		`class Values extends java.util.ArrayList<Object>{public <T> T[] toArray(T[] a){return a;}}`,
		`class Values extends java.util.ArrayList<Object>{Values(){super(3);}}`,
	} {
		h := setupParseHelper(t, source)
		scope := h.Ctx.currentFile.FindClassScope("Values")
		before := len(scope.Methods)
		if sourceArrayListRoot(scope, h.Ctx) != nil {
			t.Fatalf("unproved source root admitted: %s", source)
		}
		if len(scope.Methods) != before {
			t.Fatal("refusal mutated method declarations")
		}
	}
}

func TestSourceArrayListP1SelectedMembers(t *testing.T) {
	h := setupParseHelper(t, `class Values extends java.util.ArrayList<Object>{public boolean add(Object v){return super.add(v);} int size(int x){return x;} Object read(int x){return get(x);} int count(){return size();}}`)
	owner := h.Ctx.currentFile.FindClassScope("Values")
	prepareAbstractCollectionDefaultsForScope(owner, h.Ctx)
	for _, row := range []struct{ caller, name, result string }{{"read", "get", "java.lang.Object"}, {"count", "size", "int"}} {
		caller := owner.FindMethodByName(row.caller, nil)
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = caller
		invocation := findNode(caller.DeclarationNode, "method_invocation")
		selected := findBestMethodInHierarchy(owner, row.name, invocation.ChildByFieldName("arguments"), true, false, ctx, h.File.Source)
		if selected == nil || !selected.def.RuntimeDefault || selected.def.OriginalType != row.result {
			t.Fatalf("inherited %s lacks platform declaration: %#v", row.name, selected)
		}
	}
	method, declaring := sourceArrayListSelectedAdd(owner, h.Ctx)
	if method == nil || declaring != owner || method != owner.FindMethodByName("add", nil) || method.RuntimeDefault {
		t.Fatal("actual source add override was bypassed")
	}
	if sourceArrayListRoot(owner, h.Ctx) == nil {
		t.Fatal("ordinary business overload refused")
	}
}

func TestSourceArrayListP1MalformedMethodRefusal(t *testing.T) {
	h := setupParseHelper(t, `class Values extends java.util.ArrayList<Object>{public boolean add(Object value){return true;}}`)
	scope := h.Ctx.currentFile.FindClassScope("Values")
	method := scope.FindMethodByName("add", nil)
	method.Parameters[0].OriginalType = "String"
	if sourceArrayListRoot(scope, h.Ctx) != nil {
		t.Fatal("altered parameter metadata retained platform override authority")
	}
}

func TestSourceArrayListP1BusinessOverloads(t *testing.T) {
	h := setupParseHelper(t, `class Values extends java.util.ArrayList<Object>{public boolean add(String value){return false;} public Object get(String key){return key;} int size(int n){return n;}}`)
	if sourceArrayListRoot(h.Ctx.currentFile.FindClassScope("Values"), h.Ctx) == nil {
		t.Fatal("non-overriding business overload was refused")
	}
}
