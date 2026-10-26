package stdjava

import (
	"reflect"
	"testing"
)

func streamPhase1Recover(fn func()) (p any) {
	defer func() { p = recover() }()
	fn()
	return nil
}

func streamPhase1Consumed(t *testing.T, s Stream[int32], e *Execution) {
	t.Helper()
	if p := streamPhase1Recover(func() { streamCountExecution(e, s) }); p == nil {
		t.Fatal("consumed stream remained usable")
	} else if _, ok := p.(IllegalStateException); !ok {
		t.Fatalf("consumed stream threw %T, want IllegalStateException", p)
	}
}

func TestStreamPhase1ClaimFactoryBindBeginOrder(t *testing.T) {
	e := NewExecution()
	var events []string
	s := streamFromSource(func(got *Execution) streamBoundSource[int32] {
		if got != e {
			t.Fatal("binding execution")
		}
		events = append(events, "bind", "size")
		return streamBoundSource[int32]{1, func(_ *streamEvaluation, sink streamSink[int32]) {
			events = append(events, "pull")
			sink.Accept(7)
		}}
	})
	streamClaimAndEvaluate(e, s, func() streamSink[int32] {
		if !s.pipeline.state.linkedOrConsumed || !s.pipeline.state.source.claimed {
			t.Fatal("factory preceded claim")
		}
		events = append(events, "sink-factory")
		return &streamSinkFuncs[int32]{
			begin: func(size int64) {
				if size != 1 {
					t.Fatal(size)
				}
				events = append(events, "actual-A")
			},
			accept: func(value int32) {
				if value != 7 {
					t.Fatal(value)
				}
				events = append(events, "accumulate")
			},
			end: func() { events = append(events, "end") },
		}
	})
	want := []string{"sink-factory", "bind", "size", "actual-A", "pull", "accumulate", "end"}
	if !reflect.DeepEqual(events, want) {
		t.Fatal(events, want)
	}
	streamPhase1Consumed(t, s, e)
}

func TestStreamPhase1SetupAndBindingFailures(t *testing.T) {
	for _, phase := range []string{"factory", "bind", "size", "begin", "accept", "end"} {
		t.Run(phase, func(t *testing.T) {
			e := NewExecution()
			marker := NewIllegalStateException(phase)
			var events []string
			s := streamFromSource(func(*Execution) streamBoundSource[int32] {
				events = append(events, "bind")
				if phase == "bind" {
					panic(marker)
				}
				events = append(events, "size")
				if phase == "size" {
					panic(marker)
				}
				return streamBoundSource[int32]{1, func(_ *streamEvaluation, sink streamSink[int32]) { events = append(events, "pull"); sink.Accept(1) }}
			})
			got := streamPhase1Recover(func() {
				streamClaimAndEvaluate(e, s, func() streamSink[int32] {
					events = append(events, "factory")
					if phase == "factory" {
						panic(marker)
					}
					return &streamSinkFuncs[int32]{
						begin: func(int64) {
							events = append(events, "actual-A")
							if phase == "begin" {
								panic(marker)
							}
						},
						accept: func(int32) {
							events = append(events, "accept")
							if phase == "accept" {
								panic(marker)
							}
						},
						end: func() {
							events = append(events, "end")
							if phase == "end" {
								panic(marker)
							}
						},
					}
				})
			})
			if !sameThrowableIdentity(got, marker) {
				t.Fatalf("lost failure %T: %v", got, events)
			}
			if phase == "factory" && !reflect.DeepEqual(events, []string{"factory"}) {
				t.Fatal(events)
			}
			if phase == "bind" || phase == "size" {
				for _, event := range events {
					if event == "actual-A" || event == "pull" {
						t.Fatal("allocated/pulled after source failure", events)
					}
				}
			}
			if phase != "end" {
				for _, event := range events {
					if event == "end" {
						t.Fatal("invented successful end", events)
					}
				}
			}
			streamPhase1Consumed(t, s, e)
		})
	}
}

func TestStreamPhase1StageLinkingAndNullTiming(t *testing.T) {
	e := NewExecution()
	parent := streamFromSlice([]int32{1, 2})
	child := streamFilter(parent, func(*Execution, int32) bool { return true })
	streamPhase1Consumed(t, parent, e)
	copyOfChild := child
	if streamCountExecution(e, child) != 2 {
		t.Fatal("child lost source")
	}
	streamPhase1Consumed(t, copyOfChild, e)
	sortedSource := streamFromSlice([]int32{1})
	if p := streamPhase1Recover(func() { _ = streamSorted(sortedSource, nil) }); p == nil {
		t.Fatal("nil comparator accepted")
	}
	streamPhase1Consumed(t, sortedSource, e)
	for _, name := range []string{"map", "filter", "peek", "limit", "skip"} {
		t.Run(name, func(t *testing.T) {
			s := streamFromSlice([]int32{1})
			p := streamPhase1Recover(func() {
				switch name {
				case "map":
					_ = streamMap[int32, string](s, nil)
				case "filter":
					_ = streamFilter(s, nil)
				case "peek":
					_ = streamPeek(s, nil)
				case "limit":
					_ = streamLimit(s, -1)
				case "skip":
					_ = streamSkip(s, -1)
				}
			})
			if p == nil {
				t.Fatal("invalid argument accepted")
			}
			if streamCountExecution(e, s) != 1 {
				t.Fatal("pre-link validation consumed source")
			}
		})
	}
}

func TestStreamPhase1LiveSources(t *testing.T) {
	e := NewExecution()
	list := NewListFrom[int32](1, 2)
	s := streamFromIterable[int32](list)
	list.Add(3)
	mapped := streamMap(s, func(_ *Execution, v int32) int32 {
		if v == 1 {
			list.Set(1, 9)
		}
		return v
	})
	if got := streamToSliceExecution(e, mapped); !reflect.DeepEqual(got, []int32{1, 9, 3}) {
		t.Fatal(got)
	}
	array := PrimitiveArrayLiteral(PrimitiveTypeID("int"), int32(1), int32(2))
	a := streamFromArray[int32](array)
	array.Elements[0] = 7
	a = streamMap(a, func(_ *Execution, v int32) int32 {
		if v == 7 {
			array.Elements[1] = 8
		}
		return v
	})
	if got := streamToSliceExecution(e, a); !reflect.DeepEqual(got, []int32{7, 8}) {
		t.Fatal(got)
	}
	ref := NewReferenceArray(2, ObjectTypeID)
	one, two, changed := BoxInteger(1), BoxInteger(2), BoxInteger(3)
	ReferenceArraySet(ref, 0, one)
	ReferenceArraySet(ref, 1, two)
	r := streamFromArray[*Integer](ref)
	ReferenceArraySet(ref, 0, changed)
	if got := streamToSliceExecution(e, r); len(got) != 2 || got[0] != changed || got[1] != two {
		t.Fatal(got)
	}
	var nilArray *PrimitiveArray[int32]
	if p := streamPhase1Recover(func() { _ = streamFromArray[int32](nilArray) }); p == nil {
		t.Fatal("typed null array accepted")
	}
}

type streamPhase1Pull struct {
	values                                         []int32
	index, factories, hasNext, next, slices, sizes int
	execution                                      *Execution
}

func (s *streamPhase1Pull) Slice() []int32 { s.slices++; panic("callbackful Slice must not run") }
func (s *streamPhase1Pull) Size() int32    { s.sizes++; panic("custom Size must not run") }
func (s *streamPhase1Pull) IteratorJava2goExecution(e *Execution) JavaIterator {
	s.factories++
	s.execution = e
	return s
}
func (s *streamPhase1Pull) HasNextJava2goExecution(e *Execution) bool {
	if e != s.execution {
		panic("wrong execution")
	}
	s.hasNext++
	return s.index < len(s.values)
}
func (s *streamPhase1Pull) NextJava2goExecution(e *Execution) any {
	if e != s.execution {
		panic("wrong execution")
	}
	s.next++
	value := s.values[s.index]
	s.index++
	return value
}

type streamPhase1SliceOnly struct{ calls int }

func (s *streamPhase1SliceOnly) Slice() []int32 { s.calls++; return []int32{1, 2} }

func TestStreamPhase1CustomPullAndRefusal(t *testing.T) {
	e := NewExecution()
	source := &streamPhase1Pull{values: []int32{1, 2, 3}}
	s := streamLimit(streamFromIterable[int32](source), 1)
	if source.factories != 0 || source.slices != 0 || source.sizes != 0 {
		t.Fatal("construction prefetched source")
	}
	if got := streamToSliceExecution(e, s); !reflect.DeepEqual(got, []int32{1}) {
		t.Fatal(got)
	}
	if source.factories != 1 || source.hasNext != 1 || source.next != 1 || source.slices != 0 || source.sizes != 0 {
		t.Fatalf("prefetch: %+v", source)
	}
	zero := &streamPhase1Pull{values: []int32{1, 2}}
	if len(streamToSliceExecution(e, streamLimit(streamFromJavaCollection[int32](zero), 0))) != 0 {
		t.Fatal("limit0")
	}
	if zero.factories != 0 || zero.hasNext != 0 || zero.next != 0 {
		t.Fatal("limit0 entered custom iterator")
	}
	unknown := &streamPhase1Pull{values: []int32{1, 2}}
	if streamCountExecution(e, streamFromJavaCollection[int32](unknown)) != 2 || unknown.next != 2 || unknown.sizes != 0 {
		t.Fatal("unknown count skipped/drained custom size")
	}
	unsafe := &streamPhase1SliceOnly{}
	if p := streamPhase1Recover(func() { _ = streamFromIterable[int32](unsafe) }); p == nil {
		t.Fatal("Slice-only source accepted")
	}
	if unsafe.calls != 0 {
		t.Fatal("refusal invoked Slice")
	}
}

func TestStreamPhase1DeferredCrossTypeAndCancellation(t *testing.T) {
	e := NewExecution()
	var events []int32
	s := streamMap(streamFromSlice([]int32{3, 1, 2}), func(_ *Execution, v int32) string { events = append(events, v); return "x" })
	s = streamLimit(s, 1)
	if len(events) != 0 {
		t.Fatal("map at construction")
	}
	if got := streamToSliceExecution(e, s); !reflect.DeepEqual(got, []string{"x"}) || !reflect.DeepEqual(events, []int32{3}) {
		t.Fatal(got, events)
	}
	var matched []int32
	input := streamPeek(streamFromSlice([]int32{3, 1, 2}), func(_ *Execution, v int32) { matched = append(matched, v) })
	done := false
	streamClaimAndEvaluate(e, input, func() streamSink[int32] {
		return &streamSinkFuncs[int32]{
			accept: func(v int32) { done = v == 1 }, cancel: func() bool { return done },
		}
	})
	if !reflect.DeepEqual(matched, []int32{3, 1}) {
		t.Fatal(matched)
	}
	values := streamSkip(streamLimit(streamFromSlice([]int32{1, 2, 3}), 3), 1)
	if got := streamToSliceExecution(e, values); !reflect.DeepEqual(got, []int32{2, 3}) {
		t.Fatal(got)
	}
}

func TestStreamPhase1SortedBarrierAndDelayedBegin(t *testing.T) {
	e := NewExecution()
	var events []string
	upstream, compares := 0, 0
	s := streamSorted(streamPeek(streamFromSlice([]int32{3, 1, 2}), func(*Execution, int32) { upstream++; events = append(events, "peek") }), func(*Execution, int32, int32) int32 { compares++; events = append(events, "compare"); return 0 })
	if upstream != 0 || compares != 0 {
		t.Fatal("sort evaluated at build")
	}
	streamClaimAndEvaluate(e, streamLimit(s, 0), func() streamSink[int32] {
		return &streamSinkFuncs[int32]{
			begin: func(int64) { events = append(events, "actual-A") }, accept: func(int32) { t.Fatal("limit0 emitted") },
		}
	})
	if upstream != 3 || compares == 0 || events[len(events)-1] != "actual-A" {
		t.Fatal(upstream, compares, events)
	}
	type row struct {
		key int32
		id  string
	}
	stable := streamSorted(streamFromSlice([]row{{2, "a"}, {1, "b"}, {2, "c"}}), func(_ *Execution, a, b row) int32 { return a.key - b.key })
	if got := streamToSliceExecution(e, stable); !reflect.DeepEqual(got, []row{{1, "b"}, {2, "a"}, {2, "c"}}) {
		t.Fatal(got)
	}
	marker := NewIllegalStateException("actual-A")
	var order []string
	throwing := streamSorted(streamPeek(streamFromSlice([]int32{2, 1}), func(*Execution, int32) { order = append(order, "peek") }), func(_ *Execution, a, b int32) int32 { order = append(order, "compare"); return a - b })
	p := streamPhase1Recover(func() {
		streamClaimAndEvaluate(e, throwing, func() streamSink[int32] {
			return &streamSinkFuncs[int32]{
				begin: func(int64) { order = append(order, "actual-A"); panic(marker) }, accept: func(int32) { t.Fatal("accepted after Begin failure") },
			}
		})
	})
	if !sameThrowableIdentity(p, marker) || !reflect.DeepEqual(order, []string{"peek", "peek", "compare", "actual-A"}) {
		t.Fatal(p, order)
	}
	streamPhase1Consumed(t, throwing, e)
}

func TestStreamPhase1CurrentExecution(t *testing.T) {
	construction, consumption, closing := NewExecution(), NewExecution(), NewExecution()
	local := NewThreadLocal[string]()
	local.Set(construction, "construction")
	local.Set(consumption, "consumption")
	var seen []string
	s := streamFromSource(func(e *Execution) streamBoundSource[int32] {
		if e != consumption {
			t.Fatal("source retained construction execution")
		}
		return streamBoundSource[int32]{1, func(_ *streamEvaluation, sink streamSink[int32]) { sink.Accept(1) }}
	})
	s = streamPeek(s, func(e *Execution, _ int32) { seen = append(seen, local.Get(e)) })
	s = streamOnClose(s, func(e *Execution) {
		if e != closing {
			t.Fatal("close retained consuming execution")
		}
		seen = append(seen, "close")
	})
	if got := streamToSliceExecution(consumption, s); !reflect.DeepEqual(got, []int32{1}) || !reflect.DeepEqual(seen, []string{"consumption"}) {
		t.Fatal(got, seen)
	}
	streamCloseExecution(closing, s)
	if !reflect.DeepEqual(seen, []string{"consumption", "close"}) {
		t.Fatal(seen)
	}
}

func TestStreamPhase1CloseComposition(t *testing.T) {
	e := NewExecution()
	first, later := NewIllegalStateException("same"), NewIllegalArgumentException("later")
	var events []string
	s := streamFromSlice([]int32{1})
	s = streamOnClose(s, func(*Execution) { events = append(events, "A"); streamCloseExecution(e, s); panic(first) })
	s = streamOnClose(s, func(*Execution) { events = append(events, "B"); panic(first) })
	s = streamOnClose(s, func(*Execution) { events = append(events, "C"); panic(later) })
	if streamCountExecution(e, s) != 1 || len(events) != 0 {
		t.Fatal("terminal auto-closed")
	}
	p := streamPhase1Recover(func() { streamCloseExecution(e, s) })
	if !sameThrowableIdentity(p, first) || !reflect.DeepEqual(events, []string{"A", "B", "C"}) {
		t.Fatal(p, events)
	}
	suppressed := GetSuppressed(p)
	if len(suppressed) != 1 || !sameThrowableIdentity(suppressed[0], later) {
		t.Fatal(suppressed)
	}
	if p := streamPhase1Recover(func() { streamCloseExecution(e, s) }); p != nil || len(events) != 3 {
		t.Fatal("close repeated after failure", p, events)
	}
	streamPhase1Consumed(t, s, e)
	unconsumed := streamFromSlice([]int32{1})
	streamCloseExecution(e, unconsumed)
	streamPhase1Consumed(t, unconsumed, e)
}

func TestStreamPhase1FlatMapCleanup(t *testing.T) {
	e := NewExecution()
	mapped, innerCloses, outerCloses := 0, 0, 0
	s := streamFlatMap(streamFromSlice([]int32{1, 2, 3}), func(got *Execution, _ int32) Stream[string] {
		if got != e {
			t.Fatal("flatMap execution")
		}
		mapped++
		return streamOnClose(streamFromSlice([]string{"first", "second"}), func(got *Execution) {
			if got != e {
				t.Fatal("inner close execution")
			}
			innerCloses++
		})
	})
	s = streamOnClose(s, func(*Execution) { outerCloses++ })
	if mapped != 0 {
		t.Fatal("flatMap evaluated at build")
	}
	if got := streamToSliceExecution(e, streamLimit(s, 1)); !reflect.DeepEqual(got, []string{"first"}) || mapped != 1 || innerCloses != 1 || outerCloses != 0 {
		t.Fatal(got, mapped, innerCloses, outerCloses)
	}
	streamCloseExecution(e, s)
	if outerCloses != 1 {
		t.Fatal(outerCloses)
	}
	body, closeFailure := NewIllegalStateException("body"), NewIllegalArgumentException("close")
	failing := streamFlatMap(streamFromSlice([]int32{1}), func(*Execution, int32) Stream[int32] {
		return streamOnClose(streamFromSlice([]int32{1}), func(*Execution) { panic(closeFailure) })
	})
	p := streamPhase1Recover(func() {
		streamClaimAndEvaluate(e, failing, func() streamSink[int32] { return &streamSinkFuncs[int32]{accept: func(int32) { panic(body) }} })
	})
	if !sameThrowableIdentity(p, body) {
		t.Fatal(p)
	}
	suppressed := GetSuppressed(p)
	if len(suppressed) != 1 || !sameThrowableIdentity(suppressed[0], closeFailure) {
		t.Fatal(suppressed)
	}
}

func TestStreamPhase1SizeCountAndCollectCounting(t *testing.T) {
	e := NewExecution()
	calls := 0
	makeSized := func() Stream[int32] {
		return streamPeek(streamSorted(streamMap(streamFromSlice([]int32{3, 1, 2}), func(_ *Execution, v int32) int32 { calls++; return v }), func(_ *Execution, a, b int32) int32 { calls++; return a - b }), func(*Execution, int32) { calls++ })
	}
	if got := streamCountExecution(e, makeSized()); got != 3 || calls != 0 {
		t.Fatal("sized count evaluated callbacks", got, calls)
	}
	sink := &streamCountSink[int32]{}
	streamClaimAndEvaluate(e, makeSized(), func() streamSink[int32] { return sink })
	if sink.count != 3 || calls == 0 {
		t.Fatal("counting collector skipped pipeline", sink.count, calls)
	}
	binds, pulls := 0, 0
	unknown := streamFromSource(func(*Execution) streamBoundSource[int32] {
		binds++
		return streamBoundSource[int32]{-1, func(_ *streamEvaluation, sink streamSink[int32]) { pulls++; sink.Accept(1) }}
	})
	if streamCountExecution(e, unknown) != 1 || binds != 1 || pulls != 1 {
		t.Fatal("source bound twice", binds, pulls)
	}
	list := NewListFrom[int32](1, 2)
	late := streamFromIterable[int32](list)
	list.Add(3)
	if streamCountExecution(e, late) != 3 {
		t.Fatal("count captured source size at creation")
	}
	if streamCountExecution(e, streamLimit(streamSkip(streamFromSlice([]int32{1, 2, 3}), 1), 1)) != 1 {
		t.Fatal("size projection")
	}
}

func TestStreamPhase1NativeCompatibilityBoundary(t *testing.T) {
	e := NewExecution()
	var zero Stream[int32]
	firstCount, secondCount := streamCountExecution(e, zero), streamCountExecution(e, zero)
	if firstCount != 0 || secondCount != 0 {
		t.Fatal("native zero empty compatibility")
	}
	initialized := streamFromSlice[int32](nil)
	if initialized.pipeline == nil || streamCountExecution(e, initialized) != 0 {
		t.Fatal("empty core source uninitialized")
	}
	streamPhase1Consumed(t, initialized, e)
	legacy := NewStream[int32](1, 2)
	if legacy.pipeline != nil || legacy.Count() != 2 || legacy.Count() != 2 {
		t.Fatal("legacy behavior changed")
	}
	if p := streamPhase1Recover(func() { streamToSliceExecution(e, legacy) }); p == nil {
		t.Fatal("mixed eager/lazy source silently accepted")
	}
}

func TestStreamPhase1ListSupplierMutationFence(t *testing.T) {
	for _, name := range []string{"bulk-add", "limit-add", "sorted-add", "sorted-limit-add", "bulk-set"} {
		t.Run(name, func(t *testing.T) {
			e := NewExecution()
			list := NewListFrom[int32](2, 1)
			var events []string
			var got []int32
			s := streamPeek(streamFromIterable[int32](list), func(*Execution, int32) { events = append(events, "peek") })
			sorted := name == "sorted-add" || name == "sorted-limit-add"
			limited := name == "limit-add" || name == "sorted-limit-add"
			if sorted {
				s = streamSorted(s, func(_ *Execution, a, b int32) int32 { events = append(events, "compare"); return a - b })
			}
			if limited {
				s = streamLimit(s, 1)
			}
			p := streamPhase1Recover(func() {
				streamClaimAndEvaluate(e, s, func() streamSink[int32] {
					return &streamSinkFuncs[int32]{
						begin: func(int64) {
							events = append(events, "A")
							if name == "bulk-set" {
								list.Set(0, 9)
							} else {
								list.Add(3)
							}
						},
						accept: func(v int32) { events = append(events, "accept"); got = append(got, v) },
					}
				})
			})
			want := []int32{2, 1}
			wantEvents := []string{"A", "peek", "accept", "peek", "accept"}
			switch name {
			case "limit-add":
				want = []int32{2}
				wantEvents = []string{"A", "peek", "accept"}
			case "sorted-add":
				want = []int32{1, 2}
				wantEvents = []string{"peek", "peek", "compare", "A", "accept", "accept"}
			case "sorted-limit-add":
				want = []int32{1}
				wantEvents = []string{"peek", "peek", "compare", "A", "accept"}
			case "bulk-set":
				want = []int32{9, 1}
			}
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(events, wantEvents) {
				t.Fatal(got, events, want, wantEvents)
			}
			shouldFail := name == "bulk-add" || name == "limit-add"
			if shouldFail {
				if _, ok := p.(ConcurrentModificationException); !ok {
					t.Fatalf("got %T, want CME", p)
				}
			} else if p != nil {
				t.Fatal(p)
			}
			streamPhase1Consumed(t, s, e)
		})
	}
}

func TestStreamPhase1SourceRelease(t *testing.T) {
	for _, phase := range []string{"bind", "bind-failure", "close"} {
		t.Run(phase, func(t *testing.T) {
			e := NewExecution()
			cleared, opened := 0, 0
			var events []string
			s := streamFromSource(func(*Execution) streamBoundSource[int32] {
				opened++
				if cleared != 1 {
					t.Fatal("opener retained during callback")
				}
				events = append(events, "open")
				if phase == "bind-failure" {
					panic(NewIllegalStateException("open"))
				}
				return streamBoundSource[int32]{0, func(*streamEvaluation, streamSink[int32]) {}}
			})
			owner := s.pipeline.state.source
			clear := owner.clearSource
			owner.clearSource = func() { cleared++; events = append(events, "clear"); clear() }
			s = streamOnClose(s, func(*Execution) {
				if cleared != 1 || owner.clearSource != nil {
					t.Fatal("close callback retained source")
				}
				events = append(events, "close")
			})
			if phase == "close" {
				streamCloseExecution(e, s)
				if opened != 0 || !reflect.DeepEqual(events, []string{"clear", "close"}) {
					t.Fatal(opened, events)
				}
			} else {
				p := streamPhase1Recover(func() { streamCountExecution(e, s) })
				if (phase == "bind-failure") != (p != nil) {
					t.Fatal(p)
				}
				if opened != 1 || cleared != 1 || owner.clearSource != nil {
					t.Fatal(opened, cleared)
				}
				streamCloseExecution(e, s)
				if !reflect.DeepEqual(events, []string{"clear", "open", "close"}) {
					t.Fatal(events)
				}
			}
			streamCloseExecution(e, s)
			if cleared != 1 {
				t.Fatal("source cleared twice")
			}
		})
	}
}

func TestStreamPhase1SortedKeepsGlobalShortCircuitMode(t *testing.T) {
	e := NewExecution()
	list := NewListFrom[int32](2, 1)
	peeks, compares, begins := 0, 0, 0
	s := streamPeek(streamFromIterable[int32](list), func(*Execution, int32) { peeks++; list.Add(3) })
	s = streamLimit(streamSorted(s, func(_ *Execution, a, b int32) int32 { compares++; return a - b }), 1)
	p := streamPhase1Recover(func() {
		streamClaimAndEvaluate(e, s, func() streamSink[int32] {
			return &streamSinkFuncs[int32]{
				begin: func(int64) { begins++ }, accept: func(int32) { t.Fatal("sorted emitted after source CME") },
			}
		})
	})
	if _, ok := p.(ConcurrentModificationException); !ok || peeks != 1 || compares != 0 || begins != 0 {
		t.Fatal(p, peeks, compares, begins)
	}
}
