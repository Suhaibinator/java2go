package stdjava

const (
	CollectionTypeID         TypeID = "java.util.Collection"
	SetTypeID                TypeID = "java.util.Set"
	MapTypeID                TypeID = "java.util.Map"
	AbstractCollectionTypeID TypeID = "java.util.AbstractCollection"
	AbstractSetTypeID        TypeID = "java.util.AbstractSet"
	AbstractMapTypeID        TypeID = "java.util.AbstractMap"
)

func init() {
	RegisterJavaType(CollectionTypeID, ObjectTypeID, IterableTypeID)
	RegisterJavaType(SetTypeID, ObjectTypeID, CollectionTypeID)
	RegisterJavaType(MapTypeID, ObjectTypeID)
	RegisterJavaType(AbstractCollectionTypeID, ObjectTypeID, CollectionTypeID)
	RegisterJavaType(AbstractSetTypeID, AbstractCollectionTypeID, SetTypeID)
	RegisterJavaType(AbstractMapTypeID, ObjectTypeID, MapTypeID)
}

// Abstract collection algorithms invoke the source object's exact virtual
// callbacks. They own no collection storage and retain the calling execution.
func AbstractCollectionIsEmptyExecution(execution *Execution, collection JavaIterable) bool {
	return CollectionSizeExecution(execution, collection) == 0
}

func AbstractCollectionContainsExecution(execution *Execution, collection JavaIterable, value any) bool {
	cursor := IterableIteratorExecution(execution, collection)
	for IteratorHasNextExecution(execution, cursor) {
		item := IteratorNextExecution(execution, cursor)
		matched := javaReferenceIsNull(item)
		if !javaReferenceIsNull(value) {
			matched = ObjectEqualsExecution(execution, value, item)
		}
		if matched {
			return true
		}
	}
	return false
}

func AbstractCollectionRemoveExecution(execution *Execution, collection JavaIterable, value any) bool {
	cursor := IterableIteratorExecution(execution, collection)
	for IteratorHasNextExecution(execution, cursor) {
		item := IteratorNextExecution(execution, cursor)
		matched := javaReferenceIsNull(item)
		if !javaReferenceIsNull(value) {
			matched = ObjectEqualsExecution(execution, value, item)
		}
		if matched {
			IteratorRemoveExecution(execution, cursor)
			return true
		}
	}
	return false
}

func AbstractCollectionClearExecution(execution *Execution, collection JavaIterable) {
	cursor := IterableIteratorExecution(execution, collection)
	for IteratorHasNextExecution(execution, cursor) {
		IteratorNextExecution(execution, cursor)
		IteratorRemoveExecution(execution, cursor)
	}
}

func AbstractCollectionContainsAllExecution(execution *Execution, collection, other JavaIterable) bool {
	cursor := IterableIteratorExecution(execution, other)
	for IteratorHasNextExecution(execution, cursor) {
		if !CollectionContainsExecution(execution, collection, IteratorNextExecution(execution, cursor)) {
			return false
		}
	}
	return true
}

func CollectionContainsAllExecution(execution *Execution, collection, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(collection)
	if source, ok := collection.(interface {
		CollectionContainsAllJava2goExecution(*Execution, JavaIterable) bool
	}); ok {
		return source.CollectionContainsAllJava2goExecution(execution, other)
	}
	return AbstractCollectionContainsAllExecution(execution, collection, other)
}

func AbstractSetEqualsExecution(execution *Execution, collection JavaIterable, other any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(collection)
	if JavaReferenceEqual(collection, other) {
		return true
	}
	if !ObjectInstanceOf(other, SetTypeID) {
		return false
	}
	right := ObjectView[JavaIterable](other, CollectionTypeID)
	rightSize := CollectionSizeExecution(execution, right)
	if rightSize != CollectionSizeExecution(execution, collection) {
		return false
	}
	// JDK catches only around containsAll; a size callback exception propagates.
	return func() (equal bool) {
		defer collectionEqualityFailure(&equal)
		return CollectionContainsAllExecution(execution, collection, right)
	}()
}

func AbstractSetHashCodeExecution(execution *Execution, collection JavaIterable) int32 {
	cursor := IterableIteratorExecution(execution, collection)
	var hash int32
	for IteratorHasNextExecution(execution, cursor) {
		hash += ObjectsHashCode(IteratorNextExecution(execution, cursor), execution)
	}
	return hash
}

func AbstractSetRemoveAllExecution(execution *Execution, collection, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(other)
	modified := false
	size := CollectionSizeExecution(execution, collection)
	otherSize := CollectionSizeExecution(execution, other)
	if size > otherSize {
		cursor := IterableIteratorExecution(execution, other)
		for IteratorHasNextExecution(execution, cursor) {
			removed := CollectionRemoveExecution(execution, collection, IteratorNextExecution(execution, cursor))
			modified = modified || removed
		}
		return modified
	}
	cursor := IterableIteratorExecution(execution, collection)
	for IteratorHasNextExecution(execution, cursor) {
		if CollectionContainsExecution(execution, other, IteratorNextExecution(execution, cursor)) {
			IteratorRemoveExecution(execution, cursor)
			modified = true
		}
	}
	return modified
}
