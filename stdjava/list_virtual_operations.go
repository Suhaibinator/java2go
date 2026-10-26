package stdjava

func CollectionListIndexOfExecution(e *Execution, list JavaIterable, value any) int32 {
	requireExecution(e)
	list = abstractListCollectionView(list)
	if role, ok := list.(interface{ CollectionListIndexOfJava2goExecution(*Execution, any) int32 }); ok {
		return role.CollectionListIndexOfJava2goExecution(e, value)
	}
	if native, ok := list.(interface {
		IndexOf(any, ...*Execution) int32
	}); ok {
		return native.IndexOf(value, e)
	}
	index := int32(0)
	for _, item := range ErasedCollectionIterationElements(e, list) {
		if listElementEqual(value, item, e) {
			return index
		}
		index++
	}
	return -1
}
func CollectionListLastIndexOfExecution(e *Execution, list JavaIterable, value any) int32 {
	requireExecution(e)
	list = abstractListCollectionView(list)
	if role, ok := list.(interface{ CollectionListLastIndexOfJava2goExecution(*Execution, any) int32 }); ok {
		return role.CollectionListLastIndexOfJava2goExecution(e, value)
	}
	for index := CollectionSizeExecution(e, list) - 1; index >= 0; index-- {
		if listElementEqual(value, CollectionListGetExecution(e, list, index), e) {
			return index
		}
	}
	return -1
}
func CollectionAddAllExecution(e *Execution, list, other JavaIterable) bool {
	return ArrayDequeCollectionAddAllExecution(e, abstractListCollectionView(list), other)
}
func CollectionRemoveAllExecution(e *Execution, list, other JavaIterable) bool {
	return ArrayDequeCollectionRemoveAllExecution(e, abstractListCollectionView(list), other)
}
func CollectionRetainAllExecution(e *Execution, list, other JavaIterable) bool {
	return ArrayDequeCollectionRetainAllExecution(e, abstractListCollectionView(list), other)
}
func CollectionToArrayExecution(e *Execution, list JavaIterable) *ReferenceArray {
	return ArrayDequeCollectionToArrayExecution(e, abstractListCollectionView(list))
}
func CollectionToArrayIntoExecution(e *Execution, list JavaIterable, array *ReferenceArray) *ReferenceArray {
	return ArrayDequeCollectionToArrayIntoExecution(e, abstractListCollectionView(list), array)
}
func AbstractListAddAllExecution(e *Execution, list, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(other)
	changed := false
	for _, value := range ErasedCollectionIterationElements(e, other) {
		if CollectionAddExecution(e, list, value) {
			changed = true
		}
	}
	return changed
}
func AbstractListAddAllAtExecution(e *Execution, list JavaIterable, index int32, other JavaIterable) bool {
	listRangeCheckForAdd(index, CollectionSizeExecution(e, list))
	requireExecution(e)
	ReferenceRequireNonNull(other)
	changed := false
	for _, value := range ErasedCollectionIterationElements(e, other) {
		CollectionListAddAtExecution(e, list, index, value)
		index++
		changed = true
	}
	return changed
}
func AbstractListRemoveAllExecution(e *Execution, list, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(other)
	changed := false
	it := IterableIteratorExecution(e, list)
	for IteratorHasNextExecution(e, it) {
		if CollectionContainsExecution(e, other, IteratorNextExecution(e, it)) {
			IteratorRemoveExecution(e, it)
			changed = true
		}
	}
	return changed
}
func AbstractListRetainAllExecution(e *Execution, list, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(other)
	changed := false
	it := IterableIteratorExecution(e, list)
	for IteratorHasNextExecution(e, it) {
		if !CollectionContainsExecution(e, other, IteratorNextExecution(e, it)) {
			IteratorRemoveExecution(e, it)
			changed = true
		}
	}
	return changed
}

// The JDK treats size as an estimate and stores each next value immediately.
// A callback or ArrayStoreException must observe the already-written prefix.
func AbstractListToArrayExecution(e *Execution, list JavaIterable) *ReferenceArray {
	requireExecution(e)
	result := NewReferenceArray(CollectionSizeExecution(e, list), ObjectTypeID)
	it := IterableIteratorExecution(e, list)
	for i := int32(0); i < ReferenceArrayLength(result); i++ {
		if !IteratorHasNextExecution(e, it) {
			return abstractListArrayCopy(result, i)
		}
		ReferenceArraySet(result, i, IteratorNextExecution(e, it))
	}
	if IteratorHasNextExecution(e, it) {
		return abstractListFinishArray(e, result, it)
	}
	return result
}
func AbstractListToArrayIntoExecution(e *Execution, list JavaIterable, array *ReferenceArray) *ReferenceArray {
	requireExecution(e)
	size := CollectionSizeExecution(e, list)
	length := ReferenceArrayLength(array)
	result := array
	if length < size {
		result = NewReferenceArray(size, array.ComponentType())
	}
	it := IterableIteratorExecution(e, list)
	for i := int32(0); i < ReferenceArrayLength(result); i++ {
		if !IteratorHasNextExecution(e, it) {
			if result == array {
				ReferenceArraySet[any](result, i, nil)
			} else if length < i {
				return abstractListArrayCopy(result, i)
			} else {
				copy(array.elements[:i], result.elements[:i])
				if length > i {
					ReferenceArraySet[any](array, i, nil)
				}
			}
			return array
		}
		ReferenceArraySet(result, i, IteratorNextExecution(e, it))
	}
	if IteratorHasNextExecution(e, it) {
		return abstractListFinishArray(e, result, it)
	}
	return result
}
func abstractListArrayCopy(array *ReferenceArray, length int32) *ReferenceArray {
	result := NewReferenceArray(length, array.ComponentType())
	copy(result.elements, array.elements)
	return result
}
func abstractListFinishArray(e *Execution, array *ReferenceArray, it JavaIterator) *ReferenceArray {
	index := ReferenceArrayLength(array)
	for IteratorHasNextExecution(e, it) {
		if index == ReferenceArrayLength(array) {
			length := ReferenceArrayLength(array)
			array = abstractListArrayCopy(array, length+(length>>1)+1)
		}
		ReferenceArraySet(array, index, IteratorNextExecution(e, it))
		index++
	}
	if index != ReferenceArrayLength(array) {
		return abstractListArrayCopy(array, index)
	}
	return array
}
func AbstractListListIteratorDefaultExecution(e *Execution, list JavaIterable) JavaListIterator {
	return CollectionListIteratorExecution(e, list, 0)
}
func ListIteratorHasPreviousExecution(e *Execution, it JavaListIterator) bool {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	return it.HasPreviousJava2goExecution(e)
}
func ListIteratorPreviousExecution(e *Execution, it JavaListIterator) any {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	return it.PreviousJava2goExecution(e)
}
func ListIteratorNextIndexExecution(e *Execution, it JavaListIterator) int32 {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	return it.NextIndexJava2goExecution(e)
}
func ListIteratorPreviousIndexExecution(e *Execution, it JavaListIterator) int32 {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	return it.PreviousIndexJava2goExecution(e)
}
func ListIteratorSetExecution(e *Execution, it JavaListIterator, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	it.ListIteratorSetJava2goExecution(e, value)
}
func ListIteratorAddExecution(e *Execution, it JavaListIterator, value any) {
	requireExecution(e)
	ReferenceRequireNonNull(it)
	it.ListIteratorAddJava2goExecution(e, value)
}

func CollectionListAddAllAtExecution(e *Execution, list JavaIterable, index int32, other JavaIterable) bool {
	requireExecution(e)
	list = abstractListCollectionView(list)
	if role, ok := list.(interface {
		CollectionListAddAllAtJava2goExecution(*Execution, int32, JavaIterable) bool
	}); ok {
		return role.CollectionListAddAllAtJava2goExecution(e, index, other)
	}
	return AbstractListAddAllAtExecution(e, list, index, other)
}
func (list *List[T]) CollectionIsEmptyJava2goExecution(e *Execution) bool {
	if list.abstractState != nil {
		requireExecution(e)
		if list.abstractView != nil {
			return list.abstractView.sizeChecked() == 0
		}
		return CollectionIsEmptyExecution(e, abstractListCollectionView(list))
	}
	return list.IsEmpty()
}
func (list *List[T]) CollectionContainsJava2goExecution(e *Execution, value any) bool {
	if list.abstractState != nil {
		requireExecution(e)
		if list.abstractView != nil {
			return AbstractCollectionContainsExecution(e, list, value)
		}
		return CollectionContainsExecution(e, abstractListCollectionView(list), value)
	}
	return list.Contains(value, e)
}
func (list *List[T]) CollectionRemoveJava2goExecution(e *Execution, value any) bool {
	if list.abstractState != nil {
		requireExecution(e)
		if list.abstractView != nil {
			return AbstractCollectionRemoveExecution(e, list, value)
		}
		return CollectionRemoveExecution(e, abstractListCollectionView(list), value)
	}
	return list.RemoveObject(value, e)
}
func (list *List[T]) CollectionClearJava2goExecution(e *Execution) {
	if list.abstractState != nil {
		requireExecution(e)
		if list.abstractView != nil {
			view := list.abstractView
			view.check()
			CollectionListRemoveRangeExecution(e, view.root, view.offset, view.offset+view.size)
			view.update(-view.size)
			return
		}
		CollectionClearExecution(e, abstractListCollectionView(list))
		return
	}
	list.Clear()
}
func (list *List[T]) CollectionContainsAllJava2goExecution(e *Execution, other JavaIterable) bool {
	if list.abstractState != nil && list.abstractView == nil {
		requireExecution(e)
		return CollectionContainsAllExecution(e, abstractListCollectionView(list), other)
	}
	return AbstractCollectionContainsAllExecution(e, list, other)
}
func (list *List[T]) CollectionToArrayJava2goExecution(e *Execution) *ReferenceArray {
	if list.abstractState != nil {
		requireExecution(e)
		if list.abstractView != nil {
			return AbstractListToArrayExecution(e, list)
		}
		return CollectionToArrayExecution(e, abstractListCollectionView(list))
	}
	return ReferenceArrayLiteral(ObjectTypeID, list.javaListElements()...)
}
func (list *List[T]) CollectionToArrayIntoJava2goExecution(e *Execution, array *ReferenceArray) *ReferenceArray {
	requireExecution(e)
	ReferenceRequireNonNull(array)
	if list.abstractState != nil {
		if list.abstractView != nil {
			return AbstractListToArrayIntoExecution(e, list, array)
		}
		return CollectionToArrayIntoExecution(e, abstractListCollectionView(list), array)
	}
	values := list.javaListElements()
	result := array
	if ReferenceArrayLength(result) < int32(len(values)) {
		result = NewReferenceArray(len(values), array.ComponentType())
	}
	for i, v := range values {
		ReferenceArraySet(result, i, v)
	}
	if ReferenceArrayLength(result) > int32(len(values)) {
		ReferenceArraySet[any](result, len(values), nil)
	}
	return result
}
func (list *List[T]) CollectionAddAllJava2goExecution(e *Execution, other JavaIterable) bool {
	requireExecution(e)
	ReferenceRequireNonNull(other)
	if list.abstractState != nil {
		if list.abstractView != nil {
			return list.abstractView.addAll(e, list.abstractView.size, other)
		}
		return CollectionAddAllExecution(e, abstractListCollectionView(list), other)
	}
	if list.fixed && list.viewRoot == nil {
		return AbstractListAddAllExecution(e, list, other)
	}
	values := CollectionToArrayExecution(e, other)
	count := ReferenceArrayLength(values)
	if list.viewRoot != nil {
		if count == 0 {
			return false
		}
		entries := make([]any, count)
		for i := range entries {
			entries[i] = ReferenceArrayGet[any](values, i, ObjectTypeID)
		}
		list.viewInsert(list.viewSize, entries, true)
		return true
	}
	list.requireResizable()
	list.modCount++
	if count == 0 {
		return false
	}
	list.promoteCollectionStorage()
	for i := int32(0); i < count; i++ {
		list.erasedElements = append(list.erasedElements, ReferenceArrayGet[any](values, i, ObjectTypeID))
	}
	return true
}
func (list *List[T]) CollectionRemoveAllJava2goExecution(e *Execution, other JavaIterable) bool {
	if list.abstractState != nil && list.abstractView == nil {
		return CollectionRemoveAllExecution(e, abstractListCollectionView(list), other)
	}
	return list.collectionBatchRemove(e, other, false)
}
func (list *List[T]) CollectionRetainAllJava2goExecution(e *Execution, other JavaIterable) bool {
	if list.abstractState != nil && list.abstractView == nil {
		return CollectionRetainAllExecution(e, abstractListCollectionView(list), other)
	}
	return list.collectionBatchRemove(e, other, true)
}
func (list *List[T]) collectionBatchRemove(e *Execution, other JavaIterable, retain bool) (changed bool) {
	requireExecution(e)
	ReferenceRequireNonNull(other)
	if list.abstractState != nil || list.allocationKind == listAllocationLinked || list.viewRoot != nil || list.fixed || list.array != nil {
		if retain {
			return AbstractListRetainAllExecution(e, list, other)
		}
		return AbstractListRemoveAllExecution(e, list, other)
	}
	size := list.Size()
	read, write := int32(0), int32(0)
	defer func() {
		for read < size {
			if write != read {
				list.collectionSet(write, list.rawGet(read))
			}
			read++
			write++
		}
		if write != size {
			list.removeRange(write, size)
			list.modCount += uint64(size - write - 1)
			changed = true
		}
	}()
	for read < size {
		value := list.rawGet(read)
		keep := CollectionContainsExecution(e, other, value) == retain
		if keep {
			if write != read {
				list.collectionSet(write, value)
			}
			write++
		}
		read++
	}
	return write != size
}
func CollectionListRemoveRangeExecution(e *Execution, list JavaIterable, from, to int32) {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	if role, ok := list.(interface {
		CollectionListRemoveRangeJava2goExecution(*Execution, int32, int32)
	}); ok {
		role.CollectionListRemoveRangeJava2goExecution(e, from, to)
		return
	}
	AbstractListRemoveRangeExecution(e, list, from, to)
}

func (list *List[T]) CollectionListAddAllAtJava2goExecution(e *Execution, index int32, other JavaIterable) bool {
	requireExecution(e)
	if list.abstractState != nil {
		if list.abstractView != nil {
			return list.abstractView.addAll(e, index, other)
		}
		return CollectionListAddAllAtExecution(e, abstractListCollectionView(list), index, other)
	}
	listRangeCheckForAdd(index, list.Size())
	ReferenceRequireNonNull(other)
	if list.fixed && list.viewRoot == nil {
		return AbstractListAddAllAtExecution(e, list, index, other)
	}
	array := CollectionToArrayExecution(e, other)
	count := ReferenceArrayLength(array)
	values := make([]any, count)
	for i := range values {
		values[i] = ReferenceArrayGet[any](array, i, ObjectTypeID)
	}
	if list.viewRoot != nil {
		if count == 0 {
			return false
		}
		list.viewInsert(index, values, true)
		return true
	}
	list.requireResizable()
	list.modCount++
	if count == 0 {
		return false
	}
	list.promoteCollectionStorage()
	oldSize := len(list.erasedElements)
	list.erasedElements = append(list.erasedElements, make([]any, count)...)
	copy(list.erasedElements[int(index)+int(count):], list.erasedElements[int(index):oldSize])
	copy(list.erasedElements[int(index):], values)
	return true
}
