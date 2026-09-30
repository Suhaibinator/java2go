package stdjava

import "sync"

// Class is the erased runtime identity represented by java.lang.Class<T>.
type Class struct {
	typeID TypeID
}

var classLiterals sync.Map

func ClassLiteral(id TypeID) *Class {
	if existing, ok := classLiterals.Load(id); ok {
		return existing.(*Class)
	}
	literal := &Class{typeID: id}
	actual, _ := classLiterals.LoadOrStore(id, literal)
	return actual.(*Class)
}

func (class *Class) TypeID() TypeID {
	if class == nil {
		panic(NewNullPointerException("Class value is null"))
	}
	return class.typeID
}

// IsInstance tests nominal reference membership without initializing a class.
// The receiver is checked before returning false for null or primitive classes.
func (class *Class) IsInstance(value any) bool {
	id := class.TypeID()
	if isPrimitiveTypeID(id) || id == "void" {
		return false
	}
	return ObjectInstanceOf(value, id)
}

// ObjectGetClass implements getClass on a reference with a reified Java type.
// It uses the same descriptors as casts and reference-array store checks.
func ObjectGetClass(value any) *Class {
	if nilJavaReference(value) {
		panic(NewNullPointerException("getClass on null"))
	}
	id, ok := ObjectDynamicType(value)
	if !ok {
		panic(NewIllegalArgumentException("reference has no Java class descriptor"))
	}
	return ClassLiteral(id)
}
