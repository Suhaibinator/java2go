package stdjava

import (
	"reflect"
	"strings"
	"sync"
)

// ClassDescriptor describes the bounded public reflection surface emitted by
// the transpiler. Callbacks retain Java initialization and execution semantics.
// Descriptors are registered without constructing or initializing Java classes.
type ClassDescriptor struct {
	Type                TypeID
	Interface           bool
	InheritedAnnotation bool
	Initialize          func(*Execution)
	Construct           func(*Execution) any
	Fields              []FieldDescriptor
	Methods             []MethodDescriptor
	Annotations         []TypeID
}
type FieldDescriptor struct {
	Name, GoName string
	Type         TypeID
	Final        bool
}
type MethodDescriptor struct {
	Name, GoName string
	Return       TypeID
}

var reflectionRegistry sync.Map

func RegisterClassDescriptor(descriptor ClassDescriptor) {
	descriptor.Fields = append([]FieldDescriptor(nil), descriptor.Fields...)
	descriptor.Methods = append([]MethodDescriptor(nil), descriptor.Methods...)
	descriptor.Annotations = append([]TypeID(nil), descriptor.Annotations...)
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
func (class *Class) IsAnnotationPresent(annotation *Class) bool {
	class.TypeID()
	id := annotation.TypeID()
	inherited := classDescriptor(id).InheritedAnnotation
	for current := class; current != nil; current = current.GetSuperclass() {
		for _, candidate := range classDescriptor(current.TypeID()).Annotations {
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
	owner     *Class
	construct func(*Execution) any
}

func (class *Class) GetConstructor(parameters ...*Class) *Constructor {
	construct := classDescriptor(class.TypeID()).Construct
	if len(parameters) != 0 || construct == nil {
		panic(reflectionException("NoSuchMethodException", class.GetName()+".<init>"))
	}
	return &Constructor{class, construct}
}
func (constructor *Constructor) NewInstance(execution *Execution, arguments ...any) (result any) {
	if constructor == nil {
		panic(NewNullPointerException("constructor is null"))
	}
	if len(arguments) != 0 {
		panic(NewIllegalArgumentException("wrong number of constructor arguments"))
	}
	if initialize := classDescriptor(constructor.owner.TypeID()).Initialize; initialize != nil {
		initialize(execution)
	}
	defer reflectionInvocationPanic()
	return constructor.construct(execution)
}

type Field struct {
	owner      *Class
	descriptor FieldDescriptor
}

func (class *Class) GetField(name string) *Field {
	class.TypeID()
	if nilJavaReference(name) {
		panic(NewNullPointerException("field name is null"))
	}
	for current := class; current != nil; current = current.GetSuperclass() {
		for _, field := range classDescriptor(current.TypeID()).Fields {
			if field.Name == name {
				return &Field{current, field}
			}
		}
	}
	panic(reflectionException("NoSuchFieldException", name))
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
func (field *Field) Get(receiver any) any {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	value := reflectionReceiver(receiver, field.owner).Elem().FieldByName(field.descriptor.GoName)
	if field.descriptor.Type == StringTypeID && nilJavaReference(value.Interface()) {
		return nil
	}
	return reflectionBox(value.Interface(), field.descriptor.Type)
}
func (field *Field) Set(receiver, value any) {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	target := reflectionReceiver(receiver, field.owner).Elem().FieldByName(field.descriptor.GoName)
	if field.descriptor.Final {
		panic(reflectionException("IllegalAccessException", "final field"))
	}
	converted := reflectionUnbox(value, field.descriptor.Type)
	if nilJavaReference(converted) {
		if isPrimitiveTypeID(field.descriptor.Type) {
			panic(NewIllegalArgumentException("null primitive field value"))
		}
		if target.Kind() == reflect.String {
			target.SetString(NullString())
			return
		}
		target.SetZero()
		return
	}
	if !isPrimitiveTypeID(field.descriptor.Type) {
		actual, known := ObjectDynamicType(converted)
		if known && !JavaTypeAssignable(actual, field.descriptor.Type) {
			panic(NewIllegalArgumentException("field value has wrong Java type"))
		}
		if carrier, ok := converted.(JavaObjectInfoCarrier); ok {
			if info := carrier.JavaObjectInfo(); info != nil {
				if view := info.resolveView(field.descriptor.Type); view != nil {
					converted = view
				}
			}
		}
	}
	source := reflect.ValueOf(converted)
	if !source.Type().AssignableTo(target.Type()) {
		panic(NewIllegalArgumentException("field value has wrong type"))
	}
	target.Set(source)
}

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
		exception := reflectionException("InvocationTargetException", "reflective invocation failed")
		exception.state.cause = caught
		panic(exception)
	}
}
func reflectionException(name, message string) Exception {
	return Exception{newThrowableBase(name, message)}
}
func init() {
	RegisterException("ReflectiveOperationException", "Exception")
	for _, name := range []string{"ClassNotFoundException", "NoSuchMethodException", "NoSuchFieldException", "IllegalAccessException", "InvocationTargetException"} {
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
