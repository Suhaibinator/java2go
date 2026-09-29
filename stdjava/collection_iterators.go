package stdjava

// List iteration observes current slots rather than a Slice snapshot. Like
// ArrayList.Itr, hasNext compares the live size and next checks structural
// revision before reading; replacement via set remains visible and nonstructural.
type listJavaIterator[T any] struct {
	source   *List[T]
	index    int32
	expected uint64
}

func (list *List[T]) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(list)
	return &listJavaIterator[T]{source: list, expected: list.modCount}
}
func (cursor *listJavaIterator[T]) HasNextJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	return cursor.index != cursor.source.Size()
}
func (cursor *listJavaIterator[T]) NextJava2goExecution(execution *Execution) any {
	requireExecution(execution)
	if cursor.expected != cursor.source.modCount {
		panic(NewConcurrentModificationException(""))
	}
	if cursor.index >= cursor.source.Size() {
		panic(NewNoSuchElementException(""))
	}
	value := cursor.source.Get(cursor.index)
	cursor.index++
	return value
}

// Map-backed iteration retains next-record identity, not a snapshot of element
// values. Replacing a value remains visible; key changes fail at next().
type mapJavaIterator[K, V any] struct {
	source   *Map[K, V]
	records  []*mapRecord[K, V]
	index    int
	expected uint64
	keys     bool
}

func newMapJavaIterator[K, V any](source *Map[K, V], keys bool) JavaIterator {
	ReferenceRequireNonNull(source)
	return &mapJavaIterator[K, V]{source: source, records: append([]*mapRecord[K, V](nil), source.entries...), expected: source.modCount, keys: keys}
}
func (cursor *mapJavaIterator[K, V]) HasNextJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	return cursor.index < len(cursor.records)
}
func (cursor *mapJavaIterator[K, V]) NextJava2goExecution(execution *Execution) any {
	requireExecution(execution)
	// TreeMap's nextEntry checks exhaustion before structural revision;
	// HashMap's nextNode performs those checks in the opposite order.
	if cursor.source.sorted && cursor.index >= len(cursor.records) {
		panic(NewNoSuchElementException(""))
	}
	if cursor.expected != cursor.source.modCount {
		panic(NewConcurrentModificationException(""))
	}
	if cursor.index >= len(cursor.records) {
		panic(NewNoSuchElementException(""))
	}
	record := cursor.records[cursor.index]
	cursor.index++
	if cursor.keys {
		return record.entry.Key
	}
	return record.entry.Value
}
func (set *Set[T]) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(set)
	return newMapJavaIterator(set.backing, true)
}
func (view *mapValuesIterable[K, V]) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return newMapJavaIterator(view.source, false)
}
