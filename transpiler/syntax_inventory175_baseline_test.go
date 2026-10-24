package transpiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/internal/cliownership"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
)

type syntax175Fixture struct {
	files   []parsing.SourceFile
	ctx     Ctx
	session *cliownership.Session
}

func syntax175Sources(t *testing.T, count, statements int) syntax175Fixture {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	var files []parsing.SourceFile
	for n := 0; n < count; n++ {
		var source strings.Builder
		fmt.Fprintf(&source, "package inventory; class Cell%d { int field%d; ", n, n)
		for method := 0; method < 4; method++ {
			fmt.Fprintf(&source, "public int read%d(int value) { ", method)
			for j := 0; j < statements; j++ {
				source.WriteString("value++; ")
			}
			source.WriteString("return value; } ")
		}
		source.WriteString("class Nested { int nested; } }")
		file := parsing.SourceFile{Name: fmt.Sprintf("Cell%d.java", n), Source: []byte(source.String())}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	session := cliownership.New()
	t.Cleanup(session.Close)
	ctx := Ctx{currentFile: files[0].Symbols, currentClass: files[0].Symbols.BaseClass, callableSubclasses: &callableSubclassSourceInventory{}}
	ctx.syntaxInvocation = newSyntaxInvocation(session, files)
	if ctx.syntaxInvocation == nil {
		t.Fatal("valid owned fixture did not receive invocation")
	}
	return syntax175Fixture{files, ctx, session}
}

func syntax175Names(count int) []string {
	result := make([]string, count)
	for n := range result {
		result[n] = fmt.Sprintf("__syntaxInventoryAbsent%d", n)
	}
	return result
}

func TestSyntaxSession175LifecycleAndProvenance(t *testing.T) {
	f := syntax175Sources(t, 2, 2)
	invocation := f.ctx.syntaxInvocation
	if f.ctx.Clone().syntaxInvocation != invocation {
		t.Fatal("Clone lost owned invocation")
	}
	if newSyntaxInvocation(nil, f.files) != nil {
		t.Fatal("nil capability admitted")
	}
	foreign := parsing.SourceFile{Name: "Foreign.java", Source: []byte("package foreign; class Foreign {}")}
	if err := foreign.ParseAST(); err != nil {
		t.Fatal(err)
	}
	foreign.ParseSymbols()
	symbol.AddSymbolsToPackage(foreign.Symbols)
	if newSyntaxInvocation(f.session, f.files) != nil {
		t.Fatal("late foreign graph admitted")
	}
	invocation.close()
	if invocation.active {
		t.Fatal("retired batch remains active")
	}
	f.session.Close()
	if f.session.Active() {
		t.Fatal("retired process remains active")
	}
}

func TestSyntaxSession175UnsupportedConfigurationFallsBack(t *testing.T) {
	logger := log.StandardLogger()
	oldOut, oldFormatter, oldHooks, oldPool := logger.Out, logger.Formatter, logger.Hooks, logger.BufferPool
	t.Cleanup(func() {
		logger.Out = oldOut
		logger.Formatter = oldFormatter
		logger.Hooks = oldHooks
		logger.BufferPool = oldPool
	})
	logger.Out = os.Stderr
	logger.Formatter = &log.TextFormatter{}
	logger.Hooks = log.LevelHooks{}
	logger.BufferPool = nil
	if !standaloneSyntaxConfigurationOK(os.Stdout) {
		t.Fatal("default CLI configuration excluded")
	}
	if standaloneSyntaxConfigurationOK(&bytes.Buffer{}) {
		t.Fatal("custom conversion writer admitted")
	}
	logger.Out = io.Discard
	if standaloneSyntaxConfigurationOK(os.Stdout) {
		t.Fatal("custom logging writer admitted")
	}
	root := t.TempDir()
	java := filepath.Join(root, "Plain.java")
	if err := os.WriteFile(java, []byte("class Plain { int value; }"), 0600); err != nil {
		t.Fatal(err)
	}
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	session := cliownership.New()
	if err := RunStandalone([]string{"-strict", "-w", "-output", filepath.Join(root, "generated"), java}, session); err != nil {
		t.Fatalf("unsupported configuration rejected valid conversion: %v", err)
	}
	if session.Active() {
		t.Fatal("fallback leaked process session")
	}
	logger.Out = os.Stderr
	logger.Formatter = &log.JSONFormatter{}
	if standaloneSyntaxConfigurationOK(os.Stdout) {
		t.Fatal("nonstandard formatter admitted")
	}
}

func TestSyntaxInventory175ExistingScannerOracle(t *testing.T) {
	f := syntax175Sources(t, 2, 8)
	names := append(syntax175Names(8), "Cell0", "field0", "Field0", "read0", "Read0", "Read0Java2goExecution", "Read0Java2goExactExecution", "nested", "Nested", "")
	for _, name := range names {
		before := syntax175OldSourceIdentifierExists(name, f.ctx)
		after := superclassEmbeddingSourceIdentifierExists(name, f.ctx)
		if before != after {
			t.Fatalf("%q old=%t current=%t", name, before, after)
		}
	}
	// Generated-name collisions remain live even while syntax belongs to a batch.
	for _, file := range f.files {
		file.Symbols.BaseClass.Methods = append(file.Symbols.BaseClass.Methods, &symbol.Definition{Name: "Read0Java2goExecution"})
	}
	for _, name := range []string{"Read0Java2goExecution", "Read0Java2goExecution1", "Read0Java2goExactExecution1"} {
		if syntax175OldSourceIdentifierExists(name, f.ctx) != superclassEmbeddingSourceIdentifierExists(name, f.ctx) {
			t.Fatalf("live collision changed %q", name)
		}
	}
}

type syntax175Work struct {
	Files, Queries int
	Warm           bool
	Allocs         float64
	Bytes          uint64
	Cgo            int64
	Scopes         int
}

func syntax175Measure(t *testing.T, files, queries int, warm bool) syntax175Work {
	t.Helper()
	f := syntax175Sources(t, files, 32)
	names := syntax175Names(queries)
	run := func() {
		ctx := f.ctx
		if !warm {
			ctx.syntaxInvocation = newSyntaxInvocation(f.session, f.files)
			defer ctx.syntaxInvocation.close()
		}
		for _, name := range names {
			if superclassEmbeddingSourceIdentifierExists(name, ctx) {
				t.Fatalf("unexpected fixture collision %q", name)
			}
		}
		runtime.KeepAlive(f.files)
	}
	run() // Wrappers and tree pointer interning warm outside counters.
	allocs := testing.AllocsPerRun(2, run)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cgo := runtime.NumCgoCall()
	for n := 0; n < 3; n++ {
		run()
	}
	cgo = runtime.NumCgoCall() - cgo
	runtime.ReadMemStats(&after)
	scopes := len(symbol.GlobalScope.FindPackage("inventory").Files)
	result := syntax175Work{files, queries, warm, allocs, (after.TotalAlloc - before.TotalAlloc) / 3, cgo / 3, scopes}
	if scopes != files {
		t.Fatalf("source graph lost: %d != %d", scopes, files)
	}
	encoded, _ := json.Marshal(result)
	t.Log("SYNTAX_WORK175", string(encoded))
	return result
}

func TestSyntaxInventory175WorkBaseline(t *testing.T) {
	for _, size := range [][2]int{{2, 4}, {8, 16}, {16, 32}} {
		for _, warm := range []bool{false, true} {
			t.Run(fmt.Sprintf("files%d_queries%d_warm%t", size[0], size[1], warm), func(t *testing.T) { syntax175Measure(t, size[0], size[1], warm) })
		}
	}
}
