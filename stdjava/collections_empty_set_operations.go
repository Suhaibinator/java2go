package stdjava

// The native companion contracts below do not register general compiler
// collection array, bulk, functional, spliterator or serialization frontends.
func emptyCollectionsRequireNonNull(value any) {
	if javaReferenceIsNull(value) {
		panic(NewNullPointerException(NullString()))
	}
}
func (set *emptySet) CollectionToArrayJava2goExecution(execution *Execution) *ReferenceArray {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return NewReferenceArray(0, ObjectTypeID)
}
func (set *emptySet) CollectionToArrayIntoJava2goExecution(execution *Execution, array *ReferenceArray) *ReferenceArray {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	if array == nil {
		// JDK EmptySet.toArray first reads the length of its formal parameter a.
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8(`Cannot read the array length because "a" is null`)))
	}
	if ReferenceArrayLength(array) > 0 {
		ReferenceArraySet[any](array, 0, nil)
	}
	return array
}
func (set *emptySet) CollectionAddAllJava2goExecution(execution *Execution, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	modified := false
	for _, value := range ErasedCollectionIterationElements(execution, other) {
		if CollectionAddExecution(execution, set, value) {
			modified = true
		}
	}
	return modified
}
func (set *emptySet) CollectionRetainAllJava2goExecution(execution *Execution, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	emptyCollectionsRequireNonNull(other)
	return false
}
func (set *emptySet) CollectionRemoveAllJava2goExecution(execution *Execution, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	emptyCollectionsRequireNonNull(other)
	return AbstractSetRemoveAllExecution(execution, set, other)
}
func (set *emptySet) CollectionForEachJava2goExecution(execution *Execution, action any) {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	emptyCollectionsRequireNonNull(action)
}
func (set *emptySet) CollectionRemoveIfJava2goExecution(execution *Execution, filter any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	emptyCollectionsRequireNonNull(filter)
	return false
}
