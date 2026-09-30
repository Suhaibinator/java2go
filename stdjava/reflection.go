package stdjava

import (
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

// ClassDescriptor describes the bounded reflection surface emitted by
// the transpiler. Callbacks retain Java initialization and execution semantics.
// Descriptors are registered without constructing or initializing Java classes.
type ClassDescriptor struct {
	SimpleName          string
	HasSimpleName       bool
	Type                TypeID
	Interface           bool
	Enum                bool
	InheritedAnnotation bool
	Initialize          func(*Execution)
	Construct           func(*Execution) any
	Fields              []FieldDescriptor
	Methods             []MethodDescriptor
	Annotations         []TypeID
	Modifiers           int32
	HasModifiers        bool
	GenericSuperclass   *ReflectTypeDescriptor
	TypeParameters      []TypeVariableDescriptor
	genericVariables    []*metadataTypeVariable
	Constructors        []ConstructorDescriptor
	AnnotationValues    []AnnotationDescriptor
	// Initialization is the shared coordinator used by direct active uses.
	// A callback that enters an independently used coordinator must bind it
	// here; without one, Initialize is a reflection-owned initialization body.
	Initialization *ClassInitialization
}
type FieldDescriptor struct {
	nameJavaString *JavaString
	Name, GoName   string
	Type           TypeID
	Final          bool
	NonPublic      bool
	EnumConstant   bool
	StaticGet      func(*Execution) any
	Modifiers      int32
	HasModifiers   bool
	GenericType    *ReflectTypeDescriptor
	Annotations    []AnnotationDescriptor
	Get            func(*Execution, any) any
	Set            func(*Execution, any, any)
	StaticSet      func(*Execution, any)
}
type MethodDescriptor struct {
	nameJavaString *JavaString
	Name, GoName   string
	Return         TypeID
}

var reflectionRegistry sync.Map

func RegisterClassDescriptor(descriptor ClassDescriptor) {
	if initialize := descriptor.Initialize; initialize != nil {
		state := descriptor.Initialization
		if state == nil {
			// Hand-authored descriptors supply an initialization callback. Give every
			// reflected member and Class.forName one shared, reentrant coordinator.
			state = NewClassInitialization(string(descriptor.Type))
			descriptor.Initialization = state
			descriptor.Initialize = func(execution *Execution) { state.Ensure(execution, initialize) }
		} else {
			// Generated callbacks already enter this coordinator. Do not wrap them
			// in Ensure again: that would be mistaken for recursive initialization.
			descriptor.Initialize = func(execution *Execution) {
				requireExecution(execution)
				if !state.isInitialized() {
					initialize(execution)
				}
			}
		}
	}
	descriptor = cloneReflectionMetadata(descriptor)
	descriptor.Fields = append([]FieldDescriptor(nil), descriptor.Fields...)
	descriptor.Methods = append([]MethodDescriptor(nil), descriptor.Methods...)
	descriptor.Annotations = append([]TypeID(nil), descriptor.Annotations...)
	for index := range descriptor.Fields {
		descriptor.Fields[index].nameJavaString = reflectionIdentifierJavaString(descriptor.Fields[index].Name)
	}
	for index := range descriptor.Methods {
		descriptor.Methods[index].nameJavaString = reflectionIdentifierJavaString(descriptor.Methods[index].Name)
	}
	reflectionRegistry.Store(descriptor.Type, descriptor)
}
func classDescriptor(id TypeID) ClassDescriptor {
	if descriptor, ok := reflectionRegistry.Load(id); ok {
		return descriptor.(ClassDescriptor)
	}
	return ClassDescriptor{Type: id}
}
func ClassForName(execution *Execution, name string) *Class {
	if nilJavaReference(name) {
		panic(NewNullPointerException("class name is null"))
	}
	id := TypeID(name)
	javaTypeRegistry.RLock()
	_, exists := javaTypeRegistry.types[id]
	javaTypeRegistry.RUnlock()
	if !exists {
		panic(reflectionException("ClassNotFoundException", name))
	}
	if initialize := classDescriptor(id).Initialize; initialize != nil {
		initialize(execution)
	}
	return ClassLiteral(id)
}
func (class *Class) GetName() string {
	return strings.TrimPrefix(string(class.TypeID()), primitiveTypePrefix)
}

// GetSimpleName uses source metadata for nested/local/anonymous classes and
// recursively derives array names from their component descriptors.
func (class *Class) GetSimpleName() string {
	id := class.TypeID()
	if component, ok := arrayComponentTypeID(id); ok {
		return ClassLiteral(component).GetSimpleName() + "[]"
	}
	if descriptor := classDescriptor(id); descriptor.HasSimpleName {
		return descriptor.SimpleName
	}
	name := strings.TrimPrefix(string(id), primitiveTypePrefix)
	if index := strings.LastIndexAny(name, ".$"); index >= 0 {
		name = name[index+1:]
	}
	return name
}

func (class *Class) GetSuperclass() *Class {
	id := class.TypeID()
	if classDescriptor(id).Interface {
		return nil
	}
	javaTypeRegistry.RLock()
	parent := javaTypeRegistry.types[id].super
	javaTypeRegistry.RUnlock()
	if parent == "" {
		return nil
	}
	return ClassLiteral(parent)
}
func (class *Class) IsAssignableFrom(other *Class) bool {
	return JavaTypeAssignable(other.TypeID(), class.TypeID())
}
func (class *Class) IsEnum() bool {
	return classDescriptor(class.TypeID()).Enum
}
func (class *Class) IsAnnotationPresent(annotation *Class) bool {
	class.TypeID()
	id := annotation.TypeID()
	inherited := classDescriptor(id).InheritedAnnotation
	for current := class; current != nil; current = current.GetSuperclass() {
		descriptor := classDescriptor(current.TypeID())
		for _, candidate := range descriptor.AnnotationValues {
			if candidate.Type == id {
				return true
			}
		}
		for _, candidate := range descriptor.Annotations {
			if candidate == id {
				return true
			}
		}
		if !inherited {
			break
		}
	}
	return false
}

type Constructor struct {
	owner      *Class
	descriptor ConstructorDescriptor
	accessible atomic.Bool
}

type Field struct {
	owner      *Class
	descriptor FieldDescriptor
	accessible atomic.Bool
}

func (class *Class) GetField(name string) *Field {
	class.TypeID()
	if nilJavaReference(name) {
		panic(NewNullPointerException("field name is null"))
	}
	for current := class; current != nil; current = current.GetSuperclass() {
		for _, field := range classDescriptor(current.TypeID()).Fields {
			if field.Name == name && reflectionFieldPublic(field) {
				return &Field{owner: current, descriptor: field}
			}
		}
	}
	panic(reflectionException("NoSuchFieldException", name))
}

// GetDeclaredField includes non-public metadata but searches only this class.
// Finding a descriptor does not grant access to read or write the field.
func (class *Class) GetDeclaredField(name string) *Field {
	class.TypeID()
	if nilJavaReference(name) {
		panic(NewNullPointerException("field name is null"))
	}
	for _, field := range classDescriptor(class.TypeID()).Fields {
		if field.Name == name {
			return &Field{owner: class, descriptor: field}
		}
	}
	panic(reflectionException("NoSuchFieldException", name))
}
func (field *Field) IsEnumConstant() bool {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	return field.descriptor.EnumConstant
}
func (field *Field) GetName() string {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	return field.descriptor.Name
}
func reflectionReceiver(receiver any, owner *Class) reflect.Value {
	if nilJavaReference(receiver) {
		panic(NewNullPointerException("reflection receiver is null"))
	}
	id, ok := ObjectDynamicType(receiver)
	if !ok || !JavaTypeAssignable(id, owner.TypeID()) {
		panic(NewIllegalArgumentException("receiver has wrong class"))
	}
	if carrier, ok := receiver.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			if view := info.resolveView(owner.TypeID()); view != nil {
				return reflect.ValueOf(view)
			}
		}
	}
	return reflect.ValueOf(receiver)
}

// requireInstanceReceiver preserves Field.checkAccess ordering: an instance
// null fails before access permission, but receiver type checking remains in
// the accessor. Static descriptors ignore the supplied receiver entirely.

type Method struct {
	owner      *Class
	descriptor MethodDescriptor
}

func (class *Class) GetMethod(name string, parameters ...*Class) *Method {
	class.TypeID()
	if nilJavaReference(name) {
		panic(NewNullPointerException("method name is null"))
	}
	if len(parameters) == 0 {
		for current := class; current != nil; current = current.GetSuperclass() {
			for _, method := range classDescriptor(current.TypeID()).Methods {
				if method.Name == name {
					return &Method{current, method}
				}
			}
		}
	}
	panic(reflectionException("NoSuchMethodException", name))
}
func (method *Method) GetName() string {
	if method == nil {
		panic(NewNullPointerException("method is null"))
	}
	return method.descriptor.Name
}
func (method *Method) Invoke(execution *Execution, receiver any, arguments ...any) any {
	if method == nil {
		panic(NewNullPointerException("method is null"))
	}
	if len(arguments) != 0 {
		panic(NewIllegalArgumentException("wrong number of method arguments"))
	}
	// Validate against the declaring class, then select the most-derived
	// override. The generated execution body itself is statically bound.
	targetReceiver := reflectionReceiver(receiver, method.owner)
	selected := method.descriptor
	if actual, ok := ObjectDynamicType(receiver); ok {
		for current := ClassLiteral(actual); current != nil; current = current.GetSuperclass() {
			found := false
			for _, candidate := range classDescriptor(current.TypeID()).Methods {
				if candidate.Name == method.descriptor.Name {
					selected = candidate
					targetReceiver = reflectionReceiver(receiver, current)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	target := targetReceiver.MethodByName(selected.GoName)
	defer reflectionInvocationPanic()
	values := target.Call([]reflect.Value{reflect.ValueOf(execution)})
	if len(values) == 0 {
		return nil
	}
	return reflectionBox(values[0].Interface(), method.descriptor.Return)
}
func reflectionInvocationPanic() {
	if caught := recover(); caught != nil {
		exception := Exception{newJavaThrowableBase("InvocationTargetException", nil)}
		exception.state.cause = caught
		exception.state.causeInitialized = true
		panic(exception)
	}
}
func reflectionException(name, message string) Exception {
	return Exception{newThrowableBase(name, message)}
}
func init() {
	RegisterException("ReflectiveOperationException", "Exception")
	for _, name := range []string{"ClassNotFoundException", "NoSuchMethodException", "NoSuchFieldException", "IllegalAccessException", "InstantiationException", "InvocationTargetException"} {
		RegisterException(name, "ReflectiveOperationException")
	}
}
func reflectionBox(value any, id TypeID) any {
	switch id {
	case PrimitiveBooleanTypeID:
		return BoxBoolean(value.(bool))
	case PrimitiveByteTypeID:
		return BoxByte(value.(int8))
	case PrimitiveShortTypeID:
		return BoxShort(value.(int16))
	case PrimitiveCharTypeID:
		return BoxCharacter(value.(rune))
	case PrimitiveIntTypeID:
		return BoxInteger(value.(int32))
	case PrimitiveLongTypeID:
		return BoxLong(value.(int64))
	case PrimitiveFloatTypeID:
		return BoxFloat(value.(float32))
	case PrimitiveDoubleTypeID:
		return BoxDouble(value.(float64))
	}
	if nilJavaReference(value) {
		return nil
	}
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			if view := info.resolveView(info.DynamicType()); view != nil {
				return view
			}
		}
	}
	return value
}

// Reflection permits unboxing followed by primitive widening, never narrowing.
func reflectionUnbox(value any, id TypeID) any {
	if !isPrimitiveTypeID(id) {
		return value
	}
	var primitive any
	var from TypeID
	switch box := value.(type) {
	case *Boolean:
		if box != nil {
			primitive, from = UnboxBoolean(box), PrimitiveBooleanTypeID
		}
	case *Byte:
		if box != nil {
			primitive, from = UnboxByte(box), PrimitiveByteTypeID
		}
	case *Short:
		if box != nil {
			primitive, from = UnboxShort(box), PrimitiveShortTypeID
		}
	case *Character:
		if box != nil {
			primitive, from = UnboxCharacter(box), PrimitiveCharTypeID
		}
	case *Integer:
		if box != nil {
			primitive, from = UnboxInteger(box), PrimitiveIntTypeID
		}
	case *Long:
		if box != nil {
			primitive, from = UnboxLong(box), PrimitiveLongTypeID
		}
	case *Float:
		if box != nil {
			primitive, from = UnboxFloat(box), PrimitiveFloatTypeID
		}
	case *Double:
		if box != nil {
			primitive, from = UnboxDouble(box), PrimitiveDoubleTypeID
		}
	}
	if from == id {
		return primitive
	}
	allowed := map[TypeID][]TypeID{
		PrimitiveByteTypeID:  {PrimitiveShortTypeID, PrimitiveIntTypeID, PrimitiveLongTypeID, PrimitiveFloatTypeID, PrimitiveDoubleTypeID},
		PrimitiveShortTypeID: {PrimitiveIntTypeID, PrimitiveLongTypeID, PrimitiveFloatTypeID, PrimitiveDoubleTypeID},
		PrimitiveCharTypeID:  {PrimitiveIntTypeID, PrimitiveLongTypeID, PrimitiveFloatTypeID, PrimitiveDoubleTypeID},
		PrimitiveIntTypeID:   {PrimitiveLongTypeID, PrimitiveFloatTypeID, PrimitiveDoubleTypeID},
		PrimitiveLongTypeID:  {PrimitiveFloatTypeID, PrimitiveDoubleTypeID},
		PrimitiveFloatTypeID: {PrimitiveDoubleTypeID},
	}
	targets := map[TypeID]reflect.Type{
		PrimitiveShortTypeID: reflect.TypeOf(int16(0)), PrimitiveIntTypeID: reflect.TypeOf(int32(0)),
		PrimitiveLongTypeID: reflect.TypeOf(int64(0)), PrimitiveFloatTypeID: reflect.TypeOf(float32(0)), PrimitiveDoubleTypeID: reflect.TypeOf(float64(0)),
	}
	for _, destination := range allowed[from] {
		if destination == id {
			return reflect.ValueOf(primitive).Convert(targets[id]).Interface()
		}
	}
	panic(NewIllegalArgumentException("field value cannot be unboxed and widened to field type"))
}

// ReflectionArguments expands an explicitly supplied Object[] varargs array.
// A null array denotes zero arguments, as it does in java.lang.reflect.
func ReflectionArguments(array any) []any {
	if nilJavaReference(array) {
		return nil
	}
	if reference, ok := array.(*ReferenceArray); ok {
		return ReferenceArrayIterationElements(reference)
	}
	values := reflect.ValueOf(array)
	if values.Kind() != reflect.Slice {
		panic(NewIllegalArgumentException("reflection arguments must be a reference array"))
	}
	arguments := make([]any, values.Len())
	for i := range arguments {
		arguments[i] = values.Index(i).Interface()
	}
	return arguments
}
func ReflectionClassArguments(array any) []*Class {
	values := ReflectionArguments(array)
	parameters := make([]*Class, len(values))
	for i, value := range values {
		if nilJavaReference(value) {
			continue
		}
		class, ok := value.(*Class)
		if !ok {
			panic(NewIllegalArgumentException("reflection parameter type must be Class"))
		}
		parameters[i] = class
	}
	return parameters
}
