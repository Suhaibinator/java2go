//go:build darwin || linux

package stdjava

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"syscall"
	"testing"
	"time"
)

func positionalPanic(t *testing.T, name string, message *JavaString, action func()) any {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); action() }()
	throwable, ok := failure.(Throwable)
	if !ok || throwable.ThrowableTypeName() != name {
		t.Fatalf("want %s; recovered %#v", name, failure)
	}
	got := JavaThrowableMessageExecution(NewExecution(), failure)
	if got != message && (got == nil || message == nil || !got.Equals(message)) {
		t.Fatalf("%s message = %v; want %v", name, got, message)
	}
	if GetCause(failure) != nil {
		t.Fatal("operation error unexpectedly supplied cause")
	}
	return failure
}
func positionalRAF(t *testing.T, canonical bool, mode string) *RandomAccessFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.bin")
	if err := os.WriteFile(path, []byte{0, 1, 2, 3, 4, 5, 6, 7}, 0600); err != nil {
		t.Fatal(err)
	}
	var r *RandomAccessFile
	if canonical {
		r = NewRandomAccessFileStringExecution(NewExecution(), JavaStringFromHostUTF8(path), JavaStringFromHostUTF8(mode))
	} else {
		r = NewRandomAccessFile(path, mode)
	}
	t.Cleanup(func() { r.Close() })
	return r
}
func TestFileChannelPositionalWriteOrdinaryAndViews(t *testing.T) {
	for _, canonical := range []bool{false, true} {
		for _, mode := range []string{"rw", "rws", "rwd"} {
			for _, kind := range []string{"wrapped", "duplicate", "slice", "readonly"} {
				t.Run(mode+"/"+kind+"/"+map[bool]string{false: "legacy", true: "canonical"}[canonical], func(t *testing.T) {
					r := positionalRAF(t, canonical, mode)
					c := r.GetChannel()
					r.SeekPosition(6)
					a := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 17, 2, 3, 4, 5, 6, 7, 8)
					owner := ByteBufferWrap(a, 1, 6).Mark()
					view := owner
					switch kind {
					case "duplicate":
						view = owner.Duplicate()
					case "slice":
						view = owner.Slice()
					case "readonly":
						view = owner.AsReadOnlyBuffer()
					}
					view.Mark()
					start, limit, mark := view.position, view.limit, view.mark
					backing := slices.Clone(a.Elements)
					n := FileChannelWriteAtExecution(NewExecution(), c, view, 9)
					if n != 6 || view.position != start+6 || view.limit != limit || view.mark != mark || !slices.Equal(a.Elements, backing) {
						t.Fatalf("buffer/count: n=%d p=%d lim=%d mark=%d", n, view.position, view.limit, view.mark)
					}
					if view != owner && owner.position != 1 {
						t.Fatal("alias cursor moved")
					}
					if r.GetFilePointer() != 6 || r.Length() != 15 || c != r.GetChannel() {
						t.Fatal("cursor/growth/shared channel")
					}
					got := make([]byte, 6)
					if n, err := r.state.file.ReadAt(got, 9); n != 6 || err != nil || !bytes.Equal(got, []byte{2, 3, 4, 5, 6, 7}) {
						t.Fatalf("file bytes %v count=%d err=%v", got, n, err)
					}
					view.Reset()
					if view.position != mark {
						t.Fatal("mark lost")
					}
					r.Close()
					if c.IsOpen() {
						t.Fatal("RAF/channel close identity")
					}
				})
			}
		}
	}
	r := positionalRAF(t, true, "rw")
	r.SeekPosition(7)
	b := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 11, 12)).AsReadOnlyBuffer()
	const wide int64 = (1 << 31) + 17
	if n := FileChannelWriteAtExecution(NewExecution(), r.GetChannel(), b, wide); n != 2 || b.position != 2 || r.GetFilePointer() != 7 || r.Length() != wide+2 {
		t.Fatal("64-bit offset/cursor")
	}
	got := make([]byte, 2)
	if _, err := r.state.file.ReadAt(got, wide); err != nil || !bytes.Equal(got, []byte{11, 12}) {
		t.Fatalf("wide read %v/%v", got, err)
	}
}
func TestFileChannelPositionalWriteCanonicalOrdering(t *testing.T) {
	negative := JavaStringLiteralUTF16([]uint16{'N', 'e', 'g', 'a', 't', 'i', 'v', 'e', ' ', 'p', 'o', 's', 'i', 't', 'i', 'o', 'n'})
	for _, mode := range []string{"r", "rw"} {
		for _, closed := range []bool{false, true} {
			for _, which := range []string{"null-negative", "negative", "empty", "valid"} {
				t.Run(mode+"/"+map[bool]string{false: "open", true: "closed"}[closed]+"/"+which, func(t *testing.T) {
					r := positionalRAF(t, true, mode)
					c := r.GetChannel()
					if closed {
						c.Close()
					}
					b := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 1, 2, 3)).SetPosition(1).Mark()
					var src = b
					var offset int64
					name := ""
					var message *JavaString
					switch which {
					case "null-negative":
						src = nil
						offset = -1
						name = "NullPointerException"
					case "negative":
						offset = -1
						name = "IllegalArgumentException"
						message = negative
					case "empty":
						b.SetLimit(1)
					}
					if name == "" && closed {
						name = "ClosedChannelException"
					}
					if name == "" && mode == "r" {
						name = "NonWritableChannelException"
					}
					if name != "" {
						failure := positionalPanic(t, name, message, func() { FileChannelWriteAtExecution(NewExecution(), c, src, offset) })
						if name == "IllegalArgumentException" && JavaThrowableMessageExecution(NewExecution(), failure) != negative {
							t.Fatal("negative message not canonical literal")
						}
					} else {
						n := FileChannelWriteAtExecution(NewExecution(), c, src, offset)
						if n != map[bool]int32{true: 0, false: 2}[which == "empty"] {
							t.Fatal("count")
						}
					}
					if name != "" && (b.position != 1 || b.mark != 1) {
						t.Fatal("failed validation moved buffer")
					}
					if c.IsOpen() != !closed {
						t.Fatal("validation unexpectedly closed")
					}
				})
			}
		}
	}
}
func TestFileChannelPositionalWriteCallerPreinterrupt(t *testing.T) {
	for _, which := range []string{"valid", "empty", "null", "negative", "closed", "readmode", "close-error"} {
		t.Run(which, func(t *testing.T) {
			mode := "rw"
			if which == "readmode" {
				mode = "r"
			}
			r := positionalRAF(t, true, mode)
			c := r.GetChannel()
			b := ByteBufferWrap(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 1, 2, 3)).SetPosition(1).Mark()
			src := b
			var offset int64
			name := "ClosedByInterruptException"
			var message *JavaString
			switch which {
			case "empty":
				b.SetLimit(1)
			case "null":
				src = nil
				name = "NullPointerException"
			case "negative":
				offset = -1
				name = "IllegalArgumentException"
				message = JavaStringFromHostUTF8("Negative position")
			case "closed":
				c.Close()
				name = "ClosedChannelException"
			case "readmode":
				name = "NonWritableChannelException"
			case "close-error":
				if err := r.state.file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			execution := NewExecution()
			thread := ThreadCurrentThread(execution)
			thread.Interrupt()
			positionalPanic(t, name, message, func() { FileChannelWriteAtExecution(execution, c, src, offset) })
			if !thread.consumeInterrupt() || thread.consumeInterrupt() {
				t.Fatal("interrupt flag consumed/reset incorrectly")
			}
			if b.position != 1 || b.mark != 1 {
				t.Fatal("preinterrupt buffer effects")
			}
			if c.IsOpen() != (which == "null" || which == "negative" || which == "readmode") {
				t.Fatal("preinterrupt closure order")
			}
		})
	}
	r := positionalRAF(t, true, "rw")
	caller, other := &Execution{thread: NewThread(nil)}, &Execution{thread: NewThread(nil)}
	ThreadCurrentThread(caller).Interrupt()
	if n := FileChannelWriteAtExecution(other, r.GetChannel(), ByteBufferAllocate(0), 0); n != 0 || !r.GetChannel().IsOpen() {
		t.Fatal("other caller flag leaked")
	}
	if !ThreadCurrentThread(caller).consumeInterrupt() {
		t.Fatal("other caller consumed flag")
	}
}
func TestFileChannelPositionalWriteCountAndCompletion(t *testing.T) {
	r := positionalRAF(t, true, "rw")
	b := ByteBufferAllocate(8).SetPosition(1).Mark()
	if n, retry := fileChannelWriteFinish(r.state, b, 3, nil); n != 3 || retry || b.position != 4 || b.mark != 1 {
		t.Fatal("partial effects")
	}
	if n, retry := fileChannelWriteFinish(r.state, b, 0, syscall.EINTR); n != 0 || !retry || b.position != 4 {
		t.Fatal("open EINTR retry")
	}
	if n, retry := fileChannelWriteFinish(r.state, b, 0, syscall.EAGAIN); n != 0 || retry || b.position != 4 {
		t.Fatal("open EAGAIN zero")
	}
	r.Close()
	if n, retry := fileChannelWriteFinish(r.state, b, 1, nil); n != 1 || retry || b.position != 5 {
		t.Fatal("completed positive close effects")
	}
	for _, err := range []error{nil, syscall.EINTR, syscall.EAGAIN, syscall.EFBIG} {
		positionalPanic(t, "AsynchronousCloseException", nil, func() { fileChannelWriteFinish(r.state, b, 0, err) })
	}
	open := positionalRAF(t, true, "rw")
	source := ByteBufferAllocate(8).SetPosition(1).Mark()
	positionalPanic(t, "IOException", JavaStringFromHostUTF8("File too large"), func() { fileChannelWriteFinish(open.state, source, 2, syscall.EFBIG) })
	if source.position != 3 || source.mark != 1 {
		t.Fatal("positive effect rolled back before error")
	}
}
func TestFileChannelPositionalWriteEmptyCompletion(t *testing.T) {
	r := positionalRAF(t, true, "rw")
	r.SeekPosition(6)
	empty := ByteBufferAllocate(0).AsReadOnlyBuffer()
	if n := FileChannelWriteAtExecution(NewExecution(), r.GetChannel(), empty, int64(^uint64(0)>>1)); n != 0 || r.Length() != 8 || r.GetFilePointer() != 6 {
		t.Fatal("empty syscall/growth/cursor")
	}
	// Deterministic common-finish control: external close after entry snapshot,
	// before completion. No native syscall or active cancellation is simulated.
	r.Close()
	positionalPanic(t, "AsynchronousCloseException", nil, func() { fileChannelWriteFinish(r.state, empty, 0, nil) })
}
func TestFileChannelPositionalWriteRetainedLeaseClose(t *testing.T) {
	r := positionalRAF(t, true, "rw")
	raw, err := r.state.file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	entered, release, leaseDone, closeDone := make(chan struct{}), make(chan struct{}), make(chan error, 1), make(chan any, 1)
	go func() { leaseDone <- raw.Write(func(uintptr) bool { close(entered); <-release; return true }) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("lease did not enter")
	}
	go func() {
		var failure any
		func() { defer func() { failure = recover() }(); r.Close() }()
		closeDone <- failure
	}()
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
	select {
	case failure := <-closeDone:
		if failure != nil {
			t.Errorf("close panic %v", failure)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first close not joined")
	}
	if !observed {
		t.Fatal("closed state unavailable while lease retained (state.mu/FD cycle)")
	}
	// A second-close completion ordering promise is deliberately absent.
}
func TestFileChannelPositionalNonWritableDescriptor(t *testing.T) {
	base := newJavaThrowableBase("NonWritableChannelException", nil)
	actual := base.JavaDynamicTypeID()
	if actual != TypeID("java.nio.channels.NonWritableChannelException") || !JavaTypeAssignable(actual, BuiltinThrowableTypeID("IllegalStateException")) || base.javaThrowableMessage() != nil {
		t.Fatalf("descriptor=%q", actual)
	}
	for _, name := range []string{"foreign.NonWritableChannelException", "source.NonWritableChannelException"} {
		foreign := newJavaThrowableBase(name, nil)
		if foreign.JavaDynamicTypeID() != TypeID(name) || JavaTypeAssignable(foreign.JavaDynamicTypeID(), actual) {
			t.Fatal("foreign/source name became canonical")
		}
	}
}
