package stdjava

import "testing"

func TestMathAddExactJavaWidths(t *testing.T) {
	type javaInt int32
	type javaLong int64
	for _, test := range []struct {
		name, message string
		call          func()
	}{
		{"int positive", "integer overflow", func() { MathAddExact(int32(2147483647), int32(1)) }},
		{"int negative", "integer overflow", func() { MathAddExact(int32(-2147483648), int32(-1)) }},
		{"long positive", "long overflow", func() { MathAddExact(int64(9223372036854775807), int64(1)) }},
		{"long negative", "long overflow", func() { MathAddExact(int64(-9223372036854775808), int64(-1)) }},
		{"named int", "integer overflow", func() { MathAddExact(javaInt(2147483647), javaInt(1)) }},
		{"named long", "long overflow", func() { MathAddExact(javaLong(9223372036854775807), javaLong(1)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if !CaughtAs(got, "ArithmeticException") || GetMessage(got) != test.message {
					t.Fatalf("got %v, want ArithmeticException: %s", got, test.message)
				}
			}()
			test.call()
			t.Fatal("overflow did not panic")
		})
	}
	if got := MathAddExact(int32(2147483647), int32(-1)); got != 2147483646 {
		t.Fatalf("nonoverflowing int sum: %d", got)
	}
	if got := MathAddExact(javaLong(2147483647), javaLong(1)); got != 2147483648 {
		t.Fatalf("wide sum: %d", got)
	}
}
