package stdjava

import "sync/atomic"

// ThreadLocal values live on the logical execution, so pooled workers retain
// values between jobs and terminated executions release their entire store.
// Numeric keys avoid retaining ThreadLocal objects through the store itself.
type ThreadLocal[T any] struct {
	key     uint64
	initial any
}
type threadLocalValue[T any] struct{ value T }

var nextThreadLocalKey atomic.Uint64

func NewThreadLocal[T any]() *ThreadLocal[T] { return &ThreadLocal[T]{key: nextThreadLocalKey.Add(1)} }
func ThreadLocalWithInitial[T any](supplier any) *ThreadLocal[T] {
	ReferenceRequireNonNull(supplier)
	local := NewThreadLocal[T]()
	local.initial = supplier
	return local
}
func (*ThreadLocal[T]) JavaDynamicTypeID() TypeID { return "ThreadLocal" }
func (l *ThreadLocal[T]) Get(execution *Execution) T {
	ReferenceRequireNonNull(l)
	requireExecution(execution)
	if stored, ok := execution.threadLocals[l.key]; ok {
		return stored.(threadLocalValue[T]).value
	}
	value := collectionZero[T]()
	if l.initial != nil {
		value = CallSupplierExecution[T](execution, l.initial)
	}
	l.Set(execution, value)
	return value
}
func (l *ThreadLocal[T]) Set(execution *Execution, value T) {
	ReferenceRequireNonNull(l)
	requireExecution(execution)
	if execution.threadLocals == nil {
		execution.threadLocals = make(map[uint64]any)
	}
	execution.threadLocals[l.key] = threadLocalValue[T]{value}
}
func (l *ThreadLocal[T]) Remove(execution *Execution) {
	ReferenceRequireNonNull(l)
	requireExecution(execution)
	delete(execution.threadLocals, l.key)
}
func init() {
	RegisterJavaType("ThreadLocal", ObjectTypeID)
	RegisterJavaType("Supplier", ObjectTypeID)
}

// SupplierFuncAdapter propagates the initializer caller's execution without
// changing Supplier's Java functional method or checked-exception signature.
type SupplierFuncAdapter[T any] struct{ get func(*Execution) T }

func NewSupplierFuncAdapter[T any](get func(*Execution) T) *SupplierFuncAdapter[T] {
	return &SupplierFuncAdapter[T]{get}
}
func NewPlainSupplierFuncAdapter[T any](get func() T) *SupplierFuncAdapter[T] {
	return NewSupplierFuncAdapter(func(_ *Execution) T { return get() })
}
func (s *SupplierFuncAdapter[T]) Get() T                                     { return s.get(NewExecution()) }
func (s *SupplierFuncAdapter[T]) GetJava2goExecution(execution *Execution) T { return s.get(execution) }

// Supplier is the Java functional interface; the optional hidden method keeps
// callbacks on the execution which invokes them, including pre-created values.
type Supplier[T any] interface{ Get() T }

func GetSupplierExecution[T any](execution *Execution, supplier Supplier[T]) T {
	return CallSupplierExecution[T](execution, supplier)
}
func CallSupplierExecution[T any](execution *Execution, supplier any) T {
	ReferenceRequireNonNull(supplier)
	switch initial := supplier.(type) {
	case interface{ GetJava2goExecution(*Execution) T }:
		return initial.GetJava2goExecution(execution)
	case Supplier[T]:
		return initial.Get()
	case func(*Execution) T:
		return initial(execution)
	case func() T:
		return initial()
	default:
		panic(NewUnsupportedOperationException("Supplier has no get implementation"))
	}
}
