package domain

import stdjava "github.com/NickyBoy89/java2go/stdjava"

type TaskJava2goDispatch interface {
	Name() string
	NameJava2goExecution(__java2goExecution *stdjava.Execution) string
	Run(input string) int32
	RunJava2goExecution(__java2goExecution *stdjava.Execution, input string) int32
}
type Task struct {
	Java2goTaskSelf TaskJava2goDispatch
	*stdjava.ObjectInfo
	id string
}

var __java2goReferenceTypeRegistrationTask = func() bool {
	stdjava.RegisterJavaType(stdjava.TypeID("com.acme.domain.Task"), stdjava.ObjectTypeID)
	return true
}()

func (tk *Task) Java2goReferenceDynamicType() stdjava.TypeID {
	return stdjava.TypeID("com.acme.domain.Task")
}
func (tk *Task) Java2goReferenceView(__java2goRequestedType stdjava.TypeID) any {
	switch __java2goRequestedType {
	case stdjava.TypeID("com.acme.domain.Task"):
		return tk
	default:
		return nil
	}
}
func (tk *Task) Java2goSetTaskSelf(self any) {
	tk.Java2goTaskSelf = self.(TaskJava2goDispatch)
}

type TaskI interface {
	Name() string
	Run(input string) int32
}
type TaskJava2goExecution interface {
	NameJava2goExecution(__java2goExecution *stdjava.Execution) string
	RunJava2goExecution(__java2goExecution *stdjava.Execution, input string) int32
}

func NewTask(id string) *Task {
	return NewTaskJava2goExecution(stdjava.NewExecution(), id)
}
func NewTaskJava2goExecution(__java2goExecution *stdjava.Execution, id string) *Task {
	return NewTaskJava2goWithSelfJava2goExecution(__java2goExecution, nil, id)
}
func NewTaskJava2goWithSelf(__java2goMostDerived any, id string) *Task {
	return NewTaskJava2goWithSelfJava2goExecution(stdjava.NewExecution(), __java2goMostDerived, id)
}
func NewTaskJava2goWithSelfJava2goExecution(__java2goExecution *stdjava.Execution, __java2goMostDerived any, id string) *Task {
	tk := new(Task)
	tk.id = "\xffjava2go:null-string\x00"
	if __java2goMostDerived == nil {
		__java2goMostDerived = tk
	}
	if __java2goSubobjectInstaller, __java2goHasSubobjectInstaller := __java2goMostDerived.(interface {
		Java2goInstallTaskSubobjectFrom636F6D2E61636D652E646F6D61696E2E5461736B(*Task)
	}); __java2goHasSubobjectInstaller {
		__java2goSubobjectInstaller.Java2goInstallTaskSubobjectFrom636F6D2E61636D652E646F6D61696E2E5461736B(tk)
	}
	tk.ObjectInfo = stdjava.NewGeneratedObjectInfo(__java2goMostDerived)
	tk.Java2goSetTaskSelf(__java2goMostDerived)
	tk.id = id
	return tk
}
func (tk *Task) Name() string {
	return tk.NameJava2goExecution(stdjava.NewExecution())
}
func (tk *Task) NameJava2goExecution(__java2goExecution *stdjava.Execution) string {
	if tk == nil {
		_ = *tk
	}
	return tk.id
}
func (tk *Task) Run(input string) int32 {
	return tk.RunJava2goExecution(stdjava.NewExecution(), input)
}
func (tk *Task) RunJava2goExecution(__java2goExecution *stdjava.Execution, input string) int32 {
	panic("abstract method run not implemented")
	return 0
}
