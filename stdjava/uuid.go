package stdjava

import (
	"crypto/md5"
	"encoding/binary"
	"strconv"
)

// JavaUUID is the immutable value and identity of canonical java.util.UUID.
// Its nominal Comparable relation is erased; it does not declare generic
// reflection metadata or an exact-self comparable descriptor.
type JavaUUID struct{ most, least int64 }

func init()                                   { RegisterJavaType("java.util.UUID", ObjectTypeID, SerializableTypeID, ComparableTypeID) }
func (*JavaUUID) JavaDynamicTypeID() TypeID   { return "java.util.UUID" }
func NewJavaUUID(most, least int64) *JavaUUID { return &JavaUUID{most: most, least: least} }

// uuidASCII creates runtime-authored diagnostic text only. Application input
// and exception suffixes retain their original UTF16 units throughout parsing.
func uuidASCII(text string) *JavaString {
	units := make([]uint16, len(text))
	for i := range text {
		units[i] = uint16(text[i])
	}
	return &JavaString{units: units}
}
func requireJavaUUID(value *JavaUUID) *JavaUUID {
	if value == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return value
}
func UUIDFromStringJavaString(text *JavaString) *JavaUUID {
	if text == nil {
		panic(NewJavaNullPointerExceptionMessage(uuidASCII(`Cannot invoke "String.length()" because "name" is null`)))
	}
	if len(text.units) > 36 {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{'U', 'U', 'I', 'D', ' ', 's', 't', 'r', 'i', 'n', 'g', ' ', 't', 'o', 'o', ' ', 'l', 'a', 'r', 'g', 'e'})))
	}
	var dashes [4]int
	count := 0
	for i, c := range text.units {
		if c == '-' {
			if count == len(dashes) {
				panic(NewJavaIllegalArgumentExceptionMessage(ConcatJavaStrings(uuidASCII("Invalid UUID string: "), text)))
			}
			dashes[count] = i
			count++
		}
	}
	if count != len(dashes) {
		panic(NewJavaIllegalArgumentExceptionMessage(ConcatJavaStrings(uuidASCII("Invalid UUID string: "), text)))
	}
	var parts [5]uint64
	start := 0
	for i := range parts {
		end := len(text.units)
		if i < len(dashes) {
			end = dashes[i]
		}
		parts[i] = uuidParseHexSegment(text.units[start:end])
		start = end + 1
	}
	most := (parts[0]&0xffffffff)<<32 | (parts[1]&0xffff)<<16 | parts[2]&0xffff
	least := (parts[3]&0xffff)<<48 | parts[4]&0xffffffffffff
	return NewJavaUUID(int64(most), int64(least))
}

// The JDK's short-form parser reads positive signed-long segments before
// masking the fields. It accepts a leading plus and BMP Character.digit hex
// forms, and reports the failing index relative to the individual segment.
func uuidParseHexSegment(units []uint16) uint64 {
	if len(units) == 0 {
		panic(NewJavaNumberFormatException(JavaStringLiteralUTF16(nil)))
	}
	index := 0
	if units[0] == '+' {
		index++
	}
	fail := func(i int) {
		prefix := uuidASCII("Error at index " + strconv.Itoa(i) + ` in: "`)
		message := ConcatJavaStrings(prefix, &JavaString{units: units})
		message = ConcatJavaStrings(message, uuidASCII(`"`))
		panic(NewJavaNumberFormatException(message))
	}
	if index == len(units) {
		fail(index)
	}
	const maximum = uint64(1<<63 - 1)
	var value uint64
	for ; index < len(units); index++ {
		digit := javaIntegerDigit21(units[index])
		if digit < 0 || digit >= 16 || value > maximum/16 || value == maximum/16 && uint64(digit) > maximum%16 {
			fail(index)
		}
		value = value*16 + uint64(digit)
	}
	return value
}
func UUIDNameFromBytes(name *PrimitiveArray[int8]) *JavaUUID {
	if name == nil {
		panic(NewJavaNullPointerExceptionMessage(uuidASCII(`Cannot read the array length because "input" is null`)))
	}
	bytes := make([]byte, len(name.Elements))
	for i, v := range name.Elements {
		bytes[i] = byte(v)
	}
	digest := md5.Sum(bytes)
	digest[6] = (digest[6] & 15) | 0x30
	digest[8] = (digest[8] & 63) | 0x80
	return NewJavaUUID(int64(binary.BigEndian.Uint64(digest[:8])), int64(binary.BigEndian.Uint64(digest[8:])))
}
func (value *JavaUUID) GetMostSignificantBits() int64  { return requireJavaUUID(value).most }
func (value *JavaUUID) GetLeastSignificantBits() int64 { return requireJavaUUID(value).least }
func (value *JavaUUID) Equals(other any) bool {
	requireJavaUUID(value)
	right, ok := other.(*JavaUUID)
	return ok && right != nil && value.most == right.most && value.least == right.least
}
func (value *JavaUUID) HashCode() int32 {
	requireJavaUUID(value)
	bits := uint64(value.most) ^ uint64(value.least)
	return int32(bits ^ (bits >> 32))
}
func (value *JavaUUID) CompareTo(other *JavaUUID) int32 {
	requireJavaUUID(value)
	if other == nil {
		panic(NewJavaNullPointerExceptionMessage(uuidASCII(`Cannot read field "mostSigBits" because "val" is null`)))
	}
	if value.most < other.most {
		return -1
	}
	if value.most > other.most {
		return 1
	}
	if value.least < other.least {
		return -1
	}
	if value.least > other.least {
		return 1
	}
	return 0
}
func (value *JavaUUID) EqualsJava2goExecution(_ *Execution, other any) bool {
	return value.Equals(other)
}
func (value *JavaUUID) HashCodeJava2goExecution(_ *Execution) int32 { return value.HashCode() }
func (value *JavaUUID) CompareToJava2goExecution(_ *Execution, other *JavaUUID) int32 {
	return value.CompareTo(other)
}
func (value *JavaUUID) StringJava2goExecution(_ *Execution) *JavaString {
	requireJavaUUID(value)
	var bytes [16]byte
	binary.BigEndian.PutUint64(bytes[:8], uint64(value.most))
	binary.BigEndian.PutUint64(bytes[8:], uint64(value.least))
	const hex = "0123456789abcdef"
	units := make([]uint16, 0, 36)
	for i, b := range bytes {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			units = append(units, '-')
		}
		units = append(units, uint16(hex[b>>4]), uint16(hex[b&15]))
	}
	return &JavaString{units: units}
}
