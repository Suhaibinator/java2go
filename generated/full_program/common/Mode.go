package common

type Mode struct {
	enumName	string
	enumOrdinal	int
}

const (
	_mode_ordinal_FAST	= iota
	_mode_ordinal_SAFE
)

var (
	FAST	= func() *Mode {
		inst := &Mode{}
		inst.enumName = "FAST"
		inst.enumOrdinal = _mode_ordinal_FAST
		return inst
	}()
	SAFE	= func() *Mode {
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
func (me *Mode) Ordinal() int {
	return me.enumOrdinal
}
func (me *Mode) CompareTo(other *Mode) int {
	return me.enumOrdinal - other.enumOrdinal
}
