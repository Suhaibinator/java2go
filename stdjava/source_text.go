package stdjava

import (
	"reflect"
	"sync"
)

// Source text dispatch is declaration metadata, not a structural Go protocol.
// Entries contain no instances and are installed with the Java type metadata,
// before Java constructors or static initializers can invoke virtual methods.
var sourceToStringSelectors sync.Map

func RegisterJavaSourceToString(id TypeID, selector string) {
	sourceToStringSelectors.Store(id, selector)
}

func sourceToStringSelector(actual TypeID) (TypeID, string, bool) {
	javaTypeRegistry.RLock()
	defer javaTypeRegistry.RUnlock()
	seen := make(map[TypeID]bool)
	for current := actual; current != "" && !seen[current]; current = javaTypeRegistry.types[current].super {
		seen[current] = true
		if selector, ok := sourceToStringSelectors.Load(current); ok {
			return current, selector.(string), true
		}
	}
	return "", "", false
}

func callRegisteredSourceToString(execution *Execution, value any) (string, bool) {
	actual, ok := ObjectDynamicType(value)
	if !ok {
		return "", false
	}
	owner, selector, ok := sourceToStringSelector(actual)
	if !ok {
		return "", false
	}
	// Use the declaring class's subobject when inherited. The original
	// most-derived self remains installed there for calls from the Java body.
	if owner != actual {
		if carrier, ok := value.(JavaObjectInfoCarrier); ok && carrier.JavaObjectInfo() != nil {
			if view := carrier.JavaObjectInfo().resolveView(owner); !javaReferenceIsNull(view) {
				value = view
			}
		}
	}
	method := reflect.ValueOf(value).MethodByName(selector)
	if !method.IsValid() || method.Type().NumIn() != 1 || method.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || method.Type().NumOut() != 1 || method.Type().Out(0) != reflect.TypeOf("") {
		panic("invalid registered Java toString execution selector: " + selector)
	}
	return method.Call([]reflect.Value{reflect.ValueOf(execution)})[0].String(), true
}
