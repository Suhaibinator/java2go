package transpiler

import (
	"bytes"
	"go/ast"
	"go/parser"
	"reflect"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

// The sources below are byte-identical to the independently JDK21-calibrated
// full Reader clone workflow. This parser regression supplements its unchanged
// whole-program acceptance; it does not replace the runtime oracle.
func canonicalRecursiveAliasWorkflowSources() map[string]string {
	return map[string]string{
		"cloneapp/Derived.java": "package cloneapp;\nimport clonebase.Base;\npublic class Derived<T> extends Base<T> {\n public int own=11;\n public Derived(){super();}\n protected int code(){return own;}\n}\n",
		"cloneapp/Main.java":    "package cloneapp;\nimport clonebase.Base;\nimport java.io.Reader;\npublic final class Main {\n static class Aliased extends Reader implements java.lang.Cloneable {\n  public Object ReaderBase=new Object();\n  public int own=4;\n  public int read(char[] b,int o,int n){if(n==0)return 0;b[o]=(char)own;return 1;}\n  public void close(){}\n  Object nativeLock(){return lock;}\n  void nativeLock(Object value){lock=value;}\n  Aliased copy()throws CloneNotSupportedException{return(Aliased)super.clone();}\n }\n static class AliasChild extends Aliased{public Object ReaderBase=new Object();}\n static class Denied extends Reader {\n  public int read(char[]b,int o,int n){return -1;}public void close(){}\n  Object copy()throws CloneNotSupportedException{return super.clone();}\n }\n public static void main(String[]args)throws Exception {\n  Derived<String> original=new Derived<String>();\n  int constructors=Base.constructed,initializers=Base.initialized;\n  Base<String> copy=original.copy();\n  Derived<String> derived=(Derived<String>)copy;\n  derived.own=23;copy.state(19);\n  System.out.println(\"hierarchy:\"+(copy!=original)+\":\"+(copy.getClass()==original.getClass())+\":\"+copy.shallow(original)+\":\"+(constructors==Base.constructed)+\":\"+(initializers==Base.initialized));\n  System.out.println(\"fields:\"+original.state()+\":\"+copy.state()+\":\"+original.read()+\":\"+copy.read()+\":\"+original.sourceCalls+\":\"+copy.sourceCalls);\n  System.out.println(\"default:\"+(copy.nativeLock()==original));\n  Object sentinel=new Object();original.nativeLock(sentinel);\n  Base<String> other=original.copy();other.nativeLock(null);\n  System.out.println(\"isolation:\"+(copy.nativeLock()==original)+\":\"+(original.nativeLock()==sentinel)+\":\"+(other.nativeLock()==null));\n  original.nativeLock(null);Base<String> nullable=original.copy();\n  System.out.println(\"nullable:\"+(nullable.nativeLock()==null)+\":\"+(constructors==Base.constructed));\n  copy.userCallback.run();\n  System.out.println(\"userClosure:\"+original.callbackCalls()+\":\"+copy.callbackCalls()+\":\"+(original.userCallback==copy.userCallback));\n  synchronized(original){System.out.println(\"monitorOriginal:\"+Thread.holdsLock(original)+\":\"+Thread.holdsLock(copy));}\n  synchronized(copy){System.out.println(\"monitorClone:\"+Thread.holdsLock(original)+\":\"+Thread.holdsLock(copy));}\n  Thread worker=new Thread(()->{try{Thread.currentThread().interrupt();int result=copy.read();System.out.println(\"callback:\"+result+\":\"+copy.workerSeen+\":\"+copy.interruptedSeen+\":\"+(original.sourceCalls==1));}catch(Exception failure){throw new RuntimeException(failure);}finally{Thread.interrupted();}});\n  worker.start();worker.join();\n  AliasChild first=new AliasChild();Aliased alias=first.copy();alias.own=9;\n  System.out.println(\"alias:\"+(alias!=first)+\":\"+(alias.nativeLock()==first)+\":\"+(alias.ReaderBase==((Aliased)first).ReaderBase)+\":\"+first.read()+\":\"+alias.read());\n  boolean denied=false;try{new Denied().copy();}catch(CloneNotSupportedException expected){denied=true;}\n  System.out.println(\"denied:\"+denied);\n }\n}\n",
		"clonebase/Base.java":   "package clonebase;\nimport java.io.Reader;\npublic class Base<T> extends Reader implements java.lang.Cloneable {\n public static int constructed;\n public static int initialized;\n private final Object shared = new Object();\n private final int[] values = new int[]{3};\n private final Base<T> initialSelf = this;\n private volatile int state = 7;\n private int callbackCalls;\n public int sourceCalls;\n public boolean workerSeen;\n public boolean interruptedSeen;\n private final Thread constructorThread = Thread.currentThread();\n public Runnable userCallback = () -> { callbackCalls++; };\n private final int field = ++initialized;\n public Base(){super();constructed++;}\n public Base(Object monitor){super(monitor);constructed++;}\n public int read(char[] buffer,int offset,int length) {\n  sourceCalls++;\n  workerSeen=Thread.currentThread()!=constructorThread;\n  interruptedSeen=Thread.currentThread().isInterrupted();\n  if(length==0)return 0;\n  buffer[offset]=(char)code();return 1;\n }\n protected int code(){return state;}\n public void close(){}\n public Base<T> copy()throws CloneNotSupportedException{return(Base<T>)super.clone();}\n public Object nativeLock(){return lock;}\n public void nativeLock(Object value){lock=value;}\n public boolean shallow(Base<T> original){return shared==original.shared&&values==original.values&&initialSelf==original&&field==original.field;}\n public void state(int value){state=value;}\n public int state(){return state;}\n public int callbackCalls(){return callbackCalls;}\n}\n",
	}
}

func canonicalRecursiveAliasPrint(t *testing.T, node any) string {
	t.Helper()
	var out bytes.Buffer
	if err := printer.Fprint(&out, token.NewFileSet(), node); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestCanonicalGenericRecursiveDispatchPhysicalSignatureEdges(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, canonicalRecursiveAliasWorkflowSources())
	base := findQualifiedSourceClass("clonebase.Base")
	if base == nil {
		t.Fatal("missing actual Base declaration")
	}
	ctx = classScopeCtx(base, ctx)
	if !canonicalGenericClass(base, ctx) || canonicalGenericFamily(base, ctx) == nil {
		t.Fatal("full workflow did not select canonical physical family")
	}
	raw := availableCanonicalGenericName(base.Class.Name + "Java2goErased")
	dispatch := generateClassDispatchInterface(ctx)
	t.Logf("actual declared dispatch: %s", canonicalRecursiveAliasPrint(t, dispatch))
	file, ok := ParseNode(ctx.currentFile.ParsedRoot(), ctx.currentFile.Source, ctx).(*ast.File)
	if !ok {
		t.Fatal("actual registered Java file failed emission")
	}
	publicAlias := false
	signatureEdges := 0
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.Name.Name == base.Class.Name {
					publicAlias = ts.Assign.IsValid() && canonicalRecursiveAliasPrint(t, ts.Type) == raw
				}
				iface, ok := ts.Type.(*ast.InterfaceType)
				if !ok || ts.Name.Name != availableCanonicalGenericName(classDispatchTypeName(base)+"Java2goErased") {
					continue
				}
				for _, field := range iface.Methods.List {
					if len(field.Names) != 1 {
						continue
					}
					name := field.Names[0].Name
					if name != "Copy" && name != "Shallow" {
						continue
					}
					ft := field.Type.(*ast.FuncType)
					edge := ft.Results.List[0].Type
					if name == "Shallow" {
						edge = ft.Params.List[0].Type
					}
					signatureEdges++
					if got := canonicalRecursiveAliasPrint(t, edge); got != "*"+raw {
						t.Errorf("dispatch %s retains recursive public alias: got %s want *%s", name, got, raw)
					}
				}
			}
		case *ast.FuncDecl:
			if node.Recv == nil || len(node.Recv.List) != 1 || canonicalRecursiveAliasPrint(t, node.Recv.List[0].Type) != "*"+raw {
				continue
			}
			switch node.Name.Name {
			case "Copy", "CopyJava2goExecution", "Java2goReflectFieldGetinitialSelfJava2goExecution":
				signatureEdges++
				got := canonicalRecursiveAliasPrint(t, node.Type.Results.List[0].Type)
				if got != "*"+raw {
					t.Errorf("receiver %s retains recursive public alias: got %s want *%s", node.Name.Name, got, raw)
				}
			case "Shallow", "ShallowJava2goExecution", "Java2goReflectFieldSetinitialSelfJava2goExecution":
				signatureEdges++
				got := canonicalRecursiveAliasPrint(t, node.Type.Params.List[len(node.Type.Params.List)-1].Type)
				if got != "*"+raw {
					t.Errorf("receiver %s retains recursive public alias: got %s want *%s", node.Name.Name, got, raw)
				}
			}
		}
	}
	if signatureEdges != 8 {
		t.Fatalf("actual signature coverage %d want8", signatureEdges)
	}
	if !publicAlias {
		t.Fatal("public generic alias identity must remain intact")
	}
}

func TestCanonicalGenericRecursiveDispatchOrdinaryGenericControl(t *testing.T) {
	out := normalizeSpaces(renderGoFileFromJava(t, `public class Plain<T>{Plain<T> copy(){return this;}void accept(Plain<T> value){}}`))
	if strings.Contains(out, "PlainJava2goErased") || !strings.Contains(out, "copy() *Plain[T]") || !strings.Contains(out, "accept(value *Plain[T])") {
		t.Fatal("ordinary invariant generic signature changed:", out)
	}
}

func canonicalRecursiveSignatureType(t *testing.T, text string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(text)
	if err != nil {
		t.Fatal(err)
	}
	return expr
}

func TestCanonicalGenericRecursiveDispatchCertificateShields(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, canonicalRecursiveAliasWorkflowSources())
	base := findQualifiedSourceClass("clonebase.Base")
	derived := findQualifiedSourceClass("cloneapp.Derived")
	ctx = classScopeCtx(base, ctx)
	ctx.genericFamilies = &genericFamilyAnalysis{}
	if canonicalGenericFamily(base, ctx) == nil || canonicalGenericFamily(derived, ctx) == nil {
		t.Fatal("actual workflow canonical family absent")
	}
	ctx.importAliases = map[string]string{"cloneapp": "models"}
	ctx.usedImports = map[string]bool{"cloneapp": true}
	raw := availableCanonicalGenericName(base.Class.Name + "Java2goErased")
	derivedRaw := availableCanonicalGenericName(derived.Class.Name + "Java2goErased")
	check := func(t *testing.T, local Ctx, source, want string) {
		t.Helper()
		signature := &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{{Type: canonicalRecursiveSignatureType(t, source)}}}}
		aliases := map[string]string{}
		for k, v := range local.importAliases {
			aliases[k] = v
		}
		imports := map[string]bool{}
		for k, v := range local.usedImports {
			imports[k] = v
		}
		canonicalGenericPhysicalSignature(signature, local)
		if got := canonicalRecursiveAliasPrint(t, signature.Params.List[0].Type); got != want {
			t.Fatalf("signature=%s got=%s want=%s", source, got, want)
		}
		if !reflect.DeepEqual(aliases, local.importAliases) || !reflect.DeepEqual(imports, local.usedImports) {
			t.Fatal("signature proof mutated imports")
		}
	}
	for _, test := range []struct{ name, source, want string }{
		{"samePackage", "*Base[any]", "*" + raw},
		{"actualImportedNamespace", "*models.Derived[any]", "*models." + derivedRaw},
		{"unknownQualifier", "*foreign.Derived[any]", "*foreign.Derived[any]"},
		{"wrongPackageOwner", "*models.Base[any]", "*models.Base[any]"},
		{"wrongArity", "*Base[any, any]", "*Base[any, any]"},
		{"nonPointer", "Base[any]", "Base[any]"},
		{"unknownAlias", "*Missing[any]", "*Missing[any]"},
		{"nativeLookalike", "*stdjava.StringBuilder[any]", "*stdjava.StringBuilder[any]"},
		{"nestedFunction", "func(*Base[any]) *models.Derived[any]", "func(*" + raw + ") *models." + derivedRaw},
	} {
		t.Run(test.name, func(t *testing.T) { check(t, ctx, test.source, test.want) })
	}
	t.Run("missingImportCertificate", func(t *testing.T) {
		local := ctx
		local.importAliases = map[string]string{}
		check(t, local, "*models.Derived[any]", "*models.Derived[any]")
	})
	t.Run("nilFile", func(t *testing.T) { local := ctx; local.currentFile = nil; check(t, local, "*Base[any]", "*Base[any]") })
	t.Run("wrongCurrentFile", func(t *testing.T) {
		local := ctx
		local.currentFile = findFileScopeForClassScope(derived, ctx)
		check(t, local, "*Base[any]", "*Base[any]")
	})
	for _, binder := range []string{"Base", "models"} {
		t.Run("methodBinder_"+binder, func(t *testing.T) {
			source := "*Base[any]"
			if binder == "models" {
				source = "*models.Derived[any]"
			}
			signature := &ast.FuncType{TypeParams: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(binder)}, Type: ast.NewIdent("any")}}}, Params: &ast.FieldList{List: []*ast.Field{{Type: canonicalRecursiveSignatureType(t, source)}}}}
			before := canonicalRecursiveAliasPrint(t, signature)
			canonicalGenericPhysicalSignature(signature, ctx)
			if canonicalRecursiveAliasPrint(t, signature) != before {
				t.Fatal("method binder was reinterpreted as class alias or package")
			}
		})
	}
	t.Run("detachedIdenticalTree", func(t *testing.T) {
		twin := parsing.SourceFile{Name: "Twin.java", Source: append([]byte(nil), ctx.currentFile.Source...)}
		if err := twin.ParseAST(); err != nil {
			t.Fatal(err)
		}
		twin.ParseSymbols()
		saved := base.Class
		copy := *base.Class
		copy.DeclarationNode = twin.Symbols.BaseClass.Class.DeclarationNode
		base.Class = &copy
		defer func() { base.Class = saved }()
		check(t, ctx, "*Base[any]", "*Base[any]")
	})
	t.Run("ambiguousRegisteredAlias", func(t *testing.T) {
		original := ctx.currentFile.TopLevelClasses
		duplicate := *base
		ctx.currentFile.TopLevelClasses = append(append([]*symbol.ClassScope(nil), original...), &duplicate)
		defer func() { ctx.currentFile.TopLevelClasses = original }()
		check(t, ctx, "*Base[any]", "*Base[any]")
	})
	t.Run("bodyIsNotSignature", func(t *testing.T) {
		expression := canonicalRecursiveSignatureType(t, "Base[any](nil)")
		before := canonicalRecursiveAliasPrint(t, expression)
		canonicalGenericPhysicalSignature(expression, ctx)
		if canonicalRecursiveAliasPrint(t, expression) != before {
			t.Fatal("value expression changed")
		}
	})
	t.Run("allocatedRawCollision", func(t *testing.T) {
		collision := parsing.SourceFile{Name: "collision/BaseJava2goErased.java", Source: []byte("package collision;public class BaseJava2goErased{}")}
		if err := collision.ParseAST(); err != nil {
			t.Fatal(err)
		}
		collision.ParseSymbols()
		symbol.AddSymbolsToPackage(collision.Symbols)
		defer delete(symbol.GlobalScope.Packages, "collision")
		allocated := availableCanonicalGenericName(base.Class.Name + "Java2goErased")
		if allocated == raw {
			t.Fatal("real source declaration did not reserve raw name")
		}
		check(t, ctx, "*Base[any]", "*"+allocated)
	})
}
