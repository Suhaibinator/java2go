package stdjava

type bufferedStringReaderReferenceState struct {
	// Reader.lock survives close independently of the readable source and buffer.
	lock, source   *StringReader
	units          []uint16
	next, valid    int32
	skipLF, closed bool
}

func newBufferedStringReaderReference(source *StringReader, size int32) *BufferedReader {
	return &BufferedReader{reference: &bufferedStringReaderReferenceState{lock: source, source: source, units: make([]uint16, int(size))}}
}
func bufferedStringReaderFill(e *Execution, state *bufferedStringReaderReferenceState) {
	buffer := NewPrimitiveArray[rune](len(state.units), PrimitiveTypeID("char"))
	n := StringReaderReadCharsExecution(e, state.source, buffer, 0, int32(len(buffer.Elements)))
	if n > 0 {
		for i := int32(0); i < n; i++ {
			state.units[i] = uint16(buffer.Elements[i])
		}
		state.valid = n
		state.next = 0
	}
}

func BufferedStringReaderReadLineExecution(e *Execution, reader *BufferedReader) *JavaString {
	if reader == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	state := reader.reference
	if state == nil {
		return reader.ReadLineReference()
	}
	guard := MonitorEnterExecution(e, state.lock)
	defer MonitorExitExecution(guard)
	if state.closed {
		stringReaderClosed()
	}
	var line []uint16
	omitLF := state.skipLF
	for {
		if state.next >= state.valid {
			bufferedStringReaderFill(e, state)
		}
		if state.next >= state.valid {
			if len(line) > 0 {
				return NewJavaStringUTF16(line)
			}
			return nil
		}
		if omitLF && state.units[state.next] == '\n' {
			state.next++
		}
		state.skipLF = false
		omitLF = false
		for state.next < state.valid {
			ch := state.units[state.next]
			state.next++
			if ch == '\n' || ch == '\r' {
				if ch == '\r' {
					state.skipLF = true
				}
				return NewJavaStringUTF16(line)
			}
			line = append(line, ch)
		}
	}
}
func BufferedStringReaderCloseExecution(e *Execution, reader *BufferedReader) {
	if reader == nil {
		panic(NullPointerException{newJavaThrowableBase("NullPointerException", nil)})
	}
	state := reader.reference
	if state == nil {
		reader.Close()
		return
	}
	guard := MonitorEnterExecution(e, state.lock)
	defer MonitorExitExecution(guard)
	if state.closed {
		return
	}
	defer func() { state.closed = true; state.source = nil; state.units = nil }()
	StringReaderCloseExecution(e, state.source)
}
