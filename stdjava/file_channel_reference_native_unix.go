//go:build darwin || linux

package stdjava

func fileChannelReadAtNativeExecution(execution *Execution, channel *FileChannel, dst *ByteBuffer, position int64) int32 {
	return FileChannelReadAtExecution(execution, channel, dst, position)
}
