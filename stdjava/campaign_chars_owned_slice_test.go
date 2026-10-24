package stdjava

import (
	"slices"
	"testing"
)

// These controls assert the Java-facing snapshot and UTF16 contracts. They do
// not inspect stream storage or require a particular allocation count.
func TestCampaignCharsIndependentSnapshots(t *testing.T) {
	original := []uint16{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 0xffff}
	text := NewJavaStringUTF16(original)
	copyText := CopyJavaString(text)
	want := []int32{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 0xffff}
	first := JavaStringCharsStream(text)
	second := JavaStringCharsStream(copyText)
	original[0] = 'X'
	exported := first.ToSlice()
	exported[1] = 'Y'
	for name, stream := range map[string]Stream[int32]{
		"first":  first,
		"second": second,
		"later":  JavaStringCharsStream(text),
	} {
		if got := stream.ToSlice(); !slices.Equal(got, want) {
			t.Fatalf("%s stream = %x, want %x", name, got, want)
		}
		if stream.Count() != int64(len(want)) || stream.FindFirst().Get() != 0 {
			t.Fatalf("%s stream lost count or first NUL unit", name)
		}
	}
	if !slices.Equal(text.UTF16Copy(), []uint16{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 0xffff}) {
		t.Fatal("source String content changed through a constructor input or export")
	}
	if text == copyText || !text.Equals(copyText) {
		t.Fatal("chars changed copied String identity or equality")
	}
}

func TestCampaignCharsEmptyAndInvocationFailure(t *testing.T) {
	for _, input := range [][]uint16{nil, {}} {
		stream := JavaStringCharsStream(NewJavaStringUTF16(input))
		if stream.Count() != 0 || len(stream.ToSlice()) != 0 || stream.FindFirst().IsPresent() || StreamSum(stream) != 0 {
			t.Fatal("empty String chars must have no units and a zero sum")
		}
	}
	returned := false
	func() {
		defer func() {
			if recovered := recover(); !CaughtAs(recovered, "NullPointerException") {
				t.Fatalf("chars(null) = %v, want invocation-time NullPointerException", recovered)
			}
		}()
		_ = JavaStringCharsStream(nil)
		returned = true
	}()
	if returned {
		t.Fatal("chars(null) returned before a terminal operation")
	}
}

var campaignCharsStreamSink Stream[int32]
var campaignCharsSliceSink []int32
var campaignCharsSumSink int32

func campaignCharsBenchmarkInput(size int) *JavaString {
	pattern := []uint16{0, 'A', 0xd83d, 0xde00, 0xd800, 0xdfff, 0xffff, 'Z'}
	units := make([]uint16, size)
	for index := range units {
		units[index] = pattern[index%len(pattern)]
	}
	return NewJavaStringUTF16(units)
}

// Keep creation+reduction separate from the public export operation, which
// necessarily creates its own mutable snapshot.
func BenchmarkCampaignCharsCreateAndSum(b *testing.B) {
	for _, test := range []struct {
		name string
		size int
	}{{"empty", 0}, {"16units", 16}, {"4096units", 4096}} {
		b.Run(test.name, func(b *testing.B) {
			text := campaignCharsBenchmarkInput(test.size)
			want := StreamSum(JavaStringCharsStream(text))
			b.ReportAllocs()
			b.SetBytes(int64(test.size * 2))
			b.ResetTimer()
			var total int32
			for index := 0; index < b.N; index++ {
				stream := JavaStringCharsStream(text)
				total = StreamSum(stream)
				campaignCharsStreamSink = stream
			}
			b.StopTimer()
			campaignCharsSumSink = total
			if total != want {
				b.Fatalf("sum = %d, want %d", total, want)
			}
		})
	}
}

func BenchmarkCampaignCharsCreateAndExport(b *testing.B) {
	for _, test := range []struct {
		name string
		size int
	}{{"empty", 0}, {"16units", 16}, {"4096units", 4096}} {
		b.Run(test.name, func(b *testing.B) {
			text := campaignCharsBenchmarkInput(test.size)
			want := JavaStringCharsStream(text).ToSlice()
			b.ReportAllocs()
			b.SetBytes(int64(test.size * 2))
			b.ResetTimer()
			for index := 0; index < b.N; index++ {
				campaignCharsSliceSink = JavaStringCharsStream(text).ToSlice()
			}
			b.StopTimer()
			if !slices.Equal(campaignCharsSliceSink, want) {
				b.Fatal("export changed UTF16 units")
			}
		})
	}
}
