package stdjava

import "testing"

// The null/access order follows JDK21 Field.checkAccess: instance access
// obtains obj.getClass() before checking access; static access ignores obj.
// The unchanged enum-reflection-init-v2 JVM oracle independently observes the
// null instance and denied static cases before the class is initialized.
func TestReflectionInstanceNullBeforeAccess(t *testing.T) {
	field := &Field{descriptor: FieldDescriptor{NonPublic: true}}
	execution := NewExecution()
	var typedNull *reflectionFixture
	for _, receiver := range []any{nil, typedNull} {
		expectBoxedException(t, "NullPointerException", func() { field.Get(receiver) })
		expectBoxedException(t, "NullPointerException", func() { field.GetExecution(execution, receiver) })
		expectBoxedException(t, "NullPointerException", func() { field.Set(receiver, nil) })
	}
}

func TestReflectionPrivateWrongReceiverStillChecksAccessFirst(t *testing.T) {
	field := &Field{owner: registerReflectionFixture(), descriptor: FieldDescriptor{NonPublic: true}}
	// A nonnull receiver is inspected for its class only; type compatibility
	// belongs to the accessor, after access permission. Do not eagerly run
	// reflectionReceiver before checking NonPublic.
	expectBoxedException(t, "IllegalAccessException", func() { field.Get("wrong receiver") })
	expectBoxedException(t, "IllegalAccessException", func() { field.GetExecution(NewExecution(), "wrong receiver") })
	expectBoxedException(t, "IllegalAccessException", func() { field.Set("wrong receiver", nil) })
}

func TestReflectionStaticIgnoresReceiverAndPreservesInitialization(t *testing.T) {
	const id TypeID = "test.ReflectionReceiverOrder.Static"
	execution := NewExecution()
	initialized, reads := 0, 0
	state := NewClassInitialization(string(id))
	RegisterJavaType(id, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(actual *Execution) {
		if actual != execution {
			t.Fatal("initializer lost caller execution")
		}
		state.Ensure(actual, func(*Execution) { initialized++ })
	}})
	getter := func(actual *Execution) any {
		if actual != execution {
			t.Fatal("getter lost caller execution")
		}
		reads++
		return int32(7)
	}
	private := &Field{owner: ClassLiteral(id), descriptor: FieldDescriptor{NonPublic: true, StaticGet: getter, Type: PrimitiveIntTypeID}}
	for _, receiver := range []any{nil, "ignored"} {
		expectBoxedException(t, "IllegalAccessException", func() { private.Get(receiver) })
		expectBoxedException(t, "IllegalAccessException", func() { private.GetExecution(execution, receiver) })
		expectBoxedException(t, "IllegalAccessException", func() { private.Set(receiver, nil) })
	}
	if initialized != 0 || reads != 0 {
		t.Fatal("denied static access initialized or read the class")
	}
	public := &Field{owner: ClassLiteral(id), descriptor: FieldDescriptor{StaticGet: getter, Type: PrimitiveIntTypeID}}
	for _, receiver := range []any{nil, "ignored"} {
		if got := UnboxInteger(public.GetExecution(execution, receiver).(*Integer)); got != 7 {
			t.Fatalf("read = %d", got)
		}
	}
	if initialized != 1 || reads != 2 {
		t.Fatalf("initializations=%d reads=%d", initialized, reads)
	}
}
