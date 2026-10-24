package stdjava

// CharsetJavaName exposes the canonical Java String without changing the native
// Name/String APIs. The name belongs to its Charset; no global map retains it.
// Custom Java Charset constructors and their supplied name references are not
// modeled. They must acquire an explicit canonical constructor before using this
// boundary, rather than accidentally interning an arbitrary native name.
func CharsetJavaName(charset *Charset) *JavaString {
	ReferenceRequireNonNull(charset)
	switch charset {
	case US_ASCII, ISO_8859_1, UTF_8, UTF_16BE, UTF_16LE, UTF_16:
	default:
		panic(NewUnsupportedOperationException("custom Charset canonical name reference is unavailable"))
	}
	charset.javaNameOnce.Do(func() {
		units := make([]uint16, len(charset.name))
		for i := range charset.name {
			units[i] = uint16(charset.name[i])
		}
		charset.javaName = JavaStringLiteralUTF16(units)
	})
	return charset.javaName
}

func (charset *Charset) StringJava2goExecution(*Execution) *JavaString {
	return CharsetJavaName(charset)
}
