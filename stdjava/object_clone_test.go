package stdjava

import "testing"

func TestObjectCloneArrayRuntime(t *testing.T) {
	p := PrimitiveArrayLiteral(PrimitiveIntTypeID, int32(1), int32(2))
	cp := ObjectCloneExecution(NewExecution(), p).(*PrimitiveArray[int32])
	cp.Elements[0] = 8
	if cp == p || p.Elements[0] != 1 || cp.JavaArrayTypeID() != p.JavaArrayTypeID() {
		t.Fatal("primitive clone must retain descriptor and independent elements")
	}
	child := NewReferenceArray(0, ObjectTypeID)
	a := ReferenceArrayLiteral(ArrayTypeID(ObjectTypeID), child)
	ca := ObjectCloneExecution(NewExecution(), a).(*ReferenceArray)
	if ca == a || ca.componentType != a.componentType || ca.elements[0] != child {
		t.Fatal("reference clone is shallow")
	}
	ca.elements[0] = nil
	if a.elements[0] != child {
		t.Fatal("array storage aliased")
	}
	empty := NewPrimitiveArray[int32](0, PrimitiveIntTypeID)
	if ObjectCloneExecution(NewExecution(), empty) == empty {
		t.Fatal("empty arrays need fresh identity")
	}
}
func TestObjectCloneFailureNominalType(t *testing.T) {
	defer func() {
		r := recover()
		e, ok := r.(CloneNotSupportedException)
		if !ok {
			t.Fatalf("wrong failure %T", r)
		}
		if e.Message() != "java.lang.String" || e.JavaDynamicTypeID() != "java.lang.CloneNotSupportedException" || !CaughtAs(e, "Exception") {
			t.Fatalf("wrong clone failure: %v", e)
		}
	}()
	ObjectCloneExecution(NewExecution(), NewJavaStringUTF16([]uint16{120}))
	t.Fatal("non Cloneable was cloned")
}
