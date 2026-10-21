package stdjava

import "testing"

func TestRunnableCanonicalDeclaredIdentity(t *testing.T) {
	if RunnableTypeID != TypeID("java.lang.Runnable") {
		t.Fatalf("runtime Runnable declaration remains noncanonical: %q", RunnableTypeID)
	}
	if ClassLiteral(RunnableTypeID).GetName() != "java.lang.Runnable" {
		t.Fatal("Runnable class literal names backend alias")
	}
	if !JavaTypeAssignable(ThreadTypeID, RunnableTypeID) {
		t.Fatal("Thread Runnable ancestry lost")
	}
}

func TestRunnableCanonicalArrayCastAndThread(t *testing.T) {
	task := NewPlainRunnableFuncAdapter(func() {})
	thread := NewThread(task)
	if ReferenceCheckCast[Runnable](task, TypeID("java.lang.Runnable")) != task || ReferenceCheckCast[Runnable](thread, TypeID("java.lang.Runnable")) != thread {
		t.Fatal("native SAM/Thread cast changed reference")
	}
	array := ReferenceArrayLiteral(RunnableTypeID, task, thread)
	if ReferenceArrayGet[Runnable](array, 0, RunnableTypeID) != task || ReferenceArrayGet[Runnable](array, 1, RunnableTypeID) != thread {
		t.Fatal("Runnable[] transport lost identity")
	}
	var absent Runnable
	if ReferenceCheckCast[Runnable](absent, RunnableTypeID) != nil {
		t.Fatal("typed-null SAM cast fabricated value")
	}
	defer func() {
		value := recover()
		if !CaughtAs(value, "ClassCastException") {
			t.Fatalf("native non-SAM acquired Runnable: %v", value)
		}
		want := "class java.lang.StringBuilder cannot be cast to class java.lang.Runnable (java.lang.StringBuilder and java.lang.Runnable are in module java.base of loader 'bootstrap')"
		if GetMessage(value) != want {
			t.Fatalf("actual JDK bytecode diagnostic mismatch: %q", GetMessage(value))
		}
	}()
	ReferenceCheckCast[Runnable](NewStringBuilder(), RunnableTypeID)
}

type canonicalRunnableSource struct{ *ObjectInfo }

func TestRunnableCanonicalSourceHomonymShield(t *testing.T) {
	const sourceID TypeID = "Runnable"
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
	source := &canonicalRunnableSource{}
	source.ObjectInfo = NewObjectInfo(sourceID, func(TypeID) any { return source })
	if ReferenceCheckCast[*canonicalRunnableSource](source, sourceID) != source {
		t.Fatal("source Runnable identity changed")
	}
	native := NewPlainRunnableFuncAdapter(func() {})
	if ObjectInstanceOf(source, RunnableTypeID) || ObjectInstanceOf(native, sourceID) {
		t.Fatal("source Runnable borrowed canonical native interface")
	}
	requireCheckCastFailure(t, func() { ReferenceCheckCast[Runnable](source, RunnableTypeID) })
	requireCheckCastFailure(t, func() { ReferenceCheckCast[*canonicalRunnableSource](native, sourceID) })
}
