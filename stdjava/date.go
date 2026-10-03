package stdjava

import "time"

// DateValue is the Java util.Date reference protocol. SQL subclasses remain
// distinct allocations and retain their dynamic class through this interface.
type DateValue interface {
	JavaDateMarker()
	JavaDynamicTypeID() TypeID
	GetTime() int64
	SetTime(int64)
	Equals(any) bool
	HashCode() int32
	CompareTo(DateValue) int32
}

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
	value, ok := other.(DateValue)
	return ok && !javaReferenceIsNull(value) && date.GetTime() == value.GetTime()
}
func (date *Date) HashCode() int32 {
	value := date.GetTime()
	return int32(uint64(value) ^ uint64(value)>>32)
}
func (date *Date) CompareTo(other DateValue) int32 {
	ReferenceRequireNonNull(other)
	left, right := date.GetTime(), other.GetTime()
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
func (*Date) JavaDateMarker()           {}
func (*Date) JavaDynamicTypeID() TypeID { return "java.util.Date" }
func init() {
	RegisterJavaType("java.util.Date", ObjectTypeID, SerializableTypeID, CloneableTypeID, ComparableTypeID)
}
