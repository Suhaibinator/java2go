package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Reflection initializes the declaring class after access permission and before
// a first read or refused static final write. Independent wrappers share that
// class state while each getter/setter retains its own invoking execution.
func TestReflectionStaticAccessorInitialization(t *testing.T) {
	t.Run("read_then_final_write_with_new_execution", func(t *testing.T) {
		const id TypeID = "test.ReflectionAccessor.FinalAfterRead"
		first, later := NewExecution(), NewExecution()
		initializers, reads := 0, 0
		RegisterJavaType(id, ObjectTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(e *Execution) {
			if e != first {
				t.Fatal("first accessor lost invoking execution")
			}
			initializers++
		}, Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, Final: true, StaticGet: func(e *Execution) any {
			if e != first && e != later {
				t.Fatal("getter lost invoking execution")
			}
			reads++
			return int32(7)
		}}}})
		field := ClassLiteral(id).GetDeclaredField("VALUE")
		if initializers != 0 {
			t.Fatal("lookup initialized class")
		}
		field.GetExecution(first, nil)
		field.GetExecution(later, "ignored")
		// Independently looked-up wrappers and Class.forName share the same class.
		ClassLiteral(id).GetDeclaredField("VALUE").GetExecution(later, nil)
		ClassForName(later, string(id))
		expectBoxedException(t, "IllegalAccessException", func() { field.Set(nil, BoxInteger(9)) })
		if initializers != 1 || reads != 3 {
			t.Fatalf("initializers=%d reads=%d", initializers, reads)
		}
	})
	t.Run("first_final_write_initializes_before_refusal", func(t *testing.T) {
		const id TypeID = "test.ReflectionAccessor.FinalFirst"
		execution := NewExecution()
		initializers, writes := 0, 0
		RegisterJavaType(id, ObjectTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(e *Execution) {
			if e != execution {
				t.Fatal("first final write lost execution")
			}
			initializers++
		}, Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, Final: true, StaticGet: func(*Execution) any { return int32(7) }, StaticSet: func(*Execution, any) { writes++ }}}})
		field := ClassLiteral(id).GetDeclaredField("VALUE")
		expectBoxedException(t, "IllegalAccessException", func() { field.SetExecution(execution, nil, BoxInteger(9)) })
		if initializers != 1 || writes != 0 {
			t.Fatalf("initializers=%d writes=%d", initializers, writes)
		}
	})
	t.Run("inherited_static_and_denied_access", func(t *testing.T) {
		const base, child TypeID = "test.ReflectionAccessor.Base", "test.ReflectionAccessor.Child"
		execution := NewExecution()
		baseInitializers, childInitializers := 0, 0
		RegisterJavaType(base, ObjectTypeID)
		RegisterJavaType(child, base)
		getter := func(e *Execution) any {
			if e != execution {
				t.Fatal("inherited getter lost execution")
			}
			return int32(11)
		}
		RegisterClassDescriptor(ClassDescriptor{Type: base, Initialize: func(e *Execution) {
			if e != execution {
				t.Fatal("base initializer lost execution")
			}
			baseInitializers++
		}, Fields: []FieldDescriptor{{Name: "VISIBLE", Type: PrimitiveIntTypeID, StaticGet: getter}, {Name: "HIDDEN", Type: PrimitiveIntTypeID, NonPublic: true, StaticGet: getter}}})
		RegisterClassDescriptor(ClassDescriptor{Type: child, Initialize: func(*Execution) { childInitializers++ }})
		hidden := ClassLiteral(base).GetDeclaredField("HIDDEN")
		expectBoxedException(t, "IllegalAccessException", func() { hidden.SetExecution(execution, nil, BoxInteger(1)) })
		if baseInitializers != 0 || childInitializers != 0 {
			t.Fatal("denied access initialized class")
		}
		field := ClassLiteral(child).GetField("VISIBLE")
		if field.GetDeclaringClass() != ClassLiteral(base) {
			t.Fatal("inherited field lost declaring class")
		}
		field.GetExecution(execution, nil)
		if baseInitializers != 1 || childInitializers != 0 {
			t.Fatalf("base=%d child=%d", baseInitializers, childInitializers)
		}
	})
	t.Run("initializer_failure_before_final_refusal", func(t *testing.T) {
		const id TypeID = "test.ReflectionAccessor.Failed"
		execution := NewExecution()
		state := NewClassInitialization(string(id))
		target := NewIllegalStateException("initializer target")
		bodies, callbacks := 0, 0
		RegisterJavaType(id, ObjectTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(e *Execution) {
			callbacks++
			state.Ensure(e, func(*Execution) { bodies++; panic(target) })
		}, Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, Final: true, StaticGet: func(*Execution) any { t.Fatal("failed class getter ran"); return nil }}}})
		field := ClassLiteral(id).GetDeclaredField("VALUE")
		failure := reflection64Failure(func() { field.SetExecution(execution, nil, BoxInteger(1)) })
		if !CaughtAs(failure, "ExceptionInInitializerError") || !JavaReferenceEqual(GetCause(failure), target) {
			t.Fatalf("first initializer failure=%v", failure)
		}
		expectBoxedException(t, "NoClassDefFoundError", func() { field.GetExecution(NewExecution(), nil) })
		if callbacks != 1 || bodies != 1 {
			t.Fatalf("callbacks=%d bodies=%d", callbacks, bodies)
		}
	})
	t.Run("reentrant_same_execution", func(t *testing.T) {
		const id TypeID = "test.ReflectionAccessor.Reentrant"
		execution := NewExecution()
		callbacks, reads, observed := 0, 0, int32(-1)
		state := NewClassInitialization(string(id))
		var field *Field
		value := int32(0)
		RegisterJavaType(id, ObjectTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(e *Execution) {
			callbacks++
			state.Ensure(e, func(*Execution) { observed = UnboxInteger(field.GetExecution(e, nil).(*Integer)); value = 13 })
		}, Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, StaticGet: func(e *Execution) any {
			if e != execution {
				t.Fatal("reentrant read lost execution")
			}
			reads++
			return value
		}}}})
		field = ClassLiteral(id).GetDeclaredField("VALUE")
		if got := UnboxInteger(field.GetExecution(execution, nil).(*Integer)); got != 13 {
			t.Fatalf("value=%d", got)
		}
		if observed != 0 || callbacks != 1 || reads != 2 {
			t.Fatalf("observed=%d callbacks=%d reads=%d", observed, callbacks, reads)
		}
	})
	t.Run("concurrent_accessor_creation", func(t *testing.T) {
		const id TypeID = "test.ReflectionAccessor.Concurrent"
		var callbacks, reads atomic.Int32
		RegisterJavaType(id, ObjectTypeID)
		RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(*Execution) { callbacks.Add(1) }, Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, StaticGet: func(*Execution) any { reads.Add(1); return int32(17) }}}})
		field := ClassLiteral(id).GetDeclaredField("VALUE")
		var workers sync.WaitGroup
		for index := 0; index < 12; index++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				if got := UnboxInteger(field.GetExecution(NewExecution(), nil).(*Integer)); got != 17 {
					t.Errorf("value=%d", got)
				}
			}()
		}
		workers.Wait()
		if callbacks.Load() != 1 || reads.Load() != 12 {
			t.Fatalf("callbacks=%d reads=%d", callbacks.Load(), reads.Load())
		}
	})
}
