package stdjava

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/transform"
)

// Charset implements the six mandatory standard charsets used by Java byte
// string conversion. Additional installed encodings are not modeled.
type Charset struct{ name string }

var (
	US_ASCII   = &Charset{"US-ASCII"}
	ISO_8859_1 = &Charset{"ISO-8859-1"}
	UTF_8      = &Charset{"UTF-8"}
	UTF_16BE   = &Charset{"UTF-16BE"}
	UTF_16LE   = &Charset{"UTF-16LE"}
	UTF_16     = &Charset{"UTF-16"}
)

func (c *Charset) Name() string            { return c.name }
func (c *Charset) String() string          { return c.name }
func (*Charset) JavaDynamicTypeID() TypeID { return "Charset" }
func CharsetForName(name string) *Charset {
	StringRequireNonNull(name)
	switch strings.ToUpper(strings.ReplaceAll(name, "_", "-")) {
	case "UTF-8", "UTF8":
		return UTF_8
	case "US-ASCII", "ASCII", "ISO646-US":
		return US_ASCII
	case "ISO-8859-1", "ISO8859-1", "LATIN1", "L1":
		return ISO_8859_1
	case "UTF-16", "UTF16":
		return UTF_16
	case "UTF-16BE", "UTF16BE", "UNICODEBIGUNMARKED":
		return UTF_16BE
	case "UTF-16LE", "UTF16LE", "UNICODELITTLEUNMARKED":
		return UTF_16LE
	}
	panic(newThrowableBase("UnsupportedCharsetException", name))
}
func init() {
	RegisterJavaType("Charset", ObjectTypeID)
	RegisterException("UnsupportedCharsetException", "IllegalArgumentException")
	RegisterException("UnsupportedEncodingException", "IOException")
}

func StringGetBytes(value string, charsets ...any) *PrimitiveArray[int8] {
	StringRequireNonNull(value)
	charset := UTF_8
	if len(charsets) != 0 {
		ReferenceRequireNonNull(charsets[0])
		switch selected := charsets[0].(type) {
		case *Charset:
			charset = selected
		case string:
			charset = charsetForEncodingName(selected)
		default:
			panic(NewIllegalArgumentException("getBytes requires Charset or charset name"))
		}
	}
	ReferenceRequireNonNull(charset)
	var data []byte
	switch charset {
	case UTF_8:
		data = []byte(value)
	case US_ASCII, ISO_8859_1:
		max := rune(127)
		if charset == ISO_8859_1 {
			max = 255
		}
		for _, r := range value {
			if r > max {
				data = append(data, '?')
			} else {
				data = append(data, byte(r))
			}
		}
	case UTF_16, UTF_16BE, UTF_16LE:
		var order binary.ByteOrder = binary.BigEndian
		if charset == UTF_16LE {
			order = binary.LittleEndian
		}
		if charset == UTF_16 && len(value) != 0 {
			data = append(data, 0xfe, 0xff)
		}
		for _, unit := range StringChars(value) {
			var encoded [2]byte
			order.PutUint16(encoded[:], uint16(unit))
			data = append(data, encoded[:]...)
		}
	default:
		panic(NewUnsupportedOperationException("charset encoding is not supported"))
	}
	return signedByteArray(data)
}

// StringFromChars preserves complete surrogate pairs. Isolated surrogate units
// cannot yet be represented by the runtime's UTF-8 String ABI; fail explicitly
// rather than silently replacing legal Java char[] contents.
func StringFromChars(units []rune) string {
	out := make([]rune, 0, len(units))
	for i := 0; i < len(units); i++ {
		unit := units[i]
		if utf16.IsSurrogate(unit) {
			if unit >= 0xdc00 || i+1 == len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
				panic(NewUnsupportedOperationException("String storage does not yet preserve isolated UTF-16 surrogates"))
			}
			out = append(out, utf16.DecodeRune(unit, units[i+1]))
			i++
		} else {
			out = append(out, unit)
		}
	}
	return string(out)
}

func StringToCharArray(value string) *PrimitiveArray[rune] {
	StringRequireNonNull(value)
	return PrimitiveArrayLiteral(PrimitiveCharTypeID, StringChars(value)...)
}

// StringNew covers String(), String(String), String(char[]), String(byte[]),
// and String(byte[], Charset). It copies the source array as Java requires.
func StringNew(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	ReferenceRequireNonNull(args[0])
	switch value := args[0].(type) {
	case string:
		return StringRequireNonNull(value)
	case *PrimitiveArray[rune]:
		return StringFromChars(value.Elements)
	case *PrimitiveArray[int8]:
		charset := UTF_8
		if len(args) > 1 {
			ReferenceRequireNonNull(args[1])
			switch selected := args[1].(type) {
			case *Charset:
				charset = selected
			case string:
				charset = charsetForEncodingName(selected)
			default:
				panic(NewIllegalArgumentException("String byte constructor requires Charset"))
			}
		}
		return decodeCharset(unsignedBytes(value.Elements), charset)
	}
	panic(NewUnsupportedOperationException("String constructor overload is not supported"))
}

// Byte constructors and InputStreamReader use the same replacement policy for
// malformed encoded input. This does not apply to String(char[]), whose legal
// isolated UTF-16 surrogates must not be replaced by an encoding decoder.
func decodeCharset(data []byte, charset *Charset) string {
	decoded, _, err := transform.Bytes(newInputStreamDecoder(charset), data)
	if err != nil {
		// All supported decoders replace malformed input and consume EOF tails.
		// An error here therefore indicates a decoder implementation defect.
		panic(err)
	}
	return string(decoded)
}

// The legacy String overloads translate unavailable encoding names to the
// checked IOException subtype, unlike Charset.forName's unchecked exception.
func charsetForEncodingName(name string) (charset *Charset) {
	defer func() {
		if failure := recover(); failure != nil {
			if CaughtAs(failure, "UnsupportedCharsetException") {
				panic(newThrowableBase("UnsupportedEncodingException", name))
			}
			panic(failure)
		}
	}()
	return CharsetForName(name)
}
