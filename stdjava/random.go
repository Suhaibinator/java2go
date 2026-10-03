package stdjava

import (
	"crypto/rand"
	"encoding/binary"
	"sync/atomic"
)

const randomMultiplier uint64 = 0x5deece66d
const randomMask uint64 = (1 << 48) - 1

// Random implements java.util.Random's specified 48-bit generator. Atomic seed
// transitions retain its thread-safe sequence; compound calls may interleave.
// Gaussian generation, streams and subclass overrides are not yet modeled.
type Random struct{ seed atomic.Uint64 }

func NewRandom(seeds ...int64) *Random {
	var seed int64
	if len(seeds) == 0 {
		var entropy [8]byte
		if _, err := rand.Read(entropy[:]); err != nil {
			panic(NewRuntimeException(err.Error()))
		}
		seed = int64(binary.LittleEndian.Uint64(entropy[:]))
	} else {
		seed = seeds[0]
	}
	r := &Random{}
	r.SetSeed(seed)
	return r
}
func (r *Random) SetSeed(seed int64) { r.seed.Store((uint64(seed) ^ randomMultiplier) & randomMask) }
func (r *Random) next(bits uint) int32 {
	for {
		old := r.seed.Load()
		next := (old*randomMultiplier + 11) & randomMask
		if r.seed.CompareAndSwap(old, next) {
			return int32(next >> (48 - bits))
		}
	}
}
func (r *Random) NextInt(bounds ...int32) int32 {
	if len(bounds) == 0 {
		return r.next(32)
	}
	bound := bounds[0]
	if bound <= 0 {
		panic(NewIllegalArgumentException("bound must be positive"))
	}
	if bound&-bound == bound {
		return int32((int64(bound) * int64(r.next(31))) >> 31)
	}
	for {
		bits := r.next(31)
		value := bits % bound
		// Java's signed 32-bit overflow is part of the rejection condition.
		if bits-value+(bound-1) >= 0 {
			return value
		}
	}
}
func (r *Random) NextBoolean() bool  { return r.next(1) != 0 }
func (r *Random) NextLong() int64    { return (int64(r.next(32)) << 32) + int64(r.next(32)) }
func (r *Random) NextFloat() float32 { return float32(r.next(24)) / float32(1<<24) }
func (r *Random) NextDouble() float64 {
	return float64((int64(r.next(26))<<27)+int64(r.next(27))) / float64(1<<53)
}
func (r *Random) NextBytes(bytes *PrimitiveArray[int8]) {
	ReferenceRequireNonNull(bytes)
	for i := 0; i < len(bytes.Elements); {
		value := r.next(32)
		for n := 0; n < 4 && i < len(bytes.Elements); n++ {
			bytes.Elements[i] = int8(value)
			i++
			value >>= 8
		}
	}
}
func (*Random) JavaDynamicTypeID() TypeID { return "Random" }
func init()                               { RegisterJavaType("Random", ObjectTypeID) }
