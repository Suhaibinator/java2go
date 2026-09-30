package stdjava

import (
	"slices"
	"testing"
)

func charsetEncodingReferenceFailure(t *testing.T, name string, action func() any) (failure any) {
	t.Helper()
	defer func() {
		failure = recover()
		if !CaughtAs(failure, name) {
			t.Errorf("expected %s, got %#v", name, failure)
		}
	}()
	got := action()
	t.Fatalf("expected %s, returned %T", name, got)
	return nil
}

func TestCharsetEncodingReferenceAliasesAndNativeBoundary(t *testing.T) {
	cases := []struct {
		name string
		want *Charset
	}{{"UTF8", UTF_8}, {"unicode-1-1-utf-8", UTF_8}, {"UTF_16", UTF_16}, {"UnicodeBig", UTF_16}, {"ISO-10646-UCS-2", UTF_16BE}, {"UnicodeLittleUnmarked", UTF_16LE}, {"ISO_8859-1:1987", ISO_8859_1}, {"819", ISO_8859_1}, {"ANSI_X3.4-1968", US_ASCII}, {"646", US_ASCII}}
	for _, c := range cases {
		name := NewJavaStringUTF16(charsetEncodingASCIIUnits(c.name))
		if got := CharsetForNameReference(name); got != c.want {
			t.Fatalf("alias %q changed singleton", c.name)
		}
	}
	var nativeName string = UTF_8.Name()
	if nativeName != "UTF-8" || CharsetForName("UTF8") != UTF_8 || UTF_8.String() != "UTF-8" {
		t.Fatal("native Charset APIs changed")
	}
}

func TestCharsetEncodingReferenceExceptionAndNullOrder(t *testing.T) {
	nilFailure := charsetEncodingReferenceFailure(t, "IllegalArgumentException", func() any { return CharsetForNameReference(nil) })
	if message := JavaThrowableMessageDefault(nilFailure); message == nil || !slices.Equal(message.UTF16Copy(), charsetEncodingASCIIUnits("Null charset name")) {
		t.Fatal("lookup null contract lost JDK message")
	}
	illegal := NewJavaStringUTF16([]uint16{'A', 0xd800})
	missing := NewJavaStringUTF16(charsetEncodingASCIIUnits("x-java2go-no-such-encoding-467923"))
	value := NewJavaStringUTF16([]uint16{'A'})
	bytes := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 65)
	failure := charsetEncodingReferenceFailure(t, "IllegalCharsetNameException", func() any { return CharsetForNameReference(illegal) })
	if JavaThrowableMessageDefault(failure) != illegal {
		t.Fatal("illegal charset exception rendered or copied name")
	}
	for _, name := range []*JavaString{illegal, missing} {
		a := charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", func() any { return JavaStringGetBytesNamed(value, name) })
		b := charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", func() any { return JavaStringFromBytesNamed(bytes, name) })
		if JavaThrowableMessageDefault(a) != name || JavaThrowableMessageDefault(b) != name {
			t.Fatal("checked encoding exception changed original name reference")
		}
	}
	charsetEncodingReferenceFailure(t, "UnsupportedCharsetException", func() any { return CharsetForNameReference(missing) })
	charsetEncodingReferenceFailure(t, "UnsupportedCharsetException", func() any { return CharsetForNameReference(NewJavaStringUTF16(charsetEncodingASCIIUnits("UTF_8"))) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(nil, missing) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any { return JavaStringGetBytesNamed(value, nil) })
	charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", func() any { return JavaStringFromBytesNamed(nil, missing) })
	charsetEncodingReferenceFailure(t, "NullPointerException", func() any {
		return JavaStringFromBytesNamed(nil, NewJavaStringUTF16(charsetEncodingASCIIUnits("UTF8")))
	})
	charsetEncodingReferenceFailure(t, "UnsupportedEncodingException", func() any { return JavaStringFromBytesRangeNamed(bytes, -1, 2, missing) })
}

func TestCharsetEncodingReferenceCopiesAndRangeBounds(t *testing.T) {
	name := NewJavaStringUTF16(charsetEncodingASCIIUnits("UTF8"))
	data := PrimitiveArrayLiteral[int8](PrimitiveByteTypeID, 90, 65, -61, -87, 66, 90)
	a, b := JavaStringFromBytesRangeNamed(data, 1, 4, name), JavaStringFromBytesRange(data, 1, 4, UTF_8)
	if a == b || !slices.Equal(a.UTF16Copy(), []uint16{'A', 0xe9, 'B'}) || !a.Equals(b) {
		t.Fatal("byte range decoding lost value or fresh allocation")
	}
	data.Elements[1] = 88
	output := JavaStringGetBytesNamed(a, name)
	output.Elements[0] = 89
	if a.CharAt(0) != 'A' || JavaStringGetBytesNamed(a, name).Elements[0] != 65 {
		t.Fatal("byte boundary aliases mutable input/output")
	}
	emptyA, emptyB := JavaStringFromBytesNamed(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID), name), JavaStringFromBytesNamed(PrimitiveArrayLiteral[int8](PrimitiveByteTypeID), name)
	if emptyA == emptyB || emptyA.Length() != 0 || emptyB.Length() != 0 {
		t.Fatal("empty decode must allocate fresh wrapper")
	}
	for _, bounds := range [][2]int32{{-1, 1}, {1, -1}, {5, 2}, {2147483647, 2147483647}} {
		charsetEncodingReferenceFailure(t, "StringIndexOutOfBoundsException", func() any { return JavaStringFromBytesRangeNamed(data, bounds[0], bounds[1], name) })
	}
	if JavaStringFromBytesRangeNamed(data, 6, 0, name).Length() != 0 {
		t.Fatal("end empty range rejected")
	}
}

// These installed JDK21 codecs have no modeled encoder/decoder. They must
// remain explicit runtime blockers, not false Java UnsupportedCharset errors.
func TestCharsetEncodingReferenceInstalledBoundary(t *testing.T) {
	for _, name := range []string{"windows-1252", "Cp1252", "UTF-32", "UTF_32", "Shift_JIS", "UnicodeLittle"} {
		text := NewJavaStringUTF16(charsetEncodingASCIIUnits(name))
		a := charsetEncodingReferenceFailure(t, "UnsupportedOperationException", func() any { return CharsetForNameReference(text) })
		b := charsetEncodingReferenceFailure(t, "UnsupportedOperationException", func() any { return JavaStringGetBytesNamed(NewJavaStringUTF16([]uint16{'A'}), text) })
		if CaughtAs(a, "UnsupportedCharsetException") || CaughtAs(b, "UnsupportedEncodingException") {
			t.Fatal("installed unimplemented codec mislabeled as unavailable")
		}
	}
}

func charsetEncodingASCIIUnits(value string) []uint16 {
	units := make([]uint16, len(value))
	for i := range units {
		units[i] = uint16(value[i])
	}
	return units
}
