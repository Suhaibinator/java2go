package domain

import common "com/acme/common"

type ParseTask struct {
	*Task
}

func NewParseTask(id string) *ParseTask {
	pk := new(ParseTask)
	pk.Task = NewTask(id)
	return pk
}
func (pk *ParseTask) Run(input string) int32 {
	trim := common.NewMapperFuncAdapter[string, string](func(v string) string {
		return v
	})
	normalized := trim.Map(input)
	if func() bool {
		_, ok := any(normalized).(string)
		return ok
	}() {
		return int32(len(normalized))
	}
	return 0
}
