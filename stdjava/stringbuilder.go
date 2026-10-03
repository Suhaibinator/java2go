package stdjava

import "fmt"

// StringBuilder stores UTF-16 code units, including isolated surrogates while
// the builder is being edited. String conversion retains the runtime's explicit
// limitation for isolated surrogates. StringBuffer synchronization is not yet
// modeled by this shared representation.
type StringBuilder struct {
	buf []rune
}

// NewStringBuilder returns an empty StringBuilder, matching `new StringBuilder()`.
func NewStringBuilder() *StringBuilder {
	return &StringBuilder{}
}

// NewStringBuilderCapacity implements the int constructor. Capacity is a
// storage reservation, not initial text or length; growth/capacity observation
// is not modeled by this runtime surface.
func NewStringBuilderCapacity(capacity int32) *StringBuilder {
	if capacity < 0 {
		panic(NewNegativeArraySizeException(fmt.Sprint(capacity)))
	}
	return &StringBuilder{buf: make([]rune, 0, int(capacity))}
}

// NewStringBuilderString returns a StringBuilder seeded with the given string,
// matching `new StringBuilder(String)`.
func NewStringBuilderString(s string) *StringBuilder {
	return &StringBuilder{buf: StringChars(StringRequireNonNull(s))}
}

// Append appends the textual representation of value to the builder and returns
// the builder for chaining, matching StringBuilder.append. The value is
// formatted with the same rules Java applies for the common overloads: strings
// and runes append directly, everything else uses its default string form.
func (b *StringBuilder) Append(value any) *StringBuilder {
	switch v := value.(type) {
	case *JavaString:
		return b.AppendJavaString(v)
	case string:
		b.buf = append(b.buf, StringChars(v)...)
	case rune:
		b.buf = append(b.buf, v)
	case []rune:
		b.buf = append(b.buf, v...)
	case bool:
		b.buf = append(b.buf, StringChars(fmt.Sprintf("%t", v))...)
	default:
		b.buf = append(b.buf, StringChars(fmt.Sprintf("%v", v))...)
	}
	return b
}

// Insert inserts the textual representation of value at the given UTF-16 offset,
// matching StringBuilder.insert.
func (b *StringBuilder) Insert(offset int32, value any) *StringBuilder {
	b.checkOffset(offset)
	var inserted []rune
	switch v := value.(type) {
	case *JavaString:
		return b.InsertJavaString(offset, v)
	case string:
		inserted = StringChars(v)
	case rune:
		inserted = []rune{v}
	case []rune:
		inserted = v
	case bool:
		inserted = StringChars(fmt.Sprintf("%t", v))
	default:
		inserted = StringChars(fmt.Sprintf("%v", v))
	}
	tail := append([]rune{}, b.buf[offset:]...)
	b.buf = append(b.buf[:offset], inserted...)
	b.buf = append(b.buf, tail...)
	return b
}

// Length returns the number of characters currently held, matching
// StringBuilder.length.
func (b *StringBuilder) Length() int32 {
	return int32(len(b.buf))
}

// CharAt returns the character at the given index, matching
// StringBuilder.charAt.
func (b *StringBuilder) CharAt(index int32) rune {
	b.checkIndex(index)
	return b.buf[index]
}

// DeleteCharAt removes the character at the given index and returns the builder,
// matching StringBuilder.deleteCharAt.
func (b *StringBuilder) DeleteCharAt(index int32) *StringBuilder {
	b.checkIndex(index)
	b.buf = append(b.buf[:index], b.buf[index+1:]...)
	return b
}

// Reverse preserves existing high/low surrogate pairs. Reversing two formerly
// unpaired low/high units may form a new valid pair, as Java specifies.
func (b *StringBuilder) Reverse() *StringBuilder {
	for i, j := 0, len(b.buf)-1; i < j; i, j = i+1, j-1 {
		b.buf[i], b.buf[j] = b.buf[j], b.buf[i]
	}
	for i := 0; i+1 < len(b.buf); i++ {
		if b.buf[i] >= 0xdc00 && b.buf[i] <= 0xdfff && b.buf[i+1] >= 0xd800 && b.buf[i+1] <= 0xdbff {
			b.buf[i], b.buf[i+1] = b.buf[i+1], b.buf[i]
			i++
		}
	}
	return b
}

// String returns the accumulated string, matching StringBuilder.toString.
func (b *StringBuilder) String() string {
	return StringFromChars(b.buf)
}

// Compile-time assertion that StringBuilder satisfies fmt.Stringer so that
// transpiled `toString()` calls and string concatenation behave as expected.
var _ fmt.Stringer = (*StringBuilder)(nil)

// Character overloads keep units out of Go's scalar-to-string conversion.
func (b *StringBuilder) AppendChar(value rune) *StringBuilder { return b.Append(value) }
func (b *StringBuilder) InsertChar(offset int32, value rune) *StringBuilder {
	return b.Insert(offset, value)
}
func (b *StringBuilder) AppendChars(value *PrimitiveArray[rune]) *StringBuilder {
	ReferenceRequireNonNull(value)
	return b.Append(value.Elements)
}
func (b *StringBuilder) InsertChars(offset int32, value *PrimitiveArray[rune]) *StringBuilder {
	b.checkOffset(offset)
	ReferenceRequireNonNull(value)
	return b.Insert(offset, value.Elements)
}
func (b *StringBuilder) checkIndex(index int32) {
	if index < 0 || int64(index) >= int64(len(b.buf)) {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Index %d out of bounds for length %d", index, len(b.buf))))
	}
}
func (b *StringBuilder) checkOffset(offset int32) {
	if offset < 0 || int64(offset) > int64(len(b.buf)) {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d) out of bounds for length %d", offset, len(b.buf), len(b.buf))))
	}
}
