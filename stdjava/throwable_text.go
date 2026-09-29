package stdjava

import (
	"reflect"
	"sync"
)

// These registries contain compiler-resolved methods, not object instances.
// Load completes before invoking Java code, which may reenter these protocols.
var throwableToStringOverrides sync.Map
var throwableLocalizedMessageOverrides sync.Map

func RegisterThrowableToStringOverride(prototype any, invoke func(*Execution, any) string) {
	throwableToStringOverrides.Store(reflect.TypeOf(prototype), invoke)
}

func RegisterThrowableLocalizedMessageOverride(prototype any, invoke func(*Execution, any) string) {
	throwableLocalizedMessageOverrides.Store(reflect.TypeOf(prototype), invoke)
}

// ThrowableToStringExecution performs virtual Java dispatch while leaving the
// native Error and String methods available for Go diagnostics.
func ThrowableToStringExecution(execution *Execution, receiver any) string {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := throwableToStringOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) string)(execution, receiver)
	}
	// Existing generated execution companions and runtime implementations also
	// expose Java overrides through this protocol. Preserve their exact result,
	// including a null String, without falling back to native Error formatting.
	if stringer, ok := receiver.(executionStringer); ok {
		return stringer.StringJava2goExecution(execution)
	}
	if rendered, ok := callCollisionSafeExecutionStringer(execution, receiver); ok {
		return rendered
	}
	return ThrowableToStringDefaultExecution(execution, receiver)
}

// ThrowableToStringDefaultExecution is the inherited Throwable body used by
// super.toString: only toString dispatch is skipped; localized message remains
// virtual on the original most-derived receiver.
func ThrowableToStringDefaultExecution(execution *Execution, receiver any) string {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	name := ObjectGetClass(receiver).GetName()
	message := ThrowableLocalizedMessageExecution(execution, receiver)
	if StringIsNull(message) {
		return name
	}
	return name + ": " + message
}

func ThrowableLocalizedMessageExecution(execution *Execution, receiver any) string {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := throwableLocalizedMessageOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) string)(execution, receiver)
	}
	return ThrowableLocalizedMessageDefaultExecution(execution, receiver)
}

// Throwable's default localized message is a virtual call to getMessage.
func ThrowableLocalizedMessageDefaultExecution(execution *Execution, receiver any) string {
	return ThrowableMessageExecution(execution, receiver)
}
