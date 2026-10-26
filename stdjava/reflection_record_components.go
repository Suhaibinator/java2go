package stdjava

const RecordComponentTypeID TypeID = "java.lang.reflect.RecordComponent"

// RecordComponentDescriptor is declaration metadata in Java header order.
// RecordComponentsKnown distinguishes a provided empty header from unavailable
// component metadata on a genuine record declaration.
type RecordComponentDescriptor struct {
	Name string
	Type TypeID
}

// RecordComponent is a platform-created immutable reference, not a structural
// interface capability. Class metadata is its only construction path.
type RecordComponent struct {
	owner *Class
	name  *JavaString
	typ   *Class
}

func (*RecordComponent) JavaDynamicTypeID() TypeID { return RecordComponentTypeID }
func (component *RecordComponent) requireMetadata() {
	if component == nil {
		panic(NewNullPointerException("record component is null"))
	}
	if component.owner == nil || component.name == nil || component.typ == nil {
		panic(NewUnsupportedOperationException("record component metadata is unavailable"))
	}
}
func (component *RecordComponent) GetNameJavaString() *JavaString {
	component.requireMetadata()
	return component.name
}
func (component *RecordComponent) GetType() *Class {
	component.requireMetadata()
	return component.typ
}
func (class *Class) IsRecord() bool {
	if class == nil {
		panic(NewNullPointerException("class is null"))
	}
	return classDescriptor(class.TypeID()).Record
}
func (class *Class) GetRecordComponents() *ReferenceArray {
	if class == nil {
		panic(NewNullPointerException("class is null"))
	}
	descriptor := classDescriptor(class.TypeID())
	if !descriptor.Record {
		return nil
	}
	if !descriptor.RecordComponentsKnown {
		panic(NewUnsupportedOperationException("record component metadata is unavailable"))
	}
	for _, component := range descriptor.RecordComponents {
		if component.Name == "" || component.Type == "" {
			panic(NewUnsupportedOperationException("record component metadata is unavailable"))
		}
	}
	result := NewReferenceArray(len(descriptor.RecordComponents), RecordComponentTypeID)
	for index, component := range descriptor.RecordComponents {
		ReferenceArraySet(result, index, &RecordComponent{owner: class, name: reflectionIdentifierJavaString(component.Name), typ: ClassLiteral(component.Type)})
	}
	return result
}
