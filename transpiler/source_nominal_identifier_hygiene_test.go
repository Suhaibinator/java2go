package transpiler

import (
	"go/ast"
	"testing"
)

// These are independent controls. The complete canonical-family clone fixture
// remains unchanged and pending its separate erased member projection repair.
func TestSourceNominalIdentifierHygieneStrictJVM(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"constructorAllocation", `package names;
class Parcel { final int code; Parcel(int parcel){code=parcel;} }
public class Main { public static void main(String[] args){Parcel item=new Parcel(7);System.out.println(item.code);} }`, "7\n"},
		{"ordinaryFieldAccess", `package names;
class Parcel { final int code; Parcel(int input){code=input;} }
public class Main {
 static int read(Parcel parcel){return parcel.code;}
 public static void main(String[] args){Parcel parcel=new Parcel(8);System.out.println(read(parcel)+":"+parcel.code);}
}`, "8:8\n"},
		{"suffixCollision", `package names;
class Parcel { final int code; Parcel(int parcel,int parcelJava2goLocal,int parcelJava2goLocal1){code=100*parcel+10*parcelJava2goLocal+parcelJava2goLocal1;} }
public class Main { public static void main(String[] args){System.out.println(new Parcel(2,3,4).code);} }`, "234\n"},
		{"lateSuperclassArgument", `package names;
class Parcel { final int code; Parcel(int input){code=input;} }
abstract class Holder<T> {final T item;Holder(T input){item=input;}abstract T read();}
interface Factory {<S> Holder<S> create();}
class Child extends Holder<Parcel> {Child(Parcel parcel){super(parcel);}Parcel read(){return item;}}
public class Main {public static void main(String[] args){Parcel item=new Parcel(9);Child child=new Child(item);System.out.println((child.read()==item)+":"+item.code);}}`, "true:9\n"},
		{"lateGenericRead", `package names;
class Parcel {final int code;Parcel(int input){code=input;}}
class Box<T>{T item;Box(T input){item=input;}T read(){return item;}}
public class Main {
 static int take(Parcel item){return item.code;}
 static int read(Box<Parcel> box,int parcel){return take(box.read())+parcel;}
 public static void main(String[] args){System.out.println(read(new Box<Parcel>(new Parcel(6)),4));}
}`, "10\n"},
		{"captureBinding", `package names;
class Parcel {}
public class Main {
 static int work(int parcel){class Local {final int captured=parcel;int add(int argument){return captured+argument;}}return new Local().add(3)+parcel;}
 public static void main(String[] args){System.out.println(work(5));}
}`, "13\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runCampaignCompilerStrictProjectOracle(t, map[string]string{
				"pom.xml":                       `<project><groupId>names</groupId><artifactId>names</artifactId><version>1</version></project>`,
				"src/main/java/names/Main.java": test.source,
			}, "names.Main", test.want)
		})
	}
}

func TestSourceNominalDependencyAliasStrictJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                           `<project><groupId>names</groupId><artifactId>names</artifactId><version>1</version></project>`,
		"src/main/java/foreign/Parcel.java": `package foreign;public class Parcel {public final int code;public Parcel(int input){code=input;}}`,
		"src/main/java/names/Box.java":      `package names;class Box<T>{T item;Box(T input){item=input;}T read(){return item;}}`,
		"src/main/java/names/Main.java": `package names;import foreign.Parcel;
public class Main {
 static int take(Parcel item){return item.code;}
 static int read(Box<Parcel> box,int foreign,int Parcel){return take(box.read())+foreign+Parcel;}
 public static void main(String[] args){System.out.println(read(new Box<Parcel>(new Parcel(6)),3,2));}
}`,
	}, "names.Main", "11\n")
}

// Java's keyword new cannot be a source binding. Exercise lowering demand with
// valid parsed declarations, rather than fabricate an invalid JVM control.
func TestConstructorAllocationBuiltinDemand(t *testing.T) {
	for _, test := range []struct {
		name, source, method string
		want                 bool
	}{
		{"allocation", `class Parcel {Parcel(int input){}}`, "Parcel", true},
		{"thisDelegation", `class Parcel {Parcel(){this(1);}Parcel(int input){}}`, "Parcel", false},
		{"ordinaryMethod", `class Parcel {void read(int input){}}`, "read", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			owner := helper.File.Symbols.FindClassScope("Parcel")
			ctx := classScopeCtx(owner, helper.Ctx)
			ctx.localScope = owner.FindMethod().ByOriginalName(test.method)[0]
			if got := localIdentifierRequiredByBody("new", ctx); got != test.want {
				t.Fatalf("allocation demand=%v want %v", got, test.want)
			}
		})
	}
}

func TestSourceNominalHygienePreservesDeclarationMetadata(t *testing.T) {
	helper := setupParseHelper(t, `class Parcel {int parcel;Parcel(int parcel,long parcelJava2goLocal,Object last){this.parcel=parcel;}}`)
	owner := helper.File.Symbols.FindClassScope("Parcel")
	constructor := owner.FindMethod().ByOriginalName("Parcel")[0]
	want := []string{"parcel", "parcelJava2goLocal", "last"}
	wantTypes := []string{"int", "long", "Object"}
	if owner.Class.Name != "parcel" || owner.Class.OriginalName != "Parcel" {
		t.Fatal("nominal declaration ABI changed")
	}
	if len(constructor.Parameters) != len(want) {
		t.Fatal("constructor parameter count changed")
	}
	for index, parameter := range constructor.Parameters {
		if parameter.OriginalName != want[index] || parameter.OriginalType != wantTypes[index] {
			t.Fatalf("source parameter ordinal %d changed: %#v", index, parameter)
		}
	}
	if field := owner.FindField().ByOriginalName("parcel")[0]; field.Name != "parcel" {
		t.Fatal("instance field ABI changed")
	}
	if constructor.Parameters[0].Name == "parcel" || constructor.Parameters[0].Name == constructor.Parameters[1].Name {
		t.Fatal("type/value collision or suffix collision remains")
	}
	descriptors := reflectCoreSlice(t, sourceReflectionConstructorDescriptors(owner, classScopeCtx(owner, helper.Ctx)))
	if len(descriptors) != 1 {
		t.Fatalf("reflection constructors=%d want 1", len(descriptors))
	}
	parameters := reflectCoreSlice(t, reflectCoreKey(t, reflectCoreComposite(t, descriptors[0]), "Parameters"))
	if len(parameters) != 3 {
		t.Fatalf("reflection parameter ordinals=%d want 3", len(parameters))
	}
	for index, want := range []string{"PrimitiveIntTypeID", "PrimitiveLongTypeID", "ObjectTypeID"} {
		selector, ok := parameters[index].(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != want {
			t.Fatalf("reflection ordinal %d=%#v want %s", index, parameters[index], want)
		}
	}
}
