//go:build !darwin && !linux

package stdjava

func fileChannelWriteNativeExecution(_ *Execution, _ *FileChannel, _ *ByteBuffer) int32 {
	panic(NewUnsupportedOperationException("native relative FileChannel write is unsupported on this target"))
}
func fileChannelWriteAtNativeExecution(_ *Execution, _ *FileChannel, _ *ByteBuffer, _ int64) int32 {
	panic(NewUnsupportedOperationException("native positional FileChannel write is unsupported on this target"))
}
