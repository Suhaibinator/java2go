package stdjava

import (
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"
)

// JavaStringGetBytes is the reference-bearing String encoding boundary. Legal
// isolated UTF16 units remain unchanged in the String; only the encoded output
// replaces them according to the selected mandatory Java charset's policy.
// The compiler supplies the default charset for overloads without a parameter.
func JavaStringGetBytes(value *JavaString, charset *Charset) *PrimitiveArray[int8] {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(charset)
	var encoded []byte
	var order binary.AppendByteOrder
	switch charset {
	case UTF_8, US_ASCII, ISO_8859_1:
	case UTF_16, UTF_16BE:
		order = binary.BigEndian
		if charset == UTF_16 && len(value.units) != 0 {
			encoded = append(encoded, 0xfe, 0xff)
		}
	case UTF_16LE:
		order = binary.LittleEndian
	default:
		panic(NewUnsupportedOperationException("charset encoding is not supported"))
	}
	for index := 0; index < len(value.units); index++ {
		unit := value.units[index]
		point := rune(unit)
		malformed := false
		if unit >= 0xd800 && unit <= 0xdbff {
			if index+1 < len(value.units) && value.units[index+1] >= 0xdc00 && value.units[index+1] <= 0xdfff {
				point = utf16.DecodeRune(point, rune(value.units[index+1]))
				index++
			} else {
				malformed = true
			}
		} else if unit >= 0xdc00 && unit <= 0xdfff {
			malformed = true
		}
		if order != nil {
			// UTF16 encoders replace each malformed unit with U+FFFD. Valid
			// supplementary scalars emit their original pair in byte order.
			if malformed {
				point = utf8.RuneError
			}
			if point > 0xffff {
				high, low := utf16.EncodeRune(point)
				encoded = order.AppendUint16(encoded, uint16(high))
				encoded = order.AppendUint16(encoded, uint16(low))
			} else {
				encoded = order.AppendUint16(encoded, uint16(point))
			}
		} else if charset == UTF_8 {
			if malformed {
				encoded = append(encoded, '?')
			} else {
				encoded = utf8.AppendRune(encoded, point)
			}
		} else {
			maximum := rune(127)
			if charset == ISO_8859_1 {
				maximum = 255
			}
			if malformed || point > maximum {
				// A valid but unmappable supplementary pair is one input
				// character and therefore contributes one replacement byte.
				encoded = append(encoded, '?')
			} else {
				encoded = append(encoded, byte(point))
			}
		}
	}
	return signedByteArray(encoded)
}

// JavaStringFromBytes decodes external bytes with the existing JVM-tested
// malformed-input replacement policy. Decoded scalars become immutable UTF16
// units in a fresh String allocation, including an empty decoded result.
// This byte boundary must not be used to construct String from legal char[].
func JavaStringFromBytes(data *PrimitiveArray[int8], charset *Charset) *JavaString {
	ReferenceRequireNonNull(data)
	ReferenceRequireNonNull(charset)
	decoded := decodeCharset(unsignedBytes(data.Elements), charset)
	return NewJavaStringUTF16(utf16.Encode([]rune(decoded)))
}
