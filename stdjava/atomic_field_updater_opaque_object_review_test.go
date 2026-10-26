package stdjava

import "testing"

const opaqueUpdaterReviewOwner TypeID = "updater.review.OpaqueOwner"

type opaqueUpdaterReviewTarget struct {
	object, text VolatileFieldCell
}

func (*opaqueUpdaterReviewTarget) JavaDynamicTypeID() TypeID { return opaqueUpdaterReviewOwner }

func TestAtomicReferenceFieldUpdaterOpaqueObjectContract(t *testing.T) {
	RegisterJavaType(opaqueUpdaterReviewOwner, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{
		Type: opaqueUpdaterReviewOwner, HasModifiers: true, Modifiers: 1,
		Fields: []FieldDescriptor{
			{Name: "object", Type: ObjectTypeID, HasModifiers: true, Modifiers: 65,
				VolatileCell: func(_ *Execution, receiver any) *VolatileFieldCell {
					return &receiver.(*opaqueUpdaterReviewTarget).object
				}},
			{Name: "text", Type: StringTypeID, HasModifiers: true, Modifiers: 65,
				VolatileCell: func(_ *Execution, receiver any) *VolatileFieldCell {
					return &receiver.(*opaqueUpdaterReviewTarget).text
				}},
		},
	})
	owner := ClassLiteral(opaqueUpdaterReviewOwner)
	objectClass := ClassLiteral(ObjectTypeID)
	a, b := NewObject(), NewObject()
	if !objectClass.IsInstance(a) || !objectClass.IsInstance(b) {
		t.Fatal("NewObject tokens must satisfy Java Object membership")
	}
	if !JavaReferenceEqual(a, a) || JavaReferenceEqual(a, b) {
		t.Fatal("NewObject must preserve alias identity and distinct allocation identity")
	}
	t.Run("set_get_alias_and_distinct_cas", func(t *testing.T) {
		target := &opaqueUpdaterReviewTarget{}
		u := NewAtomicReferenceFieldUpdater(owner, objectClass, "object", opaqueUpdaterReviewOwner)
		alias := NewAtomicReferenceFieldUpdater(owner, objectClass, "object", opaqueUpdaterReviewOwner)
		defer func() {
			if failure := recover(); failure != nil {
				t.Fatalf("Java Object updater rejected a valid NewObject reference: %T %v", failure, failure)
			}
		}()
		u.Set(target, a)
		if !JavaReferenceEqual(alias.Get(target), a) {
			t.Fatal("independent updater must read the same opaque reference cell")
		}
		if alias.CompareAndSet(target, b, nil) || !JavaReferenceEqual(u.Get(target), a) {
			t.Fatal("distinct NewObject expected value must fail CAS without mutation")
		}
		if !alias.CompareAndSet(target, a, b) || !JavaReferenceEqual(u.Get(target), b) {
			t.Fatal("alias expected value must CAS the opaque reference")
		}
		if !JavaReferenceEqual(u.GetAndSet(target, a), b) || !JavaReferenceEqual(alias.Get(target), a) {
			t.Fatal("exchange must retain old/new opaque allocation identities")
		}
	})

	t.Run("narrower_nominal_type_refuses_opaque_object", func(t *testing.T) {
		target := &opaqueUpdaterReviewTarget{}
		u := NewAtomicReferenceFieldUpdater(owner, ClassLiteral(StringTypeID), "text", opaqueUpdaterReviewOwner)
		text := NewJavaStringUTF16([]uint16{'x'})
		u.Set(target, text)
		opaqueUpdaterReviewClassCast(t, func() { u.Set(target, a) })
		opaqueUpdaterReviewClassCast(t, func() { u.CompareAndSet(target, text, a) })
		if !JavaReferenceEqual(u.Get(target), text) {
			t.Fatal("wrong nominal update must leave the prior reference unchanged")
		}
	})

	t.Run("object_type_refuses_unboxed_primitive", func(t *testing.T) {
		target := &opaqueUpdaterReviewTarget{}
		u := NewAtomicReferenceFieldUpdater(owner, objectClass, "object", opaqueUpdaterReviewOwner)
		boxed := BoxInteger(1000)
		u.Set(target, boxed)
		if objectClass.IsInstance(int32(3)) {
			t.Fatal("unboxed primitive must not satisfy Java Object membership")
		}
		opaqueUpdaterReviewClassCast(t, func() { u.Set(target, int32(3)) })
		opaqueUpdaterReviewClassCast(t, func() { u.CompareAndSet(target, boxed, int32(3)) })
		if !JavaReferenceEqual(u.Get(target), boxed) {
			t.Fatal("unboxed primitive update must leave the prior reference unchanged")
		}
	})
}

func opaqueUpdaterReviewClassCast(t *testing.T, call func()) {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); call() }()
	actual, known := ObjectDynamicType(failure)
	if !known || actual != BuiltinThrowableTypeID("ClassCastException") {
		t.Fatalf("wrong value panic = %T (%s), want ClassCastException", failure, actual)
	}
}
