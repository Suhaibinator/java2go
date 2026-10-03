package stdjava

// SQL Date and Time share util.Date millisecond behavior, but they are distinct
// nominal classes with their own textual contracts.
type SQLDate struct{ *Date }
type SQLTime struct{ *Date }
type SQLTimestamp struct {
	*Date
	nanos int32
}

func NewSQLDate(millis int64) *SQLDate { return &SQLDate{NewDate(millis)} }
func NewSQLTime(millis int64) *SQLTime { return &SQLTime{NewDate(millis)} }
func NewSQLTimestamp(millis int64) *SQLTimestamp {
	result := &SQLTimestamp{Date: NewDate(0)}
	result.SetTime(millis)
	return result
}
func (*SQLDate) JavaDynamicTypeID() TypeID      { return "java.sql.Date" }
func (*SQLTime) JavaDynamicTypeID() TypeID      { return "java.sql.Time" }
func (*SQLTimestamp) JavaDynamicTypeID() TypeID { return "java.sql.Timestamp" }
func (stamp *SQLTimestamp) SetTime(millis int64) {
	ReferenceRequireNonNull(stamp)
	stamp.millis = (millis / 1000) * 1000
	stamp.nanos = int32((millis % 1000) * 1000000)
	if stamp.nanos < 0 {
		stamp.nanos += 1000000000
		stamp.millis = ((millis / 1000) - 1) * 1000
	}
}
func (stamp *SQLTimestamp) GetTime() int64 {
	ReferenceRequireNonNull(stamp)
	return stamp.millis + int64(stamp.nanos/1000000)
}
func (stamp *SQLTimestamp) GetNanos() int32 { ReferenceRequireNonNull(stamp); return stamp.nanos }
func (stamp *SQLTimestamp) SetNanos(nanos int32) {
	ReferenceRequireNonNull(stamp)
	if nanos < 0 || nanos > 999999999 {
		panic(NewIllegalArgumentException("nanos > 999999999 or < 0"))
	}
	stamp.nanos = nanos
}
func (stamp *SQLTimestamp) Equals(value any) bool {
	ReferenceRequireNonNull(stamp)
	other, ok := value.(*SQLTimestamp)
	return ok && other != nil && stamp.GetTime() == other.GetTime() && stamp.nanos == other.nanos
}
func (stamp *SQLTimestamp) HashCode() int32 {
	value := stamp.GetTime()
	return int32(uint64(value) ^ uint64(value)>>32)
}
func (stamp *SQLTimestamp) CompareTo(value DateValue) int32 {
	ReferenceRequireNonNull(stamp)
	ReferenceRequireNonNull(value)
	other, ok := value.(*SQLTimestamp)
	if !ok {
		other = NewSQLTimestamp(value.GetTime())
	}
	left, right := stamp.GetTime(), other.GetTime()
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	if stamp.nanos < other.nanos {
		return -1
	}
	if stamp.nanos > other.nanos {
		return 1
	}
	return 0
}
func init() {
	for _, id := range []TypeID{"java.sql.Date", "java.sql.Time", "java.sql.Timestamp"} {
		RegisterJavaType(id, "java.util.Date")
	}
}

func (value *SQLDate) GetTime() int64 { ReferenceRequireNonNull(value); return value.Date.GetTime() }
func (value *SQLDate) SetTime(millis int64) {
	ReferenceRequireNonNull(value)
	value.Date.SetTime(millis)
}
func (value *SQLDate) Equals(other any) bool {
	ReferenceRequireNonNull(value)
	return value.Date.Equals(other)
}
func (value *SQLDate) HashCode() int32 { ReferenceRequireNonNull(value); return value.Date.HashCode() }
func (value *SQLDate) CompareTo(other DateValue) int32 {
	ReferenceRequireNonNull(value)
	return value.Date.CompareTo(other)
}

func (value *SQLTime) GetTime() int64 { ReferenceRequireNonNull(value); return value.Date.GetTime() }
func (value *SQLTime) SetTime(millis int64) {
	ReferenceRequireNonNull(value)
	value.Date.SetTime(millis)
}
func (value *SQLTime) Equals(other any) bool {
	ReferenceRequireNonNull(value)
	return value.Date.Equals(other)
}
func (value *SQLTime) HashCode() int32 { ReferenceRequireNonNull(value); return value.Date.HashCode() }
func (value *SQLTime) CompareTo(other DateValue) int32 {
	ReferenceRequireNonNull(value)
	return value.Date.CompareTo(other)
}
