package transpiler

// physicalSourceLine reports the row in the unchanged input bytes. Java Unicode
// escape spellings do not create physical source lines. A CRLF advances only
// after both bytes have been consumed; offsets outside the input are unknown.
func physicalSourceLine(source []byte, byteOffset uint32) uint32 {
	if uint64(byteOffset) > uint64(len(source)) {
		return 0
	}
	line := uint32(1)
	for index := uint32(0); index < byteOffset; index++ {
		switch source[index] {
		case '\r':
			if uint64(index)+1 < uint64(len(source)) && source[index+1] == '\n' {
				if index+1 >= byteOffset {
					continue
				}
				index++
			}
			line++
		case '\n':
			line++
		}
	}
	return line
}
