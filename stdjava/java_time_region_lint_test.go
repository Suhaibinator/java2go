package stdjava

import (
	"fmt"
	"testing"
)

// Expected region transitions are the unchanged JDK21 Time14 seed17 oracle's
// gap, overlap1, and overlap2 records; the seed41/97 records are identical.
func TestJavaTimeRegionArchiveLoadCapturedTransitions(t *testing.T) {
	const id = "America/New_York"
	previous, existed := javaTimeLocations.Load(id)
	javaTimeLocations.Delete(id)
	t.Cleanup(func() {
		if existed {
			javaTimeLocations.Store(id, previous)
		} else {
			javaTimeLocations.Delete(id)
		}
	})
	zone := ZoneIdOf(JavaStringFromHostUTF8(id))
	for _, check := range []struct {
		name   string
		local  *LocalDateTime
		offset int32
		want   string
		withID bool
	}{
		{"gap", LocalDateTimeOf(LocalDateOf(2024, 3, 10), LocalTimeOf(2, 30, 0, 0)), -18000, "2024-03-10T03:30:-14400", false},
		{"overlap1", LocalDateTimeOf(LocalDateOf(2024, 11, 3), LocalTimeOf(1, 30, 0, 17)), -14400, "2024-11-03T01:30:00.000000017:-14400:America/New_York", true},
		{"overlap2", LocalDateTimeOf(LocalDateOf(2024, 11, 3), LocalTimeOf(1, 30, 0, 17)), -18000, "2024-11-03T01:30:00.000000017:-18000:America/New_York", true},
	} {
		t.Run(check.name, func(t *testing.T) {
			zoned := ZonedDateTimeOfInstant(check.local, ZoneOffsetOfTotalSeconds(check.offset), zone)
			got := fmt.Sprintf("%s:%d", zoned.ToLocalDateTime().String(), zoned.GetOffset().GetTotalSeconds())
			if check.withID {
				got += ":" + timeHost(zoned.GetZone().GetID())
			}
			if got != check.want {
				t.Fatalf("captured JDK transition: %q != %q", got, check.want)
			}
		})
	}
}
