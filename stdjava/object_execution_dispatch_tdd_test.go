package stdjava

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Native runtime controls; these define no application or Java-library behavior.
type dispatchTDDStandard struct {
	want     *Execution
	seen     *Execution
	other    any
	calls    int
	throwing any
}

func (p *dispatchTDDStandard) EqualsJava2goExecution(e *Execution, other any) bool {
	if e != p.want {
		panic("wrong logical execution")
	}
	p.seen, p.other = e, other
	p.calls++
	if p.throwing != nil {
		panic(p.throwing)
	}
	return true
}
func (p *dispatchTDDStandard) HashCodeJava2goExecution(e *Execution) int32 {
	if e != p.want {
		panic("wrong logical execution")
	}
	p.seen = e
	p.calls++
	if p.throwing != nil {
		panic(p.throwing)
	}
	return 73
}

// A realistic generated-like exported method surface, not a two-method ideal.
func (*dispatchTDDStandard) Alpha()           {}
func (*dispatchTDDStandard) Beta()            {}
func (*dispatchTDDStandard) Delta()           {}
func (*dispatchTDDStandard) Epsilon()         {}
func (*dispatchTDDStandard) Gamma()           {}
func (*dispatchTDDStandard) GetClass()        {}
func (*dispatchTDDStandard) JavaObjectShape() {}
func (*dispatchTDDStandard) Omega()           {}
func (*dispatchTDDStandard) ToString()        {}
func (*dispatchTDDStandard) Zeta()            {}

var dispatchTDDHashSink int32
var dispatchTDDEqualitySink bool

func TestObjectExecutionDispatchAllocationBudget(t *testing.T) {
	execution := NewExecution()
	receiver := &dispatchTDDStandard{want: execution}
	argument := &dispatchTDDStandard{want: execution}
	// Frozen budget: at most four allocations for two source observations.
	// This tolerates bounded argument/result handling but rejects repeated
	// reflective method discovery, attributed 98.44% of full-main allocations.
	allocations := testing.AllocsPerRun(200, func() {
		dispatchTDDHashSink = ObjectHashCodeExecution(execution, receiver)
		dispatchTDDEqualitySink = ObjectEqualsExecution(execution, receiver, argument)
	})
	if dispatchTDDHashSink != 73 || !dispatchTDDEqualitySink || receiver.other != argument || receiver.seen != execution {
		t.Fatal("allocation probe lost source observations or execution")
	}
	t.Logf("exact-companion hash+equals allocations/pair=%.2f frozen-budget=4", allocations)
	if allocations > 4 {
		t.Fatalf("dispatch allocations/pair=%.2f exceeds frozen budget 4", allocations)
	}
}

type dispatchTDDSelection struct{ calls string }

func (p *dispatchTDDSelection) EqualsJava2goExecution(*Execution, any) bool {
	p.calls += "E"
	return true
}
func (p *dispatchTDDSelection) EqualsJava2goExecution1(*Execution, any) bool {
	panic("numbered equals selected before exact")
}
func (p *dispatchTDDSelection) HashCodeJava2goExecution(*Execution) int32 { p.calls += "H"; return 19 }
func (p *dispatchTDDSelection) HashCodeJava2goExecution1(*Execution) int32 {
	panic("numbered hash selected before exact")
}

type dispatchTDDRenamed struct {
	want  *Execution
	calls string
}

func (p *dispatchTDDRenamed) EqualsJava2goExecution2(e *Execution, other any) bool {
	if e != p.want || other != p {
		panic("renamed equals argument/execution")
	}
	p.calls += "E"
	return true
}
func (p *dispatchTDDRenamed) HashCodeJava2goExecution7(e *Execution) int32 {
	if e != p.want {
		panic("renamed hash execution")
	}
	p.calls += "H"
	return 23
}
func (*dispatchTDDRenamed) EqualsJava2goExecutionNative(*Execution, any) bool {
	panic("nondecimal suffix accepted")
}
func (*dispatchTDDRenamed) HashCodeJava2goExecutionNative(*Execution) int32 {
	panic("nondecimal suffix accepted")
}

type dispatchTDDOverload struct {
	want  *Execution
	calls string
}

func (*dispatchTDDOverload) EqualsJava2goExecution(*Execution, *dispatchTDDOverload) bool {
	panic("typed equals overload accepted")
}
func (*dispatchTDDOverload) HashCodeJava2goExecution(*Execution) int64 {
	panic("wrong hash return accepted")
}
func (p *dispatchTDDOverload) EqualsJava2goExecution3(e *Execution, other any) bool {
	if e != p.want || other != p {
		panic("overload fallback arguments")
	}
	p.calls += "E"
	return true
}
func (p *dispatchTDDOverload) HashCodeJava2goExecution3(e *Execution) int32 {
	if e != p.want {
		panic("overload fallback execution")
	}
	p.calls += "H"
	return 29
}

type dispatchTDDBool bool
type dispatchTDDInt32 int32
type dispatchTDDExecutionAlias = Execution
type dispatchTDDAnyAlias = any
type dispatchTDDAliasedReturn struct {
	want  *Execution
	calls string
}

func (p *dispatchTDDAliasedReturn) EqualsJava2goExecution(e *dispatchTDDExecutionAlias, other dispatchTDDAnyAlias) dispatchTDDBool {
	if e != p.want || other != p {
		panic("named bool argument/execution")
	}
	p.calls += "E"
	return true
}
func (p *dispatchTDDAliasedReturn) HashCodeJava2goExecution(e *dispatchTDDExecutionAlias) dispatchTDDInt32 {
	if e != p.want {
		panic("named int32 execution")
	}
	p.calls += "H"
	return 31
}

type dispatchTDDDefinedAny interface{}
type dispatchTDDDefinedExecution Execution
type dispatchTDDIneligible struct{ calls string }

func (*dispatchTDDIneligible) EqualsJava2goExecution(*Execution, dispatchTDDDefinedAny) bool {
	panic("defined Object interface accepted")
}
func (*dispatchTDDIneligible) HashCodeJava2goExecution(*dispatchTDDDefinedExecution) int32 {
	panic("defined Execution accepted")
}
func (p *dispatchTDDIneligible) Equals(other any) bool { p.calls += "E"; return other == p }
func (p *dispatchTDDIneligible) HashCode() int32       { p.calls += "H"; return 37 }

type dispatchTDDPromoted struct{ *dispatchTDDStandard }
type dispatchTDDOverride struct {
	*dispatchTDDStandard
	want *Execution
	seen *Execution
}

func (p *dispatchTDDOverride) HashCodeJava2goExecution(e *Execution) int32 {
	if e != p.want {
		panic("promoted override execution")
	}
	p.seen = e
	return 41
}

type dispatchTDDViewBase struct{ *ObjectInfo }

func (*dispatchTDDViewBase) EqualsJava2goExecution(*Execution, any) bool {
	panic("base equals bypassed dynamic view")
}
func (*dispatchTDDViewBase) HashCodeJava2goExecution(*Execution) int32 {
	panic("base hash bypassed dynamic view")
}

type dispatchTDDViewLeaf struct {
	*ObjectInfo
	*dispatchTDDStandard
}

func dispatchTDDRecover(t *testing.T, body func()) any {
	t.Helper()
	var caught any
	func() { defer func() { caught = recover() }(); body() }()
	if caught == nil {
		t.Fatal("expected panic")
	}
	return caught
}

func TestObjectExecutionDispatchSemantics(t *testing.T) {
	execution := NewExecution()
	t.Run("exact-before-numbered", func(t *testing.T) {
		p := &dispatchTDDSelection{}
		if ObjectHashCodeExecution(execution, p) != 19 || !ObjectEqualsExecution(execution, p, p) || p.calls != "HE" {
			t.Fatal("exact eligibility/selection")
		}
	})
	t.Run("renamed-and-nondecimal", func(t *testing.T) {
		p := &dispatchTDDRenamed{want: execution}
		if ObjectHashCodeExecution(execution, p) != 23 || !ObjectEqualsExecution(execution, p, p) || p.calls != "HE" {
			t.Fatal("renamed eligibility")
		}
	})
	t.Run("overload-and-wrong-return", func(t *testing.T) {
		p := &dispatchTDDOverload{want: execution}
		if ObjectHashCodeExecution(execution, p) != 29 || !ObjectEqualsExecution(execution, p, p) || p.calls != "HE" {
			t.Fatal("overload exclusion")
		}
	})
	t.Run("named-return-and-true-alias", func(t *testing.T) {
		p := &dispatchTDDAliasedReturn{want: execution}
		if ObjectHashCodeExecution(execution, p) != 31 || !ObjectEqualsExecution(execution, p, p) || p.calls != "HE" {
			t.Fatal("named return fallback")
		}
	})
	t.Run("defined-argument-types-ineligible", func(t *testing.T) {
		p := &dispatchTDDIneligible{}
		if ObjectHashCodeExecution(execution, p) != 37 || !ObjectEqualsExecution(execution, p, p) || p.calls != "HE" {
			t.Fatal("native fallback eligibility")
		}
	})
	t.Run("promoted-and-override", func(t *testing.T) {
		inner := &dispatchTDDStandard{want: execution}
		outer := &dispatchTDDPromoted{inner}
		if ObjectHashCodeExecution(execution, outer) != 73 || !ObjectEqualsExecution(execution, outer, outer) || inner.other != outer || inner.calls != 2 {
			t.Fatal("promoted method receiver")
		}
		override := &dispatchTDDOverride{dispatchTDDStandard: inner, want: execution}
		if ObjectHashCodeExecution(execution, override) != 41 || override.seen != execution || inner.calls != 2 {
			t.Fatal("promoted override")
		}
	})
	t.Run("canonical-view-unwrap-both-sides", func(t *testing.T) {
		leaf := &dispatchTDDViewLeaf{dispatchTDDStandard: &dispatchTDDStandard{want: execution}}
		info := NewObjectInfo("dispatch.tdd.Leaf", func(TypeID) any { return leaf })
		leaf.ObjectInfo = info
		base := &dispatchTDDViewBase{info}
		if ObjectHashCodeExecution(execution, base) != 73 || !ObjectEqualsExecution(execution, base, base) || leaf.other != leaf || leaf.seen != execution {
			t.Fatal("canonical Java view or erased argument")
		}
	})
	t.Run("null-arguments-boxes-and-nil-execution", func(t *testing.T) {
		p := &dispatchTDDStandard{want: execution}
		var absent *Integer
		for _, other := range []any{nil, absent, NullString()} {
			if !ObjectEqualsExecution(execution, p, other) || p.other != nil {
				t.Fatal("null argument normalization")
			}
		}
		boxed := NewInteger(200)
		if !ObjectEqualsExecution(execution, p, boxed) || p.other != boxed {
			t.Fatal("boxed argument identity")
		}
		slice := []int32{7, 11}
		if !ObjectEqualsExecution(execution, p, slice) || reflect.ValueOf(p.other).Pointer() != reflect.ValueOf(slice).Pointer() {
			t.Fatal("noncomparable native argument")
		}
		p.want = nil
		if ObjectHashCodeExecution(nil, p) != 73 || !ObjectEqualsExecution(nil, p, p) || p.seen != nil {
			t.Fatal("nil Execution forwarding")
		}
	})
	t.Run("receiver-null-check-after-argument-evaluation", func(t *testing.T) {
		events := ""
		receiver := func() *dispatchTDDStandard { events += "R"; return nil }
		argument := func() any { events += "A"; return nil }
		caught := dispatchTDDRecover(t, func() { ObjectEqualsExecution(execution, receiver(), argument()) })
		if _, ok := caught.(NullPointerException); !ok || events != "RA" {
			t.Fatalf("null/evaluation order %T %s", caught, events)
		}
		var absent *dispatchTDDStandard
		if _, ok := dispatchTDDRecover(t, func() { ObjectHashCodeExecution(execution, absent) }).(NullPointerException); !ok {
			t.Fatal("hash receiver null check")
		}
	})
	t.Run("throwing-companion-identity-and-once", func(t *testing.T) {
		sentinel := &struct{ label string }{"user failure"}
		p := &dispatchTDDStandard{want: execution, throwing: sentinel}
		if dispatchTDDRecover(t, func() { ObjectHashCodeExecution(execution, p) }) != sentinel || p.calls != 1 || p.seen != execution {
			t.Fatal("hash throw ordering")
		}
		if dispatchTDDRecover(t, func() { ObjectEqualsExecution(execution, p, p) }) != sentinel || p.calls != 2 || p.other != p {
			t.Fatal("equals throw ordering")
		}
	})
}

type dispatchTDDConcurrent struct{ hashes map[*Execution]int32 }
type dispatchTDDConcurrentArgument struct {
	owner     *dispatchTDDConcurrent
	execution *Execution
}

func (p *dispatchTDDConcurrent) HashCodeJava2goExecution(e *Execution) int32 {
	result, ok := p.hashes[e]
	if !ok {
		panic("wrong caller execution")
	}
	return result
}
func (p *dispatchTDDConcurrent) EqualsJava2goExecution(e *Execution, other any) bool {
	argument, ok := other.(*dispatchTDDConcurrentArgument)
	return ok && argument.owner == p && argument.execution == e
}
func TestObjectExecutionDispatchConcurrentCallers(t *testing.T) {
	p := &dispatchTDDConcurrent{hashes: make(map[*Execution]int32)}
	executions := make([]*Execution, 8)
	for worker := range executions {
		executions[worker] = NewExecution()
		p.hashes[executions[worker]] = 43 + int32(worker)
	}
	var workers sync.WaitGroup
	failures := make(chan string, 8)
	for worker := 0; worker < 8; worker++ {
		execution := executions[worker]
		expected := 43 + int32(worker)
		argument := &dispatchTDDConcurrentArgument{p, execution}
		workers.Add(1)
		go func() {
			defer workers.Done()
			for call := 0; call < 64; call++ {
				if ObjectHashCodeExecution(execution, p) != expected || !ObjectEqualsExecution(execution, p, argument) {
					failures <- fmt.Sprintf("worker result at call %d", call)
					return
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for failure := range failures {
		t.Fatal(failure)
	}
}
