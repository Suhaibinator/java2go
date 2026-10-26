package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

func ownedDeclarationFixture196(t *testing.T, sources []string) (*qualifiedLookup195Fixture, *namedDeclarationCandidates) {
	t.Helper()
	f := qualifiedLookup195Parsed(t, sources)
	session := cliownership.New()
	invocation := newSyntaxInvocation(session, f.files)
	if invocation == nil || invocation.declarations == nil {
		t.Fatal("actual parsed owned invocation unavailable")
	}
	t.Cleanup(func() { invocation.close(); session.Close() })
	f.ctx.syntaxInvocation = invocation
	f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
	activeResolutionFiles = sourceOwnershipIndex(f.ctx)
	return f, invocation.declarations
}

func requireRetiredDeclaration196(t *testing.T, facts *namedDeclarationCandidates, ctx Ctx) {
	t.Helper()
	if !facts.retired || declarationCandidates(ctx) != nil || declarationCandidates() != nil {
		t.Fatal("observed ownership loss did not permanently retire declaration facts")
	}
}

func TestNamedDeclaration196ActiveRenderAuthority(t *testing.T) {
	f, facts := ownedDeclarationFixture196(t, []string{`package p; class Use {static class Member {}}`})
	owner, child := f.ctx.currentClass, f.ctx.currentClass.Subclasses[0]
	outer := activeResolutionFiles
	if got, known := indexedDirectDeclaration(owner, "Member", f.ctx); !known || got != child {
		t.Fatal("owned direct candidate not selected")
	}
	public := f.ctx.Clone()
	public.syntaxInvocation = nil
	public.sourceOwnership = outer
	activeResolutionFiles = sourceOwnershipIndex(public)
	if activeResolutionFiles == outer || activeResolutionFiles.declarations != nil {
		t.Fatal("nested public render overwrote outer authority")
	}
	if _, known := indexedDirectDeclaration(owner, "Member", f.ctx); known {
		t.Fatal("retained outer context bypassed current public render")
	}
	child.Class.OriginalName = "Changed"
	if got := memberTypeInHierarchy(owner, "Changed", f.ctx); got != child {
		t.Fatal("nested public render did not use live original member scan")
	}
	child.Class.OriginalName = "Member"
	activeResolutionFiles = outer
	if facts.retired {
		t.Fatal("nested public render retired live outer invocation")
	}
	if got, known := indexedDirectDeclaration(owner, "Member", f.ctx); !known || got != child {
		t.Fatal("restored outer render lost its own authority")
	}

	session := cliownership.New()
	invocation := newSyntaxInvocation(session, f.files)
	if invocation == nil || invocation.declarations == nil {
		t.Fatal("fresh nested invocation unavailable")
	}
	defer session.Close()
	defer invocation.close()
	nested := f.ctx.Clone()
	nested.syntaxInvocation, nested.sourceOwnership = invocation, outer
	activeResolutionFiles = sourceOwnershipIndex(nested)
	if _, known := indexedDirectDeclaration(owner, "Member", f.ctx); known {
		t.Fatal("retained outer context bypassed new owning invocation")
	}
	if got, known := indexedDirectDeclaration(owner, "Member", nested); !known || got != child {
		t.Fatal("current nested owner lost candidate authority")
	}
	if got, known := indexedDirectDeclaration(owner, "Member", Ctx{}); !known || got != child {
		t.Fatal("fresh ancestry context did not use current render's authority")
	}
	activeResolutionFiles = outer
	if declarationCandidates(f.ctx) != facts {
		t.Fatal("nested ownership restoration lost outer facts")
	}
}

func TestNamedDeclaration196ResetRetirement(t *testing.T) {
	for _, mode := range []string{"holder", "pre-attachment-alias", "inventory", "inventory-alias"} {
		t.Run(mode, func(t *testing.T) {
			f, facts := ownedDeclarationFixture196(t, []string{`package p; class Use {static class Member {}}`})
			ctx := f.ctx.Clone()
			switch mode {
			case "holder":
				ctx.sourceOwnership = activeResolutionFiles
			case "pre-attachment-alias":
				ctx.sourceOwnership = newResolutionFileIndex()
				activeResolutionFiles = sourceOwnershipIndex(ctx)
			case "inventory-alias":
				ctx.callableSubclasses.ownership = nil
			}
			original := symbol.GlobalScope
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
			if mode == "inventory" || mode == "inventory-alias" {
				resolvedSourceInventory(ctx)
			} else {
				sourceOwnershipIndex(ctx)
			}
			symbol.GlobalScope = original
			activeResolutionFiles = sourceOwnershipIndex(ctx)
			requireRetiredDeclaration196(t, facts, ctx)
			if got := resolveClassScopeByQualifiedName(ctx, "p.Use.Member"); got != ctx.currentClass.Subclasses[0] {
				t.Fatal("retirement lost original resolution fallback")
			}
		})
	}
}

func TestNamedDeclaration196ObservedRootAndSourceRetirement(t *testing.T) {
	for _, mode := range []string{"direct-miss", "file-miss", "context-free-miss", "returned-package", "returned-qualified", "root"} {
		t.Run(mode, func(t *testing.T) {
			f, facts := ownedDeclarationFixture196(t, []string{
				`package p; class Use {static class Member {}}`,
				`package p; class Peer {static class Child {}}`,
			})
			file := f.files[0].Symbols
			if mode == "returned-package" || mode == "returned-qualified" {
				file = f.files[1].Symbols
			}
			original := *file
			if mode == "root" {
				// Whole FileScope assignment is a public way to replace the hidden
				// ParsedRoot anchor without mutating either original parser tree.
				*file = *f.files[1].Symbols
			} else {
				file.Source = append([]byte(nil), file.Source...)
				file.Source[0] = 'P' // same size; a length/pointer epoch is insufficient.
			}
			switch mode {
			case "direct-miss", "root":
				if _, known := indexedDirectDeclaration(original.BaseClass, "Absent", Ctx{}); known {
					t.Fatal("direct absence skipped observed source/root replacement")
				}
			case "file-miss":
				if _, known := indexedFileDeclaration(file, "Absent", Ctx{}); known {
					t.Fatal("file absence skipped observed source replacement")
				}
			case "context-free-miss":
				if _, known := indexedQualifiedDeclaration("p.Absent"); known {
					t.Fatal("context-free absence skipped active render source replacement")
				}
			case "returned-package":
				if _, known := indexedPackageDeclaration(symbol.GlobalScope.FindPackage("p"), "Peer", f.ctx); known {
					t.Fatal("package hit skipped returned owner's changed file")
				}
			case "returned-qualified":
				if _, known := indexedQualifiedDeclaration("p.Peer.Child"); known {
					t.Fatal("qualified hit skipped returned owner's changed file")
				}
			}
			*file = original
			requireRetiredDeclaration196(t, facts, f.ctx)
		})
	}
}

type noncomparableDeclarationFormatter196 []byte

func (noncomparableDeclarationFormatter196) Format(*log.Entry) ([]byte, error) { return nil, nil }

func TestNamedDeclaration196SessionConfigurationAndGraphLoss(t *testing.T) {
	for _, mode := range []string{"session", "invocation", "configuration", "active-graph"} {
		t.Run(mode, func(t *testing.T) {
			f, facts := ownedDeclarationFixture196(t, []string{`package p; class Use {}`})
			switch mode {
			case "session":
				f.ctx.syntaxInvocation.session.Close()
			case "invocation":
				f.ctx.syntaxInvocation.close()
			case "configuration":
				old := log.StandardLogger().Formatter
				log.StandardLogger().Formatter = noncomparableDeclarationFormatter196{1}
				defer func() { log.StandardLogger().Formatter = old }()
			case "active-graph":
				activeResolutionFiles.graph = &symbol.GlobalSymbols{}
			}
			if declarationCandidates(f.ctx) != nil {
				t.Fatal("invalid owner/configuration/graph retained authority")
			}
			// Even after a restorable configuration/holder condition is restored,
			// old candidate facts remain dead for the rest of the batch.
			if mode == "configuration" {
				log.StandardLogger().Formatter = facts.formatter
			}
			activeResolutionFiles.graph = symbol.GlobalScope
			requireRetiredDeclaration196(t, facts, f.ctx)
		})
	}
}

func TestNamedDeclaration196UnknownWholeTierFallback(t *testing.T) {
	f := qualifiedLookup195Parsed(t, []string{`package p; class Use {}`, `package p; class Broken {static class Good {}}`})
	broken := f.files[1].Symbols.BaseClass
	broken.Subclasses = append([]*symbol.ClassScope{nil}, broken.Subclasses...)
	session := cliownership.New()
	invocation := newSyntaxInvocation(session, f.files)
	if invocation == nil || invocation.declarations == nil {
		t.Fatal("unknown declaration must not eagerly panic or disable owning invocation")
	}
	defer session.Close()
	defer invocation.close()
	f.ctx.syntaxInvocation = invocation
	f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
	activeResolutionFiles = sourceOwnershipIndex(f.ctx)
	if _, known := indexedDirectDeclaration(broken, "Good", f.ctx); known {
		t.Fatal("unknown owner was indexed")
	}
	if _, known := indexedFileDeclaration(f.files[1].Symbols, "Broken", f.ctx); known {
		t.Fatal("unknown descendant failed to mark whole file unknown")
	}
	if _, known := indexedPackageDeclaration(symbol.GlobalScope.FindPackage("p"), "Use", f.ctx); known {
		t.Fatal("unknown descendant failed to mark whole package unknown")
	}
	if _, known := indexedQualifiedDeclaration("p.Use"); known {
		t.Fatal("aggregate hit skipped unknown package region")
	}
	// Qualified source lookup historically skips nil children; the hierarchy
	// lookup historically dereferences them. Unknown must preserve each path.
	if got := findQualifiedSourceClass("p.Broken.Good"); got != broken.Subclasses[1] {
		t.Fatal("qualified fallback changed its nil-skipping traversal")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("unknown hierarchy fallback skipped original nil-entry panic")
			}
		}()
		memberTypeInHierarchy(broken, "Good", f.ctx)
	}()
	late := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Late"}}
	child := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Child"}, Enclosing: late}
	late.Subclasses = []*symbol.ClassScope{child}
	if _, known := indexedDirectDeclaration(late, "Child", f.ctx); known {
		t.Fatal("late synthetic owner claimed immutable authority")
	}
	if got := memberTypeInHierarchy(late, "Child", f.ctx); got != child {
		t.Fatal("late synthetic owner lost live lookup")
	}
}

func TestNamedDeclaration196DuplicateOrderAndPartialPaths(t *testing.T) {
	f, _ := ownedDeclarationFixture196(t, []string{
		`package p; class Use {static class Same {} static class Same {}} class Top {} class Top {}`,
		`package q; class Left {} class Dup {static class Child {}}`,
		`package q; class Right {} class Dup {}`,
		`package q.Dup; class Other {}`,
	})
	use := f.ctx.currentClass
	if got, known := indexedDirectDeclaration(use, "Same", f.ctx); !known || got != use.Subclasses[0] {
		t.Fatal("same-file member duplicate order changed")
	}
	if got, known := indexedFileDeclaration(f.ctx.currentFile, "Top", f.ctx); !known || got != f.ctx.currentFile.TopLevelClasses[1] {
		t.Fatal("same-file top-level duplicate order changed")
	}
	if _, known := indexedQualifiedDeclaration("q.Dup.Child"); known {
		t.Fatal("cross-file duplicate root preselected a map iteration winner")
	}
	for repeat := 0; repeat < 16; repeat++ {
		got := findQualifiedSourceClass("q.Dup.Child")
		if got != nil && got != f.files[1].Symbols.TopLevelClasses[1].Subclasses[0] {
			t.Fatal("duplicate fallback returned an impossible original candidate")
		}
	}
	if got := findQualifiedSourceClass("q.Dup.Other"); got != f.files[3].Symbols.BaseClass {
		t.Fatal("longest existing package split changed")
	}
	if got := findQualifiedSourceClass("p.Use.Absent.Same"); got != nil {
		t.Fatal("partial member failure restarted package/suffix lookup")
	}
}

func TestNamedDeclaration196DirectMissRetainsLiveParentsAndImports(t *testing.T) {
	f, _ := ownedDeclarationFixture196(t, []string{
		`package p; import a.Base; class Use extends Base {}`,
		`package a; class Base {public static class Member {}}`,
		`package b; class Base {public static class Member {}}`,
	})
	if got := memberTypeInHierarchy(f.ctx.currentClass, "Member", f.ctx); got != f.files[1].Symbols.BaseClass.Subclasses[0] {
		t.Fatal("direct candidate absence skipped inherited member resolution")
	}
	f.ctx.currentFile.Imports["Base"] = "b"
	if got := memberTypeInHierarchy(f.ctx.currentClass, "Member", f.ctx); got != f.files[2].Symbols.BaseClass.Subclasses[0] {
		t.Fatal("declaration table retained parent/import lookup decisions")
	}
}

func TestNamedDeclaration196UnknownObservationRetires(t *testing.T) {
	for _, query := range []string{"owner", "file"} {
		t.Run(query, func(t *testing.T) {
			f := qualifiedLookup195Parsed(t, []string{`package p; class Use {}`, `package p; class Broken {static class Good {}}`})
			file, owner := f.files[1].Symbols, f.files[1].Symbols.BaseClass
			owner.Subclasses = append(owner.Subclasses, nil)
			session := cliownership.New()
			invocation := newSyntaxInvocation(session, f.files)
			if invocation == nil || invocation.declarations == nil {
				t.Fatal("unknown declaration must retain a fallback-capable invocation")
			}
			defer session.Close()
			defer invocation.close()
			f.ctx.syntaxInvocation = invocation
			f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
			activeResolutionFiles = sourceOwnershipIndex(f.ctx)
			original := file.Source
			file.Source = append([]byte(nil), original...)
			file.Source[0] = 'P'
			if query == "owner" {
				if _, known := indexedDirectDeclaration(owner, "Absent", Ctx{}); known {
					t.Fatal("unknown owner unexpectedly supplied indexed absence")
				}
			} else if _, known := indexedFileDeclaration(file, "Absent", Ctx{}); known {
				t.Fatal("unknown file unexpectedly supplied indexed absence")
			}
			file.Source = original
			requireRetiredDeclaration196(t, invocation.declarations, f.ctx)
		})
	}
}

func TestNamedDeclaration196PointerExactRootReplacement(t *testing.T) {
	f, facts := ownedDeclarationFixture196(t, []string{`package p; class Use {static class Member {}}`})
	file := f.files[0].Symbols
	original := *file
	copyRoot := *file.ParsedRoot()
	replacement := symbol.ParseSymbols(&copyRoot, file.Source)
	if replacement.ParsedRoot() == file.ParsedRoot() || !replacement.ParsedRoot().Equal(file.ParsedRoot()) {
		t.Fatal("fixture must replace the root pointer with an equal native node wrapper")
	}
	*file = *replacement
	if _, known := indexedQualifiedDeclaration("p.Use"); known {
		t.Fatal("Node.Equal incorrectly certified a replaced root pointer")
	}
	*file = original
	requireRetiredDeclaration196(t, facts, f.ctx)
}

func TestNamedDeclaration196ConversionRestoresOnErrorAndPanic(t *testing.T) {
	f, facts := ownedDeclarationFixture196(t, []string{`package p; class Use {}`})
	outer := activeResolutionFiles
	public := f.ctx.Clone()
	public.syntaxInvocation = nil
	public.sourceOwnership = outer
	missing := f.files[0]
	missing.Ast = nil
	if _, err := convertFileNode(missing, public); err == nil {
		t.Fatal("missing AST did not return its original compilation-unit error")
	}
	if activeResolutionFiles != outer || declarationCandidates(f.ctx) != facts {
		t.Fatal("conversion error failed to restore outer render authority")
	}
	invalid := f.files[0]
	invalid.Source = nil // original parsed package name access must panic on bounds.
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("invalid public source did not preserve original bounds panic")
			}
		}()
		_, _ = convertFileNode(invalid, public)
	}()
	if activeResolutionFiles != outer || declarationCandidates(f.ctx) != facts || facts.retired {
		t.Fatal("conversion panic failed to restore live outer render authority")
	}
}
