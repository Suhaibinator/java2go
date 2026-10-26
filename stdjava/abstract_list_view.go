package stdjava

import "strconv"

type abstractListView struct {
	root                   JavaIterable
	parent                 *abstractListView
	offset, size, expected int32
	random                 bool
}

func (list *List[T]) abstractListViewState() *abstractListView { return list.abstractView }
func (view *abstractListView) check() {
	if abstractListModCount(view.root) != view.expected {
		panic(NewJavaConcurrentModificationExceptionMessage(nil))
	}
}
func (view *abstractListView) sizeChecked() int32 { view.check(); return view.size }
func (view *abstractListView) get(e *Execution, index int32) any {
	listCheckIndex(index, view.size)
	view.check()
	return CollectionListGetExecution(e, view.root, view.offset+index)
}
func (view *abstractListView) set(e *Execution, index int32, value any) any {
	listCheckIndex(index, view.size)
	view.check()
	return CollectionListSetExecution(e, view.root, view.offset+index, value)
}
func (view *abstractListView) update(delta int32) {
	for current := view; current != nil; current = current.parent {
		current.size += delta
		current.expected = abstractListModCount(view.root)
	}
}
func (view *abstractListView) add(e *Execution, index int32, value any) {
	listRangeCheckForAdd(index, view.size)
	view.check()
	CollectionListAddAtExecution(e, view.root, view.offset+index, value)
	view.update(1)
}
func (view *abstractListView) remove(e *Execution, index int32) any {
	listCheckIndex(index, view.size)
	view.check()
	old := CollectionListRemoveAtExecution(e, view.root, view.offset+index)
	view.update(-1)
	return old
}
func (view *abstractListView) listIterator(e *Execution, index int32) JavaListIterator {
	view.check()
	listRangeCheckForAdd(index, view.size)
	return &abstractListSubIterator{view: view, root: CollectionListIteratorExecution(e, view.root, view.offset+index)}
}
func AbstractListSubListTypedExecution[T any](e *Execution, list JavaIterable, from, to int32) *List[T] {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	size := CollectionSizeExecution(e, list)
	if from < 0 {
		panic(NewJavaIndexOutOfBoundsExceptionMessage(uriRuntimeText("fromIndex = " + strconv.FormatInt(int64(from), 10))))
	}
	if to > size {
		panic(NewJavaIndexOutOfBoundsExceptionMessage(uriRuntimeText("toIndex = " + strconv.FormatInt(int64(to), 10))))
	}
	if from > to {
		panic(NewJavaIllegalArgumentExceptionMessage(uriRuntimeText("fromIndex(" + strconv.FormatInt(int64(from), 10) + ") > toIndex(" + strconv.FormatInt(int64(to), 10) + ")")))
	}
	view := &abstractListView{root: list, offset: from, size: to - from, expected: abstractListModCount(list), random: ObjectInstanceOf(list, "java.util.RandomAccess")}
	if parent, ok := list.(interface{ abstractListViewState() *abstractListView }); ok {
		if previous := parent.abstractListViewState(); previous != nil {
			view.root = previous.root
			view.parent = previous
			view.offset += previous.offset
		}
	}
	state := view.root.(interface{ AbstractListStateJava2go() *AbstractListState }).AbstractListStateJava2go()
	return &List[T]{allocationKind: listAllocationAbstract, abstractState: state, abstractView: view}
}
func CollectionListSubListTypedExecution[T any](e *Execution, list JavaIterable, from, to int32) *List[T] {
	return ObjectView[*List[T]](CollectionListSubListExecution(e, list, from, to), "java.util.List")
}

type abstractListSubIterator struct {
	view *abstractListView
	root JavaListIterator
}

func (it *abstractListSubIterator) HasNextJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return it.NextIndexJava2goExecution(e) < it.view.size
}
func (it *abstractListSubIterator) HasPreviousJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return it.PreviousIndexJava2goExecution(e) >= 0
}
func (it *abstractListSubIterator) NextIndexJava2goExecution(e *Execution) int32 {
	return it.root.NextIndexJava2goExecution(e) - it.view.offset
}
func (it *abstractListSubIterator) PreviousIndexJava2goExecution(e *Execution) int32 {
	return it.root.PreviousIndexJava2goExecution(e) - it.view.offset
}
func (it *abstractListSubIterator) NextJava2goExecution(e *Execution) any {
	if !it.HasNextJava2goExecution(e) {
		panic(NoSuchElementException{newJavaThrowableBase("NoSuchElementException", nil)})
	}
	return it.root.NextJava2goExecution(e)
}
func (it *abstractListSubIterator) PreviousJava2goExecution(e *Execution) any {
	if !it.HasPreviousJava2goExecution(e) {
		panic(NoSuchElementException{newJavaThrowableBase("NoSuchElementException", nil)})
	}
	return it.root.PreviousJava2goExecution(e)
}
func (it *abstractListSubIterator) IteratorRemoveJava2goExecution(e *Execution) {
	IteratorRemoveExecution(e, it.root)
	it.view.update(-1)
}
func (it *abstractListSubIterator) ListIteratorSetJava2goExecution(e *Execution, value any) {
	it.root.ListIteratorSetJava2goExecution(e, value)
}
func (it *abstractListSubIterator) ListIteratorAddJava2goExecution(e *Execution, value any) {
	it.root.ListIteratorAddJava2goExecution(e, value)
	it.view.update(1)
}
func (*abstractListSubIterator) nativeJavaInterfaces() []TypeID {
	return []TypeID{"java.util.ListIterator"}
}
func init() {
	RegisterJavaType("java.util.AbstractList$SubList", "java.util.AbstractList")
	RegisterJavaType("java.util.AbstractList$RandomAccessSubList", "java.util.AbstractList$SubList", "java.util.RandomAccess")
}

func (view *abstractListView) addAll(e *Execution, index int32, other JavaIterable) bool {
	requireExecution(e)
	listRangeCheckForAdd(index, view.size)
	count := CollectionSizeExecution(e, other)
	if count == 0 {
		return false
	}
	view.check()
	CollectionListAddAllAtExecution(e, view.root, view.offset+index, other)
	view.update(count)
	return true
}
