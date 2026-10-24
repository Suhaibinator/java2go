package stdjava

// Identical benchmark bytes must run on frozen baseline/candidate.
// Setup is excluded; public fresh results escape. No allocation count is assumed.
import (
	"fmt"
	"testing"
)

var utf8Owned181ArraySink *PrimitiveArray[int8]

func utf8Owned181Input(kind string, n int) *JavaString {
	pattern := []uint16{'A', 'B', 'C', 'D'}
	if kind == "unicode" {
		pattern = []uint16{'A', 0xe9, 0x65e5, 0xd83d, 0xde00}
	}
	if kind == "malformed" {
		pattern = []uint16{'A', 0xd800, 'B', 0xdc00, 0xd801, 0xdc00}
	}
	units := make([]uint16, n)
	for i := range units {
		units[i] = pattern[i%len(pattern)]
	}
	return NewJavaStringUTF16(units)
}
func BenchmarkUTF8Owned181GetBytes(b *testing.B) {
	for _, kind := range []string{"ascii", "unicode", "malformed"} {
		for _, n := range []int{0, 1, 17, 128, 4096, 65536} {
			b.Run(fmt.Sprintf("%s/%d", kind, n), func(b *testing.B) {
				text := utf8Owned181Input(kind, n)
				b.ReportAllocs()
				b.SetBytes(int64(len(JavaStringGetBytes(text, UTF_8).Elements)))
				for b.Loop() {
					utf8Owned181ArraySink = JavaStringGetBytes(text, UTF_8)
				}
			})
		}
	}
}
func BenchmarkUTF8Owned181CharsetControls(b *testing.B) {
	text := utf8Owned181Input("unicode", 128)
	for _, charset := range []*Charset{UTF_8, US_ASCII, ISO_8859_1, UTF_16BE, UTF_16LE, UTF_16} {
		b.Run(charset.Name(), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(JavaStringGetBytes(text, charset).Elements)))
			for b.Loop() {
				utf8Owned181ArraySink = JavaStringGetBytes(text, charset)
			}
		})
	}
}
func BenchmarkUTF8Owned181DigestPipeline(b *testing.B) {
	for _, n := range []int{17, 4096} {
		text := utf8Owned181Input("unicode", n)
		b.Run(fmt.Sprintf("fresh-digest/%d", n), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(JavaStringGetBytes(text, UTF_8).Elements)))
			for b.Loop() {
				digest := MessageDigestGetInstance("SHA-256")
				utf8Owned181ArraySink = digest.Digest(JavaStringGetBytes(text, UTF_8))
			}
		})
		b.Run(fmt.Sprintf("reused-digest/%d", n), func(b *testing.B) {
			digest := MessageDigestGetInstance("SHA-256")
			b.ReportAllocs()
			b.SetBytes(int64(len(JavaStringGetBytes(text, UTF_8).Elements)))
			for b.Loop() {
				utf8Owned181ArraySink = digest.Digest(JavaStringGetBytes(text, UTF_8))
			}
		})
	}
}
