package stdjava

import "testing"

// These expectations are selected observations from the independent, unchanged
// DateFormatSequence JVM oracle c908a05ba443907d5b6e2965bcce78df56169b8012a1650b9406129a3902478c.
func TestDateFormatCampaignCapturedDefaultsAndRestore(t *testing.T) {
	previousLocale, previousZone := LocaleGetDefault(), TimeZoneGetDefault()
	defer func() { LocaleSetDefault(previousLocale); TimeZoneSetDefault(previousZone) }()
	LocaleSetDefault(LocaleUS)
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	first := NewSimpleDateFormat("yyyy-MM-dd HH:mm:ss z", LocaleUS)
	TimeZoneSetDefault(TimeZoneGetTimeZone("GMT+05:30"))
	second := NewSimpleDateFormat("yyyy-MM-dd HH:mm:ss z", LocaleUS)
	if got := first.Format(NewDate(0)) + "|" + second.Format(NewDate(0)); got != "1970-01-01 00:00:00 UTC|1970-01-01 05:30:00 GMT+05:30" {
		t.Fatal(got)
	}
	saved := first.GetTimeZone()
	if got := first.Parse("1970-01-01 00:00:00 PST").GetTime(); got != 28800000 {
		t.Fatal(got)
	}
	if got := first.GetTimeZone().GetID(); got != "America/Los_Angeles" {
		t.Fatal(got)
	}
	first.SetTimeZone(saved)
	if got := first.Parse("1970-01-01 00:00:00 UTC trailing").GetTime(); got != 0 {
		t.Fatal(got)
	}
	if got := first.Format(NewDate(0)); got != "1970-01-01 00:00:00 UTC" {
		t.Fatal(got)
	}
}
func TestDateFormatCampaignStyleRoundtrip(t *testing.T) {
	previousLocale, previousZone := LocaleGetDefault(), TimeZoneGetDefault()
	defer func() { LocaleSetDefault(previousLocale); TimeZoneSetDefault(previousZone) }()
	LocaleSetDefault(LocaleUS)
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	cases := []struct {
		style  int32
		text   string
		millis int64
	}{
		{DateFormatSHORT, "2/29/24, 11:59\u202fPM", 1709251140000},
		{DateFormatMEDIUM, "Feb 29, 2024, 11:59:59\u202fPM", 1709251199000},
		{DateFormatLONG, "February 29, 2024, 11:59:59\u202fPM UTC", 1709251199000},
		{DateFormatFULL, "Thursday, February 29, 2024, 11:59:59\u202fPM Coordinated Universal Time", 1709251199000},
	}
	for _, row := range cases {
		format := DateFormatGetDateTimeInstance(row.style, row.style, LocaleUS)
		if got := format.Format(NewDate(1709251199000)); got != row.text {
			t.Errorf("style %d got %q want %q", row.style, got, row.text)
		}
		if got := format.Parse(row.text).GetTime(); got != row.millis {
			t.Errorf("style %d millis %d want %d", row.style, got, row.millis)
		}
	}
}
func TestDateFormatCampaignLenientAndParseError(t *testing.T) {
	previousLocale, previousZone := LocaleGetDefault(), TimeZoneGetDefault()
	defer func() { LocaleSetDefault(previousLocale); TimeZoneSetDefault(previousZone) }()
	LocaleSetDefault(LocaleUS)
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	format := NewSimpleDateFormat("MMM d, yyyy")
	if got := format.Parse("Feb 30, 2020").GetTime(); got != 1583020800000 {
		t.Fatal(got)
	}
	clock := NewSimpleDateFormat("hh:mm:ss a", LocaleUS)
	if got := clock.Parse("12:00:00 AM").GetTime(); got != 0 {
		t.Fatal(got)
	}
	if got := clock.Parse("12:00:00 PM").GetTime(); got != 43200000 {
		t.Fatal(got)
	}
	if got := clock.Format(NewDate(0)); got != "12:00:00 AM" {
		t.Fatal(got)
	}
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("missing ParseException")
		}
		if !CaughtAs(recovered, "ParseException") {
			t.Fatalf("unexpected panic %v", recovered)
		}
		failure := recovered.(ParseException)
		if ParseExceptionErrorOffset(failure) != 0 || failure.Message() != "Unparseable date: \"not a date\"" {
			t.Fatalf("unexpected ParseException: %v", failure)
		}
	}()
	NewSimpleDateFormat("yyyy-MM-dd HH:mm:ss z", LocaleUS).Parse("not a date")
}
