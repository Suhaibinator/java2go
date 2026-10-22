package stdjava

import "testing"

func TestBitSetCanonicalCloneCastAPIRED(t *testing.T) {
	original := NewBitSet(256)
	original.Set(65)
	clone := original.Clone()
	t.Logf("actual clone dynamic=%q canonical assignable=%v", clone.JavaDynamicTypeID(), JavaTypeAssignable(clone.JavaDynamicTypeID(), TypeID("java.util.BitSet")))
	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("native BitSet clone cannot be cast to canonical java.util.BitSet: %T: %s", failure, GetMessage(failure))
		}
	}()
	if got := ReferenceCheckCast[*BitSet](clone, TypeID("java.util.BitSet")); got != clone || !got.Get(65) {
		t.Fatal("canonical cast changed clone identity/state")
	}
}

func TestBitSetCanonicalIdentityHierarchyAndReflection(t *testing.T) {
	value := NewBitSet()
	if id, known := ObjectDynamicType(value); !known || id != TypeID("java.util.BitSet") {
		t.Fatalf("allocation has noncanonical identity %q known=%v", id, known)
	}
	for _, parent := range []TypeID{BitSetTypeID, ObjectTypeID, CloneableTypeID, SerializableTypeID} {
		if !ObjectInstanceOf(value, parent) || !JavaTypeAssignable(BitSetTypeID, parent) {
			t.Fatalf("canonical allocation lost %s membership", parent)
		}
		if ReferenceCheckCast[any](value, parent) != value {
			t.Fatalf("%s cast changed allocation identity", parent)
		}
	}
	class := ClassForName(NewExecution(), "java.util.BitSet")
	if class != ObjectGetClass(value) || class != ClassLiteral(BitSetTypeID) || class.GetName() != "java.util.BitSet" || class.GetSimpleName() != "BitSet" {
		t.Fatal("reflection did not retain canonical allocation/class identity")
	}
	if class.GetSuperclass() != ClassLiteral(ObjectTypeID) || !ClassLiteral(CloneableTypeID).IsAssignableFrom(class) || !ClassLiteral(SerializableTypeID).IsAssignableFrom(class) {
		t.Fatal("reflection hierarchy disagrees with native declaration")
	}
}

func TestBitSetCanonicalNullAndCloneIndependence(t *testing.T) {
	var typedNull *BitSet
	if ReferenceCheckCast[*BitSet](typedNull, BitSetTypeID) != nil || ReferenceCheckCast[*BitSet](nil, BitSetTypeID) != nil || ClassLiteral(BitSetTypeID).IsInstance(typedNull) {
		t.Fatal("canonical typed/null reference semantics changed")
	}
	original := NewBitSet(256)
	original.Set(65)
	clone := ReferenceCheckCast[*BitSet](original.Clone(), BitSetTypeID)
	clone.Set(2000)
	original.Set(130)
	if clone == original || !clone.Get(65) || !original.Get(65) || original.Get(2000) || clone.Get(130) {
		t.Fatal("clones shared mutable words or lost existing bits")
	}
	if ObjectGetClass(original) != ObjectGetClass(clone) {
		t.Fatal("clone changed Java class")
	}
}

func TestBitSetCanonicalReferenceArrays(t *testing.T) {
	value := NewBitSet(128)
	value.Set(67)
	array := NewReferenceArray(1, BitSetTypeID)
	ReferenceArraySet(array, 0, value.Clone())
	if !ReferenceArrayGet[*BitSet](array, 0, BitSetTypeID).Get(67) {
		t.Fatal("canonical clone did not survive checked array store/read")
	}
	for _, parent := range []TypeID{ObjectTypeID, CloneableTypeID, SerializableTypeID} {
		if !JavaTypeAssignable(ArrayTypeID(BitSetTypeID), ArrayTypeID(parent)) || JavaArrayCast[*ReferenceArray](array, ArrayTypeID(parent)) != array {
			t.Fatalf("canonical reference array lost covariance to %s[]", parent)
		}
	}
	component := ObjectGetClass(array).GetComponentType()
	if component != ClassLiteral(BitSetTypeID) || component.GetName() != "java.util.BitSet" {
		t.Fatal("reflection array component lost canonical identity")
	}
}

type bitSetCanonicalSourceHomonym struct{}

func (*bitSetCanonicalSourceHomonym) JavaDynamicTypeID() TypeID { return "BitSet" }

func TestBitSetCanonicalSourceHomonymShield(t *testing.T) {
	const sourceID TypeID = "BitSet"
	javaTypeRegistry.Lock()
	previous, existed := javaTypeRegistry.types[sourceID]
	javaTypeRegistry.Unlock()
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		defer javaTypeRegistry.Unlock()
		if existed {
			javaTypeRegistry.types[sourceID] = previous
		} else {
			delete(javaTypeRegistry.types, sourceID)
		}
	})
	if existed {
		t.Fatal("native runtime published an ambiguous short BitSet alias")
	}
	RegisterJavaType(sourceID, ObjectTypeID)
	RegisterJavaSourceType(sourceID)
	native, source := NewBitSet(), &bitSetCanonicalSourceHomonym{}
	if ObjectInstanceOf(native, sourceID) || ObjectInstanceOf(source, BitSetTypeID) || JavaTypeAssignable(sourceID, BitSetTypeID) || JavaTypeAssignable(BitSetTypeID, sourceID) {
		t.Fatal("same simple spelling crossed canonical/source nominal identities")
	}
	for _, cast := range []func(){
		func() { ReferenceCheckCast[any](native, sourceID) },
		func() { ReferenceCheckCast[any](source, BitSetTypeID) },
		func() { ReferenceArraySet(NewReferenceArray(1, BitSetTypeID), 0, source) },
		func() { ReferenceArraySet(NewReferenceArray(1, sourceID), 0, native) },
	} {
		func() {
			defer func() {
				if failure := recover(); failure == nil {
					t.Error("source/native homonym conversion was admitted")
				} else if !CaughtAs(failure, "ClassCastException") && !CaughtAs(failure, "ArrayStoreException") {
					t.Errorf("unexpected homonym failure %T", failure)
				}
			}()
			cast()
		}()
	}
}
