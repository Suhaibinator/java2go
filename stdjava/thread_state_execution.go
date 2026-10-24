package stdjava

// Source observer callback installation is a distinct compiler owner cut.
func ThreadIsInterruptedExecution(execution *Execution, thread *Thread) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.interruptMu.Lock()
	callback := thread.isInterruptedExecution
	pending := thread.interrupted
	thread.interruptMu.Unlock()
	if callback != nil {
		return callback(execution)
	}
	return pending
}

// This observer deliberately bypasses virtual dispatch for a future super route.
func ThreadIsInterruptedDefaultExecution(execution *Execution, thread *Thread) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.interruptMu.Lock()
	defer thread.interruptMu.Unlock()
	return thread.interrupted
}

func ThreadInterruptedExecution(execution *Execution) bool {
	requireExecution(execution)
	return ThreadCurrentThread(execution).consumeInterrupt()
}

// Match the installed JDK's empty body; no fairness/wait guarantee.
func ThreadOnSpinWaitExecution(execution *Execution) { requireExecution(execution) }
