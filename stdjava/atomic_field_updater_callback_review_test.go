package stdjava

import (
	"testing"
	"time"
)

type updaterCallbackReviewAPI struct {
	name       string
	returnsNew bool
	invoke     func(*AtomicReferenceFieldUpdater, *Execution, any, func(*Execution, any) any) any
}

func updaterCallbackReviewAPIs() []updaterCallbackReviewAPI {
	return []updaterCallbackReviewAPI{
		{"GetAndUpdate", false, func(u *AtomicReferenceFieldUpdater, e *Execution, target any, callback func(*Execution, any) any) any {
			return u.GetAndUpdateExecution(e, target, callback)
		}},
		{"UpdateAndGet", true, func(u *AtomicReferenceFieldUpdater, e *Execution, target any, callback func(*Execution, any) any) any {
			return u.UpdateAndGetExecution(e, target, callback)
		}},
		{"GetAndAccumulate", false, func(u *AtomicReferenceFieldUpdater, e *Execution, target any, callback func(*Execution, any) any) any {
			operand := NewObject()
			return u.GetAndAccumulateExecution(e, target, operand, func(actual *Execution, previous, value any) any {
				if value != operand {
					panic(NewIllegalStateException("accumulator operand changed"))
				}
				return callback(actual, previous)
			})
		}},
		{"AccumulateAndGet", true, func(u *AtomicReferenceFieldUpdater, e *Execution, target any, callback func(*Execution, any) any) any {
			operand := NewObject()
			return u.AccumulateAndGetExecution(e, target, operand, func(actual *Execution, previous, value any) any {
				if value != operand {
					panic(NewIllegalStateException("accumulator operand changed"))
				}
				return callback(actual, previous)
			})
		}},
	}
}

type updaterCallbackReviewResult struct{ value, failure any }

// The callback/provider bounds make the identity-CAS defect fail deterministically.
// The completion bound separately contains a lock regression in the reentrant write.
func updaterCallbackReviewComplete(t *testing.T, operation func() any) any {
	t.Helper()
	done := make(chan updaterCallbackReviewResult, 1)
	go func() {
		result := updaterCallbackReviewResult{}
		defer func() { result.failure = recover(); done <- result }()
		result.value = operation()
	}()
	select {
	case result := <-done:
		if result.failure != nil {
			t.Fatalf("callback RMW failed before completion: panic=%T %v", result.failure, result.failure)
		}
		return result.value
	case <-time.After(2 * time.Second):
		t.Fatal("callback RMW did not complete within bounded observation")
		return nil
	}
}

func TestAtomicReferenceFieldUpdaterCallbackOpaqueWithoutCompetingWrite(t *testing.T) {
	registerUpdaterFixture()
	for _, api := range updaterCallbackReviewAPIs() {
		t.Run(api.name, func(t *testing.T) {
			target := &updaterTarget{}
			u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
			state := &updaterExchangeReviewState{limit: 32, abort: NewIllegalStateException("callback provider retry bound")}
			opaque := updaterExchangeReviewFunction(func() *updaterExchangeReviewState { return state })
			if !ObjectInstanceOf(opaque, ObjectTypeID) || JavaReferenceEqual(opaque, opaque) {
				t.Fatal("control requires accepted non-reflexive opaque reference")
			}
			u.Set(target, opaque)
			e, replacement, calls := NewExecution(), NewObject(), 0
			result := updaterCallbackReviewComplete(t, func() any {
				return api.invoke(u, e, target, func(actual *Execution, previous any) any {
					calls++
					if calls > 8 {
						panic(NewIllegalStateException("callback identity retry bound"))
					}
					old, ok := previous.(updaterExchangeReviewFunction)
					if actual != e || !ok || old() != state {
						panic(NewIllegalStateException("callback execution or old reference changed"))
					}
					return replacement
				})
			})
			if calls != 1 {
				t.Fatalf("callback calls=%d, want 1 without competing writes", calls)
			}
			if api.returnsNew {
				if result != replacement {
					t.Fatal("RMW did not return replacement")
				}
			} else if old, ok := result.(updaterExchangeReviewFunction); !ok || old() != state {
				t.Fatal("RMW did not return old opaque reference")
			}
			if u.GetExecution(e, target) != replacement {
				t.Fatal("RMW did not store replacement")
			}
		})
	}
}

func TestAtomicReferenceFieldUpdaterCallbackReentrantWriteRetries(t *testing.T) {
	registerUpdaterFixture()
	for _, api := range updaterCallbackReviewAPIs() {
		t.Run(api.name, func(t *testing.T) {
			target := &updaterTarget{}
			u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
			before := &updaterExchangeReviewState{limit: 32, abort: NewIllegalStateException("callback provider retry bound")}
			after := &updaterExchangeReviewState{limit: 32, abort: NewIllegalStateException("callback provider retry bound")}
			first := updaterExchangeReviewFunction(func() *updaterExchangeReviewState { return before })
			competing := updaterExchangeReviewFunction(func() *updaterExchangeReviewState { return after })
			u.Set(target, first)
			e, replacement, calls := NewExecution(), NewObject(), 0
			result := updaterCallbackReviewComplete(t, func() any {
				return api.invoke(u, e, target, func(actual *Execution, previous any) any {
					calls++
					if calls > 8 {
						panic(NewIllegalStateException("callback identity retry bound"))
					}
					old, ok := previous.(updaterExchangeReviewFunction)
					expected := after
					if calls == 1 {
						expected = before
					}
					if actual != e || !ok || old() != expected {
						panic(NewIllegalStateException("callback retry did not observe current reference"))
					}
					if calls == 1 {
						u.SetExecution(actual, target, competing)
					}
					return replacement
				})
			})
			if calls != 2 {
				t.Fatalf("callback calls=%d, want 2 after reentrant write", calls)
			}
			if api.returnsNew {
				if result != replacement {
					t.Fatal("RMW did not return replacement")
				}
			} else if old, ok := result.(updaterExchangeReviewFunction); !ok || old() != after {
				t.Fatal("RMW did not return old reference from successful retry")
			}
			if u.GetExecution(e, target) != replacement {
				t.Fatal("RMW did not store replacement after retry")
			}
		})
	}
}
