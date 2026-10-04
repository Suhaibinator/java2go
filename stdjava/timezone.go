package stdjava

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
)

// TimeZone keeps the Java-visible ID separate from the transition rules: setID
// changes the label, not the UTC offset or daylight-saving rules.
type TimeZone struct {
	id            string
	javaID        *JavaString
	location      *time.Location
	rawOffsetDiff int32
}

func (*TimeZone) JavaDynamicTypeID() TypeID { return "java.util.TimeZone" }
func (zone *TimeZone) GetID() string {
	ReferenceRequireNonNull(zone)
	return dateFormatNativeText(zone.javaID)
}
func (zone *TimeZone) SetID(id string) {
	ReferenceRequireNonNull(zone)
	StringRequireNonNull(id)
	zone.id = id
	zone.javaID = dateFormatNativeJavaString(id)
}
func (zone *TimeZone) Clone() *TimeZone { ReferenceRequireNonNull(zone); copy := *zone; return &copy }
func (zone *TimeZone) GetOffset(millis int64) int32 {
	ReferenceRequireNonNull(zone)
	// The legacy JDK ZoneInfo transition table starts at 1900; earlier
	// instants use the current raw offset rather than IANA local-mean time.
	if millis-int64(zone.rawOffsetDiff) < -2208988800000 {
		return zone.GetRawOffset()
	}
	_, offset := time.UnixMilli(millis - int64(zone.rawOffsetDiff)).In(zone.location).Zone()
	return int32(offset*1000) + zone.rawOffsetDiff
}
func (zone *TimeZone) GetRawOffset() int32 {
	ReferenceRequireNonNull(zone)
	now := time.Now().In(zone.location)
	// ZoneInfo's raw offset is the current standard offset, not the offset of
	// an arbitrary supplied historical instant. Search the current annual rules
	// for a non-DST instant (including southern-hemisphere/negative-DST zones).
	for month := 0; month < 12; month++ {
		value := now.AddDate(0, month, 0)
		if !value.IsDST() {
			_, offset := value.Zone()
			return int32(offset*1000) + zone.rawOffsetDiff
		}
	}
	_, offset := now.Zone()
	return int32(offset*1000) + zone.rawOffsetDiff
}
func TimeZoneGetTimeZone(id string) *TimeZone {
	StringRequireNonNull(id)
	if strings.HasPrefix(id, "GMT") && len(id) > 3 && (id[3] == '+' || id[3] == '-') {
		if zone := customGMTTimeZone(id); zone != nil {
			return zone
		}
		return newTimeZoneWithID("GMT", time.UTC)
	}
	// Java's legacy short IDs are not all present in the IANA database.
	aliases := map[string]string{"ACT": "Australia/Darwin", "AET": "Australia/Sydney", "AGT": "America/Argentina/Buenos_Aires", "ART": "Africa/Cairo", "AST": "America/Anchorage", "BET": "America/Sao_Paulo", "BST": "Asia/Dhaka", "CAT": "Africa/Harare", "CNT": "America/St_Johns", "CST": "America/Chicago", "CTT": "Asia/Shanghai", "EAT": "Africa/Addis_Ababa", "ECT": "Europe/Paris", "IET": "America/Indiana/Indianapolis", "IST": "Asia/Kolkata", "JST": "Asia/Tokyo", "MIT": "Pacific/Apia", "NET": "Asia/Yerevan", "NST": "Pacific/Auckland", "PLT": "Asia/Karachi", "PNT": "America/Phoenix", "PRT": "America/Puerto_Rico", "PST": "America/Los_Angeles", "SST": "Pacific/Guadalcanal", "VST": "Asia/Ho_Chi_Minh"}
	name := id
	if alias, ok := aliases[id]; ok {
		name = alias
	}
	if id == "GMT" || id == "UTC" {
		return newTimeZoneWithID(id, time.UTC)
	}
	if location, err := time.LoadLocation(name); err == nil && id != "Local" && id != "" {
		return newTimeZoneWithID(id, location)
	}
	return newTimeZoneWithID("GMT", time.UTC)
}
func customGMTTimeZone(id string) *TimeZone {
	if len(id) < 5 || (id[3] != '+' && id[3] != '-') {
		return nil
	}
	digits := id[4:]
	hours, minutes := 0, 0
	parts := strings.Split(digits, ":")
	if len(parts) == 2 {
		if len(parts[0]) < 1 || len(parts[0]) > 2 || len(parts[1]) != 2 {
			return nil
		}
	} else if len(parts) == 1 {
		if len(digits) > 4 || len(digits) == 0 {
			return nil
		}
		if len(digits) > 2 {
			parts = []string{digits[:len(digits)-2], digits[len(digits)-2:]}
		}
	} else {
		return nil
	}
	for _, part := range parts {
		for _, digit := range part {
			if digit < '0' || digit > '9' {
				return nil
			}
		}
	}
	hours, _ = strconv.Atoi(parts[0])
	if len(parts) == 2 {
		minutes, _ = strconv.Atoi(parts[1])
	}
	if hours > 23 || minutes > 59 {
		return nil
	}
	offset := (hours*60 + minutes) * 60
	if id[3] == '-' {
		offset = -offset
	}
	canonical := fmt.Sprintf("GMT%c%02d:%02d", id[3], hours, minutes)
	return newTimeZoneWithID(canonical, time.FixedZone(canonical, offset))
}

var defaultTimeZoneState struct {
	sync.RWMutex
	zone *TimeZone
}

func systemTimeZone() *TimeZone {
	id := strings.TrimPrefix(os.Getenv("TZ"), ":")
	if id == "" {
		if path, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
			if index := strings.Index(path, "/zoneinfo/"); index >= 0 {
				id = path[index+10:]
			}
		}
	}
	if id != "" {
		return TimeZoneGetTimeZone(id)
	}
	return newTimeZoneWithID(time.Local.String(), time.Local)
}
func TimeZoneGetDefault() *TimeZone {
	defaultTimeZoneState.Lock()
	defer defaultTimeZoneState.Unlock()
	if defaultTimeZoneState.zone == nil {
		defaultTimeZoneState.zone = systemTimeZone()
	}
	return defaultTimeZoneState.zone.Clone()
}
func TimeZoneSetDefault(zone *TimeZone) {
	defaultTimeZoneState.Lock()
	defer defaultTimeZoneState.Unlock()
	if zone == nil {
		defaultTimeZoneState.zone = nil
	} else {
		defaultTimeZoneState.zone = zone.Clone()
	}
}
func init() {
	RegisterJavaType("java.util.TimeZone", ObjectTypeID, SerializableTypeID, CloneableTypeID)
}

func newTimeZoneWithID(id string, location *time.Location) *TimeZone {
	return &TimeZone{id: id, javaID: dateFormatNativeJavaString(id), location: location}
}
func (zone *TimeZone) GetIDJavaString() *JavaString {
	ReferenceRequireNonNull(zone)
	return zone.javaID
}
func (zone *TimeZone) SetIDJavaString(id *JavaString) {
	ReferenceRequireNonNull(zone)
	ReferenceRequireNonNull(id)
	// Lookup names are scalar metadata; the Java-visible label retains all units.
	zone.id, _ = dateFormatScalarLookupText(id)
	zone.javaID = id
}
func TimeZoneGetTimeZoneJavaString(id *JavaString) *TimeZone {
	ReferenceRequireNonNull(id)
	text, scalar := dateFormatScalarLookupText(id)
	if !scalar {
		// Such units cannot name an IANA or custom GMT zone. Retain the existing
		// unknown-zone GMT result without introducing replacement characters.
		return TimeZoneGetTimeZone("")
	}
	return TimeZoneGetTimeZone(text)
}

func (zone *TimeZone) SetRawOffset(offset int32) {
	ReferenceRequireNonNull(zone)
	zone.rawOffsetDiff += offset - zone.GetRawOffset()
}
func (zone *TimeZone) HasSameRules(other *TimeZone) bool {
	ReferenceRequireNonNull(zone)
	if other == nil || zone.GetRawOffset() != other.GetRawOffset() {
		return false
	}
	if zone.location == other.location {
		return true
	}
	// ZoneInfo rule identity excludes labels and covers its 1900..2037 table.
	// Compare transitions through public zone bounds, including daylight flags.
	at := time.Unix(-2208988800, 0)
	end := time.Unix(2145916800, 0)
	for at.Before(end) {
		left, right := at.In(zone.location), at.In(other.location)
		_, lo := left.Zone()
		_, ro := right.Zone()
		if int32(lo*1000)+zone.rawOffsetDiff != int32(ro*1000)+other.rawOffsetDiff || left.IsDST() != right.IsDST() {
			return false
		}
		_, le := left.ZoneBounds()
		_, re := right.ZoneBounds()
		next := end
		if !le.IsZero() && le.Before(next) {
			next = le
		}
		if !re.IsZero() && re.Before(next) {
			next = re
		}
		if !next.After(at) {
			return false
		}
		at = next
	}
	return true
}
