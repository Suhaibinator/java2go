package stdjava

import (
	"math"
	"reflect"
)

// JavaNumber is the generated representation of a Java type parameter bounded
// by java.lang.Number. Boxed numeric values use their Go scalar counterparts.
type JavaNumber interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

func numericValue(value any) reflect.Value {
	if value == nil {
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
	reflected := numericValue(value)
	if reflected.Kind() == reflect.Float32 || reflected.Kind() == reflect.Float64 {
		return reflected.Float()
	}
	return float64(reflected.Int())
}

func NumberFloatValue(value any) float32 { return float32(NumberDoubleValue(value)) }

func numberIntValue(value any) int32 {
	reflected := numericValue(value)
	if reflected.Kind() != reflect.Float32 && reflected.Kind() != reflect.Float64 {
		return int32(reflected.Int())
	}
	floating := reflected.Float()
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

func NumberIntValue(value any) int32   { return numberIntValue(value) }
func NumberByteValue(value any) int8   { return int8(numberIntValue(value)) }
func NumberShortValue(value any) int16 { return int16(numberIntValue(value)) }

func NumberLongValue(value any) int64 {
	reflected := numericValue(value)
	if reflected.Kind() != reflect.Float32 && reflected.Kind() != reflect.Float64 {
		return reflected.Int()
	}
	floating := reflected.Float()
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

// DoubleValueOf implements the String and numeric overloads of Double.valueOf.
func DoubleValueOf(value any) float64 {
	if text, ok := value.(string); ok {
		return ParseDouble(text)
	}
	return NumberDoubleValue(value)
}
