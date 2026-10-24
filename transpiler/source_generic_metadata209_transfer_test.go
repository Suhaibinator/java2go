package transpiler

import (
	"bytes"
	"context"
	"maps"
	"reflect"
	"runtime"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func metadata209DirectTransfer(t *testing.T, f nonDeclaration209Fixture, want bool) {
	t.Helper()
	declaring, found := genericMethodOwnerContext(f.method)
	if !found {
		t.Fatal("fresh declaring context absent")
	}
	got := genericMethodDeclaringSyntaxContext(declaring, f.ctx)
	expected := declaring
	if want {
		expected.syntaxInvocation = f.ctx.syntaxInvocation
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatal("transfer changed fields beyond the admitted invocation, or failed refusal")
	}
	metadata209Compare(t, f)
	runtime.KeepAlive(f)
}

func TestMetadata209DirectCertificateAndIsolation(t *testing.T) {
	for _, metadata := range []string{"package p;", "@Marker package p;import meta.Marker;import q.Cell;"} {
		t.Run(metadata, func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, metadata)
			if sourceGenericDemandOwnedFiles(f.ctx.syntaxInvocation) || !sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
				t.Fatal("new syntax proof or unchanged legacy refusal absent")
			}
			metadata209DirectTransfer(t, f, true)
			// A filename is not a premise; actual roots, ownership and symbol closure are.
			inv := f.ctx.syntaxInvocation
			meta := f.files[len(f.files)-1].Symbols
			owned := inv.files[meta]
			owned.Name = "arbitrary/Ordinary.java"
			inv.files[meta] = owned
			if !sourceGenericDemandOwnedCompilationUnits(inv) {
				t.Fatal("certificate depended on a filename")
			}
			before := f.ctx
			f.ctx.callableSubclasses = &callableSubclassSourceInventory{}
			f.ctx.sourceOwnership = &resolutionFileIndex{graph: symbol.GlobalScope, files: map[*symbol.ClassScope]*symbol.FileScope{f.owner: meta}}
			f.ctx.expectedType = "q.Cell<S>"
			f.ctx.importAliases = map[string]string{"owned/path": "owned"}
			f.ctx.usedImports = map[string]bool{"owned/path": true}
			aliases, used := maps.Clone(f.ctx.importAliases), maps.Clone(f.ctx.usedImports)
			f.ctx.localScope = f.method
			f.ctx.syntheticTypeParameters = f.method.TypeParameters
			metadata209DirectTransfer(t, f, true)
			if !reflect.DeepEqual(aliases, f.ctx.importAliases) || !reflect.DeepEqual(used, f.ctx.usedImports) {
				t.Fatal("caller import ownership mutated")
			}
			if f.ctx.syntaxInvocation != before.syntaxInvocation || f.ctx.callableSubclasses == nil || len(f.ctx.sourceOwnership.files) != 1 {
				t.Fatal("caller holders were replaced")
			}
			runtime.KeepAlive(f)
		})
	}
}

func TestMetadata209DirectSymbolClosure(t *testing.T) {
	for _, mode := range []string{"base-only", "top-level-only", "both"} {
		t.Run(mode, func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, "@Marker package p;import meta.Marker;")
			metadata209DirectTransfer(t, f, true)
			meta := f.files[len(f.files)-1].Symbols
			base, tops := meta.BaseClass, meta.TopLevelClasses
			extra := f.files[4].Symbols.BaseClass
			defer func() { meta.BaseClass, meta.TopLevelClasses = base, tops }()
			if mode != "top-level-only" {
				meta.BaseClass = extra
			}
			if mode != "base-only" {
				meta.TopLevelClasses = []*symbol.ClassScope{extra}
			}
			if sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
				t.Fatal("symbol-only extra class admitted")
			}
			metadata209DirectTransfer(t, f, false)
			meta.BaseClass, meta.TopLevelClasses = base, tops
			metadata209DirectTransfer(t, f, true)
			runtime.KeepAlive(f)
		})
	}
}

func TestMetadata209DirectOwnershipAndForeignRoots(t *testing.T) {
	for _, active := range []bool{false, true} {
		name := "fresh-topology"
		if active {
			name = "stale-active-index"
		}
		t.Run(name, func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, "package p;")
			metadata209DirectTransfer(t, f, true)
			sourceOwnershipIndex(f.ctx)
			previousActive := activeResolutionFiles
			if active {
				activeResolutionFiles = newResolutionFileIndex()
			}
			defer func() { activeResolutionFiles = previousActive }()
			left, right := f.files[4].Symbols, f.files[3].Symbols
			oldLeft, oldRight := left.TopLevelClasses, right.TopLevelClasses
			defer func() { left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight }()
			left.TopLevelClasses = nil
			right.TopLevelClasses = append(append([]*symbol.ClassScope(nil), oldRight...), oldLeft[0])
			// Root ownership cannot be certified by the warmed caller or active map.
			metadata209DirectTransfer(t, f, false)
			left.TopLevelClasses, right.TopLevelClasses = oldLeft, oldRight
			metadata209DirectTransfer(t, f, true)
			runtime.KeepAlive(f)
		})
	}
	t.Run("nondeclaring-foreign-root", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		metadata209DirectTransfer(t, f, true)
		twin, tree := resolveImport205Parse(t, "TwinNoise.java", string(f.files[4].Source))
		f.trees = append(f.trees, tree)
		owner := f.files[4].Symbols.BaseClass
		root := owner.Class.DeclarationNode
		defer func() { owner.Class.DeclarationNode = root }()
		owner.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
		metadata209DirectTransfer(t, f, false)
		owner.Class.DeclarationNode = root
		metadata209DirectTransfer(t, f, true)
		runtime.KeepAlive(f)
	})
	t.Run("foreign-file-and-duplicate-membership", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		metadata209DirectTransfer(t, f, true)
		extra, tree := resolveImport205Parse(t, "foreign/Added.java", "package foreign;class Added{q.Cell<?> demanded;}")
		f.trees = append(f.trees, tree)
		symbol.AddSymbolsToPackage(extra.Symbols)
		if sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
			t.Fatal("foreign file admitted")
		}
		metadata209DirectTransfer(t, f, false)
		delete(symbol.GlobalScope.Packages, "foreign")
		pkg := symbol.GlobalScope.Packages["noise"]
		pkg.Files["duplicate"] = f.files[4].Symbols
		if sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
			t.Fatal("duplicate file admitted")
		}
		metadata209DirectTransfer(t, f, false)
		delete(pkg.Files, "duplicate")
		metadata209DirectTransfer(t, f, true)
		runtime.KeepAlive(f)
	})
}

func TestMetadata209DirectDemandAndDeclarationRefusals(t *testing.T) {
	for _, text := range []string{
		"@meta.Marker(q.Cell.class) package p;import q.Cell;",
		"@meta.Marker(1) package p;",
		"package p;class Added{q.Cell<?> demanded;}",
		"package p;import static java.lang.Integer.*;",
	} {
		t.Run(text, func(t *testing.T) {
			resolveImport205State(t)
			f := metadata209BuildMode(t, 4, false, "package p;")
			// Keep annotation argument witnesses valid against a matching live provider.
			markerText := "package meta;public @interface Marker{int value();}"
			if bytes.Contains([]byte(text), []byte("class)")) {
				markerText = "package meta;public @interface Marker{Class<?> value();}"
			}
			marker, markerTree := metadata209ParseUnit(t, "meta/Marker.java", markerText)
			f.trees = append(f.trees, markerTree)
			symbol.GlobalScope.Packages["meta"].Files["Marker"] = marker.Symbols
			f.files[len(f.files)-2] = marker
			replacement, tree := metadata209ParseUnit(t, "Any.java", text)
			f.trees = append(f.trees, tree)
			f.files[len(f.files)-1] = replacement
			fresh := newSyntaxInvocation(f.session, f.files)
			if fresh == nil {
				t.Fatal("valid owned current-root setup absent")
			}
			defer fresh.close()
			f.ctx.syntaxInvocation = fresh
			if sourceGenericDemandOwnedCompilationUnits(fresh) {
				t.Fatal("demand/declaration syntax admitted")
			}
			metadata209DirectTransfer(t, f, false)
			runtime.KeepAlive(f)
		})
	}
	t.Run("unknown-error-and-missing-root", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		inv := f.ctx.syntaxInvocation
		meta := f.files[len(f.files)-1].Symbols
		owned := inv.files[meta]
		missing := owned
		missing.Ast = nil
		inv.files[meta] = missing
		if sourceGenericDemandOwnedCompilationUnits(inv) {
			t.Fatal("missing owned anchor admitted")
		}
		metadata209DirectTransfer(t, f, false)
		inv.files[meta] = owned
		if sourceGenericDemandInertCompilationUnit(f.owner.Class.DeclarationNode) {
			t.Fatal("non-program declaration root admitted")
		}
		metadata209DirectTransfer(t, f, true)
		runtime.KeepAlive(f)
	})
}

func TestMetadata209DirectSourceAndTreeMutation(t *testing.T) {
	t.Run("bytes-and-source-copy", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@Marker package p;import meta.Marker;import q.Cell;")
		metadata209DirectTransfer(t, f, true)
		meta := f.files[len(f.files)-1].Symbols
		at := bytes.Index(meta.Source, []byte("q.Cell"))
		if at < 0 {
			t.Fatal("source witness absent")
		}
		meta.Source[at] = 'p'
		if sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
			t.Fatal("in-place metadata edit admitted")
		}
		metadata209DirectTransfer(t, f, false)
		meta.Source[at] = 'q'
		original := meta.Source
		meta.Source = append([]byte(nil), original...)
		metadata209DirectTransfer(t, f, true)
		meta.Source = original
		runtime.KeepAlive(f)
	})
	t.Run("tree-edit", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;import q.Cell;")
		metadata209DirectTransfer(t, f, true)
		meta := f.files[len(f.files)-1]
		at := uint32(bytes.Index(meta.Source, []byte("q.Cell")))
		f.trees[len(f.trees)-1].Edit(sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}})
		if !meta.Ast.HasChanges() || sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
			t.Fatal("real tree edit not refused")
		}
		metadata209DirectTransfer(t, f, false)
		runtime.KeepAlive(f)
	})
}

func TestMetadata209DirectInvocationLifetime(t *testing.T) {
	t.Run("closed-nested-fresh-and-panic", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "@Marker package p;import meta.Marker;")
		metadata209DirectTransfer(t, f, true)
		outer := f.ctx.syntaxInvocation
		nested := newSyntaxInvocation(f.session, f.files)
		if nested == nil {
			t.Fatal("nested invocation absent")
		}
		func() {
			defer func() {
				if recover() != "metadata209-panic" {
					t.Fatal("panic witness changed")
				}
			}()
			defer nested.close()
			f.ctx.syntaxInvocation = nested
			metadata209DirectTransfer(t, f, true)
			panic("metadata209-panic")
		}()
		if nested.active || nested.demandSyntax != nil {
			t.Fatal("panic retained nested authority")
		}
		metadata209DirectTransfer(t, f, false)
		f.ctx.syntaxInvocation = outer
		metadata209DirectTransfer(t, f, true)
		outer.close()
		metadata209DirectTransfer(t, f, false)
		replacement, tree := metadata209ParseUnit(t, "Current.java", "@Marker package p;import meta.Marker;import p.Cell;")
		f.trees = append(f.trees, tree)
		f.files[len(f.files)-1] = replacement
		fresh := newSyntaxInvocation(f.session, f.files)
		if fresh == nil {
			t.Fatal("fresh same-graph reparse absent")
		}
		defer fresh.close()
		f.ctx.syntaxInvocation = fresh
		// A caller holder may be ready and old; it must never cross to declaring state.
		metadata209DirectTransfer(t, f, true)
		if outer.active || outer.demandSyntax != nil {
			t.Fatal("retired invocation revived")
		}
		f.session.Close()
		metadata209DirectTransfer(t, f, false)
		runtime.KeepAlive(f)
	})
	t.Run("graph-and-public-context", func(t *testing.T) {
		resolveImport205State(t)
		f := metadata209BuildMode(t, 4, false, "package p;")
		metadata209DirectTransfer(t, f, true)
		saved := symbol.GlobalScope
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: saved.Packages}
		if sourceGenericDemandOwnedCompilationUnits(f.ctx.syntaxInvocation) {
			t.Fatal("foreign graph admitted")
		}
		metadata209DirectTransfer(t, f, false)
		symbol.GlobalScope = saved
		f.ctx.syntaxInvocation = nil
		metadata209DirectTransfer(t, f, false)
		runtime.KeepAlive(f)
	})
}

func TestMetadata209DirectSyntaxErrors(t *testing.T) {
	for _, text := range []string{"package p; @", "package ;", "package p; class", "package p; import ;"} {
		t.Run(text, func(t *testing.T) {
			parser := sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(java.GetLanguage())
			tree, err := parser.ParseCtx(context.Background(), nil, []byte(text))
			if err != nil {
				t.Fatal(err)
			}
			defer tree.Close()
			root := tree.RootNode()
			if !root.HasError() {
				t.Fatal("parse-error witness absent")
			}
			if sourceGenericDemandInertCompilationUnit(root) {
				t.Fatal("error/missing syntax certified")
			}
			runtime.KeepAlive(tree)
		})
	}
}
