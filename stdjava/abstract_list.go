package stdjava

import "strconv"

const AbstractListTypeID TypeID = "java.util.AbstractList"

// AbstractListState is the protected state of one AbstractList allocation.
// It never borrows a delegate's structural generation.
type AbstractListState struct {
	ModCount int32
	storage  any
	owner    *ObjectInfo
}

func NewAbstractListProjection[T any]() *List[T] {
	base := &List[T]{allocationKind: listAllocationAbstract}
	base.abstractState = &AbstractListState{storage: base}
	return base
}
func (base *List[T]) AbstractListStateJava2go() *AbstractListState {
	ReferenceRequireNonNull(base)
	if base.abstractState == nil {
		panic(NewUnsupportedOperationException("reference has no AbstractList state"))
	}
	return base.abstractState
}
func BindAbstractListSourceReference[T any](base *List[T], info *ObjectInfo) {
	ReferenceRequireNonNull(base)
	state := base.abstractState
	if base.allocationKind != listAllocationAbstract || state == nil || state.storage != base || state.owner != nil || base.sourceInfo != nil || base.abstractView != nil || info == nil || !registeredJavaSourceValue(info) || !JavaTypeAssignable(info.DynamicType(), "java.util.AbstractList") {
		panic(NewUnsupportedOperationException("AbstractList projection has no complete unowned protocol"))
	}
	if info.resolveView("java.util.AbstractList") != base || info.resolveView("java.util.List") != base {
		panic(NewUnsupportedOperationException("AbstractList provider does not own this projection"))
	}
	for _, id := range []TypeID{info.DynamicType(), CollectionTypeID, IterableTypeID} {
		value := info.resolveView(id)
		carrier, ok := value.(JavaObjectInfoCarrier)
		if !ok || nilJavaReference(value) || carrier.JavaObjectInfo() != info {
			panic(NewUnsupportedOperationException("AbstractList provider has no matching source identity"))
		}
		if _, ok := value.(JavaIterable); !ok {
			panic(NewUnsupportedOperationException("AbstractList provider has no iterable role"))
		}
		methods, ok := value.(interface {
			AbstractListStateJava2go() *AbstractListState
			CollectionListGetJava2goExecution(*Execution, int32) any
			CollectionSizeJava2goExecution(*Execution) int32
			CollectionListSetJava2goExecution(*Execution, int32, any) any
			CollectionListAddAtJava2goExecution(*Execution, int32, any)
			CollectionListRemoveAtJava2goExecution(*Execution, int32) any
		})
		if !ok || methods.AbstractListStateJava2go() != state {
			panic(NewUnsupportedOperationException("AbstractList provider has no complete virtual roles"))
		}
	}
	base.sourceInfo = info
	state.owner = info
}
func (base *List[T]) abstractListBoundSourceReference() (any, *ObjectInfo, *AbstractListState) {
	if base == nil || base.abstractState == nil || base.abstractState.storage != base || base.abstractState.owner == nil || base.abstractState.owner != base.sourceInfo {
		return nil, nil, nil
	}
	return base, base.sourceInfo, base.abstractState
}
func abstractListCollectionView(collection JavaIterable) JavaIterable {
	ReferenceRequireNonNull(collection)
	if native, ok := collection.(interface {
		abstractListBoundSourceReference() (any, *ObjectInfo, *AbstractListState)
	}); ok {
		base, info, state := native.abstractListBoundSourceReference()
		if info != nil {
			if !registeredJavaSourceValue(info) || !JavaTypeAssignable(info.DynamicType(), "java.util.AbstractList") || info.resolveView("java.util.List") != base || info.resolveView("java.util.AbstractList") != base || state.owner != info {
				panic(NewUnsupportedOperationException("AbstractList projection lost owned source identity"))
			}
			source := info.resolveView(info.DynamicType())
			carrier, owned := source.(JavaObjectInfoCarrier)
			iterable, iterates := source.(JavaIterable)
			if !owned || !iterates || carrier.JavaObjectInfo() != info {
				panic(NewUnsupportedOperationException("AbstractList owner lost its iterable role"))
			}
			return iterable
		}
	}
	return collection
}
func (base *List[T]) CollectionSizeJava2goExecution(e *Execution) int32 {
	if base.abstractView != nil {
		requireExecution(e)
		return base.abstractView.sizeChecked()
	}
	if base.abstractState != nil {
		requireExecution(e)
		source := abstractListCollectionView(base)
		if source == base {
			panic(NewUnsupportedOperationException("AbstractList source is not bound"))
		}
		return CollectionSizeExecution(e, source)
	}
	return base.Size()
}
func (base *List[T]) CollectionListGetJava2goExecution(e *Execution, index int32) any {
	if base.abstractState != nil {
		requireExecution(e)
		if base.abstractView != nil {
			return base.abstractView.get(e, index)
		}
		source := abstractListCollectionView(base)
		if source == base {
			panic(NewUnsupportedOperationException("AbstractList source is not bound"))
		}
		return CollectionListGetExecution(e, source, index)
	}
	return base.rawGet(index)
}
func (base *List[T]) CollectionListSetJava2goExecution(e *Execution, index int32, value any) any {
	if base.abstractState != nil {
		requireExecution(e)
		if base.abstractView != nil {
			return base.abstractView.set(e, index, value)
		}
		source := abstractListCollectionView(base)
		if source == base {
			panic(NewUnsupportedOperationException("AbstractList source is not bound"))
		}
		return CollectionListSetExecution(e, source, index, value)
	}
	return base.collectionSet(index, value)
}
func (base *List[T]) CollectionListAddAtJava2goExecution(e *Execution, index int32, value any) {
	if base.abstractState != nil {
		requireExecution(e)
		if base.abstractView != nil {
			base.abstractView.add(e, index, value)
			return
		}
		source := abstractListCollectionView(base)
		if source == base {
			panic(NewUnsupportedOperationException("AbstractList source is not bound"))
		}
		CollectionListAddAtExecution(e, source, index, value)
		return
	}
	if base.viewRoot == nil && base.fixed {
		base.requireResizable()
	}
	size := base.Size()
	listRangeCheckForAdd(index, size)
	base.requireResizable()
	if base.viewRoot != nil {
		base.viewInsert(index, []any{value}, true)
		return
	}
	base.promoteCollectionStorage()
	base.erasedElements = append(base.erasedElements, nil)
	copy(base.erasedElements[index+1:], base.erasedElements[index:])
	base.erasedElements[index] = value
	base.modCount++
}
func (base *List[T]) CollectionListRemoveAtJava2goExecution(e *Execution, index int32) any {
	if base.abstractState != nil {
		requireExecution(e)
		if base.abstractView != nil {
			return base.abstractView.remove(e, index)
		}
		source := abstractListCollectionView(base)
		if source == base {
			panic(NewUnsupportedOperationException("AbstractList source is not bound"))
		}
		return CollectionListRemoveAtExecution(e, source, index)
	}
	return base.rawRemove(index)
}
func CollectionListAddAtExecution(e *Execution, list JavaIterable, index int32, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	if target, ok := list.(interface{ CollectionListAddAtJava2goExecution(*Execution, int32, any) }); ok {
		target.CollectionListAddAtJava2goExecution(e, index, value)
		return
	}
	panic(NewUnsupportedOperationException("list indexed add is unavailable"))
}
func AbstractListAddExecution(e *Execution, list JavaIterable, value any) bool {
	CollectionListAddAtExecution(e, list, CollectionSizeExecution(e, list), value)
	return true
}
func AbstractListSetExecution(e *Execution, list JavaIterable, index int32, value any) any {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	panic(UnsupportedOperationException{newJavaThrowableBase("UnsupportedOperationException", nil)})
}
func AbstractListAddAtExecution(e *Execution, list JavaIterable, index int32, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	panic(UnsupportedOperationException{newJavaThrowableBase("UnsupportedOperationException", nil)})
}
func AbstractListRemoveAtExecution(e *Execution, list JavaIterable, index int32) any {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	panic(UnsupportedOperationException{newJavaThrowableBase("UnsupportedOperationException", nil)})
}
func listRangeCheckForAdd(index, size int32) {
	if index < 0 || index > size {
		panic(NewJavaIndexOutOfBoundsExceptionMessage(uriRuntimeText("Index: " + strconv.FormatInt(int64(index), 10) + ", Size: " + strconv.FormatInt(int64(size), 10))))
	}
}
func listCheckIndex(index, size int32) {
	if index < 0 || index >= size {
		panic(NewJavaIndexOutOfBoundsExceptionMessage(uriRuntimeText("Index " + strconv.FormatInt(int64(index), 10) + " out of bounds for length " + strconv.FormatInt(int64(size), 10))))
	}
}
func (base *List[T]) abstractListNativeState() *AbstractListState { return base.abstractState }
func abstractListModCount(list JavaIterable) int32 {
	if view, ok := list.(interface{ abstractListViewState() *abstractListView }); ok {
		if state := view.abstractListViewState(); state != nil {
			return state.expected
		}
	}
	if native, ok := list.(interface {
		abstractListNativeState() *AbstractListState
		nativeListModCount() int32
	}); ok {
		if state := native.abstractListNativeState(); state != nil {
			return state.ModCount
		}
		return native.nativeListModCount()
	}
	if source, ok := list.(interface{ AbstractListStateJava2go() *AbstractListState }); ok {
		return source.AbstractListStateJava2go().ModCount
	}
	if native, ok := list.(interface{ nativeListModCount() int32 }); ok {
		return native.nativeListModCount()
	}
	panic(NewUnsupportedOperationException("list has no structural state"))
}
func (list *List[T]) nativeListModCount() int32 { return int32(list.storageRoot().modCount) }

func AbstractListIndexOfExecution(e *Execution, list JavaIterable, value any) int32 {
	cursor := CollectionListIteratorExecution(e, list, 0)
	for IteratorHasNextExecution(e, cursor) {
		item := IteratorNextExecution(e, cursor)
		if listElementEqual(value, item, e) {
			return cursor.PreviousIndexJava2goExecution(e)
		}
	}
	return -1
}
func AbstractListLastIndexOfExecution(e *Execution, list JavaIterable, value any) int32 {
	cursor := CollectionListIteratorExecution(e, list, CollectionSizeExecution(e, list))
	for cursor.HasPreviousJava2goExecution(e) {
		if listElementEqual(value, cursor.PreviousJava2goExecution(e), e) {
			return cursor.NextIndexJava2goExecution(e)
		}
	}
	return -1
}
func AbstractListEqualsExecution(e *Execution, list JavaIterable, other any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	if JavaReferenceEqual(list, other) {
		return true
	}
	if !ObjectInstanceOf(other, "java.util.List") {
		return false
	}
	right := ObjectView[JavaIterable](other, CollectionTypeID)
	leftIt, rightIt := CollectionListIteratorExecution(e, list, 0), CollectionListIteratorExecution(e, right, 0)
	for IteratorHasNextExecution(e, leftIt) && IteratorHasNextExecution(e, rightIt) {
		if !listElementEqual(IteratorNextExecution(e, leftIt), IteratorNextExecution(e, rightIt), e) {
			return false
		}
	}
	return !IteratorHasNextExecution(e, leftIt) && !IteratorHasNextExecution(e, rightIt)
}
func AbstractListHashCodeExecution(e *Execution, list JavaIterable) int32 {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	hash := int32(1)
	for _, value := range ErasedCollectionIterationElements(e, list) {
		hash = 31*hash + ObjectsHashCode(value, e)
	}
	return hash
}
func AbstractListClearExecution(e *Execution, list JavaIterable) {
	CollectionListRemoveRangeExecution(e, list, 0, CollectionSizeExecution(e, list))
}
func AbstractListRemoveRangeExecution(e *Execution, list JavaIterable, from, to int32) {
	cursor := CollectionListIteratorExecution(e, list, from)
	for count := to - from; count > 0; count-- {
		IteratorNextExecution(e, cursor)
		IteratorRemoveExecution(e, cursor)
	}
}
func AbstractListStringExecution(e *Execution, list JavaIterable) *JavaString {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	it := IterableIteratorExecution(e, list)
	if !IteratorHasNextExecution(e, it) {
		return JavaStringLiteralUTF16([]uint16{'[', ']'})
	}
	units := []uint16{'['}
	for {
		value := IteratorNextExecution(e, it)
		if JavaReferenceEqual(value, list) {
			units = append(units, []uint16{'(', 't', 'h', 'i', 's', ' ', 'C', 'o', 'l', 'l', 'e', 'c', 't', 'i', 'o', 'n', ')'}...)
		} else {
			units = append(units, JavaStringTextOperandExecution(e, value).units...)
		}
		if !IteratorHasNextExecution(e, it) {
			break
		}
		units = append(units, ',', ' ')
	}
	units = append(units, ']')
	return NewJavaStringUTF16(units)
}

func (list *List[T]) requireNativeListStorage() {
	if list != nil && list.abstractState != nil {
		panic(NewUnsupportedOperationException("AbstractList projection requires execution-aware source dispatch"))
	}
}
