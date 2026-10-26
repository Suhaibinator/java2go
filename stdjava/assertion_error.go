package stdjava

import "reflect"

// NewAssertionErrorExecution implements Java's public constructor overloads.
// Primitive char uses the separate entry point so its numeric Go representation
// is never confused with the Java int overload.
func NewAssertionErrorExecution(execution *Execution, arguments ...any) AssertionError {
	if execution == nil {
		execution = NewExecution()
	}
	message := NullString()
	var cause any
	initialized := false
	switch len(arguments) {
	case 0:
	case 1:
		detail := arguments[0]
		message = StringValueOfExecution(execution, detail)
		if throwable, ok := detail.(Throwable); ok && !javaReferenceIsNull(throwable) {
			// Runtime Throwable.String retains legacy Go formatting; Java constructors
			// require the JDK qualified toString form. Source overrides use the caller's
			// execution-aware StringValueOf dispatch above.
			reflected := reflect.TypeOf(detail)
			if reflected.Kind() == reflect.Pointer {
				reflected = reflected.Elem()
			}
			if reflected.PkgPath() == "github.com/NickyBoy89/java2go/stdjava" {
				message = exceptionCauseMessage(detail)
			}
			cause = detail
			initialized = true
		}
	case 2:
		message = StringReferenceValue(arguments[0])
		if !javaReferenceIsNull(arguments[1]) {
			cause = arguments[1]
		}
		initialized = true
	default:
		panic(NewIllegalArgumentException("unsupported AssertionError constructor arity"))
	}
	base := newThrowableBase("AssertionError", message)
	base.state.cause = cause
	base.state.causeInitialized = initialized
	return AssertionError{base}
}
func NewAssertionErrorCharExecution(execution *Execution, value rune) AssertionError {
	return NewAssertionErrorExecution(execution, StringFromChars([]rune{value}))
}
