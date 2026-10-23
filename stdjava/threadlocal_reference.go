package stdjava

// Java withInitial captures its original Supplier reference without invoking
// it or adapting its result to the ThreadLocal consumer's type argument.
type threadLocalReferenceInitializer struct{ supplier any }

func ThreadLocalWithInitialReference(supplier any) *ThreadLocal[any] {
	ReferenceRequireNonNull(supplier)
	local := NewThreadLocal[any]()
	local.referenceKind = threadLocalReferenceSupplied
	local.initial = threadLocalReferenceInitializer{supplier: supplier}
	return local
}

// GetObject publishes the erased value before a typed Get or generated Java
// consumer projects it. A present nil is cached, and an abrupt initializer does
// not publish a final result or undo any explicit reentrant side effects.
func (local *ThreadLocal[T]) GetObject(execution *Execution) any {
	ReferenceRequireNonNull(local)
	requireExecution(execution)
	if stored, present := execution.threadLocals[local.key]; present {
		return stored.(threadLocalValue).value
	}
	var value any = collectionZero[T]()
	if local.initial != nil {
		value = threadLocalInitialReferenceValue[T](execution, local.initial)
	}
	local.SetObject(execution, value)
	return value
}

func (local *ThreadLocal[T]) SetObject(execution *Execution, value any) {
	ReferenceRequireNonNull(local)
	requireExecution(execution)
	if execution.threadLocals == nil {
		execution.threadLocals = make(map[uint64]any)
	}
	execution.threadLocals[local.key] = threadLocalValue{value: value}
}

func threadLocalInitialReferenceValue[T any](execution *Execution, initial any) any {
	if reference, java := initial.(threadLocalReferenceInitializer); java {
		return SupplierGetReferenceExecution(execution, reference.supplier)
	}
	if _, java := initial.(JavaSupplier); java {
		return SupplierGetReferenceExecution(execution, initial)
	}
	// A source Supplier's canonical ancestry is a promise of its owned sealed
	// view. A missing/foreign provider must fail there, never fall back to Get.
	if carrier, source := initial.(JavaObjectInfoCarrier); source {
		if info := carrier.JavaObjectInfo(); info != nil && JavaTypeAssignable(info.DynamicType(), SupplierTypeID) {
			return SupplierGetReferenceExecution(execution, initial)
		}
	}
	// Preserve the public native Go callback API for ordinary funcs/Supplier[T]
	// values that do not present a Java Supplier reference protocol.
	return CallSupplierExecution[T](execution, initial)
}
