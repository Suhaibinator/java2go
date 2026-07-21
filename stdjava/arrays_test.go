package stdjava

import "testing"

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
