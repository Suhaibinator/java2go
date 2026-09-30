package stdjava

// Hash-map copy construction owns new records while preserving key/value references.
// The capacity overload affects allocation only; it does not narrow raw elements.
func NewMapWithArgument[K, V any](argument any, executions ...*Execution) *Map[K, V] {
	switch capacity := argument.(type) {
	case int:
		if capacity < 0 {
			panic(NewIllegalArgumentException("Illegal initial capacity"))
		}
		return NewMap[K, V]()
	case int32:
		if capacity < 0 {
			panic(NewIllegalArgumentException("Illegal initial capacity"))
		}
		return NewMap[K, V]()
	}
	result := NewMap[K, V]()
	result.PutAll(argument, executions...)
	return result
}
