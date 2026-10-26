package stdjava

import (
	"strconv"
)

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

// ObjectDefaultJavaStringExecution allocates Object.toString's Java reference
// result. The nominal class name and virtual hashCode share the caller's logical
// execution; this path does not invoke host formatting or a native text method.
func ObjectDefaultJavaStringExecution(execution *Execution, value any) *JavaString {
	ReferenceRequireNonNull(value)
	value = collectionObjectView(value)
	units := append([]uint16(nil), ClassJavaName(ObjectGetClass(value)).units...)
	units = append(units, '@')
	units = append(units, JavaIntegerToHexString(ObjectHashCodeExecution(execution, value)).units...)
	return &JavaString{units: units}
}
