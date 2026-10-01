package stdjava

import (
	"reflect"
	"sync"
	"unicode/utf16"
)

// ObjectInstanceOf checks Java's nominal type relation without dereferencing
// null. In particular, a typed nil wrapper is never an instanceof match.
func ObjectInstanceOf(value any, expected TypeID) bool {
	if nilJavaReference(value) {
		return false
	}
	if expected == ObjectTypeID && !unboxedPrimitiveValue(value) {
		return true
	}
	actual, ok := ObjectDynamicType(value)
	return (ok && JavaTypeAssignable(actual, expected)) || nativeJavaInterfaceAssignable(value, expected)
}

// ObjectPattern supplies the source-declared view after a successful pattern
// test. ObjectView preserves both wrapper identity and generated base views.
func ObjectPattern[T any](value any, expected TypeID) (T, bool) {
	if !ObjectInstanceOf(value, expected) {
		var zero T
		return zero, false
	}
	return ObjectView[T](value, expected), true
}

// ObjectEqualsExecution implements a virtual equals invocation through Object
// or Number. Its arguments are evaluated before the receiver's null check.
func ObjectEqualsExecution(execution *Execution, left, right any) bool {
	ReferenceRequireNonNull(left)
	left, right = collectionObjectView(left), collectionObjectView(right)
	if javaReferenceIsNull(right) {
		right = nil
	}
	// The exact companion precedes every collision-renamed companion in the
	// method set. Keep reflection for renamed methods and other signatures.
	if equals, ok := left.(interface {
		EqualsJava2goExecution(*Execution, any) bool
	}); ok {
		return equals.EqualsJava2goExecution(execution, right)
	}
	if result, ok := objectExecutionMethod(execution, left, "EqualsJava2goExecution", []any{right}); ok {
		return result.Bool()
	}
	if equals, ok := left.(interface{ Equals(any) bool }); ok {
		return equals.Equals(right)
	}
	if value, ok := left.(string); ok {
		return StringEquals(value, right)
	}
	return JavaReferenceEqual(left, right)
}

// StringEquals implements String.equals(Object), whose argument may have any
// reference type and whose receiver must be non-null.
func StringEquals(left string, right any) bool {
	StringRequireNonNull(left)
	text, ok := right.(string)
	return ok && !StringIsNull(text) && left == text
}

// ObjectHashCodeExecution invokes the dynamic Java hashCode implementation.
// The default object's hash is identity-based; its numeric value is unspecified
// by Java. Boxed values expose the exact per-wrapper hash instead.
func ObjectHashCodeExecution(execution *Execution, value any) int32 {
	ReferenceRequireNonNull(value)
	value = collectionObjectView(value)
	if hash, ok := value.(interface {
		HashCodeJava2goExecution(*Execution) int32
	}); ok {
		return hash.HashCodeJava2goExecution(execution)
	}
	if result, ok := objectExecutionMethod(execution, value, "HashCodeJava2goExecution", nil); ok {
		return int32(result.Int())
	}
	if hash, ok := value.(interface{ HashCode() int32 }); ok {
		return hash.HashCode()
	}
	if text, ok := value.(string); ok {
		var hash int32
		for _, unit := range utf16.Encode([]rune(text)) {
			hash = hash*31 + int32(unit)
		}
		return hash
	}
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			pointer := uint64(reflect.ValueOf(info).Pointer())
			return int32(pointer ^ (pointer >> 32))
		}
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan, reflect.Slice, reflect.Func:
		pointer := uint64(reflected.Pointer())
		return int32(pointer ^ (pointer >> 32))
	default:
		panic(NewClassCastException("value has no Java object identity"))
	}
}

type objectExecutionMethodKey struct {
	receiver reflect.Type
	name     string
}

// Go method sets and signatures are immutable. Cache discovery, including a
// missing companion, without retaining a bound receiver, execution, arguments,
// return value, or Java type registration. LoadOrStore publishes a complete
// immutable candidate list for concurrent callers.
var objectExecutionMethodMetadata sync.Map

func objectExecutionMethodCandidates(receiver reflect.Type, name string) []reflect.Method {
	key := objectExecutionMethodKey{receiver: receiver, name: name}
	if cached, ok := objectExecutionMethodMetadata.Load(key); ok {
		return cached.([]reflect.Method)
	}
	var candidates []reflect.Method
	executionType := reflect.TypeOf((*Execution)(nil))
	for index := 0; index < receiver.NumMethod(); index++ {
		methodInfo := receiver.Method(index)
		if !isCollisionSafeExecutionMethodName(methodInfo.Name, name) {
			continue
		}
		// Method metadata includes the receiver before the logical execution.
		signature := methodInfo.Type
		if signature.NumIn() < 2 || signature.In(1) != executionType || signature.NumOut() != 1 {
			continue
		}
		if (name == "EqualsJava2goExecution" && signature.Out(0).Kind() != reflect.Bool) ||
			(name == "HashCodeJava2goExecution" && signature.Out(0).Kind() != reflect.Int32) {
			continue
		}
		// equals(SomeClass) is an overload, not Object.equals(Object).
		// Only the erased Object parameter participates in virtual equality.
		if name == "EqualsJava2goExecution" && (signature.NumIn() < 3 || signature.In(2) != reflect.TypeOf((*any)(nil)).Elem()) {
			continue
		}
		candidates = append(candidates, methodInfo)
	}
	actual, _ := objectExecutionMethodMetadata.LoadOrStore(key, candidates)
	return actual.([]reflect.Method)
}

// Generated execution companions may be renamed to avoid a source member
// collision. Discover their immutable signatures once, then bind the current
// receiver, arguments, and logical execution for every invocation.
func objectExecutionMethod(execution *Execution, receiver any, name string, args []any) (reflect.Value, bool) {
	value := reflect.ValueOf(receiver)
	for _, candidate := range objectExecutionMethodCandidates(value.Type(), name) {
		signature := candidate.Type
		if signature.NumIn() != len(args)+2 {
			continue
		}
		parameters := []reflect.Value{reflect.ValueOf(execution)}
		compatible := true
		for argumentIndex, arg := range args {
			expected := signature.In(argumentIndex + 2)
			if nilJavaReference(arg) {
				parameters = append(parameters, reflect.Zero(expected))
			} else if argument := reflect.ValueOf(arg); argument.Type().AssignableTo(expected) {
				parameters = append(parameters, argument)
			} else {
				compatible = false
				break
			}
		}
		if compatible {
			return value.Method(candidate.Index).Call(parameters)[0], true
		}
	}
	return reflect.Value{}, false
}

// Every generated superclass view shares ObjectInfo with its most-derived
// object. Resolve that receiver before invoking virtual Object methods so an
// erased/base-typed key observes the override and the same default identity.
func collectionObjectView(value any) any {
	if !javaReferenceIsNull(value) {
		if carrier, ok := value.(JavaObjectInfoCarrier); ok {
			if info := carrier.JavaObjectInfo(); info != nil {
				if dynamic := info.resolveView(info.DynamicType()); !javaReferenceIsNull(dynamic) {
					return dynamic
				}
			}
		}
	}
	return value
}
