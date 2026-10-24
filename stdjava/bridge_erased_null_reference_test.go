package stdjava_test

import (
	"reflect"
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

// These methods must never be consulted by Java reference equality.
type bridgeNullIdentityImpostor struct{ *j.ObjectInfo }

func (*bridgeNullIdentityImpostor) IsNull() bool    { panic("IsNull callback") }
func (*bridgeNullIdentityImpostor) Equals(any) bool { panic("equals callback") }
func (*bridgeNullIdentityImpostor) String() string  { panic("toString callback") }

func TestBridgeErasedCanonicalNullRuntime(t *testing.T) {
	t.Run("erased_typed_null", func(t *testing.T) {
		var text *j.JavaString
		var array *j.ReferenceArray
		var source *bridgeNullIdentityImpostor
		var erased any = text
		if reflect.TypeOf(erased) != reflect.TypeOf(text) || !reflect.ValueOf(erased).IsNil() {
			t.Fatal("test did not preserve the typed-null erasure boundary")
		}
		if !j.JavaReferenceEqual(erased, nil) || !j.JavaReferenceEqual(nil, erased) ||
			!j.JavaReferenceEqual(erased, array) || !j.JavaReferenceEqual(source, erased) {
			t.Fatal("erased Java null did not retain reference equality")
		}
	})

	t.Run("canonical_pointer_identity", func(t *testing.T) {
		first := j.NewJavaStringUTF16([]uint16{'x', 0xd800})
		copy := j.CopyJavaString(first)
		empty := j.NewJavaStringUTF16(nil)
		if !first.Equals(copy) || j.JavaReferenceEqual(first, copy) {
			t.Fatal("equal UTF16 content collapsed distinct String allocations")
		}
		if !j.JavaReferenceEqual(any(first), first) || j.JavaReferenceEqual(empty, nil) {
			t.Fatal("canonical pointer identity or non-null empty String changed")
		}
	})

	t.Run("operands_once_left_to_right", func(t *testing.T) {
		var text *j.JavaString
		calls := 0
		trace := []string{}
		read := func() any {
			calls++
			trace = append(trace, "read")
			return text
		}
		right := func() any {
			trace = append(trace, "right")
			return nil
		}
		if !j.JavaReferenceEqual(read(), right()) || calls != 1 ||
			!reflect.DeepEqual(trace, []string{"read", "right"}) {
			t.Fatalf("erased equality effects: calls=%d trace=%q", calls, trace)
		}
		trace = nil
		if !j.JavaReferenceEqual(right(), read()) || calls != 2 ||
			!reflect.DeepEqual(trace, []string{"right", "read"}) {
			t.Fatalf("reversed equality effects: calls=%d trace=%q", calls, trace)
		}
	})

	t.Run("abrupt_operand_identity", func(t *testing.T) {
		failure := &struct{ marker int }{7}
		rightCalls := 0
		var caught any
		func() {
			defer func() { caught = recover() }()
			left := func() any { panic(failure) }
			right := func() any { rightCalls++; return nil }
			j.JavaReferenceEqual(left(), right())
		}()
		if caught != failure || rightCalls != 0 {
			t.Fatal("abrupt left operand changed exception identity or evaluated right")
		}
	})

	t.Run("source_impostor_and_host_nonnull", func(t *testing.T) {
		first := &bridgeNullIdentityImpostor{j.NewObjectInfo("bridge.SourceString", nil)}
		alias := &bridgeNullIdentityImpostor{first.ObjectInfo}
		second := &bridgeNullIdentityImpostor{j.NewObjectInfo("bridge.SourceString", nil)}
		if !j.JavaReferenceEqual(first, alias) || j.JavaReferenceEqual(first, second) ||
			j.JavaReferenceEqual(first, nil) {
			t.Fatal("source object identity changed or structural callbacks were consulted")
		}
		for _, value := range []any{"", false, int32(0), new(int), []int{}, map[string]int{}} {
			if j.JavaReferenceEqual(value, nil) || j.JavaReferenceEqual(nil, value) {
				t.Fatalf("non-null host representation %T collapsed to Java null", value)
			}
		}
	})
}
