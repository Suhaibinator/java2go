package transpiler

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Exercise the existing runtime harness in child test processes so a rejected
// package clause can be asserted without changing the generated application.
func TestGeneratedBehaviorPackageBinding(t *testing.T) {
	cases := []struct {
		name      string
		rejection string
	}{
		{name: "main-template"},
		{name: "named-template"},
		{name: "matching-explicit"},
		{name: "external-template"},
		{name: "matching-external"},
		{name: "mismatching-explicit", rejection: `companion Go test package "other" does not match generated package "modern"`},
		{name: "mismatching-external", rejection: `companion Go test package "other_test" does not match generated package "modern"`},
		{name: "malformed-generated", rejection: "parse generated Go package"},
		{name: "malformed-companion", rejection: "parse companion Go test package"},
	}
	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGeneratedBehaviorPackageBindingChild$", "-test.v")
			cmd.Env = append(os.Environ(), "JAVA2GO_PACKAGE_BINDING_CASE="+scenario.name)
			cmd.WaitDelay = 5 * time.Second
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("harness child timeout: %v\n%s", ctx.Err(), output)
			}
			if scenario.rejection == "" {
				if err != nil {
					t.Fatalf("package binding behavior failed: %v\n%s", err, output)
				}
				if !strings.Contains(string(output), "generated TestAnswer assertion verified") {
					t.Fatalf("runtime assertion did not run:\n%s", output)
				}
			} else if err == nil || !strings.Contains(string(output), scenario.rejection) {
				t.Fatalf("expected package rejection %q, got %v\n%s", scenario.rejection, err, output)
			}
		})
	}
}

func TestGeneratedBehaviorPackageBindingChild(t *testing.T) {
	scenario := os.Getenv("JAVA2GO_PACKAGE_BINDING_CASE")
	if scenario == "" {
		return
	}
	generatedPackage, companionPackage, external := "modern", "main", false
	switch scenario {
	case "main-template":
		generatedPackage = "main"
	case "named-template":
	case "matching-explicit":
		companionPackage = "modern"
	case "external-template":
		companionPackage, external = "main_test", true
	case "matching-external":
		companionPackage, external = "modern_test", true
	case "mismatching-explicit":
		companionPackage = "other"
	case "mismatching-external":
		companionPackage, external = "other_test", true
	case "malformed-generated":
	case "malformed-companion":
	default:
		t.Fatalf("unknown package binding scenario %q", scenario)
	}
	generated := "// package main in a comment is not the package clause.\npackage " + generatedPackage + "\n\nfunc Answer() int { return 37 }\n"
	// These complete companion assertions are identical in red and candidate.
	companion := "// package modern in a comment is not the package clause.\npackage " + companionPackage + "\n\nimport \"testing\"\n\nfunc TestAnswer(t *testing.T) {\n got := Answer()\n if got != 37 { t.Fatalf(\"Answer() = %d, want 37\", got) }\n}\n"
	if external {
		companion = "// package modern in a comment is not the package clause.\npackage " + companionPackage + "\n\nimport (\"testing\"; fixture \"generated\")\n\nfunc TestAnswer(t *testing.T) {\n got := fixture.Answer()\n if got != 37 { t.Fatalf(\"Answer() = %d, want 37\", got) }\n}\n"
	}
	if scenario == "malformed-generated" {
		generated = "func Answer() int { return 37 }\n"
	}
	if scenario == "malformed-companion" {
		companion = "package\nfunc TestAnswer() {}\n"
	}
	runGoTestInTempModule(t, generated, companion)
	t.Log("generated TestAnswer assertion verified")
}
