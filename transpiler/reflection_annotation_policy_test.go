package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"testing"
)

func TestReflectionAnnotationPolicyDeclarationOwnership(t *testing.T) {
	const source = `import java.lang.annotation.Retention;import java.lang.annotation.RetentionPolicy;@Retention(RetentionPolicy.RUNTIME) @interface Mark {} class Host {}`
	h := setupParseHelper(t, source)
	file := h.Ctx.currentFile
	scope := file.FindClassScope("Mark")
	if scope == nil {
		t.Fatal("annotation declaration missing")
	}
	if policy, known := reflectionSourceAnnotationPolicy(scope); !known || policy.retention != "RUNTIME" {
		t.Fatal("actual owned runtime policy lost", policy, known)
	}
	copied := *scope
	if _, known := reflectionSourceAnnotationPolicy(&copied); known {
		t.Fatal("copied scope acquired declaration policy")
	}
	foreign := parsing.SourceFile{Name: "Foreign.java", Source: []byte(source)}
	if err := foreign.ParseAST(); err != nil {
		t.Fatal(err)
	}
	node := scope.Class.DeclarationNode
	beforeSource := file.Source
	t.Cleanup(func() { scope.Class.DeclarationNode = node; file.Source = beforeSource })
	scope.Class.DeclarationNode = findNode(foreign.Ast, "annotation_type_declaration")
	if _, known := reflectionSourceAnnotationPolicy(scope); known {
		t.Fatal("foreign same-span AST acquired policy")
	}
	scope.Class.DeclarationNode = node
	original := file.Source
	file.Source = append(append([]byte{}, original...), byte(' '))
	if _, known := reflectionSourceAnnotationPolicy(scope); known {
		t.Fatal("stale source snapshot acquired policy")
	}
	file.Source = original
	if policy, known := reflectionSourceAnnotationPolicy(scope); !known || policy.retention != "RUNTIME" {
		t.Fatal("restored actual source lost policy")
	}
}

func TestReflectionAnnotationPolicyStaticValuesJDK21(t *testing.T) {
	runAccessibleAnnotationFixture(t, "annotation_policy_static", "policy.StaticPolicyWitness")
}

func TestReflectionAnnotationMetadataUsesOwnedDeclarationContext(t *testing.T) {
	ctx, scope := reader132ActualLateScope(t)
	ctx.currentClass = scope
	ctx.localScope = nil
	if got := reflectionAnnotationNominalOwner("Override", nil, ctx); got != "java.lang.Override" {
		t.Fatal("owned metadata incorrectly required executable caller", got)
	}
	if got := reflectionAnnotationNominalOwner("Retention", nil, ctx); got != "" {
		t.Fatal("missing annotation import borrowed canonical policy", got)
	}
	if got := reflectionAnnotationNominalOwner("java.lang.annotation.Retention", nil, ctx); got != "java.lang.annotation.Retention" {
		t.Fatal("owned qualified declaration lost canonical identity", got)
	}
	copied := *scope
	wrong := ctx.Clone()
	wrong.currentClass = &copied
	if got := reflectionAnnotationNominalOwner("Override", nil, wrong); got != "" {
		t.Fatal("copied scope donated declaration ownership", got)
	}
	missing := ctx.Clone()
	missing.anonymousClasses = nil
	if got := reflectionAnnotationNominalOwner("Override", nil, missing); got != "" {
		t.Fatal("missing late registry donated annotation identity", got)
	}
	wrong = ctx.Clone()
	file := *ctx.currentFile
	wrong.currentFile = &file
	if got := reflectionAnnotationNominalOwner("Override", nil, wrong); got != "" {
		t.Fatal("copied source file donated annotation identity", got)
	}
	shadow := ordinaryDemandTestContext(t, map[string]string{"Host.java": `class Host<Override> {}`}, "Host.java")
	if got := reflectionAnnotationNominalOwner("Override", nil, shadow); got != "" {
		t.Fatal("source TYPE binder borrowed native annotation", got)
	}
}
