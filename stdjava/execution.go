package stdjava

// Execution identifies one logical Java thread while it evaluates generated
// code. Runtime operations that have Java-thread-relative semantics propagate
// this token explicitly instead of trying to infer a goroutine identifier from
// unsupported runtime internals.
//
// The token is intentionally opaque to generated programs. It owns reentrant
// monitors today and is also the natural home for other per-Java-thread state,
// such as recursive class-initialization tracking. Keeping it non-zero-sized
// guarantees that distinct live executions have distinct pointer identities.
type Execution struct {
	_ byte
}

// NewExecution starts an independent logical Java execution.
func NewExecution() *Execution {
	return &Execution{}
}
