//go:build darwin || linux

package stdjava

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode/utf16"
)

func readRPanic(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}
func readRMessage(failure any) string {
	value := JavaThrowableMessageDefault(failure)
	if value == nil {
		return "null"
	}
	return string(utf16.Decode(value.units))
}
func readRFile(t *testing.T, seed int) (*Execution, *RandomAccessFile, []byte) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.bin")
	data := make([]byte, 16)
	for i := range data {
		data[i] = byte(seed + 13*i)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	e := &Execution{thread: NewThread(nil)}
	f := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("r"))
	t.Cleanup(f.Close)
	return e, f, data
}
func readRWorkflow(t *testing.T, seed int) string {
	t.Helper()
	e, f, _ := readRFile(t, seed)
	c := f.GetChannel()
	f.SeekPosition(6)
	var out strings.Builder
	read := func(label string, dst *ByteBuffer, position int64) {
		failure := readRPanic(func() {
			count := FileChannelReadAtExecution(e, c, dst, position)
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

func TestFileChannelReadAtJDKWholeVectors(t *testing.T) {
	for _, seed := range []int{17, 41, 97} {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			if got := readRWorkflow(t, seed); got != readRJDKWhole[seed] {
				t.Fatalf("whole JDK mismatch\ngot:\n%s\nwant:\n%s", got, readRJDKWhole[seed])
			}
		})
	}
}
func TestFileChannelReadAtPreinterruptAndCallerIsolation(t *testing.T) {
	e, f, _ := readRFile(t, 17)
	other := &Execution{thread: NewThread(nil)}
	thread := ThreadCurrentThread(e)
	ThreadInterruptDefaultExecution(other, thread)
	failure := readRPanic(func() { FileChannelReadAtExecution(e, f.GetChannel(), ByteBufferAllocate(0).AsReadOnlyBuffer(), 0) })
	if !CaughtAs(failure, "ClosedByInterruptException") || JavaThrowableMessageDefault(failure) != nil || f.GetChannel().IsOpen() || !ThreadIsInterruptedDefaultExecution(other, thread) || ThreadInterruptedExecution(other) {
		t.Fatal("preinterrupt order/flag/caller/cleanup changed")
	}
	if !ThreadInterruptedExecution(e) || ThreadInterruptedExecution(e) {
		t.Fatal("caller flag not retained and clearable once")
	}
	e, f, _ = readRFile(t, 41)
	ThreadInterruptDefaultExecution(e, ThreadCurrentThread(other))
	if n := FileChannelReadAtExecution(e, f.GetChannel(), ByteBufferAllocate(2), 0); n != 2 || !f.GetChannel().IsOpen() {
		t.Fatal("another caller flag affected read")
	}
}
func TestFileChannelReadAtCompletionAndRetry(t *testing.T) {
	e, f, _ := readRFile(t, 17)
	_ = e
	dst := ByteBufferAllocate(5).SetPosition(1).Mark()
	stage := []byte{3, 4, 5, 6}
	count, retry := fileChannelReadAtFinish(f.state, dst, stage, 2, nil, false)
	if count != 2 || retry || dst.position != 3 || dst.array.Elements[1] != 3 || dst.array.Elements[2] != 4 || dst.mark != 1 {
		t.Fatal("positive effects")
	}
	f.Close()
	failure := readRPanic(func() { fileChannelReadAtFinish(f.state, dst, stage, 1, syscall.EIO, false) })
	if !CaughtAs(failure, "IOException") || dst.position != 4 || dst.array.Elements[3] != 3 {
		t.Fatal("positive effects before error lost")
	}
	for _, empty := range []bool{true, false} {
		failure = readRPanic(func() { fileChannelReadAtFinish(f.state, dst, nil, 0, nil, empty) })
		if !CaughtAs(failure, "AsynchronousCloseException") || JavaThrowableMessageDefault(failure) != nil {
			t.Fatal("incomplete external close")
		}
	}
	_, f, _ = readRFile(t, 41)
	dst = ByteBufferAllocate(1)
	for _, v := range []struct {
		err   error
		want  int32
		retry bool
	}{{nil, -1, false}, {syscall.EINTR, 0, true}, {syscall.EAGAIN, 0, false}} {
		got, again := fileChannelReadAtFinish(f.state, dst, nil, 0, v.err, false)
		if got != v.want || again != v.retry || dst.position != 0 {
			t.Fatal("EOF/error/retry ordering")
		}
	}
	if got, again := fileChannelReadAtFinish(f.state, dst, nil, 0, nil, true); got != 0 || again {
		t.Fatal("empty treated as EOF")
	}
	if failure := readRPanic(func() { fileChannelReadAtFinish(f.state, dst, nil, 0, syscall.EIO, false) }); !CaughtAs(failure, "IOException") {
		t.Fatal("error treated as EOF")
	}
}
func TestFileChannelReadAtLeaseCloseAndWideOffset(t *testing.T) {
	_, f, _ := readRFile(t, 97)
	raw, err := f.state.file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	leased := make(chan error, 1)
	go func() { leased <- raw.Read(func(uintptr) bool { close(entered); <-release; return true }) }()
	<-entered
	closed := make(chan struct{})
	go func() { f.Close(); close(closed) }()
	deadline := time.Now().Add(time.Second)
	for f.GetChannel().IsOpen() {
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("close state unavailable during retained lease")
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	if err := <-leased; err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("close/lease deadlock")
	}
	e, f, _ := readRFile(t, 17)
	dst := ByteBufferAllocate(1)
	if got := FileChannelReadAtExecution(e, f.GetChannel(), dst, int64(1)<<33); got != -1 || dst.position != 0 {
		t.Fatal("wide absolute position changed")
	}
}

// Frozen from preproducer installed JDK21 whole applications, three identical repeats per seed.
var readRJDKWhole = map[int]string{
	17: "normal:count=4:dst=5/5:cursor=6\nmark=1\nbyte=0:-1\nbyte=1:43\nbyte=2:56\nbyte=3:69\nbyte=4:82\nbyte=5:-1\nbyte=6:-1\nbyte=7:-1\nbyte=8:-1\nbyte=9:-1\nbyte=10:-1\nbyte=11:-1\nduplicate:count=2:dst=4/4:cursor=6\nslice:count=2:dst=2/4:cursor=6\neof:count=-1:dst=0/4:cursor=6\nbeyond:count=-1:dst=0/4:cursor=6\nempty:count=0:dst=0/0:cursor=6\nreadonly:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nreadonly-empty:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nnull-negative:java.lang.NullPointerException:message=null:cause-null=true\nnegative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-null-negative:java.lang.NullPointerException:message=null:cause-null=true\nclosed-negative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-readonly:java.nio.channels.ClosedChannelException:message=null:cause-null=true\nclosed=false\n",
	41: "normal:count=4:dst=5/5:cursor=6\nmark=1\nbyte=0:-1\nbyte=1:67\nbyte=2:80\nbyte=3:93\nbyte=4:106\nbyte=5:-1\nbyte=6:-1\nbyte=7:-1\nbyte=8:-1\nbyte=9:-1\nbyte=10:-1\nbyte=11:-1\nduplicate:count=2:dst=4/4:cursor=6\nslice:count=2:dst=2/4:cursor=6\neof:count=-1:dst=0/4:cursor=6\nbeyond:count=-1:dst=0/4:cursor=6\nempty:count=0:dst=0/0:cursor=6\nreadonly:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nreadonly-empty:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nnull-negative:java.lang.NullPointerException:message=null:cause-null=true\nnegative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-null-negative:java.lang.NullPointerException:message=null:cause-null=true\nclosed-negative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-readonly:java.nio.channels.ClosedChannelException:message=null:cause-null=true\nclosed=false\n",
	97: "normal:count=4:dst=5/5:cursor=6\nmark=1\nbyte=0:-1\nbyte=1:123\nbyte=2:-120\nbyte=3:-107\nbyte=4:-94\nbyte=5:-1\nbyte=6:-1\nbyte=7:-1\nbyte=8:-1\nbyte=9:-1\nbyte=10:-1\nbyte=11:-1\nduplicate:count=2:dst=4/4:cursor=6\nslice:count=2:dst=2/4:cursor=6\neof:count=-1:dst=0/4:cursor=6\nbeyond:count=-1:dst=0/4:cursor=6\nempty:count=0:dst=0/0:cursor=6\nreadonly:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nreadonly-empty:java.lang.IllegalArgumentException:message=Read-only buffer:cause-null=true\nnull-negative:java.lang.NullPointerException:message=null:cause-null=true\nnegative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-null-negative:java.lang.NullPointerException:message=null:cause-null=true\nclosed-negative:java.lang.IllegalArgumentException:message=Negative position:cause-null=true\nclosed-readonly:java.nio.channels.ClosedChannelException:message=null:cause-null=true\nclosed=false\n",
}
