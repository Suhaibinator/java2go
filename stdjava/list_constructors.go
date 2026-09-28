package stdjava

// NewListWithArgument handles the capacity and collection-copy overloads. A
// copy owns its element storage; its element references retain Java identity.
func NewListWithArgument[T any](argument any) *List[T] {
	switch capacity := argument.(type) {
	case int:
		return newListWithCapacity[T](int64(capacity))
	case int32:
		return newListWithCapacity[T](int64(capacity))
	}
	ReferenceRequireNonNull(argument)
	source := AsIterable[T](argument)
	return NewListFrom(source.Slice()...)
}
func newListWithCapacity[T any](capacity int64) *List[T] {
	if capacity < 0 {
		panic(NewIllegalArgumentException("Illegal Capacity"))
	}
	return &List[T]{elements: make([]T, 0, int(capacity))}
}
