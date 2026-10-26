package transpiler

import (
	"strings"
	"testing"
)

func TestSourceArrayListP1StorageDefaultsIdentity(t *testing.T) {
	out := renderGoFileFromJava(t, `class Values extends java.util.ArrayList<Object>{Object item; Values(){this(null);} Values(Object value){item=value;} Object read(){return get(0);} int count(){return size();}} class Child extends Values{Child(){super();}}`)
	for _, want := range []string{"*stdjava.List[any]", "stdjava.NewArrayList[any]", "stdjava.BindArrayListSourceReference", "stdjava.ArrayListSourceMethods", "CollectionAddJava2goExecution", "ArrayListBaseAddExecution", "ArrayListBaseGetExecution", "ArrayListBaseSizeExecution", "IteratorJava2goExecution", "JavaObjectInfo", "java.util.ArrayList", "java.util.List", "java.util.Collection", "java.lang.Iterable"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s\n%s", want, out)
		}
	}
	for _, bad := range []string{"*ArrayList", ".get(", ".size(", "stdjava.NewList"} {
		if strings.Contains(out, bad) {
			t.Errorf("unlowered native parent/default %s", bad)
		}
	}
	if strings.Count(out, "stdjava.NewArrayList[any]") != 1 || strings.Count(out, "stdjava.BindArrayListSourceReference(") != 1 {
		t.Fatal("this/source constructor chain allocates or binds native storage more than once")
	}
}

func TestSourceArrayListP1SuperKeepsSourceBody(t *testing.T) {
	out := renderGoFileFromJava(t, `class Parent extends java.util.ArrayList<Object>{public boolean add(Object v){return super.add(v);}} class Child extends Parent{public boolean add(Object v){return super.add(v);}}`)
	if strings.Count(out, "stdjava.ArrayListBaseAddExecution(") != 1 {
		t.Fatal("child super skipped the intervening source add body")
	}
	if !strings.Contains(out, ".Parent.AddJava2goExecution(") && !strings.Contains(out, ".parent.AddJava2goExecution(") {
		t.Fatalf("child super failed to select actual source parent\n%s", out)
	}
}
