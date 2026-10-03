package stdjava

import (
	"reflect"
	"testing"
)

// This test compiles against the original runtime: missing APIs fail at runtime,
// before any proposed ByteOrder type or primitive method is referenced in code.
func TestNIOPrimitiveMethodSurface(t *testing.T) {
	buffer := reflect.TypeOf(ByteBufferAllocate(16))
	for _, name := range []string{"Order", "SetOrder", "GetShort", "PutShort", "PutShortAt", "GetInt", "PutInt", "PutIntAt", "GetLong", "PutLong", "PutLongAt", "GetFloat", "PutFloat", "PutFloatAt", "GetDouble", "PutDouble", "PutDoubleAt"} {
		if _, ok := buffer.MethodByName(name); !ok {
			t.Errorf("ByteBuffer missing %s", name)
		}
	}
}
