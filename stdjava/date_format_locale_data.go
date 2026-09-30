package stdjava

import "strings"

// English Gregorian symbols and style patterns are from the pinned JDK21 CLDR
// FormatData_en resource (src.zip d887da897fc0065a2532500ee92320ce3bc633ccd6c4e637304045ce5d90de43).
// This first formatter slice supports en and en-US. Other locale resource
// providers need their own data and are rejected rather than formatted as US.
type dateFormatSymbols struct {
	months, shortMonths [12]string
	days, shortDays     [7]string
	ampm, eras          [2]string
	dates, times        [4]string
}

var dateFormatEnglish = dateFormatSymbols{
	months:      [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	shortMonths: [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
	days:        [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
	shortDays:   [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"},
	ampm:        [2]string{"AM", "PM"}, eras: [2]string{"BC", "AD"},
	dates: [4]string{"EEEE, MMMM d, y", "MMMM d, y", "MMM d, y", "M/d/yy"},
	times: [4]string{"h:mm:ss\u202fa zzzz", "h:mm:ss\u202fa z", "h:mm:ss\u202fa", "h:mm\u202fa"},
}

func dateFormatLocaleSymbols(locale *Locale) *dateFormatSymbols {
	ReferenceRequireNonNull(locale)
	name := locale.tag.String()
	if name == "en" || name == "en-US" {
		return &dateFormatEnglish
	}
	panic(NewUnsupportedOperationException("DateFormat locale data: " + name))
}

type dateFormatZoneData struct {
	id    string
	names [6]string
}

func dateFormatZoneName(zone *TimeZone, millis int64, long bool) string {
	index := 1
	if long {
		index = 0
	}
	if dateFormatZoneDaylight(zone, millis) {
		index += 2
	}
	for _, data := range dateFormatEnglishZones {
		if data.id == zone.id && data.names[index] != "" {
			return data.names[index]
		}
	}
	// TimeZone.getDisplayName preserves a custom GMT label even when its
	// mutable raw offset has changed (JDK21 TimeZone.java fallback).
	if strings.HasPrefix(zone.id, "GMT") && len(zone.id) > 3 && (zone.id[3] == '+' || zone.id[3] == '-') {
		return zone.id
	}
	return "GMT" + dateFormatZoneOffset(zone.GetOffset(millis), 'X', 3)
}
func dateFormatZoneDaylight(zone *TimeZone, millis int64) bool {
	return zone.GetOffset(millis) != zone.GetRawOffset()
}
func dateFormatZoneOffset(offset int32, field rune, count int) string {
	if offset == 0 && field == 'X' {
		return "Z"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours, minutes := offset/3600000, (offset/60000)%60
	result := sign + dateFormatNumber(hours, 2)
	if field == 'X' && count == 1 {
		return result
	}
	if field == 'X' && count == 3 {
		result += ":"
	}
	return result + dateFormatNumber(minutes, 2)
}
func dateFormatMatch(units []rune, start int, text string) (int, bool) {
	needle := StringChars(text)
	if len(needle) == 0 || start < 0 || start+len(needle) > len(units) {
		return start, false
	}
	if strings.EqualFold(string(units[start:start+len(needle)]), string(needle)) {
		return start + len(needle), true
	}
	return start, false
}
