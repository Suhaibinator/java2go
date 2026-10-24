package stdjava

// Time parsing failures retain the original UTF-16 text and the Java error index.
type DateTimeException struct{ ThrowableBase }
type DateTimeParseException struct {
	DateTimeException
	parsed *JavaString
	index  int32
}
type ZoneRulesException struct{ DateTimeException }

func NewDateTimeException(message string) DateTimeException {
	return DateTimeException{newThrowableBase("DateTimeException", message)}
}
func NewDateTimeParseException(message string, text *JavaString, index int32) DateTimeParseException {
	return DateTimeParseException{DateTimeException{newThrowableBase("DateTimeParseException", message)}, text, index}
}
func NewZoneRulesException(message string) ZoneRulesException {
	return ZoneRulesException{DateTimeException{newThrowableBase("ZoneRulesException", message)}}
}
func (e DateTimeParseException) GetErrorIndex() int32         { return e.index }
func (e DateTimeParseException) GetParsedString() *JavaString { return e.parsed }
func DateTimeParseExceptionErrorIndex(value any) int32 {
	ReferenceRequireNonNull(value)
	return value.(interface{ GetErrorIndex() int32 }).GetErrorIndex()
}
func DateTimeParseExceptionParsedString(value any) *JavaString {
	ReferenceRequireNonNull(value)
	return value.(interface{ GetParsedString() *JavaString }).GetParsedString()
}
func init() {
	for _, e := range []struct{ name, id, parent string }{
		{"DateTimeException", "java.time.DateTimeException", "RuntimeException"},
		{"DateTimeParseException", "java.time.format.DateTimeParseException", "DateTimeException"},
		{"ZoneRulesException", "java.time.zone.ZoneRulesException", "DateTimeException"},
	} {
		descriptor := builtinThrowableDescriptors["RuntimeException"]
		descriptor.id = TypeID(e.id)
		descriptor.parent = e.parent
		builtinThrowableDescriptors[e.name] = descriptor
		RegisterException(e.name, e.parent)
		RegisterJavaType(TypeID(e.id), BuiltinThrowableTypeID(e.parent))
	}
}
