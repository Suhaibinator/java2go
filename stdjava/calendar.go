package stdjava

import (
	"fmt"
	"time"
)

const (
	CalendarERA int32 = iota
	CalendarYEAR
	CalendarMONTH
	CalendarWEEK_OF_YEAR
	CalendarWEEK_OF_MONTH
	CalendarDAY_OF_MONTH
	CalendarDAY_OF_YEAR
	CalendarDAY_OF_WEEK
	CalendarDAY_OF_WEEK_IN_MONTH
	CalendarAM_PM
	CalendarHOUR
	CalendarHOUR_OF_DAY
	CalendarMINUTE
	CalendarSECOND
	CalendarMILLISECOND
	CalendarZONE_OFFSET
	CalendarDST_OFFSET
)

var calendarFieldNames = [17]string{"ERA", "YEAR", "MONTH", "WEEK_OF_YEAR", "WEEK_OF_MONTH", "DAY_OF_MONTH", "DAY_OF_YEAR", "DAY_OF_WEEK", "DAY_OF_WEEK_IN_MONTH", "AM_PM", "HOUR", "HOUR_OF_DAY", "MINUTE", "SECOND", "MILLISECOND", "ZONE_OFFSET", "DST_OFFSET"}

// Calendar is a nominal Java base protocol. GregorianCalendar is its concrete
// implementation with a distinct class descriptor, not a Calendar class alias.
type Calendar interface {
	JavaDynamicTypeID() TypeID
	Get(int32) int32
	Set(...int32)
	Clear(...int32)
	SetLenient(bool)
	IsLenient() bool
	GetTime() *Date
	SetTime(DateValue)
	GetTimeInMillis() int64
	SetTimeInMillis(int64)
	GetTimeZone() *TimeZone
	SetTimeZone(*TimeZone)
}

// GregorianCalendar retains unset/computed/user-set field state. Conversion
// uses the JDK default Julian/Gregorian cutover and resolves zone transitions
// independently of Go time.Date's unspecified choice for ambiguous wall times.
// Week-based field selection and custom cutovers are not yet implemented.
type GregorianCalendar struct {
	zone      *TimeZone
	fields    [17]int32
	stamps    [17]int
	nextStamp int
	millis    int64
	valid     bool
	lenient   bool
}

func (*GregorianCalendar) JavaDynamicTypeID() TypeID { return "java.util.GregorianCalendar" }
func NewGregorianCalendar(arguments ...any) *GregorianCalendar {
	calendar := &GregorianCalendar{zone: TimeZoneGetDefault(), lenient: true, nextStamp: 2}
	switch len(arguments) {
	case 0:
	case 1:
		if zone, ok := arguments[0].(*TimeZone); ok {
			ReferenceRequireNonNull(zone)
			calendar.zone = zone
		} else if locale, ok := arguments[0].(*Locale); ok {
			ReferenceRequireNonNull(locale)
		} else {
			panic(NewNullPointerException("zone or locale is null"))
		}
	case 2:
		zone, ok := arguments[0].(*TimeZone)
		if !ok || zone == nil {
			panic(NewNullPointerException("zone is null"))
		}
		ReferenceRequireNonNull(arguments[1])
		calendar.zone = zone
	case 3, 5, 6:
		calendar.Clear()
		values := make([]int32, len(arguments))
		for i, value := range arguments {
			values[i] = calendarInteger(value)
		}
		calendar.Set(values...)
		calendar.Set(CalendarMILLISECOND, 0)
		return calendar
	default:
		panic(NewIllegalArgumentException("unsupported GregorianCalendar constructor"))
	}
	calendar.SetTimeInMillis(time.Now().UnixMilli())
	return calendar
}
func calendarInteger(value any) int32 {
	switch value := value.(type) {
	case int:
		return int32(value)
	case int32:
		return value
	case int64:
		return int32(value)
	default:
		panic(NewIllegalArgumentException("calendar field is not an int"))
	}
}
func (calendar *GregorianCalendar) Clear(fields ...int32) {
	ReferenceRequireNonNull(calendar)
	if len(fields) == 0 {
		calendar.fields = [17]int32{}
		calendar.stamps = [17]int{}
		calendar.nextStamp = 2
	} else {
		field := fields[0]
		calendarCheckField(field)
		calendar.fields[field] = 0
		calendar.stamps[field] = 0
	}
	calendar.valid = false
}
func calendarCheckField(field int32) {
	if field < 0 || field >= 17 {
		panic(NewArrayIndexOutOfBoundsException(fmt.Sprint(field)))
	}
}
func calendarWritableField(field int32) bool {
	return field == CalendarERA || field == CalendarYEAR || field == CalendarMONTH || field == CalendarDAY_OF_MONTH || field == CalendarAM_PM || field == CalendarHOUR || field == CalendarHOUR_OF_DAY || field == CalendarMINUTE || field == CalendarSECOND || field == CalendarMILLISECOND
}
func (calendar *GregorianCalendar) Set(values ...int32) {
	ReferenceRequireNonNull(calendar)
	if len(values) == 2 {
		field, value := values[0], values[1]
		calendarCheckField(field)
		if !calendarWritableField(field) {
			panic(NewUnsupportedOperationException("calendar field selection: " + calendarFieldNames[field]))
		}
		calendar.fields[field] = value
		calendar.stamps[field] = calendar.nextStamp
		calendar.nextStamp++
		calendar.valid = false
		return
	}
	if len(values) != 3 && len(values) != 5 && len(values) != 6 {
		panic(NewIllegalArgumentException("unsupported Calendar.set arity"))
	}
	for i, field := range []int32{CalendarYEAR, CalendarMONTH, CalendarDAY_OF_MONTH, CalendarHOUR_OF_DAY, CalendarMINUTE, CalendarSECOND} {
		if i < len(values) {
			calendar.Set(field, values[i])
		}
	}
}
func (calendar *GregorianCalendar) SetLenient(value bool) {
	ReferenceRequireNonNull(calendar)
	calendar.lenient = value
}
func (calendar *GregorianCalendar) IsLenient() bool {
	ReferenceRequireNonNull(calendar)
	return calendar.lenient
}
func (calendar *GregorianCalendar) GetTime() *Date { return NewDate(calendar.GetTimeInMillis()) }
func (calendar *GregorianCalendar) SetTime(date DateValue) {
	ReferenceRequireNonNull(date)
	calendar.SetTimeInMillis(date.GetTime())
}
func (calendar *GregorianCalendar) GetTimeZone() *TimeZone {
	ReferenceRequireNonNull(calendar)
	return calendar.zone
}
func (calendar *GregorianCalendar) SetTimeZone(zone *TimeZone) {
	ReferenceRequireNonNull(calendar)
	ReferenceRequireNonNull(zone)
	calendar.zone = zone
	if calendar.valid {
		calendar.populateFields()
	}
}
func (calendar *GregorianCalendar) SetTimeInMillis(millis int64) {
	ReferenceRequireNonNull(calendar)
	calendar.millis = millis
	calendar.valid = true
	calendar.stamps = [17]int{}
	calendar.populateFields()
}
func (calendar *GregorianCalendar) Get(field int32) int32 {
	ReferenceRequireNonNull(calendar)
	calendarCheckField(field)
	calendar.GetTimeInMillis()
	if field == CalendarWEEK_OF_YEAR || field == CalendarWEEK_OF_MONTH || field == CalendarDAY_OF_WEEK_IN_MONTH {
		panic(NewUnsupportedOperationException("calendar week field: " + calendarFieldNames[field]))
	}
	return calendar.fields[field]
}
func (calendar *GregorianCalendar) fieldOr(field int32, fallback int32) int32 {
	if calendar.stamps[field] == 0 {
		return fallback
	}
	return calendar.fields[field]
}
func (calendar *GregorianCalendar) GetTimeInMillis() int64 {
	ReferenceRequireNonNull(calendar)
	if calendar.valid {
		return calendar.millis
	}
	original := calendar.fields
	if !calendar.lenient {
		maximum := [17]int32{1, 292278994, 11, 53, 6, 31, 366, 7, 6, 1, 11, 23, 59, 59, 999, 14 * 3600000, 2 * 3600000}
		minimum := [17]int32{0, 1, 0, 1, 0, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, -13 * 3600000, 0}
		for field, value := range calendar.fields {
			if calendar.stamps[field] >= 2 && (value < minimum[field] || value > maximum[field]) {
				panic(NewIllegalArgumentException(calendarFieldNames[field]))
			}
		}
	}
	year := int64(calendar.fieldOr(CalendarYEAR, 1970))
	if calendar.fieldOr(CalendarERA, 1) == 0 {
		year = 1 - year
	}
	month := int64(calendar.fieldOr(CalendarMONTH, 0))
	year += calendarFloorDiv(month, 12)
	month = calendarFloorMod(month, 12) + 1
	day := int64(calendar.fieldOr(CalendarDAY_OF_MONTH, 1))
	// A civil day in the 1582 cutover gap has no strict-calendar instant.
	if !calendar.lenient && year == 1582 && month == 10 && day >= 5 && day <= 14 {
		panic(NewIllegalArgumentException("the specified date doesn't exist"))
	}
	gregorian := year > 1582 || (year == 1582 && (month > 10 || (month == 10 && day >= 15)))
	fixed := calendarJulianDay(year, month, 1, gregorian) + day - 1
	hour := int64(calendar.fieldOr(CalendarHOUR_OF_DAY, 0))
	if max(calendar.stamps[CalendarHOUR], calendar.stamps[CalendarAM_PM]) > calendar.stamps[CalendarHOUR_OF_DAY] {
		hour = int64(calendar.fieldOr(CalendarHOUR, 0) + 12*calendar.fieldOr(CalendarAM_PM, 0))
	}
	wall := (fixed-2440588)*86400000 + hour*3600000 + int64(calendar.fieldOr(CalendarMINUTE, 0))*60000 + int64(calendar.fieldOr(CalendarSECOND, 0))*1000 + int64(calendar.fieldOr(CalendarMILLISECOND, 0))
	millis := calendar.zone.resolveWallMillis(wall)
	calendar.millis = millis
	calendar.populateFields()
	if !calendar.lenient {
		for field, value := range original {
			if calendar.stamps[field] >= 2 && calendar.fields[field] != value {
				normalized := calendar.fields[field]
				calendar.fields = original
				panic(NewIllegalArgumentException(fmt.Sprintf("%s: %d -> %d", calendarFieldNames[field], value, normalized)))
			}
		}
	}
	calendar.valid = true
	return millis
}
func (zone *TimeZone) resolveWallMillis(wall int64) int64 {
	offsets := map[int32]struct{}{}
	for hours := -48; hours <= 48; hours += 6 {
		offsets[zone.GetOffset(wall+int64(hours)*3600000)] = struct{}{}
	}
	var result int64
	found := false
	gapDelta := int64(0)
	gapResult := int64(0)
	for offset := range offsets {
		candidate := wall - int64(offset)
		delta := int64(zone.GetOffset(candidate)) - int64(offset)
		if delta == 0 {
			if !found || candidate > result {
				result = candidate
				found = true
			}
		}
		if delta > 0 && (gapDelta == 0 || delta < gapDelta) {
			gapDelta = delta
			gapResult = candidate
		}
	}
	if found {
		return result
	}
	if gapDelta != 0 {
		return gapResult
	}
	return wall - int64(zone.GetOffset(wall))
}
func calendarFloorDiv(value, divisor int64) int64 {
	quotient := value / divisor
	if value%divisor < 0 {
		quotient--
	}
	return quotient
}
func calendarFloorMod(value, divisor int64) int64 {
	return value - calendarFloorDiv(value, divisor)*divisor
}
func calendarJulianDay(year, month, day int64, gregorian bool) int64 {
	a := (14 - month) / 12
	y := year + 4800 - a
	m := month + 12*a - 3
	fixed := day + (153*m+2)/5 + 365*y + calendarFloorDiv(y, 4)
	if gregorian {
		return fixed - calendarFloorDiv(y, 100) + calendarFloorDiv(y, 400) - 32045
	}
	return fixed - 32083
}
func calendarCivilFromJulianDay(day int64) (year, month, date int64) {
	if day >= 2299161 {
		instant := time.Unix((day-2440588)*86400, 0).UTC()
		return int64(instant.Year()), int64(instant.Month()), int64(instant.Day())
	}
	c := day + 32082
	d := calendarFloorDiv(4*c+3, 1461)
	e := c - calendarFloorDiv(1461*d, 4)
	m := calendarFloorDiv(5*e+2, 153)
	return d - 4800 + calendarFloorDiv(m, 10), m + 3 - 12*calendarFloorDiv(m, 10), e - calendarFloorDiv(153*m+2, 5) + 1
}
func (calendar *GregorianCalendar) populateFields() {
	offset := int64(calendar.zone.GetOffset(calendar.millis))
	// Keep the offset remainder and its carry without overflowing millis+offset.
	wallRemainder := calendarFloorMod(calendar.millis, 1000) + offset
	seconds := calendarFloorDiv(calendar.millis, 1000) + calendarFloorDiv(wallRemainder, 1000)
	fixed := calendarFloorDiv(seconds, 86400) + 2440588
	year, month, day := calendarCivilFromJulianDay(fixed)
	era := int32(1)
	displayYear := year
	if year <= 0 {
		era = 0
		displayYear = 1 - year
	}
	within := calendarFloorMod(seconds, 86400)
	calendar.fields[CalendarERA] = era
	calendar.fields[CalendarYEAR] = int32(displayYear)
	calendar.fields[CalendarMONTH] = int32(month - 1)
	calendar.fields[CalendarDAY_OF_MONTH] = int32(day)
	calendar.fields[CalendarHOUR_OF_DAY] = int32(within / 3600)
	calendar.fields[CalendarHOUR] = int32(within / 3600 % 12)
	calendar.fields[CalendarAM_PM] = int32(within / 43200)
	calendar.fields[CalendarMINUTE] = int32(within / 60 % 60)
	calendar.fields[CalendarSECOND] = int32(within % 60)
	calendar.fields[CalendarMILLISECOND] = int32(calendarFloorMod(wallRemainder, 1000))
	calendar.fields[CalendarDAY_OF_WEEK] = int32(calendarFloorMod(fixed+1, 7) + 1)
	calendar.fields[CalendarDAY_OF_YEAR] = int32(fixed - calendarJulianDay(year, 1, 1, year > 1582) + 1)
	raw := calendar.zone.GetRawOffset()
	calendar.fields[CalendarZONE_OFFSET] = raw
	calendar.fields[CalendarDST_OFFSET] = int32(offset) - raw
	for i := range calendar.stamps {
		if calendar.stamps[i] == 0 {
			calendar.stamps[i] = 1
		}
	}
}
func init() {
	RegisterJavaType("java.util.Calendar", ObjectTypeID, SerializableTypeID, CloneableTypeID, ComparableTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: "java.util.Calendar"})
	RegisterJavaType("java.util.GregorianCalendar", "java.util.Calendar")
}
