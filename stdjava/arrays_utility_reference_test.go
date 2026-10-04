package stdjava

import (
	"testing"
	"unsafe"
)

func TestJavaArraysNominalReferenceIdentity(t *testing.T) {
	if unsafe.Sizeof(JavaArrays{}) == 0 {
		t.Fatal("utility carrier has zero-sized identity")
	}
	first, second := &JavaArrays{}, &JavaArrays{}
	if JavaReferenceEqual(first, second) || !JavaReferenceEqual(first, first) {
		t.Fatal("nominal carrier identity")
	}
	var null *JavaArrays
	if !JavaReferenceEqual(null, nil) {
		t.Fatal("typed null lost")
	}
	if first.JavaDynamicTypeID() != TypeID("java.util.Arrays") {
		t.Fatal("utility Java type")
	}
	if !JavaTypeAssignable(TypeID("java.util.Arrays"), ObjectTypeID) {
		t.Fatal("utility Object parent")
	}
}
