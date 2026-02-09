package common

type Mapper[T any, R any] interface {
	Map(value T) R
}
type MapperFuncAdapter[T any, R any] struct {
	fn func(value T) R
}

func (fa *MapperFuncAdapter[T, R]) Map(value T) R {
	return fa.fn(value)
}
func NewMapperFuncAdapter[T any, R any](fn func(value T) R) Mapper[T, R] {
	return &MapperFuncAdapter[T, R]{fn: fn}
}
