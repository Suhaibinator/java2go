package app

import stdjava "github.com/NickyBoy89/java2go/stdjava"

type CloseTrace struct {
	value string
}

var __java2goReferenceTypeRegistrationCloseTrace = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.app.CloseTrace"), stdjava.ObjectTypeID)
	return true
}()

func (ce *CloseTrace) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.app.CloseTrace")
}
func NewCloseTrace() *CloseTrace {
	return NewCloseTraceJava2goExecution(stdjava.NewExecution())
}
func NewCloseTraceJava2goExecution(__java2goExecution *stdjava.Execution) *CloseTrace {
	ce := new(CloseTrace)
	ce.value = "\xffjava2go:null-string\x00"
	ce.value = ""
	return ce
}
func (ce *CloseTrace) Append(piece string) {
	ce.AppendJava2goExecution(stdjava.NewExecution(), piece)
}
func (ce *CloseTrace) AppendJava2goExecution(__java2goExecution *stdjava.Execution, piece string) {
	if ce == nil {
		_ = *ce
	}
	func(dst *string) func(rhs string) string {
		old := *dst
		return func(rhs string) string {
			value := stdjava.StringValueOf(old) + stdjava.StringValueOf(rhs)
			*dst = value
			return value
		}
	}(&ce.value)(piece)
}
func (ce *CloseTrace) Get() string {
	return ce.GetJava2goExecution(stdjava.NewExecution())
}
func (ce *CloseTrace) GetJava2goExecution(__java2goExecution *stdjava.Execution) string {
	if ce == nil {
		_ = *ce
	}
	return ce.value
}
