package stdjava

// Execution-aware constructor companions preserve the caller's logical Java
// execution when a cause-only overload invokes a source Throwable.toString.
// The original Go constructors continue to create an entry execution through
// newExceptionConstructorBase when no Java execution token is available.
func NewExceptionExecution(execution *Execution, arguments ...any) Exception {
	return Exception{newExceptionConstructorBaseExecution(execution, "Exception", arguments...)}
}
func NewRuntimeExceptionExecution(execution *Execution, arguments ...any) RuntimeException {
	return RuntimeException{newExceptionConstructorBaseExecution(execution, "RuntimeException", arguments...)}
}
func NewIllegalArgumentExceptionExecution(execution *Execution, arguments ...any) IllegalArgumentException {
	return IllegalArgumentException{newExceptionConstructorBaseExecution(execution, "IllegalArgumentException", arguments...)}
}
func NewIllegalStateExceptionExecution(execution *Execution, arguments ...any) IllegalStateException {
	return IllegalStateException{newExceptionConstructorBaseExecution(execution, "IllegalStateException", arguments...)}
}
func NewUnsupportedEncodingExceptionExecution(execution *Execution, arguments ...any) UnsupportedEncodingException {
	return UnsupportedEncodingException{newExceptionConstructorBaseExecution(execution, "UnsupportedEncodingException", arguments...)}
}
