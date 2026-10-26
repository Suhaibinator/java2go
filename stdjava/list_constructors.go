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
	return newListFromCollection[T](argument, listAllocationUnknown, executions...)
}
func newListWithCapacity[T any](capacity int64) *List[T] {
	return newListWithAllocationCapacity[T](capacity, listAllocationUnknown)
}
