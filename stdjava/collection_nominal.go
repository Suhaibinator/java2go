package stdjava

const (
	nativeCollectionTypeID TypeID = "java.util.Collection"
	nativeSetTypeID        TypeID = "java.util.Set"
	nativeListTypeID       TypeID = "java.util.List"
)

func init() {
	RegisterJavaType(nativeCollectionTypeID, ObjectTypeID, IterableTypeID)
	RegisterJavaType(nativeSetTypeID, ObjectTypeID, nativeCollectionTypeID)
	RegisterJavaType(nativeListTypeID, ObjectTypeID, nativeCollectionTypeID)
}

// Only runtime types can implement this private carrier. It records proven
// interface membership without inventing a concrete class for shared backends.
// Source objects continue using their generated nominal descriptors.
func nativeJavaInterfaceAssignable(value any, expected TypeID) bool {
	carrier, ok := value.(interface{ nativeJavaInterfaces() []TypeID })
	if !ok || javaReferenceIsNull(value) {
		return false
	}
	for _, nominal := range carrier.nativeJavaInterfaces() {
		if JavaTypeAssignable(nominal, expected) {
			return true
		}
	}
	return false
}
