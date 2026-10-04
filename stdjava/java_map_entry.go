package stdjava

// JavaMapEntry preserves an entry object while its generic key/value slots are
// erased. Source implementations and native live entries share this boundary.
type JavaMapEntry interface {
	GetKeyJava2goExecution(*Execution) any
	GetValueJava2goExecution(*Execution) any
	SetValueJava2goExecution(*Execution, any) any
}

const JavaMapEntryTypeID TypeID = "java.util.Map$Entry"

func init() { RegisterJavaType(JavaMapEntryTypeID, ObjectTypeID) }

func MapEntryGetKeyExecution(execution *Execution, entry JavaMapEntry) any {
	requireExecution(execution)
	ReferenceRequireNonNull(entry)
	return entry.GetKeyJava2goExecution(execution)
}

func MapEntryGetValueExecution(execution *Execution, entry JavaMapEntry) any {
	requireExecution(execution)
	ReferenceRequireNonNull(entry)
	return entry.GetValueJava2goExecution(execution)
}

// A caller narrows the completed old-value result after the source write.
func MapEntrySetValueExecution(execution *Execution, entry JavaMapEntry, value any) any {
	requireExecution(execution)
	ReferenceRequireNonNull(entry)
	return entry.SetValueJava2goExecution(execution, value)
}
