package stdjava

import (
	"strconv"
	"sync/atomic"
)

// Atomic arrays keep their fixed-length storage private. Individual elements
// have volatile/atomic semantics; copying a primitive array never aliases it.

const AtomicIntegerArrayTypeID TypeID = "java.util.concurrent.atomic.AtomicIntegerArray"

type AtomicIntegerArray struct{ elements []atomic.Int32 }

func (*AtomicIntegerArray) JavaDynamicTypeID() TypeID { return AtomicIntegerArrayTypeID }

func NewAtomicIntegerArray(length int32) *AtomicIntegerArray {
	if length < 0 {
		panic(NewNegativeArraySizeException(strconv.FormatInt(int64(length), 10)))
	}
	return &AtomicIntegerArray{elements: make([]atomic.Int32, int(length))}
}

func NewAtomicIntegerArrayFromArray(source *PrimitiveArray[int32]) *AtomicIntegerArray {
	if source == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	result := NewAtomicIntegerArray(int32(len(source.Elements)))
	for index, value := range source.Elements {
		result.elements[index].Store(value)
	}
	return result
}

func (array *AtomicIntegerArray) Length() int32 {
	if array == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return int32(len(array.elements))
}

func (array *AtomicIntegerArray) element(index int32) *atomic.Int32 {
	if array == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	if index < 0 || int64(index) >= int64(len(array.elements)) {
		panic(NewArrayIndexOutOfBoundsException("Index " + strconv.FormatInt(int64(index), 10) + " out of bounds for length " + strconv.Itoa(len(array.elements))))
	}
	return &array.elements[index]
}

func (array *AtomicIntegerArray) Get(index int32) int32        { return array.element(index).Load() }
func (array *AtomicIntegerArray) Set(index int32, value int32) { array.element(index).Store(value) }
func (array *AtomicIntegerArray) CompareAndSet(index int32, expected, update int32) bool {
	return array.element(index).CompareAndSwap(expected, update)
}
func (array *AtomicIntegerArray) GetAndSet(index int32, value int32) int32 {
	return array.element(index).Swap(value)
}
func (array *AtomicIntegerArray) GetAndAdd(index int32, delta int32) int32 {
	return array.element(index).Add(delta) - delta
}
func (array *AtomicIntegerArray) AddAndGet(index int32, delta int32) int32 {
	return array.element(index).Add(delta)
}

const AtomicLongArrayTypeID TypeID = "java.util.concurrent.atomic.AtomicLongArray"

type AtomicLongArray struct{ elements []atomic.Int64 }

func (*AtomicLongArray) JavaDynamicTypeID() TypeID { return AtomicLongArrayTypeID }

func NewAtomicLongArray(length int32) *AtomicLongArray {
	if length < 0 {
		panic(NewNegativeArraySizeException(strconv.FormatInt(int64(length), 10)))
	}
	return &AtomicLongArray{elements: make([]atomic.Int64, int(length))}
}

func NewAtomicLongArrayFromArray(source *PrimitiveArray[int64]) *AtomicLongArray {
	if source == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	result := NewAtomicLongArray(int32(len(source.Elements)))
	for index, value := range source.Elements {
		result.elements[index].Store(value)
	}
	return result
}

func (array *AtomicLongArray) Length() int32 {
	if array == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	return int32(len(array.elements))
}

func (array *AtomicLongArray) element(index int32) *atomic.Int64 {
	if array == nil {
		panic(NewJavaNullPointerExceptionMessage(nil))
	}
	if index < 0 || int64(index) >= int64(len(array.elements)) {
		panic(NewArrayIndexOutOfBoundsException("Index " + strconv.FormatInt(int64(index), 10) + " out of bounds for length " + strconv.Itoa(len(array.elements))))
	}
	return &array.elements[index]
}

func (array *AtomicLongArray) Get(index int32) int64        { return array.element(index).Load() }
func (array *AtomicLongArray) Set(index int32, value int64) { array.element(index).Store(value) }
func (array *AtomicLongArray) CompareAndSet(index int32, expected, update int64) bool {
	return array.element(index).CompareAndSwap(expected, update)
}
func (array *AtomicLongArray) GetAndSet(index int32, value int64) int64 {
	return array.element(index).Swap(value)
}
func (array *AtomicLongArray) GetAndAdd(index int32, delta int64) int64 {
	return array.element(index).Add(delta) - delta
}
func (array *AtomicLongArray) AddAndGet(index int32, delta int64) int64 {
	return array.element(index).Add(delta)
}

func init() {
	RegisterJavaType(AtomicIntegerArrayTypeID, ObjectTypeID, SerializableTypeID)
	RegisterJavaType(AtomicLongArrayTypeID, ObjectTypeID, SerializableTypeID)
}
