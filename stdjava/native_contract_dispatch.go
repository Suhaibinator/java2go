package stdjava

import (
	"fmt"
	"reflect"
	"sync"
)

// Source contracts retain only declaration metadata. Generic Go instantiations,
// object views and executions are bound afresh for each invocation.
type sourceErasedContractKey struct {
	owner, contract TypeID
	descriptor      string
}
type sourceErasedContract struct {
	owner, contract      TypeID
	descriptor, selector string
}

var sourceErasedContracts = struct {
	sync.RWMutex
	entries map[sourceErasedContractKey]sourceErasedContract
}{entries: make(map[sourceErasedContractKey]sourceErasedContract)}

type sourceErasedContractInvariantError string

func (e sourceErasedContractInvariantError) Error() string { return string(e) }

// RegisterSourceErasedContract publishes the exact compiler-selected erased
// entry. Registration does not resolve an object or execute Java initialization.
func RegisterSourceErasedContract(owner, contract TypeID, descriptor, selector string) {
	if owner == "" || contract == "" || descriptor == "" || selector == "" {
		panic(sourceErasedContractInvariantError("empty source erased contract metadata"))
	}
	key := sourceErasedContractKey{owner, contract, descriptor}
	entry := sourceErasedContract{owner, contract, descriptor, selector}
	sourceErasedContracts.Lock()
	prior, exists := sourceErasedContracts.entries[key]
	if !exists {
		sourceErasedContracts.entries[key] = entry
	}
	sourceErasedContracts.Unlock()
	if exists && prior != entry {
		panic(sourceErasedContractInvariantError("conflicting source erased contract registration"))
	}
}

// Only concrete superclass edges select implementation ownership. Interfaces
// establish nominal admission but never choose an entry by traversal order.
func registeredSourceErasedContract(actual, contract TypeID, descriptor string) (sourceErasedContract, bool) {
	seen := make(map[TypeID]bool)
	for current := actual; current != "" && !seen[current]; {
		seen[current] = true
		sourceErasedContracts.RLock()
		entry, found := sourceErasedContracts.entries[sourceErasedContractKey{current, contract, descriptor}]
		sourceErasedContracts.RUnlock()
		if found {
			return entry, true
		}
		javaTypeRegistry.RLock()
		parent := javaTypeRegistry.types[current].super
		javaTypeRegistry.RUnlock()
		current = parent
	}
	return sourceErasedContract{}, false
}

func callRegisteredSourceErasedContractExecution(execution *Execution, left, right any, contract TypeID, descriptor string) (int32, bool) {
	actual, known := ObjectDynamicType(left)
	if !known {
		return 0, false
	}
	entry, found := registeredSourceErasedContract(actual, contract, descriptor)
	if !found {
		return 0, false
	}
	if !JavaTypeAssignable(actual, contract) {
		panic(NewClassCastException("registered source does not implement erased contract"))
	}
	receiver := left
	var info *ObjectInfo
	if carrier, ok := left.(JavaObjectInfoCarrier); ok {
		info = carrier.JavaObjectInfo()
	}
	if info != nil {
		// Even equal dynamic IDs can arrive through an embedded base alias.
		receiver = info.resolveView(entry.owner)
	} else if actual != entry.owner {
		panic(sourceErasedContractInvariantError("source erased contract ancestor has no object view provider"))
	}
	if nilJavaReference(receiver) {
		panic(sourceErasedContractInvariantError("source erased contract owner view is absent"))
	}
	method := reflect.ValueOf(receiver).MethodByName(entry.selector)
	if !method.IsValid() {
		panic(sourceErasedContractInvariantError(fmt.Sprintf("source erased contract entry %s is absent", entry.selector)))
	}
	signature := method.Type()
	if signature.IsVariadic() || signature.NumIn() != 2 || signature.NumOut() != 1 || signature.In(0) != reflect.TypeFor[*Execution]() || signature.In(1) != reflect.TypeFor[any]() || signature.Out(0) != reflect.TypeFor[int32]() {
		panic(sourceErasedContractInvariantError("source erased contract entry has invalid ABI"))
	}
	argument := reflect.ValueOf(right)
	if right == nil {
		argument = reflect.Zero(reflect.TypeFor[any]())
	}
	if execution == nil {
		execution = NewExecution()
	}
	return int32(method.Call([]reflect.Value{reflect.ValueOf(execution), argument})[0].Int()), true
}
