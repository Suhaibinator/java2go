package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"strings"
	"testing"
)

func TestInnerCarrierReceiverAndBinderIdentityJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 public static void main(String[] args) {
  Carrier<String> owner = new Carrier<String>();
  Carrier<Integer> otherOwner = new Carrier<Integer>();
  Carrier<String>.Cell<Integer> same = owner.new Cell<Integer>();
  Carrier<Integer>.Cell<Integer> other = otherOwner.new Cell<Integer>();
  Carrier<String>.Cell<Child> sameBound = owner.new Cell<Child>();
  Carrier<Integer>.Cell<Child> otherBound = otherOwner.new Cell<Child>();
  System.out.print(Probe.run(owner,same,other,sameBound,otherBound));
 }
}`, "1:2:3:4:5:6", map[string]string{"Carrier.java": innerCarrierIdentitySource})
}

func TestInnerCarrierRawFormalJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `class Storage<T> {
 class Part<U> {}
 int choose(Part value){return 1;} long choose(Object value){return 2L;}
}
public class Main {
 public static void main(String[]args){Storage<String> owner=new Storage<String>();Storage<Integer> otherOwner=new Storage<Integer>();Storage<String>.Part<Integer> same=owner.new Part<Integer>();Storage<Integer>.Part<Integer> other=otherOwner.new Part<Integer>();System.out.print(owner.choose(same)+":"+owner.choose(other));}
}`, "1:1")
}

func TestInnerCarrierRawOwnerJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {public static void main(String[]args){Box<String> owner=new Box<String>();Box<Integer> otherOwner=new Box<Integer>();Box<String>.Member same=owner.new Member();Box<Integer>.Member other=otherOwner.new Member();System.out.print(Probe.run(owner,same,other));}}`, "1:1:3:4:5:6", map[string]string{"Box.java": innerCarrierRawOwnerSource})
}

func TestInnerOwnerLayoutPlanUsesDeclarationDependencies(t *testing.T) {
	helper := setupParseHelper(t, `class Owner<T>{class Middle<V>{class Inner<U>{}}class Sibling<S>{}static class Static<S>{}}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	middle, sibling, static := owner.Subclasses[0], owner.Subclasses[1], owner.Subclasses[2]
	inner := middle.Subclasses[0]
	plan, err := planGenericFamily(inner, helper.Ctx)
	if err != nil {
		t.Fatalf("complete enclosing layout refused:%v", err)
	}
	for _, member := range []*symbol.ClassScope{owner, middle, inner, sibling} {
		if _, ok := plan.members[member]; !ok {
			t.Errorf("physical family omitted declaration %s", member.Class.OriginalName)
		}
	}
	if _, ok := plan.members[static]; ok {
		t.Fatal("static nested declaration joined enclosing storage")
	}
	for _, parent := range []*symbol.ClassScope{owner, middle} {
		for _, parameter := range parent.TypeParameters {
			if _, ok := plan.representations[parameter.Declaration]; !ok {
				t.Fatal("carried declaration representation missing")
			}
		}
	}
	if classScopeDescendsFrom(inner, owner, helper.Ctx) {
		t.Fatal("physical enclosing dependency became source inheritance")
	}
	if _, err := planGenericFamily(static, helper.Ctx); err != nil {
		t.Fatalf("independent static generic leaf rejected:%v", err)
	}
}

func TestInnerOwnerLayoutRejectsMismatchedDeclaration(t *testing.T) {
	helper := setupParseHelper(t, `class Owner<T>{class Inner<U>{}}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	inner := owner.Subclasses[0]
	inner.TypeParameters[0] = symbol.NewTypeParam("T", nil)
	if plan, err := planGenericFamily(inner, helper.Ctx); err == nil || plan != nil {
		t.Fatal("same spelling with foreign declaration satisfied enclosing layout proof")
	}
}

func TestInnerOwnerLayoutPreservesCompleteAudit(t *testing.T) {
	for _, test := range []struct{ name, source string }{
		{"runtime container", `class Owner<T>{java.util.List<T> values;class Inner<U>{}}`},
		{"owned array", `class Owner<T>{T[] values;class Inner<U>{}}`},
		{"constructor body", `class Owner<T>{T value;Owner(T value){this.value=value;T copy=this.value;}class Inner<U>{}}`},
		{"source bound", `class Bound{}class Owner<T extends Bound>{class Inner<U>{}}`},
		{"intersection bound", `class Owner<T extends java.lang.Number & java.io.Serializable>{class Inner<U>{}}`},
		{"unsafe sibling", `class Owner<T>{class Inner<U>{}class Sibling<V>{java.util.List<V> values;}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			inner := owner.Subclasses[0]
			if plan, err := planGenericFamily(inner, helper.Ctx); err == nil || plan != nil {
				t.Fatalf("unsafe complete family admitted:%s", test.source)
			}
			if canonicalGenericFamily(inner, helper.Ctx) != nil {
				t.Fatal("demand bypassed rejected complete family")
			}
		})
	}
}

func TestInnerOwnerLayoutRawTypedMutationJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `public class Main {
 static int reads;
 static class Owner<T> {
  T value; Owner(T seed){value=seed;} void put(T next){value=next;}
  class Cell<U> {U value;Cell(U seed){value=seed;}U read(){reads++;return value;}void put(U next){value=next;}T owner(){return Owner.this.value;}}
  Object take(Cell value){return value.owner();}
 }
 public static void main(String[]args) {
  Owner<String> owner=new Owner<String>("seed");Owner<Integer> other=new Owner<Integer>(23);
  Owner<String>.Cell<String> cell=owner.new Cell<String>("old");Owner<Integer>.Cell<Integer> otherCell=other.new Cell<Integer>(31);
  Owner.Cell raw=cell;
  System.out.println("same="+((Object)raw==(Object)cell));
  raw.put(7);Object broad=raw.read();System.out.println("broad="+broad);
  try {String bad=cell.read();System.out.println("wrong="+bad);}catch(ClassCastException expected){System.out.println("cast="+reads);}
  raw.put("new");System.out.println("repair="+cell.read());
  Owner rawOwner=owner;rawOwner.put(17);Object outerBroad=raw.owner();System.out.println("outer="+outerBroad);
  try {String bad=cell.owner();System.out.println("wrong="+bad);}catch(ClassCastException expected){System.out.println("outercast");}
  System.out.println("other="+owner.take(otherCell));
 }
}`, "same=true\nbroad=7\ncast=2\nrepair=new\nouter=17\noutercast\nother=23\n")
}

func TestInnerOwnerLayoutConcreteBoundReadCastTimingJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `class Base{} class Child extends Base{} class Alien{}
class Owner<T> {
 class Cell<U>{U value;Cell(U seed){value=seed;}U read(){Main.reads++;return value;}void put(U next){value=next;}}
 <T extends Base>T echo(Cell<T> value){T got=value.read();Main.bodies++;return got;}
}
public class Main {
 static int reads,bodies;
 public static void main(String[]args) {
  Owner<String> owner=new Owner<String>();Child child=new Child();Owner<String>.Cell<Child> cell=owner.new Cell<Child>(child);
  Child returned=owner.echo(cell);System.out.println("same="+(returned==child));
  Owner.Cell raw=cell;raw.put(new Base());
  Base broad=owner.echo(cell);System.out.println("broad="+(broad!=null)+":"+reads+":"+bodies);
  try{Child narrow=owner.echo(cell);System.out.println("wrong="+narrow);}catch(ClassCastException expected){System.out.println("resultcast="+reads+":"+bodies);}
  raw.put(new Alien());try{Base bad=owner.echo(cell);System.out.println("wrong="+bad);}catch(ClassCastException expected){System.out.println("readcast="+reads+":"+bodies);}
 }
}`, "same=true\nbroad=true:2:2\nresultcast=3:3\nreadcast=4:3\n")
}

func TestInnerOwnerLayoutRuntimeArrayDescriptorsJDK21(t *testing.T) {
	runCampaignThrowableStrictSingleFileOracle(t, `class Base{}class Child extends Base{}class Other{}
class Owner<T>{class Cell<U>{} <T extends Base>void store(Cell<T> cell,T[] values){values[0]=(T)new Base();}}
public class Main {
 public static void main(String[]args) {
  Owner<String> owner=new Owner<String>();Owner<Integer> other=new Owner<Integer>();
  Owner<String>.Cell<Child> cell=owner.new Cell<Child>();Owner<Integer>.Cell<Integer> second=other.new Cell<Integer>();
  Owner.Cell[] array=new Owner.Cell[]{cell};Object[] alias=array;alias[0]=second;
  System.out.println("alias="+((Object)array==(Object)alias)+":"+((Object)array[0]==(Object)second));
  System.out.println("component="+array.getClass().getComponentType().getName());
  try{alias[0]=new Other();System.out.println("wrong");}catch(ArrayStoreException expected){System.out.println("nominalstore");}
  Child[] values=new Child[]{new Child()};
  try{owner.store(cell,values);System.out.println("wrong");}catch(ArrayStoreException expected){System.out.println("boundstore");}
  System.out.println("preserved="+(values[0]!=null));
 }
}`, "alias=true:true\ncomponent=Owner$Cell\nnominalstore\nboundstore\npreserved=true\n")
}

func TestInnerOwnerLayoutMethodAndClassShadowRemainDistinct(t *testing.T) {
	helper := setupParseHelper(t, `class Owner<T>{class Inner<T>{<T>T echo(T value){return value;}}}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	inner := owner.Subclasses[0]
	method := inner.FindMethodByName("echo", nil)
	plan, err := planGenericFamily(inner, helper.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.representations) != 2 {
		t.Fatalf("physical class declaration count=%d,want2", len(plan.representations))
	}
	if _, present := plan.representations[method.TypeParameters[0].Declaration]; present {
		t.Fatal("method declaration joined class physical storage")
	}
	if owner.TypeParameters[0].Declaration == inner.OwnTypeParameters()[0].Declaration {
		t.Fatal("class shadows collapsed")
	}
	if strings.TrimSpace(method.TypeParameters[0].EmittedName()) == "" {
		t.Fatal("method alias missing")
	}
}

func TestInnerOwnerLayoutEnclosingSelectorCollisionControls(t *testing.T) {
	owner := &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Owner"}}
	inner := &symbol.ClassScope{IsInner: true, Enclosing: owner}
	if got := inner.EnclosingFieldName(); got != "owner" {
		t.Fatalf("default enclosing selector=%q", got)
	}
	inner.Methods = []*symbol.Definition{{Name: "owner", IsStatic: true}, {Name: "owner", Constructor: true}, nil}
	if got := inner.EnclosingFieldName(); got != "owner" {
		t.Fatalf("nonreceiver declaration renamed enclosing selector:%q", got)
	}
	inner.Methods = append(inner.Methods, &symbol.Definition{OriginalName: "originalJavaName", Name: "owner"}, &symbol.Definition{Name: "owner_"})
	inner.Fields = []*symbol.Definition{{Name: "owner__"}, nil}
	for run := 0; run < 3; run++ {
		if got := inner.EnclosingFieldName(); got != "owner___" {
			t.Fatalf("collision-free emitted selector=%q", got)
		}
	}
	if inner.Methods[3].Name != "owner" || inner.EnclosingField != "" {
		t.Fatal("query mutated declaration names")
	}
	inner.EnclosingField = "allocatedOuter"
	if got := inner.EnclosingFieldName(); got != "allocatedOuter" {
		t.Fatalf("allocated override changed:%q", got)
	}
	inner.IsInner = false
	if got := inner.EnclosingFieldName(); got != "" {
		t.Fatalf("noninner selector=%q", got)
	}
	var absent *symbol.ClassScope
	if absent.EnclosingFieldName() != "" {
		t.Fatal("nil scope has selector")
	}
}

func TestInnerOwnerLayoutEnclosingSelectorResolvedNames(t *testing.T) {
	helper := setupParseHelper(t, `class Owner {class Inner {int owner(){return 1;} int owner(int x){return x;} int owner_=2; class Deep {int inner(){return 3;}}}}`)
	owner := helper.File.Symbols.FindClassScope("Owner")
	inner := owner.Subclasses[0]
	deep := inner.Subclasses[0]
	for _, scope := range []*symbol.ClassScope{inner, deep} {
		name := scope.EnclosingFieldName()
		for _, method := range scope.Methods {
			if !method.IsStatic && !method.Constructor && method.Name == name {
				t.Fatalf("enclosing selector collides with resolved method %q", name)
			}
		}
		for _, field := range scope.Fields {
			if !field.IsStatic && field.Name == name {
				t.Fatalf("enclosing selector collides with resolved field %q", name)
			}
		}
	}
	if inner.Enclosing != owner || deep.Enclosing != inner {
		t.Fatal("selector allocation changed lexical owner identity")
	}
}

func TestInnerOwnerLayoutEnclosingInheritedPackageCycleJDK21(t *testing.T) {
	runCampaignCompilerProjectOracle(t, map[string]string{
		"pom.xml":                    `<project><groupId>probe</groupId><artifactId>enclosing-collision</artifactId><version>1</version></project>`,
		"src/main/java/Main.java":    `public class Main {public static void main(String[]args){b.Sub sub=new b.Sub();a.Owner.Inner inner=sub.new Inner();System.out.println(inner.read()+":"+inner.peer());}}`,
		"src/main/java/a/Owner.java": `package a; public class Owner {int value=7;public class Inner {int owner(){return Owner.this.value;}public int read(){return owner();}public int peer(){return b.Sub.other();}}}`,
		"src/main/java/b/Sub.java":   `package b; public class Sub extends a.Owner {public static int other(){return 9;}}`,
	}, "Main", "7:9\n")
}
