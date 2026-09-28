package stdjava

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
	if named, ok := value.(interface{ GetTypeNameJava2goExecution(*Execution) string }); ok {
		return named.GetTypeNameJava2goExecution(execution)
	}
	if named, ok := value.(interface{ GetTypeName() string }); ok {
		return named.GetTypeName()
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
	RegisterJavaType(JavaClassTypeID, ObjectTypeID, ReflectTypeTypeID, SerializableTypeID)
}
