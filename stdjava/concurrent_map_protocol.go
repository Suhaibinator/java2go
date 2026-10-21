package stdjava

type JavaConcurrentMap interface {
	JavaMap
	MapPutIfAbsentJava2goExecution(*Execution, any, any) any
}

const ConcurrentMapTypeID TypeID = "java.util.concurrent.ConcurrentMap"
const ConcurrentHashMapTypeID TypeID = "java.util.concurrent.ConcurrentHashMap"

func init() {
	RegisterJavaType(ConcurrentMapTypeID, ObjectTypeID, MapTypeID)
	RegisterJavaType(ConcurrentHashMapTypeID, AbstractMapTypeID, ConcurrentMapTypeID)
}
func ConcurrentMapReferenceView(value any) JavaConcurrentMap {
	return ObjectView[JavaConcurrentMap](collectionObjectView(value), ConcurrentMapTypeID)
}
func (*ConcurrentHashMap[K, V]) nativeJavaInterfaces() []TypeID {
	return []TypeID{MapTypeID, AbstractMapTypeID, ConcurrentMapTypeID}
}
func (*ConcurrentHashMap[K, V]) JavaDynamicTypeID() TypeID { return ConcurrentHashMapTypeID }
func (c *ConcurrentHashMap[K, V]) mapPutObjectExecution(e *Execution, key, value any) any {
	return c.PutObject(key, value, e)
}
func (c *ConcurrentHashMap[K, V]) MapEntrySetJava2goExecution(e *Execution) JavaIterable {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	return c.view(concurrentEntries)
}
func (c *ConcurrentHashMap[K, V]) MapKeySetJava2goExecution(e *Execution) JavaIterable {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	return c.view(concurrentKeys)
}
func (c *ConcurrentHashMap[K, V]) MapValuesJava2goExecution(e *Execution) JavaIterable {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	return c.view(concurrentValues)
}
func (c *ConcurrentHashMap[K, V]) MapSizeJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	return c.Size()
}
func (c *ConcurrentHashMap[K, V]) MapIsEmptyJava2goExecution(e *Execution) bool {
	requireExecution(e)
	return c.IsEmpty()
}
func (c *ConcurrentHashMap[K, V]) MapContainsKeyJava2goExecution(e *Execution, key any) bool {
	requireExecution(e)
	return c.ContainsKey(key, e)
}
func (c *ConcurrentHashMap[K, V]) MapContainsValueJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	return c.ContainsValue(value, e)
}
func (c *ConcurrentHashMap[K, V]) MapGetJava2goExecution(e *Execution, key any) any {
	requireExecution(e)
	return c.GetObject(key, e)
}
func (c *ConcurrentHashMap[K, V]) MapPutJava2goExecution(e *Execution, key, value any) any {
	requireExecution(e)
	return c.PutObject(key, value, e)
}
func (c *ConcurrentHashMap[K, V]) MapRemoveJava2goExecution(e *Execution, key any) any {
	requireExecution(e)
	return c.RemoveObject(key, e)
}
func (c *ConcurrentHashMap[K, V]) MapClearJava2goExecution(e *Execution) {
	requireExecution(e)
	c.Clear()
}
func (c *ConcurrentHashMap[K, V]) MapPutAllJava2goExecution(e *Execution, input any) {
	requireExecution(e)
	c.PutAll(input, e)
}
func (c *ConcurrentHashMap[K, V]) MapPutIfAbsentJava2goExecution(e *Execution, key, value any) any {
	requireExecution(e)
	return c.PutIfAbsentObject(key, value, e)
}
func (c *ConcurrentHashMap[K, V]) MapGetOrDefaultJava2goExecution(e *Execution, key, value any) any {
	requireExecution(e)
	return c.GetOrDefaultObject(key, value, e)
}
func (c *ConcurrentHashMap[K, V]) MapComputeIfAbsentJava2goExecution(e *Execution, key, mapping any, apply func(*Execution, any) any) any {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	panic(NewUnsupportedOperationException("ConcurrentHashMap.computeIfAbsent atomic runtime protocol is unavailable"))
}

func (c *ConcurrentHashMap[K, V]) StringJava2goExecution(e *Execution) *JavaString {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	cursor := c.view(concurrentEntries).IteratorJava2goExecution(e)
	units := []uint16{'{'}
	for IteratorHasNextExecution(e, cursor) {
		entry := IteratorNextExecution(e, cursor).(JavaMapEntry)
		key := MapEntryGetKeyExecution(e, entry)
		value := MapEntryGetValueExecution(e, entry)
		for i, element := range []any{key, value} {
			if i != 0 {
				units = append(units, '=')
			}
			if JavaReferenceEqual(element, c) {
				units = append(units, '(', 't', 'h', 'i', 's', ' ', 'M', 'a', 'p', ')')
			} else {
				units = append(units, JavaStringTextOperandExecution(e, element).units...)
			}
		}
		if IteratorHasNextExecution(e, cursor) {
			units = append(units, ',', ' ')
		}
	}
	units = append(units, '}')
	return &JavaString{units: units}
}
