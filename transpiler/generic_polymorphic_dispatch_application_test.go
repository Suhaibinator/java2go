package transpiler

import (
	"os"
	"strings"
	"testing"
)

func TestGenericPolymorphicDispatchApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/basic/GenericDispatchApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "base:interface:indirect:222")
}

func TestConcreteCovariantDispatchApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/covariant/CovariantDispatchApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "13:20:cast:true")
}

func TestGenericPolymorphicDispatchTimingApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/timing/GenericDispatchTimingApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "cast:null:null:nominal:232334")
}

func TestOrdinaryCovariantDispatchApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/factory/FactoryDispatchApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "2:true:true:60")
}

func TestBoundedGenericDispatchApplication(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/bounded/BoundedDispatchApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "7:9:16")
}

func TestGenericPolymorphicDispatchAssignmentView(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/targetview/GenericTargetProbe.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "2")
}

func TestGenericPolymorphicDispatchShadowedBounds(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/generic_polymorphic_dispatch/src/parity/dispatch/shadowing/ShadowedBoundsApp.java")
	if err != nil {
		t.Fatal(err)
	}
	assertGeneratedLocalConstructorResult(t, strings.SplitN(string(source), "\n", 2)[1], "9:true:true")
}
