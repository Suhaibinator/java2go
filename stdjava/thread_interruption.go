package stdjava

// Interrupt requests cooperative interruption. Currently execution-aware latch
// waits observe this request; sleep, Future.get and monitor waits do not yet.
func (t *Thread) Interrupt() {
	t.interruptMu.Lock()
	defer t.interruptMu.Unlock()
	if t.interruptSignal == nil {
		t.interruptSignal = make(chan struct{})
	}
	if !t.interrupted {
		t.interrupted = true
		close(t.interruptSignal)
	}
}
func (t *Thread) interruptChannel() <-chan struct{} {
	t.interruptMu.Lock()
	defer t.interruptMu.Unlock()
	if t.interruptSignal == nil {
		t.interruptSignal = make(chan struct{})
	}
	return t.interruptSignal
}
func (t *Thread) consumeInterrupt() bool {
	t.interruptMu.Lock()
	defer t.interruptMu.Unlock()
	if !t.interrupted {
		return false
	}
	t.interrupted = false
	t.interruptSignal = make(chan struct{})
	return true
}
func init() { RegisterException("InterruptedException", "Exception") }
