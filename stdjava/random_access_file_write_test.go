package stdjava

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rafWriteChannelPosition(t *testing.T, c *FileChannel) int64 {
	t.Helper()
	position, err := c.state.file.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	}
	return position
}
func rafWriteArray(values ...int8) *PrimitiveArray[int8] {
	return PrimitiveArrayLiteral(PrimitiveByteTypeID, values...)
}
func rafWriteSignedBytes(data []byte) string {
	parts := make([]string, len(data))
	for i, b := range data {
		parts[i] = fmt.Sprint(int8(b))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
func rafWriteObserved(t *testing.T, e *Execution, label string, action func()) string {
	t.Helper()
	failure := func() (value any) { defer func() { value = recover() }(); action(); return nil }()
	if failure == nil {
		return label + "|ok\n"
	}
	throwable, ok := failure.(Throwable)
	if !ok {
		t.Fatalf("non-Java panic %T:%v", failure, failure)
	}
	message := JavaThrowableMessageExecution(e, failure)
	text := "null"
	if message != nil {
		text = string(unsignedBytes(JavaStringGetBytes(message, UTF_8).Elements))
	}
	id := BuiltinThrowableTypeID(throwable.ThrowableTypeName())
	return fmt.Sprintf("%s|%s|message=%s|cause-null=%t\n", label, id, text, GetCause(failure) == nil)
}
func TestRAFWriteNativeJDKState(t *testing.T) {
	e := NewExecution()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.data")
	seed := func(p string, b []byte) {
		t.Helper()
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	open := func(p, mode string) *RandomAccessFile {
		return NewRandomAccessFileStringExecution(e, JavaStringFromHostUTF8(p), JavaStringFromHostUTF8(mode))
	}
	seed(path, []byte{10, 11, 12, 13})
	r := open(path, "rw")
	defer r.Close()
	c := r.GetChannel()
	var out strings.Builder
	state := func(label string, r *RandomAccessFile, c *FileChannel, p string) {
		t.Helper()
		pos, channel := "closed", "closed"
		if c.IsOpen() {
			pos = fmt.Sprint(r.GetFilePointer())
			channel = fmt.Sprint(rafWriteChannelPosition(t, c))
		}
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&out, "%s|open=%t|pos=%s|channel=%s|length=%d|bytes=%s\n", label, c.IsOpen(), pos, channel, len(data), rafWriteSignedBytes(data))
	}
	call := func(label string, action func(), r *RandomAccessFile, c *FileChannel, p string) {
		out.WriteString(rafWriteObserved(t, e, label, action))
		state(label+"-state", r, c, p)
	}
	fmt.Fprintf(&out, "same-channel=%t\n", c == r.GetChannel())
	r.SeekPosition(3)
	length := func(n int64) func() { return func() { RandomAccessFileSetLengthExecution(e, r, n) } }
	call("truncate", length(2), r, c, path)
	call("extend", length(6), r, c, path)
	r.SeekPosition(9)
	call("same-length-clamp", length(6), r, c, path)
	r.SeekPosition(9)
	call("extend-clamp", length(8), r, c, path)
	call("negative", length(-1), r, c, path)
	call("negative-min", length(math.MinInt64), r, c, path)
	call("zero", length(0), r, c, path)
	input := rafWriteArray(0, -128, -1, 127)
	call("write-bytes", func() { RandomAccessFileWriteBytesExecution(e, r, input) }, r, c, path)
	fmt.Fprintf(&out, "input=%s\n", rafWriteSignedBytes(unsignedBytes(input.Elements)))
	r.SeekPosition(1)
	call("overwrite", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray(33, 34)) }, r, c, path)
	call("empty", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray()) }, r, c, path)
	call("null-array", func() { RandomAccessFileWriteBytesExecution(e, r, nil) }, r, c, path)
	ThreadInterruptExecution(e, ThreadCurrentThread(e))
	RandomAccessFileWriteBytesExecution(e, r, rafWriteArray(55))
	fmt.Fprintf(&out, "interrupted-write|ok|interrupt=%t|owner-pos=%d\n", ThreadIsInterruptedExecution(e, ThreadCurrentThread(e)), r.GetFilePointer())
	RandomAccessFileSetLengthExecution(e, r, 5)
	pending := ThreadIsInterruptedExecution(e, ThreadCurrentThread(e))
	pos, size := r.GetFilePointer(), r.Length()
	cleared := ThreadInterruptedExecution(e)
	fmt.Fprintf(&out, "interrupted-length|ok|interrupt=%t|owner-pos=%d|owner-length=%d|cleared=%t\n", pending, pos, size, cleared)
	state("after-cleared-interrupt", r, c, path)
	c.Close()
	call("closed-negative", length(-1), r, c, path)
	call("closed-zero", length(0), r, c, path)
	call("closed-null", func() { RandomAccessFileWriteBytesExecution(e, r, nil) }, r, c, path)
	call("closed-empty", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray()) }, r, c, path)
	call("closed-data", func() { RandomAccessFileWriteBytesExecution(e, r, rafWriteArray(9)) }, r, c, path)
	r.Close()
	for _, mode := range []string{"r", "rw", "rwd", "rws"} {
		p := filepath.Join(dir, mode+".data")
		seed(p, []byte{1, 2})
		f := open(p, mode)
		ch := f.GetChannel()
		call(mode+"-negative", func() { RandomAccessFileSetLengthExecution(e, f, -1) }, f, ch, p)
		call(mode+"-same", func() { RandomAccessFileSetLengthExecution(e, f, 2) }, f, ch, p)
		call(mode+"-extend", func() { RandomAccessFileSetLengthExecution(e, f, 4) }, f, ch, p)
		call(mode+"-empty", func() { RandomAccessFileWriteBytesExecution(e, f, rafWriteArray()) }, f, ch, p)
		call(mode+"-null", func() { RandomAccessFileWriteBytesExecution(e, f, nil) }, f, ch, p)
		call(mode+"-data", func() { RandomAccessFileWriteBytesExecution(e, f, rafWriteArray(66)) }, f, ch, p)
		f.Close()
	}
	big := filepath.Join(dir, "large.data")
	large := open(big, "rw")
	defer large.Close()
	RandomAccessFileWriteBytesExecution(e, large, rafWriteArray(7, 8))
	large.SeekPosition(6)
	n := int64(1<<33) + 3
	RandomAccessFileSetLengthExecution(e, large, n)
	fmt.Fprintf(&out, "large|pos=%d|channel=%d|length=%d\n", large.GetFilePointer(), rafWriteChannelPosition(t, large.GetChannel()), large.Length())
	large.SeekPosition(n - 1)
	RandomAccessFileWriteBytesExecution(e, large, rafWriteArray(77))
	fmt.Fprintf(&out, "large-tail|pos=%d|length=%d\n", large.GetFilePointer(), large.Length())
	RandomAccessFileSetLengthExecution(e, large, 2)
	fmt.Fprintf(&out, "large-truncate|pos=%d|channel=%d|length=%d\n", large.GetFilePointer(), rafWriteChannelPosition(t, large.GetChannel()), large.Length())
	large.Close()
	b, err := os.ReadFile(big)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Fprintf(&out, "large-bytes=%s\n", rafWriteSignedBytes(b))
	if out.String() != rafWriteStateJDK {
		t.Fatalf("native RAF/JDK whole state mismatch\ngot:\n%s\nwant:\n%s", out.String(), rafWriteStateJDK)
	}
	fmt.Print(out.String())
}

const rafWriteStateJDK = `same-channel=true
truncate|ok
truncate-state|open=true|pos=2|channel=2|length=2|bytes=[10, 11]
extend|ok
extend-state|open=true|pos=2|channel=2|length=6|bytes=[10, 11, 0, 0, 0, 0]
same-length-clamp|ok
same-length-clamp-state|open=true|pos=6|channel=6|length=6|bytes=[10, 11, 0, 0, 0, 0]
extend-clamp|ok
extend-clamp-state|open=true|pos=8|channel=8|length=8|bytes=[10, 11, 0, 0, 0, 0, 0, 0]
negative|java.io.IOException|message=Invalid argument|cause-null=true
negative-state|open=true|pos=8|channel=8|length=8|bytes=[10, 11, 0, 0, 0, 0, 0, 0]
negative-min|java.io.IOException|message=Invalid argument|cause-null=true
negative-min-state|open=true|pos=8|channel=8|length=8|bytes=[10, 11, 0, 0, 0, 0, 0, 0]
zero|ok
zero-state|open=true|pos=0|channel=0|length=0|bytes=[]
write-bytes|ok
write-bytes-state|open=true|pos=4|channel=4|length=4|bytes=[0, -128, -1, 127]
input=[0, -128, -1, 127]
overwrite|ok
overwrite-state|open=true|pos=3|channel=3|length=4|bytes=[0, 33, 34, 127]
empty|ok
empty-state|open=true|pos=3|channel=3|length=4|bytes=[0, 33, 34, 127]
null-array|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
null-array-state|open=true|pos=3|channel=3|length=4|bytes=[0, 33, 34, 127]
interrupted-write|ok|interrupt=true|owner-pos=4
interrupted-length|ok|interrupt=true|owner-pos=4|owner-length=5|cleared=true
after-cleared-interrupt|open=true|pos=4|channel=4|length=5|bytes=[0, 33, 34, 55, 0]
closed-negative|java.io.IOException|message=Stream Closed|cause-null=true
closed-negative-state|open=false|pos=closed|channel=closed|length=5|bytes=[0, 33, 34, 55, 0]
closed-zero|java.io.IOException|message=Stream Closed|cause-null=true
closed-zero-state|open=false|pos=closed|channel=closed|length=5|bytes=[0, 33, 34, 55, 0]
closed-null|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
closed-null-state|open=false|pos=closed|channel=closed|length=5|bytes=[0, 33, 34, 55, 0]
closed-empty|ok
closed-empty-state|open=false|pos=closed|channel=closed|length=5|bytes=[0, 33, 34, 55, 0]
closed-data|java.io.IOException|message=Stream Closed|cause-null=true
closed-data-state|open=false|pos=closed|channel=closed|length=5|bytes=[0, 33, 34, 55, 0]
r-negative|java.io.IOException|message=Invalid argument|cause-null=true
r-negative-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
r-same|java.io.IOException|message=Invalid argument|cause-null=true
r-same-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
r-extend|java.io.IOException|message=Invalid argument|cause-null=true
r-extend-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
r-empty|ok
r-empty-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
r-null|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
r-null-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
r-data|java.io.IOException|message=Bad file descriptor|cause-null=true
r-data-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rw-negative|java.io.IOException|message=Invalid argument|cause-null=true
rw-negative-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rw-same|ok
rw-same-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rw-extend|ok
rw-extend-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rw-empty|ok
rw-empty-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rw-null|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
rw-null-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rw-data|ok
rw-data-state|open=true|pos=1|channel=1|length=4|bytes=[66, 2, 0, 0]
rwd-negative|java.io.IOException|message=Invalid argument|cause-null=true
rwd-negative-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rwd-same|ok
rwd-same-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rwd-extend|ok
rwd-extend-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rwd-empty|ok
rwd-empty-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rwd-null|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
rwd-null-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rwd-data|ok
rwd-data-state|open=true|pos=1|channel=1|length=4|bytes=[66, 2, 0, 0]
rws-negative|java.io.IOException|message=Invalid argument|cause-null=true
rws-negative-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rws-same|ok
rws-same-state|open=true|pos=0|channel=0|length=2|bytes=[1, 2]
rws-extend|ok
rws-extend-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rws-empty|ok
rws-empty-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rws-null|java.lang.NullPointerException|message=Cannot read the array length because "b" is null|cause-null=true
rws-null-state|open=true|pos=0|channel=0|length=4|bytes=[1, 2, 0, 0]
rws-data|ok
rws-data-state|open=true|pos=1|channel=1|length=4|bytes=[66, 2, 0, 0]
large|pos=6|channel=6|length=8589934595
large-tail|pos=8589934595|length=8589934595
large-truncate|pos=2|channel=2|length=2
large-bytes=[7, 8]
`
