package stdjava

const SupplierTypeID TypeID = "java.util.function.Supplier"

// JavaSupplier erases only the result carrier. Runtime-owned implementations
// retain the original Java reference and the consuming caller's Execution.
type JavaSupplier interface {
	supplierGetJava2goExecution(*Execution) any
	supplierRootInfo() *ObjectInfo
}

func (*SupplierFuncAdapter[T]) nativeJavaInterfaces() []TypeID { return []TypeID{SupplierTypeID} }
func (*SupplierFuncAdapter[T]) supplierRootInfo() *ObjectInfo  { return nil }
func (s *SupplierFuncAdapter[T]) supplierGetJava2goExecution(e *Execution) any {
	requireExecution(e)
	ReferenceRequireNonNull(s)
	return s.get(e)
}

// SupplierSourceView binds a declaration-selected body to its original source
// allocation. This view does not give an unknown lambda an interface class ID.
type SupplierSourceView[T any] struct {
	rootInfo *ObjectInfo
	get      func(*Execution) T
}

func NewSupplierSourceView[T any](rootInfo *ObjectInfo, get func(*Execution) T) *SupplierSourceView[T] {
	if rootInfo == nil || !registeredJavaSourceValue(rootInfo) || !JavaTypeAssignable(rootInfo.DynamicType(), SupplierTypeID) || get == nil {
		panic(NewUnsupportedOperationException("Supplier source declaration has no complete reference protocol"))
	}
	return &SupplierSourceView[T]{rootInfo: rootInfo, get: get}
}
func (s *SupplierSourceView[T]) JavaObjectInfo() *ObjectInfo {
	if s == nil {
		return nil
	}
	return s.rootInfo
}
func (s *SupplierSourceView[T]) supplierRootInfo() *ObjectInfo { return s.JavaObjectInfo() }
func (s *SupplierSourceView[T]) Get() T                        { return s.GetJava2goExecution(NewExecution()) }
func (s *SupplierSourceView[T]) GetJava2goExecution(e *Execution) T {
	requireExecution(e)
	ReferenceRequireNonNull(s)
	return s.get(e)
}
func (s *SupplierSourceView[T]) supplierGetJava2goExecution(e *Execution) any {
	return s.GetJava2goExecution(e)
}

// Source providers must return a sealed view of the same ObjectInfo; a detached
// or differently owned view cannot donate a body merely through its Go shape.
func SupplierReferenceView(value any) JavaSupplier {
	if nilJavaReference(value) {
		return nil
	}
	if carrier, ok := value.(JavaObjectInfoCarrier); ok {
		if info := carrier.JavaObjectInfo(); info != nil {
			if !registeredJavaSourceValue(info) || !JavaTypeAssignable(info.DynamicType(), SupplierTypeID) {
				panic(NewUnsupportedOperationException("Supplier source reference has no canonical ancestry"))
			}
			view, ok := info.resolveView(SupplierTypeID).(JavaSupplier)
			if !ok || nilJavaReference(view) || view.supplierRootInfo() != info {
				panic(NewUnsupportedOperationException("Supplier source reference has no owned view"))
			}
			return view
		}
	}
	return ReferenceCheckCast[JavaSupplier](value, SupplierTypeID)
}

func SupplierGetReferenceExecution(e *Execution, supplier any) any {
	requireExecution(e)
	ReferenceRequireNonNull(supplier)
	return SupplierReferenceView(supplier).supplierGetJava2goExecution(e)
}

func init() { RegisterJavaType(SupplierTypeID, ObjectTypeID) }
