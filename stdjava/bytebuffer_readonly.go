package stdjava

import "io"

func byteBufferRequireWritable(b *ByteBuffer) {
	ReferenceRequireNonNull(b)
	if b.readOnly {
		panic(newJavaThrowableBase("ReadOnlyBufferException", nil))
	}
}
func (b *ByteBuffer) IsReadOnly() bool { ReferenceRequireNonNull(b); return b.readOnly }
func (b *ByteBuffer) AsReadOnlyBuffer() *ByteBuffer {
	view := b.Duplicate()
	view.readOnly = true
	return view
}
func (b *ByteBuffer) PutArrayAt(index int32, source *PrimitiveArray[int8], offset, length int32) *ByteBuffer {
	byteBufferRequireWritable(b)
	byteBufferCheckRange(index, length, b.limit)
	ReferenceRequireNonNull(source)
	byteBufferCheckRange(offset, length, int32(len(source.Elements)))
	copy(b.array.Elements[b.offset+index:b.offset+index+length], source.Elements[offset:offset+length])
	return b
}
func (b *ByteBuffer) PutArrayAtWhole(index int32, source *PrimitiveArray[int8]) *ByteBuffer {
	ReferenceRequireNonNull(b)
	ReferenceRequireNonNull(source)
	return b.PutArrayAt(index, source, 0, int32(len(source.Elements)))
}
func (b *ByteBuffer) PutBufferAt(index int32, source *ByteBuffer, offset, length int32) *ByteBuffer {
	byteBufferRequireWritable(b)
	byteBufferCheckRange(index, length, b.limit)
	ReferenceRequireNonNull(source)
	byteBufferCheckRange(offset, length, source.limit)
	copy(b.array.Elements[b.offset+index:b.offset+index+length], source.array.Elements[source.offset+offset:source.offset+offset+length])
	return b
}

// This entry corrects validation and preinterrupt behavior for the existing
// relative read. Active native I/O cancellation remains a separate frontier.
func FileChannelReadExecution(execution *Execution, c *FileChannel, buffer *ByteBuffer) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(c)
	c.state.mu.Lock()
	closed := c.state.closed
	c.state.mu.Unlock()
	if closed {
		panic(newJavaThrowableBase("ClosedChannelException", nil))
	}
	thread := ThreadCurrentThread(execution)
	thread.interruptMu.Lock()
	interrupted := thread.interrupted
	thread.interruptMu.Unlock()
	if interrupted {
		fileChannelReadCloseForInterrupt(c)
		panic(newJavaThrowableBase("ClosedByInterruptException", nil))
	}
	ReferenceRequireNonNull(buffer)
	if buffer.readOnly {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringLiteralUTF16([]uint16{'R', 'e', 'a', 'd', '-', 'o', 'n', 'l', 'y', ' ', 'b', 'u', 'f', 'f', 'e', 'r'})))
	}
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if c.state.closed {
		panic(newJavaThrowableBase("ClosedChannelException", nil))
	}
	if buffer.Remaining() == 0 {
		return 0
	}
	bytes := make([]byte, buffer.Remaining())
	count, err := c.state.file.Read(bytes)
	for i := 0; i < count; i++ {
		buffer.array.Elements[int(buffer.offset+buffer.position)+i] = int8(bytes[i])
	}
	buffer.position += int32(count)
	if count > 0 {
		return int32(count)
	}
	if err == io.EOF {
		return -1
	}
	if err != nil {
		throwIOException(err)
	}
	return 0
}
func fileChannelReadCloseForInterrupt(c *FileChannel) {
	defer func() {
		if failure := recover(); failure != nil {
			if !isSubtypeOf(throwableTypeName(failure), "IOException") {
				panic(failure)
			}
		}
	}()
	c.state.close()
}
func init() {
	RegisterException("ReadOnlyBufferException", "UnsupportedOperationException")
	RegisterException("AsynchronousCloseException", "ClosedChannelException")
	RegisterException("ClosedByInterruptException", "AsynchronousCloseException")
}
