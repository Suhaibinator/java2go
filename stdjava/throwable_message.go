package stdjava

import (
	"reflect"
	"sync"
)

// Entries are compiler-resolved Java overrides, keyed by concrete Go type.
// Overloads and renamed execution companions are never selected by spelling.
var throwableMessageOverrides sync.Map

func RegisterThrowableMessage(prototype any, invoke func(*Execution, any) string) {
	throwableMessageOverrides.Store(reflect.TypeOf(prototype), invoke)
}

func ThrowableMessageExecution(execution *Execution, receiver any) string {
	ReferenceRequireNonNull(receiver)
	receiver = collectionObjectView(receiver)
	if invoke, ok := throwableMessageOverrides.Load(reflect.TypeOf(receiver)); ok {
		return invoke.(func(*Execution, any) string)(execution, receiver)
	}
	return ThrowableMessageDefault(receiver)
}

// An explicit super.getMessage invokes the inherited builtin body without
// redispatching to an override on the most-derived receiver.
func ThrowableMessageDefault(receiver any) string {
	ReferenceRequireNonNull(receiver)
	return receiver.(Throwable).Message()
}
