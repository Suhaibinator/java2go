package stdjava

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

func timeRequire[T any](value *T) *T { ReferenceRequireNonNull(value); return value }
func timeText(execution *Execution, value any) *JavaString {
	ReferenceRequireNonNull(value)
	if text, ok := value.(*JavaString); ok {
		return text
	}
	length := CharSequenceLength(execution, value)
	units := make([]uint16, length)
	for i := int32(0); i < length; i++ {
		units[i] = uint16(CharSequenceCharAt(execution, value, i))
	}
	return NewJavaStringUTF16(units)
}
func timeHost(text *JavaString) string {
	ReferenceRequireNonNull(text)
	return string(utf16.Decode(text.units))
}
func timeYear(year int32) string {
	if year >= 0 && year < 10000 {
		return fmt.Sprintf("%04d", year)
	}
	if year < 0 && year > -10000 {
		return fmt.Sprintf("-%04d", -year)
	}
	if year > 9999 {
		return "+" + strconv.FormatInt(int64(year), 10)
	}
	return strconv.FormatInt(int64(year), 10)
}
func timeFraction(nano int32, grouped bool) string {
	if nano == 0 {
		return ""
	}
	digits := fmt.Sprintf("%09d", nano)
	if grouped {
		if nano%1000000 == 0 {
			digits = digits[:3]
		} else if nano%1000 == 0 {
			digits = digits[:6]
		}
	} else {
		digits = strings.TrimRight(digits, "0")
	}
	return "." + digits
}
func timeRange(field string, value int32, min, max int32) {
	if value < min || value > max {
		panic(NewDateTimeException(fmt.Sprintf("Invalid value for %s (valid values %d - %d): %d", field, min, max, value)))
	}
}
func timeParseFailure(text *JavaString, index int32) {
	panic(NewDateTimeParseException("Text '"+timeHost(text)+"' could not be parsed", text, index))
}

func JavaTimeStringExecution(execution *Execution, value any) *JavaString {
	ReferenceRequireNonNull(value)
	return value.(interface{ StringJava2goExecution(*Execution) *JavaString }).StringJava2goExecution(execution)
}
