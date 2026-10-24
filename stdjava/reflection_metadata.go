package stdjava

// Reflection metadata is declarative. Registering or copying it never invokes
// class initialization, construction, annotation factories, or field access.
type ConstructorDescriptor struct {
	Parameters []TypeID
	Modifiers  int32
	Construct  func(*Execution, []any) any
}
type AnnotationDescriptor struct {
	Type    TypeID
	Factory func(*Execution) any
}
type ReflectTypeKind uint8

const (
	ReflectClassKind ReflectTypeKind = iota
	ReflectParameterizedKind
	ReflectGenericArrayKind
	ReflectWildcardKind
	ReflectVariableKind
)

// Variable nodes are finite leaves identified by their Java declaration/name.
// Bounds are held on the declaring ClassDescriptor; no second registry is used.
type ReflectTypeDescriptor struct {
	Kind                     ReflectTypeKind
	Raw                      TypeID
	Owner                    *ReflectTypeDescriptor
	Arguments                []ReflectTypeDescriptor
	Component                *ReflectTypeDescriptor
	UpperBounds, LowerBounds []ReflectTypeDescriptor
	VariableDeclaration      TypeID
	VariableName             string
}
type TypeVariableDescriptor struct {
	Name   string
	Bounds []ReflectTypeDescriptor
}

func cloneReflectTypeDescriptor(input *ReflectTypeDescriptor) *ReflectTypeDescriptor {
	if input == nil {
		return nil
	}
	output := *input
	output.Owner = cloneReflectTypeDescriptor(input.Owner)
	output.Component = cloneReflectTypeDescriptor(input.Component)
	output.Arguments = cloneReflectTypeDescriptors(input.Arguments)
	output.UpperBounds = cloneReflectTypeDescriptors(input.UpperBounds)
	output.LowerBounds = cloneReflectTypeDescriptors(input.LowerBounds)
	return &output
}
func cloneReflectTypeDescriptors(input []ReflectTypeDescriptor) []ReflectTypeDescriptor {
	if input == nil {
		return nil
	}
	output := make([]ReflectTypeDescriptor, len(input))
	for index := range input {
		output[index] = *cloneReflectTypeDescriptor(&input[index])
	}
	return output
}
func cloneReflectionMetadata(input ClassDescriptor) ClassDescriptor {
	output := input
	output.GenericSuperclass = cloneReflectTypeDescriptor(input.GenericSuperclass)
	output.TypeParameters = append([]TypeVariableDescriptor(nil), input.TypeParameters...)
	for index := range output.TypeParameters {
		output.TypeParameters[index].Bounds = cloneReflectTypeDescriptors(input.TypeParameters[index].Bounds)
	}
	output.genericVariables = make([]*metadataTypeVariable, len(output.TypeParameters))
	for index, variable := range output.TypeParameters {
		output.genericVariables[index] = &metadataTypeVariable{declaration: input.Type, name: variable.Name, bounds: variable.Bounds}
	}
	if input.Constructors != nil {
		output.Constructors = make([]ConstructorDescriptor, len(input.Constructors))
		copy(output.Constructors, input.Constructors)
	}
	for index := range output.Constructors {
		output.Constructors[index].Parameters = append([]TypeID(nil), input.Constructors[index].Parameters...)
	}
	output.AnnotationValues = append([]AnnotationDescriptor(nil), input.AnnotationValues...)
	output.Fields = append([]FieldDescriptor(nil), input.Fields...)
	for index := range output.Fields {
		output.Fields[index].GenericType = cloneReflectTypeDescriptor(input.Fields[index].GenericType)
		output.Fields[index].Annotations = append([]AnnotationDescriptor(nil), input.Fields[index].Annotations...)
	}
	return output
}
