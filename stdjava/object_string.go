package stdjava

import "strconv"

// ObjectDefaultStringExecution is Object.toString's implementation. It bypasses
// a toString override while retaining virtual hashCode dispatch and the most
// derived class identity, as a source call to super.toString() requires.
func ObjectDefaultStringExecution(execution *Execution, value any) string {
	ReferenceRequireNonNull(value)
	value = collectionObjectView(value)
	name := ObjectGetClass(value).GetName()
	hash := ObjectHashCodeExecution(execution, value)
	return name + "@" + strconv.FormatUint(uint64(uint32(hash)), 16)
}
