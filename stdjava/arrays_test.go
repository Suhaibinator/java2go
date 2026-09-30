package stdjava

import "testing"

func TestNewArrayPreservesJavaLengthAndEmptyIdentityStorage(t *testing.T) {
	empty := NewArray[int32](int8(0))
	if len(empty) != 0 {
		t.Fatalf("NewArray length = %d, want 0", len(empty))
	}
	if cap(empty) != 1 {
		t.Fatalf("NewArray capacity = %d, want retained identity capacity 1", cap(empty))
	}

	values := NewArray[int32](int32(3))
	if len(values) != 3 || cap(values) != 3 {
		t.Fatalf("NewArray shape = len %d cap %d, want len 3 cap 3", len(values), cap(values))
	}
}

func TestNewStringArrayUsesJavaNullDefaults(t *testing.T) {
	values := NewArray[string](int32(3))
	for index, value := range values {
		if !StringIsNull(value) {
			t.Fatalf("NewArray[string](3)[%d] = %q, want Java null String", index, value)
		}
	}
	if len(NewArray[string](0)) != 0 {
		t.Fatal("empty String array must retain zero visible elements")
	}

	rows := NewArray[[]string](2)
	if rows[0] != nil || rows[1] != nil {
		t.Fatalf("partially allocated multidimensional rows = %#v, want nil references", rows)
	}
}

func TestArrayLiteralPreservesElementsAndEmptyIdentityStorage(t *testing.T) {
	empty := ArrayLiteral[int32]()
	if len(empty) != 0 || cap(empty) != 1 {
		t.Fatalf("empty ArrayLiteral shape = len %d cap %d, want len 0 cap 1", len(empty), cap(empty))
	}
	secondEmpty := ArrayLiteral[int32]()
	if &empty[:1][0] == &secondEmpty[:1][0] {
		t.Fatal("separate empty ArrayLiteral calls reused backing identity")
	}

	values := ArrayLiteral[int32](1, 2, 3)
	if len(values) != 3 || values[0] != 1 || values[2] != 3 {
		t.Fatalf("ArrayLiteral values = %v, want [1 2 3]", values)
	}
}

func TestNewArrayNegativeLengthThrowsJavaException(t *testing.T) {
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		NewArray[int32](int16(-1))
	}()
	if !CaughtAs(recovered, "NegativeArraySizeException") {
		t.Fatalf("negative NewArray panic = %T (%v), want NegativeArraySizeException", recovered, recovered)
	}
}

var arraySetBenchmarkSink int32

func TestArraySetChecksAndReturnsStoredValue(t *testing.T) {
	values := []int32{1, 2, 3}
	if got := ArraySet(values, int16(1), int32(9)); got != 9 || values[1] != 9 {
		t.Fatalf("ArraySet() = %d, values[1] = %d; want 9, 9", got, values[1])
	}
}

func TestArraySetExceptionIdentity(t *testing.T) {
	tests := []struct {
		name string
		call func()
		want string
	}{
		{"null", func() { ArraySet([]int32(nil), int32(0), int32(1)) }, "NullPointerException"},
		{"empty", func() { ArraySet([]int32{}, int32(0), int32(1)) }, "ArrayIndexOutOfBoundsException"},
		{"negative", func() { ArraySet([]int32{1}, int8(-1), int32(1)) }, "ArrayIndexOutOfBoundsException"},
		{"too-large", func() { ArraySet([]int32{1}, int32(1), int32(1)) }, "ArrayIndexOutOfBoundsException"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				if !CaughtAs(recovered, tc.want) {
					t.Fatalf("recovered %T (%v), want %s", recovered, recovered, tc.want)
				}
			}()
			tc.call()
		})
	}
}

func BenchmarkArraySet(b *testing.B) {
	values := make([]int32, 1024)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		arraySetBenchmarkSink = ArraySet(values, iteration&1023, int32(iteration))
	}
}

func BenchmarkDirectArraySet(b *testing.B) {
	values := make([]int32, 1024)
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		index := iteration & 1023
		values[index] = int32(iteration)
		arraySetBenchmarkSink = values[index]
	}
}
