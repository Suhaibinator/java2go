package domain

import (
	common "com/acme/common"
	stdjava "github.com/NickyBoy89/java2go/stdjava"
)

type ParseTask struct {
	*Task
}

func (pk *ParseTask) Java2goInstallTaskSubobjectFrom636F6D2E61636D652E646F6D61696E2E5461736B(__java2goSubobject *Task) {
	pk.Task = __java2goSubobject
}

var __java2goReferenceTypeRegistrationParseTask = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.domain.ParseTask"), stdjava.TypeID("com.acme.domain.Task"))
	return true
}()

func (pk *ParseTask) Java2goReferenceDynamicType() stdjava.TypeID {
	return stdjava.TypeID("com.acme.domain.ParseTask")
}
func (pk *ParseTask) Java2goReferenceView(__java2goRequestedType stdjava.TypeID) any {
	switch __java2goRequestedType {
	case stdjava.TypeID("com.acme.domain.ParseTask"):
		return pk
	case stdjava.TypeID("com.acme.domain.Task"):
		return pk.Task
	default:
		return nil
	}
}
func (pk *ParseTask) Java2goSetParseTaskSelf(self any) {
	pk.Task.Java2goSetTaskSelf(self)
}
func NewParseTask(id string) *ParseTask {
	return NewParseTaskJava2goExecution(stdjava.NewExecution(), id)
}
func NewParseTaskJava2goExecution(__java2goExecution *stdjava.Execution, id string) *ParseTask {
	return NewParseTaskJava2goWithSelfJava2goExecution(__java2goExecution, nil, id)
}
func NewParseTaskJava2goWithSelf(__java2goMostDerived any, id string) *ParseTask {
	return NewParseTaskJava2goWithSelfJava2goExecution(stdjava.NewExecution(), __java2goMostDerived, id)
}
func NewParseTaskJava2goWithSelfJava2goExecution(__java2goExecution *stdjava.Execution, __java2goMostDerived any, id string) *ParseTask {
	pk := new(ParseTask)
	if __java2goMostDerived == nil {
		__java2goMostDerived = pk
	}
	if __java2goSubobjectInstaller, __java2goHasSubobjectInstaller := __java2goMostDerived.(interface {
		Java2goInstallParseTaskSubobjectFrom636F6D2E61636D652E646F6D61696E2E50617273655461736B(*ParseTask)
	}); __java2goHasSubobjectInstaller {
		__java2goSubobjectInstaller.Java2goInstallParseTaskSubobjectFrom636F6D2E61636D652E646F6D61696E2E50617273655461736B(pk)
	}
	pk.Task = NewTaskJava2goWithSelfJava2goExecution(__java2goExecution, __java2goMostDerived, id)
	pk.Java2goSetParseTaskSelf(__java2goMostDerived)
	return pk
}
func (pk *ParseTask) Run(input string) int32 {
	return pk.RunJava2goExecution(stdjava.NewExecution(), input)
}
func (pk *ParseTask) RunJava2goExecution(__java2goExecution *stdjava.Execution, input string) int32 {
	if pk == nil {
		_ = *pk
	}
	trim := common.NewMapperFuncAdapterJava2goExecution[string, string](func(__java2goExecution *stdjava.Execution, v string) string {
		return v
	})
	normalized := func() string {
		__java2goInvocationReceiver := trim
		__java2goInvocationArg0 := input
		__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
		}(__java2goInvocationReceiver).(common.MapperJava2goExecution[string, string])
		if __java2goHasExecutionReceiver {
			return __java2goExecutionReceiver.MapJava2goExecution(__java2goExecution, __java2goInvocationArg0)
		}
		return __java2goInvocationReceiver.Map(__java2goInvocationArg0)
	}()
	if stdjava.ObjectInstanceOf(normalized, stdjava.StringTypeID) {
		return stdjava.StringLength(stdjava.StringRequireNonNull(normalized))
	}
	return 0
}
