package common

import stdjava "github.com/NickyBoy89/java2go/stdjava"

type Mapper[T any, R any] interface {
	Map(value T) R
}
type MapperJava2goExecution[T any, R any] interface {
	MapJava2goExecution(__java2goExecution *stdjava.Execution, value T) R
}
type MapperFuncAdapter[T any, R any] struct {
	fn func(__java2goExecution *stdjava.Execution, value T) R
}

func (fa *MapperFuncAdapter[T, R]) Map(value T) R {
	return fa.MapJava2goExecution(stdjava.NewExecution(), value)
}
func (fa *MapperFuncAdapter[T, R]) MapJava2goExecution(__java2goExecution *stdjava.Execution, value T) R {
	return fa.fn(__java2goExecution, value)
}
func NewMapperFuncAdapter[T any, R any](fn func(value T) R) Mapper[T, R] {
	return &MapperFuncAdapter[T, R]{fn: func(__java2goExecution *stdjava.Execution, value T) R {
		return fn(value)
	}}
}
func NewMapperFuncAdapterJava2goExecution[T any, R any](fn func(__java2goExecution *stdjava.Execution, value T) R) Mapper[T, R] {
	return &MapperFuncAdapter[T, R]{fn: fn}
}
