package stdjava

// ByteBuffer provides the array-backed position/limit and bulk-read operations
// used by Java codecs. The backing Java byte[] retains identity and mutations.
// Direct buffers, marks, byte order and primitive views are not yet modeled.
type ByteBuffer struct {
	array    *PrimitiveArray[int8]
	position int32
	limit    int32
}

func ByteBufferWrap(array *PrimitiveArray[int8], bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(array)
	length := int32(len(array.Elements))
	buffer := &ByteBuffer{array: array, limit: length}
	if len(bounds) == 2 {
		offset, count := bounds[0], bounds[1]
		if offset < 0 || count < 0 || offset > length-count {
			panic(NewIndexOutOfBoundsException("ByteBuffer.wrap range"))
		}
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
func (b *ByteBuffer) Remaining() int32             { return b.limit - b.position }
func (b *ByteBuffer) HasArray() bool               { return true }
func (b *ByteBuffer) Array() *PrimitiveArray[int8] { return b.array }
func (b *ByteBuffer) Position() int32              { return b.position }
func (b *ByteBuffer) Limit() int32                 { return b.limit }
func (b *ByteBuffer) SetPosition(position int32) *ByteBuffer {
	if position < 0 || position > b.limit {
		panic(NewIllegalArgumentException("position out of bounds"))
	}
	b.position = position
	return b
}
func (b *ByteBuffer) GetInto(target *PrimitiveArray[int8], bounds ...int32) *ByteBuffer {
	ReferenceRequireNonNull(target)
	offset, length := int32(0), int32(len(target.Elements))
	if len(bounds) == 2 {
		offset, length = bounds[0], bounds[1]
	}
	if offset < 0 || length < 0 || offset > int32(len(target.Elements))-length {
		panic(NewIndexOutOfBoundsException("ByteBuffer.get range"))
	}
	if length > b.Remaining() {
		panic(newThrowableBase("BufferUnderflowException", ""))
	}
	copy(target.Elements[offset:offset+length], b.array.Elements[b.position:b.position+length])
	b.position += length
	return b
}
func (*ByteBuffer) JavaDynamicTypeID() TypeID { return "ByteBuffer" }
func init() {
	RegisterJavaType("ByteBuffer", ObjectTypeID)
	RegisterException("BufferUnderflowException", "RuntimeException")
}
