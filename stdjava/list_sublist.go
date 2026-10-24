package stdjava

// SubList retains the original allocation and its structural revision. Only
// mutations through this view update this view and its ancestors; independent
// siblings and descendants keep their revision and fail at their next check.
func (list *List[T]) SubList(from, to int32) *List[T] {
	size := list.uncheckedSize()
	if from < 0 {
		panic(NewIndexOutOfBoundsException("fromIndex"))
	}
	if to > size {
		panic(NewIndexOutOfBoundsException("toIndex"))
	}
	if from > to {
		panic(NewIllegalArgumentException("fromIndex > toIndex"))
	}
	view := &List[T]{viewRoot: list.storageRoot(), viewOffset: from, viewSize: to - from, modCount: list.modCount}
	if list.viewRoot != nil {
		view.viewParent = list
		view.viewOffset += list.viewOffset
	}
	return view
}
func (list *List[T]) storageRoot() *List[T] {
	if list.viewRoot != nil {
		return list.viewRoot
	}
	return list
}
func (list *List[T]) uncheckedSize() int32 {
	if list.viewRoot != nil {
		return list.viewSize
	}
	return list.Size()
}
func (list *List[T]) checkViewModification() {
	if list.viewRoot != nil && list.modCount != list.viewRoot.modCount {
		panic(NewConcurrentModificationException(""))
	}
}
func (list *List[T]) checkElementIndex(index int32) {
	if index < 0 || index >= list.uncheckedSize() {
		panic(NewIndexOutOfBoundsException("Index"))
	}
}
func (list *List[T]) updateViewSizes(delta int32) {
	for view := list; view != nil && view.viewRoot != nil; view = view.viewParent {
		view.viewSize += delta
		view.modCount = view.viewRoot.modCount
	}
}
func (list *List[T]) viewInsert(index int32, values []any, erased bool) {
	if index < 0 || index > list.viewSize {
		panic(NewIndexOutOfBoundsException("Index"))
	}
	list.checkViewModification()
	list.requireResizable()
	root := list.viewRoot
	at := int(list.viewOffset + index)
	if erased {
		root.promoteCollectionStorage()
	}
	if root.erasedStorage {
		root.erasedElements = append(root.erasedElements, make([]any, len(values))...)
		copy(root.erasedElements[at+len(values):], root.erasedElements[at:len(root.erasedElements)-len(values)])
		copy(root.erasedElements[at:], values)
	} else {
		typed := make([]T, len(values))
		for i, value := range values {
			typed[i] = collectionElementView[T](value)
		}
		root.elements = append(root.elements, make([]T, len(values))...)
		copy(root.elements[at+len(values):], root.elements[at:len(root.elements)-len(values)])
		copy(root.elements[at:], typed)
	}
	root.modCount++
	list.updateViewSizes(int32(len(values)))
}

// Clear vacated slots before reslicing so removed object references are released.
func (list *List[T]) removeRange(from, to int32) {
	if from != to {
		list.requireResizable()
	}
	if list.fixed {
		return
	}
	list.modCount++
	if list.erasedStorage {
		end := len(list.erasedElements)
		copy(list.erasedElements[from:], list.erasedElements[to:])
		size := end - int(to-from)
		clear(list.erasedElements[size:])
		list.erasedElements = list.erasedElements[:size]
	} else {
		end := len(list.elements)
		copy(list.elements[from:], list.elements[to:])
		size := end - int(to-from)
		clear(list.elements[size:])
		list.elements = list.elements[:size]
	}
}
func (list *List[T]) collectionSubList(from, to int32) JavaIterable { return list.SubList(from, to) }
func CollectionListSubListExecution(execution *Execution, collection JavaIterable, from, to int32) JavaIterable {
	ReferenceRequireNonNull(collection)
	if list, ok := collection.(interface {
		collectionSubList(int32, int32) JavaIterable
	}); ok {
		return list.collectionSubList(from, to)
	}
	panic(NewUnsupportedOperationException("list subList is unavailable"))
}

// List.sort takes a snapshot, then advances its iterator before each set. The
// iterator observes structural changes made by comparisons and distinguishes
// exhaustion after a view shrinks from invalidation by another collection.
func (list *List[T]) writeSortedView(execution *Execution, count int, set func(int32)) {
	if execution == nil {
		execution = NewExecution()
	}
	cursor := list.IteratorJava2goExecution(execution)
	for index := 0; index < count; index++ {
		cursor.NextJava2goExecution(execution)
		set(int32(index))
	}
}
