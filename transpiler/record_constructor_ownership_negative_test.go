package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestRecordConstructorEmptyRequiresCurrentOwnedSource(t *testing.T) {
	for _, mutation := range []string{"graph", "source", "name"} {
		t.Run(mutation, func(t *testing.T) {
			h := setupParseHelper(t, `class Host { record Empty() {} }`)
			owner := h.File.Symbols.FindClassScope("Empty")
			if owner == nil {
				t.Fatal("record missing")
			}
			if _, known := recordConstructorComponents(owner); !known {
				t.Fatal("owned empty record rejected")
			}
			switch mutation {
			case "graph":
				graph, active := symbol.GlobalScope, activeResolutionFiles
				symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
				activeResolutionFiles = nil
				defer func() { symbol.GlobalScope = graph; activeResolutionFiles = active }()
			case "source":
				before := append([]byte(nil), h.File.Symbols.Source...)
				h.File.Symbols.Source[0] ^= 1
				defer copy(h.File.Symbols.Source, before)
			case "name":
				before := owner.Class.OriginalName
				owner.Class.OriginalName = "Other"
				defer func() { owner.Class.OriginalName = before }()
			}
			if _, known := recordConstructorComponents(owner); known {
				t.Fatal("empty record borrowed missing/stale/name-mismatched ownership")
			}
		})
	}
}

func TestRecordConstructorRejectsDonatedWrittenNode(t *testing.T) {
	h := setupParseHelper(t, `class Host { record R(int x){R(int x){this.x=x;} private R(String s){this(s.length());}} record S(int x){S(int x){this.x=x;}} }`)
	owner := h.File.Symbols.FindClassScope("R")
	other := h.File.Symbols.FindClassScope("S")
	ctx := classScopeCtx(owner, h.Ctx)
	canonical := recordCanonicalConstructor(owner, ctx)
	foreign := recordCanonicalConstructor(other, h.Ctx)
	if canonical == nil || foreign == nil {
		t.Fatal("canonical fixture missing")
	}
	prior := canonical.DeclarationNode
	canonical.DeclarationNode = foreign.DeclarationNode
	if recordCanonicalConstructor(owner, ctx) != nil {
		t.Fatal("canonical selected a donated written constructor node")
	}
	canonical.DeclarationNode = prior
	var overload *symbol.Definition
	for _, method := range owner.Methods {
		if method.Constructor && method != canonical {
			overload = method
		}
	}
	if overload == nil {
		t.Fatal("overload fixture missing")
	}
	prior = overload.DeclarationNode
	overload.DeclarationNode = foreign.DeclarationNode
	defer func() { overload.DeclarationNode = prior }()
	if recordCanonicalConstructor(owner, ctx) != nil {
		t.Fatal("foreign overload left canonical metadata eligible")
	}
}

func TestRecordConstructorTailFailureStrictlyRefuses(t *testing.T) {
	h := setupParseHelper(t, `class Host {record R(int x){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = nil
	strictRoutingState(t)
	var failure any
	func() { defer func() { failure = recover() }(); recordCanonicalTail(ctx, "receiver") }()
	refused, known := failure.(strictModeError)
	if !known || refused.diagnostic.Kind != "record canonical constructor component ownership" {
		t.Fatalf("tail failure did not explicitly refuse: %T %v", failure, failure)
	}
}

func TestRecordConstructorSameNameNestedOwnerDonation(t *testing.T) {
	for _, components := range []string{"", "int x"} {
		t.Run(components, func(t *testing.T) {
			h := setupParseHelper(t, "class A {record R("+components+"){}} class B {record R("+components+"){}}")
			a := h.File.Symbols.FindClassScope("A").FindClassScope("R")
			b := h.File.Symbols.FindClassScope("B").FindClassScope("R")
			if _, known := recordConstructorComponents(a); !known {
				t.Fatal("genuine A.R rejected")
			}
			if _, known := recordConstructorComponents(b); !known {
				t.Fatal("genuine B.R rejected")
			}
			before := a.Class.DeclarationNode
			a.Class.DeclarationNode = b.Class.DeclarationNode
			defer func() { a.Class.DeclarationNode = before }()
			if _, known := recordConstructorComponents(a); known {
				t.Fatal("same-name foreign lexical owner accepted")
			}
		})
	}
}
