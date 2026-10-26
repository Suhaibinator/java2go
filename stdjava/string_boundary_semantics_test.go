package stdjava_test

import (
	"fmt"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
	"sync"
	"testing"
)

var boundaryTextSink string

type boundaryNamedString string
type boundaryStringer struct{ calls *int }

func (v boundaryStringer) String() string { *v.calls++; return "unexpected" }
func boundaryPanic(t *testing.T, f func(), class, message string) {
	t.Helper()
	defer func() {
		recovered := recover()
		throwable, ok := recovered.(stdjava.Throwable)
		if !ok || throwable.ThrowableTypeName() != class || throwable.Message() != message {
			t.Fatalf("panic got %T(%v), want %s(%s)", recovered, recovered, class, message)
		}
	}()
	f()
	t.Fatal("required panic absent")
}
func TestStringBoundarySemanticControls(t *testing.T) {
	for _, text := range []string{"", "ready", "a\x00b", "λ😀", string([]byte{0xff, 0xc0, 0xaf})} {
		if stdjava.StringIsNull(text) || stdjava.StringReferenceValue(text) != text || stdjava.StringRequireNonNull(text) != text {
			t.Fatalf("valid native text changed: %q", text)
		}
	}
	for _, null := range []any{nil, stdjava.NullString()} {
		if !stdjava.StringIsNull(null) || stdjava.StringReferenceValue(null) != stdjava.NullString() {
			t.Fatal("null representation changed")
		}
		boundaryPanic(t, func() { boundaryTextSink = stdjava.StringRequireNonNull(null) }, "NullPointerException", "String method called on null")
	}
	calls := 0
	for _, invalid := range []any{int32(7), boundaryNamedString("x"), (*string)(nil), []byte(nil), map[string]int(nil), boundaryStringer{&calls}} {
		if stdjava.StringIsNull(invalid) {
			t.Fatalf("native invalid value treated as null: %T", invalid)
		}
		want := fmt.Sprintf("cannot use %T as String", invalid)
		boundaryPanic(t, func() { boundaryTextSink = stdjava.StringReferenceValue(invalid) }, "ClassCastException", want)
		boundaryPanic(t, func() { boundaryTextSink = stdjava.StringRequireNonNull(invalid) }, "ClassCastException", want)
	}
	if calls != 0 {
		t.Fatal("type diagnostic called Stringer")
	}
}

type boundaryFormatter struct{ calls *int }

func (v boundaryFormatter) Format(_ fmt.State, _ rune) { *v.calls++ }

type boundaryNilStringer struct{}

func (*boundaryNilStringer) String() string { panic("type diagnostic must not call String") }
func TestStringBoundaryFormatterAndDynamicTypes(t *testing.T) {
	calls := 0
	for _, value := range []any{boundaryFormatter{&calls}, (*boundaryNilStringer)(nil), [2]int32{1, 2}, (chan int)(nil), (func())(nil), struct{ Field []int }{[]int{1}}, &struct{ N int }{3}} {
		want := fmt.Sprintf("cannot use %T as String", value)
		boundaryPanic(t, func() { boundaryTextSink = stdjava.StringRequireNonNull(value) }, "ClassCastException", want)
		boundaryPanic(t, func() { boundaryTextSink = stdjava.StringReferenceValue(value) }, "ClassCastException", want)
	}
	if calls != 0 {
		t.Fatal("type diagnostic invoked Formatter")
	}
}
func TestStringBoundaryConcurrentCheckedReads(t *testing.T) {
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for iteration := 0; iteration < 32; iteration++ {
				value := stdjava.StringRequireNonNull("a\x00λ😀")
				for index, want := range []rune{'a', 0, 0x03bb, 0xd83d, 0xde00} {
					if got := stdjava.StringCharAt(value, int32(index)); got != want {
						t.Errorf("checked unit %d got %x want %x", index, got, want)
					}
				}
			}
		}()
	}
	workers.Wait()
}
