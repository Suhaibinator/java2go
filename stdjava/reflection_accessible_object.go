package stdjava

import "sync/atomic"

// JavaAccessibleObject is the native capability of an actual reflected member.
// Its private state accessor prevents a Go method-set lookalike from acquiring
// an access override. Java casts additionally require the nominal JDK TypeID.
type JavaAccessibleObject interface {
	reflectionAccessState() *atomic.Bool
	GetModifiers() int32
	GetDeclaringClass() *Class
	GetAnnotationExecution(*Execution, *Class) any
	GetAnnotationsExecution(*Execution) *ReferenceArray
}

// JavaMember is the physical view used by the native Field/Method/Constructor
// implementations. The Java Member identity remains a nominal registry edge.
type JavaMember interface {
	JavaAccessibleObject
	GetName() string
	IsSynthetic() bool
}

func (field *Field) reflectionAccessState() *atomic.Bool {
	ReferenceRequireNonNull(field)
	return &field.accessible
}
func (method *Method) reflectionAccessState() *atomic.Bool {
	method.requireMethod()
	return &method.accessible
}
func (constructor *Constructor) reflectionAccessState() *atomic.Bool {
	ReferenceRequireNonNull(constructor)
	return &constructor.accessible
}
func (field *Field) IsSynthetic() bool             { return field.GetModifiers()&4096 != 0 }
func (constructor *Constructor) IsSynthetic() bool { return constructor.GetModifiers()&4096 != 0 }
func (constructor *Constructor) GetName() string   { return constructor.GetDeclaringClass().GetName() }

func AccessibleObjectIsAccessible(member JavaAccessibleObject) bool {
	ReferenceRequireNonNull(member)
	return member.reflectionAccessState().Load()
}

func ReflectionMemberName(member JavaMember) *JavaString {
	ReferenceRequireNonNull(member)
	return uriRuntimeText(member.GetName())
}

// FieldStringExecution formats the actual declaration, without reading storage
// or triggering declaring-class initialization.
func FieldStringExecution(execution *Execution, field *Field) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(field)
	return uriRuntimeText(reflectionMemberDescription(field))
}
