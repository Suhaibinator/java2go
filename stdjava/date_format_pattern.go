package stdjava

import (
	"strings"
	"time"
)

type dateFormatToken struct {
	field   rune
	count   int
	literal []uint16
}

func compileDateFormatPatternUnits(units []rune) []dateFormatToken {
	var result []dateFormatToken
	quoted := false
	literal := func(unit rune) {
		if len(result) == 0 || result[len(result)-1].field != 0 {
			result = append(result, dateFormatToken{})
		}
		last := len(result) - 1
		result[last].literal = append(result[last].literal, uint16(unit))
	}
	for i := 0; i < len(units); i++ {
		c := units[i]
		if c == '\'' {
			if i+1 < len(units) && units[i+1] == '\'' {
				literal('\'')
				i++
				continue
			}
			quoted = !quoted
			continue
		}
		if quoted || (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') {
			literal(c)
			continue
		}
		if !strings.ContainsRune("GyMdkHmsSEDFwWahKzZYuXL", c) {
			panic(NewIllegalArgumentException("Illegal pattern character '" + string(c) + "'"))
		}
		count := 1
		for i+1 < len(units) && units[i+1] == c {
			count++
			i++
		}
		if c == 'X' && count > 3 {
			panic(NewIllegalArgumentException("invalid ISO 8601 format: length=" + dateFormatNumber(int32(count), 1)))
		}
		result = append(result, dateFormatToken{field: c, count: count})
	}
	if quoted {
		panic(NewIllegalArgumentException("Unterminated quote"))
	}
	return result
}
func dateFormatNumeric(token dateFormatToken) bool {
	switch token.field {
	case 'M', 'L':
		return token.count <= 2
	case 'y', 'd', 'D', 'H', 'k', 'h', 'K', 'm', 's', 'S', 'u', 'Y', 'w', 'W', 'F':
		return true
	}
	return false
}
func dateFormatParseNumber(units []rune, start, limit int) (int32, int, bool) {
	end := len(units)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	index := start
	for index < end && (units[index] == ' ' || units[index] == '\t') {
		index++
	}
	negative := false
	if index < end && units[index] == '-' {
		negative = true
		index++
	}
	first := index
	var value int64
	for index < end {
		digit := CharDigit(units[index], 10)
		if digit < 0 {
			break
		}
		value = value*10 + int64(digit)
		index++
	}
	if index == first {
		return 0, start, false
	}
	if negative {
		value = -value
	}
	return int32(value), index, true
}
func dateFormatParseText(units []rune, start int, values []string) (int32, int, bool) {
	best := start
	found := int32(-1)
	for i, text := range values {
		if end, ok := dateFormatMatch(units, start, text); ok && end > best {
			best = end
			found = int32(i)
		}
	}
	return found, best, found >= 0
}

// parsePositionUnits preserves Java's lexical and calendar failure boundaries. In
// particular, a successful parse does not clear a previously set errorIndex.
func (f *SimpleDateFormat) parsePositionUnits(units []rune, pos *ParsePosition) (result *Date) {
	start := int(pos.index)
	index := start
	if start < 0 {
		StringCharAt("", int32(start))
	}
	fail := func(at int) *Date { pos.errorIndex = int32(at); pos.index = int32(start); return nil }
	type fieldValue struct{ field, value int32 }
	var fields []fieldValue
	var forcedOffset *int32
	var ambiguousYear bool
	for n, token := range f.tokens {
		if token.field == 0 {
			for _, literal := range token.literal {
				if index >= len(units) || units[index] != rune(literal) {
					return fail(index)
				}
				index++
			}
			continue
		}
		count := 0
		if n+1 < len(f.tokens) && dateFormatNumeric(f.tokens[n+1]) {
			count = token.count
		}
		field := int32(-1)
		var value int32
		var end int
		var ok bool
		switch token.field {
		case 'G':
			value, end, ok = dateFormatParseText(units, index, f.symbols.eras[:])
			field = CalendarERA
		case 'M', 'L':
			if token.count >= 3 {
				value, end, ok = dateFormatParseText(units, index, f.symbols.months[:])
				v, e, o := dateFormatParseText(units, index, f.symbols.shortMonths[:])
				if o && (!ok || e > end) {
					value, end, ok = v, e, o
				}
			} else {
				value, end, ok = dateFormatParseNumber(units, index, count)
				value--
			}
			field = CalendarMONTH
		case 'E':
			value, end, ok = dateFormatParseText(units, index, f.symbols.days[:])
			_, e, o := dateFormatParseText(units, index, f.symbols.shortDays[:])
			if o && (!ok || e > end) {
				end, ok = e, o
			}
			// Explicit calendar dates select the day; weekday is descriptive when
			// year/month/day are all present. Week-only selection remains unsupported.
		case 'a':
			value, end, ok = dateFormatParseText(units, index, f.symbols.ampm[:])
			field = CalendarAM_PM
		case 'z', 'Z', 'X':
			var zone *TimeZone
			var offset *int32
			zone, offset, end, ok = f.parseZone(units, index, token)
			if ok {
				if zone != nil {
					f.calendar.SetTimeZone(zone)
				}
				forcedOffset = offset
			}
		default:
			if !dateFormatNumeric(token) {
				panic(NewUnsupportedOperationException("date pattern field: " + string(token.field)))
			}
			value, end, ok = dateFormatParseNumber(units, index, count)
			switch token.field {
			case 'y':
				field = CalendarYEAR
				if token.count <= 2 && end-index == 2 && index >= 0 && index+1 < len(units) && CharDigit(units[index], 10) >= 0 && CharDigit(units[index+1], 10) >= 0 {
					century := f.centuryYear / 100 * 100
					boundary := f.centuryYear % 100
					ambiguousYear = value == boundary
					if value < boundary {
						century += 100
					}
					value += century
				}
			case 'd':
				field = CalendarDAY_OF_MONTH
			case 'H':
				field = CalendarHOUR_OF_DAY
			case 'k':
				field = CalendarHOUR_OF_DAY
				if value == 24 {
					value = 0
				}
			case 'K':
				field = CalendarHOUR
			case 'h':
				field = CalendarHOUR
				if value == 12 {
					value = 0
				}
			case 'm':
				field = CalendarMINUTE
			case 's':
				field = CalendarSECOND
			case 'S':
				field = CalendarMILLISECOND
			default:
				panic(NewUnsupportedOperationException("calendar selection for date pattern: " + string(token.field)))
			}
		}
		if !ok {
			return fail(index)
		}
		index = end
		if field >= 0 {
			fields = append(fields, fieldValue{field, value})
		}
	}
	pos.index = int32(index)
	defer func() {
		if recovered := recover(); recovered != nil {
			if CaughtAs(recovered, "IllegalArgumentException") {
				result = fail(index)
			} else {
				panic(recovered)
			}
		}
	}()
	ReferenceRequireNonNull(f.calendar.zone)
	f.calendar.Clear()
	for _, entry := range fields {
		f.calendar.Set(entry.field, entry.value)
	}
	date := f.calendar.GetTime()
	if ambiguousYear && date.GetTime() < f.centuryStart {
		f.calendar.Set(CalendarYEAR, f.calendar.Get(CalendarYEAR)+100)
		date = f.calendar.GetTime()
	}
	if forcedOffset != nil {
		millis := date.GetTime()
		date = NewDate(millis + int64(f.calendar.zone.GetOffset(millis)-*forcedOffset))
		f.calendar.SetTime(date)
	}
	return date
}
func (f *SimpleDateFormat) parseZone(units []rune, start int, token dateFormatToken) (*TimeZone, *int32, int, bool) {
	index := start
	if index >= len(units) {
		return nil, nil, start, false
	}
	iso := token.field == 'X'
	if iso && units[index] == 'Z' {
		offset := int32(0)
		return nil, &offset, index + 1, true
	}
	if !iso {
		if end, ok := dateFormatMatch(units, index, "GMT"); ok {
			index = end
			if index >= len(units) || (units[index] != '+' && units[index] != '-') {
				offset := int32(0)
				return nil, &offset, index, true
			}
		}
	}
	if index < len(units) && (units[index] == '+' || units[index] == '-') {
		sign := int32(1)
		if units[index] == '-' {
			sign = -1
		}
		index++
		digit := func() (int32, bool) {
			if index >= len(units) || units[index] < '0' || units[index] > '9' {
				return 0, false
			}
			v := int32(units[index] - '0')
			index++
			return v, true
		}
		a, ok := digit()
		if !ok {
			return nil, nil, start, false
		}
		b, hasSecond := digit()
		var hours int32
		if hasSecond {
			hours = a*10 + b
		} else {
			hours = a
		}
		minutes := int32(0)
		gmt := start+3 <= len(units) && strings.EqualFold(string(units[start:start+3]), "GMT")
		if gmt || iso && token.count == 3 {
			if index >= len(units) || units[index] != ':' {
				return nil, nil, start, false
			}
			index++
		}
		if gmt || !iso || token.count >= 2 {
			a, ok = digit()
			if !ok {
				return nil, nil, start, false
			}
			b, ok = digit()
			if !ok {
				return nil, nil, start, false
			}
			minutes = a*10 + b
		}
		if !gmt && !hasSecond || hours > 23 || minutes > 59 {
			return nil, nil, start, false
		}
		offset := sign * (hours*3600000 + minutes*60000)
		return nil, &offset, index, true
	}
	if iso {
		return nil, nil, start, false
	}
	// Match current zone first, then the default zone, then provider order, as
	// SimpleDateFormat does for ambiguous abbreviations such as CST.
	candidates := make([]dateFormatZoneData, 0, len(dateFormatEnglishZones)+2)
	for _, id := range []string{f.calendar.zone.id, TimeZoneGetDefault().id} {
		for _, data := range dateFormatEnglishZones {
			if data.id == id {
				candidates = append(candidates, data)
				break
			}
		}
	}
	candidates = append(candidates, dateFormatEnglishZones...)
	for _, data := range candidates {
		for kind, name := range data.names[:4] {
			if end, ok := dateFormatMatch(units, start, name); ok {
				zone := TimeZoneGetTimeZone(data.id)
				offset := zone.GetRawOffset()
				if kind >= 2 { // use the zone's actual daylight saving amount, not a universal hour.
					now := time.Now().In(zone.location)
					for m := 0; m < 12; m++ {
						probe := now.AddDate(0, m, 0)
						if probe.IsDST() {
							_, seconds := probe.Zone()
							offset = int32(seconds * 1000)
							break
						}
					}
				}
				return zone, &offset, end, true
			}
		}
	}
	return nil, nil, start, false
}
