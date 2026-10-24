package stdjava

type IntUnaryOperator interface{ ApplyAsInt(int32) int32 }
type IntUnaryOperatorFuncAdapter struct {
	*ObjectInfo
	call func(*Execution, int32) int32
}

func NewIntUnaryOperatorFuncAdapter(call func(*Execution, int32) int32) *IntUnaryOperatorFuncAdapter {
	return &IntUnaryOperatorFuncAdapter{call: call}
}
func NewPlainIntUnaryOperatorFuncAdapter(call func(int32) int32) *IntUnaryOperatorFuncAdapter {
	return NewIntUnaryOperatorFuncAdapter(func(_ *Execution, v0 int32) int32 { return call(v0) })
}
func (f *IntUnaryOperatorFuncAdapter) ApplyAsInt(v0 int32) int32 {
	return f.ApplyAsIntJava2goExecution(NewExecution(), v0)
}
func (f *IntUnaryOperatorFuncAdapter) ApplyAsIntJava2goExecution(e *Execution, v0 int32) int32 {
	ReferenceRequireNonNull(f)
	return f.call(e, v0)
}
func (f *IntUnaryOperatorFuncAdapter) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.IntUnaryOperator"
}
func CallIntUnaryOperatorExecution(e *Execution, f IntUnaryOperator, v0 int32) int32 {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ ApplyAsIntJava2goExecution(*Execution, int32) int32 }); ok {
		return x.ApplyAsIntJava2goExecution(e, v0)
	}
	return f.ApplyAsInt(v0)
}
func IntUnaryOperatorCallbackExecution(f IntUnaryOperator) func(*Execution, int32) int32 {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 int32) int32 { return CallIntUnaryOperatorExecution(e, f, v0) }
}

type IntBinaryOperator interface{ ApplyAsInt(int32, int32) int32 }
type IntBinaryOperatorFuncAdapter struct {
	*ObjectInfo
	call func(*Execution, int32, int32) int32
}

func NewIntBinaryOperatorFuncAdapter(call func(*Execution, int32, int32) int32) *IntBinaryOperatorFuncAdapter {
	return &IntBinaryOperatorFuncAdapter{call: call}
}
func NewPlainIntBinaryOperatorFuncAdapter(call func(int32, int32) int32) *IntBinaryOperatorFuncAdapter {
	return NewIntBinaryOperatorFuncAdapter(func(_ *Execution, v0 int32, v1 int32) int32 { return call(v0, v1) })
}
func (f *IntBinaryOperatorFuncAdapter) ApplyAsInt(v0 int32, v1 int32) int32 {
	return f.ApplyAsIntJava2goExecution(NewExecution(), v0, v1)
}
func (f *IntBinaryOperatorFuncAdapter) ApplyAsIntJava2goExecution(e *Execution, v0 int32, v1 int32) int32 {
	ReferenceRequireNonNull(f)
	return f.call(e, v0, v1)
}
func (f *IntBinaryOperatorFuncAdapter) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.IntBinaryOperator"
}
func CallIntBinaryOperatorExecution(e *Execution, f IntBinaryOperator, v0 int32, v1 int32) int32 {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface {
		ApplyAsIntJava2goExecution(*Execution, int32, int32) int32
	}); ok {
		return x.ApplyAsIntJava2goExecution(e, v0, v1)
	}
	return f.ApplyAsInt(v0, v1)
}
func IntBinaryOperatorCallbackExecution(f IntBinaryOperator) func(*Execution, int32, int32) int32 {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 int32, v1 int32) int32 { return CallIntBinaryOperatorExecution(e, f, v0, v1) }
}

type LongUnaryOperator interface{ ApplyAsLong(int64) int64 }
type LongUnaryOperatorFuncAdapter struct {
	*ObjectInfo
	call func(*Execution, int64) int64
}

func NewLongUnaryOperatorFuncAdapter(call func(*Execution, int64) int64) *LongUnaryOperatorFuncAdapter {
	return &LongUnaryOperatorFuncAdapter{call: call}
}
func NewPlainLongUnaryOperatorFuncAdapter(call func(int64) int64) *LongUnaryOperatorFuncAdapter {
	return NewLongUnaryOperatorFuncAdapter(func(_ *Execution, v0 int64) int64 { return call(v0) })
}
func (f *LongUnaryOperatorFuncAdapter) ApplyAsLong(v0 int64) int64 {
	return f.ApplyAsLongJava2goExecution(NewExecution(), v0)
}
func (f *LongUnaryOperatorFuncAdapter) ApplyAsLongJava2goExecution(e *Execution, v0 int64) int64 {
	ReferenceRequireNonNull(f)
	return f.call(e, v0)
}
func (f *LongUnaryOperatorFuncAdapter) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.LongUnaryOperator"
}
func CallLongUnaryOperatorExecution(e *Execution, f LongUnaryOperator, v0 int64) int64 {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ ApplyAsLongJava2goExecution(*Execution, int64) int64 }); ok {
		return x.ApplyAsLongJava2goExecution(e, v0)
	}
	return f.ApplyAsLong(v0)
}
func LongUnaryOperatorCallbackExecution(f LongUnaryOperator) func(*Execution, int64) int64 {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 int64) int64 { return CallLongUnaryOperatorExecution(e, f, v0) }
}

type LongBinaryOperator interface{ ApplyAsLong(int64, int64) int64 }
type LongBinaryOperatorFuncAdapter struct {
	*ObjectInfo
	call func(*Execution, int64, int64) int64
}

func NewLongBinaryOperatorFuncAdapter(call func(*Execution, int64, int64) int64) *LongBinaryOperatorFuncAdapter {
	return &LongBinaryOperatorFuncAdapter{call: call}
}
func NewPlainLongBinaryOperatorFuncAdapter(call func(int64, int64) int64) *LongBinaryOperatorFuncAdapter {
	return NewLongBinaryOperatorFuncAdapter(func(_ *Execution, v0 int64, v1 int64) int64 { return call(v0, v1) })
}
func (f *LongBinaryOperatorFuncAdapter) ApplyAsLong(v0 int64, v1 int64) int64 {
	return f.ApplyAsLongJava2goExecution(NewExecution(), v0, v1)
}
func (f *LongBinaryOperatorFuncAdapter) ApplyAsLongJava2goExecution(e *Execution, v0 int64, v1 int64) int64 {
	ReferenceRequireNonNull(f)
	return f.call(e, v0, v1)
}
func (f *LongBinaryOperatorFuncAdapter) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.LongBinaryOperator"
}
func CallLongBinaryOperatorExecution(e *Execution, f LongBinaryOperator, v0 int64, v1 int64) int64 {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface {
		ApplyAsLongJava2goExecution(*Execution, int64, int64) int64
	}); ok {
		return x.ApplyAsLongJava2goExecution(e, v0, v1)
	}
	return f.ApplyAsLong(v0, v1)
}
func LongBinaryOperatorCallbackExecution(f LongBinaryOperator) func(*Execution, int64, int64) int64 {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 int64, v1 int64) int64 { return CallLongBinaryOperatorExecution(e, f, v0, v1) }
}

type UnaryOperator[T any] interface{ Apply(T) T }
type UnaryOperatorFuncAdapter[T any] struct {
	*ObjectInfo
	call func(*Execution, T) T
}

func NewUnaryOperatorFuncAdapter[T any](call func(*Execution, T) T) *UnaryOperatorFuncAdapter[T] {
	return &UnaryOperatorFuncAdapter[T]{call: call}
}
func NewPlainUnaryOperatorFuncAdapter[T any](call func(T) T) *UnaryOperatorFuncAdapter[T] {
	return NewUnaryOperatorFuncAdapter[T](func(_ *Execution, v0 T) T { return call(v0) })
}
func (f *UnaryOperatorFuncAdapter[T]) Apply(v0 T) T {
	return f.ApplyJava2goExecution(NewExecution(), v0)
}
func (f *UnaryOperatorFuncAdapter[T]) ApplyJava2goExecution(e *Execution, v0 T) T {
	ReferenceRequireNonNull(f)
	return f.call(e, v0)
}
func (f *UnaryOperatorFuncAdapter[T]) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.UnaryOperator"
}
func CallUnaryOperatorExecution[T any](e *Execution, f UnaryOperator[T], v0 T) T {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ ApplyJava2goExecution(*Execution, T) T }); ok {
		return x.ApplyJava2goExecution(e, v0)
	}
	return f.Apply(v0)
}
func UnaryOperatorCallbackExecution[T any](f UnaryOperator[T]) func(*Execution, T) T {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 T) T { return CallUnaryOperatorExecution[T](e, f, v0) }
}

type BinaryOperator[T any] interface{ Apply(T, T) T }
type BinaryOperatorFuncAdapter[T any] struct {
	*ObjectInfo
	call func(*Execution, T, T) T
}

func NewBinaryOperatorFuncAdapter[T any](call func(*Execution, T, T) T) *BinaryOperatorFuncAdapter[T] {
	return &BinaryOperatorFuncAdapter[T]{call: call}
}
func NewPlainBinaryOperatorFuncAdapter[T any](call func(T, T) T) *BinaryOperatorFuncAdapter[T] {
	return NewBinaryOperatorFuncAdapter[T](func(_ *Execution, v0 T, v1 T) T { return call(v0, v1) })
}
func (f *BinaryOperatorFuncAdapter[T]) Apply(v0 T, v1 T) T {
	return f.ApplyJava2goExecution(NewExecution(), v0, v1)
}
func (f *BinaryOperatorFuncAdapter[T]) ApplyJava2goExecution(e *Execution, v0 T, v1 T) T {
	ReferenceRequireNonNull(f)
	return f.call(e, v0, v1)
}
func (f *BinaryOperatorFuncAdapter[T]) JavaDynamicTypeID() TypeID {
	if f.ObjectInfo != nil {
		return f.DynamicType()
	}
	return "java.util.function.BinaryOperator"
}
func CallBinaryOperatorExecution[T any](e *Execution, f BinaryOperator[T], v0 T, v1 T) T {
	ReferenceRequireNonNull(f)
	if x, ok := f.(interface{ ApplyJava2goExecution(*Execution, T, T) T }); ok {
		return x.ApplyJava2goExecution(e, v0, v1)
	}
	return f.Apply(v0, v1)
}
func BinaryOperatorCallbackExecution[T any](f BinaryOperator[T]) func(*Execution, T, T) T {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, v0 T, v1 T) T { return CallBinaryOperatorExecution[T](e, f, v0, v1) }
}

func init() {
	RegisterJavaType("java.util.function.IntUnaryOperator", ObjectTypeID)
	RegisterJavaType("java.util.function.IntBinaryOperator", ObjectTypeID)
	RegisterJavaType("java.util.function.LongUnaryOperator", ObjectTypeID)
	RegisterJavaType("java.util.function.LongBinaryOperator", ObjectTypeID)
	RegisterJavaType("java.util.function.UnaryOperator", ObjectTypeID, "java.util.function.Function")
	RegisterJavaType("java.util.function.BinaryOperator", ObjectTypeID, "java.util.function.BiFunction")
}

// Java generic operator parameters remain typed at the SAM boundary; updater
// storage remains erased, and uses the runtime's nominal null-preserving view.
func ReferenceUnaryOperatorCallbackExecution[T any](f UnaryOperator[T], id TypeID) func(*Execution, any) any {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, value any) any { return CallUnaryOperatorExecution(e, f, ObjectView[T](value, id)) }
}
func ReferenceBinaryOperatorCallbackExecution[T any](f BinaryOperator[T], id TypeID) func(*Execution, any, any) any {
	if javaReferenceIsNull(f) {
		return nil
	}
	return func(e *Execution, a, b any) any {
		return CallBinaryOperatorExecution(e, f, ObjectView[T](a, id), ObjectView[T](b, id))
	}
}

func NewIntUnaryOperatorSourceView(info *ObjectInfo, call func(*Execution, int32) int32) *IntUnaryOperatorFuncAdapter {
	return &IntUnaryOperatorFuncAdapter{ObjectInfo: info, call: call}
}

func NewIntBinaryOperatorSourceView(info *ObjectInfo, call func(*Execution, int32, int32) int32) *IntBinaryOperatorFuncAdapter {
	return &IntBinaryOperatorFuncAdapter{ObjectInfo: info, call: call}
}

func NewLongUnaryOperatorSourceView(info *ObjectInfo, call func(*Execution, int64) int64) *LongUnaryOperatorFuncAdapter {
	return &LongUnaryOperatorFuncAdapter{ObjectInfo: info, call: call}
}

func NewLongBinaryOperatorSourceView(info *ObjectInfo, call func(*Execution, int64, int64) int64) *LongBinaryOperatorFuncAdapter {
	return &LongBinaryOperatorFuncAdapter{ObjectInfo: info, call: call}
}

func NewUnaryOperatorSourceView[T any](info *ObjectInfo, call func(*Execution, T) T) *UnaryOperatorFuncAdapter[T] {
	return &UnaryOperatorFuncAdapter[T]{ObjectInfo: info, call: call}
}

func NewBinaryOperatorSourceView[T any](info *ObjectInfo, call func(*Execution, T, T) T) *BinaryOperatorFuncAdapter[T] {
	return &BinaryOperatorFuncAdapter[T]{ObjectInfo: info, call: call}
}
