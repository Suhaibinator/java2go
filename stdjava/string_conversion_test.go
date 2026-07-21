package stdjava

import (
	"math"
	"testing"
)

type namedStringValue string

func (v namedStringValue) String() string { return string(v) }

func TestDoubleToStringMatchesJavaFormatting(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "positive zero", value: 0, want: "0.0"},
		{name: "negative zero", value: math.Copysign(0, -1), want: "-0.0"},
		{name: "integral value", value: 98, want: "98.0"},
		{name: "decimal value", value: 37.25, want: "37.25"},
		{name: "lower plain boundary", value: 0.001, want: "0.001"},
		{name: "below plain boundary", value: 0.0001, want: "1.0E-4"},
		{name: "upper plain boundary", value: 9_999_999, want: "9999999.0"},
		{name: "scientific boundary", value: 10_000_000, want: "1.0E7"},
		{name: "minimum subnormal", value: math.SmallestNonzeroFloat64, want: "4.9E-324"},
		{name: "positive infinity", value: math.Inf(1), want: "Infinity"},
		{name: "negative infinity", value: math.Inf(-1), want: "-Infinity"},
		{name: "not a number", value: math.NaN(), want: "NaN"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DoubleToString(test.value); got != test.want {
				t.Fatalf("DoubleToString(%v) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestFloatToStringMatchesJavaSubnormalFormatting(t *testing.T) {
	if got := FloatToString(math.SmallestNonzeroFloat32); got != "1.4E-45" {
		t.Fatalf("FloatToString(Float.MIN_VALUE) = %q", got)
	}
}

func TestStringValueOfUsesJavaFloatingAndStringerSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "null", value: nil, want: "null"},
		{name: "double", value: float64(98), want: "98.0"},
		{name: "float", value: float32(37), want: "37.0"},
		{name: "integer", value: int32(2), want: "2"},
		{name: "boolean", value: true, want: "true"},
		{name: "string", value: "text", want: "text"},
		{name: "stringer", value: namedStringValue("FRI"), want: "FRI"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := StringValueOf(test.value); got != test.want {
				t.Fatalf("StringValueOf(%#v) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}
