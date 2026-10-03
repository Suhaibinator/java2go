package stdjava

type BiFunction[T, U, R any] interface{ Apply(a T, b U) R }
type BiFunctionFuncAdapter[T, U, R any] struct {
	*ObjectInfo
	call func(*Execution, T, U) R
}

func NewBiFunctionFuncAdapter[T, U, R any](call func(*Execution, T, U) R) *BiFunctionFuncAdapter[T, U, R] {
	return &BiFunctionFuncAdapter[T, U, R]{call: call}
}
func NewPlainBiFunctionFuncAdapter[T, U, R any](call func(T, U) R) *BiFunctionFuncAdapter[T, U, R] {
	return NewBiFunctionFuncAdapter(func(_ *Execution, a T, b U) R { return call(a, b) })
}
func NewBiFunctionSourceView[T, U, R any](info *ObjectInfo, call func(*Execution, T, U) R) *BiFunctionFuncAdapter[T, U, R] {
	return &BiFunctionFuncAdapter[T, U, R]{ObjectInfo: info, call: call}
}
func (f *BiFunctionFuncAdapter[T, U, R]) Apply(a T, b U) R {
	return f.ApplyJava2goExecution(NewExecution(), a, b)
}
func (f *BiFunctionFuncAdapter[T, U, R]) ApplyJava2goExecution(e *Execution, a T, b U) R {
	ReferenceRequireNonNull(f)
	return f.call(e, a, b)
}
func (f *BiFunctionFuncAdapter[T, U, R]) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.BiFunction"
}
func CallBiFunctionExecution[T, U, R any](e *Execution, f BiFunction[T, U, R], a T, b U) R {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ ApplyJava2goExecution(*Execution, T, U) R }); ok {
		return x.ApplyJava2goExecution(e, a, b)
	}
	return f.Apply(a, b)
}
func BiFunctionCallbackExecution[T, U, R any](e *Execution, f BiFunction[T, U, R]) func(T, U) R {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(a T, b U) R { return CallBiFunctionExecution(e, f, a, b) }
}

type Consumer[T any] interface{ Accept(a T) }
type ConsumerFuncAdapter[T any] struct {
	*ObjectInfo
	call func(*Execution, T)
}

func NewConsumerFuncAdapter[T any](call func(*Execution, T)) *ConsumerFuncAdapter[T] {
	return &ConsumerFuncAdapter[T]{call: call}
}
func NewPlainConsumerFuncAdapter[T any](call func(T)) *ConsumerFuncAdapter[T] {
	return NewConsumerFuncAdapter(func(_ *Execution, a T) { call(a) })
}
func NewConsumerSourceView[T any](info *ObjectInfo, call func(*Execution, T)) *ConsumerFuncAdapter[T] {
	return &ConsumerFuncAdapter[T]{ObjectInfo: info, call: call}
}
func (f *ConsumerFuncAdapter[T]) Accept(a T) { f.AcceptJava2goExecution(NewExecution(), a) }
func (f *ConsumerFuncAdapter[T]) AcceptJava2goExecution(e *Execution, a T) {
	ReferenceRequireNonNull(f)
	f.call(e, a)
}
func (f *ConsumerFuncAdapter[T]) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.Consumer"
}
func CallConsumerExecution[T any](e *Execution, f Consumer[T], a T) {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ AcceptJava2goExecution(*Execution, T) }); ok {
		x.AcceptJava2goExecution(e, a)
		return
	}
	f.Accept(a)
}
func ConsumerCallbackExecution[T any](e *Execution, f Consumer[T]) func(T) {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(a T) { CallConsumerExecution(e, f, a) }
}
func init() {
	RegisterJavaType("java.util.function.BiFunction", ObjectTypeID)
	RegisterJavaType("java.util.function.Consumer", ObjectTypeID)
}

func isNativeFunctionalTypeID(id TypeID) bool {
	switch id {
	case "java.util.function.Function", "java.util.function.BiFunction", "java.util.function.Consumer", "java.util.function.UnaryOperator", "java.util.function.BinaryOperator", "java.util.function.IntUnaryOperator", "java.util.function.IntBinaryOperator", "java.util.function.LongUnaryOperator", "java.util.function.LongBinaryOperator":
		return true
	}
	return false
}

// The default method validates after at construction, while the captured Java
// receiver is invoked under each consuming caller's Execution.
func FunctionAndThenExecution[T, R, V any](e *Execution, f Function[T, R], after Function[R, V]) Function[T, V] {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(after)
	return NewFunctionFuncAdapter(func(caller *Execution, value T) V {
		return CallFunctionExecution(caller, after, CallFunctionExecution(caller, f, value))
	})
}
func BiFunctionAndThenExecution[T, U, R, V any](e *Execution, f BiFunction[T, U, R], after Function[R, V]) BiFunction[T, U, V] {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(after)
	return NewBiFunctionFuncAdapter(func(caller *Execution, a T, b U) V {
		return CallFunctionExecution(caller, after, CallBiFunctionExecution(caller, f, a, b))
	})
}
func FunctionComposeExecution[T, R, V any](e *Execution, f Function[T, R], before Function[V, T]) Function[V, R] {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(before)
	return NewFunctionFuncAdapter(func(caller *Execution, value V) R {
		return CallFunctionExecution(caller, f, CallFunctionExecution(caller, before, value))
	})
}
func ConsumerAndThenExecution[T any](e *Execution, f Consumer[T], after Consumer[T]) Consumer[T] {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(after)
	return NewConsumerFuncAdapter(func(caller *Execution, value T) {
		CallConsumerExecution(caller, f, value)
		CallConsumerExecution(caller, after, value)
	})
}
func IntUnaryOperatorAndThenExecution(e *Execution, f IntUnaryOperator, other IntUnaryOperator) IntUnaryOperator {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(other)
	return NewIntUnaryOperatorFuncAdapter(func(caller *Execution, value int32) int32 {
		return CallIntUnaryOperatorExecution(caller, other, CallIntUnaryOperatorExecution(caller, f, value))
	})
}
func IntUnaryOperatorComposeExecution(e *Execution, f IntUnaryOperator, other IntUnaryOperator) IntUnaryOperator {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(other)
	return NewIntUnaryOperatorFuncAdapter(func(caller *Execution, value int32) int32 {
		return CallIntUnaryOperatorExecution(caller, f, CallIntUnaryOperatorExecution(caller, other, value))
	})
}
func LongUnaryOperatorAndThenExecution(e *Execution, f LongUnaryOperator, other LongUnaryOperator) LongUnaryOperator {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(other)
	return NewLongUnaryOperatorFuncAdapter(func(caller *Execution, value int64) int64 {
		return CallLongUnaryOperatorExecution(caller, other, CallLongUnaryOperatorExecution(caller, f, value))
	})
}
func LongUnaryOperatorComposeExecution(e *Execution, f LongUnaryOperator, other LongUnaryOperator) LongUnaryOperator {
	ReferenceRequireNonNull(f)
	ReferenceRequireNonNull(other)
	return NewLongUnaryOperatorFuncAdapter(func(caller *Execution, value int64) int64 {
		return CallLongUnaryOperatorExecution(caller, f, CallLongUnaryOperatorExecution(caller, other, value))
	})
}
