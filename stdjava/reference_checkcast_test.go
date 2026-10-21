package stdjava

import (
	"sync"
	"testing"
)

func requireCheckCastFailure(t *testing.T, run func()) {
	t.Helper()
	defer func() {
		if value := recover(); !CaughtAs(value, "ClassCastException") {
			t.Fatalf("expected ClassCastException, got %T: %v", value, value)
		}
	}()
	run()
}

func TestReferenceCheckCastNativeIdentityAndNull(t *testing.T) {
	builder, buffer := NewStringBuilder(), NewStringBuffer()
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*StringBuilder](builder, "java.lang.StringBuffer") })
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*StringBuilder](buffer, "java.lang.StringBuilder") })
	if ReferenceCheckCast[*StringBuilder](buffer, "java.lang.StringBuffer") != buffer {
		t.Fatal("same-family pointer changed")
	}
	if ReferenceCheckCast[any](builder, CharSequenceTypeID) != builder {
		t.Fatal("interface view changed allocation")
	}
	var absent *StringBuilder
	for _, value := range []any{nil, absent} {
		if ReferenceCheckCast[*StringBuilder](value, "java.lang.StringBuffer") != nil {
			t.Fatal("null reference fabricated allocation")
		}
	}
	if !StringIsNull(ReferenceCheckCast[string](nil, StringTypeID)) {
		t.Fatal("legacy native string null sentinel lost")
	}
}

type checkCastSource struct{ *ObjectInfo }
type checkCastBase struct{ *ObjectInfo }

func TestReferenceCheckCastSourceProviderAndUnknownShield(t *testing.T) {
	const childID TypeID = "checkcast.source.Child"
	const baseID TypeID = "checkcast.source.Base"
	RegisterJavaType(baseID, ObjectTypeID)
	RegisterJavaType(childID, baseID)
	RegisterJavaSourceType(baseID)
	RegisterJavaSourceType(childID)
	child, base := &checkCastSource{}, &checkCastBase{}
	info := NewObjectInfo(childID, func(requested TypeID) any {
		if requested == baseID {
			return base
		}
		return child
	})
	child.ObjectInfo, base.ObjectInfo = info, info
	if ReferenceCheckCast[*checkCastBase](child, baseID) != base {
		t.Fatal("source superclass provider bypassed")
	}
	if ReferenceCheckCast[*checkCastSource](base, childID) != child {
		t.Fatal("most-derived source view lost")
	}
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*StringBuilder](child, "java.lang.StringBuilder") })
	opaque := &struct{ value byte }{1}
	requireCheckCastFailure(t, func() { ReferenceCheckCast[any](opaque, ObjectTypeID) })
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*StringBuilder](opaque, "java.lang.StringBuilder") })
	// An old native collection proves only interface membership. The guard must
	// preserve that existing proof without pretending it is an ArrayList class.
	list := NewList[int32]()
	if ReferenceCheckCast[any](list, "java.util.List") != list {
		t.Fatal("private native interface proof lost")
	}
	requireCheckCastFailure(t, func() { ReferenceCheckCast[any](list, "java.util.LinkedList") })
}

func TestReferenceCheckCastArrayAndReadOnlyConcurrent(t *testing.T) {
	builder := NewStringBuilder()
	array := ReferenceArrayLiteral("java.lang.StringBuilder", builder)
	if ReferenceCheckCast[*ReferenceArray](array, ArrayTypeID(CharSequenceTypeID)) != array {
		t.Fatal("array covariance/identity changed")
	}
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*ReferenceArray](array, ArrayTypeID("java.lang.StringBuffer")) })
	var workers sync.WaitGroup
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 100; j++ {
				if ReferenceCheckCast[*StringBuilder](builder, "java.lang.StringBuilder") != builder {
					t.Error("readonly cast changed pointer")
				}
			}
		}()
	}
	workers.Wait()
}
