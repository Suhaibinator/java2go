package stdjava

// ThreadInterruptExecution enters the receiver's interrupt implementation with
// the caller's Java execution. Source callback installation is a separate
// compiler prerequisite; native Thread values use the existing default request.
func ThreadInterruptExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.interruptMu.Lock()
	interrupt := thread.interruptExecution
	thread.interruptMu.Unlock()
	if interrupt != nil {
		// An override runs on every invocation, even if the default flag was already
		// set. In particular, an override which throws or omits super must not alter
		// the flag implicitly. Never hold the native lock across Java code.
		interrupt(execution)
		return
	}
	thread.Interrupt()
}

// ThreadInterruptDefaultExecution invokes Thread's own implementation without
// virtual callback dispatch, for an eventual canonical super.interrupt route.
func ThreadInterruptDefaultExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.Interrupt()
}
