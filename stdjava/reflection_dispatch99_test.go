package stdjava

import "testing"

type reflectionDispatch99Fixture struct {
	id        TypeID
	execution *Execution
	calls     int
}

func (v *reflectionDispatch99Fixture) JavaDynamicTypeID() TypeID { return v.id }
func (v *reflectionDispatch99Fixture) Base(e *Execution) int32 {
	if e != v.execution {
		panic("lost execution")
	}
	v.calls++
	return 11
}
func (v *reflectionDispatch99Fixture) Child(e *Execution) int32 {
	if e != v.execution {
		panic("lost execution")
	}
	v.calls++
	return 29
}
func (v *reflectionDispatch99Fixture) WrongReturn(e *Execution) int32 {
	panic("wrong return descriptor")
}
func (v *reflectionDispatch99Fixture) WrongOverload(e *Execution, x int32) int32 {
	panic("wrong overload")
}

func TestReflectionDispatch99VirtualAccess(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		baseModifiers, childModifiers int32
		samePackage                   bool
		want                          int32
	}{
		{"protected", 4, 4, false, 29}, {"public", 1, 1, false, 29}, {"package_same", 0, 0, true, 29},
		{"package_foreign_shadow", 0, 1, false, 11}, {"private", 2, 1, true, 11},
		{"private_shadow", 0, 2, false, 11}, {"static_shadow", 0, 9, false, 11}, {"final", 17, 1, true, 11},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := "dispatch99." + tc.name
			base, child := TypeID(prefix+".base.Base"), TypeID(prefix+".child.Child")
			if tc.samePackage {
				child = TypeID(prefix + ".base.Child")
			}
			RegisterJavaType(base, ObjectTypeID)
			RegisterJavaType(child, base)
			RegisterClassDescriptor(ClassDescriptor{Type: base, Methods: []MethodDescriptor{{Name: "value", GoName: "Base", Return: PrimitiveIntTypeID, Modifiers: tc.baseModifiers, HasModifiers: true}}})
			RegisterClassDescriptor(ClassDescriptor{Type: child, Methods: []MethodDescriptor{
				{Name: "value", GoName: "Child", Return: PrimitiveIntTypeID, Modifiers: tc.childModifiers, HasModifiers: true},
				{Name: "value", GoName: "WrongReturn", Return: ObjectTypeID, Modifiers: 1, HasModifiers: true},
				{Name: "value", GoName: "WrongOverload", Return: PrimitiveIntTypeID, ParameterTypes: []TypeID{PrimitiveIntTypeID}, Modifiers: 1, HasModifiers: true},
			}})
			e := NewExecution()
			v := &reflectionDispatch99Fixture{id: child, execution: e}
			m := ClassLiteral(base).GetDeclaredMethod("value")
			m.SetAccessible(true)
			if got := UnboxInteger(m.Invoke(e, v).(*Integer)); got != tc.want {
				t.Fatalf("dispatch=%d want=%d", got, tc.want)
			}
			if v.calls != 1 {
				t.Fatalf("calls=%d", v.calls)
			}
			expectBoxedException(t, "IllegalArgumentException", func() { m.Invoke(e, v, BoxInteger(1)) })
			if v.calls != 1 {
				t.Fatal("argument rejection invoked body")
			}
		})
	}
}
func TestReflectionDispatch99PackageOverrideChain(t *testing.T) {
	base, middle, child := TypeID("dispatch99.chain.base.Base"), TypeID("dispatch99.chain.base.Middle"), TypeID("dispatch99.chain.child.Child")
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(middle, base)
	RegisterJavaType(child, middle)
	for _, row := range []struct {
		id        TypeID
		modifiers int32
		goName    string
	}{{base, 0, "Base"}, {middle, 1, "Base"}, {child, 1, "Child"}} {
		RegisterClassDescriptor(ClassDescriptor{Type: row.id, Methods: []MethodDescriptor{{Name: "value", GoName: row.goName, Return: PrimitiveIntTypeID, Modifiers: row.modifiers, HasModifiers: true}}})
	}
	e := NewExecution()
	v := &reflectionDispatch99Fixture{id: child, execution: e}
	m := ClassLiteral(base).GetDeclaredMethod("value")
	m.SetAccessible(true)
	if got := UnboxInteger(m.Invoke(e, v).(*Integer)); got != 29 {
		t.Fatalf("transitive override=%d want=29", got)
	}
}
func TestReflectionDispatch99InterfaceStaticLookup(t *testing.T) {
	root, leaf, child, base, sub := TypeID("dispatch99.api.Root"), TypeID("dispatch99.api.Leaf"), TypeID("dispatch99.impl.Child"), TypeID("dispatch99.api.Base"), TypeID("dispatch99.api.Sub")
	RegisterJavaType(root, ObjectTypeID)
	RegisterJavaType(leaf, ObjectTypeID, root)
	RegisterJavaType(child, ObjectTypeID, leaf)
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(sub, base)
	marker := MethodDescriptor{Name: "marker", Return: PrimitiveIntTypeID, Modifiers: 9, HasModifiers: true, StaticFunction: func(*Execution) int32 { return 7 }}
	RegisterClassDescriptor(ClassDescriptor{Type: root, Interface: true, Methods: []MethodDescriptor{marker, {Name: "label", GoName: "Base", Return: PrimitiveIntTypeID, Modifiers: 1, HasModifiers: true}}})
	RegisterClassDescriptor(ClassDescriptor{Type: leaf, Interface: true})
	RegisterClassDescriptor(ClassDescriptor{Type: child})
	RegisterClassDescriptor(ClassDescriptor{Type: base, Methods: []MethodDescriptor{marker}})
	for _, id := range []TypeID{leaf, child} {
		expectBoxedException(t, "NoSuchMethodException", func() { ClassLiteral(id).GetMethod("marker") })
		if m := ClassLiteral(id).GetMethod("label"); m.GetDeclaringClass().TypeID() != root {
			t.Fatal("lost inherited instance interface method")
		}
	}
	for _, id := range []TypeID{root, sub} {
		if got := UnboxInteger(ClassLiteral(id).GetMethod("marker").Invoke(NewExecution(), nil).(*Integer)); got != 7 {
			t.Fatalf("direct/interface or inherited/class static=%d", got)
		}
	}
}
