package stdjava

type Appendable interface{ JavaAppendableMarker() }
type Flushable interface{ JavaFlushableMarker() }

func FlushableFlushExecution(execution *Execution, value Flushable) {
	ReferenceRequireNonNull(value)
	if source, ok := value.(interface{ JavaFlush(*Execution) }); ok {
		source.JavaFlush(execution)
		return
	}
	if source, ok := value.(interface{ FlushJava2goExecution(*Execution) }); ok {
		source.FlushJava2goExecution(execution)
		return
	}
	if builtin, ok := value.(interface{ Flush() }); ok {
		builtin.Flush()
		return
	}
	panic(NewClassCastException("Flushable implementation lacks flush()"))
}
func AppendableAppendExecution(execution *Execution, value Appendable, text any, bounds ...int32) Appendable {
	return appendableAppend(execution, value, text, false, bounds...)
}
func appendableAppend(execution *Execution, value Appendable, text any, baseDefault bool, bounds ...int32) Appendable {
	ReferenceRequireNonNull(value)
	if character, ok := text.(rune); ok && len(bounds) == 0 {
		if source, ok := value.(interface {
			JavaAppendChar(*Execution, rune) Appendable
		}); ok && !baseDefault {
			return source.JavaAppendChar(execution, character)
		}
		if writer, ok := value.(Writer); ok {
			WriterWriteExecution(execution, writer, int32(character))
			return value
		}
		if builder, ok := value.(*StringBuilder); ok {
			builder.Append(character)
			return builder
		}
	}
	if len(bounds) == 0 {
		if source, ok := value.(interface {
			JavaAppendSequence(*Execution, any) Appendable
		}); ok && !baseDefault {
			return source.JavaAppendSequence(execution, text)
		}
	} else if source, ok := value.(interface {
		JavaAppendRange(*Execution, any, int32, int32) Appendable
	}); ok && !baseDefault {
		return source.JavaAppendRange(execution, text, bounds[0], bounds[1])
	}
	if text == nil || javaReferenceIsNull(text) {
		text = "null"
	}
	if writer, ok := value.(Writer); ok {
		if len(bounds) == 2 {
			return WriterAppendExecution(execution, writer, characterSubSequence(execution, text, bounds[0], bounds[1]))
		}
		WriterWriteExecution(execution, writer, StringValueOfExecution(execution, text))
		return value
	}
	start, end := int32(0), CharSequenceLength(execution, text)
	if len(bounds) == 2 {
		start, end = bounds[0], bounds[1]
		checkCharacterRange(CharSequenceLength(execution, text), start, end-start)
	}
	units := make([]rune, end-start)
	for i := range units {
		units[i] = CharSequenceCharAt(execution, text, start+int32(i))
	}
	if writer, ok := value.(Writer); ok {
		WriterWriteCharsExecution(execution, writer, PrimitiveArrayLiteral(PrimitiveTypeID("char"), units...), 0, int32(len(units)))
		return value
	}
	if builder, ok := value.(*StringBuilder); ok {
		for _, unit := range units {
			builder.Append(unit)
		}
		return builder
	}
	panic(NewClassCastException("Appendable implementation lacks append"))
}
func (*StringBuilder) JavaAppendableMarker() {}

func characterSubSequence(execution *Execution, text any, start, end int32) any {
	if source, ok := objectExecutionMethod(execution, text, "SubSequenceJava2goExecution", []any{start, end}); ok {
		return source.Interface()
	}
	checkCharacterRange(CharSequenceLength(execution, text), start, end-start)
	units := make([]rune, end-start)
	for i := range units {
		units[i] = CharSequenceCharAt(execution, text, start+int32(i))
	}
	return StringFromChars(units)
}
func WriterAppendDefaultExecution(execution *Execution, writer Writer, text any, bounds ...int32) Writer {
	return appendableAppend(execution, writer, text, true, bounds...).(Writer)
}
