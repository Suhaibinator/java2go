package stdjava

// CloneNotSupportedException is Object.clone's checked failure. Its descriptor
// and catch hierarchy retain java.lang.Exception as the exact parent.
type CloneNotSupportedException struct{ ThrowableBase }

func NewCloneNotSupportedException(message string) CloneNotSupportedException {
	return CloneNotSupportedException{newThrowableBase("CloneNotSupportedException", message)}
}
func init() { RegisterException("CloneNotSupportedException", "Exception") }

// generatedObjectCloner is a compiler-reserved protocol. The compiler emits
// shallow field copies in each declaring package; runtime reflection never
// copies host structs or invokes Java constructors.
type generatedObjectCloner interface{ Java2goCloneSubobject(any) any }
type javaArrayCloner interface{ javaCloneArray() any }

// ObjectCloneExecution invokes Object.clone nonvirtually. An inherited source
// receiver is resolved back to its complete allocation before selecting its
// generated copier, preserving the most-derived Java class.
func ObjectCloneExecution(_ *Execution, value any) any {
	ReferenceRequireNonNull(value)
	dynamic, known := ObjectDynamicType(value)
	if !known || !JavaTypeAssignable(dynamic, CloneableTypeID) {
		panic(NewCloneNotSupportedException(string(dynamic)))
	}
	if array, ok := value.(javaArrayCloner); ok {
		return array.javaCloneArray()
	}
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			if complete := info.resolveView(dynamic); complete != nil {
				value = complete
			}
		}
	}
	javaTypeRegistry.RLock()
	source := javaTypeRegistry.types[dynamic].source
	javaTypeRegistry.RUnlock()
	if copier, ok := value.(generatedObjectCloner); source && ok {
		cloned := copier.Java2goCloneSubobject(nil)
		if resultType, known := ObjectDynamicType(cloned); known && resultType == dynamic {
			return cloned
		}
		panic(NewUnsupportedOperationException("incomplete Object.clone copier for " + string(dynamic)))
	}
	// Native classes require a runtime-owned copier rather than structural host
	// copying. Missing metadata must not silently produce a partial source clone.
	panic(NewUnsupportedOperationException("missing Object.clone copier for " + string(dynamic)))
}
func (array *PrimitiveArray[T]) javaCloneArray() any {
	result := NewPrimitiveArray[T](len(array.Elements), array.componentType)
	copy(result.Elements, array.Elements)
	return result
}
func (array *ReferenceArray) javaCloneArray() any {
	result := NewReferenceArray(len(array.elements), array.componentType)
	copy(result.elements, array.elements)
	return result
}

// ArrayCloneExecution retains the exact reified array ABI while allowing a Java
// clone invocation to be used as a statement. Array cloning preserves Go type.
func ArrayCloneExecution[T any](execution *Execution, array T) T {
	return ObjectCloneExecution(execution, array).(T)
}
