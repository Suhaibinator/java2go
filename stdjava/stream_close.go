package stdjava

func streamOnClose[T any](s Stream[T], action func(*Execution)) Stream[T] {
	node := streamNodeFor(s)
	streamRequireAvailable(node.state)
	if action == nil {
		panic(NewNullPointerException("nil stream close handler"))
	}
	node.state.source.closeHandlers = append(node.state.source.closeHandlers, action)
	return Stream[T]{pipeline: node}
}

func streamCloseExecution[T any](e *Execution, s Stream[T]) {
	requireExecution(e)
	node := streamNodeFor(s)
	source := node.state.source
	node.state.linkedOrConsumed = true
	if source.closed {
		return
	}
	source.closed = true
	streamReleaseSource(source)
	handlers := source.closeHandlers
	source.closeHandlers = nil
	var primary any
	for _, action := range handlers {
		var failed any
		func() {
			defer func() { failed = recover() }()
			action(e)
		}()
		if failed != nil {
			failed = NormalizePanic(failed)
			if primary == nil {
				primary = failed
			} else {
				// Streams.composeWithExceptions preserves the original failure
				// even if suppression itself fails (including self suppression).
				func() {
					defer func() { _ = recover() }()
					AddSuppressed(primary, failed)
				}()
			}
		}
	}
	if primary != nil {
		panic(primary)
	}
}
