package stdjava

// The native record is itself the Java entry allocation. Retaining it keeps
// detached entries alive without looking up a later mapping under the same key.
func (record *mapRecord[K, V]) GetKeyJava2goExecution(execution *Execution) any {
	requireExecution(execution)
	ReferenceRequireNonNull(record)
	return record.key
}
func (record *mapRecord[K, V]) GetValueJava2goExecution(execution *Execution) any {
	requireExecution(execution)
	ReferenceRequireNonNull(record)
	return record.value
}
func (record *mapRecord[K, V]) SetValueJava2goExecution(execution *Execution, value any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(record)
	old := record.value
	record.value = value
	return old
}
func (record *mapRecord[K, V]) nativeJavaInterfaces() []TypeID {
	return []TypeID{JavaMapEntryTypeID}
}
func (record *mapRecord[K, V]) javaEntry() (any, any) { return record.key, record.value }
func (record *mapRecord[K, V]) EqualsJava2goExecution(execution *Execution, other any) bool {
	ReferenceRequireNonNull(record)
	entry, ok := liveEntryAccess(other)
	return ok && ObjectsEqual(record.key, entry.key(execution), execution) &&
		ObjectsEqual(record.value, entry.value(execution), execution)
}
func (record *mapRecord[K, V]) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(record)
	return ObjectsHashCode(record.key, execution) ^ ObjectsHashCode(record.value, execution)
}

// Entry getters may execute translated Java code. Retain the entry and fetch
// each component only where the calling JDK operation would evaluate it.
// Legacy internal entry values already contain their components and run no code.
type liveEntryAccessor struct {
	source JavaMapEntry
	legacy javaEntryValue
}

func liveEntryAccess(value any) (liveEntryAccessor, bool) {
	if javaReferenceIsNull(value) {
		return liveEntryAccessor{}, false
	}
	if entry, ok := value.(javaEntryValue); ok {
		return liveEntryAccessor{legacy: entry}, true
	}
	if entry, ok := value.(JavaMapEntry); ok {
		return liveEntryAccessor{source: entry}, true
	}
	return liveEntryAccessor{}, false
}
func (entry liveEntryAccessor) key(execution *Execution) any {
	if entry.source != nil {
		return MapEntryGetKeyExecution(execution, entry.source)
	}
	key, _ := entry.legacy.javaEntry()
	return key
}
func (entry liveEntryAccessor) value(execution *Execution) any {
	if entry.source != nil {
		return MapEntryGetValueExecution(execution, entry.source)
	}
	_, value := entry.legacy.javaEntry()
	return value
}
func liveEntryElements(execution *Execution, value any) (any, any, bool) {
	entry, ok := liveEntryAccess(value)
	if !ok {
		return nil, nil, false
	}
	return entry.key(execution), entry.value(execution), true
}

// Object-returning map boundaries keep checkcasts at the generated consumer.
func (m *Map[K, V]) GetObject(key any, execution ...*Execution) any {
	if record, _, _ := m.find(key, optionalComparisonExecution(execution)); record != nil {
		return record.value
	}
	return nil
}
func (m *Map[K, V]) GetOrDefaultObject(key any, fallback any, execution ...*Execution) any {
	if record, _, _ := m.find(key, optionalComparisonExecution(execution)); record != nil {
		return record.value
	}
	return fallback
}
func (m *Map[K, V]) PutObject(key any, value any, execution ...*Execution) any {
	return m.putObject(key, value, false, optionalComparisonExecution(execution))
}
func (m *Map[K, V]) PutIfAbsentObject(key any, value any, execution ...*Execution) any {
	return m.putObject(key, value, true, optionalComparisonExecution(execution))
}

// Native Map.put shares the erased operation boundary with source maps.
func (m *Map[K, V]) mapPutObjectExecution(execution *Execution, key, value any) any {
	return m.PutObject(key, value, execution)
}
