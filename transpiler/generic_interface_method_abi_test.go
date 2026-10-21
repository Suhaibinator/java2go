package transpiler

import (
	"reflect"
	"testing"
)

func TestGenericInterfaceABIUnboundedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `interface Poly { <T> T id(T value); default <T> T round(T value){return id(value);} }
class Named implements Poly {public <T>T id(T value){Main.calls++;return value;}}
class Inherited extends Named {}
class Value {int code(){return 9;}}
public class Main {
 static int calls, receivers,arguments;
 static Poly receiver(Poly p){receivers++;return p;} static <T>T argument(T p){arguments++;return p;}
 public static void main(String[]args){Poly named=new Named();Poly inherited=new Inherited();Poly anonymous=new Poly(){public <T>T id(T value){calls++;return value;}};Value value=new Value();
 String text=receiver(named).round(argument("text")); Value returned=receiver(inherited).round(argument(value));Value third=anonymous.round(value); Object broad=named.id(value);Value nil=named.round((Value)null);
 System.out.println(text+":"+(returned==value)+":"+(third==value)+":"+(broad==value)+":"+(nil==null)+":"+calls+":"+receivers+":"+arguments);}
}
`}, "Main", "text:true:true:true:true:5:2:2\n")
}

func TestGenericInterfaceABIBoundedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `class Base {int code(){return 2;}}class Child extends Base {int code(){return 9;}}
interface Poly {<T extends Base>T id(T value);default <T extends Base>T round(T value){return id(value);}}
class Named implements Poly {public <T extends Base>T id(T value){Main.calls++;return value;}}
public class Main {static int calls;public static void main(String[]args){Poly p=new Named();Child child=new Child();Child narrow=p.round(child);Base broad=p.round(child);Object object=p.id(child);Child nil=p.round((Child)null);p.id(child);System.out.println((narrow==child)+":"+broad.code()+":"+(object==child)+":"+(nil==null)+":"+calls);}}
`}, "Main", "true:9:true:true:5\n")
}

func TestGenericInterfaceABIBoxJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `class Box<T>{T value;Box(T value){this.value=value;}}
interface Poly{<T>Box<T> id(Box<T> value);default <T>Box<T> round(Box<T> value){return id(value);}}
class Named implements Poly{public <U>Box<U> id(Box<U> value){Main.calls++;return value;}}
public class Main{static int calls;public static void main(String[]args){Poly p=new Named();Box<String> a=new Box<String>("first");Box<Integer> b=new Box<Integer>(7);Box<String> x=p.round(a);Box<Integer> y=p.id(b);System.out.println(((Object)x==(Object)a)+":"+((Object)y==(Object)b)+":"+x.value+":"+y.value+":"+calls);}}
`}, "Main", "true:true:first:7:2\n")
}

func TestGenericInterfaceABIMethodReferenceJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `interface Poly{<T>T id(T value);}public class Main{static <T>T identity(T x){return x;}public static void main(String[]args){Poly p=Main::identity;System.out.println(p.id("ref")+":"+p.id(7));}}
`}, "Main", "ref:7\n")
}

func TestGenericInterfaceABIInheritedJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `interface Parent{<T>T id(T value);default <T>T round(T value){return id(value);}}
interface Derived extends Parent{default <T>T inherited(T value){return round(value);}}
class Named implements Derived{public <T>T id(T value){Main.calls++;return value;}}
public class Main{static int calls;public static void main(String[]args){Derived p=new Named();String text=p.inherited("inherited");Integer count=p.inherited(7);System.out.println(text+":"+count+":"+calls);}}
`}, "Main", "inherited:7:2\n")
}

func TestGenericInterfaceABIExecutionJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `interface Poly{<T>T id(T value);default <T>T round(T value){return id(value);}}
class Named implements Poly{public <T>T id(T value){Main.calls++;if(Main.thread!=Thread.currentThread())throw new IllegalStateException("token");return value;}}
public class Main{static Thread thread;static int calls;public static void main(String[]args){thread=Thread.currentThread();Poly p=new Named();String text=p.round("execution");System.out.println(text+":"+calls);}}
`}, "Main", "execution:1\n")
}

func TestGenericInterfaceABICastTimingJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `class Base{}class Child extends Base{}
interface Poly{<T extends Base>T id(T value);default <T extends Base>T round(T value){return id(value);}}
class Named implements Poly{public <T extends Base>T id(T value){Main.calls++;return (T)new Base();}}
public class Main{static int calls;public static void main(String[]args){Poly p=new Named();Base broad=p.round(new Child());System.out.println("broad="+(broad!=null)+":"+calls);try{Child narrow=p.round(new Child());System.out.println("wrong");}catch(ClassCastException expected){System.out.println("narrow="+calls);}finally{System.out.println("finally="+calls);}}}
`}, "Main", "broad=true:1\nnarrow=2\nfinally=2\n")
}

func TestGenericInterfaceABIOverloadsJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/Main.java": `class Base{}class Other{}
interface Poly{<T extends Base>int select(T value);<T extends Other>long select(T value);}
class Named implements Poly{public <U extends Base>int select(U value){return 1;}public <U extends Other>long select(U value){return 2L;}}
public class Main{public static void main(String[]args){Poly p=new Named();System.out.println(p.select(new Base())+":"+p.select(new Other()));}}
`}, "Main", "1:2\n")
}

func TestGenericInterfaceABIMultipackageJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`,
		"src/main/java/api/Box.java": `package api; public class Box<T>{public T value;public Box(T value){this.value=value;}}
`,
		"src/main/java/api/Factory.java": `package api;public interface Factory<A>{<T>Box<T> create(Box<T> input);<U>Box<U> cross(A context,Box<U> input);default <U>Box<U> route(A context,Box<U> input){return cross(context,input);}}
`,
		"src/main/java/app/Main.java": `package app;public class Main{public static void main(String[]args){api.Factory<Object> p=new impl.Named<Object>();api.Box<String> first=new api.Box<String>("multi");api.Box<Integer> second=new api.Box<Integer>(9);api.Box<String> x=p.route("context",first);api.Box<Integer> y=p.create(second);System.out.println(((Object)x==(Object)first)+":"+((Object)y==(Object)second)+":"+x.value+":"+y.value);}}
`,
		"src/main/java/impl/Named.java": `package impl;public class Named<T> implements api.Factory<T>{public <T>api.Box<T> create(api.Box<T> input){return input;}public <U>api.Box<U> cross(T context,api.Box<U> input){return create(input);}}
`}, "app.Main", "true:true:multi:9\n")
}

func TestGenericInterfaceABIPlanDeclarationIdentity(t *testing.T) {
	helper := setupParseHelper(t, `class T2{}interface Owner<T>{<T>T id(T value);<U>T classValue(U value);<T>T2 nominal(T value);<T>T[] array(T[] value);<T>java.util.List<T> list(java.util.List<T> value);}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = owner
	def := owner.FindMethodByName("id", nil)
	plan := planGenericMethodEntryABI(owner, def, ctx)
	if plan == nil || plan.owner != owner || plan.method != def || len(plan.methodDeclarations) != 1 || plan.methodDeclarations[0] != def.TypeParameters[0].Declaration || plan.methodDeclarations[0] == owner.TypeParameters[0].Declaration {
		t.Fatal("method descriptor lost declaration ownership")
	}
	if !collectionObjectConsumer(plan.parameterJavaTypes[0], ctx) || !collectionObjectConsumer(plan.resultJavaType, ctx) {
		t.Fatalf("method erasure=%v ->%s", plan.parameterJavaTypes, plan.resultJavaType)
	}
	actualDescriptor, actualOK := javaTypeDescriptorExpr(plan.parameterJavaTypes[0], ctx)
	objectDescriptor, objectOK := javaTypeDescriptorExpr("java.lang.Object", ctx)
	if !actualOK || !objectOK || !reflect.DeepEqual(actualDescriptor, objectDescriptor) {
		t.Fatal("unqualified Object erasure lost canonical descriptor identity")
	}
	copy := *def
	if planGenericMethodEntryABI(owner, &copy, ctx) != nil {
		t.Fatal("foreign copied definition acquired owner plan")
	}
	classDef := owner.FindMethodByName("classValue", nil)
	classPlan := planGenericMethodEntryABI(owner, classDef, ctx)
	if classPlan == nil || classPlan.resultJavaType != owner.TypeParameters[0].EmittedName() {
		t.Fatalf("class-owned result became method erasure: %+v", classPlan)
	}
	nominal := planGenericMethodEntryABI(owner, owner.FindMethodByName("nominal", nil), ctx)
	if nominal == nil || nominal.resultJavaType != "T2" {
		t.Fatalf("nominal return stolen by allocated method alias: %+v", nominal)
	}
	for _, name := range []string{"array", "list"} {
		if planGenericMethodEntryABI(owner, owner.FindMethodByName(name, nil), ctx) != nil {
			t.Fatalf("unproved invariant %s carrier admitted", name)
		}
	}
}

func TestGenericInterfaceABIClassEntryEligibilityUnchanged(t *testing.T) {
	helper := setupParseHelper(t, `class Base{}class Owner<T>{<U>U scalar(U value){return value;}<U>java.util.List<U> nested(java.util.List<U> value){return value;}<U extends Base>U bounded(U value){return value;}}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	ctx := helper.Ctx.Clone()
	ctx.currentClass = owner
	for _, def := range owner.Methods {
		if def.RequiresHelper {
			if (planGenericMethodEntryABI(owner, def, ctx) != nil) != genericMethodHasErasedEntry(def) {
				t.Errorf("class eligibility changed: %s", def.OriginalName)
			}
		}
	}
}

func TestGenericInterfaceABIBoundBinderJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `class Base{}
class Child extends Base{}
interface Poly{<T extends Base>T id(T value);default<T extends Base>T round(T value){return id(value);}}
class Named implements Poly{public<T extends Base>T id(T value){Main.calls++;return value;}}
public class Main{
 static int calls;static<Base>Child call(Poly p,Child value){return p.round(value);}
 public static void main(String[]args){Poly p=new Named();Child child=new Child();Child seen=call(p,child);Child nil=call(p,(Child)null);System.out.println((seen==child)+":"+(nil==null)+":"+calls);}
}
`}, "Main", "true:true:2\n")
}

func TestGenericInterfaceABINullReceiverJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-abi</artifactId><version>1</version></project>`, "src/main/java/Main.java": `interface Parent<A>{<T>T id(T value);default<T>T round(T value){return id(value);}}
interface Derived<A> extends Parent<A>{default<T>T inherited(T value){return round(value);}}
class Named<A> implements Derived<A>{public<T>T id(T value){Main.bodies++;return value;}}
class Value{}
public class Main{
 static int receivers,arguments,bodies;static Derived<Object> receiver(Derived<Object> value){receivers++;return value;}static Value argument(Value value){arguments++;return value;}
 public static void main(String[]args){Derived<Object> p=new Named<Object>();Value value=new Value();Value first=receiver(p).inherited(argument(value));System.out.println((first==value)+":"+receivers+":"+arguments+":"+bodies);
 try{Value ignored=receiver(null).inherited(argument(value));System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+receivers+":"+arguments+":"+bodies);}finally{System.out.println("finally="+bodies);}}
}
`}, "Main", "true:1:1:1\nnull=2:2:1\nfinally=1\n")
}
