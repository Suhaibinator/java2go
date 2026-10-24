package stdjava

import "reflect"

// callRegisteredSourceJavaString uses declaration metadata to select the Java
// method, independently of ordinary source methods that resemble Go protocols.
// It preserves the returned reference, including a null result. The metadata
// lookup releases its locks before resolving the owner view or invoking Java.
func callRegisteredSourceJavaString(execution *Execution, value any) (*JavaString, bool) {
	method, selector, ok := registeredSourceToStringMethod(value)
	if !ok {
		return nil, false
	}
	if !method.IsValid() || method.Type().NumIn() != 1 || method.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || method.Type().NumOut() != 1 || method.Type().Out(0) != reflect.TypeOf((*JavaString)(nil)) {
		panic("invalid registered Java reference toString execution selector: " + selector)
	}
	return method.Call([]reflect.Value{reflect.ValueOf(execution)})[0].Interface().(*JavaString), true
}
