package stdjava

// ThreadSourceMethods carries only implementations selected from actual Java
// declarations by the compiler. Nil slots retain Thread's native defaults.
// The constructor Runnable target is separate from the source Run override.
type ThreadSourceMethods struct {
	Interrupt     func(*Execution)
	IsInterrupted func(*Execution) bool
	Run           func(*Execution)
	Start         func(*Execution)
}

// BindThreadSourceReference attaches source identity and callbacks to the one
// Thread constructed by the actual native super constructor. Generated code
// binds before source initialization or publication; neither identity nor slots
// may subsequently be replaced. Binding itself invokes no source callback.
func BindThreadSourceReference(thread *Thread, identity *ObjectInfo, methods ThreadSourceMethods) *Thread {
	ReferenceRequireNonNull(thread)
	if identity == nil || !JavaTypeAssignable(identity.DynamicType(), ThreadTypeID) {
		panic(NewIllegalArgumentException("source Thread requires Thread allocation identity"))
	}
	thread.interruptMu.Lock()
	defer thread.interruptMu.Unlock()
	if thread.sourceObjectInfo != nil || thread.started.Load() {
		panic(NewIllegalStateException("source Thread already bound or started"))
	}
	thread.sourceObjectInfo = identity
	thread.interruptExecution = methods.Interrupt
	thread.isInterruptedExecution = methods.IsInterrupted
	thread.sourceRunExecution = methods.Run
	thread.sourceStartExecution = methods.Start
	return thread
}

// JavaObjectInfo exposes the same allocation identity as the generated source
// hierarchy. Native-only Thread values retain their existing pointer identity.
func (t *Thread) JavaObjectInfo() *ObjectInfo {
	if t == nil {
		return nil
	}
	return t.sourceObjectInfo
}

// ThreadRunExecution dispatches with the caller token; direct run never starts
// a worker or substitutes the constructor Runnable for a source override.
func ThreadRunExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.interruptMu.Lock()
	callback := thread.sourceRunExecution
	thread.interruptMu.Unlock()
	if callback != nil {
		callback(execution)
		return
	}
	ThreadRunDefaultExecution(execution, thread)
}

// ThreadRunDefaultExecution is the actual Thread.run body for a genuine native
// super call. It executes only the constructor target, or does nothing if nil.
func ThreadRunDefaultExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	RunRunnableExecution(execution, thread.run)
}

// ThreadStartExecution preserves caller execution for an overridable start.
func ThreadStartExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	thread.interruptMu.Lock()
	callback := thread.sourceStartExecution
	thread.interruptMu.Unlock()
	if callback != nil {
		callback(execution)
		return
	}
	ThreadStartDefaultExecution(execution, thread)
}

// ThreadStartDefaultExecution keeps the native one-shot lifecycle and creates
// a fresh worker execution whose current Thread is the same canonical state.
func ThreadStartDefaultExecution(execution *Execution, thread *Thread) {
	requireExecution(execution)
	ReferenceRequireNonNull(thread)
	if !thread.started.CompareAndSwap(false, true) {
		panic(NewIllegalThreadStateException("thread already started"))
	}
	go func() {
		defer close(thread.done)
		defer reportUncaughtTaskException()
		ThreadRunExecution(&Execution{thread: thread}, thread)
	}()
}
