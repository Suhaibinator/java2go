package stdjava

import "sort"

type streamSinkFuncs[T any] struct {
	begin        func(int64)
	accept       func(T)
	cancel       func() bool
	end          func()
	shortCircuit bool
}

// Optional internal metadata leaves Reader's four-method sink seam unchanged.
// Source fail-fast timing distinguishes bulk traversal from tryAdvance loops.
type streamShortCircuitMode interface{ streamShortCircuiting() bool }

func streamSinkShortCircuit[T any](sink streamSink[T]) bool {
	if mode, ok := sink.(streamShortCircuitMode); ok {
		return mode.streamShortCircuiting()
	}
	return false
}
func (s *streamSinkFuncs[T]) streamShortCircuiting() bool { return s.shortCircuit }

func (s *streamSinkFuncs[T]) Begin(size int64) {
	if s.begin != nil {
		s.begin(size)
	}
}
func (s *streamSinkFuncs[T]) Accept(value T)              { s.accept(value) }
func (s *streamSinkFuncs[T]) CancellationRequested() bool { return s.cancel != nil && s.cancel() }
func (s *streamSinkFuncs[T]) End() {
	if s.end != nil {
		s.end()
	}
}

func streamMap[T, R any](s Stream[T], mapper func(*Execution, T) R) Stream[R] {
	if mapper == nil {
		panic(NewNullPointerException("nil stream mapper"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[R] {
		return &streamNode[R]{state: state, exactSize: parent.exactSize,
			drive: func(evaluation *streamEvaluation, sink streamSink[R]) {
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: streamSinkShortCircuit(sink),
					begin:        sink.Begin,
					accept:       func(value T) { sink.Accept(mapper(evaluation.execution, value)) },
					cancel:       sink.CancellationRequested, end: sink.End,
				})
			}}
	})
}

func streamFilter[T any](s Stream[T], predicate func(*Execution, T) bool) Stream[T] {
	if predicate == nil {
		panic(NewNullPointerException("nil stream predicate"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[T] {
		return &streamNode[T]{state: state,
			exactSize: func(*streamEvaluation) (int64, bool) { return -1, false },
			drive: func(evaluation *streamEvaluation, sink streamSink[T]) {
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: streamSinkShortCircuit(sink),
					begin:        func(int64) { sink.Begin(-1) },
					accept: func(value T) {
						if predicate(evaluation.execution, value) {
							sink.Accept(value)
						}
					},
					cancel: sink.CancellationRequested, end: sink.End,
				})
			}}
	})
}

func streamPeek[T any](s Stream[T], action func(*Execution, T)) Stream[T] {
	if action == nil {
		panic(NewNullPointerException("nil stream action"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[T] {
		return &streamNode[T]{state: state, exactSize: parent.exactSize,
			drive: func(evaluation *streamEvaluation, sink streamSink[T]) {
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: streamSinkShortCircuit(sink),
					begin:        sink.Begin,
					accept:       func(value T) { action(evaluation.execution, value); sink.Accept(value) },
					cancel:       sink.CancellationRequested, end: sink.End,
				})
			}}
	})
}

func streamLimitedSize(size, limit int64) int64 {
	if size < 0 {
		return -1
	}
	if size > limit {
		return limit
	}
	return size
}

func streamLimit[T any](s Stream[T], n int64) Stream[T] {
	if n < 0 {
		panic(NewIllegalArgumentException("negative stream limit"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[T] {
		return &streamNode[T]{state: state,
			exactSize: func(evaluation *streamEvaluation) (int64, bool) {
				size, known := parent.exactSize(evaluation)
				return streamLimitedSize(size, n), known
			},
			drive: func(evaluation *streamEvaluation, sink streamSink[T]) {
				remaining := n
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: true,
					begin:        func(size int64) { sink.Begin(streamLimitedSize(size, n)) },
					accept: func(value T) {
						if remaining > 0 {
							remaining--
							sink.Accept(value)
						}
					},
					cancel: func() bool { return remaining == 0 || sink.CancellationRequested() },
					end:    sink.End,
				})
			}}
	})
}

func streamSkippedSize(size, skip int64) int64 {
	if size < 0 {
		return -1
	}
	if skip >= size {
		return 0
	}
	return size - skip
}

func streamSkip[T any](s Stream[T], n int64) Stream[T] {
	if n < 0 {
		panic(NewIllegalArgumentException("negative stream skip"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[T] {
		return &streamNode[T]{state: state,
			exactSize: func(evaluation *streamEvaluation) (int64, bool) {
				size, known := parent.exactSize(evaluation)
				return streamSkippedSize(size, n), known
			},
			drive: func(evaluation *streamEvaluation, sink streamSink[T]) {
				remaining := n
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: streamSinkShortCircuit(sink),
					begin:        func(size int64) { sink.Begin(streamSkippedSize(size, n)) },
					accept: func(value T) {
						if remaining > 0 {
							remaining--
							return
						}
						sink.Accept(value)
					},
					cancel: sink.CancellationRequested, end: sink.End,
				})
			}}
	})
}

// This is a deferred stable barrier, not a claim of JDK's exact comparator
// invocation trace. That algorithm/trace gate remains outstanding for adoption.
func streamSorted[T any](s Stream[T], compare func(*Execution, T, T) int32) Stream[T] {
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[T] {
		// Java sorted(Comparator) validates after the parent stage was linked.
		if compare == nil {
			panic(NewNullPointerException("nil stream comparator"))
		}
		return &streamNode[T]{state: state, exactSize: parent.exactSize,
			drive: func(evaluation *streamEvaluation, sink streamSink[T]) {
				var values []T
				parent.drive(evaluation, &streamSinkFuncs[T]{
					// Sorted masks cancellation but retains the pipeline's global
					// short-circuit traversal mode for source revision checks.
					shortCircuit: streamSinkShortCircuit(sink),
					accept:       func(value T) { values = append(values, value) },
					// Sorting masks downstream cancellation during accumulation.
					cancel: func() bool { return false },
					end: func() {
						sort.SliceStable(values, func(i, j int) bool { return compare(evaluation.execution, values[i], values[j]) < 0 })
						sink.Begin(int64(len(values)))
						for _, value := range values {
							if sink.CancellationRequested() {
								break
							}
							sink.Accept(value)
						}
						sink.End()
					},
				})
			}}
	})
}

// This typed native mapper has no Java-null Stream representation. Java flatMap
// adoption needs Reader's nullable source adapter; zero native Stream is empty.
func streamFlatMap[T, R any](s Stream[T], mapper func(*Execution, T) Stream[R]) Stream[R] {
	if mapper == nil {
		panic(NewNullPointerException("nil stream flat mapper"))
	}
	return streamLinkNode(s, func(parent *streamNode[T], state *streamStageState) *streamNode[R] {
		return &streamNode[R]{state: state,
			exactSize: func(*streamEvaluation) (int64, bool) { return -1, false },
			drive: func(evaluation *streamEvaluation, sink streamSink[R]) {
				parent.drive(evaluation, &streamSinkFuncs[T]{
					shortCircuit: streamSinkShortCircuit(sink),
					begin:        func(int64) { sink.Begin(-1) },
					accept: func(value T) {
						inner := mapper(evaluation.execution, value)
						func() {
							defer CloseResource(func() { streamCloseExecution(evaluation.execution, inner) })
							streamClaimAndEvaluate(evaluation.execution, inner, func() streamSink[R] {
								return &streamSinkFuncs[R]{accept: sink.Accept, cancel: sink.CancellationRequested, shortCircuit: streamSinkShortCircuit(sink)}
							})
						}()
					},
					cancel: sink.CancellationRequested, end: sink.End,
				})
			}}
	})
}
