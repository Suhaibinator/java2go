package stdjava

const CollectionsEmptySetTypeID TypeID = "java.util.Collections$EmptySet"

// The JDK EMPTY_SET allocation is shared by every erased instantiation.
// ObjectInfo gives its nonzero carrier the same identity in all nominal views.
type emptySet struct{ *ObjectInfo }

var emptySetSingleton = &emptySet{}

func init() {
	RegisterJavaType(CollectionsEmptySetTypeID, AbstractSetTypeID, SerializableTypeID)
	emptySetSingleton.ObjectInfo = NewObjectInfo(CollectionsEmptySetTypeID, func(requested TypeID) any {
		if JavaTypeAssignable(CollectionsEmptySetTypeID, requested) {
			return emptySetSingleton
		}
		return nil
	})
}

func EmptySet() JavaIterable { return emptySetSingleton }

func (set *emptySet) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return emptySetIteratorSingleton
}
func (set *emptySet) CollectionAddJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	panic(NewUnsupportedOperationException(NullString()))
}
func (set *emptySet) CollectionRemoveJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return false
}
func (set *emptySet) CollectionContainsJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return false
}
func (set *emptySet) CollectionSizeJava2goExecution(execution *Execution) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return 0
}
func (set *emptySet) CollectionIsEmptyJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return true
}
func (set *emptySet) CollectionClearJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
}
func (set *emptySet) CollectionContainsAllJava2goExecution(execution *Execution, other JavaIterable) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	if javaReferenceIsNull(other) {
		// The pinned JDK body dereferences its Collection parameter c here.
		message := `Cannot invoke "` + string(CollectionTypeID) + `.isEmpty()" because "c" is null`
		panic(NewJavaNullPointerExceptionMessage(JavaStringFromHostUTF8(message)))
	}
	return CollectionIsEmptyExecution(execution, other)
}
func (set *emptySet) EqualsJava2goExecution(execution *Execution, other any) bool {
	ReferenceRequireNonNull(set)
	// Legacy ObjectsEqual calls have no execution; explicit callers keep theirs.
	if execution == nil {
		execution = NewExecution()
	}
	return AbstractSetEqualsExecution(execution, set, other)
}
func (set *emptySet) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(set)
	return 0
}
func (set *emptySet) StringJava2goExecution(execution *Execution) *JavaString {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return JavaStringLiteralUTF16([]uint16{'[', ']'})
}

// These private/native roles preserve compatibility with existing erased Set
// equality without adding storage or exposing a mutable generic facade.
func (set *emptySet) javaSetElements() []any { ReferenceRequireNonNull(set); return nil }
func (set *emptySet) Size() int32            { ReferenceRequireNonNull(set); return 0 }
func (set *emptySet) Contains(value any, executions ...*Execution) bool {
	ReferenceRequireNonNull(set)
	return false
}
func (set *emptySet) Equals(other any) bool { return set.EqualsJava2goExecution(nil, other) }
func (set *emptySet) HashCode() int32       { return set.HashCodeJava2goExecution(nil) }
func (set *emptySet) String() string        { ReferenceRequireNonNull(set); return "[]" }
