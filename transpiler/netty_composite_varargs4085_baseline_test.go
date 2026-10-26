package transpiler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func composite4085AssertCall(t *testing.T, file parsing.SourceFile, className, caller, callee, parameter string) {
	t.Helper()
	owner := file.Symbols.FindClassScope(className)
	if owner == nil {
		t.Fatal("owner missing")
	}
	var method *symbol.Definition
	for _, m := range owner.Methods {
		if m.OriginalName == caller && len(m.Parameters) == 2 && m.Parameters[0].OriginalType == "boolean" && executionParameterIsVariadic(m, 1) {
			method = m
		}
	}
	if method == nil {
		t.Fatal("spread caller missing")
	}
	ctx := Ctx{currentFile: file.Symbols, currentClass: owner, localScope: method}
	var invocation *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(file.Source) == callee {
			invocation = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(method.DeclarationNode)
	if invocation == nil {
		t.Fatal("call missing")
	}
	args := nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
	if len(args) != 4 {
		t.Fatalf("call arity %d", len(args))
	}
	var origin inferredJavaTypeOrigin
	typ, known := inferExprJavaType(args[2], ctx, file.Source, &origin)
	_, displayRank := javaArrayTypeParts(typ)
	resolution := findBestMethodInHierarchy(owner, callee, invocation.ChildByFieldName("arguments"), true, false, ctx, file.Source)
	selectedArity := -1
	if resolution != nil && resolution.def != nil {
		selectedArity = len(resolution.def.Parameters)
	}
	t.Logf("actual parameter=%s stored=%s IsVariadic=%t inferred=%q known=%t displayRank=%d originRank=%d nominal=%t unresolved=%t selectedArity=%d", parameter, method.Parameters[1].OriginalType, method.IsVariadic, typ, known, displayRank, origin.arrayRank, origin.nominalScope != nil, origin.unresolved, selectedArity)
	if !known || displayRank != 1 || origin.arrayRank != 1 {
		t.Errorf("spread parameter must retain one array rank in both display and declaration origin")
	}
	if resolution == nil || resolution.def == nil || selectedArity != 4 {
		t.Errorf("four-argument source call must select four-argument array overload")
	}
}

func TestNettyComposite4085ActualSourcesArrayOriginRED(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	root := filepath.Join("testdata", "netty_varargs_origin_4_1_138_Final", "io", "netty", "buffer")
	var files []parsing.SourceFile
	for _, name := range []string{"ByteBuf.java", "CompositeByteBuf.java"} {
		p := filepath.Join(root, name)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		f := parsing.SourceFile{Name: p, Source: data}
		if err = f.ParseAST(); err != nil {
			t.Fatal(err)
		}
		f.ParseSymbols()
		symbol.AddSymbolsToPackage(f.Symbols)
		files = append(files, f)
	}
	ResolveFiles(files)
	composite4085AssertCall(t, files[1], "CompositeByteBuf", "addComponents", "addComponents0", "buffers")
}

func TestNettyComposite4085RenamedArrayOriginRED(t *testing.T) {
	h := setupParseHelper(t, `class Chunk {} class Adapter<T> {} class Batch {
	 int count;
	 Batch append(boolean grow,Chunk... chunks){route(grow,count,chunks,0);return this;}
	 private Batch route(boolean grow,int at,Chunk[] chunks,int offset){return this;}
	 private <T> int route(boolean grow,int at,Adapter<T> adapter,T[] chunks,int offset){return at;}
	}`)
	composite4085AssertCall(t, h.File, "Batch", "append", "route", "chunks")
}
