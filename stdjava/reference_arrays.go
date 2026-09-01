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
	BooleanTypeID      TypeID = "java.lang.Boolean"
	ByteTypeID         TypeID = "java.lang.Byte"
	ShortTypeID        TypeID = "java.lang.Short"
	CharacterTypeID    TypeID = "java.lang.Character"
	IntegerTypeID      TypeID = "java.lang.Integer"
	LongTypeID         TypeID = "java.lang.Long"
	FloatTypeID        TypeID = "java.lang.Float"
	DoubleTypeID       TypeID = "java.lang.Double"
	// ThrowableTypeID follows the descriptor spelling currently emitted for the
	// built-in exception hierarchy. Generated source classes use qualified binary
	// names; java.lang exception intrinsics intentionally retain their Java simple
	// names so they also match ThrowableTypeName and catch dispatch.
	ThrowableTypeID TypeID = "Throwable"
)

const (
	primitiveTypePrefix = "primitive:"
	arrayTypePrefix     = "array:"
)

const (
	PrimitiveBooleanTypeID TypeID = "primitive:boolean"
	PrimitiveByteTypeID    TypeID = "primitive:byte"
	PrimitiveShortTypeID   TypeID = "primitive:short"
	PrimitiveCharTypeID    TypeID = "primitive:char"
	PrimitiveIntTypeID     TypeID = "primitive:int"
	PrimitiveLongTypeID    TypeID = "primitive:long"
	PrimitiveFloatTypeID   TypeID = "primitive:float"
	PrimitiveDoubleTypeID  TypeID = "primitive:double"
)

// PrimitiveTypeID returns the descriptor used for one Java primitive type.
func PrimitiveTypeID(name string) TypeID {
	normalized := strings.TrimSpace(name)
	switch normalized {
	case "boolean":
		return PrimitiveBooleanTypeID
	case "byte":
		return PrimitiveByteTypeID
	case "short":
		return PrimitiveShortTypeID
	case "char":
		return PrimitiveCharTypeID
	case "int":
		return PrimitiveIntTypeID
	case "long":
		return PrimitiveLongTypeID
	case "float":
		return PrimitiveFloatTypeID
	case "double":
		return PrimitiveDoubleTypeID
	default:
		// Keep the historical descriptor spelling for validation paths. Array
		// constructors reject unknown primitive descriptors deterministically;
		// callers that only inspect the descriptor still receive primitive:<name>.
		return TypeID(primitiveTypePrefix + normalized)
	}
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
		BooleanTypeID:   {super: ObjectTypeID},
		ByteTypeID:      {super: ObjectTypeID},
		ShortTypeID:     {super: ObjectTypeID},
		CharacterTypeID: {super: ObjectTypeID},
		IntegerTypeID:   {super: ObjectTypeID},
		LongTypeID:      {super: ObjectTypeID},
		FloatTypeID:     {super: ObjectTypeID},
		DoubleTypeID:    {super: ObjectTypeID},
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

// registeredJavaViewCandidates returns the nominal views of actual from most
// specific to least specific. ObjectView uses this only after the requested
// Java descriptor has already passed assignability, when type erasure means the
// requested descriptor (often Object) is less precise than its instantiated Go
// result type.
func registeredJavaViewCandidates(actual TypeID) []TypeID {
	javaTypeRegistry.RLock()
	defer javaTypeRegistry.RUnlock()

	seen := make(map[TypeID]struct{})
	queue := []TypeID{actual}
	result := make([]TypeID, 0, 4)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == "" {
			continue
		}
		if _, duplicate := seen[current]; duplicate {
			continue
		}
		seen[current] = struct{}{}
		result = append(result, current)

		registered, ok := javaTypeRegistry.types[current]
		if !ok {
			continue
		}
		if registered.super != "" {
			queue = append(queue, registered.super)
		}
		queue = append(queue, registered.interfaces...)
	}
	return result
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

// JavaObjectInfo lets generated hierarchy roots embed *ObjectInfo directly.
// The method is then promoted through every embedded superclass view, so all
// views of one generated Java object expose the same immutable identity.
func (info *ObjectInfo) JavaObjectInfo() *ObjectInfo {
	return info
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

type generatedObjectMetadata interface {
	Java2goReferenceDynamicType() TypeID
	Java2goReferenceView(TypeID) any
}

// NewGeneratedObjectInfo captures the most-derived generated receiver before
// superclass construction. The generated view method resolves that receiver
// to any statically requested class/interface view after an array read.
func NewGeneratedObjectInfo(value any) *ObjectInfo {
	metadata, ok := value.(generatedObjectMetadata)
	if !ok {
		panic(NewIllegalArgumentException(fmt.Sprintf("generated object %T has no Java reference metadata", value)))
	}
	return NewObjectInfo(metadata.Java2goReferenceDynamicType(), metadata.Java2goReferenceView)
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
	// Generated boxed values currently use their fixed-width Go scalar ABI.
	// Recording those widths here lets Object[] enforce its Java component at
	// runtime instead of treating every boxed primitive as an untyped Go value.
	switch value.(type) {
	case bool:
		return BooleanTypeID, true
	case int8:
		return ByteTypeID, true
	case int16:
		return ShortTypeID, true
	case int32:
		// rune and int32 are aliases in Go. Integer is the conservative erased
		// identity; Character-specific casts require a future boxed-value ABI.
		return IntegerTypeID, true
	case int64:
		return LongTypeID, true
	case float32:
		return FloatTypeID, true
	case float64:
		return DoubleTypeID, true
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
	// Some unresolved anonymous Runnable implementations retain only their Go
	// structural interface shape. Check this after the explicit generated/runtime
	// carriers so a named class implementing Runnable keeps its more-specific
	// dynamic descriptor instead of collapsing to the interface descriptor.
	if _, ok := value.(Runnable); ok {
		return RunnableTypeID, true
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
		// String-typed null read needs the sentinel. The generic T[] path carries
		// ObjectTypeID after erasure, so also inspect T itself. Do not use a plain
		// type assertion here: string is assignable to T=any and would incorrectly
		// turn an Object null into a non-nil interface.
		targetType := reflect.TypeOf((*T)(nil)).Elem()
		if requested == StringTypeID || targetType.Kind() == reflect.String {
			if nullString, ok := any(NullString()).(T); ok {
				return nullString
			}
		}
		return zero
	}
	actual, ok := ObjectDynamicType(value)
	if !ok && requested == ObjectTypeID {
		// Some runtime-backed Java objects intentionally retain an opaque Go ABI
		// (notably new Object() monitor tokens and generic collection pointers).
		// Object is the one Java component for which every non-null reference is
		// assignable. A direct Go view proves this erased read is representable;
		// narrower class/interface requests still require nominal metadata below.
		if direct, directOK := value.(T); directOK {
			return direct
		}
	}
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
	if ok {
		return resolved
	}

	// A generic read may have an erased requested descriptor while T retains a
	// concrete instantiation in Go: ObjectView[*Base](child, ObjectTypeID), for
	// example. The nominal check above proves the Java read is legal; search the
	// same object's registered hierarchy for the concrete generated view that is
	// assignable to T. Never use this search to bypass the requested descriptor's
	// Java assignability check.
	for _, candidate := range registeredJavaViewCandidates(actual) {
		if candidate == requested {
			continue
		}
		if candidateView, found := info.resolveView(candidate).(T); found {
			return candidateView
		}
	}
	panic(NewClassCastException(fmt.Sprintf("Java object %s has no requested view %s", info.DynamicType(), requested)))
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

func newPrimitiveArrayForType(length int32, componentType TypeID) any {
	switch componentType {
	case PrimitiveTypeID("boolean"):
		return NewPrimitiveArray[bool](length, componentType)
	case PrimitiveTypeID("byte"):
		return NewPrimitiveArray[int8](length, componentType)
	case PrimitiveTypeID("short"):
		return NewPrimitiveArray[int16](length, componentType)
	case PrimitiveTypeID("char"):
		return NewPrimitiveArray[rune](length, componentType)
	case PrimitiveTypeID("int"):
		return NewPrimitiveArray[int32](length, componentType)
	case PrimitiveTypeID("long"):
		return NewPrimitiveArray[int64](length, componentType)
	case PrimitiveTypeID("float"):
		return NewPrimitiveArray[float32](length, componentType)
	case PrimitiveTypeID("double"):
		return NewPrimitiveArray[float64](length, componentType)
	default:
		panic(NewIllegalArgumentException("invalid primitive array component type"))
	}
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

// PrimitiveArrayElements exposes the native backing slice to proven generated
// hot loops. A nil array maps to a nil slice so LICM/versioning can preserve the
// historical lazy-failure contract; ordinary Java accesses keep using the
// checked wrapper helpers or direct Elements lowering.
func PrimitiveArrayElements[T any](array *PrimitiveArray[T]) []T {
	if array == nil {
		return nil
	}
	return array.Elements
}

// PrimitiveArrayIterationElements exposes the backing slice for a Java
// enhanced-for statement. Unlike PrimitiveArrayElements, which is also used by
// nil-preserving ABI adapters and speculative loop views, evaluating the
// iterable of an enhanced-for statement must fail immediately when it is null.
func PrimitiveArrayIterationElements[T any](array *PrimitiveArray[T]) []T {
	if array == nil {
		panic(NewNullPointerException("enhanced-for over null array"))
	}
	return array.Elements
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

// PrimitiveArrayAssign implements Java simple-array assignment order. Go
// evaluates the array, index, and value arguments from left to right before
// entering this helper; only then does the helper perform the saved array's
// null/bounds checks and store. Passing the already-evaluated value also avoids
// allocating or invoking an RHS closure for every generated array write.
func PrimitiveArrayAssign[T any, I javaArrayLength](array *PrimitiveArray[T], index I, value T) T {
	position := primitiveArrayIndex(array, index)
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

// NewReferenceArrayOf is the generated-code form of NewReferenceArray. T keeps
// the source component's Go type in the generated AST (and therefore retains a
// required cross-package import) while ReferenceArray remains the common ABI.
func NewReferenceArrayOf[T any, I javaArrayLength](length I, componentType TypeID) *ReferenceArray {
	return NewReferenceArray(length, componentType)
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

// ReferenceArrayLiteralOf is the typed generated-code form of
// ReferenceArrayLiteral; see NewReferenceArrayOf.
func ReferenceArrayLiteralOf[T any](componentType TypeID, elements ...any) *ReferenceArray {
	return ReferenceArrayLiteral(componentType, elements...)
}

// NewMultiArrayOf builds a Java multidimensional array while retaining the
// descriptor of every level. All explicit dimensions are received before the
// first check/allocation, preserving Java's left-to-right evaluation rule; all
// negative dimensions are then rejected before any partial array can escape.
// Primitive leaves use PrimitiveArray while every outer level uses
// ReferenceArray because its immediate component is itself an array reference.
func NewMultiArrayOf[T any](baseComponent TypeID, totalRank int, dimensions ...int32) *ReferenceArray {
	if totalRank < 2 || len(dimensions) == 0 || len(dimensions) > totalRank {
		panic(NewIllegalArgumentException("invalid multidimensional array rank"))
	}
	if isPrimitiveTypeID(baseComponent) {
		if !validPrimitiveTypeID(baseComponent) {
			panic(NewIllegalArgumentException("invalid primitive array component type"))
		}
	} else if !validReferenceComponentTypeID(baseComponent) {
		panic(NewIllegalArgumentException("invalid reference array component type"))
	}
	for _, dimension := range dimensions {
		if dimension < 0 {
			panic(NewNegativeArraySizeException(""))
		}
	}

	descriptorAtRank := func(rank int) TypeID {
		descriptor := baseComponent
		for level := 0; level < rank; level++ {
			descriptor = ArrayTypeID(descriptor)
		}
		return descriptor
	}
	var allocate func(rank, depth int) any
	allocate = func(rank, depth int) any {
		length := dimensions[depth]
		if rank == 1 {
			if isPrimitiveTypeID(baseComponent) {
				return newPrimitiveArrayForType(length, baseComponent)
			}
			return NewReferenceArray(length, baseComponent)
		}
		array := NewReferenceArray(length, descriptorAtRank(rank-1))
		if depth+1 < len(dimensions) {
			for index := int32(0); index < length; index++ {
				ReferenceArraySet(array, index, allocate(rank-1, depth+1))
			}
		}
		return array
	}
	return allocate(totalRank, 0).(*ReferenceArray)
}

// JavaArrayInstanceOf performs Java descriptor-based array checks. Go type
// assertions cannot distinguish Child[] from Base[] because both intentionally
// share the ReferenceArray ABI.
func JavaArrayInstanceOf(value any, expectedArrayType TypeID) bool {
	if nilJavaReference(value) || !validArrayTypeID(expectedArrayType) {
		return false
	}
	actual, ok := ObjectDynamicType(value)
	return ok && JavaTypeAssignable(actual, expectedArrayType)
}

// JavaArrayPattern combines an instanceof descriptor check with the generated
// static view needed by a pattern variable, evaluating the subject once.
func JavaArrayPattern[T any](value any, expectedArrayType TypeID) (T, bool) {
	var zero T
	if !JavaArrayInstanceOf(value, expectedArrayType) {
		return zero, false
	}
	result, ok := value.(T)
	if !ok {
		return zero, false
	}
	return result, true
}

// JavaArrayCast is the array counterpart of a Java reference cast. Null casts
// through unchanged; non-null values must carry an assignable array descriptor.
func JavaArrayCast[T any](value any, expectedArrayType TypeID) T {
	var zero T
	if nilJavaReference(value) {
		return zero
	}
	actual, ok := ObjectDynamicType(value)
	if !ok || !JavaTypeAssignable(actual, expectedArrayType) {
		panic(NewClassCastException(fmt.Sprintf("Java value %s is not assignable to %s", actual, expectedArrayType)))
	}
	result, ok := value.(T)
	if !ok {
		panic(NewClassCastException(fmt.Sprintf("Java array %s has incompatible generated view %T", actual, value)))
	}
	return result
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

// ReferenceArrayIterationElements is the raw, shared element view used by a
// generated enhanced-for range. It intentionally does not convert elements:
// generated code applies ObjectView to the current value inside the loop body,
// so a write to a future slot is observed and break/throw never converts slots
// that Java would not have visited. Returning the backing slice is safe for the
// generated read-only range use; Java stores still go through the checked API.
func ReferenceArrayIterationElements(array *ReferenceArray) []any {
	if array == nil {
		panic(NewNullPointerException("enhanced-for over null array"))
	}
	return array.elements
}

// ReferenceArrayElements converts the erased wrapper backing store to the
// concrete slice required by a generated Go variadic declaration. A Java null
// array remains a nil slice: passing (String[]) null to String... is a legal
// fixed-arity invocation whose parameter value is null. Non-null elements use
// the same nominal checked view conversion as ordinary reference-array reads.
func ReferenceArrayElements[T any](array *ReferenceArray, requestedType TypeID) []T {
	if array == nil {
		return nil
	}
	elements := make([]T, len(array.elements))
	for index, value := range array.elements {
		elements[index] = ObjectView[T](value, requestedType)
	}
	return elements
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
		// java.lang.Object accepts every Java reference, including stdjava-backed
		// values whose compact Go representation does not expose nominal metadata.
		// Keep this exception exact: an opaque value must still be rejected by a
		// covariant Child[] or interface[] target.
		opaqueObjectStore := !ok && array.componentType == ObjectTypeID
		if !opaqueObjectStore && (!ok || !JavaTypeAssignable(actualType, array.componentType)) {
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
// ReferenceArrayAssign so RHS evaluation happens before null/bounds checks.
func ReferenceArraySet[T any, I javaArrayLength](array *ReferenceArray, index I, value T) T {
	position := referenceArrayIndex(array, index)
	referenceArrayStoreAt(array, position, value)
	return value
}

// ReferenceArrayAssign implements Java simple array assignment order and
// result typing. Go evaluates the array, index, and value arguments from left
// to right before entering the helper; the helper then performs Java's null,
// bounds, and runtime component checks. The assignment result is converted to
// the expression's static component view before the runtime component
// check/store; a rejected covariant store never mutates the array.
// Generated code supplies Result explicitly when its Go view differs from the
// RHS view (for example Child assigned through a Base[] expression).
func ReferenceArrayAssign[Result any, Input any, I javaArrayLength](
	array *ReferenceArray,
	index I,
	value Input,
	staticComponentType TypeID,
) Result {
	position := referenceArrayIndex(array, index)
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
