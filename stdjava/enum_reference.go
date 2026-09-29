package stdjava

// JavaEnum is the nominal enum reference carried by generated enum objects.
// Metadata lives in the original Java object; the protocol does not introduce
// a separate wrapper or alter that object's identity.
type JavaEnum interface {
	JavaEnumMetadata() *EnumMetadata
}

const EnumTypeID TypeID = "java.lang.Enum"

// EnumMetadata records the construction-time identity of one enum constant.
// Its fields are private and its public API provides no mutation after seeding.
type EnumMetadata struct {
	name      string
	ordinal   int32
	declaring TypeID
	dynamic   TypeID
}

func NewEnumMetadata(name string, ordinal int32, declaring, dynamic TypeID) EnumMetadata {
	return EnumMetadata{name: name, ordinal: ordinal, declaring: declaring, dynamic: dynamic}
}
func (metadata *EnumMetadata) Name() string            { return metadata.name }
func (metadata *EnumMetadata) Ordinal() int32          { return metadata.ordinal }
func (metadata *EnumMetadata) DeclaringTypeID() TypeID { return metadata.declaring }
func (metadata *EnumMetadata) DynamicTypeID() TypeID   { return metadata.dynamic }

func init() {
	RegisterJavaType(EnumTypeID, ObjectTypeID, ComparableTypeID, SerializableTypeID, ConstableTypeID)
}

func enumReferenceMetadata(value JavaEnum) *EnumMetadata {
	if nilJavaReference(value) {
		panic(NewNullPointerException("enum reference is null"))
	}
	actual, known := ObjectDynamicType(value)
	if !known || !JavaTypeAssignable(actual, EnumTypeID) {
		panic(NewClassCastException("reference is not a Java enum"))
	}
	metadata := value.JavaEnumMetadata()
	if metadata == nil || metadata.dynamic != actual || metadata.declaring == "" || !JavaTypeAssignable(metadata.declaring, EnumTypeID) || !JavaTypeAssignable(actual, metadata.declaring) {
		panic(NewClassCastException("reference has incompatible enum metadata"))
	}
	return metadata
}

func EnumName(value JavaEnum) string   { return enumReferenceMetadata(value).Name() }
func EnumOrdinal(value JavaEnum) int32 { return enumReferenceMetadata(value).Ordinal() }
func EnumGetDeclaringClass(value JavaEnum) *Class {
	return ClassLiteral(enumReferenceMetadata(value).DeclaringTypeID())
}
func EnumCompareTo(value, other JavaEnum) int32 {
	left := enumReferenceMetadata(value)
	right := enumReferenceMetadata(other)
	if left.DeclaringTypeID() != right.DeclaringTypeID() {
		panic(NewClassCastException("enum constants have different declaring classes"))
	}
	return left.Ordinal() - right.Ordinal()
}
