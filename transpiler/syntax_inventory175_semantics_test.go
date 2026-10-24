package transpiler

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

func syntax175TextFixture(t *testing.T, source string) syntax175Fixture {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	file := parsing.SourceFile{Name: "Fixture.java", Source: []byte(source)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	file.ParseSymbols()
	symbol.AddSymbolsToPackage(file.Symbols)
	session := cliownership.New()
	t.Cleanup(session.Close)
	files := []parsing.SourceFile{file}
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}}
	ctx.syntaxInvocation = newSyntaxInvocation(session, files)
	if ctx.syntaxInvocation == nil {
		t.Fatal("owned parse provenance rejected")
	}
	return syntax175Fixture{files, ctx, session}
}

func syntax175Agree(t *testing.T, ctx Ctx, names []string) {
	t.Helper()
	for _, name := range names {
		old := syntax175OldSourceIdentifierExists(name, ctx)
		got := superclassEmbeddingSourceIdentifierExists(name, ctx)
		if old != got {
			t.Fatalf("%q old=%t current=%t", name, old, got)
		}
	}
}

func TestSyntaxInventory175OrderedEventsAndAllBodies(t *testing.T) {
	source := `package inventory;
 @interface Mark { String value() default "lambda annotation"; }
 @Mark("Unicode: Δ") class Fixture<T> {
  int type; public <U> U echo(U input) { return input; }
  public int[] pack(int... values) { return values; }
  private int dollar$name(int type) { return type; }
  void use() { class Local { public void local(int value) {} }
   Object object = new Object() { public void anonymous(int value) {} };
   java.util.function.IntUnaryOperator lambda = n -> n + 1;
  }
  class Nested { int member; }
  enum Choice { A { public void action(int value) {} }; public void action(int value) {} }
  record Pair(int component) { public int read(int value) { return component; } }
 }`
	f := syntax175TextFixture(t, source)
	if f.files[0].Ast.HasError() {
		t.Fatalf("valid fixture grammar error: %s", f.files[0].Ast.String())
	}
	bodies, ready := f.ctx.syntaxInvocation.queryBodies("unrelated")
	if !ready || len(bodies) != 2 {
		t.Fatalf("body inventory unavailable: %t %d", ready, len(bodies))
	}
	names := []string{"inventory", "Mark", "Fixture", "Local", "local", "LocalJava2goExecution", "anonymous", "AnonymousJava2goExecution", "n", "member", "component", "type", "Type_", "Type", "dollar$name", "_", "", "doesNotExist"}
	for _, entry := range f.ctx.syntaxInvocation.identifiers {
		for name := range entry.raw {
			names = append(names, name)
		}
		for name := range entry.exactMethods {
			names = append(names, name)
		}
		for prefix := range entry.prefixMethods {
			for _, suffix := range []string{"", "1", "2", "_", "random"} {
				names = append(names, prefix+suffix)
			}
		}
	}
	sort.Strings(names)
	beforeSource := append([]byte(nil), f.files[0].Source...)
	beforeTree := f.files[0].Ast.String()
	syntax175Agree(t, f.ctx, names)
	if !bytes.Equal(beforeSource, f.files[0].Source) || beforeTree != f.files[0].Ast.String() {
		t.Fatal("syntax/oracle query mutated inputs")
	}
	if superclassEmbeddingSourceIdentifierExists("inventory", f.ctx) {
		t.Fatal("program-only package token leaked outside registered bodies")
	}
}

func TestSyntaxInventory175FreshQueryMutationsAndFallback(t *testing.T) {
	f := syntax175Sources(t, 2, 4)
	inv := f.ctx.syntaxInvocation
	if _, ok := inv.queryBodies("absent"); !ok {
		t.Fatal("fresh inventory not usable")
	}
	// Same-length body edits must be detected from bytes, never pointer/len.
	source := f.files[0].Symbols.Source
	at := bytes.Index(source, []byte("read0"))
	if at < 0 {
		t.Fatal("missing source token")
	}
	copy(source[at:at+5], "read9")
	if _, ok := inv.queryBodies("read9"); ok {
		t.Fatal("same-length source edit admitted stale index")
	}
	syntax175Agree(t, f.ctx, []string{"read0", "read9", "Read9Java2goExecution"})
	copy(source[at:at+5], "read0")
	if _, ok := inv.queryBodies("absent"); !ok {
		t.Fatal("restored identical source not usable")
	}
	owner := f.files[0].Symbols.BaseClass
	oldMethods := owner.Methods
	for n := 0; n < 4; n++ {
		name := "Read0Java2goExecution"
		if n > 0 {
			name += fmt.Sprint(n)
		}
		owner.Methods = append(owner.Methods, &symbol.Definition{Name: name})
	}
	owner.Class.TypeParameters = append(owner.Class.TypeParameters, symbol.NewTypeParam("read0", nil))
	f.files[0].Symbols.Imports["Cell"] = "another.owner"
	syntax175Agree(t, f.ctx, []string{"Read0Java2goExecution", "Read0Java2goExecution4", "Read0Java2goExactExecution", "Cell", "read0"})
	owner.Methods = oldMethods
	// A twin parse at identical byte offsets is not the original body identity.
	twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), source...)}
	if err := twin.ParseAST(); err != nil {
		t.Fatal(err)
	}
	twin.ParseSymbols()
	saved := owner.Class.DeclarationNode
	owner.Class.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
	if _, ok := inv.queryBodies("read0"); ok {
		t.Fatal("rebound detached body admitted")
	}
	syntax175Agree(t, f.ctx, []string{"read0", "nested"})
	owner.Class.DeclarationNode = saved
	late := parsing.SourceFile{Name: "Late.java", Source: []byte("package late; class Late { public void late(int input) {} }")}
	if err := late.ParseAST(); err != nil {
		t.Fatal(err)
	}
	late.ParseSymbols()
	symbol.AddSymbolsToPackage(late.Symbols)
	if _, ok := inv.queryBodies("read0"); ok {
		t.Fatal("late root did not prevalidate entire query before an early raw hit")
	}
	syntax175Agree(t, f.ctx, []string{"Late", "late", "LateJava2goExecution", "read0"})
	graph := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	if _, ok := inv.queryBodies("absent"); ok {
		t.Fatal("replaced graph admitted")
	}
	syntax175Agree(t, f.ctx, []string{"read0", "Late"})
	symbol.GlobalScope = graph
	clone := f.ctx.Clone()
	inv.close()
	if _, ok := clone.syntaxInvocation.queryBodies("read0"); ok {
		t.Fatal("retained Clone outlived batch")
	}
	syntax175Agree(t, clone, []string{"read0", "Late"})
}

func TestSyntaxInventory175TreeAndMalformedShields(t *testing.T) {
	for _, source := range []string{"class Broken { void method( {", "class Δ { int value; }"} {
		t.Run(fmt.Sprint(len(source)), func(t *testing.T) {
			f := syntax175TextFixture(t, source)
			syntax175Agree(t, f.ctx, []string{"value", "Broken", "MethodJava2goExecution", "absent"})
			if f.files[0].Ast.HasError() {
				if _, ok := f.ctx.syntaxInvocation.queryBodies("absent"); ok {
					t.Fatal("parser ERROR admitted")
				}
			}
		})
	}
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	source := []byte("class Edit { int value; }")
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(context.Background(), nil, source)
	if err != nil {
		t.Fatal(err)
	}
	root := tree.RootNode()
	symbols := symbol.ParseSymbols(root, source)
	symbol.AddSymbolsToPackage(symbols)
	session := cliownership.New()
	defer session.Close()
	file := parsing.SourceFile{Name: "Edit.java", Source: source, Ast: root, Symbols: symbols}
	ctx := Ctx{currentFile: symbols, currentClass: symbols.BaseClass}
	ctx.syntaxInvocation = newSyntaxInvocation(session, []parsing.SourceFile{file})
	if ctx.syntaxInvocation == nil {
		t.Fatal("tree fixture provenance rejected")
	}
	at := uint32(bytes.Index(source, []byte("value")))
	edit := sitter.EditInput{StartIndex: at, OldEndIndex: at + 1, NewEndIndex: at + 1, StartPoint: sitter.Point{Column: at}, OldEndPoint: sitter.Point{Column: at + 1}, NewEndPoint: sitter.Point{Column: at + 1}}
	tree.Edit(edit)
	if _, ok := ctx.syntaxInvocation.queryBodies("value"); ok {
		t.Fatal("edited tree admitted")
	}
	syntax175Agree(t, ctx, []string{"value", "Edit", "absent"})
	ctx.syntaxInvocation.close()
	root.Edit(edit)
	syntax175Agree(t, ctx, []string{"value", "Edit", "absent"})
}

func syntax175Output(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSyntaxInventory175StandaloneAndMavenBytes(t *testing.T) {
	withCleanDiagnostics(t)
	root := t.TempDir()
	source := filepath.Join(root, "src")
	if err := writeProjectFile(filepath.Join(source, "Main.java"), []byte("package app; interface Named { int value(); } class Base { public int value() { return 1; } } class Main extends Base implements Named { public static void main(String[] args) {} }")); err != nil {
		t.Fatal(err)
	}
	oldGraph := symbol.GlobalScope
	t.Cleanup(func() { symbol.GlobalScope = oldGraph })
	outputs := []map[string]string{}
	for _, owned := range []bool{false, true} {
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		out := filepath.Join(root, fmt.Sprint(owned))
		args := []string{"-strict", "-sync", "-w", "-output", out, source}
		var err error
		if owned {
			session := cliownership.New()
			err = RunStandalone(args, session)
			if session.Active() {
				t.Fatal("standalone leaked session")
			}
		} else {
			err = Run(args)
		}
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, syntax175Output(t, out))
	}
	if !reflect.DeepEqual(outputs[0], outputs[1]) {
		t.Fatal("standalone emitted bytes differ from public fallback")
	}
	project := filepath.Join(root, "project")
	if err := writeProjectFile(filepath.Join(project, "pom.xml"), []byte("<project><groupId>app</groupId><artifactId>syntax</artifactId><version>1</version></project>")); err != nil {
		t.Fatal(err)
	}
	if err := writeProjectFile(filepath.Join(project, "src/main/java/app/Main.java"), []byte("package app; public class Main { public static void main(String[] args) {} }")); err != nil {
		t.Fatal(err)
	}
	runtimeRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	outputs = nil
	for _, owned := range []bool{false, true} {
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		before := symbol.GlobalScope
		out := filepath.Join(root, "maven"+fmt.Sprint(owned))
		args := []string{"-maven", project, "-main-class", "app.Main", "-runtime", runtimeRoot, "-module", "syntax.test/module", "-output", out}
		if owned {
			session := cliownership.New()
			err = RunStandalone(args, session)
			if session.Active() {
				t.Fatal("Maven leaked outer capability")
			}
		} else {
			err = Run(args)
		}
		if err != nil {
			t.Fatal(err)
		}
		if symbol.GlobalScope != before {
			t.Fatal("Maven failed graph restoration")
		}
		outputs = append(outputs, syntax175Output(t, out))
	}
	if !reflect.DeepEqual(outputs[0], outputs[1]) {
		t.Fatal("owned Maven bytes differ")
	}
}

func TestSyntaxInventory175CustomExitAndStrictErrorRetire(t *testing.T) {
	withCleanDiagnostics(t)
	logger := log.StandardLogger()
	exit := logger.ExitFunc
	t.Cleanup(func() { logger.ExitFunc = exit })
	logger.ExitFunc = func(int) {}
	if standaloneSyntaxConfigurationOK(os.Stdout) {
		t.Fatal("custom exit callback admitted")
	}
	logger.ExitFunc = exit
	previous := symbol.GlobalScope
	t.Cleanup(func() { symbol.GlobalScope = previous })
	fixture := filepath.Join("testdata", "strict_badexpr101", "unsupported")
	var oldErr error
	var oldDiags []Diagnostic
	for _, owned := range []bool{false, true} {
		symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
		args := []string{"-strict", "-sync", "-w", "-output", t.TempDir(), fixture}
		var err error
		if owned {
			session := cliownership.New()
			err = RunStandalone(args, session)
			if session.Active() {
				t.Fatal("strict error leaked capability")
			}
		} else {
			err = Run(args)
		}
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("existing strict fixture must fail: %v", err)
		}
		if !owned {
			oldErr = err
			oldDiags = append([]Diagnostic(nil), Diagnostics()...)
		} else if oldErr.Error() != err.Error() || !reflect.DeepEqual(oldDiags, Diagnostics()) {
			t.Fatalf("strict diagnostics changed: old=%v/%v new=%v/%v", oldErr, oldDiags, err, Diagnostics())
		}
	}
}

type syntax175PanicHook struct{}

func (syntax175PanicHook) Levels() []log.Level   { return []log.Level{log.InfoLevel} }
func (syntax175PanicHook) Fire(*log.Entry) error { panic("syntax175 hook panic") }
func TestSyntaxSession175CallbackPanicRetiresAndUnsealedStaysDynamic(t *testing.T) {
	f := syntax175Sources(t, 1, 2)
	logger := log.StandardLogger()
	hooks, level := logger.Hooks, logger.Level
	t.Cleanup(func() { logger.Hooks = hooks; logger.SetLevel(level) })
	logger.Hooks = log.LevelHooks{}
	logger.AddHook(syntax175PanicHook{})
	logger.SetLevel(log.InfoLevel)
	if _, ok := f.ctx.syntaxInvocation.queryBodies("read0"); ok {
		t.Fatal("callback configuration retained owned query")
	}
	invoke := func(owned bool) (observed any, session *cliownership.Session) {
		defer func() { observed = recover() }()
		session = cliownership.New()
		if owned {
			_ = RunStandalone([]string{"-q"}, session)
		} else {
			_ = Run([]string{"-q"})
		}
		return
	}
	old, _ := invoke(false)
	got, session := invoke(true)
	if fmt.Sprint(old) != "syntax175 hook panic" || fmt.Sprint(got) != fmt.Sprint(old) || session.Active() {
		t.Fatalf("callback panic semantics/lifetime changed: old=%v new=%v active=%t", old, got, session.Active())
	}
	logger.Hooks = hooks
	logger.SetLevel(level)
	file := f.files[0].Symbols
	source := file.Source
	file.Source = nil
	capture := func(old bool) (observed any) {
		defer func() { observed = recover() }()
		if old {
			_ = syntax175OldSourceIdentifierExists("read0", f.ctx)
		} else {
			_ = superclassEmbeddingSourceIdentifierExists("read0", f.ctx)
		}
		return
	}
	before, after := capture(true), capture(false)
	if before == nil || fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("source-bound panic changed: %v / %v", before, after)
	}
	file.Source = source
	unsealed := f.ctx.Clone()
	unsealed.syntaxInvocation = nil
	at := bytes.Index(source, []byte("read0"))
	copy(source[at:at+5], "read9")
	syntax175Agree(t, unsealed, []string{"read0", "read9", "Read9Java2goExecution"})
}
