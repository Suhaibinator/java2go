package stdjava

import "testing"

func TestTimeZoneMutableFormatterIdentityAndDefaultCopies(t *testing.T) {
	previous := TimeZoneGetDefault()
	t.Cleanup(func() { TimeZoneSetDefault(previous) })
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	format := DateFormatGetDateTimeInstance(DateFormatMEDIUM, DateFormatMEDIUM, LocaleUS)
	passed := TimeZoneGetTimeZone("GMT+02:00")
	format.SetTimeZone(passed)
	value := NewDate(1615703520000)
	if format.GetTimeZone() != passed || format.Format(value) != "Mar 14, 2021, 8:32:00\u202fAM" {
		t.Fatal("formatter lost supplied zone identity")
	}
	passed.SetRawOffset(3 * 3600000)
	if got := format.Format(value); got != "Mar 14, 2021, 9:32:00\u202fAM" {
		t.Fatal(got)
	}
	named := NewSimpleDateFormat("HH:mm z", LocaleUS)
	named.SetTimeZone(passed)
	if got := named.Format(NewDate(0)); got != "03:00 GMT+02:00" {
		t.Fatal("custom display label changed with raw offset: " + got)
	}
	if !passed.HasSameRules(TimeZoneGetTimeZone("GMT+03:00")) {
		t.Fatal("equal fixed rules after raw mutation")
	}
	TimeZoneSetDefault(passed)
	passed.SetRawOffset(4 * 3600000)
	copy := TimeZoneGetDefault()
	if copy.GetRawOffset() != 3*3600000 {
		t.Fatal("setDefault retained mutable argument")
	}
	copy.SetRawOffset(5 * 3600000)
	if TimeZoneGetDefault().GetRawOffset() != 3*3600000 {
		t.Fatal("getDefault exposed mutable stored zone")
	}
}

func TestTimeZoneMutableCloneAndTransitionRules(t *testing.T) {
	ny := TimeZoneGetTimeZone("America/New_York")
	clone := ny.Clone()
	clone.SetID("different label")
	if !ny.HasSameRules(clone) || !ny.HasSameRules(TimeZoneGetTimeZone("US/Eastern")) || ny.HasSameRules(nil) || ny.HasSameRules(TimeZoneGetTimeZone("America/Phoenix")) {
		t.Fatal("rules compared labels or ignored transitions")
	}
	clone.SetRawOffset(-6 * 3600000)
	if ny.GetRawOffset() != -5*3600000 || clone.GetOffset(1625097600000) != -5*3600000 || clone.HasSameRules(ny) {
		t.Fatal("raw offset mutation lost clone independence or DST")
	}
}
