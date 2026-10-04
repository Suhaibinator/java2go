package stdjava

import "math"

// ByteOrder is the identity-bearing JDK singleton model used by ByteBuffer.
// Its constructors are not public in Java; buffers initially use BIG_ENDIAN.
type ByteOrder struct{ name string }

var ByteOrderBIG_ENDIAN = &ByteOrder{name: "BIG_ENDIAN"}
var ByteOrderLITTLE_ENDIAN = &ByteOrder{name: "LITTLE_ENDIAN"}

func (o *ByteOrder) ToString() string { ReferenceRequireNonNull(o); return o.name }
func (o *ByteOrder) String() string   { return o.ToString() }

// StringJava2goExecution exposes the JDK singleton's immutable name reference
// at Java String.valueOf, concatenation, printing and virtual toString boundaries.
// The two JDK names are ASCII literals; literal interning preserves their identity.
func (o *ByteOrder) StringJava2goExecution(*Execution) *JavaString {
	ReferenceRequireNonNull(o)
	units := make([]uint16, len(o.name))
	for index := range o.name {
		units[index] = uint16(o.name[index])
	}
	return JavaStringLiteralUTF16(units)
}
func (*ByteOrder) JavaDynamicTypeID() TypeID { return "ByteOrder" }
func init()                                  { RegisterJavaType("ByteOrder", ObjectTypeID) }

func (b *ByteBuffer) Order() *ByteOrder {
	ReferenceRequireNonNull(b)
	if b.littleEndian {
		return ByteOrderLITTLE_ENDIAN
	}
	return ByteOrderBIG_ENDIAN
}
func (b *ByteBuffer) SetOrder(order *ByteOrder) *ByteBuffer {
	ReferenceRequireNonNull(b)
	// JDK uses an identity comparison with BIG_ENDIAN, including for null.
	b.littleEndian = order != ByteOrderBIG_ENDIAN
	return b
}
func (b *ByteBuffer) primitiveIndex(width int32, write bool, indices []int32) int32 {
	ReferenceRequireNonNull(b)
	if len(indices) == 1 {
		byteBufferCheckRange(indices[0], width, b.limit)
		return indices[0]
	}
	if width > b.limit-b.position {
		if write {
			panic(newThrowableBase("BufferOverflowException", ""))
		}
		panic(newThrowableBase("BufferUnderflowException", ""))
	}
	return b.position
}
func (b *ByteBuffer) readPrimitiveBits(width int32, indices ...int32) uint64 {
	index := b.primitiveIndex(width, false, indices)
	start := b.offset + index
	var bits uint64
	for i := int32(0); i < width; i++ {
		shift := (width - 1 - i) * 8
		if b.littleEndian {
			shift = i * 8
		}
		bits |= uint64(uint8(b.array.Elements[start+i])) << uint(shift)
	}
	if len(indices) == 0 {
		b.position += width
	}
	return bits
}
func (b *ByteBuffer) writePrimitiveBits(width int32, bits uint64, indices ...int32) *ByteBuffer {
	byteBufferRequireWritable(b)
	index := b.primitiveIndex(width, true, indices)
	start := b.offset + index
	for i := int32(0); i < width; i++ {
		shift := (width - 1 - i) * 8
		if b.littleEndian {
			shift = i * 8
		}
		b.array.Elements[start+i] = int8(bits >> uint(shift))
	}
	if len(indices) == 0 {
		b.position += width
	}
	return b
}

func (b *ByteBuffer) GetShort(indices ...int32) int16 {
	return int16(b.readPrimitiveBits(2, indices...))
}
func (b *ByteBuffer) PutShort(value int16) *ByteBuffer {
	return b.writePrimitiveBits(2, uint64(uint16(value)))
}
func (b *ByteBuffer) PutShortAt(index int32, value int16) *ByteBuffer {
	return b.writePrimitiveBits(2, uint64(uint16(value)), index)
}

func (b *ByteBuffer) GetInt(indices ...int32) int32 { return int32(b.readPrimitiveBits(4, indices...)) }
func (b *ByteBuffer) PutInt(value int32) *ByteBuffer {
	return b.writePrimitiveBits(4, uint64(uint32(value)))
}
func (b *ByteBuffer) PutIntAt(index int32, value int32) *ByteBuffer {
	return b.writePrimitiveBits(4, uint64(uint32(value)), index)
}

func (b *ByteBuffer) GetLong(indices ...int32) int64 {
	return int64(b.readPrimitiveBits(8, indices...))
}
func (b *ByteBuffer) PutLong(value int64) *ByteBuffer { return b.writePrimitiveBits(8, uint64(value)) }
func (b *ByteBuffer) PutLongAt(index int32, value int64) *ByteBuffer {
	return b.writePrimitiveBits(8, uint64(value), index)
}

func (b *ByteBuffer) GetFloat(indices ...int32) float32 {
	return math.Float32frombits(uint32(b.readPrimitiveBits(4, indices...)))
}
func (b *ByteBuffer) PutFloat(value float32) *ByteBuffer {
	return b.writePrimitiveBits(4, uint64(math.Float32bits(value)))
}
func (b *ByteBuffer) PutFloatAt(index int32, value float32) *ByteBuffer {
	return b.writePrimitiveBits(4, uint64(math.Float32bits(value)), index)
}

func (b *ByteBuffer) GetDouble(indices ...int32) float64 {
	return math.Float64frombits(b.readPrimitiveBits(8, indices...))
}
func (b *ByteBuffer) PutDouble(value float64) *ByteBuffer {
	return b.writePrimitiveBits(8, math.Float64bits(value))
}
func (b *ByteBuffer) PutDoubleAt(index int32, value float64) *ByteBuffer {
	return b.writePrimitiveBits(8, math.Float64bits(value), index)
}

func (b *ByteBuffer) GetChar(indices ...int32) rune { return rune(b.readPrimitiveBits(2, indices...)) }
func (b *ByteBuffer) PutChar(value rune) *ByteBuffer {
	return b.writePrimitiveBits(2, uint64(uint16(value)))
}
func (b *ByteBuffer) PutCharAt(index int32, value rune) *ByteBuffer {
	return b.writePrimitiveBits(2, uint64(uint16(value)), index)
}
