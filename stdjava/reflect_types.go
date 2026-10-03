package stdjava

import "reflect"

const (
	JavaClassTypeID   TypeID = "java.lang.Class"
	ReflectTypeTypeID TypeID = "java.lang.reflect.Type"
)

// ReflectType is erased because java.lang.reflect.Type has no abstract methods:
// its getTypeName default calls virtual toString. Nominal membership is enforced
// by the Java type registry, including generated implementations of Type.
type ReflectType = any

func (*Class) JavaDynamicTypeID() TypeID { return JavaClassTypeID }

// GetTypeName renders arrays in Java source notation while retaining binary
// nested-class names. Class.getName has a different array-name contract.
func (class *Class) GetTypeName() string {
	id := class.TypeID()
	if component, ok := arrayComponentTypeID(id); ok {
		return ClassLiteral(component).GetTypeName() + "[]"
	}
	return class.GetName()
}

func ReflectTypeNameExecution(execution *Execution, value ReflectType) string {
	ReferenceRequireNonNull(value)
	value = collectionObjectView(value)
	if result, ok := objectExecutionMethod(execution, value, "GetTypeNameJava2goExecution", nil); ok {
		return result.String()
	}
	if named, ok := value.(interface{ GetTypeName() string }); ok {
		return named.GetTypeName()
	}
	if registeredJavaSourceValue(value) {
		return StringValueOfExecution(execution, value)
	}
	// The default is virtual Object.toString, including Object's own default
	// when the implementation declares neither method. fmt.Sprint would expose
	// the generated Go struct in that case.
	if stringer, ok := value.(executionStringer); ok {
		return stringer.StringJava2goExecution(execution)
	}
	if rendered, ok := callCollisionSafeExecutionStringer(execution, value); ok {
		return rendered
	}
	if stringer, ok := value.(interface{ String() string }); ok {
		return stringer.String()
	}
	return ObjectDefaultStringExecution(execution, value)
}

func init() {
	RegisterJavaType(ReflectTypeTypeID, ObjectTypeID)
	RegisterJavaType(AnnotatedElementTypeID, ObjectTypeID)
	RegisterJavaType(GenericDeclarationTypeID, ObjectTypeID, AnnotatedElementTypeID)
	RegisterJavaType(ParameterizedTypeTypeID, ObjectTypeID, ReflectTypeTypeID)
	RegisterJavaType(GenericArrayTypeTypeID, ObjectTypeID, ReflectTypeTypeID)
	RegisterJavaType(WildcardTypeTypeID, ObjectTypeID, ReflectTypeTypeID)
	RegisterJavaType(TypeVariableTypeID, ObjectTypeID, ReflectTypeTypeID, AnnotatedElementTypeID)
	RegisterJavaType(JavaClassTypeID, ObjectTypeID, ReflectTypeTypeID, SerializableTypeID, GenericDeclarationTypeID, AnnotatedElementTypeID)
}

const (
	ParameterizedTypeTypeID  TypeID = "java.lang.reflect.ParameterizedType"
	GenericArrayTypeTypeID   TypeID = "java.lang.reflect.GenericArrayType"
	WildcardTypeTypeID       TypeID = "java.lang.reflect.WildcardType"
	TypeVariableTypeID       TypeID = "java.lang.reflect.TypeVariable"
	GenericDeclarationTypeID TypeID = "java.lang.reflect.GenericDeclaration"
	AnnotatedElementTypeID   TypeID = "java.lang.reflect.AnnotatedElement"
)

// Reflection protocols use nominal Java identities and erased result values.
// Java permits covariant source accessors (for example Class getRawType), which
// cannot satisfy a Go interface requiring an exactly matching any return type.
// Accessor dispatch below enforces the actual method contract instead.
type ParameterizedType = any
type GenericArrayType = any
type WildcardType = any
type TypeVariable = any
type GenericDeclaration = any
type AnnotatedElement = any

func reflectProtocolMember(execution *Execution, value any, protocol TypeID, method string) any {
	ReferenceRequireNonNull(value)
	if !ObjectInstanceOf(value, protocol) {
		panic(NewClassCastException("receiver does not implement " + string(protocol)))
	}
	value = collectionObjectView(value)
	if result, ok := objectExecutionMethod(execution, value, method+"Java2goExecution", nil); ok {
		return result.Interface()
	}
	// Runtime-provided implementations may have a public accessor without a
	// generated execution companion. Source companions are always preferred.
	target := reflect.ValueOf(value).MethodByName(method)
	if !target.IsValid() || target.Type().NumIn() != 0 || target.Type().NumOut() != 1 {
		panic(NewUnsupportedOperationException("reflection protocol accessor is unavailable: " + method))
	}
	return target.Call(nil)[0].Interface()
}
func ReflectTypeMemberExecution(execution *Execution, value any, protocol TypeID, method string) ReflectType {
	result := reflectProtocolMember(execution, value, protocol, method)
	if javaReferenceIsNull(result) {
		return nil
	}
	if !ObjectInstanceOf(result, ReflectTypeTypeID) {
		panic(NewClassCastException("reflection accessor returned a value which is not a Type"))
	}
	return result
}
func ReflectArrayMemberExecution(execution *Execution, value any, protocol TypeID, method string, component TypeID) *ReferenceArray {
	result := reflectProtocolMember(execution, value, protocol, method)
	if javaReferenceIsNull(result) {
		return nil
	}
	array, ok := result.(*ReferenceArray)
	if !ok || !JavaTypeAssignable(array.JavaArrayTypeID(), ArrayTypeID(component)) {
		panic(NewClassCastException("reflection accessor returned an incompatible array"))
	}
	return array
}
func ReflectStringMemberExecution(execution *Execution, value any, protocol TypeID, method string) string {
	result := reflectProtocolMember(execution, value, protocol, method)
	if javaReferenceIsNull(result) {
		return NullString()
	}
	return StringRequireNonNull(result)
}
func ReflectDeclarationMemberExecution(execution *Execution, value any, protocol TypeID, method string) GenericDeclaration {
	result := reflectProtocolMember(execution, value, protocol, method)
	if javaReferenceIsNull(result) {
		return nil
	}
	if !ObjectInstanceOf(result, GenericDeclarationTypeID) {
		panic(NewClassCastException("reflection accessor returned a value which is not a GenericDeclaration"))
	}
	return result
}
