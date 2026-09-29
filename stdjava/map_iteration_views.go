package stdjava

// mapKeysIterable retains its source map until each iterator is acquired.
// Slice supports internal collection consumers; Java iteration uses the live
// map cursor instead of materializing those copied key values.
type mapKeysIterable[K, V any] struct{ source *Map[K, V] }

func (view *mapKeysIterable[K, V]) Slice() []K { return view.source.KeySet() }

func MapKeysView[K, V any](source *Map[K, V]) IterableView[K] {
	ReferenceRequireNonNull(source)
	return &mapKeysIterable[K, V]{source: source}
}

func (view *mapKeysIterable[K, V]) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return newMapJavaIterator(view.source, true)
}
