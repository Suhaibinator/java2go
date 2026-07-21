package stdjava

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// nullStringSentinel preserves Java's null String reference while keeping the
// generated String ABI as Go string. The invalid UTF-8 byte cannot be produced
// by Java's UTF-16 String model, so it remains distinct from every Java String,
// including the empty string. Generated constructors install this value in
// String fields before any Java field initializer or superclass constructor can
// observe them.
const nullStringSentinel = "\xffjava2go:null-string\x00"

// NullString returns the generated representation of a null Java String
// reference. It is intentionally exposed as a function rather than a constant
// so generated code does not depend on the sentinel's spelling.
func NullString() string { return nullStringSentinel }

// StringIsNull recognizes both generated String representations that can carry
// Java null: the sentinel used at concrete-string ABI boundaries and nil in an
// interface-backed nullable local.
func StringIsNull(value any) bool {
	if value == nil {
		return true
	}
	stringValue, ok := value.(string)
	return ok && stringValue == nullStringSentinel
}

// StringReferenceValue converts an interface-backed nullable String to the
// concrete representation used by generated fields, parameters, and method
// results. Unlike StringRequireNonNull, this is a representation boundary, not
// a dereference, so Java null must pass through rather than throw.
func StringReferenceValue(value any) string {
	if StringIsNull(value) {
		return nullStringSentinel
	}
	if stringValue, ok := value.(string); ok {
		return stringValue
	}
	panic(NewClassCastException(fmt.Sprintf("cannot use %T as String", value)))
}

// StringValueOf implements the text conversion shared by Java's
// String.valueOf(Object) and the primitive print/concatenation operations.
// In particular, Java keeps a decimal point for whole floating-point values,
// spells infinities without Go's +/-Inf notation, and renders null as "null".
// fmt.Stringer is intentionally honored so generated enums can expose Java's
// Enum.toString default without leaking their Go struct representation.
func StringValueOf(value any) string {
	if StringIsNull(value) {
		return "null"
	}

	switch value := value.(type) {
	case float32:
		return FloatToString(value)
	case float64:
		return DoubleToString(value)
	default:
		return fmt.Sprint(value)
	}
}

type executionStringer interface {
	StringJava2goExecution(*Execution) string
}

// StringValueOfExecution is StringValueOf for generated Java code that is
// already running inside a logical execution. Generated enum String methods
// can acquire Java monitors, so forwarding the execution token is required when
// an enum has been erased to Object (or another interface type) before text
// conversion. Collision-renamed hidden methods are discovered structurally.
func StringValueOfExecution(execution *Execution, value any) string {
	if StringIsNull(value) {
		return "null"
	}

	switch value := value.(type) {
	case float32:
		return FloatToString(value)
	case float64:
		return DoubleToString(value)
	case executionStringer:
		return value.StringJava2goExecution(execution)
	}

	if rendered, ok := callCollisionSafeExecutionStringer(execution, value); ok {
		return rendered
	}
	return fmt.Sprint(value)
}

func callCollisionSafeExecutionStringer(execution *Execution, value any) (string, bool) {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return "", false
	}
	executionType := reflect.TypeOf((*Execution)(nil))
	stringType := reflect.TypeOf("")
	typeOfValue := reflected.Type()
	for index := 0; index < typeOfValue.NumMethod(); index++ {
		method := typeOfValue.Method(index)
		if !isCollisionSafeExecutionMethodName(method.Name, "StringJava2goExecution") ||
			method.Type.NumIn() != 2 || method.Type.In(1) != executionType ||
			method.Type.NumOut() != 1 || method.Type.Out(0) != stringType {
			continue
		}
		results := reflected.Method(index).Call([]reflect.Value{reflect.ValueOf(execution)})
		return results[0].String(), true
	}
	return "", false
}

func isCollisionSafeExecutionMethodName(name, base string) bool {
	if name == base {
		return true
	}
	suffix := strings.TrimPrefix(name, base)
	if suffix == name || suffix == "" {
		return false
	}
	for _, digit := range suffix {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// StringRequireNonNull converts the two generated representations of a Java
// String reference (a Go string value or a nullable interface slot) into the
// concrete string required by String intrinsics. Calling an instance method on
// a null Java reference throws NullPointerException.
func StringRequireNonNull(value any) string {
	if StringIsNull(value) {
		panic(NewNullPointerException("String method called on null"))
	}
	if stringValue, ok := value.(string); ok {
		return stringValue
	}
	panic(NewClassCastException(fmt.Sprintf("cannot use %T as String", value)))
}

// FloatToString formats a float32 according to Java's Float.toString rules.
func FloatToString(value float32) string {
	return javaFloatingPointString(float64(value), 32)
}

// DoubleToString formats a float64 according to Java's Double.toString rules.
func DoubleToString(value float64) string {
	return javaFloatingPointString(value, 64)
}

func javaFloatingPointString(value float64, bitSize int) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	}

	abs := math.Abs(value)
	if abs == 0 || (abs >= 1e-3 && abs < 1e7) {
		formatted := strconv.FormatFloat(value, 'f', -1, bitSize)
		if !strings.ContainsRune(formatted, '.') {
			formatted += ".0"
		}
		return formatted
	}

	formatted := strconv.FormatFloat(value, 'e', -1, bitSize)
	mantissa, exponentText, found := strings.Cut(formatted, "e")
	if !found {
		return formatted
	}
	if !strings.ContainsRune(mantissa, '.') {
		// Java's scientific form requires a fractional digit. Appending zero to
		// Go's one-digit shortest form can choose a decimal farther from the
		// binary value (Float.MIN_VALUE would become 1.0E-45). Re-round with one
		// fractional digit so Java's closest qualifying form is retained.
		formatted = strconv.FormatFloat(value, 'e', 1, bitSize)
		mantissa, exponentText, _ = strings.Cut(formatted, "e")
	}
	exponent, err := strconv.Atoi(exponentText)
	if err != nil {
		return mantissa + "E" + exponentText
	}
	return mantissa + "E" + strconv.Itoa(exponent)
}
