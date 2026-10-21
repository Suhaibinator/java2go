//go:build darwin || linux

package stdjava

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeMatrixRAF(t *testing.T, mode string) *RandomAccessFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), "write.bin")
	if err := os.WriteFile(path, []byte{10, 11, 12, 13, 14, 15, 16, 17}, 0600); err != nil {
		t.Fatal(err)
	}
	r := NewRandomAccessFileStringExecution(NewExecution(), JavaStringFromHostUTF8(path), JavaStringFromHostUTF8(mode))
	t.Cleanup(r.Close)
	return r
}
func writeMatrixSource(readonly, empty, absent bool) *ByteBuffer {
	if absent {
		return nil
	}
	b := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 40, 41, 42, 43, 44))
	b.SetPosition(1)
	if empty {
		b.SetLimit(1)
	} else {
		b.SetLimit(4)
	}
	if readonly {
		return b.AsReadOnlyBuffer()
	}
	return b
}
func writeMatrixFailure(t *testing.T, failure any) string {
	t.Helper()
	throwable, ok := failure.(Throwable)
	if !ok {
		t.Fatalf("non-Java failure %#v", failure)
	}
	message := JavaThrowableMessageExecution(NewExecution(), failure)
	text := "null"
	if message != nil {
		text = string(unsignedBytes(JavaStringGetBytes(message, UTF_8).Elements))
	}
	if GetCause(failure) != nil {
		t.Fatal("unexpected cause")
	}
	return throwable.ThrowableTypeName() + ":" + text
}
func writeMatrixCall(t *testing.T, e *Execution, c *FileChannel, b *ByteBuffer, relative bool, position int64) string {
	t.Helper()
	var count int32
	failure := channelProbePanic(func() {
		if relative {
			count = FileChannelWriteVirtualExecution(e, c, b)
		} else {
			count = FileChannelWriteAtVirtualExecution(e, c, b, position)
		}
	})
	if failure != nil {
		return "throw:" + writeMatrixFailure(t, failure)
	}
	return fmt.Sprint("return:", count)
}
func TestFileChannelWriteNativeJDKMatrix(t *testing.T) {
	var out strings.Builder
	check := func(mode string, relative, closed, readonly, empty, absent bool, position int64, tag string) {
		r := writeMatrixRAF(t, mode)
		c := r.GetChannel()
		cursorStart := int64(6)
		if relative {
			cursorStart = 2
		}
		r.SeekPosition(cursorStart)
		b := writeMatrixSource(readonly, empty, absent)
		if closed {
			c.Close()
		}
		result := writeMatrixCall(t, NewExecution(), c, b, relative, position)
		cursor := "closed"
		if c.IsOpen() {
			cursor = fmt.Sprint(r.GetFilePointer())
		}
		src := "null"
		if b != nil {
			src = fmt.Sprint(b.position, ":", b.limit)
		}
		data, err := os.ReadFile(r.state.file.Name())
		if err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(data))
		for i, value := range data {
			parts[i] = fmt.Sprint(value)
		}
		fmt.Fprintf(&out, "%s|%s|src=%s|cursor=%s|open=%t|data=%s\n", tag, result, src, cursor, c.IsOpen(), strings.Join(parts, ","))
		r.Close()
	}
	for _, mode := range []string{"rw", "r"} {
		for _, closed := range []bool{false, true} {
			for _, readonly := range []bool{false, true} {
				for _, empty := range []bool{false, true} {
					for _, absent := range []bool{false, true} {
						for _, position := range []int64{-1, 2} {
							check(mode, false, closed, readonly, empty, absent, position, fmt.Sprintf("at:%s:%t:%t:%t:%t:%d", mode, closed, readonly, empty, absent, position))
						}
					}
				}
			}
		}
	}
	for _, mode := range []string{"rw", "r"} {
		for _, closed := range []bool{false, true} {
			for _, readonly := range []bool{false, true} {
				for _, empty := range []bool{false, true} {
					for _, absent := range []bool{false, true} {
						check(mode, true, closed, readonly, empty, absent, 0, fmt.Sprintf("relative:%s:%t:%t:%t:%t", mode, closed, readonly, empty, absent))
					}
				}
			}
		}
	}
	check("rw", false, false, false, true, false, 1<<63-1, "empty-max-position")
	check("r", false, false, true, true, false, 1<<63-1, "readonly-empty-max-position")
	if out.String() != writerNativeJDK {
		t.Fatalf("actual JDK matrix mismatch\ngot:\n%s\nwant:\n%s", out.String(), writerNativeJDK)
	}
}
func TestFileChannelWritePendingJDKMatrix(t *testing.T) {
	var out strings.Builder
	for _, mode := range []string{"rw", "r"} {
		for _, relative := range []bool{false, true} {
			for _, closed := range []bool{false, true} {
				for _, absent := range []bool{false, true} {
					for _, empty := range []bool{false, true} {
						for _, position := range []int64{-1, 2} {
							r := writeMatrixRAF(t, mode)
							c := r.GetChannel()
							r.SeekPosition(6)
							b := writeMatrixSource(true, empty, absent)
							if closed {
								c.Close()
							}
							e := NewExecution()
							ThreadInterruptDefaultExecution(NewExecution(), ThreadCurrentThread(e))
							result := writeMatrixCall(t, e, c, b, relative, position)
							pending := ThreadIsInterruptedDefaultExecution(NewExecution(), ThreadCurrentThread(e))
							ThreadInterruptedExecution(e)
							cursor := "closed"
							if c.IsOpen() {
								cursor = fmt.Sprint(r.GetFilePointer())
							}
							src := "null"
							if b != nil {
								src = fmt.Sprint(b.position)
							}
							fmt.Fprintf(&out, "pending:%s:%t:%t:%t:%t:%d|%s|src=%s|cursor=%s|open=%t|pending=%t\n", mode, relative, closed, absent, empty, position, result, src, cursor, c.IsOpen(), pending)
							r.Close()
						}
					}
				}
			}
		}
	}
	if out.String() != writerPendingJDK {
		t.Fatalf("pending JDK mismatch\ngot:\n%s\nwant:\n%s", out.String(), writerPendingJDK)
	}
}
func TestFileChannelWriteCursorAliasesAndSparse(t *testing.T) {
	r := writeMatrixRAF(t, "rw")
	c := r.GetChannel()
	e := NewExecution()
	r.SeekPosition(3)
	a := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 99, -128, -1, 0, 127, 66)
	owner := ByteBufferWrap(a).SetPosition(1).SetLimit(5).Mark()
	src := owner.Slice().AsReadOnlyBuffer().Mark()
	if n := FileChannelWriteVirtualExecution(e, c, src); n != 4 || src.position != 4 || owner.position != 1 || owner.mark != 1 || r.GetFilePointer() != 7 {
		t.Fatal("relative count/cursor/alias")
	}
	got := make([]byte, 4)
	if _, err := r.state.file.ReadAt(got, 3); err != nil || !bytes.Equal(got, []byte{128, 255, 0, 127}) {
		t.Fatalf("signed source window %v/%v", got, err)
	}
	a.Elements[2] = 31
	src.Reset()
	if n := FileChannelWriteAtVirtualExecution(e, c, src, 0); n != 4 || r.GetFilePointer() != 7 {
		t.Fatal("positional changed cursor")
	}
	if _, err := r.state.file.ReadAt(got, 0); err != nil || !bytes.Equal(got, []byte{128, 31, 0, 127}) {
		t.Fatal("native window cached across alias mutation")
	}
	r.SeekPosition(1)
	src.Reset()
	if FileChannelWriteVirtualExecution(e, c, src) != 4 || r.GetFilePointer() != 5 {
		t.Fatal("seek/write shared cursor")
	}
	read := make([]byte, 1)
	if n, err := r.Read(read); n != 1 || err != nil || r.GetFilePointer() != 6 {
		t.Fatal("RAF read/write cursor")
	}
	const wide int64 = (1 << 33) + 3
	small := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 70, 71))
	if FileChannelWriteAtVirtualExecution(e, c, small, wide) != 2 || r.GetFilePointer() != 6 || r.Length() != wide+2 {
		t.Fatal("sparse positional cursor before seek")
	}
	got = make([]byte, 2)
	if _, err := r.state.file.ReadAt(got, wide); err != nil || !bytes.Equal(got, []byte{70, 71}) {
		t.Fatal("sparse bytes")
	}
	if err := r.state.file.Truncate(8); err != nil {
		t.Fatal(err)
	}
}
func TestFileChannelWriteRelativeRetainedLeaseClose(t *testing.T) {
	r := writeMatrixRAF(t, "rw")
	raw, err := r.state.file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	leaseDone, closeDone, writeDone := make(chan error, 1), make(chan any, 1), make(chan any, 1)
	go func() { leaseDone <- raw.Write(func(uintptr) bool { close(entered); <-release; return true }) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("lease did not enter")
	}
	// This live write can wait behind the retained lease. It must not own state.mu.
	go func() {
		writeDone <- channelProbePanic(func() { FileChannelWriteVirtualExecution(NewExecution(), r.GetChannel(), ByteBufferAllocate(1)) })
	}()
	go func() { closeDone <- channelProbePanic(r.Close) }()
	deadline := time.Now().Add(2 * time.Second)
	observed := false
	for time.Now().Before(deadline) {
		if !r.GetChannel().IsOpen() {
			observed = true
			break
		}
		runtime.Gosched()
	}
	close(release)
	select {
	case err := <-leaseDone:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lease not released")
	}
	for _, done := range []chan any{closeDone, writeDone} {
		select {
		case failure := <-done:
			if failure != nil && !CaughtAs(failure, "AsynchronousCloseException") && !CaughtAs(failure, "ClosedChannelException") {
				t.Errorf("unexpected close/write failure %v", failure)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("close/write did not join")
		}
	}
	if !observed {
		t.Fatal("state.mu held over write lease; closed publication unavailable")
	}
}

// Captured actual JDK21 stdout before producer; first native98 and pending64 complete matrix rows.
const writerNativeJDK = `at:rw:false:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:false:false:2|return:3|src=4:4|cursor=6|open=true|data=10,11,41,42,43,15,16,17
at:rw:false:false:false:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:false:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:true:false:2|return:0|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:false:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:false:false:2|return:3|src=4:4|cursor=6|open=true|data=10,11,41,42,43,15,16,17
at:rw:false:true:false:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:false:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:true:false:2|return:0|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:false:true:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:rw:true:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:false:false:2|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:false:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:false:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:true:false:2|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:false:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:false:false:2|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:false:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:false:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:true:false:2|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:rw:true:true:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:false:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:false:false:2|throw:NonWritableChannelException:null|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:false:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:false:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:true:false:2|throw:NonWritableChannelException:null|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:false:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:false:false:2|throw:NonWritableChannelException:null|src=1:4|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:false:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:false:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:true:false:2|throw:NonWritableChannelException:null|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:false:true:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|data=10,11,12,13,14,15,16,17
at:r:true:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:false:false:2|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:false:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:false:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:true:false:2|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:false:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:false:false:2|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:false:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:false:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:true:false:-1|throw:IllegalArgumentException:Negative position|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:true:false:2|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
at:r:true:true:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:false:false:false:false|return:3|src=4:4|cursor=5|open=true|data=10,11,41,42,43,15,16,17
relative:rw:false:false:false:true|throw:NullPointerException:Cannot invoke "java.nio.ByteBuffer.position()" because "src" is null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:false:false:true:false|return:0|src=1:1|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:false:false:true:true|throw:NullPointerException:Cannot invoke "java.nio.ByteBuffer.position()" because "src" is null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:false:true:false:false|return:3|src=4:4|cursor=5|open=true|data=10,11,41,42,43,15,16,17
relative:rw:false:true:false:true|throw:NullPointerException:Cannot invoke "java.nio.ByteBuffer.position()" because "src" is null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:false:true:true:false|return:0|src=1:1|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:false:true:true:true|throw:NullPointerException:Cannot invoke "java.nio.ByteBuffer.position()" because "src" is null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:rw:true:false:false:false|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:false:false:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:false:true:false|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:false:true:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:true:false:false|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:true:false:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:true:true:false|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:rw:true:true:true:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:false:false:false:false|throw:NonWritableChannelException:null|src=1:4|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:false:false:true|throw:NonWritableChannelException:null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:false:true:false|throw:NonWritableChannelException:null|src=1:1|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:false:true:true|throw:NonWritableChannelException:null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:true:false:false|throw:NonWritableChannelException:null|src=1:4|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:true:false:true|throw:NonWritableChannelException:null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:true:true:false|throw:NonWritableChannelException:null|src=1:1|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:false:true:true:true|throw:NonWritableChannelException:null|src=null|cursor=2|open=true|data=10,11,12,13,14,15,16,17
relative:r:true:false:false:false|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:false:false:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:false:true:false|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:false:true:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:true:false:false|throw:ClosedChannelException:null|src=1:4|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:true:false:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:true:true:false|throw:ClosedChannelException:null|src=1:1|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
relative:r:true:true:true:true|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|data=10,11,12,13,14,15,16,17
empty-max-position|return:0|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
readonly-empty-max-position|throw:NonWritableChannelException:null|src=1:1|cursor=6|open=true|data=10,11,12,13,14,15,16,17
`
const writerPendingJDK = `pending:rw:false:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=6|open=true|pending=true
pending:rw:false:false:false:false:2|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:false:false:false:true:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=6|open=true|pending=true
pending:rw:false:false:false:true:2|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:false:false:true:false:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:rw:false:false:true:false:2|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:rw:false:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:rw:false:false:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:rw:false:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=closed|open=false|pending=true
pending:rw:false:true:false:false:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:false:true:false:true:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=closed|open=false|pending=true
pending:rw:false:true:false:true:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:false:true:true:false:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:false:true:true:false:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:false:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:false:true:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:false:false:false:-1|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:false:false:false:2|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:false:false:true:-1|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:false:false:true:2|throw:ClosedByInterruptException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:false:true:false:-1|throw:ClosedByInterruptException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:false:true:false:2|throw:ClosedByInterruptException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:false:true:true:-1|throw:ClosedByInterruptException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:false:true:true:2|throw:ClosedByInterruptException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:true:false:false:-1|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:true:false:false:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:true:false:true:-1|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:true:false:true:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:rw:true:true:true:false:-1|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:true:true:false:2|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:true:true:true:-1|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:rw:true:true:true:true:2|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:r:false:false:false:false:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=6|open=true|pending=true
pending:r:false:false:false:false:2|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:false:false:false:true:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=6|open=true|pending=true
pending:r:false:false:false:true:2|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:false:false:true:false:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:r:false:false:true:false:2|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:r:false:false:true:true:-1|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:r:false:false:true:true:2|throw:NullPointerException:null|src=null|cursor=6|open=true|pending=true
pending:r:false:true:false:false:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=closed|open=false|pending=true
pending:r:false:true:false:false:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:false:true:false:true:-1|throw:IllegalArgumentException:Negative position|src=1|cursor=closed|open=false|pending=true
pending:r:false:true:false:true:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:false:true:true:false:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:r:false:true:true:false:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:r:false:true:true:true:-1|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:r:false:true:true:true:2|throw:NullPointerException:null|src=null|cursor=closed|open=false|pending=true
pending:r:true:false:false:false:-1|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:true:false:false:false:2|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:true:false:false:true:-1|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:true:false:false:true:2|throw:NonWritableChannelException:null|src=1|cursor=6|open=true|pending=true
pending:r:true:false:true:false:-1|throw:NonWritableChannelException:null|src=null|cursor=6|open=true|pending=true
pending:r:true:false:true:false:2|throw:NonWritableChannelException:null|src=null|cursor=6|open=true|pending=true
pending:r:true:false:true:true:-1|throw:NonWritableChannelException:null|src=null|cursor=6|open=true|pending=true
pending:r:true:false:true:true:2|throw:NonWritableChannelException:null|src=null|cursor=6|open=true|pending=true
pending:r:true:true:false:false:-1|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:true:true:false:false:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:true:true:false:true:-1|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:true:true:false:true:2|throw:ClosedChannelException:null|src=1|cursor=closed|open=false|pending=true
pending:r:true:true:true:false:-1|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:r:true:true:true:false:2|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:r:true:true:true:true:-1|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
pending:r:true:true:true:true:2|throw:ClosedChannelException:null|src=null|cursor=closed|open=false|pending=true
`
