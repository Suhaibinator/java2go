package stdjava

type ParseException struct {
	ThrowableBase
	errorOffset int32
}

func NewParseException(message string, offset int32) ParseException {
	return ParseException{newThrowableBase("ParseException", message), offset}
}
func (failure ParseException) GetErrorOffset() int32 { return failure.errorOffset }
func ParseExceptionErrorOffset(value any) int32 {
	ReferenceRequireNonNull(value)
	if failure, ok := value.(interface{ GetErrorOffset() int32 }); ok {
		return failure.GetErrorOffset()
	}
	panic(NewClassCastException("not a ParseException"))
}
func init() { RegisterException("ParseException", "Exception") }
