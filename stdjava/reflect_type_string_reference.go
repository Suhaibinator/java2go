package stdjava

// Canonical Type text preserves source execution companions' String references
// and UTF16 directly; native metadata strings have an explicit VM ingress.
func ReflectTypeNameJavaStringExecution(execution *Execution, value ReflectType) *JavaString {
	ReferenceRequireNonNull(value)
	value = collectionObjectView(value)
	if class, ok := value.(*Class); ok {
		if _, array := arrayComponentTypeID(class.TypeID()); !array {
			return class.GetNameJavaString()
		}
		return JavaStringFromHostUTF8(class.GetTypeName())
	}
	if variable, ok := value.(*metadataTypeVariable); ok {
		return reflectionIdentifierJavaString(variable.name)
	}
	if result, ok := objectExecutionMethod(execution, value, "GetTypeNameJava2goExecution", nil); ok {
		return reflectCanonicalStringResult(result.Interface())
	}
	if named, ok := value.(interface{ GetTypeName() *JavaString }); ok {
		return named.GetTypeName()
	}
	if named, ok := value.(interface{ GetTypeName() string }); ok {
		return reflectCanonicalStringResult(named.GetTypeName())
	}
	return JavaStringValueOfExecution(execution, value)
}
func ReflectStringMemberJavaStringExecution(execution *Execution, value any, protocol TypeID, method string) *JavaString {
	if variable, ok := value.(*metadataTypeVariable); ok && protocol == TypeVariableTypeID && method == "GetName" {
		return reflectionIdentifierJavaString(variable.name)
	}
	return reflectCanonicalStringResult(reflectProtocolMember(execution, value, protocol, method))
}
func reflectCanonicalStringResult(value any) *JavaString {
	if nilJavaReference(value) {
		return nil
	}
	if text, ok := value.(*JavaString); ok {
		return text
	}
	if text, ok := value.(string); ok {
		if StringIsNull(text) {
			return nil
		}
		return JavaStringFromHostUTF8(text)
	}
	panic(NewClassCastException("reflection accessor returned a value which is not a String"))
}
