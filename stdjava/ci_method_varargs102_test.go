package stdjava

import "testing"

func TestCI102MethodVarArgsMetadata(t *testing.T) {
	for _, flags := range []int32{0, 1, 128, 129, 1 | 8 | 128, 1 | 64 | 4096} {
		method := &Method{descriptor: MethodDescriptor{Modifiers: flags, HasModifiers: true}}
		getter, ok := any(method).(interface{ IsVarArgs() bool })
		if !ok {
			t.Fatal("canonical Method has no isVarArgs service")
		}
		if got := getter.IsVarArgs(); got != (flags&128 != 0) {
			t.Fatalf("flags=%d varargs=%t", flags, got)
		}
	}
	var absent *Method
	getter := any(absent).(interface{ IsVarArgs() bool })
	defer func() {
		if _, ok := recover().(NullPointerException); !ok {
			t.Fatal("null Method receiver did not throw canonical NullPointerException")
		}
	}()
	getter.IsVarArgs()
}
