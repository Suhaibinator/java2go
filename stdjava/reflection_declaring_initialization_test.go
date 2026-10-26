package stdjava

import "testing"

type reflectionSharedInitFixture struct{ id TypeID }

func (value *reflectionSharedInitFixture) JavaDynamicTypeID() TypeID { return value.id }

// Generated direct active uses and reflection must inspect the same class
// coordinator. A recursive read is legal while the initializer runs, but it
// must not allow later reads after that initializer fails.
func TestReflectionSharedGeneratedInitialization(t *testing.T) {
	for _, fails := range []bool{false, true} {
		name := "success"
		if fails {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			id := TypeID("test.ReflectionSharedGenerated." + name)
			state := NewClassInitialization(string(id))
			execution, later := NewExecution(), NewExecution()
			callbacks, reads, constructors := 0, 0, 0
			value := int32(0)
			target := NewIllegalStateException("direct initializer target")
			RegisterJavaType(id, ObjectTypeID)
			RegisterClassDescriptor(ClassDescriptor{Type: id, Initialization: state,
				Initialize: func(e *Execution) {
					callbacks++
					state.Ensure(e, func(*Execution) { t.Fatal("reflection retried direct initializer") })
				},
				Construct: func(e *Execution) any {
					if e != later {
						t.Fatal("constructor lost execution")
					}
					constructors++
					return &reflectionSharedInitFixture{id: id}
				},
				Fields: []FieldDescriptor{{Name: "VALUE", Type: PrimitiveIntTypeID, StaticGet: func(*Execution) any { reads++; return value }}},
			})
			field := ClassLiteral(id).GetDeclaredField("VALUE")
			failure := reflection64Failure(func() {
				state.Ensure(execution, func(e *Execution) {
					if got := UnboxInteger(field.GetExecution(e, nil).(*Integer)); got != 0 {
						t.Fatalf("recursive value=%d", got)
					}
					if fails {
						panic(target)
					}
					value = 19
				})
			})
			if fails {
				if !CaughtAs(failure, "ExceptionInInitializerError") || !JavaReferenceEqual(GetCause(failure), target) {
					t.Fatalf("initializer failure=%v", failure)
				}
				expectBoxedException(t, "NoClassDefFoundError", func() { field.GetExecution(later, nil) })
				expectBoxedException(t, "NoClassDefFoundError", func() { ClassLiteral(id).GetDeclaredField("VALUE").GetExecution(later, nil) })
				if reads != 1 || constructors != 0 {
					t.Fatalf("failed class reads=%d constructors=%d", reads, constructors)
				}
			} else {
				if failure != nil {
					t.Fatalf("successful initializer=%v", failure)
				}
				if got := UnboxInteger(field.GetExecution(later, nil).(*Integer)); got != 19 {
					t.Fatalf("later value=%d", got)
				}
				ClassLiteral(id).GetDeclaredField("VALUE").GetExecution(later, nil)
				ClassForName(later, string(id))
				ClassLiteral(id).GetConstructor().NewInstance(later)
				if callbacks != 1 || reads != 3 || constructors != 1 {
					t.Fatalf("callbacks=%d reads=%d constructors=%d", callbacks, reads, constructors)
				}
			}
		})
	}
}
