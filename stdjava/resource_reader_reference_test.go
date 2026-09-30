package stdjava

import (
	"errors"
	"io/fs"
	"reflect"
	"slices"
	"testing"
	"testing/fstest"
)

// Observed values come from the immutable JDK21 v2 nine-run oracle 4194cdf7.
func resourceReaderExpectException(t *testing.T, kind, message string, operation func()) any {
	t.Helper()
	var failure any
	func() {
		defer func() { failure = recover() }()
		operation()
	}()
	if failure == nil || !CaughtAs(failure, kind) {
		t.Fatalf("expected %s, got %T: %v", kind, failure, failure)
	}
	if message != "" && GetMessage(failure) != message {
		t.Fatalf("%s message %q, expected %q", kind, GetMessage(failure), message)
	}
	return failure
}

func TestCanonicalResourceReaderResourceResolution(t *testing.T) {
	classResourceRegistry.Lock()
	previous := classResourceRegistry.roots
	classResourceRegistry.roots = []fs.FS{fstest.MapFS{
		"probe/lines/relative.bin": {Data: []byte{0, 1, 127, 128, 255}},
		"probe/other/relative.bin": {Data: []byte{17, 41, 97}},
		"root.bin":                 {Data: []byte{9, 13, 10}},
		"probe/lines/π.bin":        {Data: []byte{2, 3, 5, 7, 11}},
	}}
	classResourceRegistry.Unlock()
	t.Cleanup(func() {
		classResourceRegistry.Lock()
		classResourceRegistry.roots = previous
		classResourceRegistry.Unlock()
	})
	for _, test := range []struct {
		name, owner, path string
		want              []int8
	}{
		{"relative", "probe.lines.Main", "relative.bin", []int8{0, 1, 127, -128, -1}},
		{"absolute", "probe.lines.Main", "/probe/lines/relative.bin", []int8{0, 1, 127, -128, -1}},
		{"root", "probe.lines.Main", "/root.bin", []int8{9, 13, 10}},
		{"other-owner", "probe.other.Anchor", "relative.bin", []int8{17, 41, 97}},
		{"unicode-bmp", "probe.lines.Main", "π.bin", []int8{2, 3, 5, 7, 11}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := ClassLiteral(TypeID(test.owner)).GetResourceAsStreamReference(JavaStringFromHostUTF8(test.path))
			if stream == nil {
				t.Fatal("resource unexpectedly absent")
			}
			if got := stream.(*ByteArrayInputStream).ReadAllBytes().Elements; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("bytes %v != %v", got, test.want)
			}
			stream.Close()
			stream.Close()
		})
	}
	if ClassLiteral("probe.lines.Main").GetResourceAsStreamReference(JavaStringFromHostUTF8("missing.bin")) != nil {
		t.Fatal("missing resource must be null")
	}
}

func TestCanonicalResourceReaderUnicodeURLContract(t *testing.T) {
	classResourceRegistry.Lock()
	previous := classResourceRegistry.roots
	classResourceRegistry.roots = []fs.FS{fstest.MapFS{
		"probe/lines/π😀.bin":     {Data: []byte{2, 3, 5, 7, 11}},
		"probe/lines/other🚀.bin": {Data: []byte{23}},
	}}
	classResourceRegistry.Unlock()
	t.Cleanup(func() {
		classResourceRegistry.Lock()
		classResourceRegistry.roots = previous
		classResourceRegistry.Unlock()
	})
	owner := ClassLiteral("probe.lines.Main")
	for _, name := range []string{"π😀.bin", "other🚀.bin"} {
		t.Run(name, func(t *testing.T) {
			failure := resourceReaderExpectException(t, "IllegalArgumentException", "Error decoding percent encoded characters", func() { owner.GetResourceAsStreamReference(JavaStringFromHostUTF8(name)) })
			if GetCause(failure) != nil {
				t.Fatal("URL decoding failure must have null cause")
			}
		})
	}
	t.Run("missing-supplementary", func(t *testing.T) {
		if owner.GetResourceAsStreamReference(JavaStringFromHostUTF8("absent😀.bin")) != nil {
			t.Fatal("absent supplementary resource must remain null")
		}
	})
}

func TestCanonicalResourceReaderLineAndEOFContract(t *testing.T) {
	for _, test := range []struct {
		name  string
		bytes []byte
		lines [][]uint16
	}{
		{"mixed", []byte{13, 10, 65, 13, 66, 10, 0, 195, 169, 240, 159, 152, 128, 13, 10, 67}, [][]uint16{{}, {65}, {66}, {0, 233, 55357, 56832}, {67}}},
		{"empty", []byte{}, nil},
		{"blank", []byte{10}, [][]uint16{{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := NewBufferedReaderReference(NewInputStreamReaderExecution(nil, NewByteArrayInputStream(test.bytes), UTF_8), 2)
			for index, want := range test.lines {
				got := reader.ReadLineReference()
				if got == nil || !slices.Equal(got.units, want) {
					t.Fatalf("line %d: %v != %v", index, got, want)
				}
			}
			if reader.ReadLineReference() != nil || reader.ReadLineReference() != nil {
				t.Fatal("repeated EOF must be null")
			}
			reader.Close()
			resourceReaderExpectException(t, "IOException", "Stream closed", func() { reader.ReadLineReference() })
		})
	}
}

type resourceReaderTrackedInput struct {
	*ByteArrayInputStream
	closes       int
	readError    error
	closeFailure any
}

func (input *resourceReaderTrackedInput) Read(buffer []byte) (int, error) {
	if input.readError != nil {
		return 0, input.readError
	}
	return input.ByteArrayInputStream.Read(buffer)
}
func (input *resourceReaderTrackedInput) Close() {
	input.closes++
	if input.closeFailure != nil {
		panic(input.closeFailure)
	}
}

func TestCanonicalResourceReaderCloseOwnership(t *testing.T) {
	input := &resourceReaderTrackedInput{ByteArrayInputStream: NewByteArrayInputStream([]byte("A\nB\n"))}
	reader := NewBufferedReaderReference(NewInputStreamReaderExecution(nil, input, UTF_8), 2)
	if line := reader.ReadLineReference(); line == nil || !reflect.DeepEqual(line.units, []uint16{'A'}) {
		t.Fatal("first line must be A")
	}
	reader.Close()
	reader.Close()
	resourceReaderExpectException(t, "IOException", "Stream closed", func() { reader.ReadLineReference() })
	if input.closes != 1 {
		t.Fatalf("close count %d != 1", input.closes)
	}

	// JDK BufferedReader.close marks itself closed in finally if source.close throws.
	failure := NewIOException("sentinel-close")
	broken := &resourceReaderTrackedInput{ByteArrayInputStream: NewByteArrayInputStream([]byte("\n\n")), closeFailure: failure}
	closing := NewBufferedReaderReference(NewInputStreamReaderExecution(nil, broken, UTF_8), 2)
	first, second := closing.ReadLineReference(), closing.ReadLineReference()
	if first == nil || second == nil || first == second {
		t.Fatal("present empty lines must have distinct String objects")
	}
	got := resourceReaderExpectException(t, "IOException", "sentinel-close", closing.Close)
	if !sameThrowableIdentity(got, failure) {
		t.Fatal("source close exception identity changed")
	}
	closing.Close()
	resourceReaderExpectException(t, "IOException", "Stream closed", func() { closing.ReadLineReference() })
	if broken.closes != 1 {
		t.Fatalf("throwing source close count %d != 1", broken.closes)
	}
}

func TestCanonicalResourceReaderFailureContracts(t *testing.T) {
	t.Run("null-name", func(t *testing.T) {
		resourceReaderExpectException(t, "NullPointerException", "", func() { ClassLiteral("probe.lines.Main").GetResourceAsStreamReference(nil) })
	})
	t.Run("null-reader", func(t *testing.T) {
		resourceReaderExpectException(t, "NullPointerException", "", func() { NewBufferedReaderReference((*InputStreamReader)(nil)) })
	})
	t.Run("zero-size", func(t *testing.T) {
		resourceReaderExpectException(t, "IllegalArgumentException", "Buffer size <= 0", func() {
			NewBufferedReaderReference(NewInputStreamReaderExecution(nil, NewByteArrayInputStream([]byte{}), UTF_8), 0)
		})
	})
	t.Run("read-failure", func(t *testing.T) {
		input := &resourceReaderTrackedInput{ByteArrayInputStream: NewByteArrayInputStream([]byte("A\n")), readError: errors.New("sentinel-read")}
		reader := NewBufferedReaderReference(NewInputStreamReaderExecution(nil, input, UTF_8), 2)
		resourceReaderExpectException(t, "IOException", "sentinel-read", func() { reader.ReadLineReference() })
		reader.Close()
	})
}
