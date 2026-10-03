package stdjava

import "testing"

type sourceThreadCanonicalWitness struct{ *ObjectInfo }

func TestThreadCanonicalDynamicDescriptorTDD(t *testing.T) {
	value := ThreadCurrentThread(NewExecution())
	actual, ok := ObjectDynamicType(value)
	if !ok || actual != TypeID("java.lang.Thread") {
		t.Fatalf("Thread runtime descriptor is not canonical java.lang.Thread: (%q,%t)", actual, ok)
	}
}
func TestThreadCanonicalReferenceArrayTransportTDD(t *testing.T) {
	value := ThreadCurrentThread(NewExecution())
	actual, ok := ObjectDynamicType(value)
	if !ok || !JavaTypeAssignable(actual, TypeID("java.lang.Thread")) {
		t.Fatalf("Thread[] canonical component rejects native Thread before array store: %q", actual)
	}
	array := NewReferenceArrayOf[*Thread](1, TypeID("java.lang.Thread"))
	if stored := ReferenceArrayAssign[*Thread](array, 0, value, TypeID("java.lang.Thread")); stored != value {
		t.Fatal("canonical Thread[] store lost original reference")
	}
	if got := ReferenceArrayGet[*Thread](array, 0, TypeID("java.lang.Thread")); got != value {
		t.Fatal("canonical Thread[] read lost original reference")
	}
}
func TestThreadCanonicalSourceShadowDeclinesTDD(t *testing.T) {
	const sourceID TypeID = "Thread"
	javaTypeRegistry.Lock()
	prior, present := javaTypeRegistry.types[sourceID]
	javaTypeRegistry.Unlock()
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		defer javaTypeRegistry.Unlock()
		if present {
			javaTypeRegistry.types[sourceID] = prior
		} else {
			delete(javaTypeRegistry.types, sourceID)
		}
	})
	RegisterJavaType(sourceID, ObjectTypeID)
	RegisterJavaSourceType(sourceID)
	source := &sourceThreadCanonicalWitness{}
	source.ObjectInfo = NewObjectInfo(sourceID, func(id TypeID) any {
		if id == sourceID || id == ObjectTypeID {
			return source
		}
		return nil
	})
	sourceArray := NewReferenceArrayOf[*sourceThreadCanonicalWitness](1, sourceID)
	ReferenceArrayAssign[*sourceThreadCanonicalWitness](sourceArray, 0, source, sourceID)
	if got := ReferenceArrayGet[*sourceThreadCanonicalWitness](sourceArray, 0, sourceID); got != source {
		t.Fatal("source Thread[] identity was rewritten")
	}
	native := ThreadCurrentThread(NewExecution())
	actual, ok := ObjectDynamicType(native)
	if !ok || JavaTypeAssignable(actual, sourceID) {
		t.Fatalf("native Thread borrowed unrelated source Thread nominal identity: %q", actual)
	}
	if JavaTypeAssignable(sourceID, TypeID("java.lang.Thread")) {
		t.Fatal("source Thread acquired unrelated canonical native identity")
	}
}
func TestThreadCanonicalObjectRunnableTransportTDD(t *testing.T) {
	thread := NewThread(NewPlainRunnableFuncAdapter(func() {}))
	if ObjectView[*Thread](thread, ObjectTypeID) != thread {
		t.Fatal("Thread Object view lost identity")
	}
	if ObjectView[Runnable](thread, RunnableTypeID) != thread {
		t.Fatal("Thread Runnable view lost identity")
	}
	objects := NewReferenceArrayOf[any](1, ObjectTypeID)
	ReferenceArrayAssign[any](objects, 0, thread, ObjectTypeID)
	if !JavaReferenceEqual(ReferenceArrayGet[any](objects, 0, ObjectTypeID), thread) {
		t.Fatal("Object[] thread identity changed")
	}
	runnables := NewReferenceArrayOf[Runnable](1, RunnableTypeID)
	ReferenceArrayAssign[Runnable](runnables, 0, thread, RunnableTypeID)
	if ReferenceArrayGet[Runnable](runnables, 0, RunnableTypeID) != thread {
		t.Fatal("Runnable[] thread identity changed")
	}
}
func TestThreadCanonicalExecutionIdentityTransportTDD(t *testing.T) {
	var received *Execution
	thread := NewThread(NewRunnableFuncAdapter(func(execution *Execution) { received = execution }))
	execution := NewExecution()
	caller := ThreadCurrentThread(execution)
	RunRunnableExecution(execution, thread)
	if received != execution || ThreadCurrentThread(received) != caller {
		t.Fatal("direct Thread Runnable callback lost caller Execution or Thread identity")
	}
	if ObjectView[*Thread](caller, ObjectTypeID) != caller {
		t.Fatal("caller Thread Object view lost identity")
	}
}
