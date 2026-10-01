package stdjava

import (
	"math"
	"reflect"
	"testing"
)

func TestCampaignScalarTextReferenceNative(t *testing.T) {
	text := JavaStringFromHostUTF8
	if got := JavaDoubleParseDouble(text(" 0x1.8p1D ")); got != 3 {
		t.Fatal(got)
	}
	if got := JavaFloatParseFloat(text("-0.0")); math.Float32bits(got) != 0x80000000 {
		t.Fatal(got)
	}
	if got := JavaByteParseByte(text("-80"), 16); got != -128 {
		t.Fatal(got)
	}
	if got := JavaShortParseShort(text("\uff11\uff12\uff13")); got != 123 {
		t.Fatal(got)
	}
	if !JavaBooleanParseBoolean(text("tRuE")) || JavaBooleanParseBoolean(nil) {
		t.Fatal("Boolean boundary")
	}
	message := NewJavaStringUTF16([]uint16{'x', 0xd800, 0, 0xdc00})
	value := NewJavaArithmeticExceptionMessage(message)
	if JavaThrowableMessageDefault(value) != message || !reflect.DeepEqual(JavaThrowableMessageDefault(value).UTF16Copy(), message.UTF16Copy()) {
		t.Fatal("lost immutable UTF16 reference")
	}
	if JavaThrowableMessageDefault(NewJavaArithmeticExceptionMessage(nil)) != nil {
		t.Fatal("null message changed")
	}
	if NewArithmeticException("legacy").Message() != "legacy" || ParseDouble("1.25") != 1.25 || ParseFloat("2.5") != 2.5 || ParseByte("12") != 12 || ParseShort("123") != 123 || !ParseBoolean("TRUE") {
		t.Fatal("host ABI changed")
	}
	if JavaIntegerParseInt(text("-2147483648")) != -2147483648 || JavaLongParseLong(text("-8000000000000000"), 16) != -9223372036854775808 {
		t.Fatal("existing canonical integer parsers changed")
	}
}
