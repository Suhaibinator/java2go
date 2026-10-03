package stdjava

import "strings"

// MapEntry is the key/value pair exposed by Map.entrySet.
type MapEntry[K, V any] struct {
	Key   K
	Value V
}

func (entry MapEntry[K, V]) GetKey() K   { return entry.Key }
func (entry MapEntry[K, V]) GetValue() V { return entry.Value }

type mapRecord[K, V any] struct {
	key   any
	value any
}

// Map uses Java hashCode/equals collision buckets. Keys may have any generated
// Go representation; their Go comparability does not determine Java equality.
// Iteration is deterministic insertion order for hash maps. Tree maps keep
// entries ordered and determine key equivalence exclusively by comparison.
// Legacy slice methods are internal snapshots; Java-facing views retain this map.
type Map[K, V any] struct {
	buckets    map[int32][]*mapRecord[K, V]
	entries    []*mapRecord[K, V]
	sorted     bool
	comparator Comparator[K]
	modCount   uint64
	keyView    any
	valueView  any
	entryView  any
}

func NewMap[K, V any]() *Map[K, V] {
	return &Map[K, V]{buckets: make(map[int32][]*mapRecord[K, V])}
}

// NewTreeMap uses Comparable's natural order; NewTreeMapWith accepts an
// explicit Comparator. A nil comparator also selects natural ordering.
func NewTreeMap[K, V any]() *Map[K, V] { return NewTreeMapWith[K, V](nil) }
func NewTreeMapWith[K, V any](comparator Comparator[K]) *Map[K, V] {
	return &Map[K, V]{sorted: true, comparator: comparator}
}

func (m *Map[K, V]) compare(key any, stored any, execution *Execution) int32 {
	if m.comparator == nil {
		return javaCompareValuesExecution(execution, key, stored)
	}
	var typed K
	if javaReferenceIsNull(key) {
		typed = collectionZero[K]()
	} else {
		var ok bool
		typed, ok = key.(K)
		if !ok {
			panic(NewClassCastException("incompatible sorted collection key"))
		}
	}
	return m.comparator(typed, collectionElementView[K](stored))
}

func (m *Map[K, V]) find(key any, execution *Execution) (*mapRecord[K, V], int32, int) {
	if m.sorted {
		if m.comparator == nil {
			ReferenceRequireNonNull(key)
		}
		low, high := 0, len(m.entries)
		for low < high {
			mid := low + (high-low)/2
			comparison := m.compare(key, m.entries[mid].key, execution)
			if comparison == 0 {
				return m.entries[mid], 0, mid
			}
			if comparison < 0 {
				high = mid
			} else {
				low = mid + 1
			}
		}
		return nil, 0, low
	}
	hash := ObjectsHashCode(key, execution)
	for _, record := range m.buckets[hash] {
		if ObjectsEqual(key, record.key, execution) {
			return record, hash, 0
		}
	}
	return nil, hash, 0
}

func (m *Map[K, V]) Put(key K, value V, execution ...*Execution) V {
	return collectionElementView[V](m.putObject(key, value, false, optionalComparisonExecution(execution)))
}

// PutIfAbsent replaces both a missing mapping and a mapping whose Java value
// is null. Reuse the located record so key identity and iteration order remain
// unchanged, and user-defined hashCode/equals run only once per lookup.
func (m *Map[K, V]) PutIfAbsent(key K, value V, execution ...*Execution) V {
	return collectionElementView[V](m.putObject(key, value, true, optionalComparisonExecution(execution)))
}

func (m *Map[K, V]) putObject(key any, value any, onlyAbsent bool, exec *Execution) any {
	record, hash, index := m.find(key, exec)
	if record != nil {
		old := record.value
		if !onlyAbsent || javaReferenceIsNull(old) {
			record.value = value
		}
		return old
	}
	m.insert(key, value, hash, index, exec)
	return nil
}

// insert reuses the successful lookup, avoiding a second call to user key
// hashCode/equals or comparison methods after a mapping callback.
func (m *Map[K, V]) insert(key any, value any, hash int32, index int, exec *Execution) {
	if m.sorted && len(m.entries) == 0 {
		m.compare(key, key, exec)
	}
	record := &mapRecord[K, V]{key: key, value: value}
	if m.sorted {
		m.entries = append(m.entries, nil)
		copy(m.entries[index+1:], m.entries[index:])
		m.entries[index] = record
	} else {
		if m.buckets == nil {
			m.buckets = make(map[int32][]*mapRecord[K, V])
		}
		m.buckets[hash] = append(m.buckets[hash], record)
		m.entries = append(m.entries, record)
	}
	m.modCount++
}

// ComputeIfAbsent follows HashMap and TreeMap's callback contract, including
// rejection of structural mutation while preserving the callback's own writes.
func (m *Map[K, V]) ComputeIfAbsent(key K, mapping func(K) V, execution ...*Execution) V {
	ReferenceRequireNonNull(m)
	ReferenceRequireNonNull(mapping)
	exec := optionalComparisonExecution(execution)
	var record *mapRecord[K, V]
	var hash int32
	var index int
	// TreeMap defers validating a key in an empty tree until a non-null value
	// actually needs insertion. A null-producing callback can therefore use null.
	if !m.sorted || len(m.entries) != 0 {
		record, hash, index = m.find(key, exec)
	}
	if record != nil && !javaReferenceIsNull(record.value) {
		return collectionElementView[V](record.value)
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
	m.insert(key, computed, hash, index, exec)
	return computed
}

func (m *Map[K, V]) Get(key any, execution ...*Execution) V {
	return m.GetOrDefault(key, collectionZero[V](), execution...)
}
func (m *Map[K, V]) GetOrDefault(key any, fallback V, execution ...*Execution) V {
	if record, _, _ := m.find(key, optionalComparisonExecution(execution)); record != nil {
		return collectionElementView[V](record.value)
	}
	return fallback
}
func (m *Map[K, V]) ContainsKey(key any, execution ...*Execution) bool {
	record, _, _ := m.find(key, optionalComparisonExecution(execution))
	return record != nil
}
func (m *Map[K, V]) ContainsValue(value any, execution ...*Execution) bool {
	for _, record := range m.entries {
		if ObjectsEqual(value, record.value, execution...) {
			return true
		}
	}
	return false
}
func (m *Map[K, V]) Remove(key any, execution ...*Execution) V {
	return collectionElementView[V](m.RemoveObject(key, execution...))
}
func (m *Map[K, V]) RemoveObject(key any, execution ...*Execution) any {
	record, _, _ := m.find(key, optionalComparisonExecution(execution))
	if record == nil {
		return nil
	}
	m.removeRecord(record)
	return record.value
}

// removeRecord removes the exact node without recomputing a mutable key's hash
// or invoking user equality/comparison callbacks from iterator.remove().
func (m *Map[K, V]) removeRecord(record *mapRecord[K, V]) {
	index := -1
	for i, candidate := range m.entries {
		if candidate == record {
			index = i
			break
		}
	}
	if index < 0 {
		panic(NewConcurrentModificationException("detached map record"))
	}
	if !m.sorted {
		for hash, bucket := range m.buckets {
			for i, candidate := range bucket {
				if candidate != record {
					continue
				}
				copy(bucket[i:], bucket[i+1:])
				bucket[len(bucket)-1] = nil
				if len(bucket) == 1 {
					delete(m.buckets, hash)
				} else {
					m.buckets[hash] = bucket[:len(bucket)-1]
				}
				break
			}
		}
	}
	copy(m.entries[index:], m.entries[index+1:])
	m.entries[len(m.entries)-1] = nil
	m.entries = m.entries[:len(m.entries)-1]
	m.modCount++
}
func (m *Map[K, V]) Size() int32   { return int32(len(m.entries)) }
func (m *Map[K, V]) IsEmpty() bool { return len(m.entries) == 0 }
func (m *Map[K, V]) Clear()        { m.modCount++; m.buckets = nil; m.entries = nil }
func (m *Map[K, V]) KeySet() []K {
	out := make([]K, len(m.entries))
	for i, record := range m.entries {
		out[i] = collectionElementView[K](record.key)
	}
	return out
}
func (m *Map[K, V]) Values() []V {
	out := make([]V, len(m.entries))
	for i, record := range m.entries {
		out[i] = collectionElementView[V](record.value)
	}
	return out
}
func (m *Map[K, V]) EntrySet() []MapEntry[K, V] {
	out := make([]MapEntry[K, V], len(m.entries))
	for i, record := range m.entries {
		out[i] = MapEntry[K, V]{Key: collectionElementView[K](record.key), Value: collectionElementView[V](record.value)}
	}
	return out
}
func (m *Map[K, V]) String() string {
	parts := make([]string, len(m.entries))
	for i, record := range m.entries {
		parts[i] = StringValueOf(record.key) + "=" + StringValueOf(record.value)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// Java Strings use a null sentinel instead of Go's empty string.
func collectionZero[T any]() T {
	var zero T
	if _, ok := any(zero).(string); ok {
		return any(NullString()).(T)
	}
	return zero
}
