package stdjava

func MapPutAllExecution(e *Execution, source JavaMap, input any) {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapPutAllJava2goExecution(*Execution, any) }); ok {
		callback.MapPutAllJava2goExecution(e, input)
		return
	}
	panic(NewUnsupportedOperationException("map putAll callback is unavailable"))
}
func MapPutIfAbsentExecution(e *Execution, source JavaMap, key, value any) any {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface {
		MapPutIfAbsentJava2goExecution(*Execution, any, any) any
	}); ok {
		return callback.MapPutIfAbsentJava2goExecution(e, key, value)
	}
	panic(NewUnsupportedOperationException("map putIfAbsent callback is unavailable"))
}
func MapGetOrDefaultExecution(e *Execution, source JavaMap, key, value any) any {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface {
		MapGetOrDefaultJava2goExecution(*Execution, any, any) any
	}); ok {
		return callback.MapGetOrDefaultJava2goExecution(e, key, value)
	}
	panic(NewUnsupportedOperationException("map getOrDefault callback is unavailable"))
}
func MapComputeIfAbsentExecution(e *Execution, source JavaMap, key, mapping any, apply func(*Execution, any) any) any {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface {
		MapComputeIfAbsentJava2goExecution(*Execution, any, any, func(*Execution, any) any) any
	}); ok {
		return callback.MapComputeIfAbsentJava2goExecution(e, key, mapping, apply)
	}
	panic(NewUnsupportedOperationException("map computeIfAbsent callback is unavailable"))
}
func MapDefaultPutIfAbsentExecution(e *Execution, source JavaMap, key, value any) any {
	prior := MapGetExecution(e, source, key)
	if javaReferenceIsNull(prior) {
		return MapPutExecution(e, source, key, value)
	}
	return prior
}
func MapDefaultGetOrDefaultExecution(e *Execution, source JavaMap, key, value any) any {
	prior := MapGetExecution(e, source, key)
	if !javaReferenceIsNull(prior) || MapContainsKeyExecution(e, source, key) {
		return prior
	}
	return value
}
func ConcurrentMapDefaultGetOrDefaultExecution(e *Execution, source JavaMap, key, value any) any {
	prior := MapGetExecution(e, source, key)
	if !javaReferenceIsNull(prior) {
		return prior
	}
	return value
}
func MapDefaultComputeIfAbsentExecution(e *Execution, source JavaMap, key, mapping any, apply func(*Execution, any) any) any {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	ReferenceRequireNonNull(mapping)
	ReferenceRequireNonNull(apply)
	prior := MapGetExecution(e, source, key)
	if !javaReferenceIsNull(prior) {
		return prior
	}
	value := apply(e, key)
	if !javaReferenceIsNull(value) {
		MapPutExecution(e, source, key, value)
	}
	return value
}
func (m *Map[K, V]) MapPutAllJava2goExecution(e *Execution, input any) {
	requireExecution(e)
	m.PutAll(input, e)
}
func (m *Map[K, V]) MapPutIfAbsentJava2goExecution(e *Execution, key, value any) any {
	requireExecution(e)
	ReferenceRequireNonNull(m)
	return m.PutIfAbsentObject(key, value, e)
}
func (m *Map[K, V]) MapGetOrDefaultJava2goExecution(e *Execution, key, value any) any {
	requireExecution(e)
	ReferenceRequireNonNull(m)
	return m.GetOrDefaultObject(key, value, e)
}
func (m *Map[K, V]) MapComputeIfAbsentJava2goExecution(e *Execution, key, mapping any, apply func(*Execution, any) any) any {
	requireExecution(e)
	ReferenceRequireNonNull(mapping)
	ReferenceRequireNonNull(apply)
	return m.ComputeIfAbsentObject(key, func(key any) any { return apply(e, key) }, e)
}

// ComputeIfAbsentObject uses the existing native Map algorithm with erased slots.
// Raw callback results are stored before any typed consumer checkcast.
func (m *Map[K, V]) ComputeIfAbsentObject(key any, mapping func(any) any, execution ...*Execution) any {
	ReferenceRequireNonNull(m)
	ReferenceRequireNonNull(mapping)
	e := optionalComparisonExecution(execution)
	var record *mapRecord[K, V]
	var hash int32
	var index int
	if !m.sorted || len(m.entries) != 0 {
		record, hash, index = m.find(key, e)
	}
	if record != nil && !javaReferenceIsNull(record.value) {
		return record.value
	}
	before := m.modCount
	computed := mapping(key)
	if m.modCount != before {
		panic(NewConcurrentModificationException("mapping function modified map"))
	}
	if record != nil && m.sorted {
		record.value = computed
		return computed
	}
	if javaReferenceIsNull(computed) {
		return computed
	}
	if record != nil {
		record.value = computed
		return computed
	}
	m.insert(key, computed, hash, index, e)
	return computed
}

// AbstractMap's inherited putAll consumes entries without consulting size.
func AbstractMapPutAllExecution(e *Execution, source JavaMap, input any) {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	ReferenceRequireNonNull(input)
	cursor := abstractMapIterator(e, MapReferenceView(input))
	for IteratorHasNextExecution(e, cursor) {
		entry := abstractMapNextEntry(e, cursor)
		key := MapEntryGetKeyExecution(e, entry)
		value := MapEntryGetValueExecution(e, entry)
		MapPutExecution(e, source, key, value)
	}
}

// ConcurrentMap's interface default arbitrates a computed value through the
// implementation's atomic putIfAbsent. Native CHM has a distinct unsupported
// atomic-compute role and never inherits this helper accidentally.
func ConcurrentMapDefaultComputeIfAbsentExecution(e *Execution, source JavaMap, key, mapping any, apply func(*Execution, any) any) any {
	requireExecution(e)
	ReferenceRequireNonNull(source)
	ReferenceRequireNonNull(mapping)
	ReferenceRequireNonNull(apply)
	ConcurrentMapReferenceView(source)
	prior := MapGetExecution(e, source, key)
	if !javaReferenceIsNull(prior) {
		return prior
	}
	value := apply(e, key)
	if javaReferenceIsNull(value) {
		return nil
	}
	winner := MapPutIfAbsentExecution(e, source, key, value)
	if !javaReferenceIsNull(winner) {
		return winner
	}
	return value
}
