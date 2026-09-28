package stdjava

import (
	"fmt"
	"sync"
)

// Reader is Java's character-stream view. Its nominal marker is independent of
// Go's byte-oriented io.Reader and of Java's overloaded source method names.
type Reader interface {
	JavaReaderMarker()
	Closeable
}
type Closeable interface{ JavaCloseableMarker() }
type ReaderBase struct{ Lock any }

func (*ReaderBase) JavaReaderMarker()    {}
func (*ReaderBase) JavaCloseableMarker() {}
func NewReaderBase(lock ...any) *ReaderBase {
	base := &ReaderBase{}
	if len(lock) > 0 {
		ReferenceRequireNonNull(lock[0])
		base.Lock = lock[0]
	}
	return base
}
func ReaderReadCharExecution(execution *Execution, reader Reader) int32 {
	ReferenceRequireNonNull(reader)
	if source, ok := reader.(interface{ JavaReaderReadChar(*Execution) int32 }); ok {
		return source.JavaReaderReadChar(execution)
	}
	return ReaderReadCharDefaultExecution(execution, reader)
}
func ReaderReadCharDefaultExecution(execution *Execution, reader Reader) int32 {
	buffer := NewPrimitiveArray[rune](1, PrimitiveTypeID("char"))
	if ReaderReadCharsExecution(execution, reader, buffer, 0, 1) == -1 {
		return -1
	}
	return int32(buffer.Elements[0])
}
func ReaderReadArrayExecution(execution *Execution, reader Reader, buffer *PrimitiveArray[rune]) int32 {
	ReferenceRequireNonNull(reader)
	if source, ok := reader.(interface {
		JavaReaderReadArray(*Execution, *PrimitiveArray[rune]) int32
	}); ok {
		return source.JavaReaderReadArray(execution, buffer)
	}
	return ReaderReadArrayDefaultExecution(execution, reader, buffer)
}
func ReaderReadArrayDefaultExecution(execution *Execution, reader Reader, buffer *PrimitiveArray[rune]) int32 {
	return ReaderReadCharsExecution(execution, reader, buffer, 0, PrimitiveArrayLength(buffer))
}
func ReaderReadCharsExecution(execution *Execution, reader Reader, buffer *PrimitiveArray[rune], offset, length int32) int32 {
	ReferenceRequireNonNull(reader)
	// Dispatch precedes validation: Java overrides can observe even null or zero-length arguments.
	if source, ok := reader.(interface {
		JavaReaderReadChars(*Execution, *PrimitiveArray[rune], int32, int32) int32
	}); ok {
		return source.JavaReaderReadChars(execution, buffer, offset, length)
	}
	panic(NewClassCastException("Reader implementation lacks read(char[],int,int)"))
}
func CloseableCloseExecution(execution *Execution, value Closeable) {
	ReferenceRequireNonNull(value)
	if source, ok := value.(interface{ JavaClose(*Execution) }); ok {
		source.JavaClose(execution)
		return
	}
	if source, ok := value.(interface{ CloseJava2goExecution(*Execution) }); ok {
		source.CloseJava2goExecution(execution)
		return
	}
	if builtin, ok := value.(interface{ Close() }); ok {
		builtin.Close()
		return
	}
	panic(NewClassCastException("Closeable implementation lacks close()"))
}

// Writer and its interfaces retain Java's nominal relationships while the
// overload bridges below preserve source implementations and UTF-16 arrays.
type Writer interface {
	JavaWriterMarker()
	Appendable
	Closeable
	Flushable
}
type WriterBase struct {
	Lock        any
	writeBuffer *PrimitiveArray[rune]
}

func (*WriterBase) JavaWriterMarker()     {}
func (*WriterBase) JavaAppendableMarker() {}
func (*WriterBase) JavaCloseableMarker()  {}
func (*WriterBase) JavaFlushableMarker()  {}
func NewWriterBase(lock ...any) *WriterBase {
	base := &WriterBase{}
	if len(lock) > 0 {
		ReferenceRequireNonNull(lock[0])
		base.Lock = lock[0]
	}
	return base
}
func WriterWriteExecution(execution *Execution, writer Writer, value any, bounds ...int32) {
	writerWrite(execution, writer, value, false, bounds...)
}
func WriterWriteDefaultExecution(execution *Execution, writer Writer, value any, bounds ...int32) {
	writerWrite(execution, writer, value, true, bounds...)
}
func writerWrite(execution *Execution, writer Writer, value any, baseDefault bool, bounds ...int32) {
	ReferenceRequireNonNull(writer)
	switch number := value.(type) {
	case int:
		value = int32(number)
	case int8:
		value = int32(number)
	case int16:
		value = int32(number)
	}
	switch value := value.(type) {
	case int32:
		if source, ok := writer.(interface{ JavaWriterWriteInt(*Execution, int32) }); ok && !baseDefault {
			source.JavaWriterWriteInt(execution, value)
			return
		}
		state := WriterState(writer)
		guard := MonitorEnterExecution(execution, state.Lock)
		defer MonitorExitExecution(guard)
		if state.writeBuffer == nil {
			state.writeBuffer = NewPrimitiveArray[rune](1024, PrimitiveTypeID("char"))
		}
		state.writeBuffer.Elements[0] = rune(uint16(value))
		WriterWriteCharsExecution(execution, writer, state.writeBuffer, 0, 1)
	case string:
		if len(bounds) == 0 {
			if source, ok := writer.(interface{ JavaWriterWriteString(*Execution, string) }); ok && !baseDefault {
				source.JavaWriterWriteString(execution, value)
				return
			}
			WriterWriteStringRangeExecution(execution, writer, value, 0, StringLength(StringRequireNonNull(value)))
			return
		}
		if baseDefault {
			WriterWriteStringRangeDefaultExecution(execution, writer, value, bounds[0], bounds[1])
		} else {
			WriterWriteStringRangeExecution(execution, writer, value, bounds[0], bounds[1])
		}
	case *PrimitiveArray[rune]:
		if len(bounds) == 0 {
			if source, ok := writer.(interface {
				JavaWriterWriteArray(*Execution, *PrimitiveArray[rune])
			}); ok && !baseDefault {
				source.JavaWriterWriteArray(execution, value)
				return
			}
			WriterWriteCharsExecution(execution, writer, value, 0, PrimitiveArrayLength(value))
			return
		}
		WriterWriteCharsExecution(execution, writer, value, bounds[0], bounds[1])
	default:
		ReferenceRequireNonNull(value)
		panic(NewClassCastException("invalid Writer.write argument"))
	}
}
func WriterWriteStringRangeExecution(execution *Execution, writer Writer, value string, offset, length int32) {
	if source, ok := writer.(interface {
		JavaWriterWriteStringRange(*Execution, string, int32, int32)
	}); ok {
		source.JavaWriterWriteStringRange(execution, value, offset, length)
		return
	}
	WriterWriteStringRangeDefaultExecution(execution, writer, value, offset, length)
}
func WriterWriteStringRangeDefaultExecution(execution *Execution, writer Writer, value string, offset, length int32) {
	state := WriterState(writer)
	guard := MonitorEnterExecution(execution, state.Lock)
	defer MonitorExitExecution(guard)
	var buffer *PrimitiveArray[rune]
	if length <= 1024 {
		if state.writeBuffer == nil {
			state.writeBuffer = NewPrimitiveArray[rune](1024, PrimitiveTypeID("char"))
		}
		buffer = state.writeBuffer
	} else {
		buffer = NewPrimitiveArray[rune](length, PrimitiveTypeID("char"))
	}
	units := StringChars(StringRequireNonNull(value))
	end := offset + length
	if offset < 0 || offset > end || end > int32(len(units)) {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Range [%d, %d) out of bounds for length %d", offset, end, len(units))))
	}
	copy(buffer.Elements, units[offset:end])
	WriterWriteCharsExecution(execution, writer, buffer, 0, length)
}
func WriterWriteCharsExecution(execution *Execution, writer Writer, value *PrimitiveArray[rune], offset, length int32) {
	ReferenceRequireNonNull(writer)
	if source, ok := writer.(interface {
		JavaWriterWriteChars(*Execution, *PrimitiveArray[rune], int32, int32)
	}); ok {
		source.JavaWriterWriteChars(execution, value, offset, length)
		return
	}
	panic(NewClassCastException("Writer implementation lacks write(char[],int,int)"))
}
func checkCharacterRange(size, offset, length int32) {
	if offset < 0 || length < 0 || int64(offset)+int64(length) > int64(size) {
		panic(NewIndexOutOfBoundsException("character range out of bounds"))
	}
}
func WriterAppendExecution(execution *Execution, writer Writer, value any, bounds ...int32) Writer {
	result := AppendableAppendExecution(execution, writer, value, bounds...)
	return result.(Writer)
}

const (
	JavaIOReaderType       TypeID = "java.io.Reader"
	JavaIOWriterType       TypeID = "java.io.Writer"
	JavaLangAppendableType TypeID = "java.lang.Appendable"
	JavaIOCloseableType    TypeID = "java.io.Closeable"
	JavaIOFlushableType    TypeID = "java.io.Flushable"
)

func init() {
	RegisterJavaType(JavaLangAppendableType, ObjectTypeID)
	RegisterJavaType(JavaIOCloseableType, ObjectTypeID)
	RegisterJavaType(JavaIOFlushableType, ObjectTypeID)
	RegisterJavaType(JavaIOReaderType, ObjectTypeID, JavaIOCloseableType)
	RegisterJavaType(JavaIOWriterType, ObjectTypeID, JavaLangAppendableType, JavaIOCloseableType, JavaIOFlushableType)
}

var writerBaseMu sync.Mutex

func EnsureWriterBase(slot **WriterBase, receiver Writer) *WriterBase {
	writerBaseMu.Lock()
	defer writerBaseMu.Unlock()
	if *slot == nil {
		*slot = NewWriterBase(receiver)
	}
	return *slot
}
func WriterState(writer Writer) *WriterBase {
	ReferenceRequireNonNull(writer)
	if state, ok := writer.(interface{ JavaWriterState() *WriterBase }); ok {
		return state.JavaWriterState()
	}
	panic(NewClassCastException("Writer implementation lacks base state"))
}
