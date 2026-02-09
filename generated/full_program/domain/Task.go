package domain

type Task struct {
	id string
}
type TaskI interface {
	Name() string
	Run(input string) int32
}

func NewTask(id string) *Task {
	tk := new(Task)
	tk.id = id
	return tk
}
func (tk *Task) Name() string {
	return tk.id
}
func (tk *Task) Run(input string) int32 {
	panic("abstract method run not implemented")
	return 0
}
