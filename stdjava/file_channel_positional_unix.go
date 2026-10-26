//go:build darwin || linux

package stdjava

import (
	"errors"
	"os"
	"syscall"
)

// FileChannelWriteAtExecution implements finite heap-buffer positional writes.
// Active native cancellation, source carriers and Java admission are separate
// prerequisites. The caller's pending interrupt is checked before native I/O.
func FileChannelWriteAtExecution(execution *Execution, channel *FileChannel, src *ByteBuffer, position int64) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(channel)
	if src == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	if position < 0 {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{'N', 'e', 'g', 'a', 't', 'i', 'v', 'e', ' ', 'p', 'o', 's', 'i', 't', 'i', 'o', 'n'})))
	}
	state := channel.state
	state.mu.Lock()
	closed, writable, file := state.closed, state.writable, state.file
	state.mu.Unlock()
	if closed {
		panic(newJavaThrowableBase("ClosedChannelException", nil))
	}
	if !writable {
		panic(newJavaThrowableBase("NonWritableChannelException", nil))
	}
	thread := ThreadCurrentThread(execution)
	thread.interruptMu.Lock()
	interrupted := thread.interrupted
	thread.interruptMu.Unlock()
	if interrupted {
		fileChannelReadCloseForInterrupt(channel)
		panic(newJavaThrowableBase("ClosedByInterruptException", nil))
	}
	remaining := src.limit - src.position
	if remaining == 0 {
		count, _ := fileChannelWriteFinish(state, src, 0, nil)
		return count
	}
	// Use the private heap window: a read-only view is a valid write source.
	// Widen before index arithmetic so valid buffers also work on 32-bit hosts.
	start := int64(src.offset) + int64(src.position)
	end := start + int64(remaining)
	window := unsignedBytes(src.array.Elements[int(start):int(end)])
	for {
		count, err := fileChannelPwriteOnce(file, window, position)
		result, retry := fileChannelWriteFinish(state, src, count, err)
		if !retry {
			return result
		}
	}
}

// The callback performs exactly one pwrite and always completes the raw lease,
// including EINTR/EAGAIN. It never takes state/Thread locks or calls Java code.
func fileChannelPwriteOnce(file *os.File, window []byte, position int64) (int, error) {
	raw, err := file.SyscallConn()
	if err != nil {
		return 0, err
	}
	count := 0
	var nativeErr error
	err = raw.Write(func(fd uintptr) bool {
		count, nativeErr = syscall.Pwrite(int(fd), window, position)
		return true
	})
	if err != nil {
		return count, err
	}
	return count, nativeErr
}

// RawConn.Write has released its descriptor lease before this completion path.
// Positive native effects precede completion/errors and are never rolled back.
func fileChannelWriteFinish(state *randomAccessState, src *ByteBuffer, count int, err error) (int32, bool) {
	if count > 0 {
		src.position += int32(count)
	}
	state.mu.Lock()
	closed := state.closed
	state.mu.Unlock()
	if count <= 0 && closed {
		panic(newJavaThrowableBase("AsynchronousCloseException", nil))
	}
	if err != nil {
		if count <= 0 && errors.Is(err, syscall.EINTR) {
			return 0, true
		}
		if count <= 0 && (errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK)) {
			return 0, false
		}
		fileChannelPositionalFailure(err)
	}
	if count < 0 {
		return 0, false
	}
	return int32(count), false
}

func fileChannelPositionalFailure(err error) {
	if errors.Is(err, syscall.EFBIG) {
		// POSIX pwrite EFBIG maps to the JDK platform strerror contract. This
		// operation's spelling is pinned by actual JDK file-size-limit controls.
		panic(NewJavaIOExceptionMessage(JavaStringFromHostUTF8("File too large")))
	}
	// Other native-error messages retain host diagnostics. JDK message parity
	// for those platform outcomes is held until actual differential evidence.
	panic(NewJavaIOExceptionMessage(JavaStringFromHostUTF8(err.Error())))
}

func init() { RegisterException("NonWritableChannelException", "IllegalStateException") }
