package stdjava

import (
	"reflect"
	"testing"
)

func collectorRuntimePanic(t *testing.T, name string, fn func()) any {
	t.Helper()
	var failure any
	func() {
		defer func() { failure = recover() }()
		fn()
	}()
	if failure == nil || !CaughtAs(failure, name) {
		t.Fatalf("want %s, got %T: %v", name, failure, failure)
	}
	return failure
}

func collectorRuntimeConsumed(t *testing.T, source Stream[int32], e *Execution) {
	t.Helper()
	collectorRuntimePanic(t, "IllegalStateException", func() { streamCountExecution(e, source) })
}

func TestCollectorPlanBindingAndTerminalExecution(t *testing.T) {
	e := NewExecution()
	var events []string
	plan := newCollectorPlan(
		func(got *Execution) *int32 {
			if got != e {
				t.Fatal("supplier lost operation Execution")
			}
			events = append(events, "A")
			return new(int32)
		},
		func(got *Execution, state *int32, value int32) {
			if got != e {
				t.Fatal("accumulator lost operation Execution")
			}
			events = append(events, "accept")
			*state += value
		},
		func(_ *Execution, left, right *int32) *int32 { *left += *right; return left },
		func(got *Execution, state *int32) int32 {
			if got != e {
				t.Fatal("finisher lost operation Execution")
			}
			events = append(events, "finish")
			return *state
		},
		func(value any) *int32 { return value.(*int32) }, false,
	)
	source := streamFromSource(func(got *Execution) streamBoundSource[int32] {
		if got != e {
			t.Fatal("source lost operation Execution")
		}
		events = append(events, "bind", "size")
		return streamBoundSource[int32]{1, func(_ *streamEvaluation, sink streamSink[int32]) {
			events = append(events, "pull")
			sink.Accept(7)
		}}
	})
	closed := 0
	source = streamOnClose(source, func(got *Execution) {
		if got != e {
			t.Fatal("close lost operation Execution")
		}
		closed++
	})
	if len(events) != 0 {
		t.Fatal("recipe construction executed callbacks", events)
	}
	if got := StreamCollectPlanExecution(e, source, plan); got != 7 {
		t.Fatal(got)
	}
	want := []string{"bind", "size", "A", "pull", "accept", "finish"}
	if !reflect.DeepEqual(events, want) {
		t.Fatal(events, want)
	}
	if closed != 0 {
		t.Fatal("collect implicitly closed outer stream")
	}
	collectorRuntimeConsumed(t, source, e)
	streamCloseExecution(e, source)
	streamCloseExecution(e, source)
	if closed != 1 {
		t.Fatal("close handler count", closed)
	}
}

func TestCollectorPlanFailuresPreserveClaimAndAllocationOrder(t *testing.T) {
	for _, phase := range []string{"bind", "size", "supplier", "accumulator", "finisher"} {
		t.Run(phase, func(t *testing.T) {
			e := NewExecution()
			marker := NewIllegalArgumentException("collector-" + phase)
			allocations, accepts, finishes := 0, 0, 0
			plan := newCollectorPlan(
				func(*Execution) *int32 {
					allocations++
					if phase == "supplier" {
						panic(marker)
					}
					return new(int32)
				},
				func(_ *Execution, state *int32, item int32) {
					accepts++
					if phase == "accumulator" {
						panic(marker)
					}
					*state += item
				},
				func(_ *Execution, left, _ *int32) *int32 { return left },
				func(_ *Execution, state *int32) int32 {
					finishes++
					if phase == "finisher" {
						panic(marker)
					}
					return *state
				},
				func(value any) *int32 { return value.(*int32) }, false,
			)
			source := streamFromSource(func(*Execution) streamBoundSource[int32] {
				if phase == "bind" || phase == "size" {
					panic(marker)
				}
				return streamBoundSource[int32]{1, func(_ *streamEvaluation, sink streamSink[int32]) { sink.Accept(1) }}
			})
			got := collectorRuntimePanic(t, "IllegalArgumentException", func() { StreamCollectPlanExecution(e, source, plan) })
			if !sameThrowableIdentity(got, marker) {
				t.Fatal("failure identity changed")
			}
			if phase == "bind" || phase == "size" {
				if allocations != 0 {
					t.Fatal("allocated A before source binding/size succeeded")
				}
			} else if allocations != 1 {
				t.Fatal("supplier count", allocations)
			}
			if phase == "supplier" && accepts != 0 {
				t.Fatal("accepted after supplier failed")
			}
			if phase != "finisher" && finishes != 0 {
				t.Fatal("finisher ran after failed evaluation")
			}
			collectorRuntimeConsumed(t, source, e)
		})
	}
}

func TestCollectorPlanSortedDelaysSupplierAndDoesNotElideTraversal(t *testing.T) {
	e := NewExecution()
	var events []string
	plan := newCollectorPlan(
		func(*Execution) *int32 { events = append(events, "A"); return new(int32) },
		func(_ *Execution, state *int32, item int32) { *state += item },
		func(_ *Execution, left, _ *int32) *int32 { return left },
		func(_ *Execution, state *int32) int32 { return *state },
		func(value any) *int32 { return value.(*int32) }, false,
	)
	source := streamPeek(streamFromSlice([]int32{2, 1}), func(got *Execution, _ int32) {
		if got != e {
			t.Fatal("peek Execution")
		}
		events = append(events, "peek")
	})
	source = streamSorted(source, func(got *Execution, a, b int32) int32 {
		if got != e {
			t.Fatal("comparison Execution")
		}
		events = append(events, "compare")
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	if got := StreamCollectPlanExecution(e, source, plan); got != 3 {
		t.Fatal(got)
	}
	if len(events) < 4 || events[0] != "peek" || events[1] != "peek" || events[len(events)-1] != "A" {
		t.Fatal("supplier was not delayed by sorted", events)
	}
	seen := 0
	plain := streamPeek(streamFromSlice([]int32{1, 2}), func(*Execution, int32) { seen++ })
	if got := StreamCollectPlanExecution(e, plain, plan); got != 3 || seen != 2 {
		t.Fatal("collector used sized-count shortcut", got, seen)
	}
}
