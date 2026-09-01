package stdjava

import "fmt"

// StringBuilder is a wrapper around strings.Builder that models the subset of
// java.lang.StringBuilder (and StringBuffer) used by transpiled code. Because
// Java's StringBuilder supports operations strings.Builder does not (insert,
// deleteCharAt, reverse, length queries after the fact), this keeps the
// accumulated runes directly rather than delegating every operation to
// strings.Builder. The append fast-path still uses strings.Builder semantics.
//
// StringBuilder is not safe for concurrent use, matching Java's StringBuilder
// (StringBuffer's synchronization is not modeled).
type StringBuilder struct {
	buf []rune
}

// NewStringBuilder returns an empty StringBuilder, matching `new StringBuilder()`.
func NewStringBuilder() *StringBuilder {
	return &StringBuilder{}
}

// NewStringBuilderString returns a StringBuilder seeded with the given string,
// matching `new StringBuilder(String)`.
func NewStringBuilderString(s string) *StringBuilder {
	return &StringBuilder{buf: []rune(s)}
}

// Append appends the textual representation of value to the builder and returns
// the builder for chaining, matching StringBuilder.append. The value is
// formatted with the same rules Java applies for the common overloads: strings
// and runes append directly, everything else uses its default string form.
func (b *StringBuilder) Append(value any) *StringBuilder {
	switch v := value.(type) {
	case string:
		b.buf = append(b.buf, []rune(v)...)
	case rune:
		b.buf = append(b.buf, v)
	case []rune:
		b.buf = append(b.buf, v...)
	case bool:
		b.buf = append(b.buf, []rune(fmt.Sprintf("%t", v))...)
	default:
		b.buf = append(b.buf, []rune(fmt.Sprintf("%v", v))...)
	}
	return b
}

// Insert inserts the textual representation of value at the given rune offset,
// matching StringBuilder.insert.
func (b *StringBuilder) Insert(offset int32, value any) *StringBuilder {
	var inserted []rune
	switch v := value.(type) {
	case string:
		inserted = []rune(v)
	case rune:
		inserted = []rune{v}
	case []rune:
		inserted = v
	case bool:
		inserted = []rune(fmt.Sprintf("%t", v))
	default:
		inserted = []rune(fmt.Sprintf("%v", v))
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
	return b.buf[index]
}

// DeleteCharAt removes the character at the given index and returns the builder,
// matching StringBuilder.deleteCharAt.
func (b *StringBuilder) DeleteCharAt(index int32) *StringBuilder {
	b.buf = append(b.buf[:index], b.buf[index+1:]...)
	return b
}

// Reverse reverses the characters in place and returns the builder, matching
// StringBuilder.reverse. Surrogate pairs are not modeled, so this reverses by
// rune.
func (b *StringBuilder) Reverse() *StringBuilder {
	for i, j := 0, len(b.buf)-1; i < j; i, j = i+1, j-1 {
		b.buf[i], b.buf[j] = b.buf[j], b.buf[i]
	}
	return b
}

// String returns the accumulated string, matching StringBuilder.toString.
func (b *StringBuilder) String() string {
	return string(b.buf)
}

// Compile-time assertion that StringBuilder satisfies fmt.Stringer so that
// transpiled `toString()` calls and string concatenation behave as expected.
var _ fmt.Stringer = (*StringBuilder)(nil)
