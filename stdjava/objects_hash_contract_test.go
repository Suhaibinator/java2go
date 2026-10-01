package stdjava

import "testing"

type objectsHashExecutionKey struct {
	execution *Execution
	trace     *[]string
	hash      int32
	mutate    func()
	failure   any
}

func (k *objectsHashExecutionKey) HashCodeJava2goExecution(execution *Execution) int32 {
	if execution != k.execution {
		panic("wrong caller Execution")
	}
	*k.trace = append(*k.trace, "hash")
	if k.mutate != nil {
		k.mutate()
	}
	if k.failure != nil {
		panic(k.failure)
	}
	return k.hash
}
func TestObjectsHashExecutionContract(t *testing.T) {
	execution := NewExecution()
	trace := []string{}
	var live *ReferenceArray
	first := &objectsHashExecutionKey{execution: execution, trace: &trace, hash: 4, mutate: func() { ReferenceArraySet(live, 1, BoxInteger(9)) }}
	live = ReferenceArrayLiteralOf[any](ObjectTypeID, first, BoxInteger(2))
	if got := ObjectsHashExecution(execution, live); got != 1094 {
		t.Fatalf("live fold: %d", got)
	}
	if len(trace) != 1 {
		t.Fatalf("dispatch trace: %v", trace)
	}
	var typedNil *JavaString
	nulls := ReferenceArrayLiteralOf[any](ObjectTypeID, nil, typedNil)
	if ObjectsHashExecution(execution, nulls) != 961 || ObjectsHashExecution(execution, nil) != 0 || ObjectsHashExecution(execution, ReferenceArrayLiteralOf[any](ObjectTypeID)) != 1 {
		t.Fatal("null/empty fold")
	}
	overflowing := ReferenceArrayLiteralOf[any](ObjectTypeID, BoxInteger(2147483647), BoxInteger(-2147483648))
	if got := ObjectsHashExecution(execution, overflowing); got != 930 {
		t.Fatalf("Java int overflow: %d", got)
	}
	marker := NewArithmeticException("")
	trace = nil
	bad := &objectsHashExecutionKey{execution: execution, trace: &trace, failure: marker}
	never := &objectsHashExecutionKey{execution: execution, trace: &trace, hash: 8}
	cleaned := false
	func() {
		defer func() {
			cleaned = true
			if got := recover(); got != marker {
				t.Fatalf("abrupt identity: %v", got)
			}
		}()
		ObjectsHashExecution(execution, ReferenceArrayLiteralOf[any](ObjectTypeID, bad, never))
		t.Fatal("missing abrupt completion")
	}()
	if !cleaned || len(trace) != 1 {
		t.Fatalf("abrupt cleanup/dispatch: %v %v", cleaned, trace)
	}
}
