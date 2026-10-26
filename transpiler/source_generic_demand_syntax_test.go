package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func demandSyntaxAgree(t *testing.T, ctx Ctx, known bool) []*symbol.ClassScope {
	t.Helper()
	indexed, gotKnown := sourceGenericViewDemandSeedsFromSyntax(ctx)
	if gotKnown != known {
		t.Fatalf("syntax known=%t want%t", gotKnown, known)
	}
	got := sourceGenericViewDemandSeeds(ctx)
	dynamic := ctx.Clone()
	dynamic.syntaxInvocation = nil
	want := sourceGenericViewDemandSeeds(dynamic)
	demand188Same(t, got, want)
	if known {
		demand188Same(t, indexed, want)
		if ctx.syntaxInvocation.demandSyntax == nil {
			t.Fatal("owned index missing")
		}
	}
	return got
}
func demandSyntaxBaseSources(use string) map[string]string {
	return map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "q/Cell.java": `package q;public class Cell<T>{T value;}`, "app/Use.java": use}
}

func TestSourceGenericDemandSyntaxLiveRoutes(t *testing.T) {
	t.Run("live-imports-aliases-and-method-replacement", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, demandSyntaxBaseSources(`package app;import p.Cell;public class Use{Cell<?> field;<Cell>void shadow(Object value){Cell local=(Cell)value;}void local(){class Cell<T>{}Cell<?> local;p.Cell<?> qualified;}Object anon(){return new Object(){q.Cell<?> hidden;};}static class Nested<T>{T value;}record Pair(int value){}}`))
		ctx.importAliases = map[string]string{"kept/path": "kept"}
		ctx.usedImports = map[string]bool{"kept/path": true}
		aliases, used := maps.Clone(ctx.importAliases), maps.Clone(ctx.usedImports)
		want := demandSyntaxAgree(t, ctx, true)
		if len(want) != 1 || want[0] != findQualifiedSourceClass("p.Cell") {
			t.Fatal("wrong source pruning/binder baseline")
		}
		use := findQualifiedSourceClass("app.Use")
		file := findFileScopeForClassScope(use, ctx)
		file.Imports["Cell"] = "q"
		got := demandSyntaxAgree(t, ctx, true)
		if len(got) != 2 {
			t.Fatalf("fresh imported owner not selected: %v", demand188Names(got))
		}
		file.Imports["Cell"] = "p"
		demand188Same(t, demandSyntaxAgree(t, ctx, true), want)
		saved := use.Methods
		use.Methods = append([]*symbol.Definition(nil), saved...)
		for i, m := range use.Methods {
			if m.OriginalName == "shadow" {
				replacement := *m
				replacement.TypeParameters = nil
				use.Methods[i] = &replacement
			}
		}
		demandSyntaxAgree(t, ctx, true)
		for left, right := 0, len(use.Methods)-1; left < right; left, right = left+1, right-1 {
			use.Methods[left], use.Methods[right] = use.Methods[right], use.Methods[left]
		}
		demandSyntaxAgree(t, ctx, true)
		use.Methods = saved
		demandSyntaxAgree(t, ctx, true)
		if !reflect.DeepEqual(ctx.importAliases, aliases) || !reflect.DeepEqual(ctx.usedImports, used) {
			t.Fatal("syntax query changed caller import aliases/usage")
		}
	})
	for _, body := range []string{`<Cell>void shadow(Object value){Cell local=(Cell)value;}`, `<Cell>Use(Object value){Cell local=(Cell)value;}`} {
		t.Run(body[:6], func(t *testing.T) {
			ctx, _, _ := demand188Fixture(t, demandSyntaxBaseSources(`package app;import p.Cell;public class Use{`+body+`}`))
			use := findQualifiedSourceClass("app.Use")
			if len(demandSyntaxAgree(t, ctx, true)) != 0 {
				t.Fatal("method/constructor binder lost")
			}
			saved := use.Methods
			use.Methods = append([]*symbol.Definition(nil), saved...)
			for i, m := range use.Methods {
				if len(m.TypeParameters) > 0 {
					replacement := *m
					replacement.TypeParameters = nil
					use.Methods[i] = &replacement
				}
			}
			got := demandSyntaxAgree(t, ctx, true)
			if len(got) != 1 || got[0] != findQualifiedSourceClass("p.Cell") {
				t.Fatal("replacement definition did not change actual seed")
			}
			use.Methods = saved
			if len(demandSyntaxAgree(t, ctx, true)) != 0 {
				t.Fatal("restored method/constructor binder lost")
			}
		})
	}
	t.Run("live-class-binder-and-header-body", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, demandSyntaxBaseSources(`package app;import p.Cell;public class Use<Cell>{void use(Object value){Cell local=(Cell)value;}}`))
		use := findQualifiedSourceClass("app.Use")
		if len(demandSyntaxAgree(t, ctx, true)) != 0 {
			t.Fatal("class binder lost")
		}
		saved := use.TypeParameters
		use.TypeParameters = nil
		if len(demandSyntaxAgree(t, ctx, true)) != 1 {
			t.Fatal("live class binder replacement ignored")
		}
		use.TypeParameters = saved
		demandSyntaxAgree(t, ctx, true)
	})
}

func TestSourceGenericDemandSyntaxLexicalAndFamilyShields(t *testing.T) {
	for _, use := range []string{
		`package app;public class Use extends p.Cell<String>{static class p{static class Cell<T>{T value;}}p.Cell<?> body;}`,
		`package app;import p.Cell;public class Use{void local(){class Cell<T>{}Cell<?> local;p.Cell<?> qualified;}Object make(){return new Object(){q.Cell<?> hidden;};}}`,
		`package app;import p.Cell;public class Use{<p>void shadow(){p.Cell<?> hidden;}q.Cell<?> field;}`,
		`package app;import p.Cell;public class Use{Cell raw;Cell<?> wildcard;Cell<String> concrete;Object make(){return new Cell<>();}Object cast(Object value){return (Cell<String>[])value;}}`,
		`package app;public record Use(p.Cell<?> component){static class Nested<T>{T value;}}`,
	} {
		t.Run(fmt.Sprint(len(use)), func(t *testing.T) {
			ctx, _, _ := demand188Fixture(t, demandSyntaxBaseSources(use))
			want := demandSyntaxAgree(t, ctx, true)
			for _, owner := range allSourceClassScopes() {
				clone := ctx.Clone()
				clone.currentFile = findFileScopeForClassScope(owner, ctx)
				clone.currentClass = owner
				demand188Same(t, demandSyntaxAgree(t, clone, true), want)
			}
		})
	}
	t.Run("family-admission-stays-fresh", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "q/Cell.java": `package q;public class Cell<T>{java.util.List<T> unsupported;}`, "app/Use.java": `package app;public class Use{p.Cell<?> accepted;q.Cell<?> rejected;}`})
		if len(demandSyntaxAgree(t, ctx, true)) != 2 {
			t.Fatal("seed inventory incorrectly supplied admission authority")
		}
		if plan, err := planGenericFamily(findQualifiedSourceClass("p.Cell"), ctx); err != nil || plan == nil {
			t.Fatal("valid family rejected")
		}
		if plan, err := planGenericFamily(findQualifiedSourceClass("q.Cell"), ctx); err == nil || plan != nil {
			t.Fatal("unsupported family admitted from syntax seed")
		}
	})
}

func TestSourceGenericDemandSyntaxOwnershipAndRetirement(t *testing.T) {
	ctx, files, session := demand188Fixture(t, demandSyntaxBaseSources(`package app;public class Use{p.Cell<?> value;}`))
	baseline := demandSyntaxAgree(t, ctx, true)
	inv := ctx.syntaxInvocation
	use := findQualifiedSourceClass("app.Use")
	file := findFileScopeForClassScope(use, ctx)
	at := bytes.Index(file.Source, []byte("p.Cell<?>"))
	if at < 0 {
		t.Fatal("missing source demand")
	}
	file.Source[at] = 'q'
	changed := demandSyntaxAgree(t, ctx, false)
	if len(changed) != 1 || changed[0] != findQualifiedSourceClass("q.Cell") {
		t.Fatal("fallback ignored live source bytes")
	}
	file.Source[at] = 'p'
	demand188Same(t, demandSyntaxAgree(t, ctx, true), baseline)
	source := file.Source
	file.Source = append([]byte(nil), source...)
	demandSyntaxAgree(t, ctx, true)
	file.Source = source
	twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), source...)}
	if err := twin.ParseAST(); err != nil {
		t.Fatal(err)
	}
	twin.ParseSymbols()
	root := use.Class.DeclarationNode
	use.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
	demandSyntaxAgree(t, ctx, false)
	use.Class.DeclarationNode = root
	demandSyntaxAgree(t, ctx, true)
	savedClasses := file.TopLevelClasses
	file.TopLevelClasses = nil
	demandSyntaxAgree(t, ctx, false)
	file.TopLevelClasses = savedClasses
	demandSyntaxAgree(t, ctx, true)
	logger := log.StandardLogger()
	output := logger.Out
	logger.Out = io.Discard
	demandSyntaxAgree(t, ctx, false)
	logger.Out = output
	other := newSyntaxInvocation(session, files)
	if other == nil {
		t.Fatal("twin owned invocation rejected")
	}
	other.demandSyntax = inv.demandSyntax
	clone := ctx.Clone()
	clone.syntaxInvocation = other
	demandSyntaxAgree(t, clone, false)
	other.close()
	graph := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	demandSyntaxAgree(t, ctx, false)
	symbol.GlobalScope = graph
	demandSyntaxAgree(t, ctx, true)
	late := parsing.SourceFile{Name: "Late.java", Source: []byte(`package late;class Late{q.Cell<?> field;}`)}
	if err := late.ParseAST(); err != nil {
		t.Fatal(err)
	}
	late.ParseSymbols()
	symbol.AddSymbolsToPackage(late.Symbols)
	demandSyntaxAgree(t, ctx, false)
	delete(symbol.GlobalScope.Packages, "late")
	demandSyntaxAgree(t, ctx, true)
	inv.close()
	if inv.demandSyntax != nil || inv.active {
		t.Fatal("close retained live syntax routes")
	}
	demandSyntaxAgree(t, ctx, false)
	session.Close()
	if session.Active() {
		t.Fatal("session still active")
	}
	public := ctx.Clone()
	public.syntaxInvocation = nil
	demandSyntaxAgree(t, public, false)
	if logger.Out != os.Stderr {
		t.Fatal("diagnostic failed to restore logger")
	}
}

func TestSourceGenericDemandSyntaxEditedTreeAndBounds(t *testing.T) {
	t.Run("edited-tree", func(t *testing.T) {
		previous := symbol.GlobalScope
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		t.Cleanup(func() { symbol.GlobalScope = previous })
		source := []byte(`class Cell<T>{T value;}class Use{Cell<?> value;}`)
		parser := sitter.NewParser()
		t.Cleanup(parser.Close)
		parser.SetLanguage(java.GetLanguage())
		tree, err := parser.ParseCtx(context.Background(), nil, source)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(tree.Close)
		file := parsing.SourceFile{Name: "Edited.java", Source: source, Ast: tree.RootNode()}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		ResolveFile(file)
		session := cliownership.New()
		t.Cleanup(session.Close)
		ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}}
		ctx.syntaxInvocation = newSyntaxInvocation(session, []parsing.SourceFile{file})
		if ctx.syntaxInvocation == nil {
			t.Fatal("real tree ownership rejected")
		}
		t.Cleanup(ctx.syntaxInvocation.close)
		demandSyntaxAgree(t, ctx, true)
		at := uint32(bytes.Index(source, []byte("value")))
		tree.Edit(sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}})
		if !file.Symbols.ParsedRoot().HasChanges() {
			t.Fatal("actual Tree.Edit did not change parse anchor")
		}
		demandSyntaxAgree(t, ctx, false)
	})
	t.Run("dynamic-bounds-panic", func(t *testing.T) {
		ctx, _, _ := demand188Fixture(t, demandSyntaxBaseSources(`package app;public class Use{p.Cell<?> value;}`))
		demandSyntaxAgree(t, ctx, true)
		file := findFileScopeForClassScope(findQualifiedSourceClass("app.Use"), ctx)
		saved := file.Source
		file.Source = nil
		invoke := func(query Ctx) (value string) {
			defer func() {
				if e := recover(); e != nil {
					value = fmt.Sprintf("%T:%v", e, e)
				}
			}()
			sourceGenericViewDemandSeeds(query)
			return "no-panic"
		}
		dynamic := ctx.Clone()
		dynamic.syntaxInvocation = nil
		want := invoke(dynamic)
		got := invoke(ctx)
		file.Source = saved
		if got != want {
			t.Fatalf("dynamic panic changed: %q/%q", want, got)
		}
	})
}
