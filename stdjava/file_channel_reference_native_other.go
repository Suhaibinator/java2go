//go:build !darwin && !linux

package stdjava

func fileChannelReadAtNativeExecution(_ *Execution, _ *FileChannel, _ *ByteBuffer, _ int64) int32 {
	panic(NewUnsupportedOperationException("native positional FileChannel read is unsupported on this target"))
}
