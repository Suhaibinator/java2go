package stdjava

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These records are the complete raw observations from independently executed
// JDK21 projects. The runtime fixture exercises descriptor APIs; original Gson
// translation and generated annotation accessors remain separate gates.
func reflection64Actual(t *testing.T, group string, seed int) string {
	t.Helper()
	var first []byte
	for repeat := 1; repeat <= 3; repeat++ {
		path := filepath.Join("testdata", "reflection_metadata64", "actual-oracles", group, fmt.Sprintf("seed-%d-repeat-%d", seed, repeat))
		output, err := os.ReadFile(path + ".stdout")
		if err != nil {
			t.Fatal(err)
		}
		stderr, err := os.ReadFile(path + ".stderr")
		if err != nil {
			t.Fatal(err)
		}
		if len(stderr) != 0 {
			t.Fatal("independent oracle stderr is not empty")
		}
		if repeat == 1 {
			first = output
		} else if !bytes.Equal(first, output) {
			t.Fatal("independent raw oracle repeats differ")
		}
	}
	return string(first)
}
func reflection64HostText(text *JavaString) string {
	return string(unsignedBytes(JavaStringGetBytes(text, UTF_8).Elements))
}

func reflection64Failure(call func()) (caught any) {
	defer func() { caught = recover() }()
	call()
	return nil
}
func reflection64ExceptionName(t *testing.T, caught any) string {
	t.Helper()
	if caught == nil {
		t.Fatal("expected Java exception")
	}
	return ObjectGetClass(caught).GetName()
}

type reflection64State struct {
	Visible        int32
	hidden         *JavaString
	Labels         any
	PackageValue   int32
	TransientValue int32
	VolatileValue  int32
	inherited      int32
	execution      *Execution
}

func (*reflection64State) JavaDynamicTypeID() TypeID { return "probe.model.State" }
func (state *reflection64State) Reflection64HiddenGetJava2goExecution(execution *Execution) *JavaString {
	state.execution = execution
	return state.hidden
}
func (state *reflection64State) Reflection64HiddenSetJava2goExecution(execution *Execution, value *JavaString) {
	state.execution = execution
	state.hidden = value
}
func reflection64FieldClass(execution *Execution, shared *int32) *Class {
	const base TypeID = "probe.model.Base"
	const id TypeID = "probe.model.State"
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(id, base)
	RegisterClassDescriptor(ClassDescriptor{Type: base, Fields: []FieldDescriptor{{Name: "inherited", GoName: "inherited", Type: PrimitiveIntTypeID, NonPublic: true, HasModifiers: true, Modifiers: 4}}})
	list := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: "java.util.List", Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: StringTypeID}}}
	RegisterClassDescriptor(ClassDescriptor{Type: id, HasModifiers: true, Modifiers: 17, Fields: []FieldDescriptor{
		{Name: "visible", GoName: "Visible", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 1},
		{Name: "hidden", Type: StringTypeID, NonPublic: true, HasModifiers: true, Modifiers: 2,
			Get: func(e *Execution, r any) any {
				return ReflectGeneratedFieldGetExecution(e, r, id, "Reflection64HiddenGetJava2goExecution")
			},
			Set: func(e *Execution, r, v any) {
				ReflectGeneratedFieldSetExecution(e, r, id, "Reflection64HiddenSetJava2goExecution", v)
			}},
		{Name: "labels", GoName: "Labels", Type: "java.util.List", NonPublic: true, HasModifiers: true, Modifiers: 4, GenericType: &list},
		{Name: "packageValue", GoName: "PackageValue", Type: PrimitiveIntTypeID, NonPublic: true, HasModifiers: true, Modifiers: 0},
		{Name: "shared", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 9, StaticGet: func(e *Execution) any {
			if e != execution {
				panic("getter lost execution")
			}
			return *shared
		}, StaticSet: func(e *Execution, v any) {
			if e != execution {
				panic("setter lost execution")
			}
			*shared = v.(int32)
		}},
		{Name: "FIXED", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 25, Final: true, StaticGet: func(*Execution) any { return int32(11) }},
		{Name: "transientValue", GoName: "TransientValue", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 129},
		{Name: "volatileValue", GoName: "VolatileValue", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65},
	}})
	return ClassLiteral(id)
}
func TestCampaignReflectionDeclaredFieldsActualJDK64(t *testing.T) {
	for _, seed := range []int{17, 41, 97} {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			execution := NewExecution()
			shared := int32(3)
			class := reflection64FieldClass(execution, &shared)
			state := &reflection64State{Visible: int32(seed), hidden: JavaStringFromHostUTF8(fmt.Sprintf("initial-%d", seed)), PackageValue: int32(seed + 1), TransientValue: 13, VolatileValue: 17, inherited: 7}
			var output strings.Builder
			fields := class.GetDeclaredFields()
			for _, entry := range ReferenceArrayIterationElements(fields) {
				field := entry.(*Field)
				fmt.Fprintf(&output, "field|%s|%d|%s|%s|%t\n", field.GetName(), field.GetModifiers(), field.GetType().GetName(), ReflectTypeNameExecution(execution, field.GetGenericType()), field.GetDeclaringClass() == class)
			}
			fmt.Fprintf(&output, "inventory|%d\n", ReferenceArrayLength(fields))
			fmt.Fprintf(&output, "inherited|%s\n", reflection64ExceptionName(t, reflection64Failure(func() { class.GetDeclaredFieldJavaString(JavaStringFromHostUTF8("inherited")) })))
			hidden := class.GetDeclaredFieldJavaString(JavaStringFromHostUTF8("hidden"))
			fmt.Fprintf(&output, "denied|%s\n", reflection64ExceptionName(t, reflection64Failure(func() { hidden.GetExecution(execution, state) })))
			hidden.SetAccessible(true)
			hidden.SetExecution(execution, state, JavaStringFromHostUTF8(fmt.Sprintf("changed-%d", seed)))
			visible := class.GetDeclaredFieldJavaString(JavaStringFromHostUTF8("visible"))
			visible.SetExecution(execution, state, BoxInteger(int32(seed+2)))
			static := class.GetDeclaredFieldJavaString(JavaStringFromHostUTF8("shared"))
			static.SetExecution(execution, nil, BoxInteger(int32(seed%7)))
			fmt.Fprintf(&output, "final|%s\n", reflection64ExceptionName(t, reflection64Failure(func() {
				class.GetDeclaredFieldJavaString(JavaStringFromHostUTF8("FIXED")).SetExecution(execution, nil, BoxInteger(int32(seed)))
			})))
			fmt.Fprintf(&output, "state|%d|%s|%d|%d|%d|%s|%d|%d\n", state.Visible, reflection64HostText(state.hidden), state.PackageValue, shared, state.inherited, reflection64HostText(hidden.GetExecution(execution, state).(*JavaString)), UnboxInteger(visible.GetExecution(execution, state).(*Integer)), UnboxInteger(static.GetExecution(execution, nil).(*Integer)))
			if got, want := output.String(), reflection64Actual(t, "fields", seed); got != want {
				t.Fatalf("complete field record mismatch\ngot:\n%swant:\n%s", got, want)
			}
			if state.execution != execution {
				t.Fatal("generated private accessor lost logical execution")
			}
		})
	}
}

type reflection64GenericState struct{ value *JavaString }

func reflection64GenericClasses() (*Class, *Class, *Class) {
	const base TypeID = "probe.model.Base"
	const order TypeID = "probe.model.Order"
	const anonymous TypeID = "probe.app.Main$1"
	RegisterJavaType(base, ObjectTypeID)
	RegisterJavaType(order, base)
	RegisterJavaType(anonymous, base)
	RegisterClassDescriptor(ClassDescriptor{Type: base, TypeParameters: []TypeVariableDescriptor{{Name: "T", Bounds: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: ObjectTypeID}}}}})
	listString := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: "java.util.List", Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: StringTypeID}}}
	mapStringInt := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: "java.util.Map", Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: StringTypeID}, {Kind: ReflectClassKind, Raw: IntegerTypeID}}}
	inherited := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: base, Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: StringTypeID}}}
	RegisterClassDescriptor(ClassDescriptor{Type: order, GenericSuperclass: &inherited, Fields: []FieldDescriptor{{Name: "lines", Type: "java.util.List", GenericType: &listString}, {Name: "metadata", Type: "java.util.Map", GenericType: &mapStringInt}}})
	captured := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: base, Arguments: []ReflectTypeDescriptor{{Kind: ReflectParameterizedKind, Raw: "java.util.List", Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: order}}}}}
	RegisterClassDescriptor(ClassDescriptor{Type: anonymous, GenericSuperclass: &captured})
	return ClassLiteral(base), ClassLiteral(order), ClassLiteral(anonymous)
}
func reflection64TypeArguments(execution *Execution, value any) *ReferenceArray {
	return ReflectArrayMemberExecution(execution, value, ParameterizedTypeTypeID, "GetActualTypeArguments", ReflectTypeTypeID)
}
func reflection64RawType(execution *Execution, value any) any {
	return ReflectTypeMemberExecution(execution, value, ParameterizedTypeTypeID, "GetRawType")
}
func TestCampaignReflectionGenericTypesActualJDK64(t *testing.T) {
	for _, seed := range []int{17, 41, 97} {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			execution := NewExecution()
			base, order, anonymous := reflection64GenericClasses()
			inherited, captured := order.GetGenericSuperclass(), anonymous.GetGenericSuperclass()
			inheritedArgs := reflection64TypeArguments(execution, inherited)
			capturedArgs := reflection64TypeArguments(execution, captured)
			nested := ReferenceArrayGet[any](capturedArgs, 0, ReflectTypeTypeID)
			var output strings.Builder
			fmt.Fprintf(&output, "inherited|%s|%t|%t|%t\n", ReflectTypeNameExecution(execution, inherited), reflection64RawType(execution, inherited) == base, ReferenceArrayGet[any](inheritedArgs, 0, ReflectTypeTypeID) == ClassLiteral(StringTypeID), ReflectTypeMemberExecution(execution, inherited, ParameterizedTypeTypeID, "GetOwnerType") == nil)
			fmt.Fprintf(&output, "anonymous|%s|%t|%t|%t\n", ReflectTypeNameExecution(execution, captured), reflection64RawType(execution, captured) == base, reflection64RawType(execution, nested) == ClassLiteral("java.util.List"), ReferenceArrayGet[any](reflection64TypeArguments(execution, nested), 0, ReflectTypeTypeID) == order)
			for _, entry := range ReferenceArrayIterationElements(order.GetDeclaredFields()) {
				field := entry.(*Field)
				fmt.Fprintf(&output, "field|%s|%s\n", field.GetName(), ReflectTypeNameExecution(execution, field.GetGenericType()))
			}
			variable := ReferenceArrayGet[any](base.GetTypeParameters(), 0, TypeVariableTypeID)
			fmt.Fprintf(&output, "variable|%s|%t|%t\n", ReflectStringMemberExecution(execution, variable, TypeVariableTypeID, "GetName"), ReflectDeclarationMemberExecution(execution, variable, TypeVariableTypeID, "GetGenericDeclaration") == base, ReferenceArrayGet[any](ReflectArrayMemberExecution(execution, variable, TypeVariableTypeID, "GetBounds", ReflectTypeTypeID), 0, ReflectTypeTypeID) == ClassLiteral(ObjectTypeID))
			fmt.Fprintf(&output, "ordinary|%t|%t\n", ClassLiteral(ObjectTypeID).GetGenericSuperclass() == nil, ClassLiteral(StringTypeID).GetGenericSuperclass() == ClassLiteral(ObjectTypeID))
			// Preserve the source control's actual aliasing mutation in the runtime
			// fixture; original compiled Java/Gson remains a separate gate.
			state := &reflection64GenericState{value: JavaStringFromHostUTF8(fmt.Sprintf("seed-%d", seed))}
			container := []*reflection64GenericState{state}
			container[0].value = ConcatJavaStrings(container[0].value, JavaStringFromHostUTF8("-mutated"))
			fmt.Fprintf(&output, "state|%s|%d\n", reflection64HostText(state.value), len(container))
			if got, want := output.String(), reflection64Actual(t, "generic", seed); got != want {
				t.Fatalf("complete generic record mismatch\ngot:\n%swant:\n%s", got, want)
			}
		})
	}
}

// Registration ownership and canonical execution text are runtime ABI
// invariants, distinct from the raw independent JVM observation comparisons.
func TestCampaignReflectionMetadataRegistrationIsolation64(t *testing.T) {
	const id TypeID = "test.ReflectionMetadataIsolation64"
	RegisterJavaType(id, ObjectTypeID)
	initialized, factories := 0, 0
	generic := ReflectTypeDescriptor{Kind: ReflectParameterizedKind, Raw: "java.util.List", Arguments: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: StringTypeID}}}
	fields := []FieldDescriptor{{Name: "data", Type: "java.util.List", GenericType: &generic}}
	parameters := []TypeVariableDescriptor{{Name: "T", Bounds: []ReflectTypeDescriptor{{Kind: ReflectClassKind, Raw: ObjectTypeID}}}}
	RegisterClassDescriptor(ClassDescriptor{Type: id, Fields: fields, TypeParameters: parameters, Initialize: func(*Execution) { initialized++ }, AnnotationValues: []AnnotationDescriptor{{Type: "test.UnusedAnnotation64", Factory: func(*Execution) any { factories++; return nil }}}})
	fields[0].Name = "changed"
	generic.Arguments[0].Raw = IntegerTypeID
	parameters[0].Name = "U"
	parameters[0].Bounds[0].Raw = StringTypeID
	class := ClassLiteral(id)
	first := class.GetDeclaredFields()
	field := ReferenceArrayGet[*Field](first, 0, ReflectFieldTypeID)
	if field.GetName() != "data" || ReflectTypeNameExecution(NewExecution(), field.GetGenericType()) != "java.util.List<java.lang.String>" {
		t.Fatal("metadata input mutation changed registered descriptor")
	}
	ReferenceArraySet(first, 0, (*Field)(nil))
	if ReferenceArrayGet[*Field](class.GetDeclaredFields(), 0, ReflectFieldTypeID) == nil {
		t.Fatal("declared field array storage leaked")
	}
	types := class.GetTypeParameters()
	ReferenceArraySet[any](types, 0, nil)
	variable := ReferenceArrayGet[any](class.GetTypeParameters(), 0, TypeVariableTypeID)
	if ReflectStringMemberExecution(NewExecution(), variable, TypeVariableTypeID, "GetName") != "T" {
		t.Fatal("variable descriptor mutation leaked")
	}
	bounds := ReflectArrayMemberExecution(NewExecution(), variable, TypeVariableTypeID, "GetBounds", ReflectTypeTypeID)
	ReferenceArraySet(bounds, 0, ClassLiteral(StringTypeID))
	if ReferenceArrayGet[any](ReflectArrayMemberExecution(NewExecution(), variable, TypeVariableTypeID, "GetBounds", ReflectTypeTypeID), 0, ReflectTypeTypeID) != ClassLiteral(ObjectTypeID) {
		t.Fatal("bounds array storage leaked")
	}
	if initialized != 0 || factories != 0 {
		t.Fatal("metadata registration or lookup invoked initialization/factory")
	}
}
