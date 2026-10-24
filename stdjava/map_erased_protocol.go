package stdjava

// The shared HashMap/LinkedHashMap/TreeMap backend proves their common map
// ancestry without inventing a concrete class descriptor or copying storage.
func (*Map[K, V]) nativeJavaInterfaces() []TypeID {
	return []TypeID{MapTypeID, AbstractMapTypeID}
}

func (m *Map[K, V]) MapEntrySetJava2goExecution(execution *Execution) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return MapEntriesView(m)
}

func (m *Map[K, V]) MapSizeJava2goExecution(execution *Execution) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.Size()
}

func (m *Map[K, V]) MapIsEmptyJava2goExecution(execution *Execution) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.IsEmpty()
}

func (m *Map[K, V]) MapContainsKeyJava2goExecution(execution *Execution, key any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.ContainsKey(key, execution)
}

func (m *Map[K, V]) MapContainsValueJava2goExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.ContainsValue(value, execution)
}

func (m *Map[K, V]) MapGetJava2goExecution(execution *Execution, key any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.GetObject(key, execution)
}

func (m *Map[K, V]) MapRemoveJava2goExecution(execution *Execution, key any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.RemoveObject(key, execution)
}

func (m *Map[K, V]) MapPutJava2goExecution(execution *Execution, key, value any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return m.PutObject(key, value, execution)
}

func (m *Map[K, V]) MapClearJava2goExecution(execution *Execution) {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	m.Clear()
}

func (m *Map[K, V]) MapKeySetJava2goExecution(execution *Execution) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return MapKeysView(m)
}

func (m *Map[K, V]) MapValuesJava2goExecution(execution *Execution) JavaIterable {
	requireExecution(execution)
	ReferenceRequireNonNull(m)
	return MapValuesView(m)
}
