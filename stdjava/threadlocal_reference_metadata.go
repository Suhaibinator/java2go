package stdjava

const ThreadLocalTypeID TypeID = "java.lang.ThreadLocal"

const threadLocalSuppliedTypeID TypeID = "java.lang.ThreadLocal$SuppliedThreadLocal"

type threadLocalReferenceKind uint8

const (
	threadLocalReferenceLegacy threadLocalReferenceKind = iota
	threadLocalReferencePlain
	threadLocalReferenceSupplied
)

// NewThreadLocalReference allocates the canonical Java reference service. The
// native Go factories retain their historical short descriptor and typed API.
func NewThreadLocalReference() *ThreadLocal[any] {
	local := NewThreadLocal[any]()
	local.referenceKind = threadLocalReferencePlain
	return local
}

func init() {
	RegisterJavaType(ThreadLocalTypeID, ObjectTypeID)
	RegisterJavaType(threadLocalSuppliedTypeID, ThreadLocalTypeID)
	for _, declaration := range []struct {
		id        TypeID
		modifiers int32
	}{{ThreadLocalTypeID, 1}, {threadLocalSuppliedTypeID, 24}} {
		descriptor := classDescriptor(declaration.id)
		if descriptor.HasModifiers && descriptor.Modifiers != declaration.modifiers {
			panic("conflicting canonical ThreadLocal declaration modifiers")
		}
		descriptor.Modifiers = declaration.modifiers
		descriptor.HasModifiers = true
		reflectionRegistry.Store(declaration.id, descriptor)
	}
}
