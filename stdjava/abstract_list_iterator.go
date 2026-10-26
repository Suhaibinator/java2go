package stdjava

// JavaListIterator is an execution-aware role after nominal Java conversion.
type JavaListIterator interface {
	JavaIterator
	HasPreviousJava2goExecution(*Execution) bool
	PreviousJava2goExecution(*Execution) any
	NextIndexJava2goExecution(*Execution) int32
	PreviousIndexJava2goExecution(*Execution) int32
	ListIteratorSetJava2goExecution(*Execution, any)
	ListIteratorAddJava2goExecution(*Execution, any)
}
type abstractListIterator struct {
	source                 JavaIterable
	cursor, last, expected int32
	native                 bool
}

func AbstractListIteratorExecution(e *Execution, list JavaIterable) JavaIterator {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	return &abstractListIterator{source: list, last: -1, expected: abstractListModCount(list)}
}
func AbstractListListIteratorExecution(e *Execution, list JavaIterable, index int32) JavaListIterator {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	listRangeCheckForAdd(index, CollectionSizeExecution(e, list))
	return &abstractListIterator{source: list, cursor: index, last: -1, expected: abstractListModCount(list)}
}
func CollectionListIteratorExecution(e *Execution, list JavaIterable, index int32) JavaListIterator {
	requireExecution(e)
	ReferenceRequireNonNull(list)
	list = abstractListCollectionView(list)
	if role, ok := list.(interface {
		CollectionListIteratorJava2goExecution(*Execution, int32) JavaListIterator
	}); ok {
		return role.CollectionListIteratorJava2goExecution(e, index)
	}
	if view, ok := list.(interface{ abstractListViewState() *abstractListView }); ok {
		if state := view.abstractListViewState(); state != nil {
			return state.listIterator(e, index)
		}
	}
	listRangeCheckForAdd(index, CollectionSizeExecution(e, list))
	return &abstractListIterator{source: list, cursor: index, last: -1, expected: abstractListModCount(list), native: true}
}
func (cursor *abstractListIterator) check() {
	if cursor.expected != abstractListModCount(cursor.source) {
		panic(NewJavaConcurrentModificationExceptionMessage(nil))
	}
}
func (cursor *abstractListIterator) HasNextJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return cursor.cursor != CollectionSizeExecution(e, cursor.source)
}
func (cursor *abstractListIterator) HasPreviousJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return cursor.cursor != 0
}
func (cursor *abstractListIterator) NextIndexJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	return cursor.cursor
}
func (cursor *abstractListIterator) PreviousIndexJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	return cursor.cursor - 1
}
func (cursor *abstractListIterator) boundsFailure(e *Execution, action func() any) (value any) {
	defer func() {
		if f := recover(); f != nil {
			f = NormalizePanic(f)
			if CaughtAs(f, "IndexOutOfBoundsException") {
				cursor.check()
				panic(newNoSuchElementExceptionFromCause(e, f))
			}
			panic(f)
		}
	}()
	return action()
}
func newNoSuchElementExceptionFromCause(e *Execution, cause any) NoSuchElementException {
	failure := NewNoSuchElementException("")
	failure.ThrowableBase = newJavaThrowableCause(e, "NoSuchElementException", cause)
	return failure
}
func (cursor *abstractListIterator) NextJava2goExecution(e *Execution) any {
	requireExecution(e)
	cursor.check()
	if cursor.native && cursor.cursor >= CollectionSizeExecution(e, cursor.source) {
		panic(NoSuchElementException{newJavaThrowableBase("NoSuchElementException", nil)})
	}
	return cursor.boundsFailure(e, func() any {
		index := cursor.cursor
		value := CollectionListGetExecution(e, cursor.source, index)
		cursor.last = index
		cursor.cursor = index + 1
		return value
	})
}
func (cursor *abstractListIterator) PreviousJava2goExecution(e *Execution) any {
	requireExecution(e)
	cursor.check()
	if cursor.native && cursor.cursor <= 0 {
		panic(NoSuchElementException{newJavaThrowableBase("NoSuchElementException", nil)})
	}
	return cursor.boundsFailure(e, func() any {
		index := cursor.cursor - 1
		value := CollectionListGetExecution(e, cursor.source, index)
		cursor.last = index
		cursor.cursor = index
		return value
	})
}
func (cursor *abstractListIterator) mutation(action func()) {
	defer func() {
		if f := recover(); f != nil {
			f = NormalizePanic(f)
			if CaughtAs(f, "IndexOutOfBoundsException") {
				panic(NewJavaConcurrentModificationExceptionMessage(nil))
			}
			panic(f)
		}
	}()
	action()
}
func (cursor *abstractListIterator) IteratorRemoveJava2goExecution(e *Execution) {
	requireExecution(e)
	if cursor.last < 0 {
		panic(NewJavaIllegalStateExceptionMessage(nil))
	}
	cursor.check()
	cursor.mutation(func() {
		CollectionListRemoveAtExecution(e, cursor.source, cursor.last)
		if cursor.last < cursor.cursor {
			cursor.cursor--
		}
		cursor.last = -1
		cursor.expected = abstractListModCount(cursor.source)
	})
}
func (cursor *abstractListIterator) ListIteratorSetJava2goExecution(e *Execution, value any) {
	requireExecution(e)
	if cursor.last < 0 {
		panic(NewJavaIllegalStateExceptionMessage(nil))
	}
	cursor.check()
	cursor.mutation(func() {
		CollectionListSetExecution(e, cursor.source, cursor.last, value)
		cursor.expected = abstractListModCount(cursor.source)
	})
}
func (cursor *abstractListIterator) ListIteratorAddJava2goExecution(e *Execution, value any) {
	requireExecution(e)
	cursor.check()
	cursor.mutation(func() {
		index := cursor.cursor
		CollectionListAddAtExecution(e, cursor.source, index, value)
		cursor.last = -1
		cursor.cursor = index + 1
		cursor.expected = abstractListModCount(cursor.source)
	})
}
func (*abstractListIterator) nativeJavaInterfaces() []TypeID {
	return []TypeID{"java.util.ListIterator"}
}
func init() { RegisterJavaType("java.util.ListIterator", "java.util.Iterator") }
