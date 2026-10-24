package stdjava

// The returned pointer is the actual native storage, even if this private
// method was promoted onto an embedding host. That host cannot certify itself
// merely by inheriting the native add hook or an allocation marker.
func (list *List[T]) arrayListBoundSourceReference() (any, *ObjectInfo) {
	if !list.arrayListNativeStorage() || list.sourceInfo == nil || list.sourceMethods.Add == nil {
		return nil, nil
	}
	return list, list.sourceInfo
}

// arrayListBoundSourceStateView certifies the source and native directions of
// one bound allocation before a collector can invoke its virtual source body.
// Public nominal metadata and structural hook membership are never authority.
func arrayListBoundSourceStateView(sourceRootInfo *ObjectInfo) JavaIterable {
	if sourceRootInfo == nil || !registeredJavaSourceValue(sourceRootInfo) || !JavaTypeAssignable(sourceRootInfo.DynamicType(), "java.util.ArrayList") {
		panic(NewUnsupportedOperationException("source ArrayList has no registered source identity"))
	}
	arrayView := sourceRootInfo.resolveView("java.util.ArrayList")
	bound, ok := arrayView.(interface {
		arrayListBoundSourceReference() (any, *ObjectInfo)
	})
	if !ok || nilJavaReference(arrayView) {
		panic(NewUnsupportedOperationException("source ArrayList has no owned native binding"))
	}
	base, boundInfo := bound.arrayListBoundSourceReference()
	if base == nil || boundInfo != sourceRootInfo || base != arrayView || base != sourceRootInfo.resolveView("java.util.List") {
		panic(NewUnsupportedOperationException("source ArrayList provider does not match its native binding"))
	}
	var root JavaIterable
	for _, requested := range []TypeID{sourceRootInfo.DynamicType(), "java.util.Collection", "java.lang.Iterable"} {
		view := sourceRootInfo.resolveView(requested)
		carrier, ok := view.(JavaObjectInfoCarrier)
		if !ok || nilJavaReference(view) || carrier.JavaObjectInfo() != sourceRootInfo {
			panic(NewUnsupportedOperationException("source ArrayList provider has no matching source identity"))
		}
		iterable, ok := view.(JavaIterable)
		if !ok {
			panic(NewUnsupportedOperationException("source ArrayList has no canonical collection view"))
		}
		if _, ok := view.(interface {
			CollectionAddJava2goExecution(*Execution, any) bool
		}); !ok {
			panic(NewUnsupportedOperationException("source ArrayList has no execution-aware add bridge"))
		}
		if requested == sourceRootInfo.DynamicType() {
			root = iterable
		}
	}
	return root
}
