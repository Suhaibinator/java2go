package stdjava

import "math/big"

// BigInteger owns its magnitude. No operation exposes or mutates that storage.
type BigInteger struct{ value big.Int }

var bigIntegerSmallValues = func() [33]*BigInteger {
	var values [33]*BigInteger
	for i := range values {
		values[i] = &BigInteger{}
		values[i].value.SetInt64(int64(i - 16))
	}
	return values
}()

func NewBigInteger(text string) *BigInteger { return &BigInteger{value: parseBigIntegerDecimal(text)} }

// NewBigIntegerJavaString is the canonical Java constructor boundary. The parser
// reads the original UTF16 units and retains them in NumberFormatException.
func NewBigIntegerJavaString(text *JavaString) *BigInteger {
	return &BigInteger{value: parseBigIntegerDecimalJavaString(text)}
}

func BigIntegerValueOf(value int64) *BigInteger {
	if value >= -16 && value <= 16 {
		return bigIntegerSmallValues[value+16]
	}
	result := &BigInteger{}
	result.value.SetInt64(value)
	return result
}
func (*BigInteger) JavaDynamicTypeID() TypeID { return "java.math.BigInteger" }
func (value *BigInteger) String() string      { ReferenceRequireNonNull(value); return value.value.String() }

// StringJava2goExecution implements the nominal Java toString result ABI.
// The native String method remains available for legacy Go callers.
func (value *BigInteger) StringJava2goExecution(_ *Execution) *JavaString {
	ReferenceRequireNonNull(value)
	if value.value.Sign() == 0 {
		return JavaStringLiteralUTF16([]uint16{'0'})
	}
	return bigNumberJavaString(value.String())
}

func (value *BigInteger) Equals(other any) bool {
	ReferenceRequireNonNull(value)
	right, ok := other.(*BigInteger)
	return ok && right != nil && value.value.Cmp(&right.value) == 0
}
func (value *BigInteger) CompareTo(other *BigInteger) int32 {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	return int32(value.value.Cmp(&other.value))
}

func (value *BigInteger) Add(other *BigInteger) *BigInteger {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	if other.value.Sign() == 0 {
		return value
	}
	if value.value.Sign() == 0 {
		return other
	}
	result := &BigInteger{}
	result.value.Add(&value.value, &other.value)
	return bigIntegerArithmeticResult(result)
}

func (value *BigInteger) Subtract(other *BigInteger) *BigInteger {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	if other.value.Sign() == 0 {
		return value
	}
	result := &BigInteger{}
	result.value.Sub(&value.value, &other.value)
	return bigIntegerArithmeticResult(result)
}

func (value *BigInteger) Multiply(other *BigInteger) *BigInteger {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	result := &BigInteger{}
	result.value.Mul(&value.value, &other.value)
	return bigIntegerArithmeticResult(result)
}

func (value *BigInteger) Divide(other *BigInteger) *BigInteger {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	if other.value.Sign() == 0 {
		panic(NewArithmeticException("BigInteger divide by zero"))
	}
	result := &BigInteger{}
	// Java integer division truncates toward zero. big.Int.Div instead uses
	// Euclidean division, which gives a different quotient for negative values.
	result.value.Quo(&value.value, &other.value)
	return bigIntegerArithmeticResult(result)
}

func (value *BigInteger) Mod(other *BigInteger) *BigInteger {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	if other.value.Sign() <= 0 {
		panic(NewArithmeticException("BigInteger: modulus not positive"))
	}
	result := &BigInteger{}
	result.value.Mod(&value.value, &other.value)
	return bigIntegerArithmeticResult(result)
}

// Arithmetic returns the canonical Java ZERO when a computation cancels to
// zero; nonzero results own fresh magnitude storage rather than valueOf caches.
func bigIntegerArithmeticResult(result *BigInteger) *BigInteger {
	if result.value.Sign() == 0 {
		return BigIntegerValueOf(0)
	}
	return result
}

func (value *BigInteger) BitLength() int32 {
	ReferenceRequireNonNull(value)
	if value.value.Sign() >= 0 {
		return int32(value.value.BitLen())
	}
	// Java excludes the sign bit from the minimal two's-complement encoding.
	// For negative n that is the unsigned bit length of ~n, including -1 -> 0.
	var complement big.Int
	complement.Not(&value.value)
	return int32(complement.BitLen())
}
func (value *BigInteger) HashCode() int32 {
	ReferenceRequireNonNull(value)
	return bigNumberIntegerHash(&value.value)
}
func (value *BigInteger) LongValue() int64 {
	ReferenceRequireNonNull(value)
	return bigNumberLowLong(&value.value)
}
func (value *BigInteger) IntValue() int32      { return int32(value.LongValue()) }
func (value *BigInteger) ShortValue() int16    { return int16(value.IntValue()) }
func (value *BigInteger) ByteValue() int8      { return int8(value.IntValue()) }
func (value *BigInteger) FloatValue() float32  { return float32(bigNumberFloat(value.String(), 32)) }
func (value *BigInteger) DoubleValue() float64 { return bigNumberFloat(value.String(), 64) }
func init()                                    { RegisterJavaType("java.math.BigInteger", NumberTypeID, ComparableTypeID) }
