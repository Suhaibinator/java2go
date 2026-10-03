package stdjava

import "math/bits"

func (d *LocalDate) HashCode() int32 {
	timeRequire(d)
	return (d.year & -2048) ^ (d.year << 11) ^ (d.month << 6) ^ d.day
}
func (t *LocalTime) HashCode() int32 {
	timeRequire(t)
	nanos := int64(t.hour)*3600000000000 + int64(t.minute)*60000000000 + int64(t.second)*1000000000 + int64(t.nano)
	return int32(nanos ^ (nanos >> 32))
}
func (l *LocalDateTime) HashCode() int32 {
	timeRequire(l)
	return l.date.HashCode() ^ l.time.HashCode()
}
func (d *MonthDay) HashCode() int32  { timeRequire(d); return d.month<<6 + d.day }
func (y *Year) HashCode() int32      { return timeRequire(y).year }
func (y *YearMonth) HashCode() int32 { timeRequire(y); return y.year ^ y.month<<27 }
func (p *Period) HashCode() int32 {
	timeRequire(p)
	return p.years + int32(bits.RotateLeft32(uint32(p.months), 8)) + int32(bits.RotateLeft32(uint32(p.days), 16))
}
func (o *OffsetTime) HashCode() int32 {
	timeRequire(o)
	return o.local.HashCode() ^ o.offset.HashCode()
}
func (o *OffsetDateTime) HashCode() int32 {
	timeRequire(o)
	return o.local.HashCode() ^ o.offset.HashCode()
}
func (z *ZonedDateTime) HashCode() int32 {
	timeRequire(z)
	return z.local.HashCode() ^ z.offset.HashCode() ^ int32(bits.RotateLeft32(uint32(ObjectHashCodeExecution(nil, z.zone)), 3))
}

func init() {
	RegisterJavaType("java.time.temporal.TemporalAccessor", ObjectTypeID)
	RegisterJavaType("java.time.temporal.Temporal", ObjectTypeID, "java.time.temporal.TemporalAccessor")
	RegisterJavaType("java.time.temporal.TemporalAmount", ObjectTypeID)
	RegisterJavaType("java.time.temporal.TemporalAdjuster", ObjectTypeID)
	RegisterJavaType("java.time.chrono.ChronoLocalDate", ObjectTypeID, "java.time.temporal.Temporal", "java.time.temporal.TemporalAdjuster", ComparableTypeID)
	RegisterJavaType("java.time.chrono.ChronoLocalDateTime", ObjectTypeID, "java.time.temporal.Temporal", "java.time.temporal.TemporalAdjuster", ComparableTypeID)
	RegisterJavaType("java.time.chrono.ChronoZonedDateTime", ObjectTypeID, "java.time.temporal.Temporal", ComparableTypeID)
	RegisterJavaType("java.time.chrono.ChronoPeriod", ObjectTypeID, "java.time.temporal.TemporalAmount")
	RegisterJavaType("java.time.Duration", ObjectTypeID, "java.time.temporal.TemporalAmount", ComparableTypeID, SerializableTypeID)
	RegisterJavaType("java.time.Instant", ObjectTypeID, "java.time.temporal.Temporal", "java.time.temporal.TemporalAdjuster", ComparableTypeID, SerializableTypeID)
	RegisterJavaType("java.time.LocalDate", ObjectTypeID, "java.time.chrono.ChronoLocalDate", SerializableTypeID)
	RegisterJavaType("java.time.LocalTime", ObjectTypeID, "java.time.temporal.Temporal", "java.time.temporal.TemporalAdjuster", ComparableTypeID, SerializableTypeID)
	RegisterJavaType("java.time.LocalDateTime", ObjectTypeID, "java.time.chrono.ChronoLocalDateTime", SerializableTypeID)
	RegisterJavaType("java.time.MonthDay", ObjectTypeID, "java.time.temporal.TemporalAccessor", "java.time.temporal.TemporalAdjuster", ComparableTypeID, SerializableTypeID)
	RegisterJavaType("java.time.Period", ObjectTypeID, "java.time.chrono.ChronoPeriod", SerializableTypeID)
	for _, name := range []string{"Year", "YearMonth", "OffsetTime", "OffsetDateTime"} {
		RegisterJavaType(TypeID("java.time."+name), ObjectTypeID, "java.time.temporal.Temporal", "java.time.temporal.TemporalAdjuster", ComparableTypeID, SerializableTypeID)
	}
	RegisterJavaType("java.time.ZonedDateTime", ObjectTypeID, "java.time.chrono.ChronoZonedDateTime", SerializableTypeID)
	RegisterJavaType("java.time.ZoneId", ObjectTypeID, SerializableTypeID)
	RegisterJavaType("java.time.ZoneRegion", "java.time.ZoneId")
	RegisterJavaType("java.time.ZoneOffset", "java.time.ZoneId", "java.time.temporal.TemporalAccessor", "java.time.temporal.TemporalAdjuster", ComparableTypeID)
}
