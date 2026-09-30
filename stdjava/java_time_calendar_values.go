package stdjava

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type LocalDate struct{ year, month, day int32 }
type LocalTime struct{ hour, minute, second, nano int32 }
type LocalDateTime struct {
	date *LocalDate
	time *LocalTime
}
type MonthDay struct{ month, day int32 }
type Year struct{ year int32 }
type YearMonth struct{ year, month int32 }
type Period struct{ years, months, days int32 }

func timeLeap(year int32) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

var timeMonthNames = [...]string{"", "JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE", "JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER"}

func timeMonthLength(year, month int32) int32 {
	if month == 2 {
		if timeLeap(year) {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}
func LocalDateOf(year, month, day int32) *LocalDate {
	timeRange("Year", year, -999999999, 999999999)
	timeRange("MonthOfYear", month, 1, 12)
	timeRange("DayOfMonth", day, 1, 31)
	if day > timeMonthLength(year, month) {
		if month == 2 && day == 29 {
			panic(NewDateTimeException(fmt.Sprintf("Invalid date 'February 29' as '%d' is not a leap year", year)))
		}
		name := strings.ToLower(timeMonthNames[month])
		name = strings.ToUpper(name[:1]) + name[1:]
		panic(NewDateTimeException(fmt.Sprintf("Invalid date '%s %d'", name, day)))
	}
	return &LocalDate{year, month, day}
}

var localMidnight = &LocalTime{}

func LocalTimeOf(hour, minute, second, nano int32) *LocalTime {
	timeRange("HourOfDay", hour, 0, 23)
	timeRange("MinuteOfHour", minute, 0, 59)
	timeRange("SecondOfMinute", second, 0, 59)
	timeRange("NanoOfSecond", nano, 0, 999999999)
	if hour == 0 && minute == 0 && second == 0 && nano == 0 {
		return localMidnight
	}
	return &LocalTime{hour, minute, second, nano}
}
func LocalDateTimeOf(date *LocalDate, clock *LocalTime) *LocalDateTime {
	return &LocalDateTime{timeRequire(date), timeRequire(clock)}
}
func MonthDayOf(month, day int32) *MonthDay {
	timeRange("MonthOfYear", month, 1, 12)
	timeRange("DayOfMonth", day, 1, 31)
	if day > timeMonthLength(2000, month) {
		panic(NewDateTimeException(fmt.Sprintf("Illegal value for DayOfMonth field, value %d is not valid for month %s", day, timeMonthNames[month])))
	}
	return &MonthDay{month, day}
}
func YearOf(year int32) *Year { timeRange("Year", year, -999999999, 999999999); return &Year{year} }
func YearMonthOf(year, month int32) *YearMonth {
	timeRange("Year", year, -999999999, 999999999)
	timeRange("MonthOfYear", month, 1, 12)
	return &YearMonth{year, month}
}

var periodZero = &Period{}

func PeriodOf(years, months, days int32) *Period {
	if years == 0 && months == 0 && days == 0 {
		return periodZero
	}
	return &Period{years, months, days}
}
func (d *LocalDate) GetYear() int32              { return timeRequire(d).year }
func (d *LocalDate) GetMonthValue() int32        { return timeRequire(d).month }
func (d *LocalDate) GetDayOfMonth() int32        { return timeRequire(d).day }
func (t *LocalTime) GetHour() int32              { return timeRequire(t).hour }
func (t *LocalTime) GetMinute() int32            { return timeRequire(t).minute }
func (t *LocalTime) GetSecond() int32            { return timeRequire(t).second }
func (t *LocalTime) GetNano() int32              { return timeRequire(t).nano }
func (l *LocalDateTime) ToLocalDate() *LocalDate { return timeRequire(l).date }
func (l *LocalDateTime) ToLocalTime() *LocalTime { return timeRequire(l).time }
func (d *MonthDay) GetMonthValue() int32         { return timeRequire(d).month }
func (d *MonthDay) GetDayOfMonth() int32         { return timeRequire(d).day }
func (y *Year) GetValue() int32                  { return timeRequire(y).year }
func (y *YearMonth) GetYear() int32              { return timeRequire(y).year }
func (y *YearMonth) GetMonthValue() int32        { return timeRequire(y).month }
func (p *Period) GetYears() int32                { return timeRequire(p).years }
func (p *Period) GetMonths() int32               { return timeRequire(p).months }
func (p *Period) GetDays() int32                 { return timeRequire(p).days }
func (d *LocalDate) String() string {
	timeRequire(d)
	return timeYear(d.year) + fmt.Sprintf("-%02d-%02d", d.month, d.day)
}
func (t *LocalTime) String() string {
	timeRequire(t)
	out := fmt.Sprintf("%02d:%02d", t.hour, t.minute)
	if t.second != 0 || t.nano != 0 {
		out += fmt.Sprintf(":%02d", t.second) + timeFraction(t.nano, true)
	}
	return out
}
func (l *LocalDateTime) String() string {
	timeRequire(l)
	return l.date.String() + "T" + l.time.String()
}
func (d *MonthDay) String() string { timeRequire(d); return fmt.Sprintf("--%02d-%02d", d.month, d.day) }
func (y *Year) String() string     { return strconv.FormatInt(int64(timeRequire(y).year), 10) }
func (y *YearMonth) String() string {
	timeRequire(y)
	year := timeYear(y.year)
	if y.year > 9999 {
		year = strings.TrimPrefix(year, "+")
	}
	return year + fmt.Sprintf("-%02d", y.month)
}
func (p *Period) String() string {
	timeRequire(p)
	if *p == (Period{}) {
		return "P0D"
	}
	out := "P"
	if p.years != 0 {
		out += fmt.Sprintf("%dY", p.years)
	}
	if p.months != 0 {
		out += fmt.Sprintf("%dM", p.months)
	}
	if p.days != 0 {
		out += fmt.Sprintf("%dD", p.days)
	}
	return out
}

var localDatePattern = regexp.MustCompile(`^([-+]?[0-9]{4,9})-([0-9]{2})-([0-9]{2})$`)
var localTimePattern = regexp.MustCompile(`^([0-9]{2}):([0-9]{2})(?::([0-9]{2})(?:\.([0-9]{1,9}))?)?$`)

func timeParsedInt(text *JavaString, part string) int32 {
	v, err := strconv.ParseInt(part, 10, 32)
	if err != nil {
		timeParseFailure(text, 0)
	}
	return int32(v)
}
func timeParsing(text *JavaString, invoke func()) {
	defer func() {
		if failed := recover(); failed != nil {
			if _, ok := failed.(DateTimeException); ok {
				timeParseFailure(text, 0)
			}
			panic(failed)
		}
	}()
	invoke()
}
func LocalDateParseExecution(execution *Execution, value any) *LocalDate {
	text := timeText(execution, value)
	host := timeHost(text)
	m := localDatePattern.FindStringSubmatch(host)
	if m == nil {
		timeParseFailure(text, 0)
	}
	if (len(m[1]) > 4 && m[1][0] != '-' && m[1][0] != '+') || (m[1][0] == '+' && len(m[1]) < 6) || m[1] == "-0000" {
		timeParseFailure(text, 0)
	}
	var out *LocalDate
	timeParsing(text, func() {
		out = LocalDateOf(timeParsedInt(text, m[1]), timeParsedInt(text, m[2]), timeParsedInt(text, m[3]))
	})
	return out
}
func LocalTimeParseExecution(execution *Execution, value any) *LocalTime {
	text := timeText(execution, value)
	m := localTimePattern.FindStringSubmatch(timeHost(text))
	if m == nil {
		timeParseFailure(text, 0)
	}
	second, nano := int32(0), int32(0)
	if m[3] != "" {
		second = timeParsedInt(text, m[3])
	}
	if m[4] != "" {
		nano = timeParsedInt(text, m[4]+strings.Repeat("0", 9-len(m[4])))
	}
	var out *LocalTime
	timeParsing(text, func() { out = LocalTimeOf(timeParsedInt(text, m[1]), timeParsedInt(text, m[2]), second, nano) })
	return out
}
func LocalDateTimeParseExecution(execution *Execution, value any) *LocalDateTime {
	text := timeText(execution, value)
	host := timeHost(text)
	split := strings.IndexAny(host, "Tt")
	if split < 0 {
		timeParseFailure(text, 0)
	}
	var date *LocalDate
	var clock *LocalTime
	func() {
		defer func() {
			if failed := recover(); failed != nil {
				if parsed, ok := failed.(DateTimeParseException); ok {
					index := parsed.index
					if date != nil {
						index += int32(split + 1)
					}
					panic(NewDateTimeParseException("Text '"+host+"' could not be parsed", text, index))
				}
				panic(failed)
			}
		}()
		date = LocalDateParseExecution(execution, JavaStringFromHostUTF8(host[:split]))
		clock = LocalTimeParseExecution(execution, JavaStringFromHostUTF8(host[split+1:]))
	}()
	return LocalDateTimeOf(date, clock)
}
func (l *LocalDateTime) timeUTC() time.Time {
	return time.Date(int(l.date.year), time.Month(l.date.month), int(l.date.day), int(l.time.hour), int(l.time.minute), int(l.time.second), int(l.time.nano), time.UTC)
}
func localDateTimeFromHost(t time.Time) *LocalDateTime {
	return LocalDateTimeOf(LocalDateOf(int32(t.Year()), int32(t.Month()), int32(t.Day())), LocalTimeOf(int32(t.Hour()), int32(t.Minute()), int32(t.Second()), int32(t.Nanosecond())))
}
func (value *LocalDate) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*LocalDate)
	return ok && other != nil && (*value == *other)
}
func (*LocalDate) JavaDynamicTypeID() TypeID { return "java.time.LocalDate" }
func (value *LocalDate) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *LocalTime) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*LocalTime)
	return ok && other != nil && (*value == *other)
}
func (*LocalTime) JavaDynamicTypeID() TypeID { return "java.time.LocalTime" }
func (value *LocalTime) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *LocalDateTime) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*LocalDateTime)
	return ok && other != nil && (value.date.Equals(other.date) && value.time.Equals(other.time))
}
func (*LocalDateTime) JavaDynamicTypeID() TypeID { return "java.time.LocalDateTime" }
func (value *LocalDateTime) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *MonthDay) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*MonthDay)
	return ok && other != nil && (*value == *other)
}
func (*MonthDay) JavaDynamicTypeID() TypeID { return "java.time.MonthDay" }
func (value *MonthDay) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *Year) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*Year)
	return ok && other != nil && (*value == *other)
}
func (*Year) JavaDynamicTypeID() TypeID { return "java.time.Year" }
func (value *Year) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *YearMonth) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*YearMonth)
	return ok && other != nil && (*value == *other)
}
func (*YearMonth) JavaDynamicTypeID() TypeID { return "java.time.YearMonth" }
func (value *YearMonth) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
func (value *Period) Equals(candidate any) bool {
	timeRequire(value)
	other, ok := candidate.(*Period)
	return ok && other != nil && (*value == *other)
}
func (*Period) JavaDynamicTypeID() TypeID { return "java.time.Period" }
func (value *Period) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(value.String())
}
