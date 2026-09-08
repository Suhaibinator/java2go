package app

import stdjava "github.com/NickyBoy89/java2go/stdjava"

type ResourceToken struct {
	trace *CloseTrace
	label string
}

var __java2goReferenceTypeRegistrationResourceToken = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.app.ResourceToken"), stdjava.ObjectTypeID)
	return true
}()

func (rn *ResourceToken) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.app.ResourceToken")
}
func NewResourceToken(trace *CloseTrace, label string) *ResourceToken {
	return NewResourceTokenJava2goExecution(stdjava.NewExecution(), trace, label)
}
func NewResourceTokenJava2goExecution(__java2goExecution *stdjava.Execution, trace *CloseTrace, label string) *ResourceToken {
	rn := new(ResourceToken)
	rn.label = "\xffjava2go:null-string\x00"
	rn.trace = trace
	rn.label = label
	return rn
}
func (rn *ResourceToken) Close() {
	rn.CloseJava2goExecution(stdjava.NewExecution())
}
func (rn *ResourceToken) CloseJava2goExecution(__java2goExecution *stdjava.Execution) {
	if rn == nil {
		_ = *rn
	}
	rn.trace.AppendJava2goExecution(__java2goExecution, rn.label)
}
