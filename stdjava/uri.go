package stdjava

import (
	"unicode/utf16"

	"github.com/NickyBoy89/java2go/stdjava/internal/normalization15"
)

// JavaURI retains the String constructor's original immutable UTF16 reference.
// Its parsed components are nominal URI state, never Go URL normalization.
type JavaURI struct {
	text                                                     *JavaString
	scheme, authority, userInfo, host, path, query, fragment *JavaString
	port                                                     int32
}

func init()                                { RegisterJavaType("java.net.URI", ObjectTypeID, SerializableTypeID, ComparableTypeID) }
func (*JavaURI) JavaDynamicTypeID() TypeID { return "java.net.URI" }

func NewJavaURIStringExecution(_ *Execution, text *JavaString) *JavaURI {
	if text == nil {
		panic(NewJavaNullPointerExceptionMessage(uriRuntimeText(`Cannot invoke "String.length()" because "this.input" is null`)))
	}
	value := &JavaURI{text: text, port: -1}
	parser := javaURIParser{input: text, value: value}
	parser.parse()
	return value
}

func requireJavaURI(value *JavaURI) *JavaURI {
	if value == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return value
}

func (value *JavaURI) StringJava2goExecution(_ *Execution) *JavaString {
	return requireJavaURI(value).text
}

func (value *JavaURI) ToASCIIStringJava2goExecution(_ *Execution) *JavaString {
	text := requireJavaURI(value).text
	nonASCII := false
	for _, unit := range text.units {
		if unit >= 128 {
			nonASCII = true
			break
		}
	}
	if !nonASCII {
		return text
	}
	units, _ := normalizeJavaUTF16(text.units, normalization15.NFC)
	for i := 0; i < len(units); i++ {
		unit := units[i]
		if unit >= 0xd800 && unit <= 0xdbff && i+1 < len(units) && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			i++
			continue
		}
		if unit >= 0xd800 && unit <= 0xdfff {
			// URI.encode uses a reporting UTF8 encoder. With JDK assertions off,
			// its caught coding failure reaches the null ByteBuffer dereference.
			panic(NewJavaNullPointerExceptionMessage(uriRuntimeText(`Cannot invoke "java.nio.ByteBuffer.hasRemaining()" because "bb" is null`)))
		}
	}
	encoded := []byte(string(utf16.Decode(units)))
	out := make([]uint16, 0, len(encoded))
	const hex = "0123456789ABCDEF"
	for _, b := range encoded {
		if b < 128 {
			out = append(out, uint16(b))
		} else {
			out = append(out, '%', uint16(hex[b>>4]), uint16(hex[b&15]))
		}
	}
	return NewJavaStringUTF16(out)
}

// Only runtime-authored ASCII diagnostics use this ingress. Application and
// syntax-exception input Strings retain their original UTF16 references.
func uriRuntimeText(text string) *JavaString {
	units := make([]uint16, len(text))
	for i := range text {
		units[i] = uint16(text[i])
	}
	return NewJavaStringUTF16(units)
}
