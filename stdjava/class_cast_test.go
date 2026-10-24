package stdjava

import (
	"sync"
	"testing"
)

func classCastCaught(call func()) (caught any) {
	defer func() { caught = recover() }()
	call()
	return nil
}

func TestClassCastNativeIdentityNullAndMessages(t *testing.T) {
	builder, buffer := NewStringBuilder(), NewStringBuffer()
	if ClassCast[*StringBuilder](ClassLiteral("java.lang.StringBuilder"), builder) != builder || ClassLiteral("java.lang.StringBuffer").Cast(buffer) != buffer {
		t.Fatal("Class.cast changed same-family identity")
	}
	for _, row := range []struct {
		token   TypeID
		value   any
		message string
	}{
		{"java.lang.StringBuffer", builder, "Cannot cast java.lang.StringBuilder to java.lang.StringBuffer"},
		{"java.lang.StringBuilder", buffer, "Cannot cast java.lang.StringBuffer to java.lang.StringBuilder"},
		{ArrayTypeID("java.lang.StringBuffer"), ReferenceArrayLiteral("java.lang.StringBuilder", builder), "Cannot cast [Ljava.lang.StringBuilder; to [Ljava.lang.StringBuffer;"},
		{PrimitiveTypeID("int"), NewInteger(2), "Cannot cast java.lang.Integer to int"},
		{"void", NewObject(), "Cannot cast java.lang.Object to void"},
	} {
		failure := classCastCaught(func() { ClassCast[any](ClassLiteral(row.token), row.value) })
		_, ok := failure.(ClassCastException)
		if !ok || GetMessage(failure) != row.message {
			t.Fatalf("Class.cast failure=%T %v, expected %s", failure, failure, row.message)
		}
	}
	var absent *StringBuilder
	for _, token := range []TypeID{"java.lang.StringBuffer", PrimitiveTypeID("int"), "void"} {
		for _, null := range []any{nil, absent} {
			if ClassCast[any](ClassLiteral(token), null) != nil {
				t.Fatal("null cast fabricated reference")
			}
		}
	}
	if !StringIsNull(ClassCast[string](ClassLiteral(StringTypeID), nil)) {
		t.Fatal("legacy null sentinel lost")
	}
	var nullClass *Class
	if failure := classCastCaught(func() { ClassCast[any](nullClass, nil) }); !CaughtAs(failure, "NullPointerException") {
		t.Fatalf("null Class=%T", failure)
	}
	for _, opaque := range []any{new(byte), &StringBuilder{}} {
		if failure := classCastCaught(func() { ClassCast[any](ClassLiteral(ObjectTypeID), opaque) }); !CaughtAs(failure, "ClassCastException") {
			t.Fatalf("unknown identity gained Object membership: %T", failure)
		}
	}
}

func TestClassCastSourceViewsArraysAndReadOnlyConcurrent(t *testing.T) {
	const baseID TypeID = "classcast.Base"
	const childID TypeID = "classcast.Child"
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
	if ClassCast[*checkCastBase](ClassLiteral(baseID), child) != base || ClassCast[*checkCastSource](ClassLiteral(childID), base) != child {
		t.Fatal("source provider projection lost")
	}
	if !JavaReferenceEqual(ClassCast[*checkCastBase](ClassLiteral(baseID), child), child) {
		t.Fatal("source view allocation identity changed")
	}
	b := NewStringBuilder()
	array := ReferenceArrayLiteral("java.lang.StringBuilder", b)
	if ClassCast[*ReferenceArray](ClassLiteral(ArrayTypeID(CharSequenceTypeID)), array) != array {
		t.Fatal("array covariance/identity changed")
	}
	ints := PrimitiveArrayLiteral(PrimitiveTypeID("int"), int32(2))
	if ClassCast[*PrimitiveArray[int32]](ClassLiteral(ArrayTypeID(PrimitiveTypeID("int"))), ints) != ints {
		t.Fatal("primitive array changed")
	}
	if ClassCast[any](ClassLiteral(CharSequenceTypeID), b) != b {
		t.Fatal("native interface proof lost")
	}
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				if ClassCast[*StringBuilder](ClassLiteral("java.lang.StringBuilder"), b) != b {
					t.Error("readonly cast changed pointer")
				}
			}
		}()
	}
	workers.Wait()
}

type classCastDispatchFace interface{ Key() int32 }
type classCastDispatchBase struct{ *ObjectInfo }

func (*classCastDispatchBase) Key() int32 { return 3 }

type classCastDispatchChild struct{ *ObjectInfo }

func (*classCastDispatchChild) Key() int32 { return 7 }

func TestClassCastSourceInterfaceProviderPrecedesPhysicalFit(t *testing.T) {
	const baseID TypeID = "classcast.dispatch.Base"
	const childID TypeID = "classcast.dispatch.Child"
	const faceID TypeID = "classcast.dispatch.Face"
	RegisterJavaType(baseID, ObjectTypeID)
	RegisterJavaType(faceID, ObjectTypeID)
	RegisterJavaType(childID, baseID, faceID)
	for _, id := range []TypeID{baseID, childID, faceID} {
		RegisterJavaSourceType(id)
	}
	base, child := &classCastDispatchBase{}, &classCastDispatchChild{}
	info := NewObjectInfo(childID, func(requested TypeID) any {
		switch requested {
		case baseID:
			return base
		case childID, faceID:
			return child
		}
		return nil
	})
	base.ObjectInfo, child.ObjectInfo = info, info
	// The incoming base fits the Go interface but is not Java's virtual receiver.
	if direct := classCastDispatchFace(base); direct.Key() != 3 {
		t.Fatal("counterexample no longer exposes the physical-fit distinction")
	}
	for _, face := range []classCastDispatchFace{ObjectView[classCastDispatchFace](base, faceID), ClassCast[classCastDispatchFace](ClassLiteral(faceID), base)} {
		if face.Key() != 7 || !JavaReferenceEqual(face, base) {
			t.Fatal("source interface cast bypassed most-derived provider")
		}
	}
	if ObjectView[*classCastDispatchBase](child, baseID) != base {
		t.Fatal("concrete base projection changed")
	}
}
