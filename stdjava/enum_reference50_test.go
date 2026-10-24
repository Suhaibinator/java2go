package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// These objects model the agreed generated descriptor seam, not a Java library
// substitute. The original real-JVM project remains the compiler acceptance.
type enumRuntimeObject50 struct {
	metadata EnumMetadata
	actual   TypeID
	Calls    int32
}

func (value *enumRuntimeObject50) JavaEnumMetadata() *EnumMetadata { return &value.metadata }
func (value *enumRuntimeObject50) JavaDynamicTypeID() TypeID       { return value.actual }

func enumRuntimeOracleLine50(t *testing.T, prefix string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/enum_runtime50/oracle.stdout")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	t.Fatalf("frozen JVM observation lacks %q", prefix)
	return ""
}

func enumRuntimeObjects50() (*enumRuntimeObject50, *enumRuntimeObject50, *enumRuntimeObject50) {
	declaring := TypeID("probe.model.Choice")
	dynamic := TypeID("probe.model.Choice$1")
	other := TypeID("probe.model.Other")
	RegisterJavaType(declaring, EnumTypeID)
	RegisterJavaType(dynamic, declaring)
	RegisterJavaType(other, EnumTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: declaring, Enum: true})
	RegisterClassDescriptor(ClassDescriptor{Type: dynamic})
	RegisterClassDescriptor(ClassDescriptor{Type: other, Enum: true})
	return &enumRuntimeObject50{metadata: NewEnumMetadata("FIRST", 0, declaring, declaring), actual: declaring}, &enumRuntimeObject50{metadata: NewEnumMetadata("SPECIAL", 1, declaring, dynamic), actual: dynamic}, &enumRuntimeObject50{metadata: NewEnumMetadata("ONLY", 0, other, other), actual: other}
}

func TestEnumRuntimeMatchesFrozenJVM50(t *testing.T) {
	first, special, other := enumRuntimeObjects50()
	inspect := enumRuntimeOracleLine50(t, "inspect=")
	wantInspect := strings.Split(inspect, ":special:")[0]
	if got := fmt.Sprintf("%s:%d:%s", EnumName(special), EnumOrdinal(special), EnumGetDeclaringClass(special).GetName()); got != wantInspect {
		t.Errorf("metadata=%q; JVM=%q", got, wantInspect)
	}
	declaring := EnumGetDeclaringClass(special)
	dynamic := ObjectGetClass(special)
	flags := fmt.Sprintf("%t:%t:%t,ancestry=%t:%t,distinct=%t", ClassLiteral(EnumTypeID).IsEnum(), declaring.IsEnum(), dynamic.IsEnum(), ClassLiteral(EnumTypeID).IsAssignableFrom(declaring), declaring.IsAssignableFrom(dynamic), dynamic != declaring)
	if want := enumRuntimeOracleLine50(t, "enum-flags="); flags != want {
		t.Errorf("flags=%q; JVM=%q", flags, want)
	}
	if got := fmt.Sprint(EnumCompareTo(first, special)); got != enumRuntimeOracleLine50(t, "compare=") {
		t.Errorf("comparison=%s", got)
	}
	if EnumCompareTo(special, first) != 1 || EnumCompareTo(special, special) != 0 {
		t.Fatal("ordinal comparison failed")
	}
	expectBoxedException(t, "ClassCastException", func() { EnumCompareTo(special, other) })
	if special.JavaEnumMetadata().DeclaringTypeID() != declaring.TypeID() || special.JavaEnumMetadata().DynamicTypeID() != dynamic.TypeID() {
		t.Fatal("metadata lost declaring/dynamic identity")
	}
	for _, target := range []TypeID{ObjectTypeID, ComparableTypeID, SerializableTypeID, ConstableTypeID} {
		if !JavaTypeAssignable(EnumTypeID, target) {
			t.Errorf("Enum lacks %s ancestry", target)
		}
	}
}

func TestEnumRuntimeNominalAndNullBoundaries50(t *testing.T) {
	first, _, _ := enumRuntimeObjects50()
	var absent *enumRuntimeObject50
	for _, call := range []func(){func() { EnumName(absent) }, func() { EnumOrdinal(absent) }, func() { EnumGetDeclaringClass(absent) }, func() { EnumCompareTo(absent, first) }, func() { EnumCompareTo(first, absent) }} {
		expectBoxedException(t, "NullPointerException", call)
	}
	shadow := &enumRuntimeObject50{metadata: NewEnumMetadata("FIRST", 0, "probe.model.Choice", "probe.model.Choice"), actual: ObjectTypeID}
	expectBoxedException(t, "ClassCastException", func() { EnumName(shadow) })
	expectBoxedException(t, "ClassCastException", func() { EnumCompareTo(first, shadow) })
	var class *Class
	expectBoxedException(t, "NullPointerException", func() { class.IsEnum() })
	var field *Field
	expectBoxedException(t, "NullPointerException", func() { field.IsEnumConstant() })
}

func TestEnumReflectionStaticAndDeclaredBoundaries50(t *testing.T) {
	_, special, _ := enumRuntimeObjects50()
	declaring := EnumGetDeclaringClass(special)
	execution := NewExecution()
	initialized := 0
	reads := 0
	state := NewClassInitialization(string(declaring.TypeID()))
	RegisterClassDescriptor(ClassDescriptor{Type: declaring.TypeID(), Enum: true,
		Initialize: func(current *Execution) {
			if current != execution {
				t.Fatal("static initialization lost invoking Execution")
			}
			state.Ensure(current, func(*Execution) { initialized++ })
		},
		Fields: []FieldDescriptor{
			{Name: "SPECIAL", Type: declaring.TypeID(), Final: true, EnumConstant: true, StaticGet: func(current *Execution) any {
				if current != execution || initialized != 1 {
					t.Fatal("static getter ran without caller initialization")
				}
				reads++
				return special
			}},
			{Name: "calls", GoName: "Calls", Type: PrimitiveIntTypeID, NonPublic: true},
			{Name: "sentinel", GoName: "Calls", Type: PrimitiveIntTypeID, Final: true},
		},
	})
	constant := declaring.GetDeclaredField("SPECIAL")
	private := declaring.GetDeclaredField("calls")
	if initialized != 0 || reads != 0 {
		t.Fatal("registration, class literal, or field lookup initialized enum")
	}
	if got := constant.GetExecution(execution, nil); got != special {
		t.Fatal("static field copied or replaced singleton")
	}
	flags := fmt.Sprintf("%t:%t:%t", constant.IsEnumConstant(), private.IsEnumConstant(), constant.GetExecution(execution, "ignored receiver") == special)
	if want := enumRuntimeOracleLine50(t, "fields="); flags != want {
		t.Errorf("field flags/identity=%q; JVM=%q", flags, want)
	}
	if initialized != 1 || reads != 2 {
		t.Fatalf("initializations=%d,reads=%d", initialized, reads)
	}
	if declaring.GetField("sentinel").IsEnumConstant() {
		t.Fatal("final ordinary field mistaken for enum constant")
	}
	expectBoxedException(t, "NoSuchFieldException", func() { declaring.GetField("calls") })
	expectBoxedException(t, "IllegalAccessException", func() { private.GetExecution(execution, special) })
	expectBoxedException(t, "IllegalAccessException", func() { private.Set(special, BoxInteger(7)) })
	expectBoxedException(t, "IllegalAccessException", func() { constant.Set(nil, special) })
	expectBoxedException(t, "NoSuchFieldException", func() { declaring.GetDeclaredField("missing") })
	expectBoxedException(t, "NullPointerException", func() { declaring.GetDeclaredField(NullString()) })
	const childID TypeID = "probe.runtime.EnumReflectionChild50"
	RegisterJavaType(childID, declaring.TypeID())
	RegisterClassDescriptor(ClassDescriptor{Type: childID})
	child := ClassLiteral(childID)
	if child.GetField("SPECIAL").owner != declaring {
		t.Fatal("public inherited field owner changed")
	}
	expectBoxedException(t, "NoSuchFieldException", func() { child.GetDeclaredField("SPECIAL") })
}
