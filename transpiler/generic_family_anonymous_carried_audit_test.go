package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"strings"
	"testing"
)

func anonymousCarriedBoundary(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/published_anonymous_carried/PublishedAnonymousCarriedBinderBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func anonymousCarriedFactory(t *testing.T) *symbol.ClassScope {
	t.Helper()
	for _, s := range allSourceClassScopes() {
		if s.Class != nil && s.Class.OriginalName == "ExternalFactory" {
			return s
		}
	}
	t.Fatal("missing real external factory declaration")
	return nil
}
func TestGenericFamilyAnonymousCarriedStorageRefusals(t *testing.T) {
	for _, typ := range []string{"java.util.List<M>", "java.util.HashMap<String,M>", "M[]"} {
		t.Run(typ, func(t *testing.T) {
			source := strings.Replace(anonymousCarriedBoundary(t), "java.util.List<M>", typ, 1)
			h := setupParseHelper(t, source)
			if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("carried method declaration donated unsupported storage", typ)
			}
		})
	}
	for _, decl := range []string{"java.util.List<M>", "java.util.HashMap<String,M>", "M[]"} {
		t.Run("capture_"+decl, func(t *testing.T) {
			source := strings.Replace(anonymousCarriedBoundary(t), "make()", "make("+decl+" input)", 1)
			source = strings.Replace(source, "java.util.List<M> unsafe;", "", 1)
			source = strings.Replace(source, "return null;", "input.toString();return null;", 1)
			h := setupParseHelper(t, source)
			if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("captured field escaped carried storage audit", decl)
			}
		})
	}
}
func TestGenericFamilyAnonymousCarriedMapAndMethodShadow(t *testing.T) {
	for _, body := range []string{"java.util.Map<String,M> supported;", `public <M> java.util.List<M> identity(java.util.List<M> value){return value;}`} {
		t.Run(body, func(t *testing.T) {
			source := strings.Replace(anonymousCarriedBoundary(t), "java.util.List<M> unsafe;", body, 1)
			h := setupParseHelper(t, source)
			factory := anonymousCarriedFactory(t)
			method := factory.Methods[0]
			if len(method.TypeParameters) != 1 {
				t.Fatal("missing own original method declaration")
			}
			p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
			if p == nil || err != nil {
				t.Fatalf("existing Map contract or own method shadow refused: %v", err)
			}
			rep, ok := p.representations[method.TypeParameters[0].Declaration]
			if !ok || rep.goName != "DateValue" || len(p.binders) != 4 {
				t.Fatal("carried declaration identity or shadow boundary lost", len(p.binders))
			}
		})
	}
}
func TestGenericFamilyAnonymousCaptureDefinitionsArePure(t *testing.T) {
	source := strings.Replace(anonymousCarriedBoundary(t), "make()", "make(M input)", 1)
	source = strings.Replace(source, "java.util.List<M> unsafe;", "", 1)
	source = strings.Replace(source, "return null;", "return input;", 1)
	h := setupParseHelper(t, source)
	h.Ctx.genericFamilies = &genericFamilyAnalysis{}
	inventory := newGenericFamilyInventory(h.Ctx)
	events := inventory.genericFamilySourceEvents(h.Ctx)
	if len(events) != 1 {
		t.Fatal("missing original anonymous event", len(events))
	}
	scope, captures, err := genericFamilyAnonymousAuditScope(events[0])
	if err != nil || scope == nil || len(captures) != 1 || captures[0].goType != nil || h.Ctx.genericFamilies.ready {
		t.Fatalf("planning entered physical capture conversion: captures=%d ready=%v err=%v", len(captures), h.Ctx.genericFamilies.ready, err)
	}
	factory := anonymousCarriedFactory(t)
	if len(scope.TypeParameters) != 1 || scope.TypeParameters[0].Declaration != factory.Methods[0].TypeParameters[0].Declaration {
		t.Fatal("capture lost exact external method declaration")
	}
}
func TestGenericFamilyAnonymousEmissionCannotDonateBinders(t *testing.T) {
	source := strings.Replace(anonymousCarriedBoundary(t), "java.util.List<M> unsafe;", "M scalar;", 1)
	h := setupParseHelper(t, source)
	p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
	if p == nil || err != nil || len(p.anonymous) != 1 {
		t.Fatalf("scalar carried body refused: %v", err)
	}
	var frame genericFamilyAnonymous
	for _, v := range p.anonymous {
		frame = v
	}
	scope := &symbol.ClassScope{Class: &symbol.Definition{Name: "Synthetic", OriginalName: "Synthetic", DeclarationNode: frame.node}, Enclosing: frame.owner, TypeParameters: append([]symbol.TypeParam(nil), frame.carried...)}
	scope.ImplementedInterfaces = []string{frame.javaType}
	ctx := h.Ctx.Clone()
	ctx.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{p}}
	beforeMembers, beforeBinders := len(p.members), len(p.binders)
	if got := canonicalGenericFamily(scope, ctx); got == nil {
		t.Fatal("exact audited synthetic instance refused")
	}
	fakeParent := *scope
	fakeParent.ImplementedInterfaces = []string{"Unrelated<M>"}
	if canonicalGenericFamily(&fakeParent, ctx) != nil {
		t.Fatal("unrelated parent borrowed anonymous body proof")
	}
	fake := *scope
	fake.TypeParameters = append([]symbol.TypeParam(nil), scope.TypeParameters...)
	fake.TypeParameters[0] = symbol.NewTypeParam("M", nil)
	if canonicalGenericFamily(&fake, ctx) != nil {
		t.Fatal("same-named fresh binder donated physical representation")
	}
	fake = *scope
	fake.TypeParameters = nil
	if canonicalGenericFamily(&fake, ctx) != nil {
		t.Fatal("missing carried binder borrowed body proof")
	}
	graph := symbol.GlobalScope
	foreign := setupParseHelper(t, source)
	foreignInventory := newGenericFamilyInventory(foreign.Ctx)
	foreignEvents := foreignInventory.genericFamilySourceEvents(foreign.Ctx)
	symbol.GlobalScope = graph
	if len(foreignEvents) != 1 {
		t.Fatal("foreign equal-span event missing")
	}
	fake = *scope
	def := *scope.Class
	fake.Class = &def
	fake.Class.DeclarationNode = foreignEvents[0].node
	if canonicalGenericFamily(&fake, ctx) != nil {
		t.Fatal("foreign equal-span AST borrowed body proof")
	}
	if len(p.members) != beforeMembers || len(p.binders) != beforeBinders {
		t.Fatal("instance checking mutated admitted plan")
	}
}

func TestGenericFamilyAnonymousProofPreservesDeclarationNamespaces(t *testing.T) {
	t.Run("captured_outer_type", func(t *testing.T) {
		b, err := os.ReadFile("testdata/anonymous_inherited_member/AnonymousInheritedMemberTypeBoundary.java")
		if err != nil {
			t.Fatal(err)
		}
		source := strings.Replace(string(b), "make()", "make(Box<M> input)", 1)
		source = strings.Replace(source, "Box<M> inherited;", "", 1)
		source = strings.Replace(source, "return null;", "input.toString();return null;", 1)
		h := setupParseHelper(t, source)
		p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
		if p == nil || err != nil {
			t.Fatalf("captured outer Box rebound under anonymous inheritance: %v", err)
		}
		if _, ok := p.members[h.File.Symbols.FindClassScope("Box")]; !ok {
			t.Fatal("original top-level captured Box missing from complete plan")
		}
	})
	t.Run("carried_bound", func(t *testing.T) {
		source := strings.Replace(anonymousCarriedBoundary(t), "interface Source<A>{A get();}", "interface Source<A>{A get();class Date{}}", 1)
		source = "import java.util.Date;\n" + strings.Replace(source, "extends java.util.Date", "extends Date", 1)
		source = strings.Replace(source, "java.util.List<M> unsafe;", "M scalar;", 1)
		h := setupParseHelper(t, source)
		factory := anonymousCarriedFactory(t)
		p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx)
		if p == nil || err != nil {
			t.Fatalf("factory Date bound rebound to inherited Source.Date: %v", err)
		}
		if p.representations[factory.Methods[0].TypeParameters[0].Declaration].goName != "DateValue" {
			t.Fatal("original factory declaration native Date representation lost")
		}
		var frame genericFamilyAnonymous
		for _, f := range p.anonymous {
			frame = f
		}
		scope := &symbol.ClassScope{Class: &symbol.Definition{Name: "Synthetic", DeclarationNode: frame.node}, Enclosing: frame.owner, TypeParameters: append([]symbol.TypeParam(nil), frame.carried...), ImplementedInterfaces: []string{frame.javaType}}
		ctx := h.Ctx.Clone()
		ctx.genericFamilies = &genericFamilyAnalysis{ready: true, plans: []*genericFamilyPlan{p}}
		if canonicalGenericFamily(scope, ctx) == nil {
			t.Fatal("late view rebound original factory bound to inherited Date")
		}
	})
}
