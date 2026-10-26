package stdjava

// The pipeline core is additive. Legacy slice streams keep their existing API;
// translated adoption must select initialized sources and migrate all readers.
// One pipeline is sequential and is not safe for concurrent terminal/close use.
type streamSink[T any] interface {
	Begin(int64)
	Accept(T)
	CancellationRequested() bool
	End()
}

type streamSourceState struct {
	claimed       bool
	closed        bool
	closeHandlers []func(*Execution)
	clearSource   func()
}

func streamReleaseSource(source *streamSourceState) {
	clear := source.clearSource
	source.clearSource = nil
	if clear != nil {
		clear()
	}
}

type streamStageState struct {
	linkedOrConsumed bool
	source           *streamSourceState
}

type streamEvaluation struct {
	execution *Execution
	bindings  map[*streamSourceState]any
}

type streamNode[T any] struct {
	state     *streamStageState
	exactSize func(*streamEvaluation) (int64, bool)
	drive     func(*streamEvaluation, streamSink[T])
}

// A bound source pushes elements only. The source node owns Begin and End.
type streamBoundSource[T any] struct {
	exactSize int64 // -1 means unknown, without probing callbackful iteration.
	drive     func(*streamEvaluation, streamSink[T])
}

func streamNodeFor[T any](s Stream[T]) *streamNode[T] {
	if s.pipeline != nil {
		return s.pipeline
	}
	if len(s.elements) != 0 {
		panic(NewUnsupportedOperationException("legacy eager Stream requires an initialized pipeline source"))
	}
	// Native zero empty has no pointer to share. Each use remains untracked;
	// every core-created source, including empty sources, has a real node.
	return streamFromSlice[T](nil).pipeline
}

func streamRequireAvailable(state *streamStageState) {
	if state.linkedOrConsumed || state.source.claimed || state.source.closed {
		panic(NewIllegalStateException("stream has already been operated upon or closed"))
	}
}

func streamClaim[T any](e *Execution, s Stream[T]) (*streamNode[T], *streamEvaluation) {
	requireExecution(e)
	node := streamNodeFor(s)
	streamRequireAvailable(node.state)
	// Claim before creating any collector container or binding the source.
	node.state.linkedOrConsumed = true
	node.state.source.claimed = true
	return node, &streamEvaluation{execution: e, bindings: make(map[*streamSourceState]any)}
}

// Reader prepares its CollectorPlan sink in terminalFactory. Actual accumulator
// allocation belongs in sink.Begin, after binding and size acquisition. Stateful
// sinks such as sorted can delay downstream Begin until their End. If setup or
// traversal panics, the source stays consumed; End is not invented.
func streamClaimAndEvaluate[T any](e *Execution, s Stream[T], terminalFactory func() streamSink[T]) {
	node, evaluation := streamClaim(e, s)
	if terminalFactory == nil {
		panic(NewNullPointerException("nil stream terminal factory"))
	}
	sink := terminalFactory()
	ReferenceRequireNonNull(sink)
	streamEvaluateClaimed(evaluation, node, sink)
}

func streamEvaluateClaimed[T any](evaluation *streamEvaluation, node *streamNode[T], sink streamSink[T]) {
	node.drive(evaluation, sink)
}

func streamExactSize[T any](evaluation *streamEvaluation, node *streamNode[T]) (int64, bool) {
	return node.exactSize(evaluation)
}

// Link first, then build: a build failure intentionally does not restore the
// predecessor, which is observable for Java sorted(null). Stateless operations
// validate their nullable argument before invoking this helper.
func streamLinkNode[T, R any](s Stream[T], build func(*streamNode[T], *streamStageState) *streamNode[R]) Stream[R] {
	parent := streamNodeFor(s)
	streamRequireAvailable(parent.state)
	parent.state.linkedOrConsumed = true
	state := &streamStageState{source: parent.state.source}
	return Stream[R]{pipeline: build(parent, state)}
}

type streamCollectSink[T any] struct{ values []T }

func (s *streamCollectSink[T]) Begin(int64)                 {}
func (s *streamCollectSink[T]) Accept(value T)              { s.values = append(s.values, value) }
func (s *streamCollectSink[T]) CancellationRequested() bool { return false }
func (s *streamCollectSink[T]) End()                        {}

func streamToSliceExecution[T any](e *Execution, s Stream[T]) []T {
	sink := &streamCollectSink[T]{}
	streamClaimAndEvaluate(e, s, func() streamSink[T] { return sink })
	return sink.values
}

type streamCountSink[T any] struct{ count int64 }

func (s *streamCountSink[T]) Begin(int64)                 {}
func (s *streamCountSink[T]) Accept(T)                    { s.count++ }
func (s *streamCountSink[T]) CancellationRequested() bool { return false }
func (s *streamCountSink[T]) End()                        {}

// Only Stream.count uses exact-size elision. Collectors.counting must supply a
// counting sink to streamClaimAndEvaluate and traverse even a sized source.
func streamCountExecution[T any](e *Execution, s Stream[T]) int64 {
	node, evaluation := streamClaim(e, s)
	if size, known := streamExactSize(evaluation, node); known {
		return size
	}
	sink := &streamCountSink[T]{}
	streamEvaluateClaimed(evaluation, node, sink)
	return sink.count
}
