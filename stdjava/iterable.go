package stdjava

// Iterable preserves a collection reference until iteration begins. Lists and
// sets implement it directly; native slice views use a small adapter.
type Iterable[T any] interface{ Slice() []T }

// IterableView exposes both internal collection access and Java iteration while
// preserving the same view object and backing collection.
type IterableView[T any] interface {
	Iterable[T]
	JavaIterable
}

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

// Map values remain a view of the source map until consumed.
type mapValuesIterable[K, V any] struct{ source *Map[K, V] }

func (view *mapValuesIterable[K, V]) Slice() []V { return view.source.Values() }

// Both collection protocols retain the same view without copying the backing map.
func MapValuesView[K, V any](source *Map[K, V]) IterableView[V] {
	ReferenceRequireNonNull(source)
	if source.valueView == nil {
		source.valueView = &mapValuesIterable[K, V]{source: source}
	}
	return source.valueView.(*mapValuesIterable[K, V])
}
