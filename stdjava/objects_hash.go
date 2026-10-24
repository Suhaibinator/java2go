package stdjava

// ObjectsHashExecution implements Objects.hash(Object...). A fixed-arity array
// remains shared: load each entry after the preceding dynamic hash invocation.
func ObjectsHashExecution(execution *Execution, values *ReferenceArray) int32 {
	if values == nil {
		return 0
	}
	hash := int32(1)
	for _, value := range ReferenceArrayIterationElements(values) {
		elementHash := int32(0)
		if !javaReferenceIsNull(value) {
			elementHash = ObjectHashCodeExecution(execution, value)
		}
		hash = 31*hash + elementHash
	}
	return hash
}
