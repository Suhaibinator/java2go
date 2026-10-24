package stdjava

import (
	"os"
	"sync"
	"unicode/utf16"
)

// JavaStringFromHostUTF8 imports an actual host process argument. The supported
// host boundary is UTF-8; malformed input follows the existing Java UTF-8
// decoder's replacement policy. This is not a bridge for Java-to-Java values.
func JavaStringFromHostUTF8(value string) *JavaString {
	decoded := decodeCharset([]byte(value), UTF_8)
	return NewJavaStringUTF16(utf16.Encode([]rune(decoded)))
}

// The UTF-8 System.out encoder retains a trailing high surrogate between print
// calls, just as PrintStream's StreamEncoder does. Java String payloads never
// undergo replacement; replacement happens only when writing the output bytes.
var javaStringStdout struct {
	sync.Mutex
	pendingHigh uint16
}

// JavaPrintStrings writes canonical Strings to the UTF-8 standard output
// boundary. Internal variadic pieces are concatenated without separators.
func JavaPrintStrings(values ...*JavaString) {
	javaWriteStrings(false, values)
}

// JavaPrintlnStrings writes canonical String pieces followed by one newline.
// An absent argument emits only the newline; a nil String emits "null".
func JavaPrintlnStrings(values ...*JavaString) {
	javaWriteStrings(true, values)
}

func javaWriteStrings(newline bool, values []*JavaString) {
	javaStringStdout.Lock()
	defer javaStringStdout.Unlock()

	var units []uint16
	if javaStringStdout.pendingHigh != 0 {
		units = append(units, javaStringStdout.pendingHigh)
	}
	for _, value := range values {
		if value == nil {
			units = append(units, 'n', 'u', 'l', 'l')
		} else {
			units = append(units, value.units...)
		}
	}
	if newline {
		units = append(units, '\n')
	}

	javaStringStdout.pendingHigh = 0
	if len(units) != 0 {
		last := units[len(units)-1]
		if last >= 0xd800 && last <= 0xdbff {
			javaStringStdout.pendingHigh = last
			units = units[:len(units)-1]
		}
	}
	if len(units) != 0 {
		encoded := JavaStringGetBytes(&JavaString{units: units}, UTF_8)
		_, _ = os.Stdout.Write(unsignedBytes(encoded.Elements))
	}
}
