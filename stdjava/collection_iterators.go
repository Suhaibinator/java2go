package stdjava

// List iteration observes current slots rather than a Slice snapshot. Like
// ArrayList.Itr, hasNext compares the live size and next checks structural
// revision before reading; replacement via set remains visible and nonstructural.
type listJavaIterator[T any] struct {
	source    *List[T]
	index     int32
	expected  uint64
	last      int32
	canRemove bool
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
	value := cursor.source.rawGet(cursor.index)
	cursor.last = cursor.index
	cursor.canRemove = true
	cursor.index++
	return value
}

func (cursor *listJavaIterator[T]) IteratorRemoveJava2goExecution(execution *Execution) {
	requireExecution(execution)
	if !cursor.canRemove {
		panic(NewIllegalStateException(""))
	}
	if cursor.expected != cursor.source.modCount {
		panic(NewConcurrentModificationException(""))
	}
	cursor.source.rawRemove(cursor.last)
	cursor.index = cursor.last
	cursor.canRemove = false
	cursor.expected = cursor.source.modCount
}

// Map-backed iteration retains next-record identity, not a snapshot of element
// values. Replacing a value remains visible; key changes fail at next().
type mapJavaIterator[K, V any] struct {
	source   *Map[K, V]
	records  []*mapRecord[K, V]
	index    int
	expected uint64
	keys     bool
	entries  bool
	last     *mapRecord[K, V]
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
	cursor.last = record
	if cursor.entries {
		return record
	}
	if cursor.keys {
		return record.key
	}
	return record.value
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

func (cursor *mapJavaIterator[K, V]) IteratorRemoveJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(cursor)
	if cursor.last == nil {
		panic(NewIllegalStateException(""))
	}
	if cursor.expected != cursor.source.modCount {
		panic(NewConcurrentModificationException(""))
	}
	cursor.source.removeRecord(cursor.last)
	cursor.expected = cursor.source.modCount
	cursor.last = nil
}
func (cursor *mapJavaIterator[K, V]) nativeJavaInterfaces() []TypeID { return []TypeID{IteratorTypeID} }
