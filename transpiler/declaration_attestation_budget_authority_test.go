package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/symbol"
)

func requireBudgetRetired199(t *testing.T, facts *namedDeclarationCandidates, ctx Ctx) {
	t.Helper()
	requireRetiredDeclaration196(t, facts, ctx)
	if facts.files != nil || facts.owners != nil || facts.packages != nil {
		t.Fatal("retired admission retained declaration tables")
	}
}

func TestDeclarationAttestationBudget199Authority(t *testing.T) {
	if declarationSourceAttestationLimit != 256 {
		t.Fatal("reviewed fixed attestation budget changed")
	}
	t.Run("256-admitted", func(t *testing.T) {
		f, facts := ownedDeclarationFixture196(t, []string{attestationBudgetSource199(256, "Use")})
		if facts.retired {
			t.Fatal("exact budget rejected")
		}
		if got, known := indexedDirectDeclaration(f.ctx.currentClass, "Member", f.ctx); !known || got != f.ctx.currentClass.Subclasses[0] {
			t.Fatal("exact budget lost direct index authority")
		}
	})
	for _, crossPackage := range []bool{false, true} {
		name, peer := "mixed-same-package", attestationBudgetSource199(257, "Peer")
		if crossPackage {
			name, peer = "mixed-other-package", "package q;"+peer[len("package p;"):]
		}
		t.Run(name, func(t *testing.T) {
			f, facts := ownedDeclarationFixture196(t, []string{attestationBudgetSource199(256, "Use"), peer})
			requireBudgetRetired199(t, facts, f.ctx)
			original := f.files[1].Symbols.Source
			f.files[1].Symbols.Source = []byte("package p; class Peer {}")
			requireBudgetRetired199(t, facts, f.ctx)
			f.files[1].Symbols.Source = original
			requireBudgetRetired199(t, facts, f.ctx)
		})
	}
	t.Run("257-rejected-before-tables", func(t *testing.T) {
		f, facts := ownedDeclarationFixture196(t, []string{attestationBudgetSource199(257, "Use")})
		requireBudgetRetired199(t, facts, f.ctx)
		if _, known := indexedFileDeclaration(f.ctx.currentFile, "Use", f.ctx); known {
			t.Fatal("rejected file supplied cached answer")
		}
	})
	t.Run("new-small-invocation-admits", func(t *testing.T) {
		f, facts := ownedDeclarationFixture196(t, []string{attestationBudgetSource199(256, "Use")})
		if declarationCandidates(f.ctx) != facts {
			t.Fatal("unrelated retired invocation poisoned fresh small owner")
		}
	})
}

func TestDeclarationAttestationBudget199AccessorLocalChecks(t *testing.T) {
	for _, replaced := range []int{0, 1, 2} {
		t.Run(fmtSizeBudget199(replaced), func(t *testing.T) {
			f, facts := ownedDeclarationFixture196(t, []string{
				attestationBudgetSource199(256, "Use"), attestationBudgetSource199(256, "Peer"), attestationBudgetSource199(256, "Third"),
			})
			same := declarationAccess(f.ctx)
			if same.facts != facts || same.checked[0] != f.files[0].Symbols || same.checked[1] != nil {
				t.Fatal("pointer-identical active/context checks were not local and deduplicated")
			}
			ctx := f.ctx.Clone()
			ctx.currentFile = f.files[1].Symbols
			access := declarationAccess(ctx)
			if !access.currentFile(f.files[2].Symbols) || !access.currentFile(f.files[1].Symbols) {
				t.Fatal("distinct-file attestation failed")
			}
			if access.checked != [3]*symbol.FileScope{f.files[0].Symbols, f.files[1].Symbols, f.files[2].Symbols} {
				t.Fatal("accessor did not attest each distinct active/context/queried file exactly once")
			}
			file := f.files[replaced].Symbols
			original := file.Source
			file.Source = append(append([]byte(nil), original...), ' ')
			if _, known := indexedFileDeclaration(f.files[2].Symbols, "Third", ctx); known {
				t.Fatal("new accessor reused an earlier accessor's source authority")
			}
			file.Source = original
			requireBudgetRetired199(t, facts, ctx)
		})
	}
}

func TestDeclarationAttestationBudget199UnknownObservedRetirement(t *testing.T) {
	for _, ownerQuery := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "owner"}[ownerQuery], func(t *testing.T) {
			f := qualifiedLookup195Parsed(t, []string{attestationBudgetSource199(256, "Use"), attestationBudgetSource199(256, "Peer")})
			unknown := f.files[1].Symbols
			unknown.BaseClass.Class.DeclarationNode = nil
			session := cliownership.New()
			defer session.Close()
			invocation := newSyntaxInvocation(session, f.files)
			if invocation == nil || invocation.declarations == nil {
				t.Fatal("owned UNKNOWN fixture unavailable")
			}
			defer invocation.close()
			ctx := f.ctx.Clone()
			ctx.syntaxInvocation = invocation
			ctx.callableSubclasses = &callableSubclassSourceInventory{}
			activeResolutionFiles = sourceOwnershipIndex(ctx)
			facts := invocation.declarations
			if facts.retired || facts.files[unknown].complete {
				t.Fatal("UNKNOWN setup lost eligible table facts")
			}
			original := unknown.Source
			unknown.Source = append(append([]byte(nil), original...), ' ')
			var known bool
			if ownerQuery {
				_, known = indexedDirectDeclaration(unknown.BaseClass, "Absent", ctx)
			} else {
				_, known = indexedFileDeclaration(unknown, "Absent", ctx)
			}
			if known {
				t.Fatal("UNKNOWN over-budget region supplied cached absence")
			}
			unknown.Source = original
			requireBudgetRetired199(t, facts, ctx)
		})
	}
}
