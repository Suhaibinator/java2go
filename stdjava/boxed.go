package stdjava

import (
	"cmp"
	"math"
	"strconv"
)

// Java wrapper objects have a distinct nominal type and an immutable payload.
// Automatic boxing and valueOf share the fixed Java caches; constructors always
// allocate, so equal values can still have different reference identities.
type Boolean struct{ value bool }
type Byte struct{ value int8 }
type Short struct{ value int16 }
type Character struct{ value uint16 }
type Integer struct{ value int32 }
type Long struct{ value int64 }
type Float struct{ value float32 }
type Double struct{ value float64 }

var booleanCache = [2]Boolean{{value: false}, {value: true}}
var byteCache = func() (cache [256]Byte) {
	for i := range cache {
		cache[i].value = int8(i - 128)
	}
	return
}()
var characterCache = func() (cache [128]Character) {
	for i := range cache {
		cache[i].value = uint16(i)
	}
	return
}()
var shortCache = func() (cache [256]Short) {
	for i := range cache {
		cache[i].value = int16(i - 128)
	}
	return
}()
var integerCache = func() (cache [256]Integer) {
	for i := range cache {
		cache[i].value = int32(i - 128)
	}
	return
}()
var longCache = func() (cache [256]Long) {
	for i := range cache {
		cache[i].value = int64(i - 128)
	}
	return
}()

// requireBox makes every direct wrapper instance invocation, including equals
// and toString, throw Java NullPointerException on a null receiver.
func requireBox[T any](value *T) *T {
	if value == nil {
		panic(NewNullPointerException("wrapper value is null"))
	}
	return value
}

func BoxBoolean(value bool) *Boolean {
	if value {
		return &booleanCache[1]
	}
	return &booleanCache[0]
}
func BoxByte(value int8) *Byte { return &byteCache[int(value)+128] }
func BoxCharacter(value rune) *Character {
	narrowed := uint16(value)
	if narrowed < 128 {
		return &characterCache[narrowed]
	}
	return NewCharacter(rune(narrowed))
}
func BoxShort(value int16) *Short {
	if value >= -128 && value <= 127 {
		return &shortCache[int(value)+128]
	}
	return NewShort(value)
}
func BoxInteger(value int32) *Integer {
	if value >= -128 && value <= 127 {
		return &integerCache[int(value)+128]
	}
	return NewInteger(value)
}
func BoxLong(value int64) *Long {
	if value >= -128 && value <= 127 {
		return &longCache[int(value)+128]
	}
	return NewLong(value)
}
func BoxFloat(value float32) *Float   { return NewFloat(value) }
func BoxDouble(value float64) *Double { return NewDouble(value) }

func NewBoolean(value bool) *Boolean   { return &Boolean{value: value} }
func UnboxBoolean(value *Boolean) bool { return requireBox(value).value }
func (value *Boolean) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return BooleanTypeID
}
func (value *Boolean) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Boolean)
	return ok && other != nil && payload == other.value
}
func (value *Boolean) HashCode() int32 {
	if value.BooleanValue() {
		return 1231
	}
	return 1237
}
func (value *Boolean) CompareTo(other *Boolean) int32 {
	left, right := value.BooleanValue(), other.BooleanValue()
	if left == right {
		return 0
	}
	if left {
		return 1
	}
	return -1
}
func (value *Boolean) String() string     { return strconv.FormatBool(value.BooleanValue()) }
func (value *Boolean) BooleanValue() bool { return UnboxBoolean(value) }

func NewByte(value int8) *Byte   { return &Byte{value: value} }
func UnboxByte(value *Byte) int8 { return requireBox(value).value }
func (value *Byte) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return ByteTypeID
}
func (value *Byte) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Byte)
	return ok && other != nil && payload == other.value
}
func (value *Byte) HashCode() int32 { return int32(value.ByteValue()) }
func (value *Byte) CompareTo(other *Byte) int32 {
	return int32(value.ByteValue()) - int32(other.ByteValue())
}
func (value *Byte) String() string       { return strconv.FormatInt(int64(value.ByteValue()), 10) }
func (value *Byte) ByteValue() int8      { return UnboxByte(value) }
func (value *Byte) ShortValue() int16    { return int16(UnboxByte(value)) }
func (value *Byte) IntValue() int32      { return int32(UnboxByte(value)) }
func (value *Byte) LongValue() int64     { return int64(UnboxByte(value)) }
func (value *Byte) FloatValue() float32  { return float32(UnboxByte(value)) }
func (value *Byte) DoubleValue() float64 { return float64(UnboxByte(value)) }

func NewShort(value int16) *Short   { return &Short{value: value} }
func UnboxShort(value *Short) int16 { return requireBox(value).value }
func (value *Short) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return ShortTypeID
}
func (value *Short) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Short)
	return ok && other != nil && payload == other.value
}
func (value *Short) HashCode() int32 { return int32(value.ShortValue()) }
func (value *Short) CompareTo(other *Short) int32 {
	return int32(value.ShortValue()) - int32(other.ShortValue())
}
func (value *Short) String() string       { return strconv.FormatInt(int64(value.ShortValue()), 10) }
func (value *Short) ByteValue() int8      { return int8(UnboxShort(value)) }
func (value *Short) ShortValue() int16    { return UnboxShort(value) }
func (value *Short) IntValue() int32      { return int32(UnboxShort(value)) }
func (value *Short) LongValue() int64     { return int64(UnboxShort(value)) }
func (value *Short) FloatValue() float32  { return float32(UnboxShort(value)) }
func (value *Short) DoubleValue() float64 { return float64(UnboxShort(value)) }

func NewCharacter(value rune) *Character   { return &Character{value: uint16(value)} }
func UnboxCharacter(value *Character) rune { return rune(requireBox(value).value) }
func (value *Character) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return CharacterTypeID
}
func (value *Character) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Character)
	return ok && other != nil && payload == other.value
}
func (value *Character) HashCode() int32 { return int32(value.CharValue()) }
func (value *Character) CompareTo(other *Character) int32 {
	return int32(value.CharValue()) - int32(other.CharValue())
}
func (value *Character) String() string  { return string(value.CharValue()) }
func (value *Character) CharValue() rune { return UnboxCharacter(value) }

func NewInteger(value int32) *Integer   { return &Integer{value: value} }
func UnboxInteger(value *Integer) int32 { return requireBox(value).value }
func (value *Integer) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return IntegerTypeID
}
func (value *Integer) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Integer)
	return ok && other != nil && payload == other.value
}
func (value *Integer) HashCode() int32 { return value.IntValue() }
func (value *Integer) CompareTo(other *Integer) int32 {
	return int32(cmp.Compare(value.IntValue(), other.IntValue()))
}
func (value *Integer) String() string       { return strconv.FormatInt(int64(value.IntValue()), 10) }
func (value *Integer) ByteValue() int8      { return int8(UnboxInteger(value)) }
func (value *Integer) ShortValue() int16    { return int16(UnboxInteger(value)) }
func (value *Integer) IntValue() int32      { return UnboxInteger(value) }
func (value *Integer) LongValue() int64     { return int64(UnboxInteger(value)) }
func (value *Integer) FloatValue() float32  { return float32(UnboxInteger(value)) }
func (value *Integer) DoubleValue() float64 { return float64(UnboxInteger(value)) }

func NewLong(value int64) *Long   { return &Long{value: value} }
func UnboxLong(value *Long) int64 { return requireBox(value).value }
func (value *Long) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return LongTypeID
}
func (value *Long) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Long)
	return ok && other != nil && payload == other.value
}
func (value *Long) HashCode() int32 {
	bits := uint64(value.LongValue())
	return int32(bits ^ (bits >> 32))
}
func (value *Long) CompareTo(other *Long) int32 {
	return int32(cmp.Compare(value.LongValue(), other.LongValue()))
}
func (value *Long) String() string       { return strconv.FormatInt(int64(value.LongValue()), 10) }
func (value *Long) ByteValue() int8      { return int8(UnboxLong(value)) }
func (value *Long) ShortValue() int16    { return int16(UnboxLong(value)) }
func (value *Long) IntValue() int32      { return int32(UnboxLong(value)) }
func (value *Long) LongValue() int64     { return UnboxLong(value) }
func (value *Long) FloatValue() float32  { return float32(UnboxLong(value)) }
func (value *Long) DoubleValue() float64 { return float64(UnboxLong(value)) }

func NewFloat(value float32) *Float   { return &Float{value: value} }
func UnboxFloat(value *Float) float32 { return requireBox(value).value }
func (value *Float) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return FloatTypeID
}
func (value *Float) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Float)
	return ok && other != nil && canonicalFloatBits(payload) == canonicalFloatBits(other.value)
}
func (value *Float) HashCode() int32 { return int32(canonicalFloatBits(value.FloatValue())) }
func (value *Float) CompareTo(other *Float) int32 {
	return javaFloatCompare(value.FloatValue(), other.FloatValue())
}
func (value *Float) String() string       { return FloatToString(value.FloatValue()) }
func (value *Float) ByteValue() int8      { return int8(numberFloatIntValue(float64(UnboxFloat(value)))) }
func (value *Float) ShortValue() int16    { return int16(numberFloatIntValue(float64(UnboxFloat(value)))) }
func (value *Float) IntValue() int32      { return numberFloatIntValue(float64(UnboxFloat(value))) }
func (value *Float) LongValue() int64     { return numberFloatLongValue(float64(UnboxFloat(value))) }
func (value *Float) FloatValue() float32  { return UnboxFloat(value) }
func (value *Float) DoubleValue() float64 { return float64(UnboxFloat(value)) }

func NewDouble(value float64) *Double   { return &Double{value: value} }
func UnboxDouble(value *Double) float64 { return requireBox(value).value }
func (value *Double) JavaDynamicTypeID() TypeID {
	requireBox(value)
	return DoubleTypeID
}
func (value *Double) Equals(argument any) bool {
	payload := requireBox(value).value
	other, ok := argument.(*Double)
	return ok && other != nil && canonicalDoubleBits(payload) == canonicalDoubleBits(other.value)
}
func (value *Double) HashCode() int32 {
	bits := canonicalDoubleBits(value.DoubleValue())
	return int32(bits ^ (bits >> 32))
}
func (value *Double) CompareTo(other *Double) int32 {
	return javaDoubleCompare(value.DoubleValue(), other.DoubleValue())
}
func (value *Double) String() string  { return DoubleToString(value.DoubleValue()) }
func (value *Double) ByteValue() int8 { return int8(numberFloatIntValue(float64(UnboxDouble(value)))) }
func (value *Double) ShortValue() int16 {
	return int16(numberFloatIntValue(float64(UnboxDouble(value))))
}
func (value *Double) IntValue() int32      { return numberFloatIntValue(float64(UnboxDouble(value))) }
func (value *Double) LongValue() int64     { return numberFloatLongValue(float64(UnboxDouble(value))) }
func (value *Double) FloatValue() float32  { return float32(UnboxDouble(value)) }
func (value *Double) DoubleValue() float64 { return UnboxDouble(value) }

// canonicalFloatBits and canonicalDoubleBits implement floatToIntBits and
// doubleToLongBits: every NaN has Java's canonical hash/equality payload.
func canonicalFloatBits(value float32) uint32 {
	if math.IsNaN(float64(value)) {
		return 0x7fc00000
	}
	return math.Float32bits(value)
}
func canonicalDoubleBits(value float64) uint64 {
	if math.IsNaN(value) {
		return 0x7ff8000000000000
	}
	return math.Float64bits(value)
}
