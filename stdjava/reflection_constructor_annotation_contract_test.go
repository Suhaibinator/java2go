package stdjava_test

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	j "github.com/NickyBoy89/java2go/stdjava"
)

type reflectionPlatformObject struct{ *j.ObjectInfo }
type reflectionPlatformAnnotation struct{ id j.TypeID }

func (a *reflectionPlatformAnnotation) JavaDynamicTypeID() j.TypeID { return a.id }

func reflectionPlatformActual(t *testing.T) map[string][]string {
	t.Helper()
	bytes, err := os.ReadFile("testdata/reflection_platform_actual_jdk21.json")
	if err != nil {
		t.Fatal(err)
	}
	var all map[string]map[string][]string
	if err := json.Unmarshal(bytes, &all); err != nil {
		t.Fatal(err)
	}
	actual := all["17"]
	if len(actual) != 31 {
		t.Fatalf("actual JDK row inventory = %d, want 31", len(actual))
	}
	return actual
}

func reflectionPlatformPanic(call func()) (failure any) {
	defer func() { failure = recover() }()
	call()
	return nil
}

func reflectionPlatformFailure(t *testing.T, failure any, class string) {
	t.Helper()
	if !j.CaughtAs(failure, class) {
		t.Fatalf("exception = %T, want %s", failure, class)
	}
}

func reflectionPlatformObservedMessage(t *testing.T, execution *j.Execution, failure any, encoded string) {
	t.Helper()
	message := j.JavaThrowableMessageExecution(execution, failure)
	if encoded == "null" {
		if message != nil {
			t.Fatal("JDK null detail message changed to non-null")
		}
		return
	}
	var expected []uint16
	if encoded != "" {
		for _, unit := range strings.Split(encoded, ",") {
			value, err := strconv.ParseUint(unit, 16, 16)
			if err != nil {
				t.Fatal(err)
			}
			expected = append(expected, uint16(value))
		}
	}
	if message == nil || !reflect.DeepEqual(message.UTF16Copy(), expected) {
		t.Fatal("detail message does not match actual JDK UTF16")
	}
}

func TestReflectionConstructorPlatformActualJDK66(t *testing.T) {
	actual := reflectionPlatformActual(t)
	t.Run("declared_lookup_without_initialization", func(t *testing.T) {
		const id j.TypeID = "reflection.platform.LookupSecret"
		j.RegisterJavaType(id, j.ObjectTypeID)
		executions := 0
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true,
			Initialize:   func(*j.Execution) { executions++ },
			Constructors: []j.ConstructorDescriptor{{Modifiers: 2, Construct: func(*j.Execution, []any) any { t.Fatal("lookup invoked constructor"); return nil }}},
		})
		class := j.ClassLiteral(id)
		ctor := class.GetDeclaredConstructor()
		if ctor.GetModifiers() != 2 || ctor.GetDeclaringClass() != class || ctor.CanAccess(nil) || executions != 0 {
			t.Fatal("private constructor metadata lookup initialized class or granted access")
		}
		failure := reflectionPlatformPanic(func() { class.GetConstructor() })
		reflectionPlatformFailure(t, failure, "NoSuchMethodException")
		if executions != 0 || actual["lookup-private"][3] != "" || actual["public-lookup-private"][4] != "" {
			t.Fatal("public-only lookup initialized a private constructor target")
		}
	})
	t.Run("access_override_isolated_and_null_varargs", func(t *testing.T) {
		const id j.TypeID = "reflection.platform.Private"
		execution := j.NewExecution()
		initialization := j.NewClassInitialization(string(id))
		trace := ""
		j.RegisterJavaType(id, j.ObjectTypeID)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true,
			Initialize: func(got *j.Execution) {
				if got != execution {
					t.Fatal("initialization lost Execution")
				}
				initialization.Ensure(got, func(*j.Execution) { trace += "S" })
			},
			Constructors: []j.ConstructorDescriptor{{Modifiers: 2, Construct: func(got *j.Execution, args []any) any {
				if got != execution || len(args) != 0 {
					t.Fatal("constructor lost execution/argument contract")
				}
				trace += "s"
				return &reflectionPlatformObject{j.NewObjectInfo(id, nil)}
			}}},
		})
		class := j.ClassLiteral(id)
		first, second := class.GetDeclaredConstructor(), class.GetDeclaredConstructor()
		reflectionPlatformFailure(t, reflectionPlatformPanic(func() { first.CanAccess(new(int)) }), "IllegalArgumentException")
		reflectionPlatformFailure(t, reflectionPlatformPanic(func() { first.NewInstance(execution, j.BoxInteger(17)) }), "IllegalAccessException")
		if trace != "" {
			t.Fatal("denied access initialized target")
		}
		first.SetAccessible(true)
		if !first.CanAccess(nil) || second.CanAccess(nil) || trace != "" {
			t.Fatal("override leaked across wrappers or initialized target")
		}
		var absentArguments []any
		one := first.NewInstance(execution, absentArguments...)
		two := first.NewInstance(execution, []any{}...)
		if j.JavaReferenceEqual(one, two) || trace != actual["private-instances"][1] {
			t.Fatal("constructor allocation/null varargs/initialization order changed")
		}
		first.SetAccessible(false)
		reflectionPlatformFailure(t, reflectionPlatformPanic(func() { first.NewInstance(execution) }), "IllegalAccessException")
		if first.CanAccess(nil) || trace != actual["private-instances"][1] {
			t.Fatal("revoked override permitted invocation")
		}
	})
	t.Run("initialize_before_wrong_argument_count", func(t *testing.T) {
		const id j.TypeID = "reflection.platform.PublicArguments"
		execution := j.NewExecution()
		initialization := j.NewClassInitialization(string(id))
		trace := ""
		j.RegisterJavaType(id, j.ObjectTypeID)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true,
			Initialize: func(got *j.Execution) {
				if got != execution {
					t.Fatal("lost Execution")
				}
				initialization.Ensure(got, func(*j.Execution) { trace += "O" })
			},
			Constructors: []j.ConstructorDescriptor{{Modifiers: 1, Construct: func(*j.Execution, []any) any {
				trace += "o"
				return &reflectionPlatformObject{j.NewObjectInfo(id, nil)}
			}}},
		})
		ctor := j.ClassLiteral(id).GetDeclaredConstructor()
		if trace != "" {
			t.Fatal("lookup initialized target")
		}
		failed := reflectionPlatformPanic(func() { ctor.NewInstance(execution, j.BoxInteger(17)) })
		reflectionPlatformFailure(t, failed, "IllegalArgumentException")
		reflectionPlatformObservedMessage(t, execution, failed, actual["public-wrongcount-first"][1])
		if trace != "O" {
			t.Fatal("argument rejection must follow target initialization and precede constructor body")
		}
		ctor.NewInstance(execution)
		if trace != "Oo" {
			t.Fatal("successful invocation repeated initialization")
		}
	})
	t.Run("target_wrapper_null_message_cause_identity_and_lock", func(t *testing.T) {
		for _, kind := range []string{"runtime", "error"} {
			id := j.TypeID("reflection.platform.Thrower." + kind)
			execution := j.NewExecution()
			units := []uint16{'c', 't', 'o', 'r', ' ', 0x03a9, 0xd800}
			var target any = j.NewJavaIllegalStateExceptionMessage(j.NewJavaStringUTF16(units))
			row := actual["target-wrapper"]
			if kind == "error" {
				target = j.NewJavaAssertionErrorExecution(execution, j.NewJavaStringUTF16([]uint16{'b', 'o', 'd', 'y', ' ', 'e', 'r', 'r', 'o', 'r'}))
				row = actual["target-error-wrapper"]
			}
			calls := 0
			j.RegisterJavaType(id, j.ObjectTypeID)
			j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true,
				Constructors: []j.ConstructorDescriptor{{Modifiers: 1, Construct: func(got *j.Execution, args []any) any {
					if got != execution || len(args) != 0 {
						t.Fatal("target callback execution/args changed")
					}
					calls++
					panic(target)
				}}},
			})
			ctor := j.ClassLiteral(id).GetDeclaredConstructor()
			first := reflectionPlatformPanic(func() { ctor.NewInstance(execution) })
			second := reflectionPlatformPanic(func() { ctor.NewInstance(execution) })
			reflectionPlatformFailure(t, first, "InvocationTargetException")
			reflectionPlatformFailure(t, first, "ReflectiveOperationException")
			reflectionPlatformFailure(t, second, "InvocationTargetException")
			reflectionPlatformObservedMessage(t, execution, first, row[1])
			if calls != 2 || j.JavaReferenceEqual(first, second) || !j.JavaReferenceEqual(j.GetCause(first), target) || !j.JavaReferenceEqual(j.GetCause(second), target) {
				t.Fatal("target callback count, fresh wrapper, or cause identity changed")
			}
			reflectionPlatformObservedMessage(t, execution, j.GetCause(first), row[3])
			rejection := reflectionPlatformPanic(func() { j.ThrowableInitCauseExecution(execution, first, nil) })
			reflectionPlatformFailure(t, rejection, "IllegalStateException")
			reflectionPlatformObservedMessage(t, execution, rejection, actual["wrapper-cause-locked"][1])
			if !j.JavaReferenceEqual(j.GetCause(first), target) || !j.JavaReferenceEqual(j.GetCause(rejection), first) {
				t.Fatal("locked cause rejected mutation without retaining wrapper/target identity")
			}
		}
	})
	t.Run("initializer_failures_escape_target_wrapper", func(t *testing.T) {
		const id j.TypeID = "reflection.platform.Broken"
		execution := j.NewExecution()
		initialization := j.NewClassInitialization(string(id))
		target := j.NewJavaIllegalStateExceptionMessage(j.NewJavaStringUTF16([]uint16{'i', 'n', 'i', 't', ' ', 0x03bb}))
		initialized, bodies := 0, 0
		j.RegisterJavaType(id, j.ObjectTypeID)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1, HasModifiers: true,
			Initialize: func(got *j.Execution) {
				initialization.Ensure(got, func(*j.Execution) { initialized++; panic(target) })
			},
			Constructors: []j.ConstructorDescriptor{{Modifiers: 1, Construct: func(*j.Execution, []any) any { bodies++; return nil }}},
		})
		ctor := j.ClassLiteral(id).GetDeclaredConstructor()
		if initialized != 0 {
			t.Fatal("failed target initialized during lookup")
		}
		first := reflectionPlatformPanic(func() { ctor.NewInstance(execution) })
		second := reflectionPlatformPanic(func() { ctor.NewInstance(execution) })
		reflectionPlatformFailure(t, first, "ExceptionInInitializerError")
		reflectionPlatformFailure(t, second, "NoClassDefFoundError")
		if j.CaughtAs(first, "InvocationTargetException") || j.CaughtAs(second, "InvocationTargetException") || initialized != 1 || bodies != 0 || !j.JavaReferenceEqual(j.GetCause(first), target) {
			t.Fatal("initialization failure was wrapped, retried, or invoked constructor")
		}
		reflectionPlatformFailure(t, j.GetCause(second), "ExceptionInInitializerError")
	})
	t.Run("abstract_rejection_precedes_initialization", func(t *testing.T) {
		const id j.TypeID = "reflection.platform.Abstract"
		initialized, bodies := 0, 0
		j.RegisterJavaType(id, j.ObjectTypeID)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: id, Modifiers: 1025, HasModifiers: true,
			Initialize:   func(*j.Execution) { initialized++ },
			Constructors: []j.ConstructorDescriptor{{Modifiers: 1, Construct: func(*j.Execution, []any) any { bodies++; return nil }}},
		})
		failed := reflectionPlatformPanic(func() { j.ClassLiteral(id).GetDeclaredConstructor().NewInstance(j.NewExecution()) })
		reflectionPlatformFailure(t, failed, "InstantiationException")
		if initialized != 0 || bodies != 0 {
			t.Fatal("abstract constructor rejection initialized or invoked target")
		}
	})
}

// Member-array defensive copies are exercised by the independently captured
// four-Java platform project and require compiled source annotation proxies.
// This runtime test checks only lookup/factory boundaries, never a handmade
// proxy's cloning implementation or annotation wrapper identity caching.
func TestReflectionAnnotationLookupPlatformActualJDK66(t *testing.T) {
	actual := reflectionPlatformActual(t)
	if actual["annotation-explicit-names"][0] != "true" || actual["annotation-default-names"][0] != "true" || actual["annotation-explicit-numbers"][0] != "true" || actual["annotation-default-numbers"][0] != "true" || actual["annotation-empty-identity"][1] != "true" {
		t.Fatal("actual compiled annotation array contract inventory changed")
	}
	const annotation j.TypeID = "reflection.platform.Payload"
	const implementation j.TypeID = "reflection.platform.PayloadProxy"
	j.RegisterJavaType(annotation, j.ObjectTypeID, j.AnnotationTypeID)
	j.RegisterJavaType(implementation, j.ObjectTypeID, annotation, j.AnnotationTypeID)
	j.RegisterClassDescriptor(j.ClassDescriptor{Type: annotation, InheritedAnnotation: true})
	t.Run("class_inherited_lookup_without_initialization", func(t *testing.T) {
		const base j.TypeID = "reflection.platform.Annotated"
		const child j.TypeID = "reflection.platform.Child"
		execution := j.NewExecution()
		factories, initialized := 0, 0
		value := &reflectionPlatformAnnotation{implementation}
		factory := func(got *j.Execution) any {
			if got != execution {
				t.Fatal("annotation factory lost Execution")
			}
			factories++
			return value
		}
		j.RegisterJavaType(base, j.ObjectTypeID)
		j.RegisterJavaType(child, base)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: base, Initialize: func(*j.Execution) { initialized++ }, AnnotationValues: []j.AnnotationDescriptor{{Type: annotation, Factory: factory}}})
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: child, Initialize: func(*j.Execution) { initialized++ }})
		if factories != 0 || initialized != 0 {
			t.Fatal("registration eagerly initialized annotation/target")
		}
		if got := j.ClassLiteral(base).GetAnnotationExecution(execution, j.ClassLiteral(annotation)); got != value {
			t.Fatal("declared nominal annotation lookup changed factory result")
		}
		if got := j.ClassLiteral(child).GetAnnotationExecution(execution, j.ClassLiteral(annotation)); got != value {
			t.Fatal("inherited annotation lookup missed declaring class")
		}
		if factories != 2 || initialized != 0 {
			t.Fatal("annotation lookup initialized target or lost lazy factory calls")
		}
	})
	t.Run("field_exact_lookup_and_missing_annotation", func(t *testing.T) {
		const owner j.TypeID = "reflection.platform.AnnotationField"
		execution := j.NewExecution()
		factories, initialized := 0, 0
		value := &reflectionPlatformAnnotation{implementation}
		factory := func(got *j.Execution) any {
			if got != execution {
				t.Fatal("field annotation factory lost Execution")
			}
			factories++
			return value
		}
		j.RegisterJavaType(owner, j.ObjectTypeID)
		j.RegisterClassDescriptor(j.ClassDescriptor{Type: owner, Initialize: func(*j.Execution) { initialized++ }, Fields: []j.FieldDescriptor{
			{Name: "text", Type: j.StringTypeID, Annotations: []j.AnnotationDescriptor{{Type: annotation, Factory: factory}}},
			{Name: "plain", Type: j.StringTypeID},
		}})
		class := j.ClassLiteral(owner)
		if class.GetDeclaredField("text").GetAnnotationExecution(execution, j.ClassLiteral(annotation)) != value {
			t.Fatal("field annotation type/factory result changed")
		}
		if class.GetDeclaredField("plain").GetAnnotationExecution(execution, j.ClassLiteral(annotation)) != nil {
			t.Fatal("missing annotation became a value")
		}
		if factories != 1 || initialized != 0 {
			t.Fatal("annotation lookup evaluated absent factory or initialized target")
		}
	})
}
