package stdjava

import "testing"

func intPredicatePanic(action func()) (caught any) {
	defer func() { caught = recover() }()
	action()
	return
}

func TestIntPredicateRuntimeExecutionAndWidth(t *testing.T) {
	caller := NewExecution()
	other := NewExecution()
	var got int32
	predicate := NewIntPredicateFuncAdapter(func(e *Execution, value int32) bool {
		if e != caller {
			t.Fatal("lost consuming execution")
		}
		got = value
		return value < 0
	})
	for _, value := range []int32{-2147483648, 2147483647} {
		if CallIntPredicateExecution(caller, predicate, value) != (value < 0) || got != value {
			t.Fatal("primitive width/result changed")
		}
	}
	negated := IntPredicateNegateExecution(other, predicate)
	if !CallIntPredicateExecution(caller, negated, 1) {
		t.Fatal("negation captured construction execution")
	}
	if !ObjectInstanceOf(predicate, "java.util.function.IntPredicate") {
		t.Fatal("missing nominal identity")
	}
}
func TestIntPredicateRuntimeShortCircuit(t *testing.T) {
	for _, leftValue := range []bool{false, true} {
		for _, rightValue := range []bool{false, true} {
			for _, and := range []bool{false, true} {
				e := NewExecution()
				calls := ""
				left := NewIntPredicateFuncAdapter(func(caller *Execution, v int32) bool {
					if caller != e || v != 7 {
						t.Fatal("left consumer boundary")
					}
					calls += "L"
					return leftValue
				})
				right := NewIntPredicateFuncAdapter(func(caller *Execution, v int32) bool {
					if caller != e || v != 7 {
						t.Fatal("right consumer boundary")
					}
					calls += "R"
					return rightValue
				})
				combined := IntPredicateOrExecution(NewExecution(), left, right)
				want := leftValue || rightValue
				wantCalls := "L"
				if !leftValue {
					wantCalls = "LR"
				}
				if and {
					combined = IntPredicateAndExecution(NewExecution(), left, right)
					want = leftValue && rightValue
					wantCalls = "L"
					if leftValue {
						wantCalls = "LR"
					}
				}
				if calls != "" {
					t.Fatal("composition invoked predicate eagerly")
				}
				if CallIntPredicateExecution(e, combined, 7) != want || calls != wantCalls {
					t.Fatalf("and=%t left=%t right=%t calls=%s", and, leftValue, rightValue, calls)
				}
			}
		}
	}
}
func TestIntPredicateRuntimeEagerNullAndExceptions(t *testing.T) {
	e := NewExecution()
	calls := 0
	left := NewIntPredicateFuncAdapter(func(*Execution, int32) bool { calls++; return true })
	for _, action := range []func(){func() { IntPredicateAndExecution(e, left, nil) }, func() { IntPredicateOrExecution(e, left, nil) }, func() { IntPredicateNegateExecution(e, nil) }, func() { CallIntPredicateExecution(e, nil, 3) }} {
		caught := intPredicatePanic(action)
		if !ObjectInstanceOf(caught, "java.lang.NullPointerException") {
			t.Fatalf("not Java NPE: %T", caught)
		}
	}
	if calls != 0 {
		t.Fatal("eager validation called predicate")
	}
	marker := NewIllegalStateException("left")
	left = NewIntPredicateFuncAdapter(func(*Execution, int32) bool { panic(marker) })
	right := NewIntPredicateFuncAdapter(func(*Execution, int32) bool { calls++; return true })
	for _, combined := range []IntPredicate{IntPredicateAndExecution(e, left, right), IntPredicateOrExecution(e, left, right)} {
		if got := intPredicatePanic(func() { CallIntPredicateExecution(e, combined, 0) }); got != marker {
			t.Fatal("exception identity lost")
		}
	}
	if calls != 0 {
		t.Fatal("right executed after left threw")
	}
}

type intPredicatePlain struct{}

func (intPredicatePlain) Test(value int32) bool { return value == 42 }
func TestIntPredicateRuntimePlainAndTypedNull(t *testing.T) {
	e := NewExecution()
	if !CallIntPredicateExecution(e, intPredicatePlain{}, 42) {
		t.Fatal("legacy primitive entry lost")
	}
	var absent *IntPredicateFuncAdapter
	for _, action := range []func(){func() { CallIntPredicateExecution(e, absent, 1) }, func() {
		IntPredicateAndExecution(e, NewPlainIntPredicateFuncAdapter(func(int32) bool { return true }), absent)
	}} {
		if caught := intPredicatePanic(action); !ObjectInstanceOf(caught, "java.lang.NullPointerException") {
			t.Fatalf("typed nil not Java NPE: %T", caught)
		}
	}
}

type intPredicateCollisionSource struct{ *ObjectInfo }

func (*intPredicateCollisionSource) Test(int32) bool { return false }
func TestIntPredicateRuntimeExplicitSourceViewIdentity(t *testing.T) {
	id := TypeID("intpredicate.control.Collision")
	RegisterJavaType(id, ObjectTypeID, "java.util.function.IntPredicate")
	e := NewExecution()
	source := &intPredicateCollisionSource{}
	source.ObjectInfo = NewObjectInfo(id, func(requested TypeID) any {
		if requested == id {
			return source
		}
		if requested == "java.util.function.IntPredicate" {
			return NewIntPredicateSourceView(source.ObjectInfo, func(caller *Execution, value int32) bool {
				if caller != e {
					t.Fatal("source callback lost caller")
				}
				return value == 9
			})
		}
		return nil
	})
	view := ObjectView[IntPredicate](source, "java.util.function.IntPredicate")
	if !JavaReferenceEqual(source, view) || !ObjectInstanceOf(view, id) || !CallIntPredicateExecution(e, view, 9) {
		t.Fatal("nominal view lost exact source identity/entry")
	}
	if source.Test(9) {
		t.Fatal("invalid source control")
	}
}

func TestIntPredicateRuntimeSourceDefaultOverrides(t *testing.T) {
	construction, consuming := NewExecution(), NewExecution()
	effects := 0
	result := func(value bool) IntPredicate {
		return NewIntPredicateFuncAdapter(func(e *Execution, _ int32) bool {
			if e != consuming {
				t.Fatal("override result captured construction execution")
			}
			return value
		})
	}
	source := NewIntPredicateSourceViewWithDefaults(nil, func(*Execution, int32) bool { return false }, IntPredicateSourceDefaults{
		Negate: func(e *Execution) IntPredicate {
			if e != construction {
				t.Fatal("negate override lost caller")
			}
			effects++
			return result(false)
		},
		And: func(e *Execution, other IntPredicate) IntPredicate {
			if e != construction || other != nil {
				t.Fatal("and override received wrong argument/execution")
			}
			effects += 10
			return result(true)
		},
		Or: func(e *Execution, other IntPredicate) IntPredicate {
			if e != construction || other != nil {
				t.Fatal("or override received wrong argument/execution")
			}
			effects += 100
			return result(false)
		},
	})
	if CallIntPredicateExecution(consuming, IntPredicateNegateExecution(construction, source), 1) {
		t.Fatal("negate override replaced by !Test")
	}
	if !CallIntPredicateExecution(consuming, IntPredicateAndExecution(construction, source, nil), 1) {
		t.Fatal("and source override lost")
	}
	if CallIntPredicateExecution(consuming, IntPredicateOrExecution(construction, source, nil), 1) {
		t.Fatal("or source override lost")
	}
	if effects != 111 {
		t.Fatalf("default overrides not each invoked once: %d", effects)
	}
}
