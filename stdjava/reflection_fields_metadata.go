package stdjava

import "reflect"

const (
	ReflectFieldTypeID               TypeID = "java.lang.reflect.Field"
	ReflectConstructorTypeID         TypeID = "java.lang.reflect.Constructor"
	AnnotationTypeID                 TypeID = "java.lang.annotation.Annotation"
	reflectionAccessibleObjectTypeID TypeID = "java.lang.reflect.AccessibleObject"
	reflectionExecutableTypeID       TypeID = "java.lang.reflect.Executable"
	reflectionMemberTypeID           TypeID = "java.lang.reflect.Member"
)

func init() {
	RegisterJavaType(AnnotationTypeID, ObjectTypeID)
	RegisterJavaType(reflectionMemberTypeID, ObjectTypeID)
	RegisterJavaType(reflectionAccessibleObjectTypeID, ObjectTypeID, AnnotatedElementTypeID)
	RegisterJavaType(reflectionExecutableTypeID, reflectionAccessibleObjectTypeID, reflectionMemberTypeID, GenericDeclarationTypeID)
	RegisterJavaType(ReflectFieldTypeID, reflectionAccessibleObjectTypeID, reflectionMemberTypeID)
	RegisterJavaType(ReflectConstructorTypeID, reflectionExecutableTypeID)
}
func (*Field) JavaDynamicTypeID() TypeID       { return ReflectFieldTypeID }
func (*Constructor) JavaDynamicTypeID() TypeID { return ReflectConstructorTypeID }
func (class *Class) GetModifiers() int32 {
	descriptor := classDescriptor(class.TypeID())
	if descriptor.HasModifiers {
		return descriptor.Modifiers
	}
	modifiers := int32(1)
	if descriptor.Interface {
		modifiers |= 512 | 1024
	}
	if descriptor.Enum {
		modifiers |= 16384
	}
	return modifiers
}
func (class *Class) GetDeclaredFields() *ReferenceArray {
	fields := classDescriptor(class.TypeID()).Fields
	result := NewReferenceArray(len(fields), ReflectFieldTypeID)
	for index, descriptor := range fields {
		ReferenceArraySet(result, index, &Field{owner: class, descriptor: descriptor})
	}
	return result
}
func (field *Field) GetModifiers() int32 {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	if field.descriptor.HasModifiers {
		return field.descriptor.Modifiers
	}
	modifiers := int32(1)
	if field.descriptor.NonPublic {
		modifiers = 2
	}
	if field.descriptor.StaticGet != nil || field.descriptor.StaticSet != nil {
		modifiers |= 8
	}
	if field.descriptor.Final {
		modifiers |= 16
	}
	if field.descriptor.EnumConstant {
		modifiers |= 16384
	}
	return modifiers
}
func (field *Field) GetType() *Class {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	return ClassLiteral(field.descriptor.Type)
}
func (field *Field) GetDeclaringClass() *Class {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	return field.owner
}
func (field *Field) SetAccessible(enabled bool) {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	field.accessible.Store(enabled)
}
func (field *Field) isStatic() bool { return field.GetModifiers()&8 != 0 }
func (field *Field) accessibleFromPublicCaller() bool {
	// Without a caller-class argument, ordinary lookup checks the public access
	// boundary. Generated source can explicitly override access on this wrapper.
	if field.accessible.Load() {
		return true
	}
	return field.GetModifiers()&1 != 0 && (field.owner == nil || field.owner.GetModifiers()&1 != 0)
}
func (field *Field) requireInstanceReceiver(receiver any) {
	if !field.isStatic() && nilJavaReference(receiver) {
		panic(NewNullPointerException("reflection receiver is null"))
	}
}
func (field *Field) CanAccess(receiver any) bool {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	if field.isStatic() {
		if !nilJavaReference(receiver) {
			panic(NewIllegalArgumentException("non-null object for static field"))
		}
	} else {
		reflectionReceiver(receiver, field.owner)
	}
	return field.accessibleFromPublicCaller()
}
func (field *Field) Get(receiver any) any { return field.GetExecution(NewExecution(), receiver) }
func (field *Field) GetExecution(execution *Execution, receiver any) any {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	field.requireInstanceReceiver(receiver)
	if !field.accessibleFromPublicCaller() {
		panic(reflectionException("IllegalAccessException", "non-public field"))
	}
	requireExecution(execution)
	var value any
	if field.isStatic() {
		if initialize := classDescriptor(field.owner.TypeID()).Initialize; initialize != nil {
			initialize(execution)
		}
		if field.descriptor.StaticGet == nil {
			panic(NewUnsupportedOperationException("static field getter metadata is unavailable"))
		}
		value = field.descriptor.StaticGet(execution)
	} else {
		target := reflectionReceiver(receiver, field.owner)
		if field.descriptor.Get != nil {
			value = field.descriptor.Get(execution, target.Interface())
		} else {
			value = target.Elem().FieldByName(field.descriptor.GoName).Interface()
		}
	}
	return reflectionBox(value, field.descriptor.Type)
}
func (field *Field) Set(receiver, value any) { field.SetExecution(NewExecution(), receiver, value) }
func (field *Field) SetExecution(execution *Execution, receiver, value any) {
	field.setExecution(execution, receiver, value, reflectionFieldValue)
}

// Typed setters share receiver, access, final and initialization checks. Their
// primitive-only conversion runs at the same point as the ordinary Set path.
func (field *Field) setExecution(execution *Execution, receiver, value any, convert func(any, TypeID) any) {
	if field == nil {
		panic(NewNullPointerException("field is null"))
	}
	field.requireInstanceReceiver(receiver)
	if !field.accessibleFromPublicCaller() {
		panic(reflectionException("IllegalAccessException", "non-public field"))
	}
	requireExecution(execution)
	// Static accessor creation initializes the declaring class before rejecting
	// a public final write; access denial above still precedes initialization.
	if field.isStatic() {
		if initialize := classDescriptor(field.owner.TypeID()).Initialize; initialize != nil {
			initialize(execution)
		}
		if field.GetModifiers()&16 != 0 {
			panic(reflectionException("IllegalAccessException", "final field"))
		}
		if field.descriptor.StaticSet == nil {
			panic(reflectionException("IllegalAccessException", "static field is not writable"))
		}
		converted := convert(value, field.descriptor.Type)
		field.descriptor.StaticSet(execution, converted)
		return
	}
	target := reflectionReceiver(receiver, field.owner)
	// Read-only instance accessors validate the receiver before rejecting a
	// final write. An access override supports ordinary instance final fields.
	if field.GetModifiers()&16 != 0 && !field.accessible.Load() {
		panic(reflectionException("IllegalAccessException", "final field"))
	}
	converted := convert(value, field.descriptor.Type)
	if field.descriptor.Set != nil {
		field.descriptor.Set(execution, target.Interface(), converted)
		return
	}
	slot := target.Elem().FieldByName(field.descriptor.GoName)
	if nilJavaReference(converted) {
		if slot.Kind() == reflect.String {
			slot.SetString(NullString())
		} else {
			slot.SetZero()
		}
		return
	}
	source := reflect.ValueOf(converted)
	if !source.Type().AssignableTo(slot.Type()) {
		panic(NewIllegalArgumentException("field value has wrong type"))
	}
	slot.Set(source)
}
func reflectionFieldValue(value any, id TypeID) any {
	converted := reflectionUnbox(value, id)
	if nilJavaReference(converted) {
		if isPrimitiveTypeID(id) {
			panic(NewIllegalArgumentException("null primitive field value"))
		}
		return nil
	}
	if !isPrimitiveTypeID(id) {
		actual, known := ObjectDynamicType(converted)
		if known && !JavaTypeAssignable(actual, id) {
			panic(NewIllegalArgumentException("field value has wrong Java type"))
		}
		if carrier, ok := converted.(JavaObjectInfoCarrier); ok {
			if info := carrier.JavaObjectInfo(); info != nil {
				if view := info.resolveView(id); view != nil {
					converted = view
				}
			}
		}
	}
	return converted
}

// Internal exported storage accessors are statically selected on the declaring
// class view, including its concrete generic instantiation. They are not Java
// Method.invoke callbacks and do not receive target-exception wrappers.
func ReflectGeneratedFieldGetExecution(execution *Execution, receiver any, declaringType TypeID, method string) any {
	requireExecution(execution)
	target := reflectionReceiver(receiver, ClassLiteral(declaringType)).MethodByName(method)
	if !target.IsValid() || target.Type().NumIn() != 1 || target.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || target.Type().NumOut() != 1 {
		panic(NewUnsupportedOperationException("generated field getter signature is unavailable"))
	}
	return target.Call([]reflect.Value{reflect.ValueOf(execution)})[0].Interface()
}
func ReflectGeneratedFieldSetExecution(execution *Execution, receiver any, declaringType TypeID, method string, value any) {
	requireExecution(execution)
	target := reflectionReceiver(receiver, ClassLiteral(declaringType)).MethodByName(method)
	if !target.IsValid() || target.Type().NumIn() != 2 || target.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || target.Type().NumOut() != 0 {
		panic(NewUnsupportedOperationException("generated field setter signature is unavailable"))
	}
	expected := target.Type().In(1)
	argument := reflect.Zero(expected)
	if !nilJavaReference(value) {
		argument = reflect.ValueOf(value)
		if !argument.Type().AssignableTo(expected) {
			panic(NewIllegalArgumentException("generated field value has wrong type"))
		}
	}
	target.Call([]reflect.Value{reflect.ValueOf(execution), argument})
}

func reflectionFieldPublic(descriptor FieldDescriptor) bool {
	if descriptor.HasModifiers {
		return descriptor.Modifiers&1 != 0
	}
	return !descriptor.NonPublic
}
