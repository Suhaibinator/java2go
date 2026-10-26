package stdjava

// StreamFromValues creates a fresh pipeline source from the supplied values.
// It owns the outer element slice and retains the element references. Callback
// execution belongs to the eventual terminal operation's Execution.
func StreamFromValues[T any](elements ...T) Stream[T] {
	return streamFromSlice(elements)
}
