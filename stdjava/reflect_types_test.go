package stdjava

import "testing"

type reflectedDefaultTypeName struct{ execution *Execution }

func (p *reflectedDefaultTypeName) StringJava2goExecution(execution *Execution) string {
	p.execution = execution
	return "virtual-default"
}

type reflectedOverrideTypeName struct{ execution *Execution }

func (p *reflectedOverrideTypeName) GetTypeNameJava2goExecution(execution *Execution) string {
	p.execution = execution
	return "override-name"
}
func (*reflectedOverrideTypeName) String() string { return "unused-toString" }

func TestReflectTypeNameDispatch(t *testing.T) {
	execution := NewExecution()
	fallback := &reflectedDefaultTypeName{}
	override := &reflectedOverrideTypeName{}
	if got := ReflectTypeNameExecution(execution, fallback); got != "virtual-default" || fallback.execution != execution {
		t.Fatalf("default dispatch %q", got)
	}
	if got := ReflectTypeNameExecution(execution, override); got != "override-name" || override.execution != execution {
		t.Fatalf("override dispatch %q", got)
	}
	defer func() {
		failure := recover()
		if failure == nil || !CaughtAs(failure, "NullPointerException") {
			t.Fatalf("null Type failure=%v", failure)
		}
	}()
	ReflectTypeNameExecution(execution, nil)
}

func TestClassReflectTypeIdentity(t *testing.T) {
	class := ClassLiteral(ArrayTypeID(ArrayTypeID(StringTypeID)))
	if class.GetTypeName() != "java.lang.String[][]" {
		t.Fatal(class.GetTypeName())
	}
	if !ObjectInstanceOf(class, ReflectTypeTypeID) || !ObjectInstanceOf(class, JavaClassTypeID) {
		t.Fatal("Class is not nominally a Type and Class")
	}
	if ObjectGetClass(class) != ClassLiteral(JavaClassTypeID) {
		t.Fatal("Class identity was confused with represented type")
	}
	if ClassLiteral(PrimitiveIntTypeID).GetTypeName() != "int" {
		t.Fatal("primitive name")
	}
}
