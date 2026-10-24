package stdjava

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func readerControlPanic(f func()) (value any) { defer func() { value = recover() }(); f(); return }
func requireReaderFailure(t *testing.T, kind, message string, f func()) {
	t.Helper()
	value := readerControlPanic(f)
	if value == nil || !CaughtAs(value, kind) || GetMessage(value) != message {
		t.Fatalf("want %s:%s; got %T:%v", kind, message, value, value)
	}
}
func readerUnits(v *JavaString) []uint16 {
	if v == nil {
		return nil
	}
	return v.UTF16Copy()
}

func TestStringReaderCanonicalUnitsAndState(t *testing.T) {
	e := NewExecution()
	text := NewJavaStringUTF16([]uint16{'A', 0xd800, 0xdc00, 0xdc01, 'Z'})
	a := NewStringReaderReference(text)
	got := []int32{}
	for i := 0; i < 6; i++ {
		got = append(got, StringReaderReadCharExecution(e, a))
	}
	if !slices.Equal(got, []int32{65, 55296, 56320, 56321, 90, -1}) {
		t.Fatal(got)
	}
	for _, n := range []int64{-1, math.MinInt64, math.MaxInt64} {
		if StringReaderSkipExecution(e, a, n) != 0 {
			t.Fatal("EOF skip")
		}
	}
	StringReaderResetExecution(e, a)
	if StringReaderReadCharExecution(e, a) != 65 || StringReaderSkipExecution(e, a, -1) != -1 || StringReaderReadCharExecution(e, a) != 65 {
		t.Fatal("initial mark/backward skip")
	}
	StringReaderMarkExecution(e, a, 0)
	buf := PrimitiveArrayLiteral[rune](PrimitiveTypeID("char"), 'x', 'x', 'x', 'x', 'x')
	if n := StringReaderReadCharsExecution(e, a, buf, 1, 3); n != 3 || !slices.Equal(buf.Elements, []rune{'x', 0xd800, 0xdc00, 0xdc01, 'x'}) {
		t.Fatal(n, buf.Elements)
	}
	StringReaderResetExecution(e, a)
	if StringReaderReadCharExecution(e, a) != 55296 || StringReaderSkipExecution(e, a, math.MinInt64) != -2 || StringReaderReadCharExecution(e, a) != 65 {
		t.Fatal("mark/min skip")
	}
	if StringReaderSkipExecution(e, a, math.MaxInt64) != 4 || StringReaderReadCharsExecution(e, a, buf, 0, 0) != 0 || StringReaderReadCharsExecution(e, a, buf, 0, 1) != -1 || !StringReaderReadyExecution(e, a) {
		t.Fatal("EOF/ready")
	}
	StringReaderCloseExecution(e, a)
	StringReaderCloseExecution(e, a)
	if !StringReaderMarkSupportedExecution(e, a) {
		t.Fatal("closed markSupported")
	}
	requireReaderFailure(t, "IllegalArgumentException", "Read-ahead limit < 0", func() { StringReaderMarkExecution(e, a, -1) })
	requireReaderFailure(t, "IOException", "Stream closed", func() { StringReaderMarkExecution(e, a, 1) })
	requireReaderFailure(t, "IOException", "Stream closed", func() { StringReaderReadCharsExecution(e, a, nil, 0, 0) })
	requireReaderFailure(t, "IOException", "Stream closed", func() { StringReaderReadCharsExecution(e, a, buf, 0, 0) })
}

func TestStringReaderCanonicalValidationAndMessageIdentity(t *testing.T) {
	e := NewExecution()
	a := NewStringReaderReference(NewJavaStringUTF16([]uint16{'Q'}))
	buf := NewPrimitiveArray[rune](5, PrimitiveTypeID("char"))
	requireReaderFailure(t, "NullPointerException", `Cannot read the array length because "cbuf" is null`, func() { StringReaderReadCharsExecution(e, a, nil, 0, 0) })
	requireReaderFailure(t, "IndexOutOfBoundsException", "Range [-1, -1 + 0) out of bounds for length 5", func() { StringReaderReadCharsExecution(e, a, buf, -1, 0) })
	requireReaderFailure(t, "NullPointerException", `Cannot invoke "String.length()" because "s" is null`, func() { NewStringReaderReference(nil) })
	StringReaderCloseExecution(e, a)
	thrown := readerControlPanic(func() { StringReaderReadyExecution(e, a) })
	if !JavaReferenceEqual(JavaThrowableMessageExecution(e, thrown), JavaStringLiteralUTF16([]uint16{'S', 't', 'r', 'e', 'a', 'm', ' ', 'c', 'l', 'o', 's', 'e', 'd'})) {
		t.Fatal("fixed Stream closed message must be literal identity")
	}
	var nilReader *StringReader
	if v := readerControlPanic(func() { StringReaderMarkSupportedExecution(e, nilReader) }); !CaughtAs(v, "NullPointerException") {
		t.Fatal(v)
	}
}

func TestStringReaderCanonicalIdentityAndSourceLifetime(t *testing.T) {
	text := NewJavaStringUTF16([]uint16{0xd800, 'Q'})
	a := NewStringReaderReference(text)
	b := NewStringReaderReference(text)
	e := NewExecution()
	if JavaReferenceEqual(a, b) || !JavaReferenceEqual(a, ObjectView[Reader](a, JavaIOReaderType)) || !JavaReferenceEqual(a, ObjectView[Closeable](a, JavaIOCloseableType)) {
		t.Fatal("reader allocation/views")
	}
	if a.reference.text != text {
		t.Fatal("constructor lost original immutable source identity")
	}
	if ReaderReadCharExecution(e, a) != 0xd800 {
		t.Fatal("Reader protocol decoded surrogate")
	}
	StringReaderCloseExecution(e, a)
	if a.reference.text != nil || a.JavaObjectInfo() == nil {
		t.Fatal("close retention/identity")
	}
}

func TestBufferedStringReaderCanonicalUnitsAndPrefetch(t *testing.T) {
	e := NewExecution()
	text := []uint16{'\r', '\n', 'A', 0xd800, '\r', 'B', 0xdc01, '\n', '\n', 'Z'}
	wants := [][]uint16{{}, {'A', 0xd800}, {'B', 0xdc01}, {}, {'Z'}, nil}
	for size := int32(1); size <= 3; size++ {
		source := NewStringReaderReference(NewJavaStringUTF16(text))
		b := NewBufferedReaderReference(source, size)
		for i, want := range wants {
			got := BufferedStringReaderReadLineExecution(e, b)
			if (got == nil) != (want == nil) || !slices.Equal(readerUnits(got), want) {
				t.Fatalf("size %d line %d got%v want%v", size, i, readerUnits(got), want)
			}
		}
		BufferedStringReaderCloseExecution(e, b)
		BufferedStringReaderCloseExecution(e, b)
	}
	source := NewStringReaderReference(NewJavaStringUTF16([]uint16{'a', '\n', 'b', '\n'}))
	b := NewBufferedReaderReference(source, 4)
	if !slices.Equal(readerUnits(BufferedStringReaderReadLineExecution(e, b)), []uint16{'a'}) || StringReaderReadCharExecution(e, source) != -1 {
		t.Fatal("fill must prefetch full sized buffer")
	}
	StringReaderCloseExecution(e, source)
	if !slices.Equal(readerUnits(BufferedStringReaderReadLineExecution(e, b)), []uint16{'b'}) {
		t.Fatal("closed source must retain prefetched chars")
	}
	requireReaderFailure(t, "IOException", "Stream closed", func() { BufferedStringReaderReadLineExecution(e, b) })
	BufferedStringReaderCloseExecution(e, b)
	BufferedStringReaderCloseExecution(e, b)
	empty := func() *JavaString {
		r := NewBufferedReaderReference(NewStringReaderReference(NewJavaStringUTF16([]uint16{'\n'})))
		s := BufferedStringReaderReadLineExecution(e, r)
		BufferedStringReaderCloseExecution(e, r)
		return s
	}
	if first, second := empty(), empty(); first == nil || second == nil || JavaReferenceEqual(first, second) || len(first.units) != 0 {
		t.Fatal("fresh empty line")
	}
}

func readerBlockedWorker(e *Execution, r *StringReader, done chan<- any) {
	done <- readerControlPanic(func() { StringReaderReadCharExecution(e, r) })
}
func bufferedClosedBlockedWorker(e *Execution, r *BufferedReader, done chan<- any) {
	done <- readerControlPanic(func() { BufferedStringReaderReadLineExecution(e, r) })
}
func requireReaderBlockedStack(t *testing.T, name string, done <-chan any) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case v := <-done:
			t.Fatalf("worker completed while Java lock held: %v", v)
		default:
		}
		stack := make([]byte, 128*1024)
		n := runtime.Stack(stack, true)
		for _, block := range strings.Split(string(stack[:n]), "\n\n") {
			if strings.Contains(block, name) && strings.Contains(block, "MonitorEnterExecution") && (strings.Contains(block, "sync.") || strings.Contains(block, "semacquire")) {
				return
			}
		}
		runtime.Gosched()
	}
	t.Fatal("same worker did not block inside actual Java monitor")
}
func TestStringReaderCanonicalMonitorAndBufferedClosedLock(t *testing.T) {
	e := NewExecution()
	source := NewStringReaderReference(NewJavaStringUTF16([]uint16{'Q'}))
	guard := MonitorEnterExecution(e, source)
	// Same execution is reentrant and keeps external ownership after the read.
	if StringReaderReadyExecution(e, source) != true || !ThreadHoldsLockExecution(e, source) {
		t.Fatal("reentrant ownership")
	}
	done := make(chan any, 1)
	go readerBlockedWorker(NewExecution(), source, done)
	requireReaderBlockedStack(t, "readerBlockedWorker", done)
	MonitorExitExecution(guard)
	select {
	case v := <-done:
		if v != nil {
			t.Fatal(v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker not released")
	}
	b := NewBufferedReaderReference(source, 1)
	BufferedStringReaderCloseExecution(e, b)
	guard = MonitorEnterExecution(e, source)
	defer func() {
		if guard != nil {
			MonitorExitExecution(guard)
		}
	}()
	done = make(chan any, 1)
	go bufferedClosedBlockedWorker(NewExecution(), b, done)
	requireReaderBlockedStack(t, "bufferedClosedBlockedWorker", done)
	MonitorExitExecution(guard)
	guard = nil
	select {
	case v := <-done:
		if !CaughtAs(v, "IOException") {
			t.Fatal(v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("closed B worker not released")
	}
	if b.reference.lock != source || b.reference.source != nil || b.reference.units != nil {
		t.Fatal("close must retain lock and release readable storage")
	}
}

func TestStringReaderReferenceLegacyStateAbsentPreservesHostAPI(t *testing.T) {
	legacy := NewStringReader("😀Q")
	if n := legacy.ReadChar(); n != 128512 {
		t.Fatal("legacy host rune API changed", n)
	}
	if legacy.JavaObjectInfo() != nil || legacy.JavaDynamicTypeID() != "" {
		t.Fatal("legacy nominal fabrication")
	}
	legacy.Close()
	if legacy.ReadChar() != 'Q' {
		t.Fatal("legacy no-op Close changed")
	}
	legacy = NewStringReader("a\nb")
	b := NewBufferedReaderReference(legacy)
	if fmt.Sprint(readerUnits(BufferedStringReaderReadLineExecution(NewExecution(), b))) != "[97]" {
		t.Fatal("legacy buffered branch")
	}
	BufferedStringReaderCloseExecution(NewExecution(), b)
	value := readerControlPanic(func() { ReaderReadCharExecution(NewExecution(), legacy) })
	if !CaughtAs(value, "UnsupportedOperationException") {
		t.Fatal("state-absent protocol refusal", value)
	}
}
