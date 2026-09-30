package stdjava

// NewJavaAssertionErrorExecution implements the Java-facing constructors with
// canonical detailMessage storage. Object conversion invokes its selected Java
// toString once and retains the returned reference, including null. The legacy
// Go constructors keep their native-string protocol in assertion_error.go.
func NewJavaAssertionErrorExecution(execution *Execution, arguments ...any) AssertionError {
	if execution == nil {
		execution = NewExecution()
	}
	var message *JavaString
	var cause any
	initialized := false
	switch len(arguments) {
	case 0:
	case 1:
		detail := arguments[0]
		// The compiler selects primitive overloads before erasing arguments. Char
		// uses its own entry point because its Go representation is also int32.
		switch value := detail.(type) {
		case bool:
			message = JavaStringValueOfBoolean(value)
		case int:
			message = JavaStringValueOfInt(int32(value))
		case int32:
			message = JavaStringValueOfInt(value)
		case int64:
			message = JavaStringValueOfLong(value)
		case float32:
			message = JavaStringValueOfFloat(value)
		case float64:
			message = JavaStringValueOfDouble(value)
		default:
			message = JavaStringValueOfExecution(execution, detail)
		}
		if !javaReferenceIsNull(detail) {
			if _, throwable := collectionObjectView(detail).(nominalThrowableText); throwable {
				cause = detail
				initialized = true
			}
		}
	case 2:
		message = ObjectView[*JavaString](arguments[0], StringTypeID)
		if !javaReferenceIsNull(arguments[1]) {
			cause = arguments[1]
		}
		initialized = true
	default:
		panic(NewIllegalArgumentException("unsupported AssertionError constructor arity"))
	}
	base := newJavaThrowableBase("AssertionError", message)
	base.state.cause = cause
	base.state.causeInitialized = initialized
	return AssertionError{base}
}

func NewJavaAssertionErrorCharExecution(execution *Execution, value rune) AssertionError {
	return NewJavaAssertionErrorExecution(execution, JavaStringValueOfChar(value))
}
