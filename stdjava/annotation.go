package stdjava

// A source annotation interface includes Annotation.annotationType. The
// physical method set is insufficient for Java identity: every public adapter
// checks the registered nominal Annotation edge before entering the value.
type JavaAnnotation interface{ AnnotationType() *Class }

func AnnotationTypeExecution(execution *Execution, annotation JavaAnnotation) *Class {
	requireExecution(execution)
	ReferenceRequireNonNull(annotation)
	if !ObjectInstanceOf(annotation, AnnotationTypeID) {
		panic(NewClassCastException("value is not a Java annotation"))
	}
	role, ok := annotation.(interface{ AnnotationTypeJava2goExecution(*Execution) *Class })
	if !ok {
		panic(NewUnsupportedOperationException("annotation execution method is unavailable"))
	}
	return role.AnnotationTypeJava2goExecution(execution)
}

func reflectionAnnotationClass(annotation *Class) TypeID {
	if annotation == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return annotation.TypeID()
}
func reflectionAnnotationsArray(execution *Execution, descriptors []AnnotationDescriptor) *ReferenceArray {
	requireExecution(execution)
	result := NewReferenceArray(len(descriptors), AnnotationTypeID)
	for index, descriptor := range descriptors {
		value, found := reflectionAnnotationValue(execution, []AnnotationDescriptor{descriptor}, descriptor.Type)
		if !found {
			panic(NewUnsupportedOperationException("annotation descriptor is unavailable"))
		}
		ReferenceArraySet(result, index, value)
	}
	return result
}
func (field *Field) GetAnnotationsExecution(execution *Execution) *ReferenceArray {
	ReferenceRequireNonNull(field)
	return reflectionAnnotationsArray(execution, field.descriptor.Annotations)
}
func (method *Method) GetAnnotationsExecution(execution *Execution) *ReferenceArray {
	method.requireMethod()
	return reflectionAnnotationsArray(execution, method.descriptor.Annotations)
}
func (constructor *Constructor) GetAnnotationsExecution(execution *Execution) *ReferenceArray {
	ReferenceRequireNonNull(constructor)
	return reflectionAnnotationsArray(execution, constructor.descriptor.Annotations)
}
func (class *Class) GetDeclaredAnnotationsExecution(execution *Execution) *ReferenceArray {
	class.TypeID()
	return reflectionAnnotationsArray(execution, classDescriptor(class.TypeID()).AnnotationValues)
}
func (method *Method) GetAnnotationExecution(execution *Execution, annotation *Class) any {
	method.requireMethod()
	id := reflectionAnnotationClass(annotation)
	value, _ := reflectionAnnotationValue(execution, method.descriptor.Annotations, id)
	return value
}
func (constructor *Constructor) GetAnnotationExecution(execution *Execution, annotation *Class) any {
	ReferenceRequireNonNull(constructor)
	id := reflectionAnnotationClass(annotation)
	value, _ := reflectionAnnotationValue(execution, constructor.descriptor.Annotations, id)
	return value
}

func reflectionClassAnnotations(class *Class) []AnnotationDescriptor {
	class.TypeID()
	result := []AnnotationDescriptor{}
	positions := map[TypeID]int{}
	if parent := class.GetSuperclass(); parent != nil {
		for _, descriptor := range reflectionClassAnnotations(parent) {
			if descriptor.Inherited || classDescriptor(descriptor.Type).InheritedAnnotation {
				positions[descriptor.Type] = len(result)
				result = append(result, descriptor)
			}
		}
	}
	for _, descriptor := range classDescriptor(class.TypeID()).AnnotationValues {
		if index, present := positions[descriptor.Type]; present {
			result[index] = descriptor
		} else {
			positions[descriptor.Type] = len(result)
			result = append(result, descriptor)
		}
	}
	return result
}
func (class *Class) GetAnnotationsExecution(execution *Execution) *ReferenceArray {
	return reflectionAnnotationsArray(execution, reflectionClassAnnotations(class))
}
