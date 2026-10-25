package transpiler

import (
	"strings"
	"testing"
)

func TestAbstractListDeclarationOwnership(t *testing.T) {
	for _, row := range []struct{ name, source string }{
		{"generic", `class Values<E> extends java.util.AbstractList<E>{public E get(int i){return null;}public int size(){return 0;}}`},
		{"concrete", `class Values extends java.util.AbstractList<String>{public String get(int i){return "a";}public int size(){return 1;}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			scope := h.Ctx.currentFile.TopLevelClasses[0]
			if sourceAbstractListRoot(scope, h.Ctx) != scope {
				t.Fatal("actual source parent not admitted")
			}
			copy := *scope
			if sourceAbstractListRoot(&copy, h.Ctx) != nil {
				t.Fatal("detached metadata borrowed owned parent")
			}
			old := scope.Superclass
			scope.Superclass = "java.util.AbstractList<Object>"
			if sourceAbstractListRoot(scope, h.Ctx) != nil {
				t.Fatal("altered parent sidecar acquired a different native view")
			}
			scope.Superclass = old
		})
	}
}
func TestAbstractListAndRandomAccessHomonymsDecline(t *testing.T) {
	for _, source := range []string{
		`class AbstractList<E>{}class Probe extends AbstractList<String>{}`,
		`import foreign.AbstractList;class Probe extends AbstractList<String>{}`,
		`class java {static class util {static class AbstractList<E>{}}}class Probe extends java.util.AbstractList<String>{}`,
		`interface RandomAccess {}class Probe implements RandomAccess{}`,
		`class Probe<RandomAccess>{RandomAccess value;}`,
	} {
		out := renderGoFileFromJava(t, source)
		for _, helper := range []string{"NewAbstractListProjection", "BindAbstractListSourceReference", "AbstractListStateJava2go"} {
			if strings.Contains(out, helper) {
				t.Fatalf("homonym borrowed native role %s: %s", helper, out)
			}
		}
	}
}
func TestAbstractListErasedSlotsKeepActualBodyAndLateRead(t *testing.T) {
	out := renderGoFileFromJava(t, `class Values<E> extends java.util.AbstractList<E>{java.util.ArrayList<E> data=new java.util.ArrayList<E>();int reads,writes;public int size(){return data.size();}public E get(int index){reads++;return data.get(index);}public E set(int index,E value){writes++;return data.set(index,value);}}`)
	for _, text := range []string{"NewAbstractListProjection[E]", "BindAbstractListSourceReference", "CollectionListGetJava2goExecution", "CollectionListSetJava2goExecution"} {
		if !strings.Contains(out, text) {
			t.Fatalf("missing native source role %s: %s", text, out)
		}
	}
	if strings.Contains(out, "*AbstractList[E]") {
		t.Fatal("undefined platform superclass retained")
	}
	if !strings.Contains(out, "value any") || !strings.Contains(out, "reads++") || !strings.Contains(out, "writes++") {
		t.Fatalf("erased source body or effects lost: %s", out)
	}
}

func TestAbstractListLocalOwnershipRefusesCopies(t *testing.T) {
	ctx, frame, _ := lateSourceOwnershipFixture(t, `class LateOwned{void produce(){class Values extends java.util.AbstractList<Object>{public Object get(int i){return null;}public int size(){return 0;}Object read(){return get(0);}}new Values().read();}}`)
	scope := ctx.currentClass
	if sourceAbstractListRoot(scope, ctx) != scope {
		t.Fatal("actual hoisted local refused")
	}
	detached := *scope
	if sourceAbstractListRoot(&detached, ctx) != nil {
		t.Fatal("detached local borrowed source view")
	}
	registration := frame.late[scope]
	copied := *registration.local
	donated := ctx.Clone()
	donated.localClasses = map[string]*localClassInfo{registration.name: &copied}
	if sourceAbstractListRoot(scope, donated) != nil || !frame.active || sourceAbstractListRoot(scope, ctx) != scope {
		t.Fatal("copied registry acquired authority or poisoned actual producer")
	}
	foreign := ctx.Clone()
	foreign.currentFile = nil
	if sourceAbstractListRoot(scope, foreign) != nil {
		t.Fatal("foreign caller file borrowed local view")
	}
	before := scope.Superclass
	scope.Superclass = "java.util.AbstractList<String>"
	if sourceAbstractListRoot(scope, ctx) != nil {
		t.Fatal("altered local header metadata acquired native view")
	}
	scope.Superclass = before
	if sourceAbstractListRoot(scope, ctx) != scope {
		t.Fatal("restored actual local header refused")
	}
	frame.close()
	if sourceAbstractListRoot(scope, ctx) != nil {
		t.Fatal("retired synthesis witness acquired native view")
	}
}

func TestAbstractListAnonymousAllocationExplicitlyRefuses(t *testing.T) {
	strictRoutingState(t)
	var failure any
	func() {
		defer func() { failure = recover() }()
		renderGoFileFromJava(t, `class Probe{Object run(){return new java.util.AbstractList<Object>(){public Object get(int i){return null;}public int size(){return 0;}};}}`)
	}()
	typed, ok := failure.(strictModeError)
	if !ok || typed.diagnostic.Kind != "intrinsic invocation" || typed.diagnostic.NodeType != "object_creation_expression" || len(Diagnostics()) != 1 {
		t.Fatalf("unexpected unsupported anonymous contract: %T %v", failure, failure)
	}
}
