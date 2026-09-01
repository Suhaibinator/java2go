package app

type AutoCloseable interface {
	Close()
}
type AutoCloseableFuncAdapter struct {
	fn func()
}

func (fa *AutoCloseableFuncAdapter) Close() {
	fa.fn()
}
func NewAutoCloseableFuncAdapter(fn func()) AutoCloseable {
	return &AutoCloseableFuncAdapter{fn: fn}
}
