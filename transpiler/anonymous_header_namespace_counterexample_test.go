package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"testing"
)

func TestGenericFamilyAnonymousCreationHeaderOriginalNamespace(t *testing.T) {
	b, err := os.ReadFile("testdata/anonymous_header_namespace/AnonymousCreationHeaderNamespaceBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	h := setupParseHelper(t, string(b))
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p == nil || err != nil {
		t.Fatalf("creation header rebound under inherited body namespace: %v", err)
	}
}

func TestGenericFamilyAnonymousHeaderParentDeclarationIdentity(t *testing.T) {
	b, err := os.ReadFile("testdata/anonymous_header_namespace/AnonymousCreationHeaderNamespaceBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	h := setupParseHelper(t, string(b))
	events := newGenericFamilyInventory(h.Ctx).genericFamilySourceEvents(h.Ctx)
	if len(events) != 1 {
		t.Fatal("missing exact header event")
	}
	if _, _, err := genericFamilyAnonymousAuditScope(events[0]); err != nil {
		t.Fatal("original header identity refused", err)
	}
	var inherited *symbol.ClassScope
	for _, c := range allSourceClassScopes() {
		if c.Enclosing != nil && c.Enclosing.Class != nil && c.Enclosing.Class.OriginalName == "Namespace" && c.Class.OriginalName == "Source" {
			inherited = c
		}
	}
	if inherited == nil {
		t.Fatal("inherited homonymous member missing")
	}
	event := events[0]
	event.parent = inherited
	if _, _, err := genericFamilyAnonymousAuditScope(event); err == nil {
		t.Fatal("unrelated inherited parent borrowed original header proof")
	}
}
