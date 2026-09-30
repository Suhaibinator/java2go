package stdjava

import "fmt"

// DateFormat is the nominal abstract formatter view. Concrete instances keep
// their calendar, locale and parsing state when viewed through this protocol.
type DateFormat interface {
	JavaDynamicTypeID() TypeID
	Format(DateValue) string
	Parse(string, ...*ParsePosition) *Date
	GetTimeZone() *TimeZone
	SetTimeZone(*TimeZone)
	SetLenient(bool)
	IsLenient() bool
}

// JavaDateFormat is the canonical String protocol on the same formatter object.
// Source DateFormat subclasses still require their own execution-aware bridge.
type JavaDateFormat interface {
	DateFormat
	FormatJavaString(DateValue) *JavaString
	ParseJavaString(*JavaString, ...*ParsePosition) *Date
}

func DateFormatGetDateTimeInstanceJavaString(dateStyle, timeStyle int32, locales ...*Locale) JavaDateFormat {
	return DateFormatGetDateTimeInstance(dateStyle, timeStyle, locales...).(JavaDateFormat)
}
func DateFormatFormatJavaString(value JavaDateFormat, date DateValue) *JavaString {
	ReferenceRequireNonNull(value)
	return value.FormatJavaString(date)
}
func DateFormatParseJavaString(value JavaDateFormat, text *JavaString, positions ...*ParsePosition) *Date {
	ReferenceRequireNonNull(value)
	return value.ParseJavaString(text, positions...)
}

const (
	DateFormatFULL int32 = iota
	DateFormatLONG
	DateFormatMEDIUM
	DateFormatSHORT
	DateFormatDEFAULT = DateFormatMEDIUM
)

func DateFormatGetDateTimeInstance(dateStyle, timeStyle int32, locales ...*Locale) DateFormat {
	locale := LocaleGetDefault()
	if len(locales) > 0 {
		locale = locales[0]
	}
	ReferenceRequireNonNull(locale)
	if dateStyle < 0 || dateStyle > 3 {
		panic(NewIllegalArgumentException("Illegal date style " + fmt.Sprint(dateStyle)))
	}
	if timeStyle < 0 || timeStyle > 3 {
		panic(NewIllegalArgumentException("Illegal time style " + fmt.Sprint(timeStyle)))
	}
	data := dateFormatLocaleSymbols(locale)
	return NewSimpleDateFormat(data.dates[dateStyle]+", "+data.times[timeStyle], locale)
}
func DateFormatFormat(value DateFormat, date DateValue) string {
	ReferenceRequireNonNull(value)
	return value.Format(date)
}
func DateFormatParse(value DateFormat, text string, positions ...*ParsePosition) *Date {
	ReferenceRequireNonNull(value)
	return value.Parse(text, positions...)
}
func DateFormatGetTimeZone(value DateFormat) *TimeZone {
	ReferenceRequireNonNull(value)
	return value.GetTimeZone()
}
func DateFormatSetTimeZone(value DateFormat, zone *TimeZone) {
	ReferenceRequireNonNull(value)
	value.SetTimeZone(zone)
}
func DateFormatSetLenient(value DateFormat, lenient bool) {
	ReferenceRequireNonNull(value)
	value.SetLenient(lenient)
}
func DateFormatIsLenient(value DateFormat) bool {
	ReferenceRequireNonNull(value)
	return value.IsLenient()
}
func init() {
	RegisterJavaType("java.text.DateFormat", "java.text.Format")
	RegisterJavaType("java.text.Format", ObjectTypeID, SerializableTypeID, CloneableTypeID)
	RegisterJavaType("java.text.SimpleDateFormat", "java.text.DateFormat")
}
