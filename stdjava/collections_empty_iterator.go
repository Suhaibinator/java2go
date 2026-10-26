package stdjava

const CollectionsEmptyIteratorTypeID TypeID = "java.util.Collections$EmptyIterator"

// The shared cursor has object identity but no mutable traversal state.
type emptyJavaIterator struct{ *ObjectInfo }

var emptySetIteratorSingleton = &emptyJavaIterator{}

func init() {
	RegisterJavaType(CollectionsEmptyIteratorTypeID, ObjectTypeID, IteratorTypeID)
	emptySetIteratorSingleton.ObjectInfo = NewObjectInfo(CollectionsEmptyIteratorTypeID, func(requested TypeID) any {
		if JavaTypeAssignable(CollectionsEmptyIteratorTypeID, requested) {
			return emptySetIteratorSingleton
		}
		return nil
	})
}
func (cursor *emptyJavaIterator) HasNextJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(cursor)
	return false
}
func (cursor *emptyJavaIterator) NextJava2goExecution(execution *Execution) any {
	requireExecution(execution)
	ReferenceRequireNonNull(cursor)
	panic(NewNoSuchElementException(NullString()))
}
func (cursor *emptyJavaIterator) IteratorRemoveJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(cursor)
	panic(NewIllegalStateExceptionExecution(execution))
}
func (cursor *emptyJavaIterator) IteratorForEachRemainingJava2goExecution(execution *Execution, action any) {
	requireExecution(execution)
	ReferenceRequireNonNull(cursor)
	emptyCollectionsRequireNonNull(action)
}
