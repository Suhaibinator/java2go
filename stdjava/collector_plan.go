package stdjava

// CollectorPlan is an internal lowering recipe, not a Java Collector reference.
// All mutable accumulation belongs to one terminal invocation, never this plan.
// Its A transport is implemented only by typed, runtime-owned closures.
type CollectorPlan[T, R any] struct {
	newState       func(*Execution) any
	accumulate     func(*Execution, any, T)
	combine        func(*Execution, any, any) any
	finish         func(*Execution, any) R
	identityFinish bool
}

// view must implement this recipe's actual declaration-owned A boundary.
// Observable reference states use their Java descriptor/view (e.g. int[]),
// rather than accepting a value just because its Go shape matches.
func newCollectorPlan[T, A, R any](
	supplier func(*Execution) A,
	accumulator func(*Execution, A, T),
	combiner func(*Execution, A, A) A,
	finisher func(*Execution, A) R,
	view func(any) A,
	identityFinish bool,
) CollectorPlan[T, R] {
	return CollectorPlan[T, R]{
		newState: func(e *Execution) any { return supplier(e) },
		accumulate: func(e *Execution, value any, item T) {
			accumulator(e, view(value), item)
		},
		combine: func(e *Execution, left, right any) any {
			return combiner(e, view(left), view(right))
		},
		finish:         func(e *Execution, value any) R { return finisher(e, view(value)) },
		identityFinish: identityFinish,
	}
}

type collectorPlanSink[T, R any] struct {
	execution *Execution
	plan      CollectorPlan[T, R]
	state     any
	begun     bool
	ended     bool
}

func (sink *collectorPlanSink[T, R]) Begin(int64) {
	if sink.begun {
		panic(NewUnsupportedOperationException("collector sink began twice"))
	}
	sink.begun = true
	sink.state = sink.plan.newState(sink.execution)
}
func (sink *collectorPlanSink[T, R]) Accept(item T) {
	sink.plan.accumulate(sink.execution, sink.state, item)
}
func (*collectorPlanSink[T, R]) CancellationRequested() bool { return false }
func (sink *collectorPlanSink[T, R]) End()                   { sink.ended = true }

// Core prerequisites, owned by Stream:
//
//	type streamSink[T any] interface {
//	    Begin(int64); Accept(T); CancellationRequested() bool; End()
//	}
//
// func streamClaimAndEvaluate[T any](*Execution, Stream[T], func() streamSink[T])
// It claims before factory, invokes factory to prepare the sink before binding,
// then acquires the source/exact size before Begin invokes the actual supplier.
// End occurs only on success; a binding/supplier/traversal panic stays consumed.
func StreamCollectPlanExecution[T, R any](e *Execution, source Stream[T], plan CollectorPlan[T, R]) R {
	requireExecution(e)
	if plan.newState == nil || plan.accumulate == nil || plan.finish == nil {
		panic(NewUnsupportedOperationException("uninitialized internal collector recipe"))
	}
	var terminal *collectorPlanSink[T, R]
	streamClaimAndEvaluate(e, source, func() streamSink[T] {
		terminal = &collectorPlanSink[T, R]{execution: e, plan: plan}
		return terminal
	})
	if terminal == nil || !terminal.ended {
		panic(NewUnsupportedOperationException("collector evaluation did not complete its sink"))
	}
	return plan.finish(e, terminal.state)
}
