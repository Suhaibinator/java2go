package stdjava

import "testing"

func TestClassComponentTypeDescriptorIdentity(t *testing.T) {
	components := []TypeID{ObjectTypeID, StringTypeID, "component.Source$Inner", "component.Choice"}
	for _, primitive := range []string{"boolean", "byte", "short", "char", "int", "long", "float", "double"} {
		components = append(components, PrimitiveTypeID(primitive))
	}
	for _, component := range components {
		t.Run(string(component), func(t *testing.T) {
			array := ClassLiteral(ArrayTypeID(component))
			want := ClassLiteral(component)
			for _, execution := range []*Execution{nil, NewExecution()} {
				if got := ClassGetComponentTypeExecution(execution, array); got != want || got != array.GetComponentType() {
					t.Fatalf("component identity = %p, want canonical %p", got, want)
				}
			}
			matrix := ClassLiteral(ArrayTypeID(array.TypeID()))
			if got := matrix.GetComponentType(); got != array || got.GetComponentType() != want {
				t.Fatal("component accessor did not strip exactly one rank")
			}
		})
	}
	for _, id := range []TypeID{ObjectTypeID, StringTypeID, PrimitiveTypeID("int"), "void", "component.Source$Inner"} {
		if ClassLiteral(id).GetComponentType() != nil {
			t.Fatalf("nonarray %s has component", id)
		}
	}
}

func TestClassComponentTypeDoesNotInitialize(t *testing.T) {
	const id TypeID = "component.Uninitialized125"
	initializations := 0
	RegisterClassDescriptor(ClassDescriptor{Type: id, Initialize: func(*Execution) { initializations++ }})
	array := ClassLiteral(ArrayTypeID(id))
	if ClassGetComponentTypeExecution(NewExecution(), array) != ClassLiteral(id) || initializations != 0 {
		t.Fatal("component query initialized or changed component identity")
	}
}

func TestClassComponentTypeNullReceiver(t *testing.T) {
	var class *Class
	for _, body := range []func(){
		func() { class.GetComponentType() },
		func() { ClassGetComponentTypeExecution(nil, class) },
		func() { ClassGetComponentTypeExecution(NewExecution(), class) },
	} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Error("null Class component query did not throw NPE")
				}
			}()
			body()
		}()
	}
}
