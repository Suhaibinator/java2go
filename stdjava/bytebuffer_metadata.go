package stdjava

// Allocation identity belongs to the native constructor, independently of the
// shared Go carrier, requested Java view, and mutable buffer cursor state.
type byteBufferAllocationKind uint8

const (
	byteBufferUnknownAllocation byteBufferAllocationKind = iota
	byteBufferHeapAllocation
	byteBufferHeapReadOnlyAllocation
)

const (
	byteBufferHeapTypeID         TypeID = "java.nio.HeapByteBuffer"
	byteBufferHeapReadOnlyTypeID TypeID = "java.nio.HeapByteBufferR"
)

func (kind byteBufferAllocationKind) dynamicTypeID() TypeID {
	switch kind {
	case byteBufferHeapAllocation:
		return byteBufferHeapTypeID
	case byteBufferHeapReadOnlyAllocation:
		return byteBufferHeapReadOnlyTypeID
	default:
		return ""
	}
}

func (kind byteBufferAllocationKind) readOnlyKind() byteBufferAllocationKind {
	switch kind {
	case byteBufferHeapAllocation, byteBufferHeapReadOnlyAllocation:
		return byteBufferHeapReadOnlyAllocation
	default:
		return byteBufferUnknownAllocation
	}
}

func init() {
	// These exact hierarchy/modifier/location facts were observed with pinned
	// JDK21 allocation and declaration-location probes before registration.
	location := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
	RegisterClassRuntimeLocation(JavaNIOBufferType, location)
	RegisterClassRuntimeLocation(JavaNIOByteBufferType, location)
	RegisterJavaType(byteBufferHeapTypeID, JavaNIOByteBufferType)
	RegisterJavaType(byteBufferHeapReadOnlyTypeID, byteBufferHeapTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: byteBufferHeapTypeID, HasModifiers: true, Modifiers: 0, RuntimeLocation: location})
	RegisterClassDescriptor(ClassDescriptor{Type: byteBufferHeapReadOnlyTypeID, HasModifiers: true, Modifiers: 16, RuntimeLocation: location})
}
