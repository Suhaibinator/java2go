package stdjava

import "testing"

func TestClassObjectContract99Identity(t *testing.T) {
	for _, id := range []TypeID{StringTypeID, JavaClassTypeID, "int", "[Ljava.lang.String;", "probe.Identity99"} {
		t.Run(string(id), func(t *testing.T) {
			class, alias := ClassLiteral(id), ClassLiteral(id)
			if class != alias {
				t.Fatal("literal identity not canonical")
			}
			execution := NewExecution()
			hash := ObjectHashCodeExecution(execution, class)
			for i := 0; i < 64; i++ {
				if ObjectHashCodeExecution(execution, alias) != hash || ObjectHashCodeExecution(NewExecution(), class) != hash {
					t.Fatal("identity hash changed")
				}
			}
			if !ObjectEqualsExecution(execution, class, alias) || !ObjectEqualsExecution(execution, alias, class) || !JavaReferenceEqual(class, alias) {
				t.Fatal("canonical alias equality failed")
			}
			if ObjectEqualsExecution(execution, class, nil) || ObjectEqualsExecution(execution, class, ClassLiteral("probe.OtherIdentity99")) || ObjectEqualsExecution(execution, class, NewJavaStringUTF16([]uint16{102, 111, 114, 101, 105, 103, 110})) {
				t.Fatal("class equality admitted null or distinct reference")
			}
			if ObjectGetClass(class) != ClassLiteral(JavaClassTypeID) {
				t.Fatal("Class object's runtime class not canonical java.lang.Class")
			}
		})
	}
}

func TestClassObjectContract99Null(t *testing.T) {
	var class *Class
	for _, body := range []func(){func() { ObjectHashCodeExecution(NewExecution(), class) }, func() { ObjectEqualsExecution(NewExecution(), class, ClassLiteral(StringTypeID)) }, func() { ObjectGetClass(class) }} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Error("null Class receiver did not throw NPE")
				}
			}()
			body()
		}()
	}
	if ObjectEqualsExecution(NewExecution(), ClassLiteral(StringTypeID), class) {
		t.Fatal("typed null equals argument not false")
	}
}
