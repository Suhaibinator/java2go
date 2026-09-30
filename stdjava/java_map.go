package stdjava

// JavaMap carries the same source map reference across erased operations.
// Logical Java key/value arguments and nominal membership remain separate.
type JavaMap interface {
	MapEntrySetJava2goExecution(*Execution) JavaIterable
}

// MapReferenceView resolves a generated superclass view to its most-derived
// source object without dereferencing null or evaluating a Java callback.
func MapReferenceView(value any) JavaMap {
	return ObjectView[JavaMap](collectionObjectView(value), MapTypeID)
}
