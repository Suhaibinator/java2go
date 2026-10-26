package stdjava

import (
	"math"
	"sync"
	"testing"
)

var metadataCostHash int32
var metadataCostEqual bool

// The original Analytics allocation profile attributes 30.26% of sampled
// bytes to repeated discovery of execution companions during Map key lookup.
// Keys and executions are prepared before measuring warmed dispatch. This is
// a structural allocation contract, independent of host wall-clock speed.
func TestObjectExecutionMetadataNativeCost(t *testing.T) {
	e := NewExecution()
	pairs := []struct {
		name        string
		left, right any
	}{
		{"UTF16String", NewJavaStringUTF16([]uint16{'A', 0xd800, 'B'}), NewJavaStringUTF16([]uint16{'A', 0xd800, 'B'})},
		{"Integer", NewInteger(300), NewInteger(300)},
		{"DoubleNegativeZero", NewDouble(math.Copysign(0, -1)), NewDouble(math.Copysign(0, -1))},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			want := ObjectHashCodeExecution(e, pair.left)
			allocations := testing.AllocsPerRun(200, func() {
				metadataCostHash = ObjectHashCodeExecution(e, pair.left)
				metadataCostEqual = ObjectEqualsExecution(e, pair.left, pair.right)
			})
			if metadataCostHash != want || !metadataCostEqual {
				t.Fatal("native value observations changed")
			}
			t.Logf("native hash+equals allocations/pair=%.2f frozen-budget=2", allocations)
			if allocations > 2 {
				t.Fatalf("native dispatch allocations/pair=%.2f exceeds frozen budget 2", allocations)
			}
		})
	}
}

type metadataCostRenamed struct {
	execution      *Execution
	id             int32
	hashes, equals int
	argument       any
	fail           any
}

func (p *metadataCostRenamed) HashCodeJava2goExecution2(e *Execution) int32 {
	if e != p.execution {
		panic("stale execution")
	}
	p.hashes++
	if p.fail != nil {
		panic(p.fail)
	}
	return p.id
}
func (p *metadataCostRenamed) EqualsJava2goExecution3(e *Execution, other any) bool {
	if e != p.execution {
		panic("stale execution")
	}
	p.equals++
	p.argument = other
	if p.fail != nil {
		panic(p.fail)
	}
	return other == p
}
func (*metadataCostRenamed) HashCodeJava2goExecutionNative(*Execution) int32 { panic("invalid suffix") }
func (*metadataCostRenamed) EqualsJava2goExecution1(*Execution, *metadataCostRenamed) bool {
	panic("typed overload")
}
func (*metadataCostRenamed) HashCode() int32 { panic("native fallback before companion") }
func (*metadataCostRenamed) Equals(any) bool { panic("native fallback before companion") }

func TestObjectExecutionMetadataRenamedCost(t *testing.T) {
	e := NewExecution()
	p := &metadataCostRenamed{execution: e, id: 53}
	allocations := testing.AllocsPerRun(200, func() {
		metadataCostHash = ObjectHashCodeExecution(e, p)
		metadataCostEqual = ObjectEqualsExecution(e, p, p)
	})
	if metadataCostHash != 53 || !metadataCostEqual || p.argument != p || p.hashes != 201 || p.equals != 201 {
		t.Fatal("renamed receiver/call count")
	}
	t.Logf("renamed hash+equals allocations/pair=%.2f frozen-budget=12", allocations)
	if allocations > 12 {
		t.Fatalf("renamed dispatch allocations/pair=%.2f exceeds frozen budget 12", allocations)
	}
}

func TestObjectExecutionMetadataReceiverAndExecution(t *testing.T) {
	e1, e2 := NewExecution(), NewExecution()
	a := &metadataCostRenamed{execution: e1, id: 59}
	b := &metadataCostRenamed{execution: e2, id: 61}
	for _, p := range []*metadataCostRenamed{a, b, a, b} {
		if ObjectHashCodeExecution(p.execution, p) != p.id || !ObjectEqualsExecution(p.execution, p, p) {
			t.Fatal("receiver/execution cached")
		}
		if ObjectEqualsExecution(p.execution, p, nil) || p.argument != nil {
			t.Fatal("false result/null argument changed")
		}
	}
	a.execution = e2
	if ObjectHashCodeExecution(e2, a) != 59 {
		t.Fatal("caller execution stale after mutation")
	}
	sentinel := &struct{ label string }{"throwing companion"}
	a.fail = sentinel
	for _, body := range []func(){func() { ObjectHashCodeExecution(e2, a) }, func() { ObjectEqualsExecution(e2, a, b) }} {
		if dispatchTDDRecover(t, body) != sentinel {
			t.Fatal("exception identity changed")
		}
	}
	if a.hashes != 4 || a.equals != 5 || b.hashes != 2 || b.equals != 4 {
		t.Fatalf("calls changed: a=%d/%d b=%d/%d", a.hashes, a.equals, b.hashes, b.equals)
	}
	var absent *metadataCostRenamed
	if _, ok := dispatchTDDRecover(t, func() { ObjectHashCodeExecution(e1, absent) }).(NullPointerException); !ok {
		t.Fatal("typed-null receiver")
	}
}

type metadataCostConcurrentA struct{ *metadataCostRenamed }
type metadataCostConcurrentB struct{ *metadataCostRenamed }

func TestObjectExecutionMetadataParallelTypes(t *testing.T) {
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			e := NewExecution()
			p := &metadataCostRenamed{execution: e, id: int32(index + 67)}
			var receiver any = &metadataCostConcurrentA{p}
			if index%2 != 0 {
				receiver = &metadataCostConcurrentB{p}
			}
			for call := 0; call < 64; call++ {
				if ObjectHashCodeExecution(e, receiver) != p.id || ObjectEqualsExecution(e, receiver, p) != true || ObjectEqualsExecution(e, receiver, nil) != false {
					t.Errorf("parallel receiver/type selection %d/%d", index, call)
					return
				}
			}
			if p.hashes != 64 || p.equals != 128 {
				t.Errorf("parallel call counts")
			}
		}(worker)
	}
	workers.Wait()
}
