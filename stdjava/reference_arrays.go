package stdjava

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// TypeID is a qualified runtime identity for a Java reference or primitive
// type. Generated code uses binary-style qualified class names, avoiding the
// simple-name collisions that are possible across Java packages.
type TypeID string

const (
	ObjectTypeID       TypeID = "java.lang.Object"
	CloneableTypeID    TypeID = "java.lang.Cloneable"
	SerializableTypeID TypeID = "java.io.Serializable"
	ComparableTypeID   TypeID = "java.lang.Comparable"
	CharSequenceTypeID TypeID = "java.lang.CharSequence"
	ConstableTypeID    TypeID = "java.lang.constant.Constable"
	ConstantDescTypeID TypeID = "java.lang.constant.ConstantDesc"
	StringTypeID       TypeID = "java.lang.String"
)

const (
	primitiveTypePrefix = "primitive:"
	arrayTypePrefix     = "array:"
)

// PrimitiveTypeID returns the descriptor used for one Java primitive type.
func PrimitiveTypeID(name string) TypeID {
	return TypeID(primitiveTypePrefix + strings.TrimSpace(name))
}

// ArrayTypeID returns the descriptor for an array whose immediate component
// has componentType. Descriptors nest, so int[][] is represented as an array
// whose component is ArrayTypeID(PrimitiveTypeID("int")).
func ArrayTypeID(componentType TypeID) TypeID {
	return TypeID(arrayTypePrefix + string(componentType))
}

func isPrimitiveTypeID(id TypeID) bool {
	return strings.HasPrefix(string(id), primitiveTypePrefix)
}

func validPrimitiveTypeID(id TypeID) bool {
	switch strings.TrimPrefix(string(id), primitiveTypePrefix) {
	case "boolean", "byte", "short", "char", "int", "long", "float", "double":
		return isPrimitiveTypeID(id)
	default:
		return false
	}
}

func arrayComponentTypeID(id TypeID) (TypeID, bool) {
	value := string(id)
	if !strings.HasPrefix(value, arrayTypePrefix) {
		return "", false
	}
	return TypeID(strings.TrimPrefix(value, arrayTypePrefix)), true
}

func validArrayTypeID(id TypeID) bool {
	component, ok := arrayComponentTypeID(id)
	if !ok || component == "" {
		return false
	}
	if isPrimitiveTypeID(component) {
		return validPrimitiveTypeID(component)
	}
	if _, nested := arrayComponentTypeID(component); nested {
		return validArrayTypeID(component)
	}
	return true
}

func validReferenceComponentTypeID(id TypeID) bool {
	if id == "" || isPrimitiveTypeID(id) {
		return false
	}
	if _, array := arrayComponentTypeID(id); array {
		return validArrayTypeID(id)
	}
	return true
}

type registeredJavaType struct {
	super      TypeID
	interfaces []TypeID
}

var javaTypeRegistry = struct {
	sync.RWMutex
	types map[TypeID]registeredJavaType
}{
	types: map[TypeID]registeredJavaType{
		ObjectTypeID:       {},
		CloneableTypeID:    {super: ObjectTypeID},
		SerializableTypeID: {super: ObjectTypeID},
		ComparableTypeID:   {super: ObjectTypeID},
		CharSequenceTypeID: {super: ObjectTypeID},
		ConstableTypeID:    {super: ObjectTypeID},
		ConstantDescTypeID: {super: ObjectTypeID},
		StringTypeID: {
			super: ObjectTypeID,
			interfaces: []TypeID{
				SerializableTypeID,
				ComparableTypeID,
				CharSequenceTypeID,
				ConstableTypeID,
				ConstantDescTypeID,
			},
		},
	},
}

// RegisterJavaType records a generated class or interface hierarchy edge.
// Registration order is irrelevant: assignability walks the registry only
// when a runtime check is actually requested.
func RegisterJavaType(id, super TypeID, interfaces ...TypeID) {
	if id == "" {
		panic(NewIllegalArgumentException("empty Java type id"))
	}
	javaTypeRegistry.Lock()
	javaTypeRegistry.types[id] = registeredJavaType{
		super:      super,
		interfaces: append([]TypeID(nil), interfaces...),
	}
	javaTypeRegistry.Unlock()
}

// JavaTypeAssignable reports Java reference assignment compatibility. Array
// covariance applies recursively only to reference components; primitive array
// components must match exactly. Every Java array is assignable to Object,
// Cloneable, and Serializable.
func JavaTypeAssignable(actual, expected TypeID) bool {
	if actual == "" || expected == "" {
		return false
	}
	if actual == expected {
		return true
	}

	actualComponent, actualArray := arrayComponentTypeID(actual)
	expectedComponent, expectedArray := arrayComponentTypeID(expected)
	if actualArray {
		if expected == ObjectTypeID || expected == CloneableTypeID || expected == SerializableTypeID {
			return true
		}
		if !expectedArray {
			return false
		}
		if isPrimitiveTypeID(actualComponent) || isPrimitiveTypeID(expectedComponent) {
			return actualComponent == expectedComponent
		}
		return JavaTypeAssignable(actualComponent, expectedComponent)
	}
	if expectedArray || isPrimitiveTypeID(actual) || isPrimitiveTypeID(expected) {
		return false
	}

	javaTypeRegistry.RLock()
	defer javaTypeRegistry.RUnlock()
	seen := map[TypeID]struct{}{}
	queue := []TypeID{actual}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == expected {
			return true
		}
		if _, visited := seen[current]; visited {
			continue
		}
		seen[current] = struct{}{}
		info, ok := javaTypeRegistry.types[current]
		if !ok {
			continue
		}
		if info.super != "" {
			queue = append(queue, info.super)
		}
		queue = append(queue, info.interfaces...)
	}
	return false
}

// ObjectInfo is shared by every generated view of one Java object. Its dynamic
// type and view provider are immutable after construction so publishing a fully
// constructed generated object cannot race with later array reads.
//
// The dynamic type
// remains the most-derived class even when a value is held through an embedded
// superclass pointer. View resolves that same object to a requested generated
// class/interface view after an array read or cast.
type ObjectInfo struct {
	dynamicType TypeID
	view        func(TypeID) any
}

func NewObjectInfo(dynamicType TypeID, view func(TypeID) any) *ObjectInfo {
	if dynamicType == "" || isPrimitiveTypeID(dynamicType) {
		panic(NewIllegalArgumentException("invalid Java object type id"))
	}
	return &ObjectInfo{dynamicType: dynamicType, view: view}
}

func (info *ObjectInfo) DynamicType() TypeID {
	if info == nil {
		return ""
	}
	return info.dynamicType
}

func (info *ObjectInfo) resolveView(requested TypeID) any {
	if info == nil || info.view == nil {
		return nil
	}
	return info.view(requested)
}

// JavaObjectInfoCarrier is implemented by generated hierarchy roots. Its
// method is promoted through embedded superclass fields.
type JavaObjectInfoCarrier interface {
	JavaObjectInfo() *ObjectInfo
}

// JavaDynamicTypeCarrier lets runtime-backed Java values participate in the
// same nominal checks as generated objects without exposing their Go shape.
// Arrays implement the narrower javaArrayTypeCarrier below; collections,
// boxed values, and other stdjava types can adopt this interface incrementally.
type JavaDynamicTypeCarrier interface {
	JavaDynamicTypeID() TypeID
}

type javaArrayTypeCarrier interface {
	JavaArrayTypeID() TypeID
}

// ObjectDynamicType returns the reified Java type of value. The boolean is
// false for Java null and for backend values that do not carry Java identity.
func ObjectDynamicType(value any) (TypeID, bool) {
	if nilJavaReference(value) {
		return "", false
	}
	if array, ok := value.(javaArrayTypeCarrier); ok {
		if arrayType := array.JavaArrayTypeID(); arrayType != "" {
			return arrayType, true
		}
	}
	if stringValue, ok := value.(string); ok {
		if StringIsNull(stringValue) {
			return "", false
		}
		return StringTypeID, true
	}
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil && info.DynamicType() != "" {
			return info.DynamicType(), true
		}
	}
	if carrier, ok := value.(JavaDynamicTypeCarrier); ok {
		if dynamicType := carrier.JavaDynamicTypeID(); dynamicType != "" {
			return dynamicType, true
		}
	}
	return "", false
}

// ObjectView resolves one stored Java object to the generated view requested
// by a statically typed read. Java null and nominal assignability are checked
// before Go assignability; superclass/interface views then use the object's
// shared provider when the stored most-derived pointer is not the requested Go
// view type.
func ObjectView[T any](value any, requested TypeID) T {
	var zero T
	if nilJavaReference(value) {
		// Generated strings retain a concrete Go string ABI, so a statically
		// String-typed null read needs the sentinel. The same null read through
		// Object/any must instead be a genuinely nil interface.
		if requested == StringTypeID {
			if nullString, ok := any(NullString()).(T); ok {
				return nullString
			}
		}
		return zero
	}
	actual, ok := ObjectDynamicType(value)
	if !ok || !JavaTypeAssignable(actual, requested) {
		panic(NewClassCastException(fmt.Sprintf("Java value %s is not assignable to %s", actual, requested)))
	}
	if direct, ok := value.(T); ok {
		return direct
	}
	carrier, ok := value.(JavaObjectInfoCarrier)
	if !ok {
		panic(NewClassCastException(fmt.Sprintf("value %T has no Java object view", value)))
	}
	info := carrier.JavaObjectInfo()
	if info == nil || info.view == nil {
		panic(NewClassCastException(fmt.Sprintf("value %T has no view for %s", value, requested)))
	}
	view := info.resolveView(requested)
	resolved, ok := view.(T)
	if !ok {
		panic(NewClassCastException(fmt.Sprintf("Java object %s has no requested view %s", info.DynamicType(), requested)))
	}
	return resolved
}

// PrimitiveArray preserves the runtime component descriptor and object
// identity of a primitive Java array while keeping its elements in a native Go
// slice. Hot generated loops can hoist Elements once and retain ordinary slice
// indexing; erased/nested uses keep the wrapper and therefore their Java type.
type PrimitiveArray[T any] struct {
	componentType TypeID
	Elements      []T
}

func NewPrimitiveArray[T any, I javaArrayLength](length I, componentType TypeID) *PrimitiveArray[T] {
	if length < 0 {
		panic(NewNegativeArraySizeException(""))
	}
	if !validPrimitiveTypeID(componentType) {
		panic(NewIllegalArgumentException("invalid primitive array component type"))
	}
	return &PrimitiveArray[T]{componentType: componentType, Elements: make([]T, int(length))}
}

func PrimitiveArrayLiteral[T any](componentType TypeID, elements ...T) *PrimitiveArray[T] {
	if !validPrimitiveTypeID(componentType) {
		panic(NewIllegalArgumentException("invalid primitive array component type"))
	}
	return &PrimitiveArray[T]{componentType: componentType, Elements: elements}
}

func (array *PrimitiveArray[T]) JavaArrayTypeID() TypeID {
	if array == nil {
		return ""
	}
	return ArrayTypeID(array.componentType)
}

func (array *PrimitiveArray[T]) ComponentType() TypeID {
	if array == nil {
		panic(NewNullPointerException("array component type on null"))
	}
	return array.componentType
}

func PrimitiveArrayLength[T any](array *PrimitiveArray[T]) int32 {
	if array == nil {
		panic(NewNullPointerException("array length on null"))
	}
	return int32(len(array.Elements))
}

func primitiveArrayIndex[T any, I javaArrayLength](array *PrimitiveArray[T], index I) int {
	if array == nil {
		panic(NewNullPointerException("array access on null"))
	}
	index64 := int64(index)
	if index64 < 0 || index64 >= int64(len(array.Elements)) {
		panic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
	}
	return int(index64)
}

func PrimitiveArrayGet[T any, I javaArrayLength](array *PrimitiveArray[T], index I) T {
	return array.Elements[primitiveArrayIndex(array, index)]
}

// PrimitiveArrayAssign checks the target before invoking rhs. Java simple
// array assignment does not evaluate its right-hand side when the array is null
// or the index is out of bounds.
func PrimitiveArrayAssign[T any, I javaArrayLength](array *PrimitiveArray[T], index I, rhs func() T) T {
	position := primitiveArrayIndex(array, index)
	value := rhs()
	array.Elements[position] = value
	return value
}

// ReferenceArray is the common ABI for every Java array whose immediate
// component is a reference type. Primitive leaf arrays use PrimitiveArray;
// outer dimensions are reference arrays because an array value is itself a
// Java reference.
type ReferenceArray struct {
	componentType TypeID
	elements      []any
}

func NewReferenceArray[I javaArrayLength](length I, componentType TypeID) *ReferenceArray {
	if length < 0 {
		panic(NewNegativeArraySizeException(""))
	}
	if !validReferenceComponentTypeID(componentType) {
		panic(NewIllegalArgumentException("invalid reference array component type"))
	}
	array := &ReferenceArray{
		componentType: componentType,
		elements:      make([]any, int(length)),
	}
	if componentType == StringTypeID {
		for index := range array.elements {
			array.elements[index] = NullString()
		}
	}
	return array
}

func (array *ReferenceArray) JavaArrayTypeID() TypeID {
	if array == nil {
		return ""
	}
	return ArrayTypeID(array.componentType)
}

func ReferenceArrayLiteral(componentType TypeID, elements ...any) *ReferenceArray {
	array := NewReferenceArray(len(elements), componentType)
	for index, element := range elements {
		ReferenceArraySet(array, index, element)
	}
	return array
}

func (array *ReferenceArray) ComponentType() TypeID {
	if array == nil {
		panic(NewNullPointerException("array component type on null"))
	}
	return array.componentType
}

func ReferenceArrayLength(array *ReferenceArray) int32 {
	if array == nil {
		panic(NewNullPointerException("array length on null"))
	}
	return int32(len(array.elements))
}

func referenceArrayIndex[I javaArrayLength](array *ReferenceArray, index I) int {
	if array == nil {
		panic(NewNullPointerException("array access on null"))
	}
	index64 := int64(index)
	if index64 < 0 || index64 >= int64(len(array.elements)) {
		panic(NewArrayIndexOutOfBoundsException("array index out of bounds"))
	}
	return int(index64)
}

func ReferenceArrayGet[T any, I javaArrayLength](array *ReferenceArray, index I, requestedType TypeID) T {
	position := referenceArrayIndex(array, index)
	return ObjectView[T](array.elements[position], requestedType)
}

func referenceArrayStoreAt(array *ReferenceArray, position int, value any) {
	isNull := nilJavaReference(value)
	if !isNull {
		actualType, ok := ObjectDynamicType(value)
		if !ok || !JavaTypeAssignable(actualType, array.componentType) {
			panic(NewArrayStoreException(fmt.Sprintf("cannot store %s in %s[]", actualType, array.componentType)))
		}
	}
	stored := any(value)
	if isNull {
		// Canonicalize typed nil pointers/interfaces before erasing them into the
		// []any storage. Otherwise an Object[] read would receive a non-nil Go
		// interface containing a nil pointer and Java `value == null` could fail.
		stored = nil
		if array.componentType == StringTypeID {
			stored = NullString()
		}
	}
	array.elements[position] = stored
}

// ReferenceArraySet stores a value that has already been evaluated. It is used
// by runtime builders; generated Java simple assignments use
// ReferenceArrayAssign so null/bounds checks happen before RHS evaluation.
func ReferenceArraySet[T any, I javaArrayLength](array *ReferenceArray, index I, value T) T {
	position := referenceArrayIndex(array, index)
	referenceArrayStoreAt(array, position, value)
	return value
}

// ReferenceArrayAssign implements Java simple array assignment order and
// result typing. The array and index are checked before rhs runs. The RHS is
// then converted to the expression's static component view before the runtime
// component check/store; a rejected covariant store never mutates the array.
// Generated code supplies Result explicitly when its Go view differs from the
// RHS view (for example Child assigned through a Base[] expression).
func ReferenceArrayAssign[Result any, Input any, I javaArrayLength](
	array *ReferenceArray,
	index I,
	rhs func() Input,
	staticComponentType TypeID,
) Result {
	position := referenceArrayIndex(array, index)
	value := rhs()
	result := ObjectView[Result](value, staticComponentType)
	referenceArrayStoreAt(array, position, value)
	return result
}

func nilJavaReference(value any) bool {
	if value == nil {
		return true
	}
	if stringValue, ok := value.(string); ok && StringIsNull(stringValue) {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
