package stdjava

import "strings"

// This file implements the map type that java.util.Map implementations
// (HashMap, TreeMap) are mapped onto. HashMap and TreeMap share one Go type;
// TreeMap's sorted iteration is approximated by sorting keys at iteration time
// only where an ordering is requested, and is otherwise insertion/Go-map order.
// The distinction is documented and not faithfully modelled.
//
// Keys must be comparable (Go map key requirement), which holds for the Java
// types used as map keys in practice (String, boxed numbers, enums).

// MapEntry is a single key/value pair, matching Map.Entry, used by entrySet
// iteration.
type MapEntry[K comparable, V any] struct {
	Key   K
	Value V
}

// Map is a generic map matching the subset of java.util.Map used by transpiled
// code. It is a pointer type so mutations are visible to all holders.
type Map[K comparable, V any] struct {
	backing map[any]V
	// keys preserves insertion order so iteration is deterministic, matching the
	// predictability transpiled programs rely on (Go map order is randomized).
	keys []K
}

// NewMap returns an empty Map, matching `new HashMap<>()` / `new TreeMap<>()`.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{backing: make(map[any]V)}
}

// Put associates key with value and returns the previous value (or the zero
// value), matching Map.put.
func (m *Map[K, V]) Put(key K, value V) V {
	normalized := collectionKey(key)
	old := m.backing[normalized]
	if _, exists := m.backing[normalized]; !exists {
		m.keys = append(m.keys, key)
	}
	m.backing[normalized] = value
	return old
}

// Get returns the value for key, or the zero value if absent, matching Map.get
// (which returns null when the key is missing).
func (m *Map[K, V]) Get(key any) V {
	return m.backing[collectionKey(key)]
}

// GetOrDefault returns the value for key, or defaultValue if absent, matching
// Map.getOrDefault.
func (m *Map[K, V]) GetOrDefault(key any, defaultValue V) V {
	if v, ok := m.backing[collectionKey(key)]; ok {
		return v
	}
	return defaultValue
}

// ContainsKey reports whether key is present, matching Map.containsKey.
func (m *Map[K, V]) ContainsKey(key any) bool {
	_, ok := m.backing[collectionKey(key)]
	return ok
}

// ContainsValue reports whether some key maps to a value equal to target,
// matching Map.containsValue (Java-style value equality).
func (m *Map[K, V]) ContainsValue(target any) bool {
	for _, k := range m.keys {
		if ObjectsEqual(target, m.backing[collectionKey(k)]) {
			return true
		}
	}
	return false
}

// Remove deletes key and returns the previous value, matching Map.remove.
func (m *Map[K, V]) Remove(key any) V {
	normalized := collectionKey(key)
	old := m.backing[normalized]
	if _, ok := m.backing[normalized]; ok {
		delete(m.backing, normalized)
		for i, k := range m.keys {
			if collectionKey(k) == normalized {
				m.keys = append(m.keys[:i], m.keys[i+1:]...)
				break
			}
		}
	}
	return old
}

// Size returns the number of entries, matching Map.size.
func (m *Map[K, V]) Size() int32 {
	return int32(len(m.backing))
}

// IsEmpty reports whether the map has no entries, matching Map.isEmpty.
func (m *Map[K, V]) IsEmpty() bool {
	return len(m.backing) == 0
}

// Clear removes all entries, matching Map.clear.
func (m *Map[K, V]) Clear() {
	m.backing = make(map[any]V)
	m.keys = nil
}

// KeySet returns the keys in insertion order, matching Map.keySet (the return is
// iteration-friendly rather than a live Set view).
func (m *Map[K, V]) KeySet() []K {
	cp := make([]K, len(m.keys))
	copy(cp, m.keys)
	return cp
}

// Values returns the values in key-insertion order, matching Map.values.
func (m *Map[K, V]) Values() []V {
	vs := make([]V, 0, len(m.keys))
	for _, k := range m.keys {
		vs = append(vs, m.backing[collectionKey(k)])
	}
	return vs
}

// EntrySet returns the entries in key-insertion order, matching Map.entrySet.
func (m *Map[K, V]) EntrySet() []MapEntry[K, V] {
	es := make([]MapEntry[K, V], 0, len(m.keys))
	for _, k := range m.keys {
		es = append(es, MapEntry[K, V]{Key: k, Value: m.backing[collectionKey(k)]})
	}
	return es
}

// String returns the Java AbstractMap.toString form, e.g. "{a=1, b=2}", in key
// insertion order, so a Map printed via fmt matches Java's output.
func (m *Map[K, V]) String() string {
	parts := make([]string, len(m.keys))
	for i, k := range m.keys {
		parts[i] = StringValueOf(k) + "=" + StringValueOf(m.backing[collectionKey(k)])
	}
	return "{" + strings.Join(parts, ", ") + "}"
}
