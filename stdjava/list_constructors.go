package stdjava

// NewListWithArgument handles the capacity and collection-copy overloads. A
// copy owns its element storage; its element references retain Java identity.
func NewListWithArgument[T any](argument any, executions ...*Execution) *List[T] {
	switch capacity := argument.(type) {
	case int:
		return newListWithCapacity[T](int64(capacity))
	case int32:
		return newListWithCapacity[T](int64(capacity))
	}
	ReferenceRequireNonNull(argument)
	if source, ok := argument.(JavaIterable); ok {
		execution := optionalComparisonExecution(executions)
		if execution == nil {
			execution = NewExecution()
		}
		list := NewList[T]()
		list.erasedStorage = true
		for _, item := range ErasedCollectionIterationElements(execution, source) {
			list.erasedElements = append(list.erasedElements, item)
		}
		return list
	}
	source := AsIterable[T](argument)
	return NewListFrom(source.Slice()...)
}
func newListWithCapacity[T any](capacity int64) *List[T] {
	if capacity < 0 {
		panic(NewIllegalArgumentException("Illegal Capacity"))
	}
	return &List[T]{elements: make([]T, 0, int(capacity))}
}
