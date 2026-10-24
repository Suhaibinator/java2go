package stdjava

import "fmt"

const JavaIOStringReaderType TypeID = "java.io.StringReader"

func init() { RegisterJavaType(JavaIOStringReaderType, JavaIOReaderType) }

type stringReaderReferenceState struct {
	text               *JavaString
	length, next, mark int32
	info               *ObjectInfo
}

// Diagnostic ingress here is ASCII JDK text, never the reader's String content.
func stringReaderDiagnosticUnits(text string) []uint16 {
	units := make([]uint16, len(text))
	for i := range text {
		units[i] = uint16(text[i])
	}
	return units
}
func stringReaderLiteral(text string) *JavaString {
	return JavaStringLiteralUTF16(stringReaderDiagnosticUnits(text))
}
func stringReaderNullDiagnostic(text string) {
	panic(NullPointerException{newJavaThrowableBase("NullPointerException", NewJavaStringUTF16(stringReaderDiagnosticUnits(text)))})
}
func stringReaderClosed() { panic(NewJavaIOExceptionMessage(stringReaderLiteral("Stream closed"))) }

// Each reader keeps the immutable String allocation, including isolated units.
func NewStringReaderReference(text *JavaString) *StringReader {
	if text == nil {
		stringReaderNullDiagnostic(`Cannot invoke "String.length()" because "s" is null`)
	}
	reader := &StringReader{reference: &stringReaderReferenceState{text: text, length: int32(len(text.units))}}
	reader.reference.info = NewObjectInfo(JavaIOStringReaderType, func(requested TypeID) any {
		switch requested {
		case ObjectTypeID, JavaIOStringReaderType, JavaIOReaderType, JavaIOCloseableType:
			return reader
		}
		return nil
	})
	return reader
}

func (reader *StringReader) JavaObjectInfo() *ObjectInfo {
	if reader == nil || reader.reference == nil {
		return nil
	}
	return reader.reference.info
}
func (reader *StringReader) JavaDynamicTypeID() TypeID {
	if reader == nil || reader.reference == nil {
		return ""
	}
	return JavaIOStringReaderType
}
func (*StringReader) JavaReaderMarker()    {}
func (*StringReader) JavaCloseableMarker() {}
func (reader *StringReader) JavaReaderReadChar(e *Execution) int32 {
	return StringReaderReadCharExecution(e, reader)
}
func (reader *StringReader) JavaReaderReadChars(e *Execution, b *PrimitiveArray[rune], off, n int32) int32 {
	return StringReaderReadCharsExecution(e, reader, b, off, n)
}
func (reader *StringReader) JavaClose(e *Execution) { StringReaderCloseExecution(e, reader) }

func stringReaderReferenceStateOf(reader *StringReader) *stringReaderReferenceState {
	if reader == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	if reader.reference == nil {
		panic(UnsupportedOperationException{newJavaThrowableBase("UnsupportedOperationException", stringReaderLiteral("canonical StringReader state required"))})
	}
	return reader.reference
}
func stringReaderEnsureOpen(state *stringReaderReferenceState) {
	if state.text == nil {
		stringReaderClosed()
	}
}

func StringReaderReadCharExecution(e *Execution, reader *StringReader) int32 {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	if state.next >= state.length {
		return -1
	}
	unit := state.text.units[state.next]
	state.next++
	return int32(unit)
}
func StringReaderReadCharsExecution(e *Execution, reader *StringReader, buffer *PrimitiveArray[rune], offset, length int32) int32 {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	if buffer == nil {
		stringReaderNullDiagnostic(`Cannot read the array length because "cbuf" is null`)
	}
	size := int64(len(buffer.Elements))
	if offset < 0 || length < 0 || int64(offset)+int64(length) > size {
		message := fmt.Sprintf("Range [%d, %d + %d) out of bounds for length %d", offset, offset, length, size)
		panic(NewJavaIndexOutOfBoundsExceptionMessage(NewJavaStringUTF16(stringReaderDiagnosticUnits(message))))
	}
	if length == 0 {
		return 0
	}
	if state.next >= state.length {
		return -1
	}
	n := length
	if remaining := state.length - state.next; n > remaining {
		n = remaining
	}
	for i := int32(0); i < n; i++ {
		buffer.Elements[offset+i] = rune(state.text.units[state.next+i])
	}
	state.next += n
	return n
}
func StringReaderSkipExecution(e *Execution, reader *StringReader, count int64) int64 {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	if state.next >= state.length {
		return 0
	}
	if maximum := int64(state.length - state.next); count > maximum {
		count = maximum
	}
	if minimum := -int64(state.next); count < minimum {
		count = minimum
	}
	state.next += int32(count)
	return count
}
func StringReaderReadyExecution(e *Execution, reader *StringReader) bool {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	return true
}
func StringReaderMarkSupportedExecution(_ *Execution, reader *StringReader) bool {
	stringReaderReferenceStateOf(reader)
	return true
}
func StringReaderMarkExecution(e *Execution, reader *StringReader, limit int32) {
	state := stringReaderReferenceStateOf(reader)
	if limit < 0 {
		panic(NewJavaIllegalArgumentExceptionMessage(stringReaderLiteral("Read-ahead limit < 0")))
	}
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	state.mark = state.next
}
func StringReaderResetExecution(e *Execution, reader *StringReader) {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	stringReaderEnsureOpen(state)
	state.next = state.mark
}
func StringReaderCloseExecution(e *Execution, reader *StringReader) {
	state := stringReaderReferenceStateOf(reader)
	guard := MonitorEnterExecution(e, reader)
	defer MonitorExitExecution(guard)
	state.text = nil
}
