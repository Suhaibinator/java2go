package stdjava

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// These 36 rows are unchanged raw observations from nine fresh JDK21 captures
// of SubsecondOffsetProbe.java, not expectations calculated by the runtime.
func TestCalendarSubsecondRawOffsetCapturedJDK21(t *testing.T) {
	raw, err := os.ReadFile("testdata/calendar_subsecond_offset/expected.stdout")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != "cda63cbc6d1d13ec74e5874ce41dfcb8d8cfd53be78695814d09972a46b0e440" {
		t.Fatal("actual JDK21 capture changed")
	}
	rows := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(rows) != 36 {
		t.Fatal("capture row inventory changed")
	}
	integer := func(current *testing.T, text string) int64 {
		current.Helper()
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			current.Fatal(err)
		}
		return value
	}
	fields := []int32{CalendarYEAR, CalendarMONTH, CalendarDAY_OF_MONTH, CalendarHOUR_OF_DAY, CalendarMINUTE, CalendarSECOND, CalendarMILLISECOND, CalendarZONE_OFFSET, CalendarDST_OFFSET}
	for _, row := range rows {
		values := map[string]string{}
		for _, part := range strings.Split(row, "|") {
			key, value, ok := strings.Cut(part, "=")
			if !ok {
				t.Fatal("invalid original capture")
			}
			values[key] = value
		}
		t.Run("case"+values["case"], func(t *testing.T) {
			offset, instant := int32(integer(t, values["offset"])), integer(t, values["instant"])
			zone := TimeZoneGetTimeZone("UTC")
			zone.SetRawOffset(offset)
			calendar := NewGregorianCalendar(zone, LocaleUS)
			calendar.SetTimeInMillis(instant)
			expectedFields := strings.Split(values["fields"], ",")
			if len(expectedFields) != len(fields) {
				t.Fatal("captured fields changed")
			}
			for index, field := range fields {
				if got, want := int64(calendar.Get(field)), integer(t, expectedFields[index]); got != want {
					t.Errorf("Calendar %s=%d want captured %d", calendarFieldNames[field], got, want)
				}
			}
			formatter := NewSimpleDateFormat("yyyy-MM-dd HH:mm:ss.SSS", LocaleUS)
			formatter.SetTimeZone(zone)
			formatter.SetLenient(false)
			text := formatter.Format(NewDate(instant))
			if text != values["text"] {
				t.Errorf("format %q want captured %q", text, values["text"])
			}
			position := NewParsePosition(0)
			position.SetErrorIndex(7)
			parsed := formatter.Parse(text, position)
			if parsed == nil {
				t.Error("strict roundtrip parse returned null")
			} else if got, want := parsed.GetTime(), integer(t, values["parsed"]); got != want {
				t.Errorf("strict roundtrip instant %d want captured %d", got, want)
			}
			if int64(position.GetIndex()) != integer(t, values["index"]) || int64(position.GetErrorIndex()) != integer(t, values["error"]) {
				t.Errorf("parse position index=%d/error=%d want captured %s/%s", position.GetIndex(), position.GetErrorIndex(), values["index"], values["error"])
			}
			if int64(zone.GetRawOffset()) != integer(t, values["raw"]) || int64(zone.GetOffset(instant)) != integer(t, values["total"]) {
				t.Error("mutable zone raw/total offset differs from actual capture")
			}
		})
	}
}
