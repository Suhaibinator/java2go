package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestReaderReadGenericOriginalDeclaredOriginAndSelection(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, canonicalRecursiveAliasWorkflowSources())
	main := findQualifiedSourceClass("cloneapp.Main")
	ctx = classScopeCtx(main, ctx)
	ctx.executionContextName = "__readerProbeExecution"
	for _, method := range main.Methods {
		if method.OriginalName == "main" {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil {
		t.Fatal("original main declaration missing")
	}
	source := ctx.currentFile.Source
	count := 0
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			object, name, arguments := node.ChildByFieldName("object"), node.ChildByFieldName("name"), node.ChildByFieldName("arguments")
			if object != nil && name != nil && name.Content(source) == "read" && nodeutil.SemanticNamedChildCount(arguments) == 0 && (object.Content(source) == "original" || object.Content(source) == "copy") {
				local := ctx
				for ancestor := node.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
					if ancestor.Type() == "lambda_expression" {
						local = contextWithLambdaParameters(ctx, ancestor.ChildByFieldName("parameters"), nil, "void", source)
						local.executionContextName = "__readerProbeExecution"
						break
					}
				}
				var origin inferredJavaTypeOrigin
				javaType, known := inferExprJavaType(object, local, source, &origin)
				base, _ := parseJavaTypeString(javaType)
				fullOwner := resolveClassScopeByQualifiedName(local, javaType)
				baseOwner := resolveClassScopeByQualifiedName(local, base)
				if !known || !completeJavaTypeOrigin(origin) || origin.nominalScope == nil || origin.nominalScope != baseOwner || len(origin.arguments) != 1 {
					t.Fatalf("actual declaration origin incomplete: type=%s known=%v origin=%+v", javaType, known, origin)
				}
				target := resolveInvocationTarget(object, local, source)
				selected, _ := findBestMethodForInvocationTarget(target, "read", arguments, true, true, local, source)
				if selected != nil && selected.def != nil && len(selected.def.Parameters) == 0 {
					t.Fatal("original Java unexpectedly declares source read() override")
				}
				operation := readerNativeReadOperation(object, "read", arguments, source, local)
				t.Logf("receiver=%s type=%s full-spelling-owner=%p actual-owner=%p ownerJava=%s declaringFile=%s arguments=%+v selected=%+v readOperation=%s", object.Content(source), javaType, fullOwner, origin.nominalScope, origin.nominalScope.Class.OriginalName, findFileScopeForClassScope(origin.nominalScope, local).Package, origin.arguments, selected, operation)
				if operation != "ReaderReadChar" {
					t.Errorf("actual Reader zero-arity route missing: %s", operation)
				}
				lowered := characterIOInvocation(object, "read", arguments, source, local)
				if lowered == nil {
					t.Errorf("complete actual generic Reader origin rejected before read0 lowering: receiver=%s type=%s fullOwner=%p actualOwner=%p", object.Content(source), javaType, fullOwner, origin.nominalScope)
				} else if got := canonicalRecursiveAliasPrint(t, lowered); !strings.Contains(got, "stdjava.ReaderReadCharExecution(__readerProbeExecution, "+object.Content(source)+")") {
					t.Errorf("lost current Execution or actual receiver: %s", got)
				}
				count++
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if count != 3 {
		t.Fatalf("original read0 occurrence coverage %d want3", count)
	}
}

func readerDeclaredOriginGuardWorkflow() map[string]string {
	return map[string]string{
		"pom.xml":                            "<project><groupId>readorigin</groupId><artifactId>readorigin</artifactId><version>1</version></project>\n",
		"src/main/java/foreign/Reader.java":  "package foreign;public class Reader<T>{public int read(){return 88;}}\n",
		"src/main/java/lib/Base.java":        "package lib;\npublic class Base<T> extends java.io.Reader {\n public int calls; public int value;\n public Base(int value){this.value=value;}\n public int read(char[] b,int o,int n){calls++;if(n==0)return 0;b[o]=(char)value;return 1;}\n public String read(long count){return \"overload:\"+count;}\n public void close(){}\n}\n",
		"src/main/java/lib/Holder.java":      "package lib;\npublic class Holder<T>{public Base<T> value;public Holder(Base<T> value){this.value=value;}}\n",
		"src/main/java/readorigin/Main.java": "package readorigin;\npublic class Main {\n static class Base{} static class Reader{}\n static class Override<T> extends lib.Base<T>{\n  Override(){super(0);}\n  public int read(){calls++;return 93;}\n  public int read(char[] b){calls++;return b==null?31:32;}\n }\n static String trace=\"\";\n static final RuntimeException BOOM=new RuntimeException(\"boom\");\n static lib.Base<String> shared=new lib.Base<String>(66);\n static lib.Base<String> receiver(){trace+=\"r\";return shared;}\n static lib.Base<String> absent(){trace+=\"r\";return null;}\n static lib.Base<String> failReceiver(){trace+=\"r\";throw BOOM;}\n static char[] argument(){trace+=\"a\";return new char[1];}\n static char[] failArgument(){trace+=\"a\";throw BOOM;}\n static int donor(lib.Holder<String> holder)throws Exception{return holder.value.read();}\n static <V> int captured(lib.Base<V> value)throws Exception{return value.read();}\n public static void main(String[]args)throws Exception{\n  lib.Base<String> base=new lib.Base<String>(65);lib.Base<?> wildcard=base;\n  lib.Holder<String> holder=new lib.Holder<String>(base);\n  System.out.println(\"decl:\"+base.read()+\":\"+donor(holder)+\":\"+captured(base)+\":\"+wildcard.read()+\":\"+base.calls);\n  Override<String> override=new Override<String>();lib.Base<String> view=override;\n  System.out.println(\"override:\"+view.read()+\":\"+view.read((char[])null)+\":\"+override.calls+\":\"+view.read(7L));\n  trace=\"\";int count=receiver().read(argument());System.out.println(\"order:\"+trace+\":\"+count+\":\"+shared.calls);\n  trace=\"\";boolean exact=false;try{receiver().read(failArgument());}catch(RuntimeException e){exact=e==BOOM;}System.out.println(\"arg:\"+trace+\":\"+exact+\":\"+shared.calls);\n  trace=\"\";exact=false;try{failReceiver().read(argument());}catch(RuntimeException e){exact=e==BOOM;}System.out.println(\"recv:\"+trace+\":\"+exact+\":\"+shared.calls);\n  trace=\"\";boolean nil=false;try{absent().read(argument());}catch(NullPointerException e){nil=true;}System.out.println(\"null:\"+trace+\":\"+nil+\":\"+shared.calls);\n  System.out.println(\"foreign:\"+new foreign.Reader<String>().read());\n }\n}\n",
	}
}

func TestReaderReadDeclaredOriginDonorOverrideAndOrderJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, readerDeclaredOriginGuardWorkflow(), "readorigin.Main", []campaignCompilerProjectObservation{
		{name: "first", stdout: "decl:65:65:65:65:4\noverride:93:31:2:overload:7\norder:ra:1:1\narg:ra:true:1\nrecv:r:true:1\nnull:ra:true:1\nforeign:88\n"},
		{name: "second", stdout: "decl:65:65:65:65:4\noverride:93:31:2:overload:7\norder:ra:1:1\narg:ra:true:1\nrecv:r:true:1\nnull:ra:true:1\nforeign:88\n"},
		{name: "third", stdout: "decl:65:65:65:65:4\noverride:93:31:2:overload:7\norder:ra:1:1\narg:ra:true:1\nrecv:r:true:1\nnull:ra:true:1\nforeign:88\n"},
	})
}

func TestReaderReadDeclaredOriginProofBoundaries(t *testing.T) {
	sources := map[string]string{
		"lib/Base.java":       `package lib;public abstract class Base<T> extends java.io.Reader{}`,
		"foreign/Reader.java": `package foreign;public class Reader<T>{public int read(){return 88;}}`,
		"probe/Main.java": `package probe;
 class java{static class io{static class Reader{}}}
 class Prefix<T> extends java.io.Reader{public int read(){return 7;}}
 class CycleA<T> extends CycleB<T>{}class CycleB<T> extends CycleA<T>{}
 public class Main {
  <Reader> int captured(lib.Base<Reader> value)throws Exception{return value.read();}
  <T extends lib.Base<String>> int binder(T value)throws Exception{return value.read();}
  int nativeOnly(java.io.Reader value)throws Exception{return value.read();}
  int array(lib.Base<String>[] value){return value.read();}
  int unknown(Missing value){return value.read();}
  int foreign(foreign.Reader<String> value){return value.read();}
  int prefix(Prefix<String> value){return value.read();}
  int cyclic(CycleA<String> value){return value.read();}
 }`,
	}
	ctx := sourceGenericViewDemandTestContext(t, sources)
	main := findQualifiedSourceClass("probe.Main")
	ctx = classScopeCtx(main, ctx)
	for _, method := range main.Methods {
		if method.Constructor {
			continue
		}
		t.Run(method.OriginalName, func(t *testing.T) {
			local := ctx
			local.localScope = method
			node := findNode(method.DeclarationNode, "method_invocation")
			if node == nil {
				t.Fatal("fixture has no read occurrence")
			}
			owner, proved := readerSourceInvocationOwner(node.ChildByFieldName("object"), local, local.currentFile.Source)
			want := method.OriginalName == "captured"
			if proved != want || proved && owner != findQualifiedSourceClass("lib.Base") {
				t.Fatalf("actual receiver declaration proof=%v owner=%p want=%v", proved, owner, want)
			}
			if want {
				var origin inferredJavaTypeOrigin
				inferExprJavaType(node.ChildByFieldName("object"), local, local.currentFile.Source, &origin)
				if len(origin.arguments) != 1 || origin.arguments[0].parameter == nil || origin.arguments[0].parameter != method.TypeParameters[0].Declaration {
					t.Fatal("real captured type-parameter identity was not retained")
				}
				t.Logf("legal nominal Base<captured Reader>: parameter=%p actual method declaration=%p", origin.arguments[0].parameter, method.TypeParameters[0].Declaration)
			}
		})
	}
	var captured *sitter.Node
	for _, method := range main.Methods {
		if method.OriginalName == "captured" {
			ctx.localScope = method
			captured = findNode(method.DeclarationNode, "method_invocation")
		}
	}
	object := captured.ChildByFieldName("object")
	t.Run("missingFile", func(t *testing.T) {
		local := ctx
		local.currentFile = nil
		if _, ok := readerSourceInvocationOwner(object, local, ctx.currentFile.Source); ok {
			t.Fatal("missing file admitted")
		}
	})
	t.Run("wrongFile", func(t *testing.T) {
		local := ctx
		local.currentFile = findFileScopeForClassScope(findQualifiedSourceClass("lib.Base"), ctx)
		if _, ok := readerSourceInvocationOwner(object, local, ctx.currentFile.Source); ok {
			t.Fatal("foreign occurrence file admitted")
		}
	})
	t.Run("detachedOccurrence", func(t *testing.T) {
		twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), ctx.currentFile.Source...)}
		if err := twin.ParseAST(); err != nil {
			t.Fatal(err)
		}
		twin.ParseSymbols()
		other := findNode(twin.Ast, "method_invocation").ChildByFieldName("object")
		if _, ok := readerSourceInvocationOwner(other, ctx, ctx.currentFile.Source); ok {
			t.Fatal("identical-span detached occurrence admitted")
		}
	})
	t.Run("detachedCapturedMethod", func(t *testing.T) {
		twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), ctx.currentFile.Source...)}
		if err := twin.ParseAST(); err != nil {
			t.Fatal(err)
		}
		twin.ParseSymbols()
		local := ctx
		for _, top := range twin.Symbols.TopLevelClasses {
			if top.Class.OriginalName == "Main" {
				for _, method := range top.Methods {
					if method.OriginalName == "captured" {
						local.localScope = method
					}
				}
			}
		}
		if _, ok := readerSourceInvocationOwner(object, local, ctx.currentFile.Source); ok {
			t.Fatal("detached captured parameter declaration admitted")
		}
	})
	t.Run("detachedReceiverDeclaration", func(t *testing.T) {
		base := findQualifiedSourceClass("lib.Base")
		file := findFileScopeForClassScope(base, ctx)
		twin := parsing.SourceFile{Name: "TwinBase.java", Source: append([]byte(nil), file.Source...)}
		if err := twin.ParseAST(); err != nil {
			t.Fatal(err)
		}
		twin.ParseSymbols()
		saved := base.Class
		duplicate := *base.Class
		duplicate.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
		base.Class = &duplicate
		defer func() { base.Class = saved }()
		if _, ok := readerSourceInvocationOwner(object, ctx, ctx.currentFile.Source); ok {
			t.Fatal("detached source receiver declaration admitted")
		}
	})
}
