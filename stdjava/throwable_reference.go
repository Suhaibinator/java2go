package stdjava

import (
	"reflect"
	"sync"
	"unicode/utf16"
)

// A canonical detailMessage is immutable and belongs to the same allocation
// state as causes/suppression. Its nil value is Java null, not an absent cache.
func newJavaThrowableBase(name string, message *JavaString) ThrowableBase {
	state := &throwableState{javaMessage: message, canonicalMessage: true}
	state.messageOnce.Do(func() {})
	return ThrowableBase{typeName: name, state: state}
}

func (t ThrowableBase) javaThrowableMessage() *JavaString {
	if t.state == nil {
		return nil
	}
	t.state.messageOnce.Do(func() {
		// Legacy native constructors are a one-way runtime diagnostic ingress. The
		// original native bytes remain available through Message/Error unchanged.
		if !StringIsNull(t.message) {
			decoded := decodeCharset([]byte(t.message), UTF_8)
			t.state.javaMessage = NewJavaStringUTF16(utf16.Encode([]rune(decoded)))
		}
	})
	return t.state.javaMessage
}

func NewJavaUnsupportedEncodingException(message *JavaString) UnsupportedEncodingException {
	return UnsupportedEncodingException{newJavaThrowableBase("UnsupportedEncodingException", message)}
}

func NewJavaRuntimeExceptionMessage(message *JavaString) RuntimeException {
	return RuntimeException{newJavaThrowableBase("RuntimeException", message)}
}

func newJavaThrowableCause(execution *Execution, name string, cause any) ThrowableBase {
	if execution == nil {
		execution = NewExecution()
	}
	var message *JavaString
	if javaReferenceIsNull(cause) {
		cause = nil
	} else {
		message = JavaThrowableToStringExecution(execution, cause)
	}
	base := newJavaThrowableBase(name, message)
	base.state.cause = cause
	base.state.causeInitialized = true
	return base
}

func NewJavaExceptionCauseExecution(execution *Execution, cause any) Exception {
	return Exception{newJavaThrowableCause(execution, "Exception", cause)}
}

func NewJavaRuntimeExceptionCauseExecution(execution *Execution, cause any) RuntimeException {
	return RuntimeException{newJavaThrowableCause(execution, "RuntimeException", cause)}
}

func NewJavaExceptionMessageCause(message *JavaString, cause any) Exception {
	base := newJavaThrowableBase("Exception", message)
	if !javaReferenceIsNull(cause) {
		base.state.cause = cause
	}
	base.state.causeInitialized = true
	return Exception{base}
}

// Canonical registries are separate from legacy native-string callbacks. Lookup
// completes before Java code runs, preserving callback reentrancy and execution.
var javaThrowableMessageOverrides sync.Map
var javaThrowableLocalizedMessageOverrides sync.Map
var javaThrowableToStringOverrides sync.Map

func RegisterJavaThrowableMessage(prototype any, invoke func(*Execution, any) *JavaString) {
	javaThrowableMessageOverrides.Store(reflect.TypeOf(prototype), invoke)
}
func RegisterJavaThrowableLocalizedMessageOverride(prototype any, invoke func(*Execution, any) *JavaString) {
	javaThrowableLocalizedMessageOverrides.Store(reflect.TypeOf(prototype), invoke)
}
func RegisterJavaThrowableToStringOverride(prototype any, invoke func(*Execution, any) *JavaString) {
	javaThrowableToStringOverrides.Store(reflect.TypeOf(prototype), invoke)
}

func JavaThrowableMessageExecution(execution *Execution, receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := javaThrowableMessageOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) *JavaString)(execution, receiver)
	}
	return JavaThrowableMessageDefault(receiver)
}
func JavaThrowableMessageDefault(receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	return receiver.(interface{ javaThrowableMessage() *JavaString }).javaThrowableMessage()
}
func JavaThrowableLocalizedMessageExecution(execution *Execution, receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := javaThrowableLocalizedMessageOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) *JavaString)(execution, receiver)
	}
	return JavaThrowableLocalizedMessageDefaultExecution(execution, receiver)
}
func JavaThrowableLocalizedMessageDefaultExecution(execution *Execution, receiver any) *JavaString {
	return JavaThrowableMessageExecution(execution, receiver)
}
func JavaThrowableToStringExecution(execution *Execution, receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := javaThrowableToStringOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) *JavaString)(execution, receiver)
	}
	if registeredJavaSourceValue(receiver) {
		if text, found := callRegisteredSourceJavaString(execution, receiver); found {
			return text
		}
	} else if source, ok := receiver.(javaReferenceStringer); ok {
		return source.StringJava2goExecution(execution)
	}
	return JavaThrowableToStringDefaultExecution(execution, receiver)
}
func JavaThrowableToStringDefaultExecution(execution *Execution, receiver any) *JavaString {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	// Qualified class names originate in runtime type metadata, not application
	// String content. Detail messages are appended as their original UTF16 units.
	name := JavaStringLiteralUTF16(utf16.Encode([]rune(ObjectGetClass(receiver).GetName())))
	message := JavaThrowableLocalizedMessageExecution(execution, receiver)
	if message == nil {
		return name
	}
	units := append(name.UTF16Copy(), ':', ' ')
	units = append(units, message.units...)
	return NewJavaStringUTF16(units)
}

// NewJavaNumberFormatException retains the parser's UTF16 diagnostic directly.
func NewJavaNumberFormatException(message *JavaString) NumberFormatException {
	return NumberFormatException{newJavaThrowableBase("NumberFormatException", message)}
}
