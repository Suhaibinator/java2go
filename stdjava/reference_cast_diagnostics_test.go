package stdjava

import (
	"fmt"
	"testing"
)

func TestReferenceCastDiagnosticsObservedJDK21(t *testing.T) {
	app := ClassRuntimeLocation{Known: true, LoaderIdentity: AppClassLoaderIdentity, LoaderName: "app"}
	for _, id := range []TypeID{"castprobe.Main$Child", "castprobe.Main$Sibling", "castprobe.Main$Marker", "castprobe.Main$StringBuffer"} {
		RegisterClassRuntimeLocation(id, app)
	}
	for _, row := range []struct{ name, actual, requested, want string }{
		{"native-forward", "java.lang.StringBuilder", "java.lang.StringBuffer", "class java.lang.StringBuilder cannot be cast to class java.lang.StringBuffer (java.lang.StringBuilder and java.lang.StringBuffer are in module java.base of loader 'bootstrap')"},
		{"native-reverse", "java.lang.StringBuffer", "java.lang.StringBuilder", "class java.lang.StringBuffer cannot be cast to class java.lang.StringBuilder (java.lang.StringBuffer and java.lang.StringBuilder are in module java.base of loader 'bootstrap')"},
		{"uuid-builder", "java.util.UUID", "java.lang.StringBuilder", "class java.util.UUID cannot be cast to class java.lang.StringBuilder (java.util.UUID and java.lang.StringBuilder are in module java.base of loader 'bootstrap')"},
		{"integer-character", "java.lang.Integer", "java.lang.Character", "class java.lang.Integer cannot be cast to class java.lang.Character (java.lang.Integer and java.lang.Character are in module java.base of loader 'bootstrap')"},
		{"sql-date-timestamp", "java.sql.Date", "java.sql.Timestamp", "class java.sql.Date cannot be cast to class java.sql.Timestamp (java.sql.Date and java.sql.Timestamp are in module java.sql of loader 'platform')"},
		{"source-sibling", "castprobe.Main$Child", "castprobe.Main$Sibling", "class castprobe.Main$Child cannot be cast to class castprobe.Main$Sibling (castprobe.Main$Child and castprobe.Main$Sibling are in unnamed module of loader 'app')"},
		{"native-marker", "java.lang.StringBuilder", "castprobe.Main$Marker", "class java.lang.StringBuilder cannot be cast to class castprobe.Main$Marker (java.lang.StringBuilder is in module java.base of loader 'bootstrap'; castprobe.Main$Marker is in unnamed module of loader 'app')"},
		{"native-source-name", "java.lang.StringBuffer", "castprobe.Main$StringBuffer", "class java.lang.StringBuffer cannot be cast to class castprobe.Main$StringBuffer (java.lang.StringBuffer is in module java.base of loader 'bootstrap'; castprobe.Main$StringBuffer is in unnamed module of loader 'app')"},
		{"source-native-name", "castprobe.Main$StringBuffer", "java.lang.StringBuffer", "class castprobe.Main$StringBuffer cannot be cast to class java.lang.StringBuffer (castprobe.Main$StringBuffer is in unnamed module of loader 'app'; java.lang.StringBuffer is in module java.base of loader 'bootstrap')"},
		{"captured-number", "java.lang.StringBuilder", "java.lang.Number", "class java.lang.StringBuilder cannot be cast to class java.lang.Number (java.lang.StringBuilder and java.lang.Number are in module java.base of loader 'bootstrap')"},
		{"array-family", "array:java.lang.StringBuilder", "array:java.lang.StringBuffer", "class [Ljava.lang.StringBuilder; cannot be cast to class [Ljava.lang.StringBuffer; ([Ljava.lang.StringBuilder; and [Ljava.lang.StringBuffer; are in module java.base of loader 'bootstrap')"},
		{"native-runnable", "java.lang.StringBuilder", "java.lang.Runnable", "class java.lang.StringBuilder cannot be cast to class java.lang.Runnable (java.lang.StringBuilder and java.lang.Runnable are in module java.base of loader 'bootstrap')"},
		{"list-class", "java.util.ArrayList", "java.util.LinkedList", "class java.util.ArrayList cannot be cast to class java.util.LinkedList (java.util.ArrayList and java.util.LinkedList are in module java.base of loader 'bootstrap')"},
		{"object-builder", "java.lang.Object", "java.lang.StringBuilder", "class java.lang.Object cannot be cast to class java.lang.StringBuilder (java.lang.Object and java.lang.StringBuilder are in module java.base of loader 'bootstrap')"},
	} {
		t.Run(row.name, func(t *testing.T) {
			actual, requested := TypeID(row.actual), TypeID(row.requested)
			if got := referenceCastMessage(actual, requested, ClassRuntimeLocationOf(actual), ClassRuntimeLocationOf(requested)); got != row.want {
				t.Fatalf("message=%q want=%q", got, row.want)
			}
			failure := referenceCastException(actual, requested)
			if _, isValue := any(failure).(ClassCastException); !isValue {
				t.Fatalf("constructor must retain exception value ABI: %T", failure)
			}
			if !CaughtAs(failure, "ClassCastException") || GetMessage(failure) != row.want {
				t.Fatalf("constructor type/message=%T %q", failure, GetMessage(failure))
			}
		})
	}
}

func TestReferenceCastDiagnosticsUnknownKeepsLegacy(t *testing.T) {
	known := ClassRuntimeLocationOf(TypeID("java.lang.StringBuilder"))
	for _, row := range []struct {
		name              string
		actual, requested TypeID
		a, b              ClassRuntimeLocation
	}{
		{"actual location", "java.lang.StringBuilder", "java.lang.StringBuffer", ClassRuntimeLocation{}, known},
		{"requested location", "java.lang.StringBuilder", "java.lang.StringBuffer", known, ClassRuntimeLocation{}},
		{"both locations", "java.lang.StringBuilder", "java.lang.StringBuffer", ClassRuntimeLocation{}, ClassRuntimeLocation{}},
		{"actual identity", "", "java.lang.StringBuffer", known, known},
		{"requested identity", "java.lang.StringBuilder", "", known, known},
	} {
		t.Run(row.name, func(t *testing.T) {
			want := fmt.Sprintf("Java value %s is not assignable to %s", row.actual, row.requested)
			if got := referenceCastMessage(row.actual, row.requested, row.a, row.b); got != want {
				t.Fatalf("unknown metadata became a proved domain: %q", got)
			}
		})
	}
	for _, actual := range []TypeID{"java.fake.Unknown", "source.Unknown", "array:java.fake.Unknown"} {
		if ClassRuntimeLocationOf(actual).Known {
			t.Fatal("unregistered identity acquired a runtime location")
		}
		want := fmt.Sprintf("Java value %s is not assignable to java.lang.StringBuffer", actual)
		if got := GetMessage(referenceCastException(actual, TypeID("java.lang.StringBuffer"))); got != want {
			t.Fatalf("unproved declaration changed diagnostic: %q", got)
		}
	}
}

func TestReferenceCastDiagnosticsTypedLoaderDomain(t *testing.T) {
	// Domain equality includes the identity field; display labels alone are not
	// an equality proof. This pure formatter control does not publish metadata.
	a := ClassRuntimeLocation{Known: true, ModuleName: "java.base", LoaderIdentity: BootstrapClassLoaderIdentity, LoaderName: "bootstrap"}
	b := a
	b.LoaderIdentity = PlatformClassLoaderIdentity
	want := "class p.A cannot be cast to class p.B (p.A is in module java.base of loader 'bootstrap'; p.B is in module java.base of loader 'bootstrap')"
	if got := referenceCastMessage("p.A", "p.B", a, b); got != want {
		t.Fatalf("typed loader identity ignored: %q", got)
	}
}

func TestReferenceCastDiagnosticsSupplementJDK21(t *testing.T) {
	for _, row := range []struct{ actual, requested, want string }{
		{"java.sql.Date", "java.lang.StringBuilder", "class java.sql.Date cannot be cast to class java.lang.StringBuilder (java.sql.Date is in module java.sql of loader 'platform'; java.lang.StringBuilder is in module java.base of loader 'bootstrap')"},
		{"java.lang.StringBuilder", "java.sql.Date", "class java.lang.StringBuilder cannot be cast to class java.sql.Date (java.lang.StringBuilder is in module java.base of loader 'bootstrap'; java.sql.Date is in module java.sql of loader 'platform')"},
		{"array:array:java.lang.StringBuilder", "array:array:java.lang.StringBuffer", "class [[Ljava.lang.StringBuilder; cannot be cast to class [[Ljava.lang.StringBuffer; ([[Ljava.lang.StringBuilder; and [[Ljava.lang.StringBuffer; are in module java.base of loader 'bootstrap')"},
		{"array:primitive:int", "array:primitive:long", "class [I cannot be cast to class [J ([I and [J are in module java.base of loader 'bootstrap')"},
		{"array:array:primitive:int", "array:primitive:int", "class [[I cannot be cast to class [I ([[I and [I are in module java.base of loader 'bootstrap')"},
	} {
		t.Run(row.actual+"_to_"+row.requested, func(t *testing.T) {
			if got := GetMessage(referenceCastException(TypeID(row.actual), TypeID(row.requested))); got != row.want {
				t.Fatalf("message=%q want fresh JDK=%q", got, row.want)
			}
		})
	}
}
