package stdjava

// Preserve the old exact-result paths, including native primitive callbacks.
// Crossing the erased reference boundary requires an explicit consumer proof.
func CallSupplierProjectedExecution[T any](e *Execution, supplier any, project func(any) T) T {
	switch supplier.(type) {
	case interface{ GetJava2goExecution(*Execution) T }, Supplier[T], func(*Execution) T, func() T:
		return CallSupplierExecution[T](e, supplier)
	}
	if project == nil {
		panic(NewUnsupportedOperationException("Supplier result projection is unavailable"))
	}
	return project(SupplierGetReferenceExecution(e, supplier))
}

// This collector still owns the native Map algorithm. A different valid Java
// Map carrier is explicitly unsupported; it must never be copied or recast.
func SupplierNativeMapProduct[K, V any](value any) *Map[K, V] {
	if nilJavaReference(value) {
		return nil
	}
	reference := MapReferenceView(value)
	if native, ok := reference.(*Map[K, V]); ok {
		return native
	}
	panic(NewUnsupportedOperationException("collector map factory result has no supported native carrier"))
}
