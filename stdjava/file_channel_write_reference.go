package stdjava

// Source declarations own their argument validation and effects. In particular,
// a canonical FileChannel parameter may carry a genuine source allocation.
func FileChannelWriteVirtualExecution(execution *Execution, channel *FileChannel, src *ByteBuffer) int32 {
	requireExecution(execution)
	if source := fileChannelSourceBacking(channel); source != nil {
		if source.methods.Write == nil {
			panic(NewUnsupportedOperationException("FileChannel source declaration has no write protocol"))
		}
		return source.methods.Write(execution, src)
	}
	return fileChannelWriteNativeExecution(execution, channel, src)
}
func FileChannelWriteAtVirtualExecution(execution *Execution, channel *FileChannel, src *ByteBuffer, position int64) int32 {
	requireExecution(execution)
	if source := fileChannelSourceBacking(channel); source != nil {
		if source.methods.WriteAt == nil {
			panic(NewUnsupportedOperationException("FileChannel source declaration has no write protocol"))
		}
		return source.methods.WriteAt(execution, src, position)
	}
	return fileChannelWriteAtNativeExecution(execution, channel, src, position)
}
