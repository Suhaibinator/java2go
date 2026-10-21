package transpiler

import (
	"go/ast"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func reader132ActualLateScope(t *testing.T) (Ctx, *symbol.ClassScope) {
	t.Helper()
	h := setupParseHelper(t, reader132OriginalJava)
	ctx := h.Ctx.Clone()
	ctx.className = ctx.currentClass.Class.Name
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls, ctx.anonClassCounter = &declarations, &counter
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.executionContextName = "__java2goExecution"
	for _, name := range []string{"check", "main"} {
		ctx.localScope = ctx.currentClass.FindMethodByName(name, nil)
		if ctx.localScope == nil {
			t.Fatal("original executable missing", name)
		}
		ParseStmt(ctx.localScope.DeclarationNode.ChildByFieldName("body"), h.File.Source, ctx)
	}
	for _, info := range ctx.anonymousClasses {
		if info != nil && info.structName == "MainAnon2" {
			return ctx, info.scope
		}
	}
	t.Fatal("actual second registered source scope missing")
	return Ctx{}, nil
}

func TestIntrinsicLexicalDeclaringFileGuards(t *testing.T) {
	ctx, scope := reader132ActualLateScope(t)
	file := ctx.currentFile
	t.Run("actual_registry_and_original_tree", func(t *testing.T) {
		if findFileScopeForClassScope(scope, ctx) != nil || !intrinsicLexicalDeclaringFile(scope, file, ctx) {
			t.Fatal("actual late anonymous scope certificate missing")
		}
		for i := 0; i < 3; i++ {
			if !intrinsicLexicalDeclaringFile(scope, file, ctx) || ctx.currentFile != file {
				t.Fatal("repeat query changed source context")
			}
		}
	})
	t.Run("nil", func(t *testing.T) {
		if intrinsicLexicalDeclaringFile(nil, file, ctx) || intrinsicLexicalDeclaringFile(scope, nil, ctx) || intrinsicLexicalDeclaringFile(&symbol.ClassScope{}, file, ctx) {
			t.Fatal("nil evidence accepted")
		}
	})
	t.Run("wrong_file", func(t *testing.T) {
		copyFile := *file
		if intrinsicLexicalDeclaringFile(scope, &copyFile, ctx) {
			t.Fatal("same source bytes in different file accepted")
		}
		wrong := ctx.Clone()
		wrong.currentFile = &copyFile
		if intrinsicLexicalDeclaringFile(scope, file, wrong) || intrinsicLexicalDeclaringFile(scope, &copyFile, wrong) {
			t.Fatal("stale current file accepted")
		}
	})
	t.Run("absent_or_different_registry_pointer", func(t *testing.T) {
		missing := ctx.Clone()
		missing.anonymousClasses = nil
		if intrinsicLexicalDeclaringFile(scope, file, missing) {
			t.Fatal("missing registry accepted")
		}
		copyScope := *scope
		if intrinsicLexicalDeclaringFile(&copyScope, file, ctx) {
			t.Fatal("copied scope with same name/declaration accepted")
		}
	})
	t.Run("detached_equal_span_tree", func(t *testing.T) {
		other := parsing.SourceFile{Name: "Other.java", Source: []byte(reader132OriginalJava)}
		if err := other.ParseAST(); err != nil {
			t.Fatal(err)
		}
		invocation := reader132NativeInvocation(other.Ast, other.Source)
		if invocation == nil {
			t.Fatal("detached control declaration missing")
		}
		creation := invocation
		for creation != nil && creation.Type() != "object_creation_expression" {
			creation = creation.Parent()
		}
		copyScope, copyClass := *scope, *scope.Class
		copyClass.DeclarationNode = creation
		copyScope.Class = &copyClass
		key, ok := anonymousClassSourceKey(creation)
		if !ok {
			t.Fatal("detached control source key missing")
		}
		forged := ctx.Clone()
		forged.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{key: {scope: &copyScope}}
		if intrinsicLexicalDeclaringFile(&copyScope, file, forged) {
			t.Fatal("different AST with identical bytes/spans accepted")
		}
	})
	t.Run("missing_or_cyclic_enclosing", func(t *testing.T) {
		original := scope.Enclosing
		defer func() { scope.Enclosing = original }()
		scope.Enclosing = nil
		if intrinsicLexicalDeclaringFile(scope, file, ctx) {
			t.Fatal("detached source owner accepted")
		}
		scope.Enclosing = scope
		if intrinsicLexicalDeclaringFile(scope, file, ctx) {
			t.Fatal("cycle accepted")
		}
		scope.Enclosing = original
		parent := original.Enclosing
		defer func() { original.Enclosing = parent }()
		original.Enclosing = scope
		if intrinsicLexicalDeclaringFile(scope, file, ctx) {
			t.Fatal("cycle after genuine source owner accepted")
		}
	})
	t.Run("local_registry", func(t *testing.T) {
		h := setupParseHelper(t, `class LocalHost {static int read(){class Local {int value(){return ((java.lang.Integer)(Object)7).intValue();}} return new Local().value();}}`)
		local := h.Ctx.Clone()
		local.className = local.currentClass.Class.Name
		local.localScope = local.currentClass.FindMethodByName("read", nil)
		var declarations []ast.Decl
		local.hoistedDecls = &declarations
		local.localClasses = make(map[string]*localClassInfo)
		local.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
		hoistLocalClass(findNode(local.localScope.DeclarationNode, "class_declaration"), h.File.Source, local)
		info := local.localClasses["Local"]
		if info == nil || !intrinsicLexicalDeclaringFile(info.scope, local.currentFile, local) {
			t.Fatal("actual original local scope not certified")
		}
	})
}

func TestIntrinsicLexicalSourceAndBinderShields(t *testing.T) {
	for _, source := range []string{
		`package app;public class Main {int read(foreign.Integer value){return value.intValue();}}`,
		`package app;public class Main {static class Integer {int intValue(){return 19;}} int read(Integer value){return value.intValue();}}`,
		`package app;public class Main<Integer extends foreign.Integer> {int read(Integer value){return value.intValue();}}`,
		`package app;public class Main {<Integer extends foreign.Integer> int read(Integer value){return value.intValue();}}`,
		`package app;public class Main {static class java {static class lang {static class Integer {int intValue(){return 23;}}}} int read(java.lang.Integer value){return value.intValue();}}`,
	} {
		t.Run(source, func(t *testing.T) {
			ctx := ordinaryDemandTestContext(t, map[string]string{"app/Main.java": source, "foreign/Integer.java": `package foreign;public class Integer {public int intValue(){return 17;}}`}, "app/Main.java")
			ctx.localScope = ctx.currentClass.FindMethodByName("read", nil)
			invocation := reader132NativeInvocation(ctx.localScope.DeclarationNode, ctx.currentFile.Source)
			if invocation == nil {
				t.Fatal("original source invocation missing")
			}
			if _, rewritten := tryInstanceIntrinsic(invocation.ChildByFieldName("object"), "intValue", ctx.currentFile.Source, ctx); rewritten {
				t.Fatal("source/binder declaration acquired native numeric callback")
			}
		})
	}
}

func TestIntrinsicLexicalNumberAccessorControlsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectObservations(t, map[string]string{
		"pom.xml":                            "<project><groupId>probe</groupId><artifactId>accessor-controls</artifactId><version>1</version></project>\n",
		"src/main/java/foreign/Integer.java": "package foreign;\nclass Base {public int intValue(){return -1;}}\npublic class Integer extends Base {public int count;public int intValue(){count++;return 73;}}\n",
		"src/main/java/probe/Main.java":      "package probe;\nimport java.io.Reader;\nimport foreign.Integer;\npublic class Main {\n static int calls;\n static Object once(){calls++;return java.lang.Integer.valueOf(7);}\n static <Integer extends foreign.Integer> int method(Integer value){return value.intValue();}\n static class Holder<Integer extends foreign.Integer> {Integer value;Holder(Integer value){this.value=value;}int get(){return value.intValue();}}\n static String observe(){return new Reader(new Object()) {\n  public int read(char[] b,int o,int n){return -1;}\n  public void close(){}\n  String values(){\n   Object object=java.lang.Integer.valueOf(7);\n   String six=((java.lang.Integer)object).byteValue()+\":\"+((java.lang.Integer)object).shortValue()+\":\"+((java.lang.Integer)object).intValue()+\":\"+((java.lang.Integer)object).longValue()+\":\"+((java.lang.Integer)object).floatValue()+\":\"+((java.lang.Integer)object).doubleValue();\n   String wrappers=((java.lang.Byte)(Object)java.lang.Byte.valueOf((byte)1)).intValue()+\":\"+((java.lang.Short)(Object)java.lang.Short.valueOf((short)2)).intValue()+\":\"+((java.lang.Integer)(Object)java.lang.Integer.valueOf(3)).intValue()+\":\"+((java.lang.Long)(Object)java.lang.Long.valueOf(4)).intValue()+\":\"+((java.lang.Float)(Object)java.lang.Float.valueOf(5.5f)).intValue()+\":\"+((java.lang.Double)(Object)java.lang.Double.valueOf(6.5)).intValue();\n   java.lang.Number wide=java.lang.Integer.valueOf(9);\n   String number=wide.byteValue()+\":\"+wide.shortValue()+\":\"+wide.intValue()+\":\"+wide.longValue()+\":\"+wide.floatValue()+\":\"+wide.doubleValue();\n   Integer source=new Integer();\n   String sourceValues=source.intValue()+\":\"+method(source)+\":\"+new Holder<Integer>(source).get()+\":\"+source.count;\n   boolean nullRejected=false,castRejected=false,arrayRejected=false;\n   try {Object absent=null;int ignored=((java.lang.Integer)absent).intValue();}catch(NullPointerException expected){nullRejected=true;}\n   try {int ignored=((java.lang.Integer)new Object()).intValue();}catch(ClassCastException expected){castRejected=true;}\n   int onceValue=((java.lang.Integer)once()).intValue();\n   java.lang.Integer[] nativeArray={java.lang.Integer.valueOf(11)};\n   Integer[] sourceArray={source};Object[] erased=sourceArray;\n   try {erased[0]=java.lang.Integer.valueOf(12);}catch(ArrayStoreException expected){arrayRejected=true;}\n   return six+\"\\n\"+wrappers+\"\\n\"+number+\"\\n\"+sourceValues+\"\\n\"+nullRejected+\":\"+castRejected+\":\"+onceValue+\":\"+calls+\":\"+nativeArray[0].intValue()+\":\"+arrayRejected+\":\"+(sourceArray[0]==source)+\"\\n\";\n  }\n }.values();}\n public static void main(String[] args){System.out.print(observe());}\n}\n",
	}, "probe.Main", []campaignCompilerProjectObservation{
		{name: "first", stdout: "7:7:7:7:7.0:7.0\n1:2:3:4:5:6\n9:9:9:9:9.0:9.0\n73:73:73:3\ntrue:true:7:1:11:true:true\n"},
		{name: "second", stdout: "7:7:7:7:7.0:7.0\n1:2:3:4:5:6\n9:9:9:9:9.0:9.0\n73:73:73:3\ntrue:true:7:1:11:true:true\n"},
		{name: "third", stdout: "7:7:7:7:7.0:7.0\n1:2:3:4:5:6\n9:9:9:9:9.0:9.0\n73:73:73:3\ntrue:true:7:1:11:true:true\n"},
	})
}
