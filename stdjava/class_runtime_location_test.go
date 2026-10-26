package stdjava

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Observe existing descriptor lookup without depending on a new API compiling.
// Absent provenance is observed as unknown, then compared with fresh JDK facts.
func observedClassRuntimeLocation(id TypeID) string {
	field := reflect.ValueOf(classDescriptor(id)).FieldByName("RuntimeLocation")
	if !field.IsValid() || !field.FieldByName("Known").Bool() {
		return "unknown"
	}
	return field.FieldByName("ModuleName").String() + "|" + field.FieldByName("LoaderName").String()
}

func TestClassRuntimeLocationJDK21(t *testing.T) {
	rows := []struct {
		id   TypeID
		want string
	}{
		{TypeID("java.lang.StringBuilder"), "java.base|bootstrap"},
		{TypeID("java.lang.AbstractStringBuilder"), "java.base|bootstrap"},
		{TypeID("java.lang.String"), "java.base|bootstrap"},
		{TypeID("java.lang.StringBuffer"), "java.base|bootstrap"},
		{TypeID("java.util.UUID"), "java.base|bootstrap"},
		{TypeID("java.lang.Boolean"), "java.base|bootstrap"},
		{TypeID("java.lang.Byte"), "java.base|bootstrap"},
		{TypeID("java.lang.Short"), "java.base|bootstrap"},
		{TypeID("java.lang.Integer"), "java.base|bootstrap"},
		{TypeID("java.lang.Long"), "java.base|bootstrap"},
		{TypeID("java.lang.Float"), "java.base|bootstrap"},
		{TypeID("java.lang.Double"), "java.base|bootstrap"},
		{TypeID("java.lang.Character"), "java.base|bootstrap"},
		{TypeID("java.lang.Number"), "java.base|bootstrap"},
		{TypeID("java.lang.CharSequence"), "java.base|bootstrap"},
		{TypeID("java.lang.Runnable"), "java.base|bootstrap"},
		{TypeID("java.lang.Object"), "java.base|bootstrap"},
		{TypeID("java.sql.Date"), "java.sql|platform"},
		{TypeID("java.sql.Timestamp"), "java.sql|platform"},
		{TypeID("java.sql.Time"), "java.sql|platform"},
		{TypeID("java.util.Date"), "java.base|bootstrap"},
		{ArrayTypeID("java.lang.StringBuilder"), "java.base|bootstrap"},
		{ArrayTypeID(PrimitiveIntTypeID), "java.base|bootstrap"},
		{TypeID("java.lang.ClassCastException"), "java.base|bootstrap"},
		{TypeID("java.util.ArrayList"), "java.base|bootstrap"},
		{TypeID("java.util.LinkedList"), "java.base|bootstrap"},
		{TypeID("primitive:boolean"), "java.base|bootstrap"},
		{TypeID("primitive:byte"), "java.base|bootstrap"},
		{TypeID("primitive:short"), "java.base|bootstrap"},
		{TypeID("primitive:char"), "java.base|bootstrap"},
		{TypeID("primitive:int"), "java.base|bootstrap"},
		{TypeID("primitive:long"), "java.base|bootstrap"},
		{TypeID("primitive:float"), "java.base|bootstrap"},
		{TypeID("primitive:double"), "java.base|bootstrap"},
		{TypeID("primitive:void"), "java.base|bootstrap"},
	}
	var mismatch strings.Builder
	for _, row := range rows {
		if got := observedClassRuntimeLocation(row.id); got != row.want {
			mismatch.WriteString(string(row.id) + ": got " + got + " want " + row.want + "\n")
		}
	}
	if mismatch.Len() != 0 {
		t.Fatal("explicit declaration location differs from fresh JDK21 facts:\n" + mismatch.String())
	}
}

var classLocationTestIdentity atomic.Uint64

func locationTestType(name string) TypeID {
	return TypeID(fmt.Sprintf("location.test.%s%d", name, classLocationTestIdentity.Add(1)))
}

func testApplicationRuntimeLocation() ClassRuntimeLocation {
	return ClassRuntimeLocation{Known: true, LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"}
}

func TestClassRuntimeLocationRegistrationOrderAndImmutability(t *testing.T) {
	location := testApplicationRuntimeLocation()
	for _, locationFirst := range []bool{false, true} {
		id := locationTestType("descriptorFirst")
		if locationFirst {
			id = locationTestType("locationFirst")
		}
		parent := TypeID("location.order.Parent")
		RegisterJavaType(parent, ObjectTypeID)
		RegisterJavaType(id, parent)
		initialized := 0
		descriptor := ClassDescriptor{Type: id, HasModifiers: true, Modifiers: 17, Fields: []FieldDescriptor{{Name: "kept"}}, Initialize: func(*Execution) { initialized++ }}
		if locationFirst {
			RegisterClassRuntimeLocation(id, location)
		}
		RegisterClassDescriptor(descriptor)
		RegisterClassRuntimeLocation(id, location)
		RegisterClassDescriptor(descriptor)
		class := ClassLiteral(id)
		got := classDescriptor(id)
		if got.RuntimeLocation != location || len(got.Fields) != 1 || got.Fields[0].Name != "kept" || got.Initialize == nil || class.GetModifiers() != 17 || class.GetSuperclass() != ClassLiteral(parent) {
			t.Fatalf("location publication replaced descriptor semantics: %+v", got)
		}
		if initialized != 0 {
			t.Fatal("location metadata initialized Java class")
		}
		got.RuntimeLocation.ModuleName = "mutated-copy"
		if ClassRuntimeLocationOf(id) != location || classDescriptor(id).RuntimeLocation != location {
			t.Fatal("returned descriptor/location mutation escaped")
		}
		RegisterClassRuntimeLocation(id, ClassRuntimeLocation{})
		if ClassRuntimeLocationOf(id) != location {
			t.Fatal("unknown fact erased published location")
		}
	}
	iface := TypeID("location.order.Interface")
	RegisterJavaType(iface, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: iface, Interface: true, HasModifiers: true, Modifiers: 1537, RuntimeLocation: location})
	if ClassLiteral(iface).GetSuperclass() != nil || ClassLiteral(iface).GetModifiers() != 1537 {
		t.Fatal("descriptor kind/flags changed by provenance")
	}
}

func TestClassRuntimeLocationUnknownArraysAndConflict(t *testing.T) {
	unknown := TypeID("java.notADeclaredJDKType.LocationProbe")
	RegisterJavaSourceType(unknown)
	if ClassRuntimeLocationOf(unknown).Known || ClassRuntimeLocationOf(ArrayTypeID(ArrayTypeID(unknown))).Known {
		t.Fatal("package spelling/source ownership invented runtime location")
	}
	id := locationTestType("arrayComponent")
	arrays := ArrayTypeID(ArrayTypeID(id))
	location := testApplicationRuntimeLocation()
	if ClassRuntimeLocationOf(arrays).Known {
		t.Fatal("unpublished component must stay unknown")
	}
	RegisterClassRuntimeLocation(id, location)
	if ClassRuntimeLocationOf(arrays) != location || classDescriptor(arrays).RuntimeLocation != location {
		t.Fatal("array did not resolve ultimate component location")
	}
	base := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
	if ClassRuntimeLocationOf(ArrayTypeID(ArrayTypeID(PrimitiveIntTypeID))) != base {
		t.Fatal("primitive array location is not explicit primitive fact")
	}
	mustReject := func(name string, action func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("conflicting/incomplete location was accepted")
				}
			}()
			action()
		})
	}
	mustReject("conflicting", func() { RegisterClassRuntimeLocation(id, base) })
	mustReject("array_override", func() { RegisterClassRuntimeLocation(arrays, base) })
	mustReject("unknown_with_facts", func() { RegisterClassRuntimeLocation("location.bad.unknown", ClassRuntimeLocation{LoaderName: "app"}) })
	mustReject("missing_loader", func() { RegisterClassRuntimeLocation("location.bad.loader", ClassRuntimeLocation{Known: true}) })
	mustReject("loader_name_conflict", func() {
		bad := location
		bad.LoaderName = "bootstrap"
		RegisterClassRuntimeLocation("location.bad.name", bad)
	})
	mustReject("unsupported_loader", func() {
		bad := location
		bad.LoaderIdentity = "custom"
		RegisterClassRuntimeLocation("location.bad.custom", bad)
	})
	mustReject("named_app_module", func() {
		bad := location
		bad.ModuleName = "made.up"
		RegisterClassRuntimeLocation("location.bad.module", bad)
	})
	if ClassRuntimeLocationOf(id) != location {
		t.Fatal("conflict changed immutable fact")
	}
}

func TestClassRuntimeLocationConcurrentReadAndRegistration(t *testing.T) {
	id := TypeID("location.concurrent.Component")
	location := testApplicationRuntimeLocation()
	RegisterClassRuntimeLocation(id, location)
	var workers sync.WaitGroup
	for i := 0; i < 16; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 200; j++ {
				RegisterClassRuntimeLocation(id, location)
				if ClassRuntimeLocationOf(ArrayTypeID(ArrayTypeID(id))) != location || classDescriptor(id).RuntimeLocation != location {
					t.Error("concurrent metadata fact changed")
					return
				}
			}
		}()
	}
	workers.Wait()
}
