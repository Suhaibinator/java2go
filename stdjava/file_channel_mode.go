package stdjava

import "os"

// Only already-validated open flags reach the immutable state constructor.
func randomAccessFileWritableFromOpenFlags(flags int) bool {
	return flags&(os.O_WRONLY|os.O_RDWR) != 0
}
