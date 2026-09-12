package stdjava

// Private erased views allow collection equality across Go instantiations,
// matching Java's erased List/Set/Map contracts without treating arrays as lists.
type javaListValue interface{ javaListElements() []any }
type javaSetValue interface {
	javaSetElements() []any
	Contains(any, ...*Execution) bool
	Size() int32
}
type javaMapValue interface {
	javaMapLookup(any, *Execution) (any, bool)
	Size() int32
}
type javaEntryValue interface{ javaEntry() (any, any) }

func (l *List[T]) javaListElements() []any {
	out := make([]any, len(l.elements))
	for i, element := range l.elements {
		out[i] = element
	}
	return out
}
func (l *List[T]) Equals(other any) bool { return l.EqualsJava2goExecution(nil, other) }
func (l *List[T]) EqualsJava2goExecution(execution *Execution, other any) bool {
	ReferenceRequireNonNull(l)
	if JavaReferenceEqual(l, other) {
		return true
	}
	if javaReferenceIsNull(other) {
		return false
	}
	right, ok := other.(javaListValue)
	if !ok {
		return false
	}
	elements := right.javaListElements()
	if len(l.elements) != len(elements) {
		return false
	}
	for i, element := range l.elements {
		if !ObjectsEqual(element, elements[i], execution) {
			return false
		}
	}
	return true
}
func (l *List[T]) HashCode() int32 { return l.HashCodeJava2goExecution(nil) }
func (l *List[T]) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(l)
	hash := int32(1)
	for _, element := range l.elements {
		hash = 31*hash + ObjectsHashCode(element, execution)
	}
	return hash
}

func (s *Set[T]) javaSetElements() []any {
	elements := s.Slice()
	out := make([]any, len(elements))
	for i, element := range elements {
		out[i] = element
	}
	return out
}
func (s *Set[T]) Equals(other any) bool { return s.EqualsJava2goExecution(nil, other) }
func (s *Set[T]) EqualsJava2goExecution(execution *Execution, other any) (equal bool) {
	defer collectionEqualityFailure(&equal)
	ReferenceRequireNonNull(s)
	if JavaReferenceEqual(s, other) {
		return true
	}
	if javaReferenceIsNull(other) {
		return false
	}
	right, ok := other.(javaSetValue)
	if !ok || s.Size() != right.Size() {
		return false
	}
	for _, element := range right.javaSetElements() {
		if !s.Contains(element, execution) {
			return false
		}
	}
	return true
}
func (s *Set[T]) HashCode() int32 { return s.HashCodeJava2goExecution(nil) }
func (s *Set[T]) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(s)
	var hash int32
	for _, element := range s.Slice() {
		hash += ObjectsHashCode(element, execution)
	}
	return hash
}

func (m *Map[K, V]) javaMapLookup(key any, execution *Execution) (any, bool) {
	if record, _, _ := m.find(key, execution); record != nil {
		return record.entry.Value, true
	}
	return nil, false
}
func (m *Map[K, V]) Equals(other any) bool { return m.EqualsJava2goExecution(nil, other) }
func (m *Map[K, V]) EqualsJava2goExecution(execution *Execution, other any) (equal bool) {
	defer collectionEqualityFailure(&equal)
	ReferenceRequireNonNull(m)
	if JavaReferenceEqual(m, other) {
		return true
	}
	if javaReferenceIsNull(other) {
		return false
	}
	right, ok := other.(javaMapValue)
	if !ok || m.Size() != right.Size() {
		return false
	}
	for _, record := range m.entries {
		value, present := right.javaMapLookup(record.entry.Key, execution)
		if !present || !ObjectsEqual(record.entry.Value, value, execution) {
			return false
		}
	}
	return true
}
func (m *Map[K, V]) HashCode() int32 { return m.HashCodeJava2goExecution(nil) }
func (m *Map[K, V]) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(m)
	var hash int32
	for _, record := range m.entries {
		hash += record.entry.HashCodeJava2goExecution(execution)
	}
	return hash
}

func (e MapEntry[K, V]) javaEntry() (any, any) { return e.Key, e.Value }
func (e MapEntry[K, V]) Equals(other any) bool { return e.EqualsJava2goExecution(nil, other) }
func (e MapEntry[K, V]) EqualsJava2goExecution(execution *Execution, other any) bool {
	if javaReferenceIsNull(other) {
		return false
	}
	right, ok := other.(javaEntryValue)
	if !ok {
		return false
	}
	key, value := right.javaEntry()
	return ObjectsEqual(e.Key, key, execution) && ObjectsEqual(e.Value, value, execution)
}
func (e MapEntry[K, V]) HashCode() int32 { return e.HashCodeJava2goExecution(nil) }
func (e MapEntry[K, V]) HashCodeJava2goExecution(execution *Execution) int32 {
	return ObjectsHashCode(e.Key, execution) ^ ObjectsHashCode(e.Value, execution)
}

func collectionEqualityFailure(equal *bool) {
	if recovered := recover(); recovered != nil {
		name := throwableTypeName(recovered)
		if isSubtypeOf(name, "ClassCastException") || isSubtypeOf(name, "NullPointerException") {
			*equal = false
			return
		}
		panic(recovered)
	}
}

func (c *ConcurrentHashMap[K, V]) javaMapLookup(key any, execution *Execution) (any, bool) {
	value, present := c.GetOk(key, execution)
	return value, present
}
func (c *ConcurrentHashMap[K, V]) Equals(other any) bool {
	return c.EqualsJava2goExecution(nil, other)
}
func (c *ConcurrentHashMap[K, V]) EqualsJava2goExecution(execution *Execution, other any) (equal bool) {
	defer collectionEqualityFailure(&equal)
	ReferenceRequireNonNull(c)
	if JavaReferenceEqual(c, other) {
		return true
	}
	if javaReferenceIsNull(other) {
		return false
	}
	right, ok := other.(javaMapValue)
	if !ok || c.Size() != right.Size() {
		return false
	}
	for _, entry := range c.EntrySet() {
		value, present := right.javaMapLookup(entry.Key, execution)
		if !present || !ObjectsEqual(entry.Value, value, execution) {
			return false
		}
	}
	return true
}
func (c *ConcurrentHashMap[K, V]) HashCode() int32 { return c.HashCodeJava2goExecution(nil) }
func (c *ConcurrentHashMap[K, V]) HashCodeJava2goExecution(execution *Execution) int32 {
	ReferenceRequireNonNull(c)
	var hash int32
	for _, entry := range c.EntrySet() {
		hash += entry.HashCodeJava2goExecution(execution)
	}
	return hash
}
