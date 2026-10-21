//go:build darwin || linux

package stdjava

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func channelVirtualRWorkflow(t *testing.T, seed int) string {
	t.Helper()
	e, f, _ := readRFile(t, seed)
	c := f.GetChannel()
	f.SeekPosition(6)
	var out strings.Builder
	read := func(label string, dst *ByteBuffer, position int64) {
		failure := readRPanic(func() {
			count := FileChannelReadAtVirtualExecution(e, c, dst, position)
			fmt.Fprintf(&out, "%s:count=%d:dst=%d/%d:cursor=%d\n", label, count, dst.position, dst.limit, f.GetFilePointer())
		})
		if failure != nil {
			typ := throwableTypeName(failure)
			name := builtinThrowableDescriptors[typ].id
			fmt.Fprintf(&out, "%s:%s:message=%s:cause-null=%t\n", label, name, readRMessage(failure), GetCause(failure) == nil)
		}
	}
	root := ByteBufferAllocate(12)
	for i := range root.array.Elements {
		root.array.Elements[i] = -1
	}
	root.SetPosition(1).SetLimit(5).Mark()
	read("normal", root, 2)
	root.Reset()
	fmt.Fprintf(&out, "mark=%d\n", root.position)
	for i, v := range root.array.Elements {
		fmt.Fprintf(&out, "byte=%d:%d\n", i, v)
	}
	duplicate := root.Duplicate().SetPosition(2).SetLimit(4)
	read("duplicate", duplicate, 7)
	read("slice", root.Slice(), 14)
	read("eof", ByteBufferAllocate(4), 16)
	read("beyond", ByteBufferAllocate(4), 32)
	read("empty", ByteBufferAllocate(0), 32)
	read("readonly", ByteBufferAllocate(4).AsReadOnlyBuffer(), 0)
	read("readonly-empty", ByteBufferAllocate(0).AsReadOnlyBuffer(), 0)
	read("null-negative", nil, -1)
	read("negative", ByteBufferAllocate(1), -1)
	c.Close()
	read("closed-null-negative", nil, -1)
	read("closed-negative", ByteBufferAllocate(1), -1)
	read("closed-readonly", ByteBufferAllocate(0).AsReadOnlyBuffer(), 0)
	fmt.Fprintf(&out, "closed=%t\n", c.IsOpen())
	return out.String()
}
func TestFileChannelVirtualNativeReadPreservesRContracts(t *testing.T) {
	for _, seed := range []int{17, 41, 97} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			if got := channelVirtualRWorkflow(t, seed); got != readRJDKWhole[seed] {
				t.Fatalf("unchanged full R oracle mismatch\ngot:\n%s\nwant:\n%s", got, readRJDKWhole[seed])
			}
		})
	}
	t.Run("NativeIdentityJDK", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "identity.bin")
		if err := os.WriteFile(path, []byte{1, 2, 3}, 0600); err != nil {
			t.Fatal(err)
		}
		e := NewExecution()
		first := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("rw"))
		second := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("r"))
		defer first.Close()
		defer second.Close()
		a, b := first.GetChannel(), second.GetChannel()
		var out strings.Builder
		fmt.Fprintf(&out, "same=%t:distinct=%t:open=%t\n", a == first.GetChannel(), a != b, FileChannelIsOpenVirtual(a))
		FileChannelCloseVirtualExecution(e, a)
		fmt.Fprintf(&out, "closed=%t:other=%t\n", FileChannelIsOpenVirtual(a), FileChannelIsOpenVirtual(b))
		if out.String() != channelNativeIdentityJDK {
			t.Fatalf("full native JDK identity mismatch: %s", out.String())
		}
		if a.JavaObjectInfo() != nil || b.JavaObjectInfo() != nil {
			t.Fatal("native abstract metadata was fabricated")
		}
	})
	t.Run("RelativeWindowAndInterrupt", func(t *testing.T) {
		e, f, data := readRFile(t, 17)
		c := f.GetChannel()
		f.SeekPosition(3)
		dst := ByteBufferAllocate(4)
		dst.SetPosition(1).SetLimit(3)
		if got := FileChannelReadVirtualExecution(e, c, dst); got != 2 || dst.position != 3 || f.GetFilePointer() != 5 || byte(dst.array.Elements[1]) != data[3] || byte(dst.array.Elements[2]) != data[4] {
			t.Fatal("relative native cursor/window changed")
		}
		other := NewExecution()
		ThreadInterruptDefaultExecution(other, ThreadCurrentThread(e))
		failure := channelProbePanic(func() { FileChannelReadVirtualExecution(e, c, nil) })
		if !CaughtAs(failure, "ClosedByInterruptException") || FileChannelIsOpenVirtual(c) || !ThreadIsInterruptedDefaultExecution(other, ThreadCurrentThread(e)) {
			t.Fatal("native relative interruption order changed")
		}
		ThreadInterruptedExecution(e)
	})
}

// Full original NativeIdentityControl stdout captured with pinned JDK21 before producer.
const channelNativeIdentityJDK = `same=true:distinct=true:open=true
closed=false:other=true
`
