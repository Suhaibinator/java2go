package stdjava

const (
	IterableTypeID TypeID = "java.lang.Iterable"
	IteratorTypeID TypeID = "java.util.Iterator"
)

// iterableFactory retains one Java object identity for each lambda allocation.
// Its nonzero-sized pointer uses the existing runtime reference/monitor identity
// path, just like FunctionFuncAdapter and CallableFuncAdapter.
type iterableFactory struct {
	factory func(*Execution) JavaIterator
}

// NewIterableExecution captures an iterator factory without invoking it. The
// callback receives the execution that later invokes iterator(), not the one
// that allocated the lambda. A null iterator result remains null until used.
func NewIterableExecution(factory func(*Execution) JavaIterator) JavaIterable {
	return &iterableFactory{factory: factory}
}

func (iterable *iterableFactory) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(iterable)
	return iterable.factory(execution)
}

func (*iterableFactory) JavaDynamicTypeID() TypeID { return IterableTypeID }

func init() {
	RegisterJavaType(IterableTypeID, ObjectTypeID)
	RegisterJavaType(IteratorTypeID, ObjectTypeID)
}
