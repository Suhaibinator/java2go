package stdjava

// ThrowableObject is the concrete storage for an allocated java.lang.Throwable
// and for source classes directly extending it. Throwable remains the common
// interface used by catches, parameters and return values across all subclasses.
// Copying the Go value preserves the Java reference through ThrowableBase.state.
type ThrowableObject struct{ ThrowableBase }

// NewThrowable is the native Go entry point. Generated Java constructors use
// NewThrowableExecution so a cause's virtual toString receives the caller token.
func NewThrowable(arguments ...any) ThrowableObject {
	return NewThrowableExecution(nil, arguments...)
}

// NewThrowableExecution implements the public no-arg, message, cause and
// message+cause constructors using the shared cause-initialization contract.
// A null String must retain the existing String sentinel at this legacy ABI
// boundary, distinguishing it from an explicitly null Throwable cause.
func NewThrowableExecution(execution *Execution, arguments ...any) ThrowableObject {
	return ThrowableObject{newExceptionConstructorBaseExecution(execution, "Throwable", arguments...)}
}
