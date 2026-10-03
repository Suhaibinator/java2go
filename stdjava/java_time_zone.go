package stdjava

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Region rules use a frozen platform TZif archive, independent of host TZ and
// ZONEINFO settings. Offset and region references retain their nominal identity.
//
//go:embed java_time_zoneinfo.zip
var javaTimeZoneArchive []byte
var javaTimeLocations sync.Map
var javaTimeOffsets sync.Map

type ZoneId interface {
	GetID() *JavaString
	JavaDynamicTypeID() TypeID
	timeLocation() *time.Location
}
type zoneRegion struct {
	id       *JavaString
	location *time.Location
}
type ZoneOffset struct {
	seconds int32
	id      *JavaString
}
type OffsetDateTime struct {
	local  *LocalDateTime
	offset *ZoneOffset
}
type OffsetTime struct {
	local  *LocalTime
	offset *ZoneOffset
}
type ZonedDateTime struct {
	local  *LocalDateTime
	offset *ZoneOffset
	zone   ZoneId
}

func (o *ZoneOffset) timeLocation() *time.Location {
	timeRequire(o)
	return time.FixedZone(o.String(), int(o.seconds))
}
func (r *zoneRegion) timeLocation() *time.Location { ReferenceRequireNonNull(r); return r.location }
func (o *ZoneOffset) GetID() *JavaString           { return timeRequire(o).id }
func javaTimeNewOffset(seconds int32) *ZoneOffset {
	offset := &ZoneOffset{seconds: seconds}
	offset.id = JavaStringFromHostUTF8(offset.String())
	return offset
}
func (r *zoneRegion) GetID() *JavaString      { ReferenceRequireNonNull(r); return r.id }
func (*ZoneOffset) JavaDynamicTypeID() TypeID { return "java.time.ZoneOffset" }
func (*zoneRegion) JavaDynamicTypeID() TypeID { return "java.time.ZoneRegion" }
func (o *ZoneOffset) GetTotalSeconds() int32  { return timeRequire(o).seconds }
func ZoneOffsetOfTotalSeconds(seconds int32) *ZoneOffset {
	if seconds < -64800 || seconds > 64800 {
		panic(NewDateTimeException("Zone offset not in valid range: -18:00 to +18:00"))
	}
	if seconds%900 == 0 {
		if cached, ok := javaTimeOffsets.Load(seconds); ok {
			return cached.(*ZoneOffset)
		}
		v, _ := javaTimeOffsets.LoadOrStore(seconds, javaTimeNewOffset(seconds))
		return v.(*ZoneOffset)
	}
	return javaTimeNewOffset(seconds)
}
func (o *ZoneOffset) Equals(candidate any) bool {
	timeRequire(o)
	other, ok := candidate.(*ZoneOffset)
	return ok && other != nil && o.seconds == other.seconds
}
func (o *ZoneOffset) HashCode() int32 { return timeRequire(o).seconds }
func (o *ZoneOffset) String() string {
	timeRequire(o)
	if o.seconds == 0 {
		return "Z"
	}
	seconds := o.seconds
	sign := "+"
	if seconds < 0 {
		sign = "-"
		seconds = -seconds
	}
	out := fmt.Sprintf("%s%02d:%02d", sign, seconds/3600, (seconds/60)%60)
	if seconds%60 != 0 {
		out += fmt.Sprintf(":%02d", seconds%60)
	}
	return out
}
func (o *ZoneOffset) StringJava2goExecution(*Execution) *JavaString { return o.GetID() }
func (r *zoneRegion) StringJava2goExecution(*Execution) *JavaString { return r.GetID() }
func (r *zoneRegion) Equals(candidate any) bool {
	ReferenceRequireNonNull(r)
	other, ok := candidate.(ZoneId)
	return ok && !javaReferenceIsNull(other) && r.id.Equals(other.GetID())
}
func (r *zoneRegion) HashCode() int32 { ReferenceRequireNonNull(r); return r.id.HashCode() }
func javaTimeLoadZone(id string) *time.Location {
	if cached, ok := javaTimeLocations.Load(id); ok {
		return cached.(*time.Location)
	}
	archive, err := zip.NewReader(bytes.NewReader(javaTimeZoneArchive), int64(len(javaTimeZoneArchive)))
	if err != nil {
		panic(NewZoneRulesException("Invalid platform time-zone database"))
	}
	for _, entry := range archive.File {
		if entry.Name != id {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			break
		}
		payload, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil || closeErr != nil {
			break
		}
		location, err := time.LoadLocationFromTZData(id, payload)
		if err != nil {
			break
		}
		actual, _ := javaTimeLocations.LoadOrStore(id, location)
		return actual.(*time.Location)
	}
	panic(NewZoneRulesException("Unknown time-zone ID: " + id))
}
func javaTimeParseOffset(id string) *ZoneOffset {
	if id == "Z" {
		return ZoneOffsetOfTotalSeconds(0)
	}
	if len(id) < 2 || (id[0] != '+' && id[0] != '-') {
		panic(NewDateTimeException("Invalid ID for ZoneOffset, invalid format: " + id))
	}
	digits := strings.ReplaceAll(id[1:], ":", "")
	if len(digits) == 1 {
		digits = "0" + digits
	}
	if len(digits) != 2 && len(digits) != 4 && len(digits) != 6 {
		panic(NewDateTimeException("Invalid ID for ZoneOffset, invalid format: " + id))
	}
	hours, err := strconv.Atoi(digits[:2])
	if err != nil {
		panic(NewDateTimeException("Invalid ID for ZoneOffset, non numeric characters found: " + id))
	}
	minutes, seconds := 0, 0
	if len(digits) >= 4 {
		minutes, err = strconv.Atoi(digits[2:4])
		if err != nil {
			panic(NewDateTimeException("Invalid ID for ZoneOffset, non numeric characters found: " + id))
		}
	}
	if len(digits) == 6 {
		seconds, err = strconv.Atoi(digits[4:])
		if err != nil {
			panic(NewDateTimeException("Invalid ID for ZoneOffset, non numeric characters found: " + id))
		}
	}
	if minutes > 59 || seconds > 59 {
		panic(NewDateTimeException("Zone offset minutes and seconds not in valid range: value " + id))
	}
	total := hours*3600 + minutes*60 + seconds
	if id[0] == '-' {
		total = -total
	}
	return ZoneOffsetOfTotalSeconds(int32(total))
}

var javaTimeRegionPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9~/._+\-]+$`)

func ZoneIdOf(value *JavaString) ZoneId {
	id := timeHost(value)
	if id == "Z" || strings.HasPrefix(id, "+") || strings.HasPrefix(id, "-") {
		return javaTimeParseOffset(id)
	}
	for _, prefix := range []string{"UTC", "GMT", "UT"} {
		if id == prefix {
			return &zoneRegion{value, time.UTC}
		}
		if strings.HasPrefix(id, prefix+"+") || strings.HasPrefix(id, prefix+"-") {
			offset := javaTimeParseOffset(id[len(prefix):])
			normalized := prefix
			if offset.seconds != 0 {
				normalized += offset.String()
			}
			return &zoneRegion{JavaStringFromHostUTF8(normalized), offset.timeLocation()}
		}
	}
	if !javaTimeRegionPattern.MatchString(id) {
		panic(NewDateTimeException("Invalid ID for region-based ZoneId, invalid format: " + id))
	}
	return &zoneRegion{value, javaTimeLoadZone(id)}
}
func OffsetDateTimeOf(local *LocalDateTime, offset *ZoneOffset) *OffsetDateTime {
	return &OffsetDateTime{timeRequire(local), timeRequire(offset)}
}
func OffsetTimeOf(local *LocalTime, offset *ZoneOffset) *OffsetTime {
	return &OffsetTime{timeRequire(local), timeRequire(offset)}
}
func ZonedDateTimeOfInstant(local *LocalDateTime, offset *ZoneOffset, zone ZoneId) *ZonedDateTime {
	timeRequire(local)
	timeRequire(offset)
	ReferenceRequireNonNull(zone)
	instant := local.timeUTC().Add(-time.Duration(offset.seconds) * time.Second)
	destination := instant.In(zone.timeLocation())
	_, seconds := destination.Zone()
	if int32(seconds) == offset.seconds {
		return &ZonedDateTime{local, offset, zone}
	}
	return &ZonedDateTime{localDateTimeFromHost(destination), ZoneOffsetOfTotalSeconds(int32(seconds)), zone}
}
func (o *OffsetDateTime) ToLocalDateTime() *LocalDateTime { return timeRequire(o).local }
func (o *OffsetDateTime) GetOffset() *ZoneOffset          { return timeRequire(o).offset }
func (o *OffsetTime) ToLocalTime() *LocalTime             { return timeRequire(o).local }
func (o *OffsetTime) GetOffset() *ZoneOffset              { return timeRequire(o).offset }
func (z *ZonedDateTime) ToLocalDateTime() *LocalDateTime  { return timeRequire(z).local }
func (z *ZonedDateTime) GetOffset() *ZoneOffset           { return timeRequire(z).offset }
func (z *ZonedDateTime) GetZone() ZoneId                  { return timeRequire(z).zone }
func (o *OffsetDateTime) String() string                  { timeRequire(o); return o.local.String() + o.offset.String() }
func (o *OffsetTime) String() string                      { timeRequire(o); return o.local.String() + o.offset.String() }
func (z *ZonedDateTime) String() string {
	timeRequire(z)
	out := z.local.String() + z.offset.String()
	if _, offset := z.zone.(*ZoneOffset); !offset {
		out += "[" + timeHost(z.zone.GetID()) + "]"
	}
	return out
}
func (o *OffsetDateTime) Equals(candidate any) bool {
	timeRequire(o)
	other, ok := candidate.(*OffsetDateTime)
	return ok && other != nil && o.local.Equals(other.local) && o.offset.Equals(other.offset)
}
func (o *OffsetTime) Equals(candidate any) bool {
	timeRequire(o)
	other, ok := candidate.(*OffsetTime)
	return ok && other != nil && o.local.Equals(other.local) && o.offset.Equals(other.offset)
}
func (z *ZonedDateTime) Equals(candidate any) bool {
	timeRequire(z)
	other, ok := candidate.(*ZonedDateTime)
	return ok && other != nil && z.local.Equals(other.local) && z.offset.Equals(other.offset) && z.zone.GetID().Equals(other.zone.GetID())
}
func (*OffsetDateTime) JavaDynamicTypeID() TypeID { return "java.time.OffsetDateTime" }
func (*OffsetTime) JavaDynamicTypeID() TypeID     { return "java.time.OffsetTime" }
func (*ZonedDateTime) JavaDynamicTypeID() TypeID  { return "java.time.ZonedDateTime" }
func (o *OffsetDateTime) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(o.String())
}
func (o *OffsetTime) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(o.String())
}
func (z *ZonedDateTime) StringJava2goExecution(*Execution) *JavaString {
	return JavaStringFromHostUTF8(z.String())
}
