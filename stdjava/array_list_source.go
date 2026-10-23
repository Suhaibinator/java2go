package stdjava

// ArrayListSourceMethods binds virtual calls on a native ArrayList projection
// to the most-derived source receiver. Each call supplies its current Execution.
type ArrayListSourceMethods struct {
	Add func(*Execution, any) bool
}

// BindArrayListSourceReference gives an actual native ArrayList allocation the
// identity and virtual add entry of its source owner. The provider must already
// be installed on that owner; all validation precedes mutation of the storage.
func BindArrayListSourceReference[T any](base *List[T], info *ObjectInfo, methods ArrayListSourceMethods) {
	ReferenceRequireNonNull(base)
	if !base.arrayListNativeStorage() || base.sourceInfo != nil || base.sourceMethods.Add != nil || info == nil || methods.Add == nil || !registeredJavaSourceValue(info) || !JavaTypeAssignable(info.DynamicType(), "java.util.ArrayList") {
		panic(NewUnsupportedOperationException("ArrayList source storage has no complete unowned protocol"))
	}
	if info.resolveView("java.util.ArrayList") != base || info.resolveView("java.util.List") != base {
		panic(NewUnsupportedOperationException("ArrayList source provider does not own this storage"))
	}
	// A nominal type id alone cannot bind storage. The whole source receiver
	// and both iterable aliases must expose this exact allocation identity and
	// its source add entry, rather than an unrelated provider or native view.
	for _, requested := range []TypeID{info.DynamicType(), "java.util.Collection", "java.lang.Iterable"} {
		view := info.resolveView(requested)
		carrier, ok := view.(JavaObjectInfoCarrier)
		if !ok || nilJavaReference(view) || carrier.JavaObjectInfo() != info {
			panic(NewUnsupportedOperationException("ArrayList source provider has no matching source identity"))
		}
		if _, ok := view.(JavaIterable); !ok {
			panic(NewUnsupportedOperationException("ArrayList source provider has no canonical iterable view"))
		}
		if _, ok := view.(interface {
			CollectionAddJava2goExecution(*Execution, any) bool
		}); !ok {
			panic(NewUnsupportedOperationException("ArrayList source provider has no execution-aware add entry"))
		}
	}
	base.sourceInfo = info
	base.sourceMethods = methods
}

// JavaObjectInfo is nil for unbound native allocations and collection views.
// Existing native metadata, identity and reflection paths retain their fallback.
func (list *List[T]) JavaObjectInfo() *ObjectInfo {
	if list == nil {
		return nil
	}
	return list.sourceInfo
}

func (list *List[T]) CollectionAddJava2goExecution(execution *Execution, value any) bool {
	ReferenceRequireNonNull(list)
	if list.sourceMethods.Add != nil {
		requireExecution(execution)
		return list.sourceMethods.Add(execution, value)
	}
	// Preserve the native erased-write algorithm, including fixed/view checks
	// and legacy callers that do not carry a Java execution.
	return list.collectionAdd(value, execution)
}
