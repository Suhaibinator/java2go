//go:build darwin || linux

package stdjava

import (
	"os"
	"syscall"
)

// FileChannelWriteExecution is the finite native relative-write entry. Source
// dispatch belongs to the virtual facade. Active native cancellation remains a
// separate frontier; the caller's pending interrupt precedes src dereference.
func FileChannelWriteExecution(execution *Execution, channel *FileChannel, src *ByteBuffer) int32 {
	requireExecution(execution)
	if fileChannelSourceBacking(channel) != nil {
		panic(NewUnsupportedOperationException("native FileChannel write requires native backing"))
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
	if src == nil {
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8("Cannot invoke \"java.nio.ByteBuffer.position()\" because \"src\" is null")))
	}
	remaining := src.limit - src.position
	if remaining == 0 {
		count, _ := fileChannelWriteFinish(state, src, 0, nil)
		return count
	}
	// A readonly view is a valid source. unsignedBytes makes one explicit signed
	// byte conversion copy, retained through zero-effect retries of this call.
	start := int64(src.offset) + int64(src.position)
	end := start + int64(remaining)
	window := unsignedBytes(src.array.Elements[int(start):int(end)])
	for {
		count, err := fileChannelWriteOnce(file, window)
		result, retry := fileChannelWriteFinish(state, src, count, err)
		if !retry {
			return result
		}
	}
}

// Perform one write at the actual shared descriptor cursor. The callback owns
// only the raw lease; it never holds lifecycle/Thread locks or calls Java code.
func fileChannelWriteOnce(file *os.File, window []byte) (int, error) {
	raw, err := file.SyscallConn()
	if err != nil {
		return 0, err
	}
	count := 0
	var nativeErr error
	err = raw.Write(func(fd uintptr) bool { count, nativeErr = syscall.Write(int(fd), window); return true })
	if err != nil {
		return count, err
	}
	return count, nativeErr
}
func fileChannelWriteNativeExecution(execution *Execution, channel *FileChannel, src *ByteBuffer) int32 {
	return FileChannelWriteExecution(execution, channel, src)
}
func fileChannelWriteAtNativeExecution(execution *Execution, channel *FileChannel, src *ByteBuffer, position int64) int32 {
	return FileChannelWriteAtExecution(execution, channel, src, position)
}
