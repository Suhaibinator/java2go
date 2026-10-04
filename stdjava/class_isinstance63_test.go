package stdjava

import "testing"

type classIsInstance63Reference struct{ id TypeID }

func (v *classIsInstance63Reference) JavaDynamicTypeID() TypeID { return v.id }

func TestClassIsInstanceNominalBoundaries63(t *testing.T) {
	const parent TypeID = "isinstance63.Parent"
	const child TypeID = "isinstance63.Child"
	const marker TypeID = "isinstance63.Marker"
	const other TypeID = "isinstance63.Other"
	RegisterJavaType(marker, ObjectTypeID)
	RegisterJavaType(parent, ObjectTypeID, marker)
	RegisterJavaType(child, parent)
	RegisterJavaType(other, ObjectTypeID)
	leaf := &classIsInstance63Reference{child}
	var typedNil *classIsInstance63Reference
	refs := NewReferenceArray(0, StringTypeID)
	ints := NewPrimitiveArray[int32](0, PrimitiveIntTypeID)
	for _, q := range []struct {
		name  string
		class *Class
		value any
		want  bool
	}{
		{"same", ClassLiteral(child), leaf, true},
		{"parent", ClassLiteral(parent), leaf, true},
		{"interface", ClassLiteral(marker), leaf, true},
		{"unrelated-identical-Go-shape", ClassLiteral(parent), &classIsInstance63Reference{other}, false},
		{"null", ClassLiteral(parent), nil, false},
		{"typed-null", ClassLiteral(parent), typedNil, false},
		{"boxed-integer", ClassLiteral(IntegerTypeID), BoxInteger(3), true},
		{"boxed-integer-number", ClassLiteral(NumberTypeID), BoxInteger(3), true},
		{"unboxed-is-not-reference", ClassLiteral(IntegerTypeID), int32(3), false},
		{"primitive", ClassLiteral(PrimitiveIntTypeID), BoxInteger(3), false},
		{"primitive-descriptor-impostor", ClassLiteral(PrimitiveIntTypeID), &classIsInstance63Reference{PrimitiveIntTypeID}, false},
		{"void", ClassLiteral(PrimitiveTypeID("void")), leaf, false},
		{"void-descriptor-impostor", ClassLiteral("void"), &classIsInstance63Reference{"void"}, false},
		{"reference-array-covariance", ClassLiteral(ArrayTypeID(ObjectTypeID)), refs, true},
		{"primitive-array-not-reference-array", ClassLiteral(ArrayTypeID(ObjectTypeID)), ints, false},
		{"primitive-array-exact", ClassLiteral(ArrayTypeID(PrimitiveIntTypeID)), ints, true},
		{"primitive-array-distinct", ClassLiteral(ArrayTypeID(PrimitiveLongTypeID)), ints, false},
		{"array-object", ClassLiteral(ObjectTypeID), ints, true},
		{"array-cloneable", ClassLiteral(CloneableTypeID), ints, true},
		{"array-serializable", ClassLiteral(SerializableTypeID), ints, true},
	} {
		t.Run(q.name, func(t *testing.T) {
			if got := q.class.IsInstance(q.value); got != q.want {
				t.Fatalf("isInstance = %t, want %t", got, q.want)
			}
		})
	}
}

func TestClassIsInstanceNullReceiverAndNoInitialization63(t *testing.T) {
	var class *Class
	for _, value := range []any{nil, BoxInteger(1)} {
		expectBoxedException(t, "NullPointerException", func() { class.IsInstance(value) })
	}
	const id TypeID = "isinstance63.Dormant"
	initialized := false
	RegisterJavaType(id, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(*Execution) { initialized = true }})
	before := ClassLiteral(id)
	if !before.IsInstance(&classIsInstance63Reference{id}) || initialized || before != ClassLiteral(id) {
		t.Fatal("isInstance changed initialization or Class cache identity")
	}
}
