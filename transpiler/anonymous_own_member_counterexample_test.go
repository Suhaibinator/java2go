package transpiler

import (
	"go/ast"
	"os"
	"strings"
	"testing"
)

func anonymousOwnMemberBoundary(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/anonymous_own_member/AnonymousOwnMemberTypeBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func TestGenericFamilyAnonymousOwnMemberTypeRefusal(t *testing.T) {
	original := anonymousOwnMemberBoundary(t)
	for _, test := range []struct{ name, member, field string }{
		{"class", `class Box<X> { java.util.List<X> unsafe; }`, `Box<M> member;`},
		{"interface", `interface Box<X> {}`, `Box<M> member;`},
		{"enum", `enum Box { ONE }`, `Box member;`},
		{"record", `record Box<X>(X value) {}`, `Box<M> member;`},
		{"annotation", `@interface Box {}`, `Box member;`},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(original, `class Box<X> { java.util.List<X> unsafe; }`, test.member, 1)
			source = strings.Replace(source, `Box<M> member;`, test.field, 1)
			h := setupParseHelper(t, source)
			if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
				t.Fatal("unmodeled own member type borrowed unrelated top-level declaration")
			}
		})
	}
	t.Run("method_local_type_unchanged", func(t *testing.T) {
		source := strings.Replace(original, `class Box<X> { java.util.List<X> unsafe; }`, "", 1)
		source = strings.Replace(source, `Box<M> member;`, "", 1)
		source = strings.Replace(source, `return null;`, `class Local<X>{java.util.List<X> separate;}return null;`, 1)
		h := setupParseHelper(t, source)
		if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p == nil || err != nil {
			t.Fatalf("separate method-local type incorrectly treated as anonymous member: %v", err)
		}
	})
}

func TestAnonymousOwnMemberTypeStrictDiagnostic(t *testing.T) {
	source := anonymousOwnMemberBoundary(t)
	h := setupParseHelper(t, source)
	events := newGenericFamilyInventory(h.Ctx).genericFamilySourceEvents(h.Ctx)
	if len(events) != 1 {
		t.Fatal("missing actual allocation event")
	}
	ctx := events[0].lexical.Clone()
	ctx.className = ctx.currentClass.Class.Name
	var declarations []ast.Decl
	counter := 0
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{}
	ctx.localClasses = map[string]*localClassInfo{}
	resetDiagnostics()
	setStrictMode(true)
	defer func() { setStrictMode(false); resetDiagnostics() }()
	var caught any
	func() { defer func() { caught = recover() }(); ParseExpr(events[0].node, h.File.Source, ctx) }()
	e, ok := caught.(strictModeError)
	if !ok || e.diagnostic.Kind != "anonymous member type declaration" {
		t.Fatalf("missing explicit own member-type strict refusal: %#v", caught)
	}
	if len(Diagnostics()) != 1 || len(declarations) != 0 {
		t.Fatal("unsupported member type emitted partial anonymous declarations")
	}
}
