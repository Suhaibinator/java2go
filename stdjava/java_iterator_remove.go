package stdjava

// JavaIteratorRemovable is the optional erased Iterator.remove operation.
// Iterator acquisition and reads retain the canonical JavaIterator protocol.
type JavaIteratorRemovable interface {
	IteratorRemoveJava2goExecution(*Execution)
}

func IteratorRemoveExecution(execution *Execution, iterator JavaIterator) {
	requireExecution(execution)
	ReferenceRequireNonNull(iterator)
	if removable, ok := iterator.(JavaIteratorRemovable); ok {
		removable.IteratorRemoveJava2goExecution(execution)
		return
	}
	panic(NewUnsupportedOperationException(""))
}
