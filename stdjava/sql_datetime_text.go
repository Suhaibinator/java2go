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

// Native Go callers retain their UTF-8/sentinel compatibility boundary. Java
// lowering calls the reference services directly and never passes through it.
func sqlNativeJavaString(text string) *JavaString {
	if StringIsNull(text) {
		return nil
	}
	return NewJavaStringUTF16(utf16.Encode([]rune(text)))
}
func SQLDateValueOf(text string) *SQLDate { return SQLDateValueOfJavaString(sqlNativeJavaString(text)) }
func SQLTimeValueOf(text string) *SQLTime { return SQLTimeValueOfJavaString(sqlNativeJavaString(text)) }
func SQLTimestampValueOf(text string) *SQLTimestamp {
	return SQLTimestampValueOfJavaString(sqlNativeJavaString(text))
}
