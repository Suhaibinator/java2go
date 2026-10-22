package transpiler

import (
	"os"
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
	defaultAST := ciABI193Parse(t, defaultOutputs["com/acme/services/UserService.go"])
	ciABI193RequireMethod(t, defaultAST, "InternalName", "*UserService", "*stdjava.JavaString")
	if field := ciABI193Field(t, defaultAST, "UserService", "token"); field != nil {
		ciABI193RequireExpr(t, field, "*stdjava.JavaString")
	} else {
		t.Fatal("annotated field missing when annotation is not excluded")
	}

	excludedAnnotations = map[string]bool{"@Skip": true}
	excludedOutputs := convertJavaProjectDir(t, root)
	excludedAST := ciABI193Parse(t, excludedOutputs["com/acme/services/UserService.go"])
	ciABI193RequireMethodAbsent(t, excludedAST, "InternalName")
	ciABI193RequireFieldAbsent(t, excludedAST, "UserService", "token")
	ciABI193RequireMethod(t, excludedAST, "PublicName", "*UserService", "*stdjava.JavaString")
}

func TestFullProgram_CastSemanticsEdgeCases(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_casts")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/casts/Casts.go"])

	if !strings.Contains(flat, "return int32(value)") {
		t.Fatalf("expected primitive cast to use Go conversion call:\n%s", outputs["com/acme/casts/Casts.go"])
	}
	castAST := ciABI193Parse(t, outputs["com/acme/casts/Casts.go"])
	castMethod := ciABI193Function(t, castAST, "AsStringJava2goExecution")
	if castMethod.Type.Results == nil || len(castMethod.Type.Results.List) != 1 {
		t.Fatal("AsString lost its canonical reference result")
	}
	ciABI193RequireExpr(t, castMethod.Type.Results.List[0].Type, "*stdjava.JavaString")
	ciABI193RequireReturn(t, castMethod, `stdjava.ObjectView[*stdjava.JavaString](value, stdjava.StringTypeID)`)
	castSource, err := os.ReadFile(filepath.Join(root, "com/acme/casts/Casts.java"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>cast-semantics</artifactId><version>1</version></project>`,
		"src/main/java/com/acme/casts/Casts.java": string(castSource),
		"src/main/java/example/Main.java":         `package example;import com.acme.casts.Casts;public class Main{public static void main(String[] args){System.out.println(Casts.fromDouble(2.9));System.out.println(Casts.asString("ok"));System.out.println(Casts.asString(null)==null);try{Casts.asString(42);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("rejected");}}}`,
	}, "example.Main", "2\nok\ntrue\nrejected\n")
}

func TestFullProgram_WildcardsAndVarianceGenerics(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_generics")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/generics/VarianceProgram.go"])

	// Number is a runtime interface shared by the numeric wrapper objects.
	if !strings.Contains(flat, "source *stdjava.List[stdjava.JavaNumber]") {
		t.Fatalf("expected '? extends Number' to retain Number's readable interface:\n%s", outputs["com/acme/generics/VarianceProgram.go"])
	}
	if !strings.Contains(flat, "sink *stdjava.List[any]") {
		t.Fatalf("expected '? super Integer' to be approximated as any:\n%s", outputs["com/acme/generics/VarianceProgram.go"])
	}
}

func TestFullProgram_MethodReferencesAndNestedConstructors(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_refs")
	outputs := convertJavaProjectDir(t, root)
	outer := normalizeSpaces(outputs["com/acme/refs/Outer.go"])

	if !strings.Contains(outer, "NewMapperFuncAdapterJava2goExecution[*stdjava.JavaString, *stdjava.JavaString](IdJava2goExecution)") {
		t.Fatalf("expected static method reference to map through SAM adapter:\n%s", outputs["com/acme/refs/Outer.go"])
	}
	// Inner (non-static) class: `this.new Inner(in)` lowers to the renamed
	// nested-class constructor and threads the enclosing instance as the leading
	// argument, e.g. NewOuterInner(or, in).
	if !strings.Contains(outer, "NewOuterInnerJava2goExecution(__java2goExecution, func() *Outer") ||
		!strings.Contains(outer, "__java2goEnclosingInstance := or") ||
		!strings.Contains(outer, "if __java2goEnclosingInstance == nil") ||
		!strings.Contains(outer, "}(), in)") {
		t.Fatalf("expected inner-class constructor call to null-check and thread the enclosing instance, got:\n%s", outputs["com/acme/refs/Outer.go"])
	}
}

func TestFullProgram_ControlFlowAndTryCatchPatterns(t *testing.T) {
	root := filepath.Join("..", "testfiles", "full_program_control")
	outputs := convertJavaProjectDir(t, root)
	flat := normalizeSpaces(outputs["com/acme/control/Flow.go"])

	// int loop counters are pinned to int32 (K1), so the init is `i := int32(0)`.
	if !strings.Contains(flat, "for i := int32(0); i < n; i++") {
		t.Fatalf("expected classic for-loop conversion with int32-pinned counter:\n%s", outputs["com/acme/control/Flow.go"])
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
