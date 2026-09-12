package stdjava

import "strings"

// MapEntry is the key/value pair exposed by Map.entrySet.
type MapEntry[K, V any] struct {
	Key   K
	Value V
}

type mapRecord[K, V any] struct {
	entry MapEntry[K, V]
}

// Map uses Java hashCode/equals collision buckets. Keys may have any generated
// Go representation; their Go comparability does not determine Java equality.
// Iteration is deterministic insertion order for hash maps. Tree maps keep
// entries ordered and determine key equivalence exclusively by comparison.
// keySet/values/entrySet remain snapshots, rather than live Java views.
type Map[K, V any] struct {
	buckets    map[int32][]*mapRecord[K, V]
	entries    []*mapRecord[K, V]
	sorted     bool
	comparator Comparator[K]
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

func (m *Map[K, V]) compare(key any, stored K, execution *Execution) int32 {
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
	return m.comparator(typed, stored)
}

func (m *Map[K, V]) find(key any, execution *Execution) (*mapRecord[K, V], int32, int) {
	if m.sorted {
		if m.comparator == nil {
			ReferenceRequireNonNull(key)
		}
		low, high := 0, len(m.entries)
		for low < high {
			mid := low + (high-low)/2
			comparison := m.compare(key, m.entries[mid].entry.Key, execution)
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
		if ObjectsEqual(key, record.entry.Key, execution) {
			return record, hash, 0
		}
	}
	return nil, hash, 0
}

func (m *Map[K, V]) Put(key K, value V, execution ...*Execution) V {
	exec := optionalComparisonExecution(execution)
	record, hash, index := m.find(key, exec)
	if record != nil {
		old := record.entry.Value
		record.entry.Value = value
		return old
	}
	if m.sorted && len(m.entries) == 0 {
		m.compare(key, key, exec)
	}
	record = &mapRecord[K, V]{entry: MapEntry[K, V]{Key: key, Value: value}}
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
	return collectionZero[V]()
}

func (m *Map[K, V]) Get(key any, execution ...*Execution) V {
	return m.GetOrDefault(key, collectionZero[V](), execution...)
}
func (m *Map[K, V]) GetOrDefault(key any, fallback V, execution ...*Execution) V {
	if record, _, _ := m.find(key, optionalComparisonExecution(execution)); record != nil {
		return record.entry.Value
	}
	return fallback
}
func (m *Map[K, V]) ContainsKey(key any, execution ...*Execution) bool {
	record, _, _ := m.find(key, optionalComparisonExecution(execution))
	return record != nil
}
func (m *Map[K, V]) ContainsValue(value any, execution ...*Execution) bool {
	for _, record := range m.entries {
		if ObjectsEqual(value, record.entry.Value, execution...) {
			return true
		}
	}
	return false
}
func (m *Map[K, V]) Remove(key any, execution ...*Execution) V {
	record, hash, index := m.find(key, optionalComparisonExecution(execution))
	if record == nil {
		return collectionZero[V]()
	}
	if !m.sorted {
		bucket := m.buckets[hash]
		for i, candidate := range bucket {
			if candidate == record {
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
		for i, candidate := range m.entries {
			if candidate == record {
				index = i
				break
			}
		}
	}
	copy(m.entries[index:], m.entries[index+1:])
	m.entries[len(m.entries)-1] = nil
	m.entries = m.entries[:len(m.entries)-1]
	return record.entry.Value
}
func (m *Map[K, V]) Size() int32   { return int32(len(m.entries)) }
func (m *Map[K, V]) IsEmpty() bool { return len(m.entries) == 0 }
func (m *Map[K, V]) Clear()        { m.buckets = nil; m.entries = nil }
func (m *Map[K, V]) KeySet() []K {
	out := make([]K, len(m.entries))
	for i, record := range m.entries {
		out[i] = record.entry.Key
	}
	return out
}
func (m *Map[K, V]) Values() []V {
	out := make([]V, len(m.entries))
	for i, record := range m.entries {
		out[i] = record.entry.Value
	}
	return out
}
func (m *Map[K, V]) EntrySet() []MapEntry[K, V] {
	out := make([]MapEntry[K, V], len(m.entries))
	for i, record := range m.entries {
		out[i] = record.entry
	}
	return out
}
func (m *Map[K, V]) String() string {
	parts := make([]string, len(m.entries))
	for i, record := range m.entries {
		parts[i] = StringValueOf(record.entry.Key) + "=" + StringValueOf(record.entry.Value)
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
