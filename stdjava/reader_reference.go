package stdjava

import (
	"bufio"
	"fmt"
	"sync"
	"unicode/utf16"
)

// referenceReaderSource owns closed state without changing BufferedReader's
// native representation or retaining readers in a global side table.
type referenceReaderSource struct {
	lineMu sync.Mutex
	mu     sync.Mutex
	source ioSource
	closed bool
}

func (source *referenceReaderSource) Read(buffer []byte) (int, error) {
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return 0, fmt.Errorf("Stream closed")
	}
	return source.source.r.Read(buffer)
}

func (source *referenceReaderSource) close() {
	source.lineMu.Lock()
	defer source.lineMu.Unlock()
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.closed {
		return
	}
	// The JDK closes its buffer in finally, even if the source close throws.
	source.closed = true
	source.source.close()
}

// NewBufferedReaderReference preserves the native pointer shape while adding
// canonical construction, sized buffering, and shared source-close ownership.
func NewBufferedReaderReference(source any, sizes ...int32) *BufferedReader {
	if nilJavaReference(source) {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	size := int32(8192)
	if len(sizes) > 0 {
		size = sizes[0]
	}
	if size <= 0 {
		panic(NewJavaIllegalArgumentExceptionMessage(JavaStringFromHostUTF8("Buffer size <= 0")))
	}
	// A FileReader constructor can be unwrapped by the existing compiler into
	// its canonical path String. This conversion occurs only at the host path.
	if path, ok := source.(*JavaString); ok {
		source = string(utf16.Decode(path.units))
	}
	state := &referenceReaderSource{source: ioSourceOf(source)}
	return &BufferedReader{
		src: ioSource{r: state, close: state.close},
		buf: bufio.NewReaderSize(state, int(size)),
	}
}

// ReadLineReference returns null only when no line was present. Every present
// line has its own immutable String object, including an empty line.
func (reader *BufferedReader) ReadLineReference() *JavaString {
	if reader == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	if source, ok := reader.src.r.(*referenceReaderSource); ok {
		source.lineMu.Lock()
		defer source.lineMu.Unlock()
		source.mu.Lock()
		closed := source.closed
		source.mu.Unlock()
		if closed {
			throwIOException(fmt.Errorf("Stream closed"))
		}
	}
	line, present := reader.ReadLineOK()
	if !present {
		return nil
	}
	return JavaStringFromHostUTF8(line)
}
