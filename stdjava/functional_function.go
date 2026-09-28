package stdjava

// Function is a Java object. Native callbacks are only an internal boundary;
// preserving the object keeps aliases, null and caller execution observable.
type Function[T, R any] interface{ Apply(T) R }

type FunctionFuncAdapter[T, R any] struct{ apply func(*Execution, T) R }

func NewFunctionFuncAdapter[T, R any](apply func(*Execution, T) R) *FunctionFuncAdapter[T, R] {
	return &FunctionFuncAdapter[T, R]{apply: apply}
}
func NewPlainFunctionFuncAdapter[T, R any](apply func(T) R) *FunctionFuncAdapter[T, R] {
	return NewFunctionFuncAdapter(func(_ *Execution, value T) R { return apply(value) })
}
func (f *FunctionFuncAdapter[T, R]) Apply(value T) R {
	return f.ApplyJava2goExecution(NewExecution(), value)
}
func (f *FunctionFuncAdapter[T, R]) ApplyJava2goExecution(execution *Execution, value T) R {
	ReferenceRequireNonNull(f)
	return f.apply(execution, value)
}
func (*FunctionFuncAdapter[T, R]) JavaDynamicTypeID() TypeID { return "java.util.function.Function" }

func CallFunctionExecution[T, R any](execution *Execution, function Function[T, R], value T) R {
	ReferenceRequireNonNull(function)
	if implementation, ok := function.(interface{ ApplyJava2goExecution(*Execution, T) R }); ok {
		return implementation.ApplyJava2goExecution(execution, value)
	}
	return function.Apply(value)
}

// Conversion evaluates and captures the Java receiver once, but does not invoke
// or dereference it. The consuming library method performs its null validation.
func FunctionCallbackExecution[T, R any](execution *Execution, function Function[T, R]) func(T) R {
	if javaReferenceIsNull(function) {
		return nil
	}
	return func(value T) R { return CallFunctionExecution(execution, function, value) }
}
func init() { RegisterJavaType("java.util.function.Function", ObjectTypeID) }
