package stdjava

// The final Throwable body shares synchronized(primary)'s actual Java monitor.
// Arguments have already been evaluated by the generated invocation boundary.
func JavaThrowableAddSuppressedExecution(execution *Execution, primary, suppressed any) {
	ReferenceRequireNonNull(primary)
	guard := MonitorEnterExecution(execution, primary)
	defer MonitorExitExecution(guard)
	if !javaReferenceIsNull(suppressed) && sameThrowableIdentity(primary, suppressed) {
		panic(NewJavaIllegalArgumentExceptionMessageCause(JavaStringLiteralUTF16([]uint16{83, 101, 108, 102, 45, 115, 117, 112, 112, 114, 101, 115, 115, 105, 111, 110, 32, 110, 111, 116, 32, 112, 101, 114, 109, 105, 116, 116, 101, 100}), primary))
	}
	if javaReferenceIsNull(suppressed) {
		panic(NewJavaNullPointerExceptionMessage(JavaStringLiteralUTF16([]uint16{67, 97, 110, 110, 111, 116, 32, 115, 117, 112, 112, 114, 101, 115, 115, 32, 97, 32, 110, 117, 108, 108, 32, 101, 120, 99, 101, 112, 116, 105, 111, 110, 46})))
	}
	appendThrowableSuppressedState(primary, suppressed)
}

// Final Throwable suppression writes use a runtime-owned carrier. A source
// business method named addSuppressedValue must remain ordinary Java code.
func appendThrowableSuppressedState(primary, suppressed any) {
	carrier, ok := primary.(interface{ throwableCauseState() *throwableState })
	if !ok {
		return
	}
	state := carrier.throwableCauseState()
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	state.suppressed = append(state.suppressed, suppressed)
}
