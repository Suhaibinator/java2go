package stdjava

import "unicode/utf16"

// Native adapters exist for legacy Go callers. Canonical Java-to-Java text
// passes directly through UTF16 units without replacement or reconstruction.
func dateFormatNativeJavaString(text string) *JavaString {
	return NewJavaStringUTF16(utf16.Encode([]rune(text)))
}
func dateFormatNativeText(text *JavaString) string {
	ReferenceRequireNonNull(text)
	return string(utf16.Decode(text.units))
}
func dateFormatJavaUnits(text *JavaString) []rune {
	ReferenceRequireNonNull(text)
	units := make([]rune, len(text.units))
	for index, unit := range text.units {
		units[index] = rune(unit)
	}
	return units
}

type dateFormatUnitBuilder struct{ units []uint16 }

func (out *dateFormatUnitBuilder) WriteUnits(units []uint16) { out.units = append(out.units, units...) }

// Only locale symbols, zone metadata, and numeric fragments enter this scalar adapter.
func (out *dateFormatUnitBuilder) WriteString(text string) {
	out.units = append(out.units, utf16.Encode([]rune(text))...)
}

// dateFormatScalarLookupText validates a metadata lookup key before decoding.
// A failed key is never passed to a host parser with replacement characters.
func dateFormatScalarLookupText(text *JavaString) (string, bool) {
	ReferenceRequireNonNull(text)
	for index := 0; index < len(text.units); index++ {
		unit := text.units[index]
		if unit >= 0xd800 && unit <= 0xdbff {
			index++
			if index >= len(text.units) || text.units[index] < 0xdc00 || text.units[index] > 0xdfff {
				return "", false
			}
		} else if unit >= 0xdc00 && unit <= 0xdfff {
			return "", false
		}
	}
	return string(utf16.Decode(text.units)), true
}
