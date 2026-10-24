package transpiler

import (
	"bytes"
	"go/format"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func resourceWriteTestContext(t *testing.T, project, class, method string, extra ...string) (Ctx, []byte) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	root := filepath.Join("resource-receiver-test", project)
	var files []parsing.SourceFile
	for _, fixture := range resourceWriteTestSources {
		if fixture.project != "original" && fixture.project != project {
			continue
		}
		file := parsing.SourceFile{Name: filepath.Join(root, fixture.name), Source: []byte(fixture.java)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
		files = append(files, file)
	}
	if len(extra) != 0 {
		file := parsing.SourceFile{Name: filepath.Join(root, "probe", "Case.java"), Source: []byte(extra[0])}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
		files = append(files, file)
	}
	for _, file := range files {
		ResolveFile(file)
	}
	scope := findQualifiedSourceClass(class)
	file := findFileScopeForClassScope(scope)
	if scope == nil || file == nil {
		t.Fatal("actual project declaration missing")
	}
	ctx := Ctx{currentClass: scope, currentFile: file, className: scope.Class.Name, localScope: scope.FindMethodByName(method, nil), executionContextName: "__actualInvocationExecution", importAliases: map[string]string{}, usedImports: map[string]bool{}}
	if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil {
		t.Fatal("actual method declaration missing")
	}
	return ctx, file.Source
}

func resourceWriteTestCalls(root *sitter.Node, source []byte) []*sitter.Node {
	var calls []*sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			name, object := node.ChildByFieldName("name"), node.ChildByFieldName("object")
			if name != nil && name.Content(source) == "write" && object != nil && object.Content(source) == "resource" {
				calls = append(calls, node)
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(root)
	return calls
}

func TestFileChannelWriteCompletedSourceResource(t *testing.T) {
	for _, row := range []struct {
		project, class, method string
		count                  int
	}{
		{"original", "app.ResourcesMain", "main", 1},
		{"oracle", "probe.ResourceReceiverOracle", "completedSource", 2},
		{"oracle", "probe.ResourceReceiverOracle", "sourceOverrideFailure", 1},
	} {
		t.Run(row.method, func(t *testing.T) {
			ctx, source := resourceWriteTestContext(t, row.project, row.class, row.method)
			calls := resourceWriteTestCalls(ctx.localScope.DeclarationNode, source)
			if len(calls) != row.count {
				t.Fatalf("actual resource write calls=%d", len(calls))
			}
			for _, call := range calls {
				object := call.ChildByFieldName("object")
				binding, known := fileChannelReadCaptureDeclaration(object, source)
				if !known || binding.declaration.Type() != "resource" {
					t.Fatal("actual lexical resource binding unavailable")
				}
				creation := binding.declaration.ChildByFieldName("value")
				if creation == nil || creation.Type() != "object_creation_expression" {
					t.Fatal("original direct source allocation changed")
				}
				// Exercise the actual existing resource lowerer, which records the
				// declared type but does not retain declaration identity in locals.
				ParseStmt(binding.declaration, source, ctx)
				var origin inferredJavaTypeOrigin
				_, inferred := inferExprJavaType(creation, ctx, source, &origin)
				if !inferred || origin.nominalScope == nil || fileChannelSourceRoot(origin.nominalScope, ctx) == nil {
					t.Fatal("actual source allocation/ancestry proof missing")
				}
				proof, proved := fileChannelWriteReceiverProofFor(object, ctx, source)
				if !proved || proof.kind != fileChannelWriteReceiverSourceResource || !sameSourceNode(proof.resource, binding.declaration) || proof.allocation != origin.nominalScope || proof.description != "" {
					t.Fatal("completed proof lost actual declaration identity")
				}
				local := ctx.localScope.FindVariable("resource")
				if local == nil || local.DeclarationNode != nil {
					t.Fatal("snapshot no longer exercises declarationless local table")
				}
				t.Logf("resource=%s@%d sourceAllocation=%s sourceRoot=%s localDeclarationNil=%t", binding.declaration.Type(), binding.declaration.StartByte(), qualifiedSourceClassName(origin.nominalScope), qualifiedSourceClassName(fileChannelSourceRoot(origin.nominalScope, ctx)), local.DeclarationNode == nil)
				if arity, ok := fileChannelWriteSignature(call, ctx, source); !ok || arity != invocationArgumentCount(call) {
					t.Fatalf("completed actual source allocation refused: arity=%d known=%t expression=%s", arity, ok, call.Content(source))
				}
				expr := lowerFileChannelWrite(ParseExpr(object, source, ctx), call, ctx, source)
				var text bytes.Buffer
				if err := format.Node(&text, token.NewFileSet(), expr); err != nil {
					t.Fatal(err)
				}
				out := text.String()
				if !strings.Contains(out, "VirtualExecution(__actualInvocationExecution,") || strings.Contains(out, "because") || strings.Contains(out, "NativeExecution") {
					t.Fatalf("completed source view lost virtual/currentExecution or acquired an invented nullable description:\n%s", out)
				}
			}
		})
	}
}

func TestFileChannelWriteResourceRefusalBoundaries(t *testing.T) {
	for _, row := range []struct{ name, members, statement string }{
		{"null", "", `try(FileChannel resource=null){resource.write((ByteBuffer)null);}`},
		{"factory", `FileChannel factory(){return new ResourceChannel(0,false);}`, `try(FileChannel resource=factory()){resource.write((ByteBuffer)null);}`},
		{"alias", "", `FileChannel alias=new ResourceChannel(0,false);try(FileChannel resource=alias){resource.write((ByteBuffer)null);}`},
		{"field", `FileChannel saved;`, `try(FileChannel resource=saved){resource.write((ByteBuffer)null);}`},
		{"cast", "", `try(FileChannel resource=(FileChannel)new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}`},
		{"conditional", "", `try(FileChannel resource=true?new ResourceChannel(0,false):null){resource.write((ByteBuffer)null);}`},
		{"native-factory", `java.io.RandomAccessFile saved;`, `try(FileChannel resource=saved.getChannel()){resource.write((ByteBuffer)null);}`},
		{"var", "", `try(var resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}`},
		{"source-declared", "", `try(ResourceChannel resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}`},
		{"anonymous", "", `try(FileChannel resource=new ResourceChannel(0,false){}){resource.write((ByteBuffer)null);}`},
		{"inner", `class Inner extends ResourceChannel{Inner(){super(0,false);}}`, `try(FileChannel resource=new Inner()){resource.write((ByteBuffer)null);}`},
		{"generic", `static class Generic<T> extends ResourceChannel{Generic(){super(0,false);}}`, `try(FileChannel resource=new Generic<String>()){resource.write((ByteBuffer)null);}`},
		{"abstract-parser-control", `abstract static class Abstract extends ResourceChannel{Abstract(){super(0,false);}}`, `try(FileChannel resource=new Abstract()){resource.write((ByteBuffer)null);}`},
		{"unknown", "", `try(FileChannel resource=new Missing()){resource.write((ByteBuffer)null);}`},
		{"mutation-parser-control", "", `try(FileChannel resource=new ResourceChannel(0,false)){resource=null;resource.write((ByteBuffer)null);}`},
		{"nearer-local-parser-control", "", `try(FileChannel resource=new ResourceChannel(0,false)){{FileChannel resource=null;resource.write((ByteBuffer)null);}}`},
		{"lambda", "", `try(FileChannel resource=new ResourceChannel(0,false)){Runnable r=()->{resource.write((ByteBuffer)null);};}`},
		{"local-class", "", `class Local extends ResourceChannel{Local(){super(0,false);}}try(FileChannel resource=new Local()){resource.write((ByteBuffer)null);}`},
		{"existing-resource", "", `FileChannel resource=new ResourceChannel(0,false);try(resource){resource.write((ByteBuffer)null);}`},
		{"initializer-before-body", `FileChannel use(int n){return null;}`, `try(FileChannel resource=new ResourceChannel(0,false);FileChannel other=use(resource.write((ByteBuffer)null))){}`},
		{"catch-outside-binding", `FileChannel resource;`, `try(FileChannel resource=new ResourceChannel(0,false)){}catch(Exception failure){resource.write((ByteBuffer)null);}`},
		{"finally-outside-binding", `FileChannel resource;`, `try(FileChannel resource=new ResourceChannel(0,false)){}finally{resource.write((ByteBuffer)null);}`},
		{"class-boundary", "", `try(FileChannel resource=new ResourceChannel(0,false)){class Local{void use(){resource.write((ByteBuffer)null);}}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			java := `package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{` + row.members + `void run()throws Exception{` + row.statement + `}}`
			ctx, source := resourceWriteTestContext(t, "original", "probe.Case", "run", java)
			calls := resourceWriteTestCalls(ctx.localScope.DeclarationNode, source)
			if len(calls) != 1 {
				t.Fatalf("calls=%d", len(calls))
			}
			object := calls[0].ChildByFieldName("object")
			if binding, found := fileChannelReadCaptureDeclaration(object, source); found && binding.declaration.Type() == "resource" {
				if typ := binding.declaration.ChildByFieldName("type"); typ != nil {
					recordLocalVariableDefinition(ctx, "resource", typ.Content(source), typ.Content(source))
				}
			}
			if proof, known := fileChannelWriteSourceResourceProof(object, ctx, source); known || proof.kind != fileChannelWriteReceiverUnavailable {
				t.Fatalf("unproved boundary acquired allocation certificate: %#v", proof)
			}
		})
	}
}

func TestFileChannelWriteResourceOwnerAndIdentityShields(t *testing.T) {
	java := `package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{FileChannel resource;void run()throws Exception{try(FileChannel resource=new ResourceChannel(0,false)){this.resource=null;try(FileChannel other=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}}}}`
	ctx, source := resourceWriteTestContext(t, "original", "probe.Case", "run", java)
	call := resourceWriteTestCalls(ctx.localScope.DeclarationNode, source)[0]
	object := call.ChildByFieldName("object")
	binding, _ := fileChannelReadCaptureDeclaration(object, source)
	recordLocalVariableDefinition(ctx, "resource", "FileChannel", "FileChannel")
	ctx.expectedType = "foreign.FileChannel"
	ctx.expectedTypeRoot = call
	proof, known := fileChannelWriteSourceResourceProof(object, ctx, source)
	if !known || !sameSourceNode(proof.resource, binding.declaration) || proof.allocation != findQualifiedSourceClass("app.ResourceChannel") || ctx.expectedType != "foreign.FileChannel" || ctx.expectedTypeRoot != call {
		t.Fatal("nested try/field write/consumer expectation changed actual allocation identity")
	}
	missing := ctx.Clone()
	missing.currentFile = nil
	if _, known := fileChannelWriteSourceResourceProof(object, missing, source); known {
		t.Fatal("missing caller file borrowed ownership")
	}
	detached := ctx.Clone()
	copy := *ctx.currentClass
	detached.currentClass = &copy
	if _, known := fileChannelWriteSourceResourceProof(object, detached, source); known {
		t.Fatal("detached class borrowed actual file")
	}
	for _, extra := range []string{
		`package probe;import app.ResourceChannel;import java.nio.ByteBuffer;class FileChannel{}class Case{void run(){try(FileChannel resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}}}`,
		`package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import foreign.FileChannel;class Case{void run(){try(FileChannel resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}}}`,
		`package probe;import app.ResourceChannel;import java.nio.ByteBuffer;class Case<FileChannel extends java.nio.channels.FileChannel>{void run(){try(FileChannel resource=new ResourceChannel(0,false)){resource.write((ByteBuffer)null);}}}`,
	} {
		c, s := resourceWriteTestContext(t, "original", "probe.Case", "run", extra)
		n := resourceWriteTestCalls(c.localScope.DeclarationNode, s)[0]
		if _, known := fileChannelWriteSourceResourceProof(n.ChildByFieldName("object"), c, s); known {
			t.Fatal("source/import/binder shadow acquired canonical resource proof")
		}
	}
}

func TestFileChannelWriteResourceStaticMemberAllocation(t *testing.T) {
	java := `package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{static class Concrete extends ResourceChannel{Concrete(){super(0,false);}}void run()throws Exception{try(FileChannel resource=((new Concrete()))){resource.write((ByteBuffer)null, -7);}}}`
	ctx, source := resourceWriteTestContext(t, "original", "probe.Case", "run", java)
	call := resourceWriteTestCalls(ctx.localScope.DeclarationNode, source)[0]
	object := call.ChildByFieldName("object")
	recordLocalVariableDefinition(ctx, "resource", "FileChannel", "FileChannel")
	proof, known := fileChannelWriteSourceResourceProof(object, ctx, source)
	if !known || proof.allocation == nil || proof.allocation.Enclosing != ctx.currentClass || !declarationHasModifier(proof.allocation.Class.DeclarationNode, "static") || proof.kind != fileChannelWriteReceiverSourceResource {
		t.Fatal("actual static member allocation lost certified canonical view")
	}
	if _, known := fileChannelWriteSignature(call, ctx, source); !known {
		t.Fatal("completed static member resource refused positional write")
	}
}

func TestFileChannelWriteResourceNullableAndArgumentOrder(t *testing.T) {
	java := `package probe;import app.ResourceChannel;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{static ByteBuffer buffer(){return null;}static long position(){return -7;}void run(Long p)throws Exception{try(FileChannel resource=new ResourceChannel(0,false)){resource.write(buffer(),position());resource.write(buffer(),p);}}}`
	ctx, source := resourceWriteTestContext(t, "original", "probe.Case", "run", java)
	for index, call := range resourceWriteTestCalls(ctx.localScope.DeclarationNode, source) {
		object := call.ChildByFieldName("object")
		recordLocalVariableDefinition(ctx, "resource", "FileChannel", "FileChannel")
		if _, known := fileChannelWriteSignature(call, ctx, source); !known {
			t.Fatal("proven direct allocation refused known arguments")
		}
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), lowerFileChannelWrite(ParseExpr(object, source, ctx), call, ctx, source)); err != nil {
			t.Fatal(err)
		}
		text := out.String()
		buffer := executionImplementationName(ctx.currentClass.FindMethodByName("buffer", nil), ctx.currentClass, ctx)
		if strings.Count(text, buffer+"(") != 1 || !strings.Contains(text, "FileChannelWriteAtVirtualExecution(__actualInvocationExecution,") {
			t.Fatalf("arguments/caller execution not staged once:\n%s", text)
		}
		if index == 0 && strings.Contains(text, "because") {
			t.Fatal("nonnull resource received invented receiver diagnostic")
		}
		if index == 1 && (!strings.Contains(text, "java.lang.Long.longValue()") || strings.Contains(text, "java.nio.channels.FileChannel.write")) {
			t.Fatal("boxed argument null order/description changed")
		}
	}
	parameter := `package probe;import java.nio.ByteBuffer;import java.nio.channels.FileChannel;class Case{void run(FileChannel resource,ByteBuffer src)throws Exception{resource.write(src);}}`
	c, s := resourceWriteTestContext(t, "original", "probe.Case", "run", parameter)
	n := resourceWriteTestCalls(c.localScope.DeclarationNode, s)[0]
	p, known := fileChannelWriteReceiverProofFor(n.ChildByFieldName("object"), c, s)
	if !known || p.kind != fileChannelWriteReceiverNullable || p.description == "" || p.resource != nil || p.allocation != nil {
		t.Fatal("existing nullable parameter description changed")
	}
}

// Exact immutable calibration sources are inline so the regression also runs
// in an ordinary checkout without the campaign workspace.
var resourceWriteTestSources = []struct{ project, name, java string }{
	{"original", "app/ResourceChannel.java", "package app;\nimport java.io.IOException;import java.nio.ByteBuffer;\npublic class ResourceChannel extends model.Base {\n int mode;boolean failClose;\n public ResourceChannel(int mode,boolean failClose){super();this.mode=mode;this.failClose=failClose;}\n public int write(ByteBuffer src){if(mode==1)throw new IllegalStateException(\"write-failure\");if(src==null)return 0;int count=src.remaining();if(count>2)count=2;for(int i=0;i<count;i++){src.get();}return count;}\n public int write(ByteBuffer src,long position){return write(src);}\n protected void implCloseChannel()throws IOException {System.out.println(\"cleanup=\"+mode+\":\"+isOpen());if(failClose)throw new IOException(\"close-\"+mode);}\n}\n"},
	{"original", "app/ResourcesMain.java", "package app;\nimport java.nio.ByteBuffer;import java.nio.channels.FileChannel;import java.io.IOException;\npublic class ResourcesMain {\n public static void main(String[] args)throws Exception {\n  try(FileChannel absent=null){System.out.println(\"null-resource\");}\n  try(ResourceChannel source=new ResourceChannel(0,false)) {ByteBuffer src=ByteBuffer.wrap(new byte[]{4,5,6}).asReadOnlyBuffer();System.out.println(\"chunks=\"+source.write(src)+\":\"+src.position()+\":\"+source.write(src)+\":\"+src.position()+\":\"+source.write(src));}\n  try(FileChannel resource=new ResourceChannel(1,true)){resource.write((ByteBuffer)null);}catch(IllegalStateException failure){System.out.println(\"relative-primary=\"+failure.getMessage()+\":\"+failure.getSuppressed().length+\":\"+failure.getSuppressed()[0].getMessage());}\n  try(FileChannel resource=new ResourceChannel(0,true)){System.out.println(\"body-ok\");}catch(IOException failure){System.out.println(\"close-only=\"+failure.getMessage()+\":\"+failure.getSuppressed().length);}\n  ResourceChannel repeated=new ResourceChannel(0,false);repeated.close();repeated.close();System.out.println(\"repeat-open=\"+repeated.isOpen());\n  try(FileChannel first=new ResourceChannel(3,true);FileChannel second=new ResourceChannel(4,true)){throw new IllegalArgumentException(\"body\");}catch(IllegalArgumentException failure){System.out.println(\"reverse=\"+failure.getMessage()+\":\"+failure.getSuppressed().length+\":\"+failure.getSuppressed()[0].getMessage()+\":\"+failure.getSuppressed()[1].getMessage());}\n }\n}\n"},
	{"original", "model/Base.java", "package model;\nimport java.io.IOException; import java.nio.ByteBuffer; import java.nio.MappedByteBuffer; import java.nio.channels.*;\npublic abstract class Base extends FileChannel {\n public FileChannel saved; public static FileChannel slot;\n int baseField=mark(\"base-init\",2);\n static int mark(String text,int value){System.out.println(text+\"=\"+value);return value;}\n protected Base(){this(7);System.out.println(\"base-this\");}\n protected Base(int value){super();System.out.println(\"base-body=\"+value+\":\"+write(null,-7));}\n public int write(ByteBuffer dst){System.out.println(\"base-single\");return 10;}\n public int write(ByteBuffer dst,long position){System.out.println(\"base-at=\"+position);return 11;}\n    public long read(ByteBuffer[] dst, int off, int len) { throw new UnsupportedOperationException(); }\n    public long write(ByteBuffer[] src, int off, int len) { throw new UnsupportedOperationException(); }\n    public long position() { return 0; }\n    public FileChannel position(long position) { return this; }\n    public long size() { return 0; }\n    public FileChannel truncate(long size) { return this; }\n    public void force(boolean meta) { }\n    public long transferTo(long p, long c, WritableByteChannel out) { throw new UnsupportedOperationException(); }\n    public long transferFrom(ReadableByteChannel in, long p, long c) { throw new UnsupportedOperationException(); }\n    public MappedByteBuffer map(MapMode mode, long p, long s) { throw new UnsupportedOperationException(); }\n    public FileLock lock(long p, long s, boolean shared) { throw new UnsupportedOperationException(); }\n    public FileLock tryLock(long p, long s, boolean shared) { throw new UnsupportedOperationException(); }\n\n public int read(ByteBuffer dst){return 0;} public int read(ByteBuffer dst,long position){return 0;}\n protected void implCloseChannel() throws IOException{System.out.println(\"base-close=\"+isOpen());}\n public void superClose() throws IOException{super.close();}\n}\n"},
	{"oracle", "probe/ResourceReceiverOracle.java", "package probe;\nimport app.ResourceChannel;\nimport java.nio.ByteBuffer;\nimport java.nio.channels.FileChannel;\n\n// Diagnostic oracle only: nullable cases intentionally remain compiler refusals.\n// No expected helpful-NPE text is authored; capture the actual JDK21 output.\npublic class ResourceReceiverOracle {\n static String effects=\"\";\n static ByteBuffer source(){effects+=\"b\";return null;}\n static long position(){effects+=\"p\";return -7;}\n static FileChannel nullValue(){effects+=\"f\";return null;}\n static void show(String label,Throwable failure){System.out.println(label+\"|\"+effects+\"|\"+failure.getClass().getSimpleName()+\":\"+failure.getMessage());}\n static void nullRelative()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=null){resource.write(source());}\n  catch(NullPointerException failure){show(\"null-relative\",failure);}\n }\n static void nullPositional()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=null){resource.write(source(),position());}\n  catch(NullPointerException failure){show(\"null-positional\",failure);}\n }\n static void factoryRelative()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=nullValue()){resource.write(source());}\n  catch(NullPointerException failure){show(\"factory-relative\",failure);}\n }\n static void factoryPositional()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=nullValue()){resource.write(source(),position());}\n  catch(NullPointerException failure){show(\"factory-positional\",failure);}\n }\n static void boxedBeforeReceiver()throws Exception {\n  effects=\"\";Long offset=null;\n  try(FileChannel resource=null){resource.write(source(),offset);}\n  catch(NullPointerException failure){show(\"boxed-before-receiver\",failure);}\n }\n static void completedSource()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=new ResourceChannel(0,false)){\n   System.out.println(\"completed-relative=\"+resource.write(source())+\":\"+effects);\n   System.out.println(\"completed-positional=\"+resource.write(source(),position())+\":\"+effects);\n  }\n }\n static void sourceOverrideFailure()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=new ResourceChannel(1,false)){resource.write(source(),position());}\n  catch(IllegalStateException failure){show(\"source-override\",failure);}\n }\n static class Throwing extends ResourceChannel {\n  Throwing(){super(0,false);effects+=\"c\";throw new IllegalStateException(\"constructor-failed\");}\n }\n static void constructorFailure()throws Exception {\n  effects=\"\";\n  try(FileChannel resource=new Throwing()){System.out.println(\"UNREACHABLE-BODY\");}\n  catch(IllegalStateException failure){show(\"constructor\",failure);}\n }\n public static void main(String[] args)throws Exception {\n  nullRelative();nullPositional();factoryRelative();factoryPositional();boxedBeforeReceiver();completedSource();sourceOverrideFailure();constructorFailure();\n }\n}\n"},
}
