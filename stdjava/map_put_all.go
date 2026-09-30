package stdjava

// The entry cursor is erased only at the collection boundary, allowing Java's
// Map<? extends K,? extends V> input without requiring identical Go instantiations.
type mapPutAllSource interface {
	Size() int32
	javaMapVisitEntries(func(any, any))
	javaMapNaturalSorted() bool
}

func (m *Map[K, V]) javaMapNaturalSorted() bool { return m.sorted && m.comparator == nil }
func (m *Map[K, V]) javaMapVisitEntries(visit func(any, any)) {
	// Preserve iterator next-record identity while reading values at next().
	// Replacing a value is non-structural; changing keys invalidates the next step.
	records := append([]*mapRecord[K, V](nil), m.entries...)
	version := m.modCount
	for _, record := range records {
		if m.modCount != version {
			panic(NewConcurrentModificationException(""))
		}
		visit(record.key, record.value)
	}
}
func (m *Map[K, V]) PutAll(source any, execution ...*Execution) {
	ReferenceRequireNonNull(m)
	ReferenceRequireNonNull(source)
	input, ok := source.(mapPutAllSource)
	if !ok {
		panic(NewClassCastException("Map.putAll source lacks map entry iteration"))
	}
	size := input.Size()
	if size == 0 {
		return
	}
	exec := optionalComparisonExecution(execution)
	// TreeMap's natural-order sorted-source fast path does not compare keys.
	// Copy records, not entry pointers: subsequent source/target writes are independent.
	if m.sorted && m.comparator == nil && len(m.entries) == 0 && input.javaMapNaturalSorted() {
		m.modCount++
		input.javaMapVisitEntries(func(key, value any) {
			m.entries = append(m.entries, &mapRecord[K, V]{key: key, value: value})
		})
		return
	}
	input.javaMapVisitEntries(func(key, value any) { m.putObject(key, value, false, exec) })
}
