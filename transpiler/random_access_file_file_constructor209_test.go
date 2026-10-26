package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func raf209TestCreation(root *sitter.Node, source []byte) *sitter.Node {
	if root == nil {
		return nil
	}
	if root.Type() == "object_creation_expression" {
		typ := root.ChildByFieldName("type")
		if typ != nil && stripJavaQualifier(typ.Content(source)) == "RandomAccessFile" {
			return root
		}
	}
	for _, child := range nodeutil.SemanticNamedChildrenOf(root) {
		if found := raf209TestCreation(child, source); found != nil {
			return found
		}
	}
	return nil
}
func raf209TestContext(t *testing.T, java string) (Ctx, []byte, *sitter.Node) {
	t.Helper()
	helper := setupParseHelper(t, java)
	ctx := helper.Ctx
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	ctx.executionContextName = "__java2goRAFCurrentExecution"
	ctx.className = ctx.currentClass.Class.Name
	if ctx.localScope == nil {
		t.Fatal("actual run declaration missing")
	}
	node := raf209TestCreation(ctx.localScope.DeclarationNode, helper.File.Source)
	if node == nil {
		t.Fatal("actual RAF construction missing")
	}
	return ctx, helper.File.Source, node
}
func raf209TestCalls(t *testing.T, generated, api string) (*ast.File, []*ast.CallExpr) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	alias := ""
	for _, imp := range file.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if path == "github.com/NickyBoy89/java2go/stdjava" {
			alias = "stdjava"
			if imp.Name != nil {
				alias = imp.Name.Name
			}
		}
	}
	var calls []*ast.CallExpr
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if ok && owner.Name == alias && selector.Sel.Name == api {
			calls = append(calls, call)
		}
		return true
	})
	return file, calls
}

func TestRAFFile209StringDeclarationContracts(t *testing.T) {
	for _, java := range []string{
		`import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile("path","r");}}`,
		`import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile((String)null,(String)null);}}`,
		`import java.io.*;class Probe{Object run(String path,String mode)throws Exception{return new RandomAccessFile(path,mode);}}`,
		`import java.io.*;class Probe<S extends String>{Object run(S path,S mode)throws Exception{return new RandomAccessFile(path,mode);}}`,
		`import java.io.*;class Probe<S extends String,T extends S>{Object run(T path,S mode)throws Exception{return new RandomAccessFile(path,mode);}}`,
		`import java.io.*;class Probe{String path="path";Object run()throws Exception{return new RandomAccessFile(path,"r");}}`,
		`import java.io.*;class Probe{static String path(){return "path";}Object run()throws Exception{return new RandomAccessFile(path(),"r");}}`,
		`import java.io.*;class Probe{Object run(boolean b)throws Exception{return new RandomAccessFile(b?"path":null,"r");}}`,
	} {
		t.Run(java, func(t *testing.T) {
			strictRoutingState(t)
			_, calls := raf209TestCalls(t, renderGoFileFromJava(t, java), "NewRandomAccessFileStringExecution")
			if len(calls) != 1 || len(calls[0].Args) != 3 || len(Diagnostics()) != 0 {
				t.Fatal("valid String constructor lost its execution-aware service")
			}
		})
	}
}

func TestRAFFile209NamespaceCoherence(t *testing.T) {
	for _, row := range []struct {
		name, java string
		want       raf209ConstructorKind
	}{
		{"ordinary-single-type", `import java.io.File;import java.io.RandomAccessFile;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209NativeFile},
		{"ordinary-wildcard", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209NativeFile},
		{"static-single-nonconflicting", `import static java.lang.Math.abs;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209NativeFile},
		{"named-package-qualified", `package probe;class Probe{Object run()throws Exception{return new java.io.RandomAccessFile(new java.io.File("x"),"r");}}`, raf209NativeFile},
		{"legacy-String-static-wildcard", `import static java.lang.Math.*;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile("x","r");}}`, raf209String},
		{"new-File-static-wildcard-held", `import static foreign.Owner.*;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209Refused},
		{"single-static-File-conflict", `import static foreign.Owner.File;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209Refused},
		{"single-static-String-conflict", `import static foreign.Owner.String;import java.io.*;class Probe{Object run(String mode)throws Exception{return new RandomAccessFile(new File("x"),mode);}}`, raf209Refused},
		{"single-static-RAF-conflict", `import static foreign.Owner.RandomAccessFile;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, raf209Refused},
		{"unrelated-bound-not-on-String-chain", `import static foreign.Owner.Noise;import java.io.*;class Probe<T extends java.lang.String,U extends Noise>{Object run(T mode)throws Exception{return new RandomAccessFile("x",mode);}}`, raf209String},
		{"single-static-String-bound-conflict", `import static foreign.Owner.String;import java.io.*;class Probe<S extends String>{Object run(S mode)throws Exception{return new RandomAccessFile(new File("x"),mode);}}`, raf209Refused},
		{"qualified-File-unaffected-by-single-static-File", `import static foreign.Owner.File;class Probe{Object run()throws Exception{return new java.io.RandomAccessFile(new java.io.File("x"),"r");}}`, raf209NativeFile},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, node := raf209TestContext(t, row.java)
			if !raf209FileCoherent(ctx.currentFile) {
				t.Fatal("actual parsed namespace must first be coherent")
			}
			if got := raf209Constructor(node, nil, ctx, source); got != row.want {
				t.Fatalf("namespace origin=%d want%d", got, row.want)
			}
			if row.name == "legacy-String-static-wildcard" {
				strictRoutingState(t)
				_, calls := raf209TestCalls(t, renderGoFileFromJava(t, row.java), "NewRandomAccessFileStringExecution")
				if len(calls) != 1 || len(Diagnostics()) != 0 {
					t.Fatal("legacy valid String static-wildcard route changed")
				}
			}
		})
	}
}

func TestRAFFile209DeclarationTamperRefusal(t *testing.T) {
	t.Run("forged-String-bound", func(t *testing.T) {
		ctx, source, node := raf209TestContext(t, `import java.io.*;class Probe<S>{Object run(S path)throws Exception{return new RandomAccessFile(path,"r");}}`)
		forged := []symbol.JavaType{{Original: "java.lang.String", TypeParameterBindings: symbol.VisibleTypeParamBindings(ctx.currentClass.TypeParameters)}}
		ctx.currentClass.TypeParameters[0].Bounds = forged
		ctx.currentClass.DeclaredTypeParameters[0].Bounds = forged
		if raf209Constructor(node, nil, ctx, source) != raf209Refused {
			t.Fatal("mutable bounds manufactured a String certificate")
		}
	})
	t.Run("forged-formal-type", func(t *testing.T) {
		ctx, source, node := raf209TestContext(t, `import java.io.*;class Probe{Object run(Object path)throws Exception{return new RandomAccessFile(path,"r");}}`)
		ctx.localScope.Parameters[0].OriginalType = "java.lang.String"
		if raf209Constructor(node, nil, ctx, source) != raf209Refused {
			t.Fatal("formal sidecar overrode actual Object syntax")
		}
	})
	for _, row := range []struct{ name, java, before, after string }{
		{"changed-formal-source", `import java.io.*;class Probe{Object run(Object mode)throws Exception{return new RandomAccessFile(new File("x"),mode);}}`, "Object mode", "String mode"},
		{"changed-cast-source", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),(Object)null);}}`, "(Object)null", "(String)null"},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, node := raf209TestContext(t, row.java)
			if !raf209FileCoherent(ctx.currentFile) {
				t.Fatal("actual source must first be coherent")
			}
			changed := []byte(strings.Replace(string(source), row.before, row.after, 1))
			if string(changed) == string(source) || len(changed) != len(source) {
				t.Fatal("exact same-length mutation absent")
			}
			ctx.currentFile.Source = changed
			if raf209Constructor(node, nil, ctx, changed) != raf209Refused {
				t.Fatal("changed source bytes borrowed registered tree")
			}
		})
	}
	for _, row := range []struct{ name, java, key, value string }{
		{"foreign-File-import-donation", `import foreign.File;class Probe{Object run()throws Exception{return new java.io.RandomAccessFile(new File("x"),"r");}}`, "File", "java.io"},
		{"foreign-String-import-donation", `import foreign.String;class Probe{Object run(String mode)throws Exception{return new java.io.RandomAccessFile(new java.io.File("x"),mode);}}`, "String", "java.lang"},
		{"package-donation", `package foreign;import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new File("x"),"r");}}`, "", "java.io"},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, node := raf209TestContext(t, row.java)
			if !raf209FileCoherent(ctx.currentFile) {
				t.Fatal("actual namespace must first be coherent")
			}
			if row.key == "" {
				ctx.currentFile.Package = row.value
			} else {
				ctx.currentFile.Imports[row.key] = row.value
			}
			if raf209Constructor(node, nil, ctx, source) != raf209Refused {
				t.Fatal("namespace sidecar overrode actual header")
			}
		})
	}
	t.Run("outer-method-cannot-own-lambda-body", func(t *testing.T) {
		ctx, source, node := raf209TestContext(t, `import java.io.*;class Probe{Object run(){java.util.concurrent.Callable<Object> c=()->new RandomAccessFile(new File("x"),"r");return c;}}`)
		if raf209Constructor(node, nil, ctx, source) != raf209Refused {
			t.Fatal("outer executable donated lexical authority")
		}
	})

	const java = `import java.io.*;class Probe{Object run()throws Exception{File file=new File("x");return new RandomAccessFile(file,"r");}}`
	for _, name := range []string{"detached-occurrence", "copied-method-definition", "detached-current-file"} {
		t.Run(name, func(t *testing.T) {
			ctx, source, node := raf209TestContext(t, java)
			if raf209Constructor(node, nil, ctx, source) != raf209NativeFile {
				t.Fatal("registered native fixture must first be eligible")
			}
			switch name {
			case "detached-occurrence":
				twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), source...)}
				if err := twin.ParseAST(); err != nil {
					t.Fatal(err)
				}
				node = raf209TestCreation(twin.Ast, source)
			case "copied-method-definition":
				method := *ctx.localScope
				ctx.localScope = &method
			case "detached-current-file":
				file := *ctx.currentFile
				ctx.currentFile = &file
			}
			if raf209Constructor(node, nil, ctx, source) != raf209Refused {
				t.Fatal("detached declaration acquired native allocation permission")
			}
		})
	}
}

func TestRAFFile209NativeLocalWriteIdentity(t *testing.T) {
	for _, row := range []struct {
		name, java string
		want       raf209ConstructorKind
	}{
		{"unwritten", `import java.io.*;class Probe{Object run()throws Exception{File file=new File("x");return new RandomAccessFile(file,"r");}}`, raf209NativeFile},
		{"nested-other-binding", `import java.io.*;class Probe{Object run()throws Exception{File file=new File("x");class Other{void change(File file){file=null;}}return new RandomAccessFile(file,"r");}}`, raf209NativeFile},
		{"nested-actual-write", `import java.io.*;class Probe{Object run(File other)throws Exception{File file=new File("x");Runnable r=()->{file=other;};return new RandomAccessFile(file,"r");}}`, raf209Refused},
		{"later-write", `import java.io.*;class Probe{Object run(File other)throws Exception{File file=new File("x");Object r=new RandomAccessFile(file,"r");file=other;return r;}}`, raf209Refused},
		{"native-alias-still-held", `import java.io.*;class Probe{Object run()throws Exception{File first=new File("x");File file=first;return new RandomAccessFile(file,"r");}}`, raf209Refused},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, node := raf209TestContext(t, row.java)
			if got := raf209Constructor(node, nil, ctx, source); got != row.want {
				t.Fatalf("native origin=%d want%d", got, row.want)
			}
		})
	}
}

func TestRAFFile209FileLoweringEvaluationContract(t *testing.T) {
	java := `import java.io.*;class Probe<__java2goRAFFile>{static String name(){return "x";}static String mode(){return "r";}Object run()throws Exception{return new RandomAccessFile(new File(name()),mode());}}`
	file, calls := raf209TestCalls(t, renderGoFileFromJava(t, java), "NewRandomAccessFileFileExecution")
	if len(calls) != 1 || len(calls[0].Args) != 3 {
		t.Fatal("one typed File service with current execution required")
	}
	call := calls[0]
	execution, ok := call.Args[0].(*ast.Ident)
	if !ok || execution.Name == "" {
		t.Fatal("constructor must forward current Execution")
	}
	var body *ast.BlockStmt
	ast.Inspect(file, func(n ast.Node) bool {
		literal, ok := n.(*ast.FuncLit)
		if !ok || len(literal.Body.List) != 3 {
			return true
		}
		ret, ok := literal.Body.List[2].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		c, ok := ret.Results[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if ok && s.Sel.Name == "NewRandomAccessFileFileExecution" {
			body = literal.Body
		}
		return true
	})
	if body == nil {
		t.Fatal("explicit ordered operand captures missing")
	}
	matchedExecution := false
	ast.Inspect(file, func(n ast.Node) bool {
		declaration, ok := n.(*ast.FuncDecl)
		if !ok || declaration.Body == nil {
			return true
		}
		owns := false
		ast.Inspect(declaration.Body, func(at ast.Node) bool {
			if at == call {
				owns = true
			}
			return true
		})
		if owns && declaration.Type.Params != nil {
			for _, field := range declaration.Type.Params.List {
				pointer, ok := field.Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				typ, ok := pointer.X.(*ast.SelectorExpr)
				if !ok || typ.Sel.Name != "Execution" {
					continue
				}
				for _, name := range field.Names {
					if name.Name == execution.Name {
						matchedExecution = true
					}
				}
			}
		}
		return true
	})
	if !matchedExecution {
		t.Fatal("File constructor did not forward its enclosing current Execution parameter")
	}
	first, ok := body.List[0].(*ast.AssignStmt)
	if !ok || first.Tok != token.DEFINE || len(first.Lhs) != 1 || len(first.Rhs) != 1 {
		t.Fatal("File operand must be captured once first")
	}
	second, ok := body.List[1].(*ast.AssignStmt)
	if !ok || second.Tok != token.DEFINE || len(second.Lhs) != 1 || len(second.Rhs) != 1 {
		t.Fatal("mode operand must be captured once second")
	}
	left, lok := first.Lhs[0].(*ast.Ident)
	right, rok := second.Lhs[0].(*ast.Ident)
	if !lok || !rok || left.Name == right.Name || left.Name == "__java2goRAFFile" {
		t.Fatal("capture names collide with Java declarations")
	}
	returned := body.List[2].(*ast.ReturnStmt).Results[0].(*ast.CallExpr)
	a, aok := returned.Args[1].(*ast.Ident)
	b, bok := returned.Args[2].(*ast.Ident)
	if !aok || !bok || a.Name != left.Name || b.Name != right.Name {
		t.Fatal("facade must consume the captured operands in order")
	}
}

type raf209CaptureFixture struct {
	declaring, lambdaCtx Ctx
	source               []byte
	creation, resource   *sitter.Node
	occurrence, lambda   *sitter.Node
}

func raf209TestCaptureFixture(t *testing.T, java string) raf209CaptureFixture {
	t.Helper()
	ctx, source, creation := raf209TestContext(t, java)
	var occurrence *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil || occurrence != nil {
			return
		}
		if node.Type() == "method_invocation" && node.ChildByFieldName("name") != nil && node.ChildByFieldName("name").Content(source) == "setLength" {
			occurrence = node.ChildByFieldName("object")
			return
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if occurrence == nil {
		t.Fatal("actual captured receiver occurrence missing")
	}
	binding, found := fileChannelReadCaptureDeclaration(occurrence, source)
	if !found || binding.declaration.Type() != "resource" || !unwrapParenthesizedExpressionNode(binding.declaration.ChildByFieldName("value")).Equal(creation) {
		t.Fatal("actual immutable resource/initializer identity missing")
	}
	var lambda *sitter.Node
	for at := occurrence; at != nil; at = at.Parent() {
		if at.Type() == "lambda_expression" {
			lambda = at
			break
		}
	}
	if lambda == nil {
		t.Fatal("actual capture lambda missing")
	}
	// Ordinary try lowering records the declared resource before it parses the
	// body and clones the lambda scope. Replay that actual statement here.
	// A lambda-created resource must never be replayed as its outer method.
	if !readerSourceNodeWithin(binding.declaration, lambda.ChildByFieldName("body")) {
		beforeDiagnostics := len(Diagnostics())
		if statement := ParseStmt(binding.declaration, source, ctx); statement == nil || len(Diagnostics()) != beforeDiagnostics {
			t.Fatal("actual outer resource statement replay failed")
		}
	}
	lambdaCtx := contextWithLambdaParameters(ctx, lambda.ChildByFieldName("parameters"), nil, "void", source)
	if lambdaCtx.localScope == nil || lambdaCtx.localScope.DeclarationNode != nil || lambdaCtx.localBindingBody == nil || !lambdaCtx.localBindingBody.Equal(lambda.ChildByFieldName("body")) {
		t.Fatal("ordinary copied lambda context premise missing")
	}
	return raf209CaptureFixture{ctx, lambdaCtx, source, creation, binding.declaration, occurrence, lambda}
}

func TestRAFFile209CapturedResourceInitializerContext(t *testing.T) {
	const tail = `){Runnable r=()->{try{f.setLength(2L);f.write(new byte[0]);}catch(java.io.IOException failure){throw new RuntimeException(failure);}};r.run();}}}`
	const stringJava = `import java.io.*;class Probe{void run(String path)throws Exception{try(RandomAccessFile f=new RandomAccessFile(path,"rw")` + tail
	const fileJava = `import java.io.*;class Probe{void run()throws Exception{try(RandomAccessFile f=new RandomAccessFile(new File("path"),"rw")` + tail
	const genericJava = `import java.io.*;class Probe{<S extends String>void run(String path,S mode)throws Exception{try(RandomAccessFile f=new RandomAccessFile(path,mode)` + tail
	for _, row := range []struct {
		name, java, api string
		kind            raf209ConstructorKind
	}{
		{"string-capture-replay", stringJava, "NewRandomAccessFileStringExecution", raf209String},
		{"file-capture-replay", fileJava, "NewRandomAccessFileFileExecution", raf209NativeFile},
		{"generic-mode-capture-replay", genericJava, "NewRandomAccessFileStringExecution", raf209String},
	} {
		t.Run(row.name, func(t *testing.T) {
			strictRoutingState(t)
			fixture := raf209TestCaptureFixture(t, row.java)
			if raf209Constructor(fixture.creation, nil, fixture.lambdaCtx, fixture.source) != raf209Refused {
				t.Fatal("current lambda must not impersonate the initializer's registered method")
			}
			proof, found := raf209ResourceInitializerContext(fixture.occurrence, fixture.resource, fixture.lambdaCtx, fixture.source)
			if !found {
				t.Fatal("actual initializer declaring executable context missing")
			}
			if proof.localScope != fixture.declaring.localScope || proof.localScope == fixture.lambdaCtx.localScope {
				t.Fatal("initializer context did not select the actual registered executable")
			}
			if kind := raf209Constructor(fixture.creation, nil, proof, fixture.source); kind != row.kind {
				t.Fatalf("initializer constructor kind=%d want%d", kind, row.kind)
			}
			if !randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
				t.Fatalf("actual capture allocation certificate missing; occurrence external type=%t", fileChannelReadExternalExpression(fixture.occurrence, "java.io.RandomAccessFile", fixture.lambdaCtx, fixture.source))
			}
			if fixture.lambdaCtx.localScope.DeclarationNode != nil || !fixture.lambdaCtx.localBindingBody.Equal(fixture.lambda.ChildByFieldName("body")) {
				t.Fatal("initializer proof mutated the occurrence context")
			}
			generated := renderGoFileFromJava(t, row.java)
			for _, api := range []string{row.api, "RandomAccessFileSetLengthExecution", "RandomAccessFileWriteBytesExecution"} {
				_, calls := raf209TestCalls(t, generated, api)
				if len(calls) != 1 {
					t.Fatalf("capture replay %s calls=%d, want1", api, len(calls))
				}
			}
			if len(Diagnostics()) != 0 {
				t.Fatalf("capture replay diagnosed: %+v", Diagnostics())
			}
		})
	}
	for _, row := range []struct {
		name   string
		mutate func(*raf209CaptureFixture)
	}{
		{"missing-lambda-frame", func(f *raf209CaptureFixture) { f.lambdaCtx.localScope = nil }},
		{"outer-method-occurrence-donation", func(f *raf209CaptureFixture) { f.lambdaCtx = f.declaring }},
		{"missing-lambda-body", func(f *raf209CaptureFixture) { f.lambdaCtx.localBindingBody = nil }},
		{"wrong-lambda-body", func(f *raf209CaptureFixture) {
			f.lambdaCtx.localBindingBody = f.declaring.localScope.DeclarationNode.ChildByFieldName("body")
		}},
		{"copied-method-frame", func(f *raf209CaptureFixture) {
			f.lambdaCtx.localScope.DeclarationNode = f.declaring.localScope.DeclarationNode
		}},
		{"missing-registered-executable", func(f *raf209CaptureFixture) { f.lambdaCtx.currentClass.Methods = nil }},
		{"forged-executable-header", func(f *raf209CaptureFixture) { f.declaring.localScope.OriginalName = "foreign" }},
		{"erased-declaring-binder", func(f *raf209CaptureFixture) { f.declaring.localScope.TypeParameters = nil }},
		{"forged-namespace", func(f *raf209CaptureFixture) { f.lambdaCtx.currentFile.Imports["RandomAccessFile"] = "foreign" }},
		{"detached-current-file", func(f *raf209CaptureFixture) { copied := *f.lambdaCtx.currentFile; f.lambdaCtx.currentFile = &copied }},
		{"wrong-resource-identity", func(f *raf209CaptureFixture) { f.resource = f.creation }},
	} {
		t.Run(row.name, func(t *testing.T) {
			fixture := raf209TestCaptureFixture(t, genericJava)
			if !randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
				t.Fatal("actual capture certificate must first be admitted")
			}
			row.mutate(&fixture)
			proof, found := raf209ResourceInitializerContext(fixture.occurrence, fixture.resource, fixture.lambdaCtx, fixture.source)
			if found && raf209Constructor(fixture.creation, nil, proof, fixture.source) != raf209Refused {
				t.Fatal("tampered capture/initializer context donated a constructor certificate")
			}
			if row.name != "wrong-resource-identity" && randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
				t.Fatal("tampered occurrence admitted native allocation")
			}
		})
	}
	t.Run("detached-initializer-tree", func(t *testing.T) {
		fixture := raf209TestCaptureFixture(t, stringJava)
		if !randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
			t.Fatal("live registered capture must first be eligible")
		}
		twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), fixture.source...)}
		if err := twin.ParseAST(); err != nil {
			t.Fatal(err)
		}
		twinCreation := raf209TestCreation(twin.Ast, twin.Source)
		if twinCreation == nil || twinCreation.Parent() == nil || twinCreation.Parent().Type() != "resource" {
			t.Fatal("same-byte detached resource premise missing")
		}
		if _, found := raf209ResourceInitializerContext(fixture.occurrence, twinCreation.Parent(), fixture.lambdaCtx, fixture.source); found {
			t.Fatal("same-byte detached resource donated its initializer")
		}
		if !randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
			t.Fatal("parse-only twin changed the live registry/capture premise")
		}
	})
	t.Run("initializer-inside-lambda-held", func(t *testing.T) {
		fixture := raf209TestCaptureFixture(t, `import java.io.*;class Probe{void run(){Runnable r=()->{try(RandomAccessFile f=new RandomAccessFile("path","rw")){f.setLength(2L);}catch(java.io.IOException failure){throw new RuntimeException(failure);}};}}`)
		if _, found := raf209ResourceInitializerContext(fixture.occurrence, fixture.resource, fixture.lambdaCtx, fixture.source); found || randomAccessFileNativeAllocation(fixture.occurrence, fixture.lambdaCtx, fixture.source) {
			t.Fatal("enclosing method borrowed ownership of a lambda initializer")
		}
		if raf209Constructor(fixture.creation, nil, fixture.declaring, fixture.source) != raf209Refused {
			t.Fatal("outer-method-cannot-own-lambda-body refusal changed")
		}
	})
}

func TestRAFFile209StringOperandProvenance(t *testing.T) {
	for _, row := range []struct {
		name, java string
		want       raf209ConstructorKind
	}{
		{"native-new-bare", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new String("x"),new String("r"));}}`, raf209String},
		{"native-new-qualified", `class Probe{Object run()throws Exception{return new java.io.RandomAccessFile(new java.lang.String("x"),new java.lang.String("r"));}}`, raf209String},
		{"generic-literal-witness", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(id("x"),id("r"));}}`, raf209String},
		{"generic-new-witness-static-owner", `class Probe<RandomAccessFile>{static int String=8;static<T>T id(T value){return value;}static Object run()throws Exception{return new java.io.RandomAccessFile(id(new java.lang.String("x")),id(new java.lang.String("r")));}}`, raf209String},
		{"generic-explicit-native-String", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String>id("x"),"r");}}`, raf209String},
		{"generic-all-T-witnesses", `import java.io.*;class Probe{static<T>T pick(T first,T second){return first;}Object run()throws Exception{return new RandomAccessFile(pick("x",new String("y")),"r");}}`, raf209String},
		{"unrelated-U-does-not-refuse-T", `import java.io.*;class Probe{static<T,U extends CharSequence>T pick(T value,U noise){return value;}Object run()throws Exception{return new RandomAccessFile(pick("x",(java.lang.String)null),"r");}}`, raf209String},
		{"caller-T-distinct-from-method-T", `import java.io.*;class Probe<T>{static<T>T id(T value){return value;}Object run(T other)throws Exception{return new RandomAccessFile(id("x"),"r");}}`, raf209String},
		{"static-TYPE-qualified-witness", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.id("x"),"r");}}`, raf209String},
		{"source-VALUE-generic-witness", `import java.io.*;class Probe{Object run(Donor value)throws Exception{return new RandomAccessFile(value.id("x"),"r");}}class Donor{<T>T id(T value){return value;}}`, raf209String},
		{"VALUE-Probe-masks-TYPE-Probe", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run(Donor Probe)throws Exception{return new RandomAccessFile(Probe.id("x"),"r");}}class Donor{Object id(String value){return value;}}`, raf209Refused},
		{"qualified-native-String-over-source-name", `import java.io.*;class String{String(java.lang.String value){}}class Probe{Object run()throws Exception{return new RandomAccessFile(new java.lang.String("x"),"r");}}`, raf209String},
		{"bounded-T-witness", `import java.io.*;class Probe{static<T extends String>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}`, raf209String},
		{"bounded-T-zero-witness-held", `import java.io.*;class Probe{static<T extends String>T id(){return null;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String>id(),"r");}}`, raf209Refused},
		{"bounded-T-null-witness-held", `import java.io.*;class Probe{static<T extends String>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String>id(null),"r");}}`, raf209Refused},
		{"bounded-T-explicit-Object-held", `import java.io.*;class Probe{static<T extends String>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<Object>id("x"),"r");}}`, raf209Refused},
		{"bounded-T-explicit-arity-held", `import java.io.*;class Probe{static<T extends String>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String,Object>id("x"),"r");}}`, raf209Refused},
		{"source-String-not-native", `import java.io.RandomAccessFile;class String{String(java.lang.String value){}}class Probe{Object run()throws Exception{return new RandomAccessFile(new String("x"),"r");}}`, raf209Refused},
		{"foreign-String-not-native", `import java.io.RandomAccessFile;import foreign.String;class Probe{Object run()throws Exception{return new RandomAccessFile(new String("x"),"r");}}`, raf209Refused},
		{"binder-String-not-native", `import java.io.RandomAccessFile;class Probe<String>{Object run()throws Exception{return new RandomAccessFile(new String("x"),"r");}}`, raf209Refused},
		{"String-class-body-held", `import java.io.*;class Probe{Object run()throws Exception{return new RandomAccessFile(new java.lang.String("x"){},"r");}}`, raf209Refused},
		{"generic-unknown-Object-held", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run(Object value)throws Exception{return new RandomAccessFile(id(value),"r");}}`, raf209Refused},
		{"generic-zero-witness-held", `import java.io.*;class Probe{static<T>T id(){return null;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String>id(),"r");}}`, raf209Refused},
		{"unrelated-U-cannot-donate-String", `import java.io.*;class Probe{static<T,U extends String>T pick(T value,U noise){return value;}Object run(Object value)throws Exception{return new RandomAccessFile(pick(value,"r"),"r");}}`, raf209Refused},
		{"generic-one-unknown-T-held", `import java.io.*;class Probe{static<T>T pick(T first,T second){return first;}Object run(Object other)throws Exception{return new RandomAccessFile(pick("x",other),"r");}}`, raf209Refused},
		{"generic-array-T-held", `import java.io.*;class Probe{static<T>T first(T[] values){return values[0];}Object run()throws Exception{return new RandomAccessFile(first(new java.lang.String[]{"x"}),"r");}}`, raf209Refused},
		{"generic-nested-T-held", `import java.io.*;class Probe{static<T>T pick(T value,java.util.List<T> values){return value;}Object run()throws Exception{return new RandomAccessFile(pick("x",null),"r");}}`, raf209Refused},
		{"generic-varargs-held", `import java.io.*;class Probe{static<T>T first(T... values){return values[0];}Object run()throws Exception{return new RandomAccessFile(first("x"),"r");}}`, raf209Refused},
		{"selected-nongeneric-Object-overload-held", `import java.io.*;class Probe{static Object id(String value){return value;}static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}`, raf209Refused},
		{"generic-explicit-foreign-String-held", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<foreign.String>id("x"),"r");}}`, raf209Refused},
		{"generic-explicit-Object-held", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<Object>id("x"),"r");}}`, raf209Refused},
		{"generic-explicit-caller-binder-held", `import java.io.*;class Probe<S>{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<S>id("x"),"r");}}`, raf209Refused},
		{"generic-explicit-arity-held", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String,Object>id("x"),"r");}}`, raf209Refused},
		{"generic-untyped-null-held", `import java.io.*;class Probe{static<T>T id(T value){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.<java.lang.String>id(null),"r");}}`, raf209Refused},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, creation := raf209TestStringOperandContext(t, row.java)
			if got := raf209Constructor(creation, nil, ctx, source); got != row.want {
				t.Fatalf("String operand certificate=%d want%d", got, row.want)
			}
			if row.want == raf209String {
				strictRoutingState(t)
				generated := renderGoFileFromJava(t, row.java)
				_, calls := raf209TestCalls(t, generated, "NewRandomAccessFileStringExecution")
				if len(calls) != 1 || len(Diagnostics()) != 0 {
					t.Fatalf("canonical String operand replay calls=%d diagnostics=%+v", len(calls), Diagnostics())
				}
			}
		})
	}
	const java = `import java.io.*;class Probe{static<T,U extends String>T pick(T value,U noise){return value;}Object run()throws Exception{return new RandomAccessFile(pick(new java.lang.String("x"),"r"),new java.lang.String("r"));}}`
	for _, row := range []struct {
		name   string
		mutate func(Ctx, *symbol.Definition, []byte)
	}{
		{"return-DirectTP-erased", func(_ Ctx, def *symbol.Definition, _ []byte) { def.DirectTypeParameter = nil }},
		{"return-DirectTP-donated-U", func(_ Ctx, def *symbol.Definition, _ []byte) {
			def.DirectTypeParameter = def.TypeParameters[1].Declaration
		}},
		{"return-binding-donated-U", func(_ Ctx, def *symbol.Definition, _ []byte) {
			def.TypeParameterBindings["T"] = def.TypeParameters[1].Declaration
		}},
		{"return-bindings-erased", func(_ Ctx, def *symbol.Definition, _ []byte) { def.TypeParameterBindings = nil }},
		{"formal-DirectTP-donated-U", func(_ Ctx, def *symbol.Definition, _ []byte) {
			def.Parameters[0].DirectTypeParameter = def.TypeParameters[1].Declaration
		}},
		{"formal-bindings-erased", func(_ Ctx, def *symbol.Definition, _ []byte) { def.Parameters[0].TypeParameterBindings = nil }},
		{"written-return-disagrees", func(_ Ctx, def *symbol.Definition, _ []byte) { def.OriginalType = "U" }},
		{"method-header-name-disagrees", func(_ Ctx, def *symbol.Definition, _ []byte) { def.OriginalName = "foreign" }},
		{"method-binders-erased", func(_ Ctx, def *symbol.Definition, _ []byte) { def.TypeParameters = nil }},
		{"provider-namespace-disagrees", func(ctx Ctx, _ *symbol.Definition, _ []byte) { ctx.currentFile.Imports["String"] = "foreign" }},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, creation := raf209TestStringOperandContext(t, java)
			if raf209Constructor(creation, nil, ctx, source) != raf209String {
				t.Fatal("registered selected T witness must first be eligible")
			}
			def := ctx.currentClass.FindMethodByName("pick", nil)
			if def == nil || len(def.TypeParameters) != 2 || def.DirectTypeParameter != def.TypeParameters[0].Declaration {
				t.Fatal("actual selected T/U source declaration premise missing")
			}
			row.mutate(ctx, def, source)
			if raf209Constructor(creation, nil, ctx, source) != raf209Refused {
				t.Fatal("tampered source return/formal/binder/header donated a String result")
			}
		})
	}
	t.Run("bounded-return-binding-donation-held", func(t *testing.T) {
		const bounded = `import java.io.*;class Probe{static<T extends String,U extends String>T pick(T value,U noise){return value;}Object run()throws Exception{return new RandomAccessFile(Probe.pick("x","r"),"r");}}`
		ctx, source, creation := raf209TestStringOperandContext(t, bounded)
		if raf209Constructor(creation, nil, ctx, source) != raf209String {
			t.Fatal("actual bounded T witness must first be eligible")
		}
		def := ctx.currentClass.FindMethodByName("pick", nil)
		if def == nil || len(def.TypeParameters) != 2 {
			t.Fatal("actual bounded T/U provider missing")
		}
		def.DirectTypeParameter = def.TypeParameters[1].Declaration
		if raf209Constructor(creation, nil, ctx, source) != raf209Refused {
			t.Fatal("bounded-return shortcut bypassed selected return-binder identity")
		}
	})

	// These routes are source-inspected counterfacts, not previously measured RED.
	for _, row := range []struct {
		name, java, subject string
		want                raf209ConstructorKind
	}{
		{"enclosing-canonical", `import java.io.*;class Probe{static<T>T id(T value){return value;}static class Inner{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}}`, "Probe.Inner", raf209String},
		{"enclosing-varargs", `import java.io.*;class Probe{static<T>T id(T... values){return values[0];}static class Inner{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}}`, "Probe.Inner", raf209Refused},
		{"enclosing-zero", `import java.io.*;class Probe{static<T>T id(){return null;}static class Inner{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}}`, "Probe.Inner", raf209Refused},
		{"enclosing-null", `import java.io.*;class Probe{static<T extends String>T id(T value){return value;}static class Inner{Object run()throws Exception{return new RandomAccessFile(id(null),"r");}}}`, "Probe.Inner", raf209Refused},
		{"enclosing-bounded-zero", `import java.io.*;class Probe{static<T extends String>T id(){return null;}static class Inner{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}}`, "Probe.Inner", raf209Refused},
		{"import-single-canonical", `package probe;import java.io.*;import static probe.Donor.id;class Probe{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}class Donor{public static<T>T id(T value){return value;}}`, "Probe", raf209String},
		{"import-single-varargs", `package probe;import java.io.*;import static probe.Donor.id;class Probe{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}class Donor{public static<T>T id(T... values){return values[0];}}`, "Probe", raf209Refused},
		{"import-single-zero", `package probe;import java.io.*;import static probe.Donor.id;class Probe{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}class Donor{public static<T>T id(){return null;}}`, "Probe", raf209Refused},
		{"import-single-null", `package probe;import java.io.*;import static probe.Donor.id;class Probe{Object run()throws Exception{return new RandomAccessFile(id(null),"r");}}class Donor{public static<T extends String>T id(T value){return value;}}`, "Probe", raf209Refused},
		{"import-single-bounded-zero", `package probe;import java.io.*;import static probe.Donor.id;class Probe{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}class Donor{public static<T extends String>T id(){return null;}}`, "Probe", raf209Refused},
		{"import-wildcard-canonical", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}class Donor{public static<T>T id(T value){return value;}}`, "Probe", raf209String},
		{"import-wildcard-varargs", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(id("x"),"r");}}class Donor{public static<T>T id(T... values){return values[0];}}`, "Probe", raf209Refused},
		{"import-wildcard-zero", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}class Donor{public static<T>T id(){return null;}}`, "Probe", raf209Refused},
		{"import-wildcard-null", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(id(null),"r");}}class Donor{public static<T extends String>T id(T value){return value;}}`, "Probe", raf209Refused},
		{"import-wildcard-bounded-zero", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(id(),"r");}}class Donor{public static<T extends String>T id(){return null;}}`, "Probe", raf209Refused},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, creation := raf209TestStringOperandContext(t, row.java, row.subject)
			if kind := raf209Constructor(creation, nil, ctx, source); kind != row.want {
				t.Fatalf("owned source route certificate=%d want%d", kind, row.want)
			}
			if row.want == raf209String {
				strictRoutingState(t)
				_, calls := raf209TestCalls(t, renderGoFileFromJava(t, row.java), "NewRandomAccessFileStringExecution")
				if len(calls) != 1 || len(Diagnostics()) != 0 {
					t.Fatal("canonical owned enclosing/imported source invocation lost its native String result")
				}
			}
		})
	}
	for _, row := range []struct{ name, java, subject string }{
		{"enclosing-binder-tamper", `import java.io.*;class Probe{static<T extends String,U extends String>T pick(T value,U noise){return value;}static class Inner{Object run()throws Exception{return new RandomAccessFile(pick("x","r"),"r");}}}`, "Probe.Inner"},
		{"import-single-binder-tamper", `package probe;import java.io.*;import static probe.Donor.pick;class Probe{Object run()throws Exception{return new RandomAccessFile(pick("x","r"),"r");}}class Donor{public static<T extends String,U extends String>T pick(T value,U noise){return value;}}`, "Probe"},
		{"import-wildcard-binder-tamper", `package probe;import java.io.*;import static probe.Donor.*;class Probe{Object run()throws Exception{return new RandomAccessFile(pick("x","r"),"r");}}class Donor{public static<T extends String,U extends String>T pick(T value,U noise){return value;}}`, "Probe"},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx, source, creation := raf209TestStringOperandContext(t, row.java, row.subject)
			if raf209Constructor(creation, nil, ctx, source) != raf209String {
				t.Fatal("actual selected route T witness must first be eligible")
			}
			selected, _, knownSource := raf209SelectSourceInvocation(invocationArgumentNode(creation, 0), ctx, source)
			if !knownSource || selected == nil || selected.def == nil || len(selected.def.TypeParameters) != 2 {
				t.Fatal("actual enclosing/imported T/U declaration missing")
			}
			selected.def.DirectTypeParameter = selected.def.TypeParameters[1].Declaration
			if raf209Constructor(creation, nil, ctx, source) != raf209Refused {
				t.Fatal("enclosing/imported return binder escaped selected declaration checks")
			}
		})
	}

}

func raf209TestStringOperandContext(t *testing.T, java string, subject ...string) (Ctx, []byte, *sitter.Node) {
	t.Helper()
	helper := setupParseHelper(t, java)
	ctx := helper.Ctx
	owner := "Probe"
	if len(subject) > 1 {
		t.Fatal("one actual fixture subject is required")
	}
	if len(subject) == 1 {
		owner = subject[0]
	}
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, owner)
	if ctx.currentClass == nil || ctx.currentClass.Class == nil {
		t.Fatal("actual registered Probe class missing")
	}
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	ctx.executionContextName = "__java2goRAFCurrentExecution"
	ctx.className = ctx.currentClass.Class.Name
	if ctx.localScope == nil {
		t.Fatal("actual Probe run declaration missing")
	}
	creation := raf209TestCreation(ctx.localScope.DeclarationNode, helper.File.Source)
	if creation == nil {
		t.Fatal("actual Probe RAF construction missing")
	}
	return ctx, helper.File.Source, creation
}
