package stdjava

import (
	"math/big"
	"strconv"
	"strings"
	"sync/atomic"
)

// BigDecimal preserves coefficient and scale independently. Equal numeric
// values with different scales intentionally have different equals/hash values.
type BigDecimal struct {
	coefficient     big.Int
	scale           int32
	javaStringCache atomic.Pointer[JavaString]
}

func NewBigDecimal(text string) *BigDecimal {
	coefficient, scale := parseBigDecimal(text)
	return &BigDecimal{coefficient: coefficient, scale: scale}
}

// NewBigDecimalJavaString is the canonical Java constructor boundary. The parser
// reads the original UTF16 units and retains them in NumberFormatException.
func NewBigDecimalJavaString(text *JavaString) *BigDecimal {
	coefficient, scale := parseBigDecimalJavaString(text)
	return &BigDecimal{coefficient: coefficient, scale: scale}
}

func (*BigDecimal) JavaDynamicTypeID() TypeID { return "java.math.BigDecimal" }
func (value *BigDecimal) Scale() int32        { ReferenceRequireNonNull(value); return value.scale }

// StringJava2goExecution implements the nominal Java toString result ABI.
// The native String method remains available for legacy Go callers.
func (value *BigDecimal) StringJava2goExecution(_ *Execution) *JavaString {
	ReferenceRequireNonNull(value)
	if cached := value.javaStringCache.Load(); cached != nil {
		return cached
	}
	text := bigNumberJavaString(value.String())
	if value.javaStringCache.CompareAndSwap(nil, text) {
		return text
	}
	return value.javaStringCache.Load()
}

func (value *BigDecimal) Equals(other any) bool {
	ReferenceRequireNonNull(value)
	right, ok := other.(*BigDecimal)
	return ok && right != nil && value.scale == right.scale && value.coefficient.Cmp(&right.coefficient) == 0
}
func (value *BigDecimal) CompareTo(other *BigDecimal) int32 {
	ReferenceRequireNonNull(value)
	ReferenceRequireNonNull(other)
	if value.scale == other.scale {
		return int32(value.coefficient.Cmp(&other.coefficient))
	}
	leftSign, rightSign := value.coefficient.Sign(), other.coefficient.Sign()
	if leftSign < rightSign {
		return -1
	}
	if leftSign > rightSign {
		return 1
	}
	if leftSign == 0 {
		return 0
	}
	left, right := bigNumberMagnitudeText(&value.coefficient), bigNumberMagnitudeText(&other.coefficient)
	leftAdjusted, rightAdjusted := int64(len(left))-int64(value.scale), int64(len(right))-int64(other.scale)
	comparison := 0
	if leftAdjusted < rightAdjusted {
		comparison = -1
	} else if leftAdjusted > rightAdjusted {
		comparison = 1
	} else {
		// Equal adjusted exponents need only the existing significant digits;
		// virtual trailing zeros avoid constructing powers for extreme scales.
		for i := 0; i < len(left) || i < len(right); i++ {
			a, b := byte('0'), byte('0')
			if i < len(left) {
				a = left[i]
			}
			if i < len(right) {
				b = right[i]
			}
			if a < b {
				comparison = -1
				break
			}
			if a > b {
				comparison = 1
				break
			}
		}
	}
	return int32(comparison * leftSign)
}
func (value *BigDecimal) HashCode() int32 {
	ReferenceRequireNonNull(value)
	return 31*bigNumberIntegerHash(&value.coefficient) + value.scale
}
func (value *BigDecimal) String() string {
	ReferenceRequireNonNull(value)
	digits := bigNumberMagnitudeText(&value.coefficient)
	adjusted := int64(len(digits)) - 1 - int64(value.scale)
	var result string
	if value.scale >= 0 && adjusted >= -6 {
		split := int64(len(digits)) - int64(value.scale)
		if value.scale == 0 {
			result = digits
		} else if split > 0 {
			result = digits[:split] + "." + digits[split:]
		} else {
			result = "0." + strings.Repeat("0", int(-split)) + digits
		}
	} else {
		result = digits[:1]
		if len(digits) > 1 {
			result += "." + digits[1:]
		}
		result += "E"
		if adjusted >= 0 {
			result += "+"
		}
		result += strconv.FormatInt(adjusted, 10)
	}
	if value.coefficient.Sign() < 0 {
		return "-" + result
	}
	return result
}
func (value *BigDecimal) LongValue() int64 {
	ReferenceRequireNonNull(value)
	if value.coefficient.Sign() == 0 {
		return 0
	}
	if value.scale == 0 {
		return bigNumberLowLong(&value.coefficient)
	}
	if value.scale <= -64 {
		return 0
	}
	if value.scale > 0 && int64(value.scale) >= int64(len(bigNumberMagnitudeText(&value.coefficient))) {
		return 0
	}
	var magnitude big.Int
	exponent := int64(value.scale)
	if exponent < 0 {
		exponent = -exponent
	}
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(exponent), nil)
	if value.scale > 0 {
		magnitude.Quo(&value.coefficient, power)
	} else {
		magnitude.Mul(&value.coefficient, power)
	}
	return bigNumberLowLong(&magnitude)
}
func (value *BigDecimal) IntValue() int32      { return int32(value.LongValue()) }
func (value *BigDecimal) ShortValue() int16    { return int16(value.IntValue()) }
func (value *BigDecimal) ByteValue() int8      { return int8(value.IntValue()) }
func (value *BigDecimal) FloatValue() float32  { return float32(bigNumberFloat(value.String(), 32)) }
func (value *BigDecimal) DoubleValue() float64 { return bigNumberFloat(value.String(), 64) }
func init()                                    { RegisterJavaType("java.math.BigDecimal", NumberTypeID, ComparableTypeID) }
