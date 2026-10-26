//go:build (darwin || linux) && f1_os_profile

package stdjava

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

// This mandatory profile is built and selected separately, with the identical
// verified child input profile used by the genuine JVM controls. Keeping this
// explicit tag avoids weakening the ordinary whole-runtime test environment.
func TestFileChannelPositionalWriteActualOSPartial(t *testing.T) {
	path := os.Getenv("F1_WORKFILE")
	seed, err := strconv.Atoi(os.Getenv("F1_SEED"))
	if path == "" || err != nil {
		t.Fatal("explicit frozen input/profile required")
	}
	execution := NewExecution()
	r := NewRandomAccessFileStringExecution(execution, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("rw"))
	c := r.GetChannel()
	defer r.Close()
	r.SeekPosition(int64(seed % 11))
	a := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, int8(seed), 31, 32, 33, 34, 35, 36, 37)
	owner := ByteBufferWrap(a).SetPosition(1).Mark()
	src := owner.AsReadOnlyBuffer().Mark()
	fmt.Printf("workflow=partial;seed=%d\n", seed)
	n := FileChannelWriteAtExecution(execution, c, src, 4093)
	fmt.Printf("partialCount=%d;seed=%d\n", n, seed)
	printState := func(tag string) {
		fmt.Printf("%s;open=%t;position=%d;limit=%d;capacity=%d;readonly=%t\n", tag, c.IsOpen(), src.position, src.limit, src.capacity, src.readOnly)
		fmt.Printf("cursor=%d;length=%d\n", r.GetFilePointer(), r.Length())
	}
	printState("partial")
	fmt.Printf("aliasPosition=%d;hasArray=%t\n", owner.position, src.HasArray())
	got := make([]byte, n)
	if count, err := r.state.file.ReadAt(got, 4093); err != nil || count != int(n) {
		t.Fatalf("readback count=%d err=%v", count, err)
	}
	text := ""
	for i, value := range got {
		if i != 0 {
			text += ","
		}
		text += strconv.Itoa(int(value))
	}
	fmt.Printf("bytes@4093=%s;restoredCursor=%d\n", text, r.GetFilePointer())
	if n != 3 || src.position != 4 || src.limit != 8 || src.mark != 1 || owner.position != 1 || r.GetFilePointer() != int64(seed%11) || r.Length() != 4096 || string(got) != string([]byte{31, 32, 33}) {
		t.Fatal("actual service effects differ from frozen JVM profile")
	}
	fmt.Println("boundaryWrite")
	failure := positionalPanic(t, "IOException", JavaStringFromHostUTF8("File too large"), func() { FileChannelWriteAtExecution(execution, c, src, 4096) })
	actual, known := ObjectDynamicType(failure)
	if !known {
		t.Fatal("exception has no nominal identity")
	}
	message := JavaThrowableMessageExecution(execution, failure)
	fmt.Printf("exception=%s;message=%s;null=%t;negativeIdentity=%t;ISE=%t;RTE=%t;IO=%t\n", actual, string(unsignedBytes(JavaStringGetBytes(message, UTF_8).Elements)), message == nil, message == JavaStringLiteralUTF16([]uint16{'N', 'e', 'g', 'a', 't', 'i', 'v', 'e', ' ', 'p', 'o', 's', 'i', 't', 'i', 'o', 'n'}), CaughtAsType(failure, BuiltinThrowableTypeID("IllegalStateException")), CaughtAsType(failure, BuiltinThrowableTypeID("RuntimeException")), CaughtAsType(failure, BuiltinThrowableTypeID("IOException")))
	printState("boundary")
	if src.position != 4 || r.GetFilePointer() != int64(seed%11) || r.Length() != 4096 {
		t.Fatal("error changed source/cursor/length")
	}
	src.Reset()
	owner.Reset()
	fmt.Printf("marks=%d,%d;backing=%d\n", src.position, owner.position, owner.Get(1))
	r.Close()
	c.Close()
	fmt.Printf("cleanup=%t;pending=%t\n", !c.IsOpen(), ThreadCurrentThread(execution).consumeInterrupt())
}
