package app

type ResourceToken struct {
	trace	*CloseTrace
	label	string
}

func NewResourceToken(trace *CloseTrace, label string) *ResourceToken {
	rn := new(ResourceToken)
	rn.trace = trace
	rn.label = label
	return rn
}
func (rn *ResourceToken) Close() {
	rn.trace.Append(rn.label)
}
