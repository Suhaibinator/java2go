package stdjava

import (
	"math"
	"math/big"
	"strconv"
)

func bigNumberMagnitudeText(value *big.Int) string {
	text := value.String()
	if value.Sign() < 0 {
		return text[1:]
	}
	return text
}
func bigNumberLowLong(value *big.Int) int64 {
	low := value.Uint64()
	if value.Sign() < 0 {
		low = 0 - low
	}
	return int64(low)
}
func bigNumberIntegerHash(value *big.Int) int32 {
	magnitude := value.Bytes()
	var hash, word uint32
	for index, digit := range magnitude {
		word = word<<8 | uint32(digit)
		if (len(magnitude)-index-1)%4 == 0 {
			hash = 31*hash + word
			word = 0
		}
	}
	if value.Sign() < 0 {
		hash = 0 - hash
	}
	return int32(hash)
}
func bigNumberFloat(text string, bits int) float64 {
	// ParseFloat rounds directly at the requested IEEE width. A range error
	// reports the required infinity result rather than a Java parsing failure.
	value, err := strconv.ParseFloat(text, bits)
	if err != nil {
		if failure, ok := err.(*strconv.NumError); !ok || failure.Err != strconv.ErrRange {
			panic(NewNumberFormatException(err.Error()))
		}
	}
	return value
}
func FloatToRawIntBits(value float32) int32   { return int32(math.Float32bits(value)) }
func DoubleToRawLongBits(value float64) int64 { return int64(math.Float64bits(value)) }
func FloatToIntBits(value float32) int32      { return int32(canonicalFloatBits(value)) }
func DoubleToLongBits(value float64) int64    { return int64(canonicalDoubleBits(value)) }
