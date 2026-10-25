package stdjava

import (
	"reflect"
	"testing"
	"unicode/utf16"
)

type accessibleModuleTarget struct{ Hidden int32 }

func (*accessibleModuleTarget) JavaDynamicTypeID() TypeID { return "access.fixture.Target" }
func accessibleModuleFailure(t *testing.T, execution *Execution, name, message string, call func()) Throwable {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); call() }()
	throwable, ok := failure.(Throwable)
	if !ok || throwable.ThrowableTypeName() != name {
		t.Fatalf("failure=%T %v; want %s", failure, failure, name)
	}
	actual := JavaThrowableMessageExecution(execution, throwable)
	if message == "<null>" && actual != nil || message != "<null>" && (actual == nil || !reflect.DeepEqual(actual.UTF16Copy(), utf16.Encode([]rune(message)))) {
		t.Fatalf("wrong %s message: %v", name, actual)
	}
	return throwable
}

// Direct validation failures have no target exception cause. Keep that
// contract explicit; reflective InvocationTargetException uses the helper above.
func accessibleModuleFailureNoCause(t *testing.T, execution *Execution, name, message string, call func()) {
	t.Helper()
	failure := accessibleModuleFailure(t, execution, name, message, call)
	if cause := GetCause(failure); cause != nil {
		t.Fatalf("unexpected direct failure cause: %T %v", cause, cause)
	}
}

// The named module is only an explicitly registered runtime policy fixture.
// The source frontend still rejects module-info.java and cannot claim full
// named-module translation from this test.
func TestAccessibleObjectNamedModulePolicyJDK21(t *testing.T) {
	execution := NewExecution()
	const target TypeID = "access.fixture.Target"
	const caller TypeID = "access.observe.AccessProbe"
	RegisterReflectionModule(ReflectionModuleDescriptor{Name: "access.witness", LoaderIdentity: AppClassLoaderIdentity, Packages: []string{"access.fixture", "access.observe"}, Exports: map[string][]string{"access.fixture": {}}})
	location := ClassRuntimeLocation{Known: true, ModuleName: "access.witness", LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"}
	RegisterJavaType(target, ObjectTypeID)
	RegisterJavaType(caller, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: target, RuntimeLocation: location, Modifiers: 1, HasModifiers: true, Fields: []FieldDescriptor{
		{Name: "privateValue", GoName: "Hidden", Type: PrimitiveIntTypeID, Modifiers: 2, HasModifiers: true, NonPublic: true},
		{Name: "publicStatic", Type: PrimitiveIntTypeID, Modifiers: 9, HasModifiers: true, StaticGet: func(*Execution) any { return int32(23) }},
	}, Constructors: []ConstructorDescriptor{{Modifiers: 2, Construct: func(actual *Execution, _ []any) any {
		if actual != execution {
			t.Fatal("lost execution")
		}
		return &accessibleModuleTarget{Hidden: 17}
	}}}})
	RegisterClassDescriptor(ClassDescriptor{Type: caller, RuntimeLocation: location, Modifiers: 1, HasModifiers: true})
	member := ClassLiteral(target).GetDeclaredField("privateValue")
	copy := ClassLiteral(target).GetDeclaredField("privateValue")
	value := &accessibleModuleTarget{Hidden: 17}
	if AccessibleObjectCanAccessExecution(execution, caller, member, value) {
		t.Fatal("private member visible across package")
	}
	checker := ClassLiteral(reflectionAccessibleObjectTypeID).GetDeclaredMethod("canAccess", ClassLiteral(ObjectTypeID))
	if UnboxBoolean(checker.InvokeWithCallerExecution(execution, caller, member, value).(*Boolean)) {
		t.Fatal("reflective checker selected fallback")
	}
	AccessibleObjectSetAccessibleExecution(execution, caller, member, true)
	if !AccessibleObjectCanAccessExecution(execution, caller, member, value) || AccessibleObjectCanAccessExecution(execution, caller, copy, value) {
		t.Fatal("override state shared across wrappers")
	}
	if got := UnboxInteger(member.GetWithCallerExecution(execution, caller, value).(*Integer)); got != 17 {
		t.Fatal(got)
	}
	accessibleModuleFailureNoCause(t, execution, "IllegalArgumentException", "null object for private int access.fixture.Target.privateValue", func() { AccessibleObjectCanAccessExecution(execution, caller, member, nil) })
	wrapped := accessibleModuleFailure(t, execution, "InvocationTargetException", "<null>", func() { checker.InvokeWithCallerExecution(execution, caller, member, nil) })
	cause, ok := GetCause(wrapped).(Throwable)
	if !ok || cause.ThrowableTypeName() != "IllegalArgumentException" || cause.Message() != "null object for private int access.fixture.Target.privateValue" {
		t.Fatalf("wrong reflective cause: %v", GetCause(wrapped))
	}
	AccessibleObjectSetAccessibleExecution(execution, caller, member, false)
	nullEntry := NewReferenceArray(2, reflectionAccessibleObjectTypeID)
	ReferenceArraySet(nullEntry, 0, member)
	accessibleModuleFailureNoCause(t, execution, "NullPointerException", `Cannot invoke "java.lang.reflect.AccessibleObject.checkCanSetAccessible(java.lang.Class)" because "ao" is null`, func() { AccessibleObjectSetAccessibleArrayExecution(execution, caller, nullEntry, true) })
	if AccessibleObjectIsAccessible(member) {
		t.Fatal("failed true bulk partially changed override")
	}
	closed := ClassLiteral(StringTypeID).GetDeclaredField("value")
	denial := `Unable to make field private final byte[] java.lang.String.value accessible: module java.base does not "opens java.lang" to module access.witness`
	accessibleModuleFailureNoCause(t, execution, "InaccessibleObjectException", denial, func() { AccessibleObjectSetAccessibleExecution(execution, caller, closed, true) })
	if AccessibleObjectTrySetAccessibleExecution(execution, caller, closed) || AccessibleObjectIsAccessible(closed) {
		t.Fatal("closed package granted override")
	}
	mixed := NewReferenceArray(2, reflectionAccessibleObjectTypeID)
	ReferenceArraySet(mixed, 0, member)
	ReferenceArraySet(mixed, 1, closed)
	accessibleModuleFailureNoCause(t, execution, "InaccessibleObjectException", denial, func() { AccessibleObjectSetAccessibleArrayExecution(execution, caller, mixed, true) })
	if AccessibleObjectIsAccessible(member) {
		t.Fatal("mixed refusal partially changed override")
	}
	static := ClassLiteral(target).GetDeclaredField("publicStatic")
	if !AccessibleObjectCanAccessExecution(execution, caller, static, nil) {
		t.Fatal("exported static public field refused")
	}
	accessibleModuleFailureNoCause(t, execution, "IllegalArgumentException", "non-null object for public static int access.fixture.Target.publicStatic", func() { AccessibleObjectCanAccessExecution(execution, caller, static, value) })
	constructor := ClassLiteral(target).GetDeclaredConstructor()
	if AccessibleObjectCanAccessExecution(execution, caller, constructor, nil) {
		t.Fatal("private constructor admitted")
	}
	AccessibleObjectSetAccessibleExecution(execution, caller, constructor, true)
	accessibleModuleFailureNoCause(t, execution, "IllegalArgumentException", "non-null object for private access.fixture.Target()", func() { AccessibleObjectCanAccessExecution(execution, caller, constructor, value) })
	if constructor.NewInstanceWithCallerExecution(execution, caller).(*accessibleModuleTarget).Hidden != 17 {
		t.Fatal("constructor callback result")
	}
	AccessibleObjectSetAccessibleExecution(execution, caller, member, true)
	accessibleModuleFailureNoCause(t, execution, "NullPointerException", `Cannot invoke "java.lang.reflect.AccessibleObject.setAccessible0(boolean)" because "ao" is null`, func() { AccessibleObjectSetAccessibleArrayExecution(execution, caller, nullEntry, false) })
	if AccessibleObjectIsAccessible(member) {
		t.Fatal("false bulk did not retain its earlier reset before null")
	}
	accessibleModuleFailureNoCause(t, execution, "NullPointerException", `Cannot read the array length because "<local2>" is null`, func() { AccessibleObjectSetAccessibleArrayExecution(execution, caller, nil, false) })
	if FieldStringExecution(execution, member) == nil {
		t.Fatal("field description missing")
	}
}

type annotationShapeOnly struct{ calls int }

func (*annotationShapeOnly) JavaDynamicTypeID() TypeID { return "reflection.access.ShapeOnly" }
func (*annotationShapeOnly) AnnotationType() *Class {
	return ClassLiteral("reflection.access.ShapeOnly")
}
func (value *annotationShapeOnly) AnnotationTypeJava2goExecution(*Execution) *Class {
	value.calls++
	return value.AnnotationType()
}

func TestAnnotationShapeDoesNotGrantNominalIdentity(t *testing.T) {
	RegisterJavaType("reflection.access.ShapeOnly", ObjectTypeID)
	value := &annotationShapeOnly{}
	accessibleModuleFailureNoCause(t, NewExecution(), "ClassCastException", "value is not a Java annotation", func() { AnnotationTypeExecution(NewExecution(), value) })
	if value.calls != 0 {
		t.Fatal("unowned nominal annotation entered shape callback")
	}
}

func TestReflectionModuleQualifiedEdgesAndProvenance(t *testing.T) {
	caller := ClassRuntimeLocation{Known: true, ModuleName: "reflection.access.friend", LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"}
	other := caller
	other.ModuleName = "reflection.access.other"
	module := ReflectionModuleDescriptor{Name: "reflection.access.owner", LoaderIdentity: AppClassLoaderIdentity, Packages: []string{"reflection.access.closed"}, Opens: map[string][]string{"reflection.access.closed": {"reflection.access.friend"}}}
	if !reflectionModuleEdge(module, "reflection.access.closed", caller, true) || !reflectionModuleEdge(module, "reflection.access.closed", caller, false) {
		t.Fatal("qualified opens did not export to same target")
	}
	if reflectionModuleEdge(module, "reflection.access.closed", other, true) || reflectionModuleEdge(module, "reflection.access.closed", other, false) {
		t.Fatal("qualified opens granted unrelated target")
	}
	module.Opens = nil
	module.Exports = map[string][]string{"reflection.access.closed": {"reflection.access.friend"}}
	if !reflectionModuleEdge(module, "reflection.access.closed", caller, false) || reflectionModuleEdge(module, "reflection.access.closed", caller, true) {
		t.Fatal("exports incorrectly granted deep reflection")
	}
	check := func(label string, call func()) {
		t.Helper()
		var failed any
		func() { defer func() { failed = recover() }(); call() }()
		if failed == nil {
			t.Fatal(label + " provenance accepted")
		}
	}
	check("unowned module package", func() {
		RegisterReflectionModule(ReflectionModuleDescriptor{Name: "reflection.access.invalid", LoaderIdentity: AppClassLoaderIdentity, Packages: []string{"reflection.access.owned"}, Exports: map[string][]string{"reflection.access.donated": {}}})
	})
	check("unregistered named app module", func() {
		RegisterClassRuntimeLocation("reflection.access.unregistered.Value", ClassRuntimeLocation{Known: true, ModuleName: "reflection.access.unregistered", LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"})
	})
}

func TestReflectionPartialInventoriesRefuseUnknown(t *testing.T) {
	const id TypeID = "reflection.access.PartialInventory"
	RegisterJavaType(id, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: id, PartialMemberMetadata: true, Modifiers: 1, HasModifiers: true, Fields: []FieldDescriptor{{Name: "known", Type: StringTypeID}}, Methods: []MethodDescriptor{{Name: "known", Return: StringTypeID, Modifiers: 1, HasModifiers: true}}, Constructors: []ConstructorDescriptor{{Parameters: []TypeID{StringTypeID}, Modifiers: 1}}})
	class := ClassLiteral(id)
	execution := NewExecution()
	if class.GetDeclaredField("known").GetName() != "known" || class.GetDeclaredMethod("known").GetName() != "known" {
		t.Fatal("supported inventory unavailable")
	}
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "complete platform member metadata is unavailable", func() { class.GetDeclaredFields() })
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "complete platform member metadata is unavailable", func() { class.GetDeclaredMethods() })
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "platform field metadata is outside the supported reflection surface", func() { class.GetDeclaredField("missing") })
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "platform method metadata is outside the supported reflection surface", func() { class.GetDeclaredMethod("missing") })
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "complete constructor metadata is unavailable", func() { class.GetDeclaredConstructor() })
	constructor := class.GetDeclaredConstructor(ClassLiteral(StringTypeID))
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "constructor callback metadata is unavailable", func() { constructor.NewInstance(execution, JavaStringLiteralUTF16([]uint16{'x'})) })
}

func TestAccessibleObjectMemberModifierDescriptionsJDK21(t *testing.T) {
	varargs := ClassLiteral("MemberDescription$Varargs")
	child := ClassLiteral("MemberDescription$Child")
	iface := ClassLiteral("MemberDescription$Default")
	RegisterClassDescriptor(ClassDescriptor{Type: iface.TypeID(), Interface: true})
	for _, row := range []struct {
		name, expected string
		member         JavaAccessibleObject
		receiver       any
	}{
		{"method", "null object for void MemberDescription$Varargs.write(java.lang.String[])", &Method{owner: varargs, descriptor: MethodDescriptor{Name: "write", Return: PrimitiveTypeID("void"), ParameterTypes: []TypeID{ArrayTypeID(StringTypeID)}, Modifiers: 128, HasModifiers: true}}, nil},
		{"constructor", "non-null object for MemberDescription$Varargs(java.lang.String[])", &Constructor{owner: varargs, descriptor: ConstructorDescriptor{Parameters: []TypeID{ArrayTypeID(StringTypeID)}, Modifiers: 128}}, new(struct{})},
		{"bridge", "null object for java.lang.Object MemberDescription$Child.read(java.lang.Object)", &Method{owner: child, descriptor: MethodDescriptor{Name: "read", Return: ObjectTypeID, ParameterTypes: []TypeID{ObjectTypeID}, Modifiers: 64 | 4096, HasModifiers: true}}, nil},
		{"default", "null object for public default int MemberDescription$Default.read()", &Method{owner: iface, descriptor: MethodDescriptor{Name: "read", Return: PrimitiveIntTypeID, Modifiers: 1, HasModifiers: true}}, nil},
	} {
		t.Run(row.name, func(t *testing.T) {
			accessibleModuleFailureNoCause(t, NewExecution(), "IllegalArgumentException", row.expected, func() { AccessibleObjectCanAccessExecution(NewExecution(), "", row.member, row.receiver) })
		})
	}
}
