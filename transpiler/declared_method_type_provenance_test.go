package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestDeclaredMethodProvenanceSourceOverload(t *testing.T) {
	h := setupParseHelper(t, `package p;class java{static class lang{static class Object{}}}class Value{int choose(Object x){return 1;}int choose(java.lang.Object x){return 2;}int run(java.lang.Object x){return choose(x);}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Value")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
	call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	selected := findBestMethodInHierarchy(ctx.currentClass, "choose", call.ChildByFieldName("arguments"), true, true, ctx, h.File.Source)
	if selected == nil || selected.def.Parameters[0].OriginalType != "java.lang.Object" {
		t.Fatal("source argument selected canonical Object overload")
	}
}

func TestDeclaredMethodProvenanceCanonicalDescriptor(t *testing.T) {
	h := setupParseHelper(t, `package p;class java{static class lang{static class Object{}}}class Value{Object root(Object x){return x;}java.lang.Object source(java.lang.Object x){return x;}}`)
	scope := h.File.Symbols.FindClassScope("Value")
	ctx := h.Ctx.Clone()
	ctx.currentClass = scope
	for _, name := range []string{"root", "source"} {
		method := scope.FindMethod().ByOriginalName(name)[0]
		params, result, ok := sourceReflectionMethodSignature(scope, method, ctx)
		if !ok || len(params) != 1 {
			t.Fatal("descriptor absent")
		}
		for _, value := range append(params, result) {
			var b bytes.Buffer
			if err := format.Node(&b, token.NewFileSet(), value); err != nil {
				t.Fatal(err)
			}
			got := b.String()
			if name == "root" && !strings.Contains(got, "ObjectTypeID") {
				t.Fatalf("canonical root descriptor rebound: %s", got)
			}
			if name == "source" && !strings.Contains(got, "p.java$lang$Object") {
				t.Fatalf("source descriptor lost declaration: %s", got)
			}
		}
	}
}

func TestDeclaredMethodProvenanceRankAndResidualPrimitiveArray(t *testing.T) {
	h := setupParseHelper(t, `class Item{} class Box<T>{}`)
	p := symbol.NewTypeParam("U", nil)
	declared := func(text string) inferredJavaTypeOrigin {
		return declaredJavaTypeOrigin(symbol.JavaType{Original: text, TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"U": p.Declaration}}, h.Ctx)
	}
	candidates := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
	gatherMethodTypeOriginBindings(declared("U[]"), declared("int[][]"), map[*symbol.TypeParamDeclaration]bool{p.Declaration: true}, candidates)
	values := candidates[p.Declaration]
	if len(values) != 1 || values[0].arrayRank != 1 || values[0].javaType != "int[]" {
		t.Fatalf("residual primitive-array binder is legal reference: %#v", values)
	}
	if got := substituteJavaTypeOrigin(declared("U[]"), map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{p.Declaration: values[0]}); got.arrayRank != 2 {
		t.Fatal("substitution lost occurrence rank")
	}
	nested := substituteJavaTypeOrigin(declared("Box<U[][]>"), map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{p.Declaration: values[0]})
	if len(nested.arguments) != 1 || nested.arguments[0].arrayRank != 3 {
		t.Fatal("nested substitution lost rank")
	}
	if sameJavaTypeOrigin(declared("Item[]"), declared("Item[][]")) {
		t.Fatal("source declaration identity concealed unequal ranks")
	}
	candidates = map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
	gatherMethodTypeOriginBindings(declared("U[]"), declared("int[]"), map[*symbol.TypeParamDeclaration]bool{p.Declaration: true}, candidates)
	if len(candidates[p.Declaration]) != 0 {
		t.Fatal("scalar primitive acquired generic binder")
	}
}

func TestDeclaredMethodProvenanceCanonicalAndSourceWrapper(t *testing.T) {
	h := setupParseHelper(t, `class Integer{} class Value{}`)
	primitive := inferredJavaTypeOrigin{javaType: "int"}
	source := declaredJavaTypeOrigin(symbol.JavaType{Original: "Integer"}, h.Ctx)
	if _, _, applies, handled := declaredOriginConversion(primitive, source, h.Ctx); !handled || applies {
		t.Fatal("boxed int acquired source Integer")
	}
	canonical := intrinsicJavaTypeOrigin("java.lang.Integer")
	if _, _, applies, handled := declaredOriginConversion(primitive, canonical, h.Ctx); !handled || !applies {
		t.Fatal("canonical boxing refused")
	}
	if _, _, applies, handled := declaredOriginConversion(source, primitive, h.Ctx); !handled || applies {
		t.Fatal("source wrapper acquired unboxing")
	}
}

func TestDeclaredMethodProvenanceAtomicBranches(t *testing.T) {
	var promoted inferredJavaTypeOrigin
	recordJoinedJavaTypeOrigin([]*inferredJavaTypeOrigin{&promoted}, "long", []inferredJavaTypeOrigin{{javaType: "int"}, {javaType: "long"}}, Ctx{})
	if promoted.unresolved || promoted.javaType != "long" || promoted.externalName != "" || promoted.nominalScope != nil {
		t.Fatal("primitive promotion invented or lost nominal proof")
	}
	var nullOnly inferredJavaTypeOrigin
	recordJoinedJavaTypeOrigin([]*inferredJavaTypeOrigin{&nullOnly}, ternaryNullJavaType, []inferredJavaTypeOrigin{{javaType: "null"}, {javaType: "null"}}, Ctx{})
	if nullOnly.unresolved || nullOnly.javaType != ternaryNullJavaType || nullOnly.nominalScope != nil || nullOnly.externalName != "" {
		t.Fatal("null-only join invented or lost nominal proof")
	}
	h := setupParseHelper(t, `class java{static class lang{class String{} class Class{}}}class Value{Object run(boolean b){return b?"left":"right";}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Value")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
	conditional := findNode(ctx.localScope.DeclarationNode, "ternary_expression")
	_, left, right := ternaryExpressionParts(conditional)
	plan, ok := canonicalStandaloneBranchPlan([]*sitter.Node{left, right}, ctx, h.File.Source, true)
	if !ok || plan.origin.externalName != "java.lang.String" {
		t.Fatal("literal physical ABI borrowed source TYPE prefix")
	}
	if _, ok := plan.physical.(*ast.StarExpr); !ok {
		t.Fatal("canonical String lost reference ABI")
	}
	ctx.genericInterfaceMethodBody = &genericInterfaceMethodBodyABI{}
	if _, ok := canonicalStandaloneBranchPlan([]*sitter.Node{left, right}, ctx, h.File.Source, true); ok {
		t.Fatal("new carrier displaced erased interface policy")
	}
}

func TestDeclaredMethodProvenanceHiddenMemberSlots(t *testing.T) {
	h := setupParseHelper(t, `class Owner<T>{class Cell<U>{U value;} <T> T read(Cell<T> input){return input.value;}}`)
	scope := h.File.Symbols.FindClassScope("Owner")
	method := scope.FindMethod().ByOriginalName("read")[0]
	ctx := classScopeCtx(scope, h.Ctx)
	ctx.localScope = method
	origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(method.Parameters[0]), ctx)
	if origin.unresolved || len(origin.arguments) != 2 || origin.arguments[0].parameter != scope.TypeParameters[0].Declaration || origin.arguments[1].parameter != method.TypeParameters[0].Declaration {
		t.Fatalf("member slots rebound: %+v", origin)
	}
}

func TestDeclaredMethodProvenanceCanonicalUnboxingWidening(t *testing.T) {
	h := setupParseHelper(t, `class Integer{}class Value{}`)
	actual := intrinsicJavaTypeOrigin("java.lang.Integer")
	if _, _, applies, handled := declaredOriginConversion(actual, inferredJavaTypeOrigin{javaType: "long"}, h.Ctx); !handled || !applies {
		t.Fatal("canonical unbox-and-widen rejected")
	}
	if _, _, applies, handled := declaredOriginConversion(actual, inferredJavaTypeOrigin{javaType: "short"}, h.Ctx); !handled || applies {
		t.Fatal("unboxing admitted narrowing")
	}
	if _, _, applies, handled := declaredOriginConversion(actual, intrinsicJavaTypeOrigin("java.lang.Number"), h.Ctx); !handled || !applies {
		t.Fatal("canonical wrapper Number ancestry rejected")
	}
}

func TestDeclaredMethodProvenanceUnresolvedAndDeclarationGuards(t *testing.T) {
	h := setupParseHelper(t, `class java{class lang{class Object{}}} class Value{}`)
	canonical := intrinsicJavaTypeOrigin("java.lang.Object")
	if _, _, applies, handled := declaredOriginConversion(inferredJavaTypeOrigin{javaType: "java.lang.Object", unresolved: true}, canonical, h.Ctx); !handled || applies {
		t.Fatal("lost provenance retried caller type text")
	}
	p := symbol.NewTypeParam("java", nil)
	binder := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.Missing", TypeParameterBindings: map[string]*symbol.TypeParamDeclaration{"java": p.Declaration}}, h.Ctx)
	if binder.externalName != "" || !binder.unresolved {
		t.Fatal("type-variable first segment borrowed package identity")
	}
	foreign := h.Ctx.Clone()
	file := *foreign.currentFile
	file.Imports = map[string]string{"String": "foreign"}
	foreign.currentFile = &file
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: "String"}, foreign)
	if origin.externalName == "java.lang.String" {
		t.Fatal("explicit foreign import borrowed canonical String")
	}
}

func TestDeclaredMethodProvenanceResidualArrayAndCanonicalInterfaceJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>array-reference-origins</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api;
public class Routes {
 public static String trace="";
 public static int objects(Object[] value){trace+="A";return value==null?105:101;}
 public static int objects(Object value){trace+="O";return value==null?205:201;}
 public static int clones(Cloneable[] value){trace+="C";return value==null?115:111;}
 public static int clones(Object value){trace+="X";return value==null?215:211;}
 public static int serials(java.io.Serializable[] value){trace+="S";return value==null?125:121;}
 public static int serials(Object value){trace+="Y";return value==null?225:221;}
 public static int task(Runnable value){trace+="N";value.run();return 31;}
 public static int task(Object value){trace+="F";return 39;}
 public static int shadow(model.Runnable value){trace+="D";value.run();return 41;}
 public static int shadow(Object value){trace+="W";return 49;}
}
`,
		"src/main/java/app/Main.java": `package app;
import api.Routes;
public class Main {
 public static class Task implements Runnable {public void run(){Routes.trace+="R";}}
 public static class ShadowTask implements model.Runnable {public void run(){Routes.trace+="Q";}}
 public static void main(String[] args){
  String[][] strings=new String[2][];
  int[][] integers=new int[3][];
  int[] scalar=new int[4];
  String[][] absent=null;
  System.out.println(Routes.objects(strings));
  System.out.println(Routes.objects(integers));
  System.out.println(Routes.objects(scalar));
  System.out.println(Routes.objects(absent));
  System.out.println(Routes.clones(strings));
  System.out.println(Routes.clones(integers));
  System.out.println(Routes.clones(scalar));
  System.out.println(Routes.serials(strings));
  System.out.println(Routes.serials(integers));
  System.out.println(Routes.serials(scalar));
  Task task=new Task(); ShadowTask shadow=new ShadowTask();
  System.out.println(Routes.task(task));
  System.out.println(Routes.task(shadow));
  System.out.println(Routes.shadow(shadow));
  System.out.println(Routes.shadow(task));
  System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/Runnable.java": `package model; public interface Runnable { void run(); }
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "101\n101\n201\n105\n111\n111\n211\n121\n121\n221\n31\n39\n41\n49\nAAOACCXSSYNRFDQW\n", stderr: ""},
		{name: "repeat2", stdout: "101\n101\n201\n105\n111\n111\n211\n121\n121\n221\n31\n39\n41\n49\nAAOACCXSSYNRFDQW\n", stderr: ""},
		{name: "repeat3", stdout: "101\n101\n201\n105\n111\n111\n211\n121\n121\n221\n31\n39\n41\n49\nAAOACCXSSYNRFDQW\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCallerBinderJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>callerbinder</artifactId><version>1</version></project>
`,
		"src/main/java/Factory.java": `public class Factory { public static Holder<T2> held=new Holder<T2>(); public static Holder<T2> make(){return held;} public static Holder<T2> absent(){return null;} }
`,
		"src/main/java/Holder.java": `public class Holder<T> implements Comparable<T> { public int compareTo(T value){return 1;} }
`,
		"src/main/java/Main.java": `public class Main { public static <T2> void invoke(){System.out.println(Routes.pick(Factory.make()));System.out.println(Routes.pick(Factory.held));System.out.println(Routes.pick(Factory.absent()));System.out.println(Routes.trace);} public static void main(String[] args){Main.<String>invoke();} }
`,
		"src/main/java/Routes.java": `public class Routes { public static String trace=""; public static int pick(Comparable<T2> value){trace+="C";return value==null?35:31;} public static int pick(Object value){trace+="F";return value==null?45:39;} }
`,
		"src/main/java/T2.java": `public class T2 { public int value(){return 7;} }
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "31\n31\n35\nCCC\n", stderr: ""},
		{name: "repeat2", stdout: "31\n31\n35\nCCC\n", stderr: ""},
		{name: "repeat3", stdout: "31\n31\n35\nCCC\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCallerCanonicalJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>callercanonical</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api;
public class Routes {
 public static String trace="";
 public static int pick(Runnable value){trace+="N";value.run();return 31;}
 public static int pick(Object value){trace+="F";return 39;}
 public static int source(app.Main.java.lang.Runnable value){trace+="D";value.run();return 41;}
 public static int source(Object value){trace+="W";return 49;}
}
`,
		"src/main/java/app/Main.java": `package app;
import api.Routes;
import model.Task;
import model.ShadowTask;
public class Main {
 public static class java { public static class lang { public interface Runnable {void run();} } }
 public static void main(String[] args){
  Task task=new Task(); ShadowTask shadow=new ShadowTask();
  System.out.println(Routes.pick(task));
  System.out.println(Routes.pick(shadow));
  System.out.println(Routes.source(shadow));
  System.out.println(Routes.source(task));
  System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/ShadowTask.java": `package model; public class ShadowTask implements app.Main.java.lang.Runnable { public void run(){api.Routes.trace+="Q";} }
`,
		"src/main/java/model/Task.java": `package model; public class Task implements Runnable { public void run(){api.Routes.trace+="R";} }
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "31\n39\n41\n49\nNRFDQW\n", stderr: ""},
		{name: "repeat2", stdout: "31\n39\n41\n49\nNRFDQW\n", stderr: ""},
		{name: "repeat3", stdout: "31\n39\n41\n49\nNRFDQW\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceAncestryEndpointArgumentsJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>callercanonical</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api;
public class Routes {
 public static String trace="";
 public static int pick(Runnable value){trace+="N";value.run();return 31;}
 public static int pick(Object value){trace+="F";return 39;}
 public static int source(app.Main.java.lang.Runnable value){trace+="D";value.run();return 41;}
 public static int source(Object value){trace+="W";return 49;}
 public static int compare(Comparable<Runnable> value){trace+="C";return 51;}
 public static int compare(Object value){trace+="X";return 59;}
}
`,
		"src/main/java/app/Main.java": `package app;
import api.Routes;
import model.Task;
import model.ShadowTask;
public class Main {
 public static class java { public static class lang { public interface Runnable {void run();} } }
 public static void main(String[] args){
  Task task=new Task(); ShadowTask shadow=new ShadowTask();
  System.out.println(Routes.pick(task));
  System.out.println(Routes.pick(shadow));
  System.out.println(Routes.source(shadow));
  System.out.println(Routes.source(task));
  System.out.println(Routes.compare(new model.CanonicalHolder()));
  System.out.println(Routes.compare(new model.SourceHolder()));
  System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/CanonicalHolder.java": `package model; public class CanonicalHolder implements Comparable<Runnable> {public int compareTo(Runnable value){return 1;}}
`,
		"src/main/java/model/ShadowTask.java": `package model; public class ShadowTask implements app.Main.java.lang.Runnable { public void run(){api.Routes.trace+="Q";} }
`,
		"src/main/java/model/SourceHolder.java": `package model; public class SourceHolder implements Comparable<app.Main.java.lang.Runnable> {public int compareTo(app.Main.java.lang.Runnable value){return 2;}}
`,
		"src/main/java/model/Task.java": `package model; public class Task implements Runnable { public void run(){api.Routes.trace+="R";} }
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "31\n39\n41\n49\n51\n59\nNRFDQWCX\n", stderr: ""},
		{name: "repeat2", stdout: "31\n39\n41\n49\n51\n59\nNRFDQWCX\n", stderr: ""},
		{name: "repeat3", stdout: "31\n39\n41\n49\n51\n59\nNRFDQWCX\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCornerDirectInvariantJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>directinvariant</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api; public class Routes { public static String trace=""; public static int pick(Comparable<Runnable> value){trace+=value==null?"c":"C";return value==null?35:31;} public static int pick(Object value){trace+=value==null?"o":"O";return value==null?49:39;} }
`,
		"src/main/java/app/Main.java": `package app;
import api.Routes;
import java.lang.Runnable;
public class Main {
 public static class java {public static class lang {public interface Runnable {void run();}}}
 public static Comparable<Runnable> rightField=new model.Right();
 public static Comparable<model.Runnable> wrongField=new model.Wrong();
 public static Comparable<model.Runnable> wrongResult(){return wrongField;}
 public static void main(String[] args){
  Comparable<Runnable> right=rightField; Comparable<model.Runnable> wrong=wrongResult();
  Comparable raw=wrong; Comparable<Runnable> absentRight=null; Comparable<model.Runnable> absentWrong=null;
  System.out.println(Routes.pick(right)); System.out.println(Routes.pick(wrong)); System.out.println(Routes.pick(raw));
  System.out.println(Routes.pick(absentRight)); System.out.println(Routes.pick(absentWrong));
  Comparable<Runnable> checked=(Comparable<Runnable>)(Object)wrong;
  System.out.println(Routes.pick(checked)); System.out.println(Routes.pick(rightField)); System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/Right.java": `package model; public class Right implements Comparable<java.lang.Runnable> {public int compareTo(java.lang.Runnable value){return 1;}}
`,
		"src/main/java/model/Runnable.java": `package model; public interface Runnable {void run();}
`,
		"src/main/java/model/Wrong.java": `package model; public class Wrong implements Comparable<model.Runnable> {public int compareTo(model.Runnable value){return 2;}}
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "31\n39\n31\n35\n49\n31\n31\nCOCcoCC\n", stderr: ""},
		{name: "repeat2", stdout: "31\n39\n31\n35\n49\n31\n31\nCOCcoCC\n", stderr: ""},
		{name: "repeat3", stdout: "31\n39\n31\n35\n49\n31\n31\nCOCcoCC\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCornerExceptionAncestorsJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>exceptionancestors</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api; public class Routes {public static String trace="";
 public static int thrown(Throwable value){trace+=value==null?"t":"T";return value==null?15:11;} public static int thrown(Object value){trace+="O";return 19;}
 public static int checked(Exception value){trace+="E";return 21;} public static int checked(Object value){trace+="F";return 29;}
 public static int runtime(RuntimeException value){trace+="R";return 31;} public static int runtime(Object value){trace+="G";return 39;}
 public static int source(model.Throwable value){trace+="S";return 41;} public static int source(Object value){trace+="O";return 49;}
}
`,
		"src/main/java/app/Main.java": `package app; import api.Routes; public class Main {
 public static class java {public static class lang {public static class Exception {} public static class Throwable {}}}
 public static void main(String[] args){model.E e=new model.E();model.R r=new model.R();model.Throwable shadow=new model.Throwable();
  System.out.println(Routes.thrown(e));System.out.println(Routes.checked(e));System.out.println(Routes.thrown(r));System.out.println(Routes.checked(r));System.out.println(Routes.runtime(r));
  System.out.println(Routes.thrown(shadow));System.out.println(Routes.checked(new Error()));System.out.println(Routes.source(shadow));System.out.println(Routes.source(e));
  model.E absent=null;System.out.println(Routes.thrown(absent));System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/E.java": `package model; public class E extends Exception {public E(){super("e");api.Routes.trace+="A";}}
`,
		"src/main/java/model/R.java": `package model; public class R extends RuntimeException {public R(){super("r");api.Routes.trace+="B";}}
`,
		"src/main/java/model/Throwable.java": `package model; public class Throwable {}
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "11\n21\n11\n21\n31\n19\n29\n41\n49\n15\nABTETEROFSOt\n", stderr: ""},
		{name: "repeat2", stdout: "11\n21\n11\n21\n31\n19\n29\n41\n49\n15\nABTETEROFSOt\n", stderr: ""},
		{name: "repeat3", stdout: "11\n21\n11\n21\n31\n19\n29\n41\n49\n15\nABTETEROFSOt\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCornerInheritedCoreJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>inheritedarguments</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api; public class Routes {public static String trace="";
 public static int pick(Comparable<model.Marker> value){trace+="C";return 51;} public static int pick(Object value){trace+="I";return 59;}
 public static int list(java.util.List<model.Marker> value){trace+="C";return 51;} public static int list(Object value){trace+="K";return 59;}
}
`,
		"src/main/java/app/Main.java": `package app;import api.Routes;public class Main {
 public static void main(String[] args){
  model.Grand<model.Marker> right=new model.Grand<model.Marker>();model.Grand<model.Other> wrong=new model.Grand<model.Other>();
  System.out.println(Routes.pick(right));System.out.println(Routes.pick(wrong));
  Comparable<model.Marker> alias=right;System.out.println(Routes.pick(alias));System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/Child.java": `package model; public class Child<U> extends Parent<U> {}
`,
		"src/main/java/model/Grand.java": `package model; public class Grand<W> extends Child<W> {}
`,
		"src/main/java/model/Marker.java": `package model; public class Marker {}
`,
		"src/main/java/model/Other.java": `package model; public class Other {}
`,
		"src/main/java/model/Parent.java": `package model; public class Parent<T> implements Comparable<T> {public int compareTo(T value){return 1;}}
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "51\n59\n51\nCIC\n", stderr: ""},
		{name: "repeat2", stdout: "51\n59\n51\nCIC\n", stderr: ""},
		{name: "repeat3", stdout: "51\n59\n51\nCIC\n", stderr: ""},
	})
}

func TestDeclaredMethodProvenanceCornerDeferredCoreJVM(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>campaign</groupId><artifactId>deferredarrays</artifactId><version>1</version></project>
`,
		"src/main/java/api/Routes.java": `package api;public class Routes {public static String trace="";
 public static int array(Comparable<model.Marker>[] values){trace+="A";return 61;}public static int array(Object[] values){trace+="J";return 69;}
 public static int pick(Comparable<model.Marker> value){trace+="C";return 31;}public static int pick(Object value){trace+="O";return 59;}
}
`,
		"src/main/java/app/Main.java": `package app;import api.Routes;public class Main {
 public static Comparable<model.Marker> once(){Routes.trace+="m";return new model.Holder<model.Marker>();}
 public static Comparable<model.Other> boom(){Routes.trace+="B";throw new IllegalStateException("boom");}
 public static <U> int live(Comparable<U> value){return Routes.pick(value);}
 public static <U> int count(U[] values){return values.length;}
 public static void main(String[] args){
  Comparable<model.Marker>[] right=new Comparable[]{new model.Holder<model.Marker>()};
  Comparable<model.Other>[] wrong=new Comparable[]{new model.Holder<model.Other>()};
  Comparable<model.Marker>[] absentRight=null;Comparable<model.Other>[] absentWrong=null;
  System.out.println(Routes.array(right));System.out.println(Routes.array(wrong));System.out.println(Routes.array(absentRight));System.out.println(Routes.array(absentWrong));
  Comparable<?> unknown=new model.Holder<model.Other>();System.out.println(Routes.pick(unknown));
  int[][] integers=new int[2][];System.out.println(count(integers));System.out.println(Routes.pick(once()));
  try{Routes.pick(boom());}catch(IllegalStateException failure){Routes.trace+="K";}finally{Routes.trace+="Z";}
  System.out.println(Routes.trace);
 }
}
`,
		"src/main/java/model/Holder.java": `package model;public class Holder<T> implements Comparable<T> {public int compareTo(T value){return 1;}}
`,
		"src/main/java/model/Marker.java": `package model;public class Marker {}
`,
		"src/main/java/model/Other.java": `package model;public class Other {}
`,
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "61\n69\n61\n69\n59\n2\n31\nAJAJOmCBKZ\n", stderr: ""},
		{name: "repeat2", stdout: "61\n69\n61\n69\n59\n2\n31\nAJAJOmCBKZ\n", stderr: ""},
		{name: "repeat3", stdout: "61\n69\n61\n69\n59\n2\n31\nAJAJOmCBKZ\n", stderr: ""},
	})
}
