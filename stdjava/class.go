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
