package stdjava

// A view is cached by the map, while every iterator owns independent position
// and modification state. Slice copies references to entries, never the entries.
type mapEntriesIterable[K, V any] struct{ source *Map[K, V] }

func MapEntriesView[K, V any](source *Map[K, V]) *mapEntriesIterable[K, V] {
	ReferenceRequireNonNull(source)
	if source.entryView == nil {
		source.entryView = &mapEntriesIterable[K, V]{source: source}
	}
	return source.entryView.(*mapEntriesIterable[K, V])
}
func (view *mapEntriesIterable[K, V]) Slice() []JavaMapEntry {
	out := make([]JavaMapEntry, len(view.source.entries))
	for i, record := range view.source.entries {
		out[i] = record
	}
	return out
}
func (view *mapEntriesIterable[K, V]) IteratorJava2goExecution(execution *Execution) JavaIterator {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return &mapJavaIterator[K, V]{source: view.source, records: append([]*mapRecord[K, V](nil), view.source.entries...), expected: view.source.modCount, entries: true}
}
func (view *mapEntriesIterable[K, V]) Size() int32   { return view.source.Size() }
func (view *mapEntriesIterable[K, V]) IsEmpty() bool { return view.source.IsEmpty() }
func (view *mapEntriesIterable[K, V]) Clear()        { view.source.Clear() }
func (view *mapEntriesIterable[K, V]) Contains(value any, execution ...*Execution) bool {
	exec := optionalComparisonExecution(execution)
	if view.source.sorted {
		// Keep the existing sorted-map path independent of HashMap's Node.equals.
		key, candidateValue, ok := liveEntryElements(exec, value)
		if !ok {
			return false
		}
		record, _, _ := view.source.find(key, exec)
		return record != nil && ObjectsEqual(record.value, candidateValue, exec)
	}
	entry, ok := liveEntryAccess(value)
	if !ok {
		return false
	}
	record, _, _ := view.source.find(entry.key(exec), exec)
	// HashMap.EntrySet.contains invokes the stored node's equality after lookup.
	// That comparison obtains the candidate key again and short-circuits its value.
	return record != nil && record.EqualsJava2goExecution(exec, value)
}
func (view *mapEntriesIterable[K, V]) Remove(value any, execution ...*Execution) bool {
	exec := optionalComparisonExecution(execution)
	key, candidateValue, ok := liveEntryElements(exec, value)
	if !ok {
		return false
	}
	record, _, _ := view.source.find(key, exec)
	if record == nil {
		return false
	}
	var equal bool
	if view.source.sorted {
		equal = ObjectsEqual(record.value, candidateValue, exec)
	} else {
		// HashMap.removeNode compares the supplied value to the stored value before
		// unlinking the record. Exceptions leave both the map and its live views intact.
		equal = ObjectsEqual(candidateValue, record.value, exec)
	}
	if !equal {
		return false
	}
	view.source.removeRecord(record)
	return true
}
func (view *mapKeysIterable[K, V]) Size() int32   { return view.source.Size() }
func (view *mapKeysIterable[K, V]) IsEmpty() bool { return view.source.IsEmpty() }
func (view *mapKeysIterable[K, V]) Clear()        { view.source.Clear() }
func (view *mapKeysIterable[K, V]) Contains(key any, execution ...*Execution) bool {
	return view.source.ContainsKey(key, execution...)
}
func (view *mapKeysIterable[K, V]) Remove(key any, execution ...*Execution) bool {
	record, _, _ := view.source.find(key, optionalComparisonExecution(execution))
	if record == nil {
		return false
	}
	view.source.removeRecord(record)
	return true
}
func (view *mapValuesIterable[K, V]) IsEmpty() bool              { return view.source.IsEmpty() }
func (*mapKeysIterable[K, V]) nativeJavaInterfaces() []TypeID    { return []TypeID{nativeSetTypeID} }
func (*mapEntriesIterable[K, V]) nativeJavaInterfaces() []TypeID { return []TypeID{nativeSetTypeID} }
func (*mapValuesIterable[K, V]) nativeJavaInterfaces() []TypeID {
	return []TypeID{nativeCollectionTypeID}
}

func (view *mapKeysIterable[K, V]) CollectionSizeJava2goExecution(execution *Execution) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Size()
}

func (view *mapKeysIterable[K, V]) CollectionIsEmptyJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.IsEmpty()
}

func (view *mapKeysIterable[K, V]) CollectionClearJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	view.Clear()
}

func (view *mapKeysIterable[K, V]) CollectionContainsJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Contains(value, execution)
}

func (view *mapKeysIterable[K, V]) CollectionRemoveJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Remove(value, execution)
}

func (view *mapEntriesIterable[K, V]) CollectionSizeJava2goExecution(execution *Execution) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Size()
}

func (view *mapEntriesIterable[K, V]) CollectionIsEmptyJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.IsEmpty()
}

func (view *mapEntriesIterable[K, V]) CollectionClearJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	view.Clear()
}

func (view *mapEntriesIterable[K, V]) CollectionContainsJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Contains(value, execution)
}

func (view *mapEntriesIterable[K, V]) CollectionRemoveJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Remove(value, execution)
}

func (view *mapValuesIterable[K, V]) CollectionSizeJava2goExecution(execution *Execution) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Size()
}

func (view *mapValuesIterable[K, V]) CollectionIsEmptyJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.IsEmpty()
}

func (view *mapValuesIterable[K, V]) CollectionClearJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	view.Clear()
}

func (view *mapValuesIterable[K, V]) CollectionContainsJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.Contains(value, execution)
}

func (view *mapValuesIterable[K, V]) CollectionRemoveJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(view)
	return view.RemoveObject(value, execution)
}
