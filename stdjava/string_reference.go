package stdjava

import (
	"fmt"
	"slices"
	"sync/atomic"
)

// JavaString is the reference-bearing String core. The current generated String
// ABI is still native Go string; this type is not a compatibility conversion.
// Its private UTF16 payload is immutable, including isolated surrogate units.
// A wrapper is a distinct, nonzero-sized, pointer-bearing Java allocation.
type JavaString struct {
	units []uint16
	// Bit32 records whether the low32 bits contain a cached Java content hash.
	// This also distinguishes an uncached value from a computed zero hash.
	cachedHash atomic.Uint64
}

// NewJavaStringUTF16 copies a host code-unit sequence into a fresh Java object.
// A nil host slice denotes empty content. A Java char[] constructor must check
// the Java array reference for null before extracting its units for this helper.
func NewJavaStringUTF16(units []uint16) *JavaString {
	return &JavaString{units: slices.Clone(units)}
}

// CopyJavaString creates a new wrapper sharing only immutable backing data.
func CopyJavaString(original *JavaString) *JavaString {
	ReferenceRequireNonNull(original)
	copy := &JavaString{units: original.units}
	copy.cachedHash.Store(original.cachedHash.Load())
	return copy
}

// JavaDynamicTypeID preserves nominal Java identity through erased references.
// String's existing registry edges describe its Java interfaces; implementing
// their complete generated-call ABI belongs to the later compiler migration.
func (*JavaString) JavaDynamicTypeID() TypeID { return StringTypeID }

func (s *JavaString) Length() int32 {
	RequireJavaString(s)
	return int32(len(s.units))
}

func (s *JavaString) CharAt(index int32) rune {
	RequireJavaString(s)
	if index < 0 || int64(index) >= int64(len(s.units)) {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("Index %d out of bounds for length %d", index, len(s.units))))
	}
	return rune(s.units[index])
}

// UTF16Copy never exposes the internal immutable backing storage.
func (s *JavaString) UTF16Copy() []uint16 {
	ReferenceRequireNonNull(s)
	return slices.Clone(s.units)
}

func (s *JavaString) Equals(other any) bool {
	ReferenceRequireNonNull(s)
	value, ok := other.(*JavaString)
	return ok && value != nil && (s == value || slices.Equal(s.units, value.units))
}

func (s *JavaString) HashCode() int32 {
	ReferenceRequireNonNull(s)
	if cached := s.cachedHash.Load(); cached>>32 != 0 {
		return int32(uint32(cached))
	}
	var hash uint32
	index := 0
	for len(s.units)-index >= 4 {
		units := s.units[index : index+4]
		hash = 923521*hash + 29791*uint32(units[0]) + 961*uint32(units[1]) + 31*uint32(units[2]) + uint32(units[3])
		index += 4
	}
	for ; index < len(s.units); index++ {
		hash = 31*hash + uint32(s.units[index])
	}
	// Racing calculations are deterministic and publish the same result.
	s.cachedHash.Store(uint64(1)<<32 | uint64(hash))
	return int32(hash)
}

func (s *JavaString) CompareTo(other *JavaString) int32 {
	ReferenceRequireNonNull(s)
	ReferenceRequireNonNull(other)
	for index := 0; index < len(s.units) && index < len(other.units); index++ {
		if difference := int32(s.units[index]) - int32(other.units[index]); difference != 0 {
			return difference
		}
	}
	return int32(len(s.units)) - int32(len(other.units))
}
