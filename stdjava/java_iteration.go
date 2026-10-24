package stdjava

// JavaIterable is the execution-aware Java iterator factory protocol. It is
// separate from the legacy slice-based Iterable so iteration need not snapshot
// a collection or lose source-defined callback behavior.
type JavaIterable interface {
	IteratorJava2goExecution(*Execution) JavaIterator
}

// JavaIterator preserves the erased result of Iterator.next. A consumer applies
// any required Java checkcast after NextJava2goExecution has completed, so a
// failed cast cannot undo cursor advancement or other callback effects.
// Optional/default Iterator operations are separate, not implied by this slice.
type JavaIterator interface {
	HasNextJava2goExecution(*Execution) bool
	NextJava2goExecution(*Execution) any
}

// IterableIteratorExecution invokes the factory once on the calling Java
// execution. A factory may return null; dereferencing that result is checked by
// the subsequent iterator operation rather than eagerly by this helper.
func IterableIteratorExecution(execution *Execution, iterable JavaIterable) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(iterable)
	return iterable.IteratorJava2goExecution(execution)
}

// IteratorHasNextExecution invokes the supplied iterator without prefetching or
// consuming an element. Callback exceptions propagate to the caller unchanged.
func IteratorHasNextExecution(execution *Execution, iterator JavaIterator) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(iterator)
	return iterator.HasNextJava2goExecution(execution)
}

// IteratorNextExecution returns the exact erased callback result, including
// null, without copying, converting, or applying a consumer's checkcast.
func IteratorNextExecution(execution *Execution, iterator JavaIterator) any {
	requireExecution(execution)
	ReferenceRequireNonNull(iterator)
	return iterator.NextJava2goExecution(execution)
}
