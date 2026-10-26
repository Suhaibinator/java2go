package stdjava

import "reflect"

// A promoted marker only proves something about its embedded native object.
// Returning the actual pointer makes the caller prove the whole receiver too.
func (list *List[T]) collectorToCollectionNativeReference() any { return list }
func (set *Set[T]) collectorToCollectionNativeReference() any   { return set }

func collectorToCollectionExactProduct(left, right any) bool {
	actual := reflect.TypeOf(left)
	return actual != nil && actual == reflect.TypeOf(right) && actual.Comparable() && left == right
}

// ObjectInfo identifies the allocation but can be promoted onto an outer Go
// host too. The supplied reference itself must be one of this allocation's
// actual providers, including a source superclass or an exact native alias.
func collectorToCollectionSourceProduct(value any, info *ObjectInfo) JavaIterable {
	whole := arrayListBoundSourceStateView(info)
	var types []TypeID
	seen := map[TypeID]bool{}
	queue := []TypeID{info.DynamicType()}
	javaTypeRegistry.RLock()
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current == "" || seen[current] {
			continue
		}
		seen[current] = true
		types = append(types, current)
		definition := javaTypeRegistry.types[current]
		queue = append(queue, definition.super)
		queue = append(queue, definition.interfaces...)
	}
	javaTypeRegistry.RUnlock()
	// Provider callbacks run outside the registry lock. They may themselves
	// consult the nominal graph while selecting a generated source view.
	for _, requested := range types {
		if collectorToCollectionExactProduct(value, info.resolveView(requested)) {
			return whole
		}
	}
	panic(NewUnsupportedOperationException("supplied source Collection is not an owned allocation view"))
}

// Collection is the accumulator's actual JDK erasure. A source ArrayList must
// donate its whole bound receiver before any promoted native method is used.
func collectorToCollectionStateView(value any) JavaIterable {
	if nilJavaReference(value) {
		return nil
	}
	actual, _ := ObjectDynamicType(value)
	javaTypeRegistry.RLock()
	source := javaTypeRegistry.types[actual].source
	javaTypeRegistry.RUnlock()
	if source {
		if !JavaTypeAssignable(actual, nativeCollectionTypeID) {
			panic(referenceCastException(actual, nativeCollectionTypeID))
		}
		carrier, present := value.(JavaObjectInfoCarrier)
		if !present {
			panic(NewUnsupportedOperationException("supplied source Collection has no matching allocation identity"))
		}
		info := carrier.JavaObjectInfo()
		if info == nil || info.DynamicType() != actual {
			panic(NewUnsupportedOperationException("supplied source Collection has no matching allocation identity"))
		}
		return collectorToCollectionSourceProduct(value, info)
	}
	if carrier, present := value.(JavaObjectInfoCarrier); present && carrier.JavaObjectInfo() != nil {
		panic(NewUnsupportedOperationException("supplied Collection has an unregistered source identity"))
	}
	native, attested := value.(interface{ collectorToCollectionNativeReference() any })
	if attested && collectorToCollectionExactProduct(native.collectorToCollectionNativeReference(), value) && nativeJavaInterfaceAssignable(value, nativeCollectionTypeID) {
		if collection, present := value.(JavaIterable); present {
			return collection
		}
	}
	panic(referenceCastException(actual, nativeCollectionTypeID))
}

// CollectorToCollectionPlan is an internal sequential recipe. It preserves the
// supplier's exact Java product through identity finish, including an empty
// result whose checkcast belongs to its eventual Java consumer. Neither recipe
// construction nor supplier return invents a concrete C cast or native facade.
func CollectorToCollectionPlan[T any](factory JavaSupplier) CollectorPlan[T, any] {
	return newCollectorPlan(
		func(e *Execution) any { return SupplierGetReferenceExecution(e, factory) },
		func(e *Execution, state any, item T) {
			requireExecution(e)
			CollectionAddExecution(e, collectorToCollectionStateView(state), item)
		},
		func(*Execution, any, any) any {
			// Collection.addAll is not repeated virtual add: ArrayList bypasses
			// add overrides, and empty/self-alias behavior also differs. No
			// parallel/combiner protocol is certified by this sequential engine.
			panic(NewUnsupportedOperationException("supplied Collection combiner is unavailable"))
		},
		func(_ *Execution, state any) any { return state },
		func(state any) any { return state },
		true,
	)
}
