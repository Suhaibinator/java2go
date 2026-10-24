package stdjava

// GetComponentType returns the reified component of an array class without
// initializing it. ClassLiteral preserves the component's canonical identity.
func (class *Class) GetComponentType() *Class {
	id := class.TypeID()
	component, array := arrayComponentTypeID(id)
	if !array {
		return nil
	}
	return ClassLiteral(component)
}

// ClassGetComponentTypeExecution shares the intrinsic execution ABI. This
// descriptor query invokes no Java code and permits the metadata-only nil
// context used by other pure Class operations.
func ClassGetComponentTypeExecution(_ *Execution, class *Class) *Class {
	return class.GetComponentType()
}
