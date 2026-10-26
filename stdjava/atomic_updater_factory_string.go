package stdjava

// Declared lookup consumes canonical UTF16 names. The validated descriptor name
// is then passed to the existing factory without a lossy host text conversion.
func atomicUpdaterDeclaredName(owner *Class, name *JavaString) (result string) {
	defer func() {
		if failure := recover(); failure != nil {
			panic(NewRuntimeException(failure))
		}
	}()
	return owner.GetDeclaredFieldJavaString(name).GetName()
}
func NewAtomicIntegerFieldUpdaterJavaString(owner *Class, name *JavaString, caller TypeID) *AtomicIntegerFieldUpdater {
	return NewAtomicIntegerFieldUpdater(owner, atomicUpdaterDeclaredName(owner, name), caller)
}
func NewAtomicLongFieldUpdaterJavaString(owner *Class, name *JavaString, caller TypeID) *AtomicLongFieldUpdater {
	return NewAtomicLongFieldUpdater(owner, atomicUpdaterDeclaredName(owner, name), caller)
}
func NewAtomicReferenceFieldUpdaterJavaString(owner, valueClass *Class, name *JavaString, caller TypeID) *AtomicReferenceFieldUpdater {
	return NewAtomicReferenceFieldUpdater(owner, valueClass, atomicUpdaterDeclaredName(owner, name), caller)
}
