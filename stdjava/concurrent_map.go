package stdjava

import "sync"

type concurrentMapRecord struct {
	key   any
	value any
}
type concurrentMapCore struct {
	mu      sync.RWMutex
	buckets map[int32][]*concurrentMapRecord
	version uint64
	size    int32
	views   [3]*concurrentMapView
}

func (c *concurrentMapCore) snapshot(hash int32) ([]*concurrentMapRecord, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buckets[hash], c.version
}
func concurrentEntryIndex(entries []*concurrentMapRecord, key any, e *Execution) int {
	for i, r := range entries {
		if ObjectsEqual(key, r.key, e) {
			return i
		}
	}
	return -1
}
func (c *concurrentMapCore) putObject(key, value any, absent bool, e *Execution) any {
	ReferenceRequireNonNull(key)
	ReferenceRequireNonNull(value)
	hash := ObjectsHashCode(key, e)
	for {
		records, version := c.snapshot(hash)
		index := concurrentEntryIndex(records, key, e)
		c.mu.Lock()
		if c.version != version {
			c.mu.Unlock()
			continue
		}
		var prior any
		if index >= 0 {
			r := records[index]
			prior = r.value
			if !absent {
				r.value = value
				c.version++
			}
		} else {
			if c.buckets == nil {
				c.buckets = make(map[int32][]*concurrentMapRecord)
			}
			updated := append([]*concurrentMapRecord(nil), records...)
			updated = append(updated, &concurrentMapRecord{key: key, value: value})
			c.buckets[hash] = updated
			c.size++
			c.version++
		}
		c.mu.Unlock()
		return prior
	}
}
func (c *concurrentMapCore) getObject(key any, e *Execution) (any, bool) {
	ReferenceRequireNonNull(key)
	records, _ := c.snapshot(ObjectsHashCode(key, e))
	index := concurrentEntryIndex(records, key, e)
	if index < 0 {
		return nil, false
	}
	c.mu.RLock()
	value := records[index].value
	c.mu.RUnlock()
	return value, true
}
func (c *concurrentMapCore) removeObject(key, value any, conditional bool, e *Execution) (any, bool) {
	ReferenceRequireNonNull(key)
	hash := ObjectsHashCode(key, e)
	for {
		records, version := c.snapshot(hash)
		index := concurrentEntryIndex(records, key, e)
		if index < 0 {
			return nil, false
		}
		c.mu.RLock()
		prior := records[index].value
		c.mu.RUnlock()
		if conditional && !ObjectsEqual(value, prior, e) {
			return nil, false
		}
		c.mu.Lock()
		if c.version != version {
			c.mu.Unlock()
			continue
		}
		updated := make([]*concurrentMapRecord, 0, len(records)-1)
		updated = append(updated, records[:index]...)
		updated = append(updated, records[index+1:]...)
		if len(updated) == 0 {
			delete(c.buckets, hash)
		} else {
			c.buckets[hash] = updated
		}
		c.version++
		c.size--
		c.mu.Unlock()
		return prior, true
	}
}
func (c *concurrentMapCore) entrySnapshot() []MapEntry[any, any] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]MapEntry[any, any], 0, c.size)
	for _, bucket := range c.buckets {
		for _, r := range bucket {
			out = append(out, MapEntry[any, any]{Key: r.key, Value: r.value})
		}
	}
	return out
}
func (c *concurrentMapCore) recordSnapshot() []*concurrentMapRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]*concurrentMapRecord, 0, c.size)
	for _, bucket := range c.buckets {
		out = append(out, bucket...)
	}
	return out
}
func (c *ConcurrentHashMap[K, V]) PutObject(key, value any, execution ...*Execution) any {
	ReferenceRequireNonNull(c)
	return c.putObject(key, value, false, optionalComparisonExecution(execution))
}
func (c *ConcurrentHashMap[K, V]) PutIfAbsentObject(key, value any, execution ...*Execution) any {
	ReferenceRequireNonNull(c)
	return c.putObject(key, value, true, optionalComparisonExecution(execution))
}
func (c *ConcurrentHashMap[K, V]) GetObject(key any, execution ...*Execution) any {
	ReferenceRequireNonNull(c)
	value, _ := c.getObject(key, optionalComparisonExecution(execution))
	return value
}
func (c *ConcurrentHashMap[K, V]) GetOrDefaultObject(key, fallback any, execution ...*Execution) any {
	ReferenceRequireNonNull(c)
	value, present := c.getObject(key, optionalComparisonExecution(execution))
	if !present {
		return fallback
	}
	return value
}
func (c *ConcurrentHashMap[K, V]) RemoveObject(key any, execution ...*Execution) any {
	ReferenceRequireNonNull(c)
	value, _ := c.removeObject(key, nil, false, optionalComparisonExecution(execution))
	return value
}
func (c *ConcurrentHashMap[K, V]) ContainsKey(key any, execution ...*Execution) bool {
	ReferenceRequireNonNull(c)
	_, present := c.getObject(key, optionalComparisonExecution(execution))
	return present
}
func (c *ConcurrentHashMap[K, V]) ContainsValue(value any, execution ...*Execution) bool {
	ReferenceRequireNonNull(c)
	ReferenceRequireNonNull(value)
	e := optionalComparisonExecution(execution)
	for _, record := range c.recordSnapshot() {
		if ObjectsEqual(value, c.recordValue(record), e) {
			return true
		}
	}
	return false
}
func (c *ConcurrentHashMap[K, V]) Size() int32 {
	ReferenceRequireNonNull(c)
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.size
}
func (c *ConcurrentHashMap[K, V]) IsEmpty() bool { return c.Size() == 0 }
func (c *ConcurrentHashMap[K, V]) Clear() {
	ReferenceRequireNonNull(c)
	c.mu.Lock()
	c.buckets = make(map[int32][]*concurrentMapRecord)
	c.size = 0
	c.version++
	c.mu.Unlock()
}
func (c *ConcurrentHashMap[K, V]) PutAll(source any, execution ...*Execution) {
	ReferenceRequireNonNull(c)
	ReferenceRequireNonNull(source)
	e := optionalComparisonExecution(execution)
	if len(execution) == 0 {
		e = NewExecution()
	}
	input := MapReferenceView(source)
	MapSizeExecution(e, input)
	cursor := abstractMapIterator(e, input)
	for IteratorHasNextExecution(e, cursor) {
		entry := abstractMapNextEntry(e, cursor)
		key := MapEntryGetKeyExecution(e, entry)
		value := MapEntryGetValueExecution(e, entry)
		c.putObject(key, value, false, e)
	}
}

func (c *concurrentMapCore) recordValue(record *concurrentMapRecord) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return record.value
}
func (c *concurrentMapCore) hashCode(e *Execution) int32 {
	var hash int32
	for _, record := range c.recordSnapshot() {
		keyHash := ObjectsHashCode(record.key, e)
		value := c.recordValue(record)
		hash += keyHash ^ ObjectsHashCode(value, e)
	}
	return hash
}
