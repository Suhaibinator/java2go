//go:build darwin || linux

package stdjava

import (
	"errors"
	"os"
	"syscall"
)

// FileChannelReadAtExecution performs finite positional reads for the existing
// RAF-backed heap channel. Active native cancellation, direct/provider channels
// and Java source carrier admission remain separate contracts.
func FileChannelReadAtExecution(execution *Execution, channel *FileChannel, dst *ByteBuffer, position int64) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(channel)
	if dst == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	if position < 0 {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringFromHostUTF8("Negative position")))
	}
	state := channel.state
	state.mu.Lock()
	closed, file := state.closed, state.file
	state.mu.Unlock()
	if closed {
		panic(newJavaThrowableBase("ClosedChannelException", nil))
	}
	// All channels currently constructed from RAF r/rw/rwd/rws are readable.
	// Do not infer support for an unrepresented write-only/provider channel.
	thread := ThreadCurrentThread(execution)
	thread.interruptMu.Lock()
	interrupted := thread.interrupted
	thread.interruptMu.Unlock()
	if interrupted {
		fileChannelReadCloseForInterrupt(channel)
		panic(newJavaThrowableBase("ClosedByInterruptException", nil))
	}
	if dst.readOnly {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringFromHostUTF8("Read-only buffer")))
	}
	remaining := dst.limit - dst.position
	if remaining == 0 {
		result, _ := fileChannelReadAtFinish(state, dst, nil, 0, nil, true)
		return result
	}
	window := make([]byte, int(remaining))
	for {
		count, err := fileChannelPreadOnce(file, window, position)
		result, retry := fileChannelReadAtFinish(state, dst, window, count, err, false)
		if !retry {
			return result
		}
	}
}

// The callback always releases the descriptor lease after exactly one pread,
// even for EINTR/EAGAIN. No Java callback or state lock runs inside the lease.
func fileChannelPreadOnce(file *os.File, window []byte, position int64) (int, error) {
	raw, err := file.SyscallConn()
	if err != nil {
		return 0, err
	}
	count := 0
	var nativeErr error
	err = raw.Read(func(fd uintptr) bool {
		count, nativeErr = syscall.Pread(int(fd), window, position)
		return true
	})
	if err != nil {
		return count, err
	}
	return count, nativeErr
}

// Positive native effects precede completion and are never rolled back. Empty
// reads and EOF both use this close check but retain distinct return values.
func fileChannelReadAtFinish(state *randomAccessState, dst *ByteBuffer, window []byte, count int, err error, empty bool) (int32, bool) {
	if count > 0 {
		start := int64(dst.offset) + int64(dst.position)
		for index := 0; index < count; index++ {
			dst.array.Elements[int(start)+index] = int8(window[index])
		}
		dst.position += int32(count)
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
		// Reuse only the existing native operation-error mapping. Untested
		// platform messages remain held; no blanket capitalization is applied.
		fileChannelPositionalFailure(err)
	}
	if count > 0 {
		return int32(count), false
	}
	if empty {
		return 0, false
	}
	return -1, false
}
