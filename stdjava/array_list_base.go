package stdjava

// This private protocol requires real stdjava List storage. Public type-id or
// add-shaped values cannot supply it. The allocation check also excludes List,
// LinkedList, array-backed fixed lists and subList views.
type arrayListNativeBase interface {
	arrayListNativeStorage() bool
	collectionAdd(any, *Execution) bool
	rawGet(int32) any
	arrayListNativeSize() int32
}

func (list *List[T]) arrayListNativeStorage() bool {
	return list != nil && list.allocationKind == listAllocationArray && !list.fixed && list.array == nil && list.viewRoot == nil && list.viewParent == nil
}

// Keep the concrete List receiver when the private operation is promoted.
func (list *List[T]) arrayListNativeSize() int32 {
	return list.Size()
}

func requireArrayListNativeBase(base any) arrayListNativeBase {
	ReferenceRequireNonNull(base)
	storage, ok := base.(arrayListNativeBase)
	if !ok || !storage.arrayListNativeStorage() {
		panic(NewUnsupportedOperationException("ArrayList native storage is unavailable"))
	}
	return storage
}

// ArrayListBaseAddExecution implements Java super.add without consulting the
// bound virtual entry, so a source override can call it without recursion.
func ArrayListBaseAddExecution(execution *Execution, base any, value any) bool {
	requireExecution(execution)
	return requireArrayListNativeBase(base).collectionAdd(value, execution)
}

func ArrayListBaseGetExecution(execution *Execution, base any, index int32) any {
	requireExecution(execution)
	return requireArrayListNativeBase(base).rawGet(index)
}

func ArrayListBaseSizeExecution(execution *Execution, base any) int32 {
	requireExecution(execution)
	return requireArrayListNativeBase(base).arrayListNativeSize()
}
