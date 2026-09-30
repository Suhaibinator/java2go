package common

import stdjava "github.com/NickyBoy89/java2go/stdjava"

type Mode struct {
	enumName    string
	enumOrdinal int32
}

var __java2goReferenceTypeRegistrationMode = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.common.Mode"), stdjava.ObjectTypeID)
	return true
}()

func (synthetic *Mode) JavaDynamicTypeID() stdjava.TypeID {
	return stdjava.TypeID("com.acme.common.Mode")
}

const (
	_mode_ordinal_FAST = iota
	_mode_ordinal_SAFE
)

var (
	FAST = func() *Mode {
		inst := &Mode{}
		inst.enumName = "FAST"
		inst.enumOrdinal = _mode_ordinal_FAST
		return inst
	}()
	SAFE = func() *Mode {
		inst := &Mode{}
		inst.enumName = "SAFE"
		inst.enumOrdinal = _mode_ordinal_SAFE
		return inst
	}()
)
var _modeValues = []*Mode{FAST, SAFE}

func ModeValues() []*Mode {
	return _modeValues
}
func ModeValueOf(name string) *Mode {
	switch name {
	case "FAST":
		return FAST
	case "SAFE":
		return SAFE
	default:
		panic("No enum constant " + name)
		return nil
	}
}
func (me *Mode) Name() string {
	return me.enumName
}
func (me *Mode) String() string {
	return me.StringJava2goExecution(stdjava.NewExecution())
}
func (me *Mode) StringJava2goExecution(__java2goExecution *stdjava.Execution) string {
	if me == nil {
		return "null"
	}
	return me.enumName
}
func (me *Mode) Ordinal() int32 {
	return me.enumOrdinal
}
func (me *Mode) CompareTo(other *Mode) int32 {
	return me.enumOrdinal - other.enumOrdinal
}
