package stdjava

import "strings"

const (
	metadataParameterizedTypeID TypeID = "sun.reflect.generics.reflectiveObjects.ParameterizedTypeImpl"
	metadataGenericArrayTypeID  TypeID = "sun.reflect.generics.reflectiveObjects.GenericArrayTypeImpl"
	metadataWildcardTypeID      TypeID = "sun.reflect.generics.reflectiveObjects.WildcardTypeImpl"
	metadataTypeVariableID      TypeID = "sun.reflect.generics.reflectiveObjects.TypeVariableImpl"
)

func init() {
	RegisterJavaType(metadataParameterizedTypeID, ObjectTypeID, ParameterizedTypeTypeID)
	RegisterJavaType(metadataGenericArrayTypeID, ObjectTypeID, GenericArrayTypeTypeID)
	RegisterJavaType(metadataWildcardTypeID, ObjectTypeID, WildcardTypeTypeID)
	RegisterJavaType(metadataTypeVariableID, ObjectTypeID, TypeVariableTypeID)
}
func ResolveReflectTypeDescriptor(descriptor *ReflectTypeDescriptor) ReflectType {
	if descriptor == nil {
		return nil
	}
	switch descriptor.Kind {
	case ReflectClassKind:
		return ClassLiteral(descriptor.Raw)
	case ReflectParameterizedKind:
		return &metadataParameterizedType{raw: ClassLiteral(descriptor.Raw), owner: ResolveReflectTypeDescriptor(descriptor.Owner), arguments: resolveReflectTypes(descriptor.Arguments)}
	case ReflectGenericArrayKind:
		return &metadataGenericArrayType{component: ResolveReflectTypeDescriptor(descriptor.Component)}
	case ReflectWildcardKind:
		upper := resolveReflectTypes(descriptor.UpperBounds)
		if len(upper) == 0 {
			upper = []ReflectType{ClassLiteral(ObjectTypeID)}
		}
		return &metadataWildcardType{upper: upper, lower: resolveReflectTypes(descriptor.LowerBounds)}
	case ReflectVariableKind:
		for _, variable := range classDescriptor(descriptor.VariableDeclaration).genericVariables {
			if variable.name == descriptor.VariableName {
				return variable
			}
		}
		panic(NewUnsupportedOperationException("reflective type variable declaration metadata is unavailable"))
	default:
		panic(NewUnsupportedOperationException("reflective type metadata kind is unavailable"))
	}
}
func resolveReflectTypes(descriptors []ReflectTypeDescriptor) []ReflectType {
	values := make([]ReflectType, len(descriptors))
	for index := range descriptors {
		values[index] = ResolveReflectTypeDescriptor(&descriptors[index])
	}
	return values
}
func reflectTypeArray(values []ReflectType) *ReferenceArray {
	return ReferenceArrayLiteral(ReflectTypeTypeID, values...)
}
func (class *Class) GetGenericSuperclass() ReflectType {
	descriptor := classDescriptor(class.TypeID())
	if descriptor.GenericSuperclass != nil {
		return ResolveReflectTypeDescriptor(descriptor.GenericSuperclass)
	}
	if parent := class.GetSuperclass(); parent != nil {
		return parent
	}
	return nil
}
func (class *Class) GetTypeParameters() *ReferenceArray {
	variables := classDescriptor(class.TypeID()).genericVariables
	result := NewReferenceArray(len(variables), TypeVariableTypeID)
	for index, variable := range variables {
		ReferenceArraySet(result, index, variable)
	}
	return result
}
func (field *Field) GetGenericType() ReflectType {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	if field.descriptor.GenericType != nil {
		return ResolveReflectTypeDescriptor(field.descriptor.GenericType)
	}
	return field.GetType()
}
func reflectTypeEqual(execution *Execution, left, right ReflectType) bool {
	if nilJavaReference(left) {
		return nilJavaReference(right)
	}
	return !nilJavaReference(right) && ObjectEqualsExecution(execution, left, right)
}
func reflectTypeSliceEqual(execution *Execution, left []ReflectType, right *ReferenceArray) bool {
	values := ReferenceArrayIterationElements(right)
	if len(left) != len(values) {
		return false
	}
	for index := range left {
		if !reflectTypeEqual(execution, left[index], values[index]) {
			return false
		}
	}
	return true
}
func reflectTypeHash(execution *Execution, value ReflectType) int32 {
	if nilJavaReference(value) {
		return 0
	}
	return ObjectHashCodeExecution(execution, value)
}
func reflectTypeSliceHash(execution *Execution, values []ReflectType) int32 {
	hash := int32(1)
	for _, value := range values {
		hash = 31*hash + reflectTypeHash(execution, value)
	}
	return hash
}
func reflectTypesNames(values []ReflectType) []string {
	names := make([]string, len(values))
	for index, value := range values {
		names[index] = ReflectTypeNameExecution(NewExecution(), value)
	}
	return names
}

type metadataParameterizedType struct {
	raw       *Class
	owner     ReflectType
	arguments []ReflectType
}

func (*metadataParameterizedType) JavaDynamicTypeID() TypeID       { return metadataParameterizedTypeID }
func (value *metadataParameterizedType) GetRawType() ReflectType   { return value.raw }
func (value *metadataParameterizedType) GetOwnerType() ReflectType { return value.owner }
func (value *metadataParameterizedType) GetActualTypeArguments() *ReferenceArray {
	return reflectTypeArray(value.arguments)
}
func (value *metadataParameterizedType) GetTypeName() string {
	name := value.raw.GetName()
	if value.owner != nil {
		leaf := value.raw.GetSimpleName()
		if owner, ok := value.owner.(*metadataParameterizedType); ok {
			leaf = strings.ReplaceAll(value.raw.GetName(), owner.raw.GetName()+"$", "")
		}
		name = ReflectTypeNameExecution(NewExecution(), value.owner) + "$" + leaf
	}
	if len(value.arguments) != 0 {
		name += "<" + strings.Join(reflectTypesNames(value.arguments), ", ") + ">"
	}
	return name
}
func (value *metadataParameterizedType) String() string { return value.GetTypeName() }
func (value *metadataParameterizedType) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.GetTypeName())
}
func (value *metadataParameterizedType) Equals(other any) bool {
	return value.EqualsJava2goExecution(NewExecution(), other)
}
func (value *metadataParameterizedType) EqualsJava2goExecution(execution *Execution, other any) bool {
	if !ObjectInstanceOf(other, ParameterizedTypeTypeID) {
		return false
	}
	return reflectTypeEqual(execution, value.owner, ReflectTypeMemberExecution(execution, other, ParameterizedTypeTypeID, "GetOwnerType")) && reflectTypeEqual(execution, value.raw, ReflectTypeMemberExecution(execution, other, ParameterizedTypeTypeID, "GetRawType")) && reflectTypeSliceEqual(execution, value.arguments, ReflectArrayMemberExecution(execution, other, ParameterizedTypeTypeID, "GetActualTypeArguments", ReflectTypeTypeID))
}
func (value *metadataParameterizedType) HashCode() int32 {
	return value.HashCodeJava2goExecution(NewExecution())
}
func (value *metadataParameterizedType) HashCodeJava2goExecution(execution *Execution) int32 {
	return reflectTypeSliceHash(execution, value.arguments) ^ reflectTypeHash(execution, value.owner) ^ reflectTypeHash(execution, value.raw)
}

type metadataGenericArrayType struct{ component ReflectType }

func (*metadataGenericArrayType) JavaDynamicTypeID() TypeID                  { return metadataGenericArrayTypeID }
func (value *metadataGenericArrayType) GetGenericComponentType() ReflectType { return value.component }
func (value *metadataGenericArrayType) GetTypeName() string {
	return ReflectTypeNameExecution(NewExecution(), value.component) + "[]"
}
func (value *metadataGenericArrayType) String() string { return value.GetTypeName() }
func (value *metadataGenericArrayType) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.GetTypeName())
}
func (value *metadataGenericArrayType) Equals(other any) bool {
	return value.EqualsJava2goExecution(NewExecution(), other)
}
func (value *metadataGenericArrayType) EqualsJava2goExecution(execution *Execution, other any) bool {
	return ObjectInstanceOf(other, GenericArrayTypeTypeID) && reflectTypeEqual(execution, value.component, ReflectTypeMemberExecution(execution, other, GenericArrayTypeTypeID, "GetGenericComponentType"))
}
func (value *metadataGenericArrayType) HashCode() int32 {
	return reflectTypeHash(NewExecution(), value.component)
}

type metadataWildcardType struct{ upper, lower []ReflectType }

func (*metadataWildcardType) JavaDynamicTypeID() TypeID { return metadataWildcardTypeID }
func (value *metadataWildcardType) GetUpperBounds() *ReferenceArray {
	return reflectTypeArray(value.upper)
}
func (value *metadataWildcardType) GetLowerBounds() *ReferenceArray {
	return reflectTypeArray(value.lower)
}
func (value *metadataWildcardType) GetTypeName() string {
	if len(value.lower) != 0 {
		return "? super " + strings.Join(reflectTypesNames(value.lower), " & ")
	}
	if len(value.upper) == 0 || (len(value.upper) == 1 && value.upper[0] == ClassLiteral(ObjectTypeID)) {
		return "?"
	}
	return "? extends " + strings.Join(reflectTypesNames(value.upper), " & ")
}
func (value *metadataWildcardType) String() string { return value.GetTypeName() }
func (value *metadataWildcardType) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.GetTypeName())
}
func (value *metadataWildcardType) Equals(other any) bool {
	return value.EqualsJava2goExecution(NewExecution(), other)
}
func (value *metadataWildcardType) EqualsJava2goExecution(execution *Execution, other any) bool {
	return ObjectInstanceOf(other, WildcardTypeTypeID) && reflectTypeSliceEqual(execution, value.upper, ReflectArrayMemberExecution(execution, other, WildcardTypeTypeID, "GetUpperBounds", ReflectTypeTypeID)) && reflectTypeSliceEqual(execution, value.lower, ReflectArrayMemberExecution(execution, other, WildcardTypeTypeID, "GetLowerBounds", ReflectTypeTypeID))
}
func (value *metadataWildcardType) HashCode() int32 {
	return reflectTypeSliceHash(NewExecution(), value.upper) ^ reflectTypeSliceHash(NewExecution(), value.lower)
}

type metadataTypeVariable struct {
	declaration TypeID
	name        string
	bounds      []ReflectTypeDescriptor
}

func (*metadataTypeVariable) JavaDynamicTypeID() TypeID { return metadataTypeVariableID }
func (value *metadataTypeVariable) GetName() string     { return value.name }
func (value *metadataTypeVariable) GetGenericDeclaration() GenericDeclaration {
	return ClassLiteral(value.declaration)
}
func (value *metadataTypeVariable) GetBounds() *ReferenceArray {
	if len(value.bounds) == 0 {
		return reflectTypeArray([]ReflectType{ClassLiteral(ObjectTypeID)})
	}
	return reflectTypeArray(resolveReflectTypes(value.bounds))
}
func (value *metadataTypeVariable) GetTypeName() string { return value.name }
func (value *metadataTypeVariable) String() string      { return value.name }
func (value *metadataTypeVariable) StringJava2goExecution(*Execution) *JavaString {
	return reflectionIdentifierJavaString(value.name)
}
func (value *metadataTypeVariable) Equals(other any) bool {
	return value.EqualsJava2goExecution(NewExecution(), other)
}
func (value *metadataTypeVariable) EqualsJava2goExecution(execution *Execution, other any) bool {
	if !ObjectInstanceOf(other, TypeVariableTypeID) {
		return false
	}
	name := ReflectStringMemberJavaStringExecution(execution, other, TypeVariableTypeID, "GetName")
	return reflectionIdentifierJavaString(value.name).Equals(name) && JavaReferenceEqual(ClassLiteral(value.declaration), ReflectDeclarationMemberExecution(execution, other, TypeVariableTypeID, "GetGenericDeclaration"))
}
func (value *metadataTypeVariable) HashCode() int32 {
	return ObjectHashCodeExecution(NewExecution(), ClassLiteral(value.declaration)) ^ reflectionIdentifierJavaString(value.name).HashCode()
}
