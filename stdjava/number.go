package stdjava

import (
	"math"
	"reflect"
)

// JavaNumber is the nullable reference interface used for java.lang.Number and
// Java type parameters bounded by it. Concrete wrapper pointers retain their
// identity when stored in this interface.
type JavaNumber interface {
	ByteValue() int8
	ShortValue() int16
	IntValue() int32
	LongValue() int64
	FloatValue() float32
	DoubleValue() float64
}

// JavaPrimitiveNumber constrains arithmetic inside primitive streams and
// collectors. Java reference type arguments must use JavaNumber instead.
type JavaPrimitiveNumber interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

func numericValue(value any) reflect.Value {
	if javaReferenceIsNull(value) {
		panic(NewNullPointerException("Number value is null"))
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64:
		return reflected
	default:
		panic(NewClassCastException("value is not a java.lang.Number"))
	}
}

func NumberDoubleValue(value any) float64 {
	if number, ok := value.(JavaNumber); ok {
		return number.DoubleValue()
	}
	reflected := numericValue(value)
	if reflected.Kind() == reflect.Float32 || reflected.Kind() == reflect.Float64 {
		return reflected.Float()
	}
	return float64(reflected.Int())
}

func NumberFloatValue(value any) float32 {
	if number, ok := value.(JavaNumber); ok {
		return number.FloatValue()
	}
	reflected := numericValue(value)
	if reflected.Kind() == reflect.Float32 || reflected.Kind() == reflect.Float64 {
		return float32(reflected.Float())
	}
	// Converting through float64 can round a long twice and change the result.
	return float32(reflected.Int())
}

func numberIntValue(value any) int32 {
	if number, ok := value.(JavaNumber); ok {
		return number.IntValue()
	}
	reflected := numericValue(value)
	if reflected.Kind() != reflect.Float32 && reflected.Kind() != reflect.Float64 {
		return int32(reflected.Int())
	}
	return numberFloatIntValue(reflected.Float())
}

func numberFloatIntValue(floating float64) int32 {
	switch {
	case math.IsNaN(floating):
		return 0
	case floating <= math.MinInt32:
		return math.MinInt32
	case floating >= math.MaxInt32:
		return math.MaxInt32
	default:
		return int32(floating)
	}
}

func NumberIntValue(value any) int32 { return numberIntValue(value) }
func NumberByteValue(value any) int8 {
	if number, ok := value.(JavaNumber); ok {
		return number.ByteValue()
	}
	return int8(numberIntValue(value))
}
func NumberShortValue(value any) int16 {
	if number, ok := value.(JavaNumber); ok {
		return number.ShortValue()
	}
	return int16(numberIntValue(value))
}

func NumberLongValue(value any) int64 {
	if number, ok := value.(JavaNumber); ok {
		return number.LongValue()
	}
	reflected := numericValue(value)
	if reflected.Kind() != reflect.Float32 && reflected.Kind() != reflect.Float64 {
		return reflected.Int()
	}
	return numberFloatLongValue(reflected.Float())
}

func numberFloatLongValue(floating float64) int64 {
	switch {
	case math.IsNaN(floating):
		return 0
	case floating <= math.MinInt64:
		return math.MinInt64
	case floating >= math.MaxInt64:
		return math.MaxInt64
	default:
		return int64(floating)
	}
}
