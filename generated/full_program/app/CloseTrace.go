package app

type CloseTrace struct {
	value string
}

func NewCloseTrace() *CloseTrace {
	ce := new(CloseTrace)
	ce.value = ""
	return ce
}
func (ce *CloseTrace) Append(piece string) {
	ce.value += piece
}
func (ce *CloseTrace) Get() string {
	return ce.value
}
