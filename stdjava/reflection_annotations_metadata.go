package stdjava

func reflectionAnnotationValue(execution *Execution, descriptors []AnnotationDescriptor, id TypeID) (any, bool) {
	for _, descriptor := range descriptors {
		if descriptor.Type != id {
			continue
		}
		if descriptor.Factory == nil {
			panic(NewUnsupportedOperationException("annotation factory metadata is unavailable"))
		}
		requireExecution(execution)
		result := descriptor.Factory(execution)
		if !ObjectInstanceOf(result, id) || !ObjectInstanceOf(result, AnnotationTypeID) {
			panic(NewClassCastException("annotation factory returned an incompatible value"))
		}
		return result, true
	}
	return nil, false
}
func (class *Class) GetAnnotation(annotation *Class) any {
	return class.GetAnnotationExecution(NewExecution(), annotation)
}
func (class *Class) GetAnnotationExecution(execution *Execution, annotation *Class) any {
	class.TypeID()
	id := annotation.TypeID()
	inherited := classDescriptor(id).InheritedAnnotation
	for current := class; current != nil; current = current.GetSuperclass() {
		if value, found := reflectionAnnotationValue(execution, classDescriptor(current.TypeID()).AnnotationValues, id); found {
			return value
		}
		if !inherited {
			break
		}
	}
	return nil
}
func (field *Field) GetAnnotation(annotation *Class) any {
	return field.GetAnnotationExecution(NewExecution(), annotation)
}
func (field *Field) GetAnnotationExecution(execution *Execution, annotation *Class) any {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	id := annotation.TypeID()
	value, _ := reflectionAnnotationValue(execution, field.descriptor.Annotations, id)
	return value
}
