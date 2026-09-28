package stdjava

import "time"

// Date stores the Java millisecond instant independently of calendar and zone.
type Date struct{ millis int64 }

func NewDate(millis ...int64) *Date {
	if len(millis) == 0 {
		return &Date{time.Now().UnixMilli()}
	}
	return &Date{millis[0]}
}
func (date *Date) GetTime() int64       { ReferenceRequireNonNull(date); return date.millis }
func (date *Date) SetTime(millis int64) { ReferenceRequireNonNull(date); date.millis = millis }
func (date *Date) Equals(other any) bool {
	ReferenceRequireNonNull(date)
	value, ok := other.(*Date)
	return ok && value != nil && date.millis == value.millis
}
func (date *Date) HashCode() int32 {
	value := date.GetTime()
	return int32(uint64(value) ^ uint64(value)>>32)
}
func (date *Date) CompareTo(other *Date) int32 {
	left, right := date.GetTime(), other.GetTime()
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
func (*Date) JavaDynamicTypeID() TypeID { return "java.util.Date" }
func init() {
	RegisterJavaType("java.util.Date", ObjectTypeID, SerializableTypeID, CloneableTypeID, ComparableTypeID)
}
