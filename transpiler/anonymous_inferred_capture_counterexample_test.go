package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"strings"
	"testing"
)

func anonymousInferredCaptureBoundary(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/anonymous_inferred_capture/AnonymousInferredCaptureBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func TestGenericFamilyAnonymousUnknownInferredCaptureRefusal(t *testing.T) {
	h := setupParseHelper(t, anonymousInferredCaptureBoundary(t))
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("unresolved var capture admitted before native container refinement")
	}
}
func TestGenericFamilyAnonymousCaptureTypeRefinementDoesNotBorrowPlan(t *testing.T) {
	source := strings.Replace(anonymousInferredCaptureBoundary(t), "var input = new java.util.ArrayList<M>();", "M input = null;", 1)
	h := setupParseHelper(t, source)
	p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
	if p == nil || err != nil {
		t.Fatalf("typed scalar capture refused before drift: %v", err)
	}
	var frame genericFamilyAnonymous
	for _, f := range p.anonymous {
		frame = f
	}
	captures := collectCapturedLocalDefinitions(objectCreationClassBody(frame.node), h.File.Source, frame.declarationCtx)
	if len(captures) != 1 {
		t.Fatal("missing source capture")
	}
	field := *captures[0].javaDef
	scope := &symbol.ClassScope{Class: &symbol.Definition{Name: "Synthetic", DeclarationNode: frame.node}, Enclosing: frame.owner, TypeParameters: append([]symbol.TypeParam(nil), frame.carried...), ImplementedInterfaces: []string{frame.javaType}, Fields: []*symbol.Definition{&field}}
	ctx := h.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{p}}
	if canonicalGenericFamily(scope, ctx) == nil {
		t.Fatal("unchanged original typed capture refused")
	}
	recordLocalVariableDefinition(frame.declarationCtx, "input", "java.util.ArrayList<M>", "*List[M]")
	refined := frame.declarationCtx.localScope.FindVariable("input")
	if refined == nil || refined.OriginalType != "java.util.ArrayList<M>" {
		t.Fatal("real metadata refinement helper did not update capture")
	}
	field = *refined
	scope.Fields = []*symbol.Definition{&field}
	if canonicalGenericFamily(scope, ctx) != nil {
		t.Fatal("late native container capture refinement borrowed scalar body proof")
	}
}

func TestGenericFamilyAnonymousCaptureSnapshotIgnoresGoNameChanges(t *testing.T) {
	source := strings.Replace(anonymousInferredCaptureBoundary(t), "var input = new java.util.ArrayList<M>();", "M input = null;", 1)
	h := setupParseHelper(t, source)
	p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
	if p == nil || err != nil {
		t.Fatal("typed capture refused", err)
	}
	var frame genericFamilyAnonymous
	for _, f := range p.anonymous {
		frame = f
	}
	captures := collectCapturedLocalDefinitions(objectCreationClassBody(frame.node), h.File.Source, frame.declarationCtx)
	if len(captures) != 1 || len(frame.captures) != 1 {
		t.Fatal("missing immutable capture proof")
	}
	field := *captures[0].javaDef
	scope := &symbol.ClassScope{Class: &symbol.Definition{Name: "Synthetic", DeclarationNode: frame.node}, Enclosing: frame.owner, TypeParameters: append([]symbol.TypeParam(nil), frame.carried...), ImplementedInterfaces: []string{frame.javaType}, Fields: []*symbol.Definition{&field}}
	ctx := h.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{p}}
	captures[0].javaDef.Name = "renamedForGo"
	field.Name = "renamedCapture"
	if canonicalGenericFamily(scope, ctx) == nil {
		t.Fatal("Go-only hygiene rename changed Java capture identity")
	}
	declaration := captures[0].javaDef.DirectTypeParameter
	forged := symbol.NewTypeParam("M", nil)
	captures[0].javaDef.DirectTypeParameter = forged.Declaration
	captures[0].javaDef.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"M": forged.Declaration}
	field = *captures[0].javaDef
	if canonicalGenericFamily(scope, ctx) != nil {
		t.Fatal("new same-spelled capture type declaration borrowed immutable proof")
	}
	if frame.captures[0].logical.parameter != declaration {
		t.Fatal("stored capture type followed mutable binding map")
	}
}
