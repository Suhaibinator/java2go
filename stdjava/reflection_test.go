package stdjava

import "testing"

type reflectionFixture struct {
	Count int32
	Label string
	Fixed int32
}

func (*reflectionFixture) JavaDynamicTypeID() TypeID             { return "test.ReflectionFixture" }
func (value *reflectionFixture) FailExecution(*Execution)        { panic(NewIllegalStateException("target")) }
func (value *reflectionFixture) CountExecution(*Execution) int32 { return value.Count }

func registerReflectionFixture() *Class {
	const id TypeID = "test.ReflectionFixture"
	RegisterJavaType(id, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{
		Type:      id,
		Construct: func(*Execution) any { return &reflectionFixture{} },
		Fields:    []FieldDescriptor{{Name: "count", GoName: "Count", Type: PrimitiveIntTypeID}, {Name: "label", GoName: "Label", Type: StringTypeID}, {Name: "fixed", GoName: "Fixed", Type: PrimitiveIntTypeID, Final: true}},
		Methods:   []MethodDescriptor{{Name: "fail", GoName: "FailExecution"}, {Name: "count", GoName: "CountExecution", Return: PrimitiveIntTypeID}},
	})
	return ClassLiteral(id)
}
func TestReflectionCheckedBoundaries(t *testing.T) {
	class := registerReflectionFixture()
	execution := NewExecution()
	value := class.GetConstructor().NewInstance(execution)
	field := class.GetField("count")
	field.Set(value, BoxByte(4))
	if got := UnboxInteger(field.Get(value).(*Integer)); got != 4 {
		t.Fatalf("count = %d", got)
	}
	if got := UnboxInteger(class.GetMethod("count").Invoke(execution, value).(*Integer)); got != 4 {
		t.Fatalf("method count = %d", got)
	}
	class.GetField("label").Set(value, nil)
	if class.GetField("label").Get(value) != nil {
		t.Fatal("null String field did not return null Object")
	}
	for _, test := range []struct {
		name, exception string
		call            func()
	}{
		{"null receiver", "NullPointerException", func() { field.Get(nil) }},
		{"wrong receiver", "IllegalArgumentException", func() { field.Get("text") }},
		{"narrowing", "IllegalArgumentException", func() { field.Set(value, BoxLong(1)) }},
		{"null primitive", "IllegalArgumentException", func() { field.Set(value, nil) }},
		{"wrong boxed kind", "IllegalArgumentException", func() { field.Set(value, BoxBoolean(true)) }},
		{"final", "IllegalAccessException", func() { class.GetField("fixed").Set(value, BoxInteger(2)) }},
		{"missing field", "NoSuchFieldException", func() { class.GetField("missing") }},
		{"missing method", "NoSuchMethodException", func() { class.GetMethod("missing") }},
		{"wrong method arity", "IllegalArgumentException", func() { class.GetMethod("count").Invoke(execution, value, BoxInteger(2)) }},
		{"wrong constructor arity", "IllegalArgumentException", func() { class.GetConstructor().NewInstance(execution, BoxInteger(2)) }},
		{"missing class", "ClassNotFoundException", func() { ClassForName(execution, "test.missing.ReflectionFixture") }},
	} {
		t.Run(test.name, func(t *testing.T) { expectBoxedException(t, test.exception, test.call) })
	}
}
func TestReflectionWrapsInvocationCause(t *testing.T) {
	class := registerReflectionFixture()
	execution := NewExecution()
	value := class.GetConstructor().NewInstance(execution)
	defer func() {
		failure := recover()
		if !CaughtAs(failure, "InvocationTargetException") || !CaughtAs(failure, "ReflectiveOperationException") {
			t.Fatalf("failure = %v", failure)
		}
		cause := GetCause(failure)
		if !CaughtAs(cause, "IllegalStateException") || GetMessage(cause) != "target" {
			t.Fatalf("cause = %v", cause)
		}
	}()
	class.GetMethod("fail").Invoke(execution, value)
	t.Fatal("expected wrapped target exception")
}
func TestReflectionRegistryDoesNotInitializeOnRegistrationOrLiteral(t *testing.T) {
	const id TypeID = "test.ReflectionInitialization"
	initialized := false
	state := NewClassInitialization(string(id))
	RegisterJavaType(id, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(execution *Execution) { state.Ensure(execution, func(*Execution) { initialized = true }) }})
	literal := ClassLiteral(id)
	if initialized {
		t.Fatal("registration/literal initialized class")
	}
	if ClassForName(NewExecution(), string(id)) != literal || !initialized {
		t.Fatal("forName did not initialize canonical class")
	}
}
