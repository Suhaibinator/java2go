package stdjava

import (
	"slices"
	"testing"
)

func TestJavaStringCharsStreamUTF16(t *testing.T) {
	units := []uint16{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 'Z'}
	text := NewJavaStringUTF16(units)
	want := []int32{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 'Z'}
	if got := JavaStringCharsStream(text).ToSlice(); !slices.Equal(got, want) {
		t.Fatalf("chars = %v, want exact UTF16 units %v", got, want)
	}
	if got := JavaStringCharsStream(text).Count(); got != int64(len(units)) {
		t.Fatalf("chars count = %d, want %d", got, len(units))
	}
	var sum int32
	for _, unit := range want {
		sum += unit
	}
	if got := StreamSum(JavaStringCharsStream(text)); got != sum {
		t.Fatalf("chars sum = %d, want %d", got, sum)
	}
	exported := JavaStringCharsStream(text).ToSlice()
	exported[0] = 42
	if !slices.Equal(text.UTF16Copy(), units) || !slices.Equal(JavaStringCharsStream(text).ToSlice(), want) {
		t.Fatal("chars stream exposed immutable String storage")
	}
	if got := JavaStringCharsStream(NewJavaStringUTF16(nil)).Count(); got != 0 {
		t.Fatalf("empty chars count = %d", got)
	}
}

func TestJavaStringCharsStreamNullReceiver(t *testing.T) {
	defer func() {
		caught := recover()
		if caught == nil || !CaughtAs(caught, "NullPointerException") {
			t.Fatalf("chars(null) panic = %v, want Java NullPointerException", caught)
		}
	}()
	_ = JavaStringCharsStream(nil)
	t.Fatal("chars(null) returned without the required invocation-time exception")
}
