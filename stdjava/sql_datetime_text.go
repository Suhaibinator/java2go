package stdjava

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

func sqlLocalCalendar(millis int64) *GregorianCalendar {
	calendar := NewGregorianCalendar(TimeZoneGetDefault())
	calendar.SetTimeInMillis(millis)
	return calendar
}
func sqlLocalMillis(year, month, day, hour, minute, second int32) int64 {
	calendar := NewGregorianCalendar(TimeZoneGetDefault())
	calendar.Clear()
	calendar.Set(year, month-1, day, hour, minute, second)
	return calendar.GetTimeInMillis()
}
func (date *SQLDate) String() string {
	ReferenceRequireNonNull(date)
	c := sqlLocalCalendar(date.GetTime())
	return fmt.Sprintf("%04d-%02d-%02d", c.Get(CalendarYEAR)%10000, c.Get(CalendarMONTH)+1, c.Get(CalendarDAY_OF_MONTH))
}
func (value *SQLTime) String() string {
	ReferenceRequireNonNull(value)
	c := sqlLocalCalendar(value.GetTime())
	return fmt.Sprintf("%02d:%02d:%02d", c.Get(CalendarHOUR_OF_DAY), c.Get(CalendarMINUTE), c.Get(CalendarSECOND))
}
func (stamp *SQLTimestamp) String() string {
	ReferenceRequireNonNull(stamp)
	c := sqlLocalCalendar(stamp.GetTime())
	fraction := strings.TrimRight(fmt.Sprintf("%09d", stamp.nanos), "0")
	if fraction == "" {
		fraction = "0"
	}
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%s", c.Get(CalendarYEAR), c.Get(CalendarMONTH)+1, c.Get(CalendarDAY_OF_MONTH), c.Get(CalendarHOUR_OF_DAY), c.Get(CalendarMINUTE), c.Get(CalendarSECOND), fraction)
}

// JDBC valueOf uses Integer's CharSequence-range parser. Preserve its UTF-16
// error index, Unicode digits, sign handling, and 32-bit overflow boundary.
func sqlParseInteger(text string) int32 {
	units := utf16.Encode([]rune(text))
	if len(units) == 0 {
		panic(NewNumberFormatException("For input string: \"\""))
	}
	fail := func(index int) {
		panic(NewNumberFormatException(fmt.Sprintf("Error at index %d in: \"%s\"", index, text)))
	}
	negative, start := false, 0
	if units[0] == '-' || units[0] == '+' {
		negative = units[0] == '-'
		start = 1
	}
	if start == len(units) {
		fail(start)
	}
	limit := int64(2147483647)
	if negative {
		limit++
	}
	var value int64
	for i := start; i < len(units); i++ {
		digit := int64(CharDigit(rune(units[i]), 10))
		if digit < 0 || value > (limit-digit)/10 {
			fail(i)
		}
		value = value*10 + digit
	}
	if negative {
		value = -value
	}
	return int32(value)
}
func sqlDateParts(text string) (year, month, day int32, valid bool) {
	parts := strings.Split(text, "-")
	if len(parts) != 3 || StringLength(parts[0]) != 4 || StringLength(parts[1]) < 1 || StringLength(parts[1]) > 2 || StringLength(parts[2]) < 1 || StringLength(parts[2]) > 2 {
		return
	}
	year, month, day = sqlParseInteger(parts[0]), sqlParseInteger(parts[1]), sqlParseInteger(parts[2])
	valid = month >= 1 && month <= 12 && day >= 1 && day <= 31
	return
}
func SQLDateValueOf(text string) *SQLDate {
	if StringIsNull(text) {
		panic(NewIllegalArgumentException())
	}
	year, month, day, valid := sqlDateParts(text)
	if !valid {
		panic(NewIllegalArgumentException())
	}
	return NewSQLDate(sqlLocalMillis(year, month, day, 0, 0, 0))
}
func SQLTimeValueOf(text string) *SQLTime {
	if StringIsNull(text) {
		panic(NewIllegalArgumentException())
	}
	parts := strings.SplitN(text, ":", 3)
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		panic(NewIllegalArgumentException())
	}
	hour, minute, second := sqlParseInteger(parts[0]), sqlParseInteger(parts[1]), sqlParseInteger(parts[2])
	return NewSQLTime(sqlLocalMillis(1970, 1, 1, hour, minute, second))
}
func SQLTimestampValueOf(text string) *SQLTimestamp {
	const invalid = "Timestamp format must be yyyy-mm-dd hh:mm:ss[.fffffffff]"
	if StringIsNull(text) {
		panic(NewIllegalArgumentException("null string"))
	}
	text = StringTrim(text)
	datePart, timePart, ok := strings.Cut(text, " ")
	if !ok {
		panic(NewIllegalArgumentException(invalid))
	}
	year, month, day, valid := sqlDateParts(datePart)
	if !valid {
		panic(NewIllegalArgumentException(invalid))
	}
	clock := strings.SplitN(timePart, ":", 3)
	if len(clock) != 3 || clock[2] == "" {
		panic(NewIllegalArgumentException(invalid))
	}
	hour, minute := sqlParseInteger(clock[0]), sqlParseInteger(clock[1])
	seconds, fraction, hasFraction := strings.Cut(clock[2], ".")
	if hasFraction && fraction == "" {
		panic(NewIllegalArgumentException(invalid))
	}
	second := sqlParseInteger(seconds)
	nanos := int32(0)
	if hasFraction {
		precision := StringLength(fraction)
		if precision < 1 || precision > 9 || CharDigit(rune(StringCharAt(fraction, 0)), 10) < 0 {
			panic(NewIllegalArgumentException(invalid))
		}
		nanos = sqlParseInteger(fraction)
		for precision < 9 {
			nanos *= 10
			precision++
		}
	}
	stamp := NewSQLTimestamp(sqlLocalMillis(year, month, day, hour, minute, second))
	stamp.SetNanos(nanos)
	return stamp
}
