package stdjava

// ObjectDefaultFinalizeExecution implements the empty java.lang.Object body
// selected by an explicit source super.finalize() call. It does not dispatch a
// source override or arrange garbage collection or finalizer scheduling.
func ObjectDefaultFinalizeExecution(execution *Execution, value any) {
	requireExecution(execution)
	ReferenceRequireNonNull(value)
}
