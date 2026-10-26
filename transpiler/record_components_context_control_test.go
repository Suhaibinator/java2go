package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/printer"
	"go/token"
	"testing"
)

func TestRecordComponentReviewSyntheticContextBound(t *testing.T) {
	h := setupParseHelper(t, `class Host{record R<T extends Number>(T count){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("genuine bounded record rejected")
	}
	var old bytes.Buffer
	if err := printer.Fprint(&old, token.NewFileSet(), before[0].typ); err != nil {
		t.Fatal(err)
	}
	copied := owner.TypeParameters[0]
	copied.Bounds = []symbol.JavaType{{Original: "Object"}}
	ctx.syntheticTypeParameters = []symbol.TypeParam{copied}
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		return
	}
	var changed bytes.Buffer
	if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
		t.Fatal(err)
	}
	if changed.String() != old.String() {
		t.Fatalf("synthetic context rebound record header erasure: before=%s after=%s", old.String(), changed.String())
	}
}

func TestRecordComponentReviewDeclarationContextNormalization(t *testing.T) {
	h := setupParseHelper(t, `class Host{record R<T extends Number>(T count){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned profile rejected")
	}
	copied := owner.TypeParameters[0]
	copied.Bounds = []symbol.JavaType{{Original: "Object"}}
	ctx.syntheticTypeParameters = []symbol.TypeParam{copied}
	ctx.localScope = &symbol.Definition{OriginalName: "unrelatedMethod", IsStatic: true, TypeParameters: []symbol.TypeParam{copied}}
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("unrelated caller function rejected owned header")
	}
	var a, b bytes.Buffer
	if err := printer.Fprint(&a, token.NewFileSet(), before[0].typ); err != nil {
		t.Fatal(err)
	}
	if err := printer.Fprint(&b, token.NewFileSet(), after[0].typ); err != nil {
		t.Fatal(err)
	}
	if a.String() != b.String() {
		t.Fatalf("caller function changed header identity: %s -> %s", a.String(), b.String())
	}
}

func TestRecordComponentReviewPhysicalDeclarationContext(t *testing.T) {
	h := setupParseHelper(t, `class Host{record R<T extends Number>(T count){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned physical profile rejected")
	}
	bridgesBefore := sourceReflectionRecordAccessorBridges(owner, ctx)
	copied := owner.TypeParameters[0]
	copied.Declaration = &symbol.TypeParamDeclaration{SourceName: "T", GoName: "ForeignFunctionT"}
	copied.Bounds = []symbol.JavaType{{Original: "Object"}}
	ctx.syntheticTypeParameters = []symbol.TypeParam{copied}
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned physical profile rejected due unrelated caller binder")
	}
	bridgesAfter := sourceReflectionRecordAccessorBridges(owner, ctx)
	for label, pair := range map[string][2]any{"descriptor": {before[0].typ, after[0].typ}, "physical": {before[0].physical, after[0].physical}, "bridge": {bridgesBefore[0], bridgesAfter[0]}} {
		var a, b bytes.Buffer
		if err := printer.Fprint(&a, token.NewFileSet(), pair[0]); err != nil {
			t.Fatal(err)
		}
		if err := printer.Fprint(&b, token.NewFileSet(), pair[1]); err != nil {
			t.Fatal(err)
		}
		if a.String() != b.String() {
			t.Fatalf("caller changed %s: %s -> %s", label, a.String(), b.String())
		}
	}
}

func TestRecordComponentReviewPositiveDeclarationContext(t *testing.T) {
	h := setupParseHelper(t, `class Host{record R<T extends Number>(T count){} class Foreign{}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned profile rejected")
	}
	bridgesBefore := sourceReflectionRecordAccessorBridges(owner, ctx)
	foreign := h.File.Symbols.FindClassScope("Foreign")
	copied := owner.TypeParameters[0]
	copied.Declaration = &symbol.TypeParamDeclaration{SourceName: "T", GoName: "ForeignFunctionT"}
	copied.Bounds = []symbol.JavaType{{Original: "Object"}}
	ctx.currentClass = foreign
	ctx.localScope = &symbol.Definition{OriginalName: "body", IsStatic: true, TypeParameters: []symbol.TypeParam{copied}}
	ctx.syntheticTypeParameters = []symbol.TypeParam{copied}
	ctx.rawGenericParameterTypes = map[string]string{"T": "Object", "count": "Object"}
	ctx.expectedType = "foreign.Counterfeit"
	ctx.memberTypeHeaderOwner = foreign
	ctx.memberTypeLookupPath = map[*symbol.ClassScope]bool{owner: true}
	ctx.localClasses = map[string]*localClassInfo{"Number": {scope: foreign, dynamicTypeID: "foreign.Counterfeit"}}
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{}: {structName: "Number", scope: foreign}}
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("body context poisoned owned declaration")
	}
	bridgesAfter := sourceReflectionRecordAccessorBridges(owner, ctx)
	for label, pair := range map[string][2]any{"descriptor": {before[0].typ, after[0].typ}, "physical": {before[0].physical, after[0].physical}, "bridge": {bridgesBefore[0], bridgesAfter[0]}} {
		var a, b bytes.Buffer
		if err := printer.Fprint(&a, token.NewFileSet(), pair[0]); err != nil {
			t.Fatal(err)
		}
		if err := printer.Fprint(&b, token.NewFileSet(), pair[1]); err != nil {
			t.Fatal(err)
		}
		if a.String() != b.String() {
			t.Fatalf("body context changed %s: %s -> %s", label, a.String(), b.String())
		}
	}
}
