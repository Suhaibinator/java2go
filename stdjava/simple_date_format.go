package stdjava

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf16"
)

// SimpleDateFormat compiles Java date pattern fields and retains its calendar.
// It has Java's unsynchronized mutable-instance contract; callers provide any
// synchronization, as Gson does when it retains formatters.
type SimpleDateFormat struct {
	pattern      *JavaString
	tokens       []dateFormatToken
	symbols      *dateFormatSymbols
	calendar     *GregorianCalendar
	centuryStart int64
	centuryYear  int32
}

func (*SimpleDateFormat) JavaDynamicTypeID() TypeID { return "java.text.SimpleDateFormat" }
func NewSimpleDateFormat(pattern string, locales ...*Locale) *SimpleDateFormat {
	StringRequireNonNull(pattern)
	return NewSimpleDateFormatJavaString(dateFormatNativeJavaString(pattern), locales...)
}

// NewSimpleDateFormatJavaString retains the exact immutable pattern reference.
func NewSimpleDateFormatJavaString(pattern *JavaString, locales ...*Locale) *SimpleDateFormat {
	ReferenceRequireNonNull(pattern)
	locale := LocaleGetDefaultCategory(LocaleCategoryFORMAT)
	if len(locales) > 0 {
		locale = locales[0]
	}
	ReferenceRequireNonNull(locale)
	tokens := compileDateFormatPatternUnits(dateFormatJavaUnits(pattern))
	calendar := NewGregorianCalendar(TimeZoneGetDefault(), locale)
	// JDK initializes the default two-digit-year interval at creation minus 80
	// years. This retained boundary must not move during later parsing calls.
	now := time.Now().In(calendar.zone.location).AddDate(-80, 0, 0)
	return &SimpleDateFormat{pattern: pattern, tokens: tokens, symbols: dateFormatLocaleSymbols(locale), calendar: calendar, centuryStart: now.UnixMilli(), centuryYear: int32(now.Year())}
}
func (f *SimpleDateFormat) ToPattern() string { return dateFormatNativeText(f.ToPatternJavaString()) }
func (f *SimpleDateFormat) ToPatternJavaString() *JavaString {
	ReferenceRequireNonNull(f)
	return f.pattern
}
func (f *SimpleDateFormat) ApplyPattern(pattern string) {
	ReferenceRequireNonNull(f)
	StringRequireNonNull(pattern)
	f.ApplyPatternJavaString(dateFormatNativeJavaString(pattern))
}
func (f *SimpleDateFormat) ApplyPatternJavaString(pattern *JavaString) {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(pattern)
	tokens := compileDateFormatPatternUnits(dateFormatJavaUnits(pattern))
	f.pattern = pattern
	f.tokens = tokens
}
func (f *SimpleDateFormat) GetTimeZone() *TimeZone {
	ReferenceRequireNonNull(f)
	return f.calendar.GetTimeZone()
}
func (f *SimpleDateFormat) SetTimeZone(zone *TimeZone) {
	ReferenceRequireNonNull(f)
	f.calendar.SetTimeZone(zone)
}
func (f *SimpleDateFormat) SetLenient(value bool) {
	ReferenceRequireNonNull(f)
	f.calendar.SetLenient(value)
}
func (f *SimpleDateFormat) IsLenient() bool {
	ReferenceRequireNonNull(f)
	return f.calendar.IsLenient()
}
func (f *SimpleDateFormat) Set2DigitYearStart(date DateValue) {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(date)
	f.centuryStart = date.GetTime()
	f.calendar.SetTime(date)
	f.centuryYear = f.calendar.Get(CalendarYEAR)
}
func (f *SimpleDateFormat) Get2DigitYearStart() *Date {
	ReferenceRequireNonNull(f)
	return NewDate(f.centuryStart)
}

func dateFormatNumber(value int32, width int) string {
	text := fmt.Sprint(value)
	if value >= 0 && len(text) < width {
		return strings.Repeat("0", width-len(text)) + text
	}
	return text
}
func (f *SimpleDateFormat) Format(date DateValue) string {
	return string(utf16.Decode(f.formatUnits(date)))
}
func (f *SimpleDateFormat) FormatJavaString(date DateValue) *JavaString {
	return NewJavaStringUTF16(f.formatUnits(date))
}
func (f *SimpleDateFormat) formatUnits(date DateValue) []uint16 {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(date)
	ReferenceRequireNonNull(f.calendar.zone)
	f.calendar.SetTime(date)
	var out dateFormatUnitBuilder
	cal := f.calendar
	for _, token := range f.tokens {
		if token.field == 0 {
			out.WriteUnits(token.literal)
			continue
		}
		var value int32
		switch token.field {
		case 'G':
			out.WriteString(f.symbols.eras[cal.Get(CalendarERA)])
			continue
		case 'y':
			value = cal.Get(CalendarYEAR)
			if token.count == 2 {
				value %= 100
			}
		case 'M', 'L':
			value = cal.Get(CalendarMONTH) + 1
			if token.count >= 4 {
				out.WriteString(f.symbols.months[value-1])
				continue
			}
			if token.count == 3 {
				out.WriteString(f.symbols.shortMonths[value-1])
				continue
			}
		case 'd':
			value = cal.Get(CalendarDAY_OF_MONTH)
		case 'D':
			value = cal.Get(CalendarDAY_OF_YEAR)
		case 'E':
			i := cal.Get(CalendarDAY_OF_WEEK) - 1
			if token.count >= 4 {
				out.WriteString(f.symbols.days[i])
			} else {
				out.WriteString(f.symbols.shortDays[i])
			}
			continue
		case 'u':
			value = (cal.Get(CalendarDAY_OF_WEEK)+5)%7 + 1
		case 'a':
			out.WriteString(f.symbols.ampm[cal.Get(CalendarAM_PM)])
			continue
		case 'H':
			value = cal.Get(CalendarHOUR_OF_DAY)
		case 'k':
			value = cal.Get(CalendarHOUR_OF_DAY)
			if value == 0 {
				value = 24
			}
		case 'K':
			value = cal.Get(CalendarHOUR)
		case 'h':
			value = cal.Get(CalendarHOUR)
			if value == 0 {
				value = 12
			}
		case 'm':
			value = cal.Get(CalendarMINUTE)
		case 's':
			value = cal.Get(CalendarSECOND)
		case 'S':
			value = cal.Get(CalendarMILLISECOND)
		case 'z':
			out.WriteString(dateFormatZoneName(cal.zone, date.GetTime(), token.count >= 4))
			continue
		case 'Z', 'X':
			out.WriteString(dateFormatZoneOffset(cal.zone.GetOffset(date.GetTime()), token.field, token.count))
			continue
		default:
			panic(NewUnsupportedOperationException("date pattern field: " + string(token.field)))
		}
		out.WriteString(dateFormatNumber(value, token.count))
	}
	return out.units
}
func (f *SimpleDateFormat) Parse(text string, positions ...*ParsePosition) *Date {
	ReferenceRequireNonNull(f)
	StringRequireNonNull(text)
	return f.ParseJavaString(dateFormatNativeJavaString(text), positions...)
}
func (f *SimpleDateFormat) ParseJavaString(text *JavaString, positions ...*ParsePosition) *Date {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(text)
	units := dateFormatJavaUnits(text)
	if len(positions) > 0 {
		ReferenceRequireNonNull(positions[0])
		return f.parsePositionUnits(units, positions[0])
	}
	position := NewParsePosition(0)
	date := f.parsePositionUnits(units, position)
	if position.index == 0 {
		message := append([]uint16{'U', 'n', 'p', 'a', 'r', 's', 'e', 'a', 'b', 'l', 'e', ' ', 'd', 'a', 't', 'e', ':', ' ', '"'}, text.units...)
		message = append(message, '"')
		panic(ParseException{newJavaThrowableBase("ParseException", NewJavaStringUTF16(message)), position.errorIndex})
	}
	return date
}
