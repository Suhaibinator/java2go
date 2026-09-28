package stdjava

// ThreadHoldsLockExecution observes ownership of the Java logical execution,
// independent of which native thread currently runs it.
func ThreadHoldsLockExecution(execution *Execution, value any) bool {
	requireExecution(execution)
	requireNonNullMonitorReference(value, "holdsLock")
	monitor := monitorRecord(value)
	monitor.mu.Lock()
	defer monitor.mu.Unlock()
	return monitor.owner == execution && monitor.depth > 0
}
