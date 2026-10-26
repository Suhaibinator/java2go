package stdjava

// readerVirtualInvocationReceiver resolves a generated superclass view to the
// same allocation's most-derived source view. Native and opaque legacy Reader
// implementations retain their existing protocol path.
func readerVirtualInvocationReceiver(reader Reader) Reader {
	carrier, hasInfo := reader.(JavaObjectInfoCarrier)
	if !hasInfo {
		return reader
	}
	info := carrier.JavaObjectInfo()
	if info == nil {
		return reader
	}
	actual := info.DynamicType()
	javaTypeRegistry.RLock()
	registered, known := javaTypeRegistry.types[actual]
	javaTypeRegistry.RUnlock()
	if !known || !registered.source || !JavaTypeAssignable(actual, JavaIOReaderType) {
		return reader
	}
	resolved, compatible := info.resolveView(actual).(Reader)
	if !compatible || javaReferenceIsNull(resolved) || !JavaReferenceEqual(reader, resolved) {
		panic(NewClassCastException("Reader implementation lacks most-derived source view"))
	}
	return resolved
}
