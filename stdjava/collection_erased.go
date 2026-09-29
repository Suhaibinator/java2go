package stdjava

import "sort"

// Erased collection operations keep the original receiver. No wrapper is
// allocated, so aliases, Object identity, and monitor ownership are unchanged.
func CollectionAddExecution(execution *Execution, collection JavaIterable, value any) bool {
	ReferenceRequireNonNull(collection)
	if mutable, ok := collection.(interface{ collectionAdd(any, *Execution) bool }); ok {
		return mutable.collectionAdd(value, execution)
	}
	panic(NewUnsupportedOperationException("collection add is unavailable"))
}
func CollectionRemoveExecution(execution *Execution, collection JavaIterable, value any) bool {
	ReferenceRequireNonNull(collection)
	if mutable, ok := collection.(interface{ RemoveObject(any, ...*Execution) bool }); ok {
		return mutable.RemoveObject(value, execution)
	}
	if mutable, ok := collection.(interface{ Remove(any, ...*Execution) bool }); ok {
		return mutable.Remove(value, execution)
	}
	panic(NewUnsupportedOperationException("collection remove is unavailable"))
}
func CollectionContainsExecution(execution *Execution, collection JavaIterable, value any) bool {
	ReferenceRequireNonNull(collection)
	if readable, ok := collection.(interface{ Contains(any, ...*Execution) bool }); ok {
		return readable.Contains(value, execution)
	}
	for _, item := range ErasedCollectionIterationElements(execution, collection) {
		if ObjectsEqual(value, item, execution) {
			return true
		}
	}
	return false
}
func CollectionSizeExecution(execution *Execution, collection JavaIterable) int32 {
	ReferenceRequireNonNull(collection)
	if readable, ok := collection.(interface{ Size() int32 }); ok {
		return readable.Size()
	}
	var size int32
	for range ErasedCollectionIterationElements(execution, collection) {
		size++
	}
	return size
}
func CollectionIsEmptyExecution(execution *Execution, collection JavaIterable) bool {
	ReferenceRequireNonNull(collection)
	if readable, ok := collection.(interface{ IsEmpty() bool }); ok {
		return readable.IsEmpty()
	}
	return !IteratorHasNextExecution(execution, IterableIteratorExecution(execution, collection))
}
func CollectionClearExecution(execution *Execution, collection JavaIterable) {
	ReferenceRequireNonNull(collection)
	if mutable, ok := collection.(interface{ Clear() }); ok {
		mutable.Clear()
		return
	}
	panic(NewUnsupportedOperationException("collection clear is unavailable"))
}
func ErasedCollectionIterationElements(execution *Execution, collection JavaIterable) func(func(int, any) bool) {
	return func(yield func(int, any) bool) {
		cursor := IterableIteratorExecution(execution, collection)
		for index := 0; IteratorHasNextExecution(execution, cursor); index++ {
			if !yield(index, IteratorNextExecution(execution, cursor)) {
				return
			}
		}
	}
}

// A raw Java write cannot check the instantiation's T: ArrayList stores Object.
// Promote the existing allocation's storage once; typed consumers check reads.
func (list *List[T]) collectionAdd(value any, execution *Execution) bool {
	list.requireResizable()
	list.promoteCollectionStorage()
	list.modCount++
	list.erasedElements = append(list.erasedElements, value)
	return true
}
func (list *List[T]) rawGet(index int32) any {
	if list.array != nil {
		return ReferenceArrayGet[any](list.array, index, ObjectTypeID)
	}
	if list.erasedStorage {
		return list.erasedElements[index]
	}
	return list.elements[index]
}
func (list *List[T]) rawSet(index int32, value any) {
	if list.array != nil {
		ReferenceArraySet(list.array, index, value)
		return
	}
	if list.erasedStorage {
		list.erasedElements[index] = value
		return
	}
	list.elements[index] = collectionElementView[T](value)
}
func (list *List[T]) rawRemove(index int32) any {
	list.requireResizable()
	old := list.rawGet(index)
	list.modCount++
	if list.erasedStorage {
		list.erasedElements = append(list.erasedElements[:index], list.erasedElements[index+1:]...)
	} else {
		list.elements = append(list.elements[:index], list.elements[index+1:]...)
	}
	return old
}
func collectionElementView[T any](value any) T {
	// Existing runtime callers also use native primitive instantiations.
	if direct, ok := value.(T); ok {
		return direct
	}
	return ObjectView[T](value, ObjectTypeID)
}

func ErasedCollectionSlice[T any](execution *Execution, collection JavaIterable) []T {
	var elements []T
	for _, item := range ErasedCollectionIterationElements(execution, collection) {
		elements = append(elements, collectionElementView[T](item))
	}
	return elements
}

func (view *mapValuesIterable[K, V]) Size() int32 { return view.source.Size() }
func (view *mapValuesIterable[K, V]) Clear()      { view.source.Clear() }
func (view *mapValuesIterable[K, V]) Contains(value any, execution ...*Execution) bool {
	return view.source.ContainsValue(value, execution...)
}
func (view *mapValuesIterable[K, V]) RemoveObject(value any, executions ...*Execution) bool {
	execution := optionalComparisonExecution(executions)
	for _, record := range view.source.entries {
		if ObjectsEqual(value, record.entry.Value, execution) {
			view.source.Remove(record.entry.Key, execution)
			return true
		}
	}
	return false
}

func (list *List[T]) GetObject(index int32) any { return list.rawGet(index) }
func (list *List[T]) SetObject(index int32, element T) any {
	old := list.rawGet(index)
	list.rawSet(index, element)
	return old
}
func (list *List[T]) RemoveAtObject(index int32) any { return list.rawRemove(index) }

func (list *List[T]) promoteCollectionStorage() {
	if list.erasedStorage {
		return
	}
	list.erasedElements = make([]any, len(list.elements), cap(list.elements))
	for index, element := range list.elements {
		list.erasedElements[index] = element
	}
	list.elements = nil
	list.erasedStorage = true
}
func CollectionListGetExecution(execution *Execution, collection JavaIterable, index int32) any {
	ReferenceRequireNonNull(collection)
	if list, ok := collection.(interface{ rawGet(int32) any }); ok {
		return list.rawGet(index)
	}
	panic(NewUnsupportedOperationException("list get is unavailable"))
}
func CollectionListSetExecution(execution *Execution, collection JavaIterable, index int32, value any) any {
	ReferenceRequireNonNull(collection)
	if list, ok := collection.(interface{ collectionSet(int32, any) any }); ok {
		return list.collectionSet(index, value)
	}
	panic(NewUnsupportedOperationException("list set is unavailable"))
}
func (list *List[T]) collectionSet(index int32, value any) any {
	old := list.rawGet(index)
	if list.array == nil && !list.fixed {
		list.promoteCollectionStorage()
	}
	list.rawSet(index, value)
	return old
}
func CollectionSortOrderedExecution(execution *Execution, collection JavaIterable) {
	ReferenceRequireNonNull(collection)
	if list, ok := collection.(interface{ collectionSort(*Execution) }); ok {
		list.collectionSort(execution)
		return
	}
	panic(NewUnsupportedOperationException("list sort is unavailable"))
}
func (list *List[T]) collectionSort(execution *Execution) {
	if list.array != nil {
		sort.SliceStable(list.array.elements, func(i, j int) bool {
			return javaCompareValuesExecution(execution, list.array.elements[i], list.array.elements[j]) < 0
		})
	} else if list.erasedStorage {
		sort.SliceStable(list.erasedElements, func(i, j int) bool {
			return javaCompareValuesExecution(execution, list.erasedElements[i], list.erasedElements[j]) < 0
		})
	} else {
		switch elements := any(list.elements).(type) {
		case []string:
			SortSlice(elements)
		case []int32:
			SortSlice(elements)
		case []int64:
			SortSlice(elements)
		case []int16:
			SortSlice(elements)
		case []int8:
			SortSlice(elements)
		default:
			SortSliceStableNatural(list.elements, execution)
		}
	}
}

func CollectionListRemoveAtExecution(execution *Execution, collection JavaIterable, index int32) any {
	ReferenceRequireNonNull(collection)
	if list, ok := collection.(interface{ rawRemove(int32) any }); ok {
		return list.rawRemove(index)
	}
	panic(NewUnsupportedOperationException("list remove is unavailable"))
}
