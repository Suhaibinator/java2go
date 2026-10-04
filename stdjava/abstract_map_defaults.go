package stdjava

func MapEntrySetExecution(execution *Execution, source JavaMap) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	return source.MapEntrySetJava2goExecution(execution)
}

func MapSizeExecution(execution *Execution, source JavaMap) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapSizeJava2goExecution(*Execution) int32 }); ok {
		return callback.MapSizeJava2goExecution(execution)
	}
	panic(NewUnsupportedOperationException("map size callback is unavailable"))
}
func MapIsEmptyExecution(execution *Execution, source JavaMap) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapIsEmptyJava2goExecution(*Execution) bool }); ok {
		return callback.MapIsEmptyJava2goExecution(execution)
	}
	panic(NewUnsupportedOperationException("map isEmpty callback is unavailable"))
}
func MapContainsKeyExecution(execution *Execution, source JavaMap, key any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapContainsKeyJava2goExecution(*Execution, any) bool }); ok {
		return callback.MapContainsKeyJava2goExecution(execution, key)
	}
	panic(NewUnsupportedOperationException("map containsKey callback is unavailable"))
}
func MapContainsValueExecution(execution *Execution, source JavaMap, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapContainsValueJava2goExecution(*Execution, any) bool }); ok {
		return callback.MapContainsValueJava2goExecution(execution, value)
	}
	panic(NewUnsupportedOperationException("map containsValue callback is unavailable"))
}
func MapGetExecution(execution *Execution, source JavaMap, key any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapGetJava2goExecution(*Execution, any) any }); ok {
		return callback.MapGetJava2goExecution(execution, key)
	}
	panic(NewUnsupportedOperationException("map get callback is unavailable"))
}
func MapRemoveExecution(execution *Execution, source JavaMap, key any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapRemoveJava2goExecution(*Execution, any) any }); ok {
		return callback.MapRemoveJava2goExecution(execution, key)
	}
	panic(NewUnsupportedOperationException("map remove callback is unavailable"))
}
func MapPutExecution(execution *Execution, source any, key, value any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	// Native maps retain their existing allocation and erased key/value storage.
	// The private protocol cannot be acquired by a similarly named source method.
	if native, ok := source.(interface {
		mapPutObjectExecution(*Execution, any, any) any
	}); ok {
		return native.mapPutObjectExecution(execution, key, value)
	}
	reference := MapReferenceView(source)
	if callback, ok := reference.(interface {
		MapPutJava2goExecution(*Execution, any, any) any
	}); ok {
		return callback.MapPutJava2goExecution(execution, key, value)
	}
	panic(NewUnsupportedOperationException("map put callback is unavailable"))
}
func MapClearExecution(execution *Execution, source JavaMap) {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapClearJava2goExecution(*Execution) }); ok {
		callback.MapClearJava2goExecution(execution)
		return
	}
	panic(NewUnsupportedOperationException("map clear callback is unavailable"))
}
func MapKeySetExecution(execution *Execution, source JavaMap) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapKeySetJava2goExecution(*Execution) JavaIterable }); ok {
		return callback.MapKeySetJava2goExecution(execution)
	}
	panic(NewUnsupportedOperationException("map keySet callback is unavailable"))
}
func MapValuesExecution(execution *Execution, source JavaMap) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	source = MapReferenceView(source)
	if callback, ok := source.(interface{ MapValuesJava2goExecution(*Execution) JavaIterable }); ok {
		return callback.MapValuesJava2goExecution(execution)
	}
	panic(NewUnsupportedOperationException("map values callback is unavailable"))
}

func abstractMapIterator(execution *Execution, source JavaMap) JavaIterator {
	return IterableIteratorExecution(execution, MapEntrySetExecution(execution, source))
}
func abstractMapNextEntry(execution *Execution, cursor JavaIterator) JavaMapEntry {
	return ObjectView[JavaMapEntry](IteratorNextExecution(execution, cursor), JavaMapEntryTypeID)
}
func AbstractMapSizeExecution(execution *Execution, source JavaMap) int32 {
	return CollectionSizeExecution(execution, MapEntrySetExecution(execution, source))
}
func AbstractMapIsEmptyExecution(execution *Execution, source JavaMap) bool {
	return MapSizeExecution(execution, source) == 0
}
func AbstractMapContainsKeyExecution(execution *Execution, source JavaMap, key any) bool {
	cursor := abstractMapIterator(execution, source)
	for IteratorHasNextExecution(execution, cursor) {
		candidate := MapEntryGetKeyExecution(execution, abstractMapNextEntry(execution, cursor))
		if javaReferenceIsNull(key) {
			if javaReferenceIsNull(candidate) {
				return true
			}
		} else if ObjectEqualsExecution(execution, key, candidate) {
			return true
		}
	}
	return false
}
func AbstractMapContainsValueExecution(execution *Execution, source JavaMap, value any) bool {
	cursor := abstractMapIterator(execution, source)
	for IteratorHasNextExecution(execution, cursor) {
		candidate := MapEntryGetValueExecution(execution, abstractMapNextEntry(execution, cursor))
		if javaReferenceIsNull(value) {
			if javaReferenceIsNull(candidate) {
				return true
			}
		} else if ObjectEqualsExecution(execution, value, candidate) {
			return true
		}
	}
	return false
}
func AbstractMapGetExecution(execution *Execution, source JavaMap, key any) any {
	cursor := abstractMapIterator(execution, source)
	for IteratorHasNextExecution(execution, cursor) {
		entry := abstractMapNextEntry(execution, cursor)
		candidate := MapEntryGetKeyExecution(execution, entry)
		matched := javaReferenceIsNull(candidate)
		if !javaReferenceIsNull(key) {
			matched = ObjectEqualsExecution(execution, key, candidate)
		}
		if matched {
			return MapEntryGetValueExecution(execution, entry)
		}
	}
	return nil
}
func AbstractMapRemoveExecution(execution *Execution, source JavaMap, key any) any {
	cursor := abstractMapIterator(execution, source)
	for IteratorHasNextExecution(execution, cursor) {
		entry := abstractMapNextEntry(execution, cursor)
		candidate := MapEntryGetKeyExecution(execution, entry)
		matched := javaReferenceIsNull(candidate)
		if !javaReferenceIsNull(key) {
			matched = ObjectEqualsExecution(execution, key, candidate)
		}
		if matched {
			old := MapEntryGetValueExecution(execution, entry)
			IteratorRemoveExecution(execution, cursor)
			return old
		}
	}
	return nil
}
func AbstractMapPutExecution(execution *Execution, source JavaMap, key, value any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	panic(NewUnsupportedOperationException(""))
}
func AbstractMapClearExecution(execution *Execution, source JavaMap) {
	CollectionClearExecution(execution, MapEntrySetExecution(execution, source))
}
func AbstractMapEqualsExecution(execution *Execution, source JavaMap, other any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(source)
	if JavaReferenceEqual(source, other) {
		return true
	}
	if !ObjectInstanceOf(other, MapTypeID) {
		return false
	}
	right := MapReferenceView(other)
	if MapSizeExecution(execution, right) != MapSizeExecution(execution, source) {
		return false
	}
	return func() (equal bool) {
		defer collectionEqualityFailure(&equal)
		cursor := abstractMapIterator(execution, source)
		for IteratorHasNextExecution(execution, cursor) {
			entry := abstractMapNextEntry(execution, cursor)
			key := MapEntryGetKeyExecution(execution, entry)
			value := MapEntryGetValueExecution(execution, entry)
			if javaReferenceIsNull(value) {
				if !javaReferenceIsNull(MapGetExecution(execution, right, key)) || !MapContainsKeyExecution(execution, right, key) {
					return false
				}
			} else if !ObjectEqualsExecution(execution, value, MapGetExecution(execution, right, key)) {
				return false
			}
		}
		return true
	}()
}
func AbstractMapHashCodeExecution(execution *Execution, source JavaMap) int32 {
	cursor := abstractMapIterator(execution, source)
	var hash int32
	for IteratorHasNextExecution(execution, cursor) {
		hash += ObjectHashCodeExecution(execution, abstractMapNextEntry(execution, cursor))
	}
	return hash
}
