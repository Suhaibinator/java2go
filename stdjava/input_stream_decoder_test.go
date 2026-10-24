package stdjava

import (
	"bytes"
	"io"
	"testing"

	"golang.org/x/text/transform"
)

type decoderChunkReader struct {
	data []byte
	size int
}

func (r *decoderChunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := min(len(p), r.size, len(r.data))
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestInputStreamDecoderMalformedUnitsAndChunkBoundaries(t *testing.T) {
	cases := []struct {
		name    string
		charset *Charset
		input   []byte
		want    string
	}{
		{"utf8-valid", UTF_8, []byte("Aé😀"), "Aé😀"},
		{"utf8-surrogate", UTF_8, []byte{0xed, 0xa0, 0x80}, "�"},
		{"utf8-truncated-surrogate", UTF_8, []byte{0xed, 0xa0}, "�"},
		{"utf8-truncated", UTF_8, []byte{0xe2, 0x82}, "�"},
		{"utf8-bad-continuation", UTF_8, []byte{0xe2, 0x82, 65}, "�A"},
		{"utf8-bad-second", UTF_8, []byte{0xe2, 65, 0x82}, "�A�"},
		{"utf8-overlong-two", UTF_8, []byte{0xc0, 0x80}, "��"},
		{"utf8-overlong-three", UTF_8, []byte{0xe0, 0x80, 0x80}, "���"},
		{"utf8-overlong-four", UTF_8, []byte{0xf0, 0x80, 0x80, 0x80}, "����"},
		{"utf8-outside-range", UTF_8, []byte{0xf4, 0x90, 0x80, 0x80}, "����"},
		{"utf8-bad-fourth", UTF_8, []byte{0xf0, 0x90, 0x80, 65}, "�A"},
		{"utf8-truncated-four", UTF_8, []byte{0xf0, 0x90, 0x80}, "�"},
		{"utf16-high-ascii", UTF_16BE, []byte{0xd8, 0, 0, 65}, "�"},
		{"utf16-high-high-low", UTF_16BE, []byte{0xd8, 0, 0xd8, 0, 0xdc, 0}, "��"},
		{"utf16-high-truncated", UTF_16BE, []byte{0xd8, 0, 0}, "�"},
		{"utf16-low-ascii", UTF_16BE, []byte{0xdc, 0, 0, 65}, "�A"},
		{"utf16-pair", UTF_16BE, []byte{0xd8, 0x3d, 0xde, 0, 0, 65}, "😀A"},
		{"utf16-le-high-ascii", UTF_16LE, []byte{0, 0xd8, 65, 0}, "�"},
		{"utf16-bom-little", UTF_16, []byte{0xff, 0xfe, 65, 0}, "A"},
		{"utf16-bom-big", UTF_16, []byte{0xfe, 0xff, 0, 65}, "A"},
		{"utf16-default-big", UTF_16, []byte{0, 65}, "A"},
		{"utf16-only-bom", UTF_16, []byte{0xfe, 0xff}, ""},
		{"utf16-explicit-keeps-bom", UTF_16BE, []byte{0xfe, 0xff, 0, 65}, "\ufeffA"},
		{"ascii", US_ASCII, []byte{65, 0xff}, "A�"},
		{"latin1", ISO_8859_1, []byte{65, 0xff}, "Aÿ"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decoder := newInputStreamDecoder(tc.charset)
			for chunk := 1; chunk <= len(tc.input); chunk++ {
				decoder.Reset()
				reader := transform.NewReader(&decoderChunkReader{data: bytes.Clone(tc.input), size: chunk}, decoder)
				var got bytes.Buffer
				// A single-byte output buffer also exercises partial UTF-8 delivery.
				var one [1]byte
				for {
					n, err := reader.Read(one[:])
					got.Write(one[:n])
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if got.String() != tc.want {
					t.Fatalf("chunk %d: got %q want %q", chunk, got.String(), tc.want)
				}
			}
		})
	}
}

func TestInputStreamDecoderDestinationBackpressure(t *testing.T) {
	for _, charset := range []*Charset{UTF_8, UTF_16BE, UTF_16LE, UTF_16} {
		input := StringGetBytes("😀", charset).Elements
		data := make([]byte, len(input))
		for i, v := range input {
			data[i] = byte(v)
		}
		decoder := newInputStreamDecoder(charset)
		nDst, nSrc, err := decoder.Transform(make([]byte, 3), data, true)
		if nDst != 0 || err != transform.ErrShortDst {
			t.Fatalf("%s: %d,%d,%v", charset.Name(), nDst, nSrc, err)
		}
		out := make([]byte, 4)
		nDst, _, err = decoder.Transform(out, data[nSrc:], true)
		if err != nil || string(out[:nDst]) != "😀" {
			t.Fatalf("%s retry: %q %v", charset.Name(), out[:nDst], err)
		}
	}
}
