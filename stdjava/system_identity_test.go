package stdjava

import (
	"reflect"
	"testing"
)

type systemIdentityOverrideProbe struct {
	*ObjectInfo
	calls int
}

func (p *systemIdentityOverrideProbe) HashCodeJava2goExecution(*Execution) int32 {
	p.calls++
	return 71
}
func TestSystemIdentityHashCodeBypassesVirtualMethods(t *testing.T) {
	e := NewExecution()
	object := &systemIdentityOverrideProbe{ObjectInfo: NewObjectInfo("identity.Override", nil)}
	first := SystemIdentityHashCode(object)
	if first != SystemIdentityHashCode(object.ObjectInfo) || first != SystemIdentityHashCode(object) || object.calls != 0 {
		t.Fatal("identity hash called virtual code or changed across superclass views")
	}
	if ObjectHashCodeExecution(e, object) != 71 || object.calls != 1 {
		t.Fatal("ordinary Object hash lost virtual override")
	}
	text := NewJavaStringUTF16([]uint16{'a'})
	pointer := uint64(reflect.ValueOf(text).Pointer())
	if got := SystemIdentityHashCode(text); got != int32(pointer^(pointer>>32)) {
		t.Fatal("identity hash used String value hash")
	}
	for _, value := range []any{nil, (*JavaString)(nil), (*PrimitiveArray[int32])(nil)} {
		if SystemIdentityHashCode(value) != 0 {
			t.Fatal("null identity hash is not zero")
		}
	}
	for _, value := range []any{NewPrimitiveArray[int32](0, PrimitiveTypeID("int")), NewReferenceArray(0, ObjectTypeID), NewMap[any, any]()} {
		if got := SystemIdentityHashCode(value); got != SystemIdentityHashCode(value) {
			t.Fatal("reference identity changed")
		}
	}
	array := NewPrimitiveArray[int32](0, PrimitiveTypeID("int"))
	if SystemIdentityHashCode(array) != ObjectHashCodeExecution(e, array) {
		t.Fatal("array default hash and identity hash diverged")
	}
	expectBoxedException(t, "ClassCastException", func() { SystemIdentityHashCode(int32(7)) })
}
