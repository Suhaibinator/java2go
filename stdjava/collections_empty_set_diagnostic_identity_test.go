package stdjava

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestCollectionsEmptySetNativeDiagnosticIdentityJDK21(t *testing.T) {
	const childFlag = "JAVA2GO_EMPTY_SET_IDENTITY_CHILD"
	if os.Getenv(childFlag) == "1" {
		execution := NewExecution()
		set := EmptySet()
		roles := emptySetTDDRolesOf(t, set)
		for _, probe := range []struct {
			name string
			call func()
		}{
			{"contains", func() { CollectionContainsAllExecution(execution, set, nil) }},
			{"array", func() { roles.CollectionToArrayIntoJava2goExecution(execution, nil) }},
		} {
			first := emptySetTDDPanic(t, "NullPointerException", probe.call)
			second := emptySetTDDPanic(t, "NullPointerException", probe.call)
			text := JavaThrowableMessageExecution(execution, first)
			other := JavaThrowableMessageExecution(execution, second)
			if text == nil || other == nil {
				t.Fatal("enhanced detail message missing")
			}
			cached := text == JavaThrowableMessageExecution(execution, first)
			distinct := text != other
			equal := text.Equals(other)
			firstInternDiffers := text != InternJavaString(text)
			canonical := InternJavaString(text) == InternJavaString(other)
			fmt.Printf("%s=%t:%t:%t:%t:%t\n", probe.name, cached, distinct, equal, firstInternDiffers, canonical)
		}
		return
	}
	// The JDK oracle starts with a fresh process intern pool. Run this one
	// same-race-binary control in a bounded child to preserve that initial state
	// across the outer package suite and repeated stress seeds. No Go build/JVM.
	oracle, err := os.ReadFile("testdata/collections_empty_set/diagnostic_identity.expected.stdout")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(context, executable, "-test.run=^TestCollectionsEmptySetNativeDiagnosticIdentityJDK21$", "-test.timeout=5s")
	command.Env = append(os.Environ(), childFlag+"=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("bounded native identity control: %v\n%s", err, output)
	}
	expected := string(oracle) + "PASS\n"
	if string(output) != expected {
		t.Fatalf("fresh-process JDK identity parity\nJVM=%q\nGo=%q", expected, output)
	}
}
