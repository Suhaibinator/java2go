package stdjava

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
)

// RandomAccessFileSetLengthExecution changes the existing descriptor's size and
// clamps its actual shared cursor only after a successful truncation. RAF I/O
// preserves a pending Java interrupt; it does not enter FileChannel's protocol.
func RandomAccessFileSetLengthExecution(execution *Execution, file *RandomAccessFile, length int64) {
	requireExecution(execution)
	ReferenceRequireNonNull(file)
	opened := randomAccessFileWriteDescriptor(file)
	if length < 0 {
		randomAccessFileWriteFailure(errors.New("invalid argument"))
	}
	if err := opened.Truncate(length); err != nil {
		randomAccessFileWriteFailure(err)
	}
	position, err := opened.Seek(0, io.SeekCurrent)
	if err != nil {
		randomAccessFileWriteFailure(err)
	}
	if position > length {
		if _, err := opened.Seek(length, io.SeekStart); err != nil {
			randomAccessFileWriteFailure(err)
		}
	}
}

// RandomAccessFileWriteBytesExecution implements the byte-array overload, which
// completes the whole range or throws after retaining every positive OS effect.
// A null array precedes descriptor validation; an empty array needs no open FD.
func RandomAccessFileWriteBytesExecution(execution *Execution, file *RandomAccessFile, bytes *PrimitiveArray[int8]) {
	requireExecution(execution)
	ReferenceRequireNonNull(file)
	if bytes == nil {
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8("Cannot read the array length because \"b\" is null")))
	}
	if len(bytes.Elements) == 0 {
		return
	}
	opened := randomAccessFileWriteDescriptor(file)
	// os.File.Write owns its descriptor lease and full-write loop. The lifecycle
	// mutex must not be held while that lease can delay concurrent close.
	if _, err := opened.Write(unsignedBytes(bytes.Elements)); err != nil {
		randomAccessFileWriteFailure(err)
	}
}

func randomAccessFileWriteDescriptor(file *RandomAccessFile) *os.File {
	state := file.state
	state.mu.Lock()
	closed, opened := state.closed, state.file
	state.mu.Unlock()
	if closed {
		randomAccessFileWriteFailure(os.ErrClosed)
	}
	return opened
}

func randomAccessFileWriteFailure(err error) {
	if errors.Is(err, os.ErrClosed) {
		panic(NewJavaIOExceptionMessage(JavaStringFromHostUTF8("Stream Closed")))
	}
	var pathError *fs.PathError
	if errors.As(err, &pathError) {
		err = pathError.Err
	}
	// Actual JDK native errno vectors pin Invalid argument, Bad file descriptor
	// and File too large. Other OS outcomes carry no additional parity claim.
	reason := err.Error()
	if len(reason) != 0 {
		reason = strings.ToUpper(reason[:1]) + reason[1:]
	}
	panic(NewJavaIOExceptionMessage(JavaStringFromHostUTF8(reason)))
}
