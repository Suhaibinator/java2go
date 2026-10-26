package stdjava

import "testing"

func TestThreadLocalCanonicalDeclarationModifiers(t *testing.T) {
	calls := 0
	plain := NewThreadLocalReference()
	supplied := ThreadLocalWithInitialReference(NewSupplierFuncAdapter(func(*Execution) any {
		calls++
		return nil
	}))
	if got := ObjectGetClass(plain).GetModifiers(); got != 1 {
		t.Errorf("plain ThreadLocal modifiers=%d, want public=1", got)
	}
	if got := ObjectGetClass(supplied).GetModifiers(); got != 24 {
		t.Errorf("SuppliedThreadLocal modifiers=%d, want static|final=24", got)
	}
	if calls != 0 {
		t.Fatal("class modifier observation invoked initializer")
	}
	if ObjectGetClass(NewThreadLocal[int32]()).GetModifiers() != 1 {
		t.Fatal("legacy native factory modifier fallback changed")
	}
}
