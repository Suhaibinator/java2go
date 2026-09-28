package stdjava

// Iterable preserves a collection reference until iteration begins. Lists and
// sets implement it directly; native slice views use a small adapter.
type Iterable[T any] interface{ Slice() []T }

type sliceIterable[T any] struct{ elements []T }

func (view *sliceIterable[T]) Slice() []T { return view.elements }

func AsIterable[T any](value any) Iterable[T] {
	if javaReferenceIsNull(value) {
		return nil
	}
	switch collection := value.(type) {
	case Iterable[T]:
		return collection
	case []T:
		return &sliceIterable[T]{collection}
	default:
		panic(NewClassCastException("incompatible collection element type"))
	}
}

// CollectionIterationElements fetches each list slot when Java's iterator
// advances, so array-backed views observe writes to slots not yet visited.
func CollectionIterationElements[T any](collection Iterable[T]) func(func(int, T) bool) {
	return func(yield func(int, T) bool) {
		ReferenceRequireNonNull(collection)
		if list, ok := collection.(*List[T]); ok && list.array != nil {
			for index := int32(0); index < list.Size(); index++ {
				if !yield(int(index), list.Get(index)) {
					return
				}
			}
			return
		}
		for index, value := range collection.Slice() {
			if !yield(index, value) {
				return
			}
		}
	}
}
