package transpiler

import (
	"path/filepath"
	"strings"
	"testing"
)

func cloneExcludedAnnotations() map[string]bool {
	copy := make(map[string]bool, len(excludedAnnotations))
	for key, value := range excludedAnnotations {
		copy[key] = value
	}
	return copy
}

func restoreExcludedAnnotations(snapshot map[string]bool) {
	excludedAnnotations = make(map[string]bool, len(snapshot))
	for key, value := range snapshot {
		excludedAnnotations[key] = value
	}
}

func TestFullProgram_AnnotationHandlingAndExclusion(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_annotations")
	snapshot := cloneExcludedAnnotations()
	t.Cleanup(func() { restoreExcludedAnnotations(snapshot) })

	excludedAnnotations = map[string]bool{}
	defaultOutputs := convertJavaProjectDir(t, root)
	defaultService := normalizeSpaces(defaultOutputs["com/acme/services/UserService.go"])

	if !strings.Contains(defaultService, "//@Skip") {
		t.Fatalf("expected annotation passthrough comments in default conversion:\n%s", defaultOutputs["com/acme/services/UserService.go"])
	}
	if !strings.Contains(defaultService, "func (ue *UserService) InternalName() string") {
		t.Fatalf("expected annotated method to be present when annotation is not excluded:\n%s", defaultOutputs["com/acme/services/UserService.go"])
	}

	excludedAnnotations = map[string]bool{"@Skip": true}
	excludedOutputs := convertJavaProjectDir(t, root)
	excludedService := normalizeSpaces(excludedOutputs["com/acme/services/UserService.go"])

	if strings.Contains(excludedService, "InternalName(") {
		t.Fatalf("expected annotated method to be excluded when @Skip is configured:\n%s", excludedOutputs["com/acme/services/UserService.go"])
	}
	if strings.Contains(excludedService, "token string") {
		t.Fatalf("expected annotated field to be excluded when @Skip is configured:\n%s", excludedOutputs["com/acme/services/UserService.go"])
	}
	if !strings.Contains(excludedService, "PublicName(") {
		t.Fatalf("expected unannotated method to remain after exclusion:\n%s", excludedOutputs["com/acme/services/UserService.go"])
	}
}

func TestFullProgram_CastSemanticsEdgeCases(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_casts")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/casts/Casts.go"])

	if !strings.Contains(flat, "return int32(value)") {
		t.Fatalf("expected primitive cast to use Go conversion call:\n%s", outputs["com/acme/casts/Casts.go"])
	}
	if !strings.Contains(flat, "return any(value).(string)") {
		t.Fatalf("expected reference cast to use type assertion over any(...):\n%s", outputs["com/acme/casts/Casts.go"])
	}
}

func TestFullProgram_WildcardsAndVarianceGenerics(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_generics")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/generics/VarianceProgram.go"])

	// java.util.List maps to the stdjava runtime type.
	if !strings.Contains(flat, "source *stdjava.List[*Number]") {
		t.Fatalf("expected '? extends Number' to map to bounded generic element type:\n%s", outputs["com/acme/generics/VarianceProgram.go"])
	}
	if !strings.Contains(flat, "sink *stdjava.List[any]") {
		t.Fatalf("expected '? super Integer' to be approximated as any:\n%s", outputs["com/acme/generics/VarianceProgram.go"])
	}
}

func TestFullProgram_MethodReferencesAndNestedConstructors(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_refs")
	outputs := convertJavaProjectDir(t, root)
	outer := normalizeSpaces(outputs["com/acme/refs/Outer.go"])

	if !strings.Contains(outer, "NewMapperFuncAdapter[string, string](Id)") {
		t.Fatalf("expected static method reference to map through SAM adapter:\n%s", outputs["com/acme/refs/Outer.go"])
	}
	// Inner (non-static) class: `this.new Inner(in)` lowers to the renamed
	// nested-class constructor and threads the enclosing instance as the leading
	// argument, e.g. NewOuterInner(or, in).
	if !strings.Contains(outer, "NewOuterInner(or, in)") {
		t.Fatalf("expected inner-class constructor call to thread the enclosing instance, got:\n%s", outputs["com/acme/refs/Outer.go"])
	}
}

func TestFullProgram_ControlFlowAndTryCatchPatterns(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_control")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/control/Flow.go"])

	if !strings.Contains(flat, "for i := 0; i < n; i++") {
		t.Fatalf("expected classic for-loop conversion:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "for j < n") {
		t.Fatalf("expected while-loop conversion:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "for {") {
		t.Fatalf("expected do-while conversion to loop with break guard:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "if !(n > 0) { break }") {
		t.Fatalf("expected do-while guard to break on negated condition:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if strings.Contains(flat, "ILLEGAL(") {
		t.Fatalf("do-while guard regressed to ILLEGAL() lowering:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "recover(") {
		t.Fatalf("expected try/catch lowering to use recover:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "100") {
		t.Fatalf("expected catch block statements to be preserved:\n%s", outputs["com/acme/control/Flow.go"])
	}
	if !strings.Contains(flat, "333") {
		t.Fatalf("expected finally block statements to be preserved:\n%s", outputs["com/acme/control/Flow.go"])
	}
}
