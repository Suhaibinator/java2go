package stdjava

import (
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// newInputStreamDecoder keeps incomplete encoded units in transform.Reader's
// input buffer. Java groups malformed units differently from Go's UTF decoders.
func newInputStreamDecoder(charset *Charset) transform.Transformer {
	switch charset {
	case UTF_8:
		return &javaUTF8Decoder{}
	case UTF_16, UTF_16BE, UTF_16LE:
		d := &javaUTF16Decoder{charset: charset}
		d.Reset()
		return d
	case ISO_8859_1:
		return charmap.ISO8859_1.NewDecoder()
	case US_ASCII:
		return asciiReaderDecoder{}
	default:
		panic(NewUnsupportedOperationException("InputStreamReader charset"))
	}
}

type javaUTF8Decoder struct{}

func (*javaUTF8Decoder) Reset() {}
func (*javaUTF8Decoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		r, size, incomplete := javaUTF8Unit(src[nSrc:], atEOF)
		if incomplete {
			return nDst, nSrc, transform.ErrShortSrc
		}
		if len(dst)-nDst < utf8.RuneLen(r) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc += size
	}
	return
}

// A malformed continuation consumes the valid prefix; a UTF-8 encoded surrogate
// consumes its entire three-byte unit, as CharsetDecoder does on the JVM.
func javaUTF8Unit(src []byte, atEOF bool) (rune, int, bool) {
	b := src[0]
	if b < 0x80 {
		return rune(b), 1, false
	}
	width := 0
	switch {
	case b >= 0xc2 && b <= 0xdf:
		width = 2
	case b >= 0xe0 && b <= 0xef:
		width = 3
	case b >= 0xf0 && b <= 0xf4:
		width = 4
	default:
		return utf8.RuneError, 1, false
	}
	for i := 1; i < width; i++ {
		if i >= len(src) {
			if atEOF {
				return utf8.RuneError, len(src), false
			}
			return 0, 0, true
		}
		c := src[i]
		if c < 0x80 || c > 0xbf || (i == 1 && ((b == 0xe0 && c < 0xa0) || (b == 0xf0 && c < 0x90) || (b == 0xf4 && c > 0x8f))) {
			return utf8.RuneError, i, false
		}
	}
	if b == 0xed && src[1] >= 0xa0 {
		return utf8.RuneError, 3, false
	}
	r, _ := utf8.DecodeRune(src[:width])
	return r, width, false
}

type javaUTF16Decoder struct {
	charset *Charset
	order   binary.ByteOrder
	initial bool
}

func (d *javaUTF16Decoder) Reset() {
	d.order = binary.BigEndian
	if d.charset == UTF_16LE {
		d.order = binary.LittleEndian
	}
	d.initial = d.charset == UTF_16
}
func (d *javaUTF16Decoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for nSrc < len(src) {
		remaining := src[nSrc:]
		r, size := rune(utf8.RuneError), 1
		if len(remaining) < 2 {
			if !atEOF {
				return nDst, nSrc, transform.ErrShortSrc
			}
		} else {
			if d.initial {
				d.initial = false
				mark := binary.BigEndian.Uint16(remaining)
				if mark == 0xfeff || mark == 0xfffe {
					if mark == 0xfffe {
						d.order = binary.LittleEndian
					}
					nSrc += 2
					continue
				}
			}
			unit := d.order.Uint16(remaining)
			size = 2
			switch {
			case unit >= 0xd800 && unit <= 0xdbff:
				if len(remaining) < 4 {
					if !atEOF {
						return nDst, nSrc, transform.ErrShortSrc
					}
					size = len(remaining)
				} else {
					next := d.order.Uint16(remaining[2:])
					size = 4
					if next >= 0xdc00 && next <= 0xdfff {
						r = utf16.DecodeRune(rune(unit), rune(next))
					}
				}
			case unit >= 0xdc00 && unit <= 0xdfff:
			default:
				r = rune(unit)
			}
		}
		if len(dst)-nDst < utf8.RuneLen(r) {
			return nDst, nSrc, transform.ErrShortDst
		}
		nDst += utf8.EncodeRune(dst[nDst:], r)
		nSrc += size
	}
	return
}
