//go:build (darwin || linux) && f1_os_profile

package stdjava

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestRAFWriteActualPartialJDK(t *testing.T) {
	path := os.Getenv("RAF_WRITE_PARTIAL_FILE")
	if path == "" {
		t.Fatal("explicit actual bounded file-size profile required")
	}
	if err := os.WriteFile(path, []byte{99}, 0600); err != nil {
		t.Fatal(err)
	}
	e := NewExecution()
	r := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(path), JavaStringFromHostUTF8("rw"))
	defer r.Close()
	r.SeekPosition(4093)
	values := rafWriteArray(13, 14, 15, 16, 17, 18, 19, 20)
	var out strings.Builder
	out.WriteString(rafWriteObserved(t, e, "partial-write", func() { RandomAccessFileWriteBytesExecution(e, r, values) }))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4096 {
		t.Fatalf("partial length %d", len(data))
	}
	fmt.Fprintf(&out, "partial-state|pos=%d|channel=%d|length=%d|prefix=%s|tail=%s|input=%s\n", r.GetFilePointer(), rafWriteChannelPosition(t, r.GetChannel()), r.Length(), rafWriteSignedBytes(data[:3]), rafWriteSignedBytes(data[4092:4096]), rafWriteSignedBytes(unsignedBytes(values.Elements)))
	out.WriteString(rafWriteObserved(t, e, "failed-extension", func() { RandomAccessFileSetLengthExecution(e, r, 4097) }))
	fmt.Fprintf(&out, "extension-state|pos=%d|length=%d\n", r.GetFilePointer(), r.Length())
	RandomAccessFileSetLengthExecution(e, r, 4090)
	fmt.Fprintf(&out, "truncate-state|pos=%d|length=%d\n", r.GetFilePointer(), r.Length())
	out.WriteString(rafWriteObserved(t, e, "second-partial", func() { RandomAccessFileWriteBytesExecution(e, r, values) }))
	fmt.Fprintf(&out, "second-state|pos=%d|length=%d\n", r.GetFilePointer(), r.Length())
	out.WriteString(rafWriteObserved(t, e, "full-boundary", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray(88)) }))
	r.Close()
	out.WriteString(rafWriteObserved(t, e, "closed-empty", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray()) }))
	if out.String() != rafWritePartialJDK {
		t.Fatalf("partial whole JDK mismatch\ngot:\n%s\nwant:\n%s", out.String(), rafWritePartialJDK)
	}
	fmt.Print(out.String())
}

func TestRAFWriteLeaseClosePublication(t *testing.T) {
	e := NewExecution()
	r := NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(t.TempDir()+"/file"), JavaStringFromHostUTF8("rw"))
	defer r.Close()
	raw, err := r.state.file.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	entered, release, leaseDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	releaseLease := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseLease)
	go func() {
		var statErr error
		err := raw.Control(func(fd uintptr) {
			close(entered)
			<-release
			var stat syscall.Stat_t
			statErr = syscall.Fstat(int(fd), &stat)
		})
		if err == nil {
			err = statErr
		}
		leaseDone <- err
	}()
	<-entered
	closed := make(chan struct{})
	go func() { r.Close(); close(closed) }()
	deadline := time.After(2 * time.Second)
	for r.GetChannel().IsOpen() {
		select {
		case <-deadline:
			releaseLease()
			t.Fatal("close failed to publish state while FD lease held")
		default:
			runtime.Gosched()
		}
	}
	// Blocking regular-file Close may return before the physical descriptor's
	// final lease is released. Its logical publication must remain available;
	// the retained callback verifies the actual descriptor is still valid.
	result := rafWriteObserved(t, e, "write", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray(1)) })
	if result != "write|java.io.IOException|message=Stream Closed|cause-null=true\n" {
		releaseLease()
		t.Fatal(result)
	}
	result = rafWriteObserved(t, e, "resize", func() { RandomAccessFileSetLengthExecution(e, r, -1) })
	if result != "resize|java.io.IOException|message=Stream Closed|cause-null=true\n" {
		releaseLease()
		t.Fatal(result)
	}
	RandomAccessFileWriteBytesExecution(e, r, rafWriteArray())
	releaseLease()
	if err := <-leaseDone; err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not complete after descriptor lease released")
	}
}

const rafWritePartialJDK = `partial-write|java.io.IOException|message=File too large|cause-null=true
partial-state|pos=4096|channel=4096|length=4096|prefix=[99, 0, 0]|tail=[0, 13, 14, 15]|input=[13, 14, 15, 16, 17, 18, 19, 20]
failed-extension|java.io.IOException|message=File too large|cause-null=true
extension-state|pos=4096|length=4096
truncate-state|pos=4090|length=4090
second-partial|java.io.IOException|message=File too large|cause-null=true
second-state|pos=4096|length=4096
full-boundary|java.io.IOException|message=File too large|cause-null=true
closed-empty|ok
`
