package stdjava

import "sync"

const (
	concurrentKeys = iota
	concurrentValues
	concurrentEntries
)

type concurrentMapView struct {
	source *concurrentMapCore
	kind   int
}

func (c *concurrentMapCore) view(kind int) *concurrentMapView {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.views[kind] == nil {
		c.views[kind] = &concurrentMapView{source: c, kind: kind}
	}
	return c.views[kind]
}
func (v *concurrentMapView) nativeJavaInterfaces() []TypeID {
	if v.kind == concurrentValues {
		return []TypeID{CollectionTypeID}
	}
	return []TypeID{SetTypeID}
}
func (v *concurrentMapView) IteratorJava2goExecution(e *Execution) JavaIterator {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	return &concurrentMapIterator{source: v.source, kind: v.kind, records: v.source.recordSnapshot()}
}
func (v *concurrentMapView) CollectionSizeJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	v.source.mu.RLock()
	defer v.source.mu.RUnlock()
	return v.source.size
}
func (v *concurrentMapView) CollectionIsEmptyJava2goExecution(e *Execution) bool {
	return v.CollectionSizeJava2goExecution(e) == 0
}
func (v *concurrentMapView) CollectionClearJava2goExecution(e *Execution) {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	c := v.source
	c.mu.Lock()
	c.buckets = make(map[int32][]*concurrentMapRecord)
	c.size = 0
	c.version++
	c.mu.Unlock()
}
func (v *concurrentMapView) CollectionContainsJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	switch v.kind {
	case concurrentKeys:
		_, present := v.source.getObject(value, e)
		return present
	case concurrentValues:
		ReferenceRequireNonNull(value)
		for _, record := range v.source.recordSnapshot() {
			if ObjectsEqual(value, v.source.recordValue(record), e) {
				return true
			}
		}
		return false
	default:
		entry, ok := liveEntryAccess(value)
		if !ok {
			return false
		}
		key := entry.key(e)
		if javaReferenceIsNull(key) {
			return false
		}
		actual, present := v.source.getObject(key, e)
		if !present {
			return false
		}
		wanted := entry.value(e)
		return !javaReferenceIsNull(wanted) && ObjectsEqual(wanted, actual, e)
	}
}
func (v *concurrentMapView) CollectionRemoveJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	switch v.kind {
	case concurrentKeys:
		_, removed := v.source.removeObject(value, nil, false, e)
		return removed
	case concurrentValues:
		if javaReferenceIsNull(value) {
			return false
		}
		cursor := v.IteratorJava2goExecution(e)
		for IteratorHasNextExecution(e, cursor) {
			if ObjectsEqual(value, IteratorNextExecution(e, cursor), e) {
				IteratorRemoveExecution(e, cursor)
				return true
			}
		}
		return false
	default:
		entry, ok := liveEntryAccess(value)
		if !ok {
			return false
		}
		key := entry.key(e)
		if javaReferenceIsNull(key) {
			return false
		}
		wanted := entry.value(e)
		if javaReferenceIsNull(wanted) {
			return false
		}
		_, removed := v.source.removeObject(key, wanted, true, e)
		return removed
	}
}
func (v *concurrentMapView) CollectionAddJava2goExecution(e *Execution, value any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	if v.kind != concurrentEntries {
		panic(NewUnsupportedOperationException(""))
	}
	entry := ObjectView[JavaMapEntry](value, JavaMapEntryTypeID)
	key := MapEntryGetKeyExecution(e, entry)
	val := MapEntryGetValueExecution(e, entry)
	return v.source.putObject(key, val, false, e) == nil
}
func (v *concurrentMapView) CollectionContainsAllJava2goExecution(e *Execution, other JavaIterable) bool {
	return AbstractCollectionContainsAllExecution(e, v, other)
}
func (v *concurrentMapView) EqualsJava2goExecution(e *Execution, other any) bool {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	if JavaReferenceEqual(v, other) {
		return true
	}
	if v.kind == concurrentValues || !ObjectInstanceOf(other, SetTypeID) {
		return false
	}
	right := ObjectView[JavaIterable](other, SetTypeID)
	return CollectionContainsAllExecution(e, v, right) && CollectionContainsAllExecution(e, right, v)
}
func (v *concurrentMapView) HashCodeJava2goExecution(e *Execution) int32 {
	requireExecution(e)
	ReferenceRequireNonNull(v)
	if v.kind == concurrentValues {
		return SystemIdentityHashCode(v)
	}
	if v.kind == concurrentEntries {
		return v.source.hashCode(e)
	}
	return AbstractSetHashCodeExecution(e, v)
}

type concurrentMapIterator struct {
	source  *concurrentMapCore
	kind    int
	records []*concurrentMapRecord
	index   int
	last    *concurrentMapRecord
}

func (*concurrentMapIterator) nativeJavaInterfaces() []TypeID { return []TypeID{IteratorTypeID} }
func (c *concurrentMapIterator) HasNextJava2goExecution(e *Execution) bool {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	return c.index < len(c.records)
}
func (c *concurrentMapIterator) NextJava2goExecution(e *Execution) any {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	if c.index >= len(c.records) {
		panic(NewNoSuchElementException(""))
	}
	r := c.records[c.index]
	c.records[c.index] = nil
	c.index++
	if c.index == len(c.records) {
		c.records = nil
		c.index = 0
	}
	c.last = r
	if c.kind == concurrentKeys {
		return r.key
	}
	c.source.mu.RLock()
	value := r.value
	c.source.mu.RUnlock()
	if c.kind == concurrentValues {
		return value
	}
	return &concurrentMapEntry{source: c.source, key: r.key, value: value}
}
func (c *concurrentMapIterator) IteratorRemoveJava2goExecution(e *Execution) {
	requireExecution(e)
	ReferenceRequireNonNull(c)
	if c.last == nil {
		panic(NewIllegalStateException(""))
	}
	key := c.last.key
	c.last = nil
	c.source.removeObject(key, nil, false, e)
}

type concurrentMapEntry struct {
	source *concurrentMapCore
	key    any
	mu     sync.Mutex
	value  any
}

func (*concurrentMapEntry) nativeJavaInterfaces() []TypeID { return []TypeID{JavaMapEntryTypeID} }
func (e *concurrentMapEntry) GetKeyJava2goExecution(caller *Execution) any {
	requireExecution(caller)
	ReferenceRequireNonNull(e)
	return e.key
}
func (e *concurrentMapEntry) GetValueJava2goExecution(caller *Execution) any {
	requireExecution(caller)
	ReferenceRequireNonNull(e)
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.value
}
func (e *concurrentMapEntry) SetValueJava2goExecution(caller *Execution, value any) any {
	requireExecution(caller)
	ReferenceRequireNonNull(e)
	ReferenceRequireNonNull(value)
	e.mu.Lock()
	prior := e.value
	e.value = value
	e.mu.Unlock()
	e.source.putObject(e.key, value, false, caller)
	return prior
}
func (e *concurrentMapEntry) EqualsJava2goExecution(caller *Execution, other any) bool {
	entry, ok := liveEntryAccess(other)
	return ok && ObjectsEqual(e.key, entry.key(caller), caller) && ObjectsEqual(e.GetValueJava2goExecution(caller), entry.value(caller), caller)
}
func (e *concurrentMapEntry) HashCodeJava2goExecution(caller *Execution) int32 {
	return ObjectsHashCode(e.key, caller) ^ ObjectsHashCode(e.GetValueJava2goExecution(caller), caller)
}
