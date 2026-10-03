package stdjava

import (
	"math"
	"slices"
	"testing"
	"unicode/utf16"
)

func TestJavaStringValueOfSupportedFinalWrappers(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  []uint16
	}{
		{"Boolean", BoxBoolean(true), utf16.Encode([]rune("true"))},
		{"Byte", BoxByte(-128), utf16.Encode([]rune("-128"))},
		{"Short", BoxShort(-32768), utf16.Encode([]rune("-32768"))},
		{"Character", BoxCharacter(0xD800), []uint16{0xD800}},
		{"Integer", BoxInteger(7), utf16.Encode([]rune("7"))},
		{"Long", BoxLong(math.MaxInt64), utf16.Encode([]rune("9223372036854775807"))},
		{"Float", BoxFloat(float32(math.Copysign(0, -1))), utf16.Encode([]rune("-0.0"))},
		{"Double", BoxDouble(math.Inf(1)), utf16.Encode([]rune("Infinity"))},
		{"null Integer", (*Integer)(nil), utf16.Encode([]rune("null"))},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := JavaStringValueOfExecution(NewExecution(), test.value)
			if got == nil || !slices.Equal(got.UTF16Copy(), test.want) {
				t.Fatalf("canonical wrapper conversion did not preserve expected UTF16 %v", test.want)
			}
		})
	}
}

type nativeStringConversionTrap struct{}

func (*nativeStringConversionTrap) String() string {
	panic("unsupported value used native formatting")
}

func TestJavaStringValueOfNativeAdapterKeepsUnsupportedGuard(t *testing.T) {
	for _, value := range []any{int32(7), &nativeStringConversionTrap{}} {
		expectBoxedException(t, "UnsupportedOperationException", func() {
			JavaStringValueOfExecution(NewExecution(), value)
		})
	}
	text := NewJavaStringUTF16([]uint16{0xD800, 0, 0xDFFF})
	if got := JavaStringValueOfExecution(NewExecution(), text); got != text {
		t.Fatal("final-wrapper adapter changed an existing String reference")
	}
}
