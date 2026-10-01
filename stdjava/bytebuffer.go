package stdjava

// ByteBuffer models a mutable heap buffer. Views retain their Java byte[]
// identity while each buffer has independent capacity, position, limit and mark.
// Direct buffers, read-only buffers, byte order and typed views remain unsupported.
type ByteBuffer struct {
	array    *PrimitiveArray[int8]
	offset   int32
	capacity int32
	position int32
	limit    int32
	mark     int32
}

func ByteBufferWrap(array *PrimitiveArray[int8], bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(array)
	length := int32(len(array.Elements))
	buffer := &ByteBuffer{array: array, capacity: length, limit: length, mark: -1}
	if len(bounds) == 2 {
		offset, count := bounds[0], bounds[1]
		byteBufferCheckRange(offset, count, length)
		buffer.position, buffer.limit = offset, offset+count
	}
	return buffer
}
func ByteBufferAllocate(capacity int32) *ByteBuffer {
	if capacity < 0 {
		panic(NewIllegalArgumentException("negative capacity"))
	}
	return ByteBufferWrap(NewPrimitiveArray[int8](capacity, PrimitiveByteTypeID))
}
func (b *ByteBuffer) Remaining() int32             { ReferenceRequireNonNull(b); return b.limit - b.position }
func (b *ByteBuffer) HasRemaining() bool           { return b.Remaining() > 0 }
func (b *ByteBuffer) HasArray() bool               { ReferenceRequireNonNull(b); return true }
func (b *ByteBuffer) Array() *PrimitiveArray[int8] { ReferenceRequireNonNull(b); return b.array }
func (b *ByteBuffer) ArrayOffset() int32           { ReferenceRequireNonNull(b); return b.offset }
func (b *ByteBuffer) Capacity() int32              { ReferenceRequireNonNull(b); return b.capacity }
func (b *ByteBuffer) Position() int32              { ReferenceRequireNonNull(b); return b.position }
func (b *ByteBuffer) Limit() int32                 { ReferenceRequireNonNull(b); return b.limit }
func (b *ByteBuffer) SetPosition(position int32) *ByteBuffer {
	ReferenceRequireNonNull(b)
	if position < 0 || position > b.limit {
		panic(NewIllegalArgumentException("position out of bounds"))
	}
	b.position = position
	if b.mark > position {
		b.mark = -1
	}
	return b
}
func (b *ByteBuffer) SetLimit(limit int32) *ByteBuffer {
	ReferenceRequireNonNull(b)
	if limit < 0 || limit > b.capacity {
		panic(NewIllegalArgumentException("limit out of bounds"))
	}
	b.limit = limit
	if b.position > limit {
		b.position = limit
	}
	if b.mark > limit {
		b.mark = -1
	}
	return b
}
func (b *ByteBuffer) Mark() *ByteBuffer { ReferenceRequireNonNull(b); b.mark = b.position; return b }
func (b *ByteBuffer) Reset() *ByteBuffer {
	ReferenceRequireNonNull(b)
	if b.mark < 0 {
		panic(newThrowableBase("InvalidMarkException", ""))
	}
	b.position = b.mark
	return b
}
func (b *ByteBuffer) Rewind() *ByteBuffer {
	ReferenceRequireNonNull(b)
	b.position = 0
	b.mark = -1
	return b
}
func (b *ByteBuffer) Flip() *ByteBuffer {
	ReferenceRequireNonNull(b)
	b.limit = b.position
	b.position = 0
	b.mark = -1
	return b
}
func (b *ByteBuffer) Clear() *ByteBuffer {
	ReferenceRequireNonNull(b)
	b.position = 0
	b.limit = b.capacity
	b.mark = -1
	return b
}
func (b *ByteBuffer) Slice(bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(b)
	index, length := b.position, b.Remaining()
	if len(bounds) == 2 {
		index, length = bounds[0], bounds[1]
		byteBufferCheckRange(index, length, b.limit)
	}
	return &ByteBuffer{array: b.array, offset: b.offset + index, capacity: length, limit: length, mark: -1}
}
func (b *ByteBuffer) Duplicate() *ByteBuffer {
	ReferenceRequireNonNull(b)
	duplicate := *b
	return &duplicate
}
func (b *ByteBuffer) Get(indices ...int32) int8 {
	ReferenceRequireNonNull(b)
	index := b.position
	if len(indices) == 1 {
		index = indices[0]
		byteBufferCheckRange(index, 1, b.limit)
	} else if b.position == b.limit {
		panic(newThrowableBase("BufferUnderflowException", ""))
	}
	value := b.array.Elements[b.offset+index]
	if len(indices) == 0 {
		b.position++
	}
	return value
}
func (b *ByteBuffer) GetInto(target *PrimitiveArray[int8], bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(b)
	ReferenceRequireNonNull(target)
	offset, length := int32(0), int32(len(target.Elements))
	if len(bounds) == 2 {
		offset, length = bounds[0], bounds[1]
	}
	byteBufferCheckRange(offset, length, int32(len(target.Elements)))
	if length > b.Remaining() {
		panic(newThrowableBase("BufferUnderflowException", ""))
	}
	start := b.offset + b.position
	copy(target.Elements[offset:offset+length], b.array.Elements[start:start+length])
	b.position += length
	return b
}
func (b *ByteBuffer) Put(value int8) *ByteBuffer {
	ReferenceRequireNonNull(b)
	if b.position == b.limit {
		panic(newThrowableBase("BufferOverflowException", ""))
	}
	b.array.Elements[b.offset+b.position] = value
	b.position++
	return b
}
func (b *ByteBuffer) PutAt(index int32, value int8) *ByteBuffer {
	ReferenceRequireNonNull(b)
	byteBufferCheckRange(index, 1, b.limit)
	b.array.Elements[b.offset+index] = value
	return b
}
func (b *ByteBuffer) PutArray(source *PrimitiveArray[int8], bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(b)
	ReferenceRequireNonNull(source)
	offset, length := int32(0), int32(len(source.Elements))
	if len(bounds) == 2 {
		offset, length = bounds[0], bounds[1]
	}
	byteBufferCheckRange(offset, length, int32(len(source.Elements)))
	if length > b.Remaining() {
		panic(newThrowableBase("BufferOverflowException", ""))
	}
	start := b.offset + b.position
	copy(b.array.Elements[start:start+length], source.Elements[offset:offset+length])
	b.position += length
	return b
}
func (b *ByteBuffer) PutBuffer(source *ByteBuffer) *ByteBuffer {
	ReferenceRequireNonNull(b)
	if b == source {
		panic(NewIllegalArgumentException("source buffer is this buffer"))
	}
	ReferenceRequireNonNull(source)
	length := source.Remaining()
	if length > b.Remaining() {
		panic(newThrowableBase("BufferOverflowException", ""))
	}
	destinationStart, sourceStart := b.offset+b.position, source.offset+source.position
	// copy preserves System.arraycopy/memmove behavior for overlapping views.
	copy(b.array.Elements[destinationStart:destinationStart+length], source.array.Elements[sourceStart:sourceStart+length])
	b.position += length
	source.position += length
	return b
}
func byteBufferCheckRange(offset, length, limit int32) {
	if offset < 0 || length < 0 || offset > limit-length {
		panic(NewIndexOutOfBoundsException("ByteBuffer range"))
	}
}
func (*ByteBuffer) JavaDynamicTypeID() TypeID { return "ByteBuffer" }
func init() {
	RegisterJavaType("ByteBuffer", ObjectTypeID)
	RegisterException("BufferUnderflowException", "RuntimeException")
	RegisterException("BufferOverflowException", "RuntimeException")
	RegisterException("InvalidMarkException", "IllegalStateException")
}
