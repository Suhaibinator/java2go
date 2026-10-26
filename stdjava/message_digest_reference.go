package stdjava

// JavaMessageDigestGetInstance preserves the original Java algorithm reference.
// Algorithm lookup uses the existing standard-library digest implementation.
func JavaMessageDigestGetInstance(algorithm *JavaString) (result *MessageDigest) {
	if algorithm == nil {
		panic(NewNullPointerException("null algorithm name"))
	}
	missing := func() NoSuchAlgorithmException {
		units := append(algorithm.UTF16Copy(), []uint16{' ', 'M', 'e', 's', 's', 'a', 'g', 'e', 'D', 'i', 'g', 'e', 's', 't', ' ', 'n', 'o', 't', ' ', 'a', 'v', 'a', 'i', 'l', 'a', 'b', 'l', 'e'}...)
		return NoSuchAlgorithmException{newJavaThrowableBase("NoSuchAlgorithmException", NewJavaStringUTF16(units))}
	}
	name := make([]byte, len(algorithm.units))
	for index, unit := range algorithm.units {
		if unit > 0x7f {
			panic(missing())
		}
		name[index] = byte(unit)
	}
	defer func() {
		if failure := recover(); failure != nil {
			if _, unavailable := failure.(NoSuchAlgorithmException); unavailable {
				panic(missing())
			}
			panic(failure)
		}
	}()
	result = MessageDigestGetInstance(string(name))
	result.algorithmReference = algorithm
	return result
}

func (digest *MessageDigest) GetAlgorithmReference() *JavaString {
	ReferenceRequireNonNull(digest)
	if digest.algorithmReference != nil {
		return digest.algorithmReference
	}
	// Native callers have no Java allocation to preserve. Supported names are ASCII.
	units := make([]uint16, len(digest.algorithm))
	for index := range units {
		units[index] = uint16(digest.algorithm[index])
	}
	return JavaStringLiteralUTF16(units)
}

// Preserve the JDK ByteBuffer overload's null detailMessage. Other overloads
// retain the existing native implementation, including failed-range state.
func (digest *MessageDigest) UpdateReference(input any, bounds ...int32) {
	if buffer, byteBuffer := input.(*ByteBuffer); byteBuffer && buffer == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	digest.Update(input, bounds...)
}
