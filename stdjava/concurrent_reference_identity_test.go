package stdjava

import "testing"

type structuralReferenceRunnable struct {
	runs int
}

func (r *structuralReferenceRunnable) Run() {
	r.runs++
}

type namedReferenceRunnable struct{}

func (*namedReferenceRunnable) Run() {}

func (*namedReferenceRunnable) JavaDynamicTypeID() TypeID {
	return TypeID("tests.NamedReferenceRunnable")
}

func TestThreadReferenceArrayUsesRuntimeNominalIdentity(t *testing.T) {
	thread := NewThread(nil)
	threads := NewReferenceArray(1, ThreadTypeID)
	ReferenceArraySet(threads, 0, thread)

	if got := ReferenceArrayGet[*Thread](threads, 0, ThreadTypeID); got != thread {
		t.Fatalf("Thread[] read = %p, want original %p", got, thread)
	}
	if !JavaTypeAssignable(ThreadTypeID, RunnableTypeID) {
		t.Fatal("Thread must be assignable to Runnable")
	}
	if !JavaTypeAssignable(ThreadTypeID, ObjectTypeID) {
		t.Fatal("Thread must be assignable to Object")
	}
}

func TestRunnableReferenceArrayRecognizesAdaptersAndStructuralObjects(t *testing.T) {
	structural := &structuralReferenceRunnable{}
	threadRuns := 0
	thread := NewThread(NewPlainRunnableFuncAdapter(func() { threadRuns++ }))
	values := []Runnable{
		NewRunnableFuncAdapter(func(*Execution) {}),
		NewPlainRunnableFuncAdapter(func() {}),
		structural,
		thread,
	}
	runnables := NewReferenceArray(int32(len(values)), RunnableTypeID)
	for index, value := range values {
		ReferenceArraySet(runnables, int32(index), value)
		if got := ReferenceArrayGet[Runnable](runnables, int32(index), RunnableTypeID); got != value {
			t.Fatalf("Runnable[] read %d = %T %p, want %T %p", index, got, got, value, value)
		}
	}
	ReferenceArrayGet[Runnable](runnables, 3, RunnableTypeID).Run()
	if threadRuns != 1 {
		t.Fatalf("Thread read through Runnable[] ran target %d times, want 1", threadRuns)
	}
}

func TestDirectThreadRunPreservesExistingExecutionWithoutSubclassPromotion(t *testing.T) {
	var received *Execution
	thread := NewThread(NewRunnableFuncAdapter(func(execution *Execution) {
		received = execution
	}))
	execution := NewExecution()
	RunRunnableExecution(execution, thread)
	if received != execution {
		t.Fatalf("direct Thread.run execution = %p, want caller token %p", received, execution)
	}
}

func TestRunnableStructuralFallbackDoesNotHideSpecificDynamicType(t *testing.T) {
	const namedType TypeID = "tests.NamedReferenceRunnable"
	RegisterJavaType(namedType, ObjectTypeID, RunnableTypeID)

	value := &namedReferenceRunnable{}
	if got, ok := ObjectDynamicType(value); !ok || got != namedType {
		t.Fatalf("ObjectDynamicType(named Runnable) = (%q, %t), want (%q, true)", got, ok, namedType)
	}
	runnables := NewReferenceArray(1, RunnableTypeID)
	ReferenceArraySet(runnables, 0, value)
}
