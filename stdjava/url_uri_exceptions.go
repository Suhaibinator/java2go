package stdjava

// URISyntaxException's base detail is the reason; its virtual message combines
// that reason with the original input and UTF16 error index.
type URISyntaxException struct {
	ThrowableBase
	input *JavaString
	index int32
}

func NewJavaURISyntaxException(input, reason *JavaString, index int32) URISyntaxException {
	if input == nil || reason == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	if index < -1 {
		panic(NewJavaIllegalArgumentExceptionMessage(nil))
	}
	return URISyntaxException{newJavaThrowableBase("URISyntaxException", reason), input, index}
}

func (failure URISyntaxException) GetInput() *JavaString  { return failure.input }
func (failure URISyntaxException) GetReason() *JavaString { return failure.javaThrowableMessage() }
func (failure URISyntaxException) GetIndex() int32        { return failure.index }
func (failure URISyntaxException) GetMessageJava2goExecution(_ *Execution) *JavaString {
	units := failure.GetReason().UTF16Copy()
	if failure.index >= 0 {
		units = append(units, uriRuntimeText(" at index ").units...)
		units = append(units, JavaStringValueOfInt(failure.index).units...)
	}
	units = append(units, ':', ' ')
	units = append(units, failure.input.units...)
	return NewJavaStringUTF16(units)
}

func URISyntaxExceptionInput(receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	return receiver.(interface{ GetInput() *JavaString }).GetInput()
}
func URISyntaxExceptionReason(receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	return receiver.(interface{ GetReason() *JavaString }).GetReason()
}
func URISyntaxExceptionIndex(receiver any) int32 {
	ReferenceRequireNonNull(receiver)
	return receiver.(interface{ GetIndex() int32 }).GetIndex()
}

func init() {
	RegisterException("URISyntaxException", "Exception")
	callback := func(execution *Execution, receiver any) *JavaString {
		return receiver.(interface{ GetMessageJava2goExecution(*Execution) *JavaString }).GetMessageJava2goExecution(execution)
	}
	RegisterJavaThrowableMessage(URISyntaxException{}, callback)
	RegisterJavaThrowableMessage((*URISyntaxException)(nil), callback)
}
