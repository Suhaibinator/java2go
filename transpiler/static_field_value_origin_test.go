package transpiler

import (
	"bytes"
	"encoding/json"
	sitter "github.com/smacker/go-tree-sitter"
	"go/format"
	"go/token"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

func TestStaticFieldStorageDeclaredOrigin(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	sources := []parsing.SourceFile{
		{Name: "model/T2.java", Source: []byte(`package model;public class T2{}`)},
		{Name: "model/Holder.java", Source: []byte(`package model;public class Holder<T>{}`)},
		{Name: "model/Task.java", Source: []byte(`package model;public abstract class Task{public abstract int key();}`)},
		{Name: "donor/Provider.java", Source: []byte(`package donor;import model.T2;import model.Holder;import model.Task;public class Provider{public static java.lang.Class<T2> token;public static Holder<T2> held;public static T2[] refs;public static int[] numbers;public static Holder<T2[]> arrays;public static Task task;public static java.util.List<T2> list;}`)},
		{Name: "consumer/Probe.java", Source: []byte(`package consumer;import donor.Provider;class Probe{static class java{static class lang{static class Class{}}}static<T2>Object run(){return Provider.held;}}`)},
	}
	for index := range sources {
		file := &sources[index]
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
	}
	for _, file := range sources {
		ResolveFile(file)
	}
	file := sources[4]
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.FindClassScope("Probe"), importAliases: map[string]string{}, usedImports: map[string]bool{}}
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	owner := sources[3].Symbols.FindClassScope("Provider")
	for _, row := range []struct{ field, want string }{
		{"token", "*stdjava.Class"}, {"held", "*model.Holder[*model.T2]"}, {"refs", "*stdjava.ReferenceArray"}, {"numbers", "*stdjava.PrimitiveArray[int32]"}, {"arrays", "*model.Holder[*stdjava.ReferenceArray]"}, {"task", "model.TaskI"}, {"list", "*stdjava.List[*model.T2]"},
	} {
		t.Run(row.field, func(t *testing.T) {
			field := owner.FindFieldByName(row.field)
			if field == nil {
				t.Fatal("missing real declared field")
			}
			before := field.OriginalType
			ctx.expectedType = "java.lang.String[]"
			ctx.expectedTypeRoot = file.Ast
			expr, ok := staticFieldDeclaredStorageType(field, owner, ctx)
			if !ok {
				t.Fatal("complete field origin did not produce storage type")
			}
			var output bytes.Buffer
			if err := format.Node(&output, token.NewFileSet(), expr); err != nil {
				t.Fatal(err)
			}
			if output.String() != row.want {
				t.Fatalf("storage type=%s want %s", output.String(), row.want)
			}
			if field.OriginalType != before || ctx.expectedType != "java.lang.String[]" || ctx.expectedTypeRoot != file.Ast {
				t.Fatal("field projection mutated declaration or caller target")
			}
		})
	}
	if !ctx.usedImports["model"] || !ctx.usedImports[stdjavaImportPath] {
		t.Fatalf("output imports missing: %v", ctx.usedImports)
	}
	if _, ok := staticFieldDeclaredStorageType(owner.FindFieldByName("held"), owner, Ctx{}); ok {
		t.Fatal("default context fabricated caller import proof")
	}
}

func TestStaticFieldStorageUnknownDoesNotPublishImports(t *testing.T) {
	h := setupParseHelper(t, `class Owner{static absent.Unknown value;}`)
	ctx := h.Ctx.Clone()
	ctx.importAliases = map[string]string{"keep": "kept"}
	ctx.usedImports = map[string]bool{"keep": true}
	owner := h.File.Symbols.FindClassScope("Owner")
	if _, ok := staticFieldDeclaredStorageType(owner.FindFieldByName("value"), owner, ctx); ok {
		t.Fatal("unresolved external declaration fabricated storage proof")
	}
	if len(ctx.importAliases) != 1 || len(ctx.usedImports) != 1 || ctx.importAliases["keep"] != "kept" || !ctx.usedImports["keep"] {
		t.Fatalf("failed proof published speculative imports: %v %v", ctx.importAliases, ctx.usedImports)
	}
	detached := &symbol.ClassScope{Class: &symbol.Definition{Name: "Owner", OriginalName: "Owner"}}
	if _, ok := staticFieldDeclaredStorageType(owner.FindFieldByName("value"), detached, ctx); ok {
		t.Fatal("detached owner borrowed caller file certificate")
	}
}

func TestStaticFieldStorageWorkflowJVM(t *testing.T) {
	var files map[string]string
	if err := json.Unmarshal([]byte(`{
  "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>storage</groupId><artifactId>workflow</artifactId><version>1</version></project>",
  "src/main/java/app/Main.java": "package app;import model.*;import state.Trace;public class Main{\n static class java{static class lang{static class Class{}}}\n static ReadOwner read(){Trace.text+=\"Q\";return null;}static WriteOwner write(){Trace.text+=\"Q\";return null;}static CompoundOwner compound(){Trace.text+=\"Q\";return null;}static UpdateOwner update(){Trace.text+=\"Q\";return null;}\n static<Leaf>void origin(){int stdjava=1;System.out.println(Provider.box==Provider.get());System.out.println(Provider.box.value.key());System.out.println(Provider.refs[0].key());System.out.println(Provider.arrays.value[0].key());System.out.println(Provider.numbers[0]);System.out.println(Provider.task.key());System.out.println(Provider.missing==null);System.out.println(Provider.token==model.Leaf.class);System.out.println(stdjava);}\n public static void main(String[]args){origin();Trace.reset();System.out.println(read().number);System.out.println(Trace.text);Trace.reset();System.out.println(write().number=Trace.mark(\"X\",9));System.out.println(Trace.text);Trace.reset();System.out.println(compound().number+=Trace.mark(\"X\",2));System.out.println(Trace.text);Trace.reset();System.out.println(update().number++);System.out.println(Trace.text);Trace.reset();System.out.println(++update().number);System.out.println(Trace.text);Trace.reset();System.out.println(VolatileOwner.number++);System.out.println(VolatileOwner.number);System.out.println(VolatileOwner.box==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.inherited==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.hidden);System.out.println(Base.hidden);System.out.println(Trace.text);}\n}",
  "src/main/java/model/Base.java": "package model;public class Base{public static int hidden=state.Trace.mark(\"B\",17);public static Box<Leaf> inherited=Provider.box;}",
  "src/main/java/model/Box.java": "package model;public class Box<T>{public T value;public Box(T v){value=v;}}",
  "src/main/java/model/Child.java": "package model;public class Child extends Base{public static int hidden=state.Trace.mark(\"D\",19);}",
  "src/main/java/model/CompoundOwner.java": "package model;public class CompoundOwner{public static int number=state.Trace.mark(\"C\",6);}",
  "src/main/java/model/Leaf.java": "package model;public class Leaf{public int value;public Leaf(int v){value=v;}public int key(){return value;}}",
  "src/main/java/model/Provider.java": "package model;public class Provider{public static Box<Leaf> box=new Box<Leaf>(new Leaf(11));public static Leaf[] refs=new Leaf[]{new Leaf(9)};public static Box<Leaf[]> arrays=new Box<Leaf[]>(refs);public static int[] numbers=new int[]{7};public static Task task=new TaskImpl();public static Leaf missing=null;public static java.lang.Class<Leaf> token=Leaf.class;public static Box<Leaf> get(){return box;}}",
  "src/main/java/model/ReadOwner.java": "package model;public class ReadOwner{public static int number=state.Trace.mark(\"R\",3);}",
  "src/main/java/model/Task.java": "package model;public abstract class Task{public abstract int key();}",
  "src/main/java/model/TaskImpl.java": "package model;public class TaskImpl extends Task{public int key(){return 13;}}",
  "src/main/java/model/UpdateOwner.java": "package model;public class UpdateOwner{public static int number=state.Trace.mark(\"U\",5);}",
  "src/main/java/model/VolatileOwner.java": "package model;public class VolatileOwner{public static volatile int number=state.Trace.mark(\"V\",2);public static volatile Box<Leaf> box=Provider.box;}",
  "src/main/java/model/WriteOwner.java": "package model;public class WriteOwner{public static int number=state.Trace.mark(\"W\",4);}",
  "src/main/java/state/Trace.java": "package state;public class Trace{public static String text=\"\";public static int mark(String name,int value){text+=name;return value;}public static void reset(){text=\"\";}}"
}
`), &files); err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "true\n11\n9\n9\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
		{name: "repeat2", stdout: "true\n11\n9\n9\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
		{name: "repeat3", stdout: "true\n11\n9\n9\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
	})
}

func TestStaticFieldStorageOperationsJVM(t *testing.T) {
	var files map[string]string
	if err := json.Unmarshal([]byte(`{
  "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>storage</groupId><artifactId>workflow</artifactId><version>1</version></project>",
  "src/main/java/app/Main.java": "package app;import model.*;import state.Trace;public class Main{\n static class java{static class lang{static class Class{}}}\n static ReadOwner read(){Trace.text+=\"Q\";return null;}static WriteOwner write(){Trace.text+=\"Q\";return null;}static CompoundOwner compound(){Trace.text+=\"Q\";return null;}static UpdateOwner update(){Trace.text+=\"Q\";return null;}\n static<Leaf>void origin(){int stdjava=1;System.out.println(Provider.box==Provider.get());System.out.println(Provider.box.value.key());System.out.println(Provider.refs[0].key());System.out.println(Provider.arrays==Provider.arrayGetter());System.out.println(Provider.numbers[0]);System.out.println(Provider.task.key());System.out.println(Provider.missing==null);System.out.println(Provider.token==model.Leaf.class);System.out.println(stdjava);}\n public static void main(String[]args){origin();Trace.reset();System.out.println(read().number);System.out.println(Trace.text);Trace.reset();System.out.println(write().number=Trace.mark(\"X\",9));System.out.println(Trace.text);Trace.reset();System.out.println(compound().number+=Trace.mark(\"X\",2));System.out.println(Trace.text);Trace.reset();System.out.println(update().number++);System.out.println(Trace.text);Trace.reset();System.out.println(++update().number);System.out.println(Trace.text);Trace.reset();System.out.println(VolatileOwner.number++);System.out.println(VolatileOwner.number);System.out.println(VolatileOwner.box==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.inherited==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.hidden);System.out.println(Base.hidden);System.out.println(Trace.text);}\n}",
  "src/main/java/model/Base.java": "package model;public class Base{public static int hidden=state.Trace.mark(\"B\",17);public static Box<Leaf> inherited=Provider.box;}",
  "src/main/java/model/Box.java": "package model;public class Box<T>{public T value;public Box(T v){value=v;}}",
  "src/main/java/model/Child.java": "package model;public class Child extends Base{public static int hidden=state.Trace.mark(\"D\",19);}",
  "src/main/java/model/CompoundOwner.java": "package model;public class CompoundOwner{public static int number=state.Trace.mark(\"C\",6);}",
  "src/main/java/model/Leaf.java": "package model;public class Leaf{public int value;public Leaf(int v){value=v;}public int key(){return value;}}",
  "src/main/java/model/Provider.java": "package model;public class Provider{public static Box<Leaf> box=new Box<Leaf>(new Leaf(11));public static Leaf[] refs=new Leaf[]{new Leaf(9)};public static Box<Leaf[]> arrays=new Box<Leaf[]>(refs);public static int[] numbers=new int[]{7};public static Task task=new TaskImpl();public static Leaf missing=null;public static java.lang.Class<Leaf> token=Leaf.class;public static Box<Leaf> get(){return box;}public static Box<Leaf[]> arrayGetter(){return arrays;}}",
  "src/main/java/model/ReadOwner.java": "package model;public class ReadOwner{public static int number=state.Trace.mark(\"R\",3);}",
  "src/main/java/model/Task.java": "package model;public abstract class Task{public abstract int key();}",
  "src/main/java/model/TaskImpl.java": "package model;public class TaskImpl extends Task{public int key(){return 13;}}",
  "src/main/java/model/UpdateOwner.java": "package model;public class UpdateOwner{public static int number=state.Trace.mark(\"U\",5);}",
  "src/main/java/model/VolatileOwner.java": "package model;public class VolatileOwner{public static volatile int number=state.Trace.mark(\"V\",2);public static volatile Box<Leaf> box=Provider.box;}",
  "src/main/java/model/WriteOwner.java": "package model;public class WriteOwner{public static int number=state.Trace.mark(\"W\",4);}",
  "src/main/java/state/Trace.java": "package state;public class Trace{public static String text=\"\";public static int mark(String name,int value){text+=name;return value;}public static void reset(){text=\"\";}}"
}
`), &files); err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "true\n11\n9\ntrue\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
		{name: "repeat2", stdout: "true\n11\n9\ntrue\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
		{name: "repeat3", stdout: "true\n11\n9\ntrue\n7\n13\ntrue\ntrue\n1\n3\nQR\n9\nQXW\n8\nQCX\n5\nQU\n7\nQ\n2\n3\ntrue\nV\ntrue\nB\n19\n17\nD\n", stderr: ""},
	})
}

func TestStaticFieldStorageReferenceWritesJVM(t *testing.T) {
	var files map[string]string
	if err := json.Unmarshal([]byte(`{
  "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>storage</groupId><artifactId>referencewrites</artifactId><version>1</version></project>",
  "src/main/java/app/Main.java": "package app;import model.*;import state.Trace;public class Main{\n static class java{static class lang{static class Class{}}}\n static RefOwner owner(){Trace.mark(\"Q\");return null;}static Box<model.Leaf> rhs(){Trace.mark(\"X\");return Provider.box;}\n static ArrayOwner array(){Trace.mark(\"Q\");return null;}static model.Leaf[] arrayRhs(){Trace.mark(\"X\");return Provider.refs;}\n static<Leaf>void run(){int stdjava=2;Trace.reset();System.out.println((owner().box=rhs())==Provider.box);System.out.println(Trace.text);System.out.println(RefOwner.box.value.key());Trace.reset();System.out.println((owner().box=null)==null);System.out.println(Trace.text);System.out.println(RefOwner.box==null);Trace.reset();System.out.println((array().refs=arrayRhs())==Provider.refs);System.out.println(Trace.text);System.out.println(ArrayOwner.refs[0].key());System.out.println((Provider.token=model.Leaf.class)==model.Leaf.class);System.out.println((VolatileOwner.box=Provider.box)==Provider.box);System.out.println(VolatileOwner.box.value.key());System.out.println(stdjava);}\n public static void main(String[]args){run();}}\n",
  "src/main/java/model/ArrayOwner.java": "package model;public class ArrayOwner{public static Leaf[] refs=init();static Leaf[] init(){state.Trace.mark(\"A\");return null;}}",
  "src/main/java/model/Box.java": "package model;public class Box<T>{public T value;public Box(T v){value=v;}}",
  "src/main/java/model/Leaf.java": "package model;public class Leaf{public int n;public Leaf(int n){this.n=n;}public int key(){return n;}}",
  "src/main/java/model/Provider.java": "package model;public class Provider{public static Box<Leaf> box=new Box<Leaf>(new Leaf(11));public static Leaf[] refs=new Leaf[]{new Leaf(9)};public static Class<Leaf> token=Leaf.class;}",
  "src/main/java/model/RefOwner.java": "package model;public class RefOwner{public static Box<Leaf> box=init();static Box<Leaf> init(){state.Trace.mark(\"I\");return null;}}",
  "src/main/java/model/VolatileOwner.java": "package model;public class VolatileOwner{public static volatile Box<Leaf> box;}",
  "src/main/java/state/Trace.java": "package state;public class Trace{public static String text=\"\";public static void mark(String s){text+=s;}public static void reset(){text=\"\";}}"
}
`), &files); err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "true\nQXI\n11\ntrue\nQ\ntrue\ntrue\nQXA\n9\ntrue\ntrue\n11\n2\n", stderr: ""},
		{name: "repeat2", stdout: "true\nQXI\n11\ntrue\nQ\ntrue\ntrue\nQXA\n9\ntrue\ntrue\n11\n2\n", stderr: ""},
		{name: "repeat3", stdout: "true\nQXI\n11\ntrue\nQ\ntrue\ntrue\nQXA\n9\ntrue\ntrue\n11\n2\n", stderr: ""},
	})
}

func TestStaticFieldStorageClassNameJVM(t *testing.T) {
	var files map[string]string
	if err := json.Unmarshal([]byte(`{
  "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>castprobe</groupId><artifactId>diagnostics</artifactId><version>1</version></project>\n",
  "src/main/java/fcast/Main.java": "package fcast;public class Main{\n static class Class{int cast(Object v){return 53;}}\n static class java{static class lang{static class Class{int cast(Object v){return 59;}}}}\n static <T2>void probe(){System.out.println(Provider.token.getName());System.out.println(Provider.token==Provider.token());}\n public static void main(String[]args){probe();}\n}\n",
  "src/main/java/fcast/Provider.java": "package fcast;public class Provider{public static java.lang.Class<T2> token=T2.class;public static java.lang.Class<T2> token(){return token;}}\n",
  "src/main/java/fcast/T2.java": "package fcast;public class T2{public int key(){return 43;}}\n"
}
`), &files); err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "fcast.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "fcast.T2\ntrue\n", stderr: ""},
		{name: "repeat2", stdout: "fcast.T2\ntrue\n", stderr: ""},
		{name: "repeat3", stdout: "fcast.T2\ntrue\n", stderr: ""},
	})
}

func TestStaticFieldStoragePrerequisiteOriginTrace(t *testing.T) {
	for _, row := range []struct{ name, files, class, method, receiver string }{
		{"Class", "{\"pom.xml\": \"<project><modelVersion>4.0.0</modelVersion><groupId>castprobe</groupId><artifactId>diagnostics</artifactId><version>1</version></project>\\n\", \"src/main/java/fcast/Main.java\": \"package fcast;public class Main{\\n static class Class{int cast(Object v){return 53;}}\\n static class java{static class lang{static class Class{int cast(Object v){return 59;}}}}\\n static <T2>void probe(){System.out.println(Provider.token.getName());System.out.println(Provider.token==Provider.token());}\\n public static void main(String[]args){probe();}\\n}\\n\", \"src/main/java/fcast/Provider.java\": \"package fcast;public class Provider{public static java.lang.Class<T2> token=T2.class;public static java.lang.Class<T2> token(){return token;}}\\n\", \"src/main/java/fcast/T2.java\": \"package fcast;public class T2{public int key(){return 43;}}\\n\"}", "Main", "probe", "Provider.token"},
		{"Array", "{\"pom.xml\": \"<project><modelVersion>4.0.0</modelVersion><groupId>storage</groupId><artifactId>workflow</artifactId><version>1</version></project>\", \"src/main/java/app/Main.java\": \"package app;import model.*;import state.Trace;public class Main{\\n static class java{static class lang{static class Class{}}}\\n static ReadOwner read(){Trace.text+=\\\"Q\\\";return null;}static WriteOwner write(){Trace.text+=\\\"Q\\\";return null;}static CompoundOwner compound(){Trace.text+=\\\"Q\\\";return null;}static UpdateOwner update(){Trace.text+=\\\"Q\\\";return null;}\\n static<Leaf>void origin(){int stdjava=1;System.out.println(Provider.box==Provider.get());System.out.println(Provider.box.value.key());System.out.println(Provider.refs[0].key());System.out.println(Provider.arrays.value[0].key());System.out.println(Provider.numbers[0]);System.out.println(Provider.task.key());System.out.println(Provider.missing==null);System.out.println(Provider.token==model.Leaf.class);System.out.println(stdjava);}\\n public static void main(String[]args){origin();Trace.reset();System.out.println(read().number);System.out.println(Trace.text);Trace.reset();System.out.println(write().number=Trace.mark(\\\"X\\\",9));System.out.println(Trace.text);Trace.reset();System.out.println(compound().number+=Trace.mark(\\\"X\\\",2));System.out.println(Trace.text);Trace.reset();System.out.println(update().number++);System.out.println(Trace.text);Trace.reset();System.out.println(++update().number);System.out.println(Trace.text);Trace.reset();System.out.println(VolatileOwner.number++);System.out.println(VolatileOwner.number);System.out.println(VolatileOwner.box==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.inherited==Provider.box);System.out.println(Trace.text);Trace.reset();System.out.println(Child.hidden);System.out.println(Base.hidden);System.out.println(Trace.text);}\\n}\", \"src/main/java/model/Base.java\": \"package model;public class Base{public static int hidden=state.Trace.mark(\\\"B\\\",17);public static Box<Leaf> inherited=Provider.box;}\", \"src/main/java/model/Box.java\": \"package model;public class Box<T>{public T value;public Box(T v){value=v;}}\", \"src/main/java/model/Child.java\": \"package model;public class Child extends Base{public static int hidden=state.Trace.mark(\\\"D\\\",19);}\", \"src/main/java/model/CompoundOwner.java\": \"package model;public class CompoundOwner{public static int number=state.Trace.mark(\\\"C\\\",6);}\", \"src/main/java/model/Leaf.java\": \"package model;public class Leaf{public int value;public Leaf(int v){value=v;}public int key(){return value;}}\", \"src/main/java/model/Provider.java\": \"package model;public class Provider{public static Box<Leaf> box=new Box<Leaf>(new Leaf(11));public static Leaf[] refs=new Leaf[]{new Leaf(9)};public static Box<Leaf[]> arrays=new Box<Leaf[]>(refs);public static int[] numbers=new int[]{7};public static Task task=new TaskImpl();public static Leaf missing=null;public static java.lang.Class<Leaf> token=Leaf.class;public static Box<Leaf> get(){return box;}}\", \"src/main/java/model/ReadOwner.java\": \"package model;public class ReadOwner{public static int number=state.Trace.mark(\\\"R\\\",3);}\", \"src/main/java/model/Task.java\": \"package model;public abstract class Task{public abstract int key();}\", \"src/main/java/model/TaskImpl.java\": \"package model;public class TaskImpl extends Task{public int key(){return 13;}}\", \"src/main/java/model/UpdateOwner.java\": \"package model;public class UpdateOwner{public static int number=state.Trace.mark(\\\"U\\\",5);}\", \"src/main/java/model/VolatileOwner.java\": \"package model;public class VolatileOwner{public static volatile int number=state.Trace.mark(\\\"V\\\",2);public static volatile Box<Leaf> box=Provider.box;}\", \"src/main/java/model/WriteOwner.java\": \"package model;public class WriteOwner{public static int number=state.Trace.mark(\\\"W\\\",4);}\", \"src/main/java/state/Trace.java\": \"package state;public class Trace{public static String text=\\\"\\\";public static int mark(String name,int value){text+=name;return value;}public static void reset(){text=\\\"\\\";}}\"}", "Main", "origin", "Provider.arrays.value"},
	} {
		t.Run(row.name, func(t *testing.T) {
			previous := symbol.GlobalScope
			symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
			t.Cleanup(func() { symbol.GlobalScope = previous })
			var files map[string]string
			if err := json.Unmarshal([]byte(row.files), &files); err != nil {
				t.Fatal(err)
			}
			var sources []parsing.SourceFile
			for name, text := range files {
				if !strings.HasSuffix(name, ".java") {
					continue
				}
				file := parsing.SourceFile{Name: name, Source: []byte(text)}
				if err := file.ParseAST(); err != nil {
					t.Fatal(err)
				}
				symbol.AddSymbolsToPackage(file.ParseSymbols())
				sources = append(sources, file)
			}
			for _, file := range sources {
				ResolveFile(file)
			}
			var current parsing.SourceFile
			for _, file := range sources {
				if strings.HasSuffix(file.Name, "/Main.java") {
					current = file
				}
			}
			scope := current.Symbols.FindClassScope(row.class)
			ctx := Ctx{currentFile: current.Symbols, currentClass: scope, importAliases: map[string]string{}, usedImports: map[string]bool{}}
			ctx.localScope = scope.FindMethod().ByOriginalName(row.method)[0]
			var receiver *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n == nil {
					return
				}
				if n.Type() == "field_access" && n.Content(current.Source) == row.receiver {
					receiver = n
					return
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					walk(n.NamedChild(i))
				}
			}
			walk(ctx.localScope.DeclarationNode)
			if receiver == nil {
				t.Fatal("original receiver missing")
			}
			var origin inferredJavaTypeOrigin
			javaType, known := inferExprJavaType(receiver, ctx, current.Source, &origin)
			if !known || !completeJavaTypeOrigin(origin) {
				t.Fatalf("missing complete origin: %s %+v", javaType, origin)
			}
			owner := ""
			if origin.declaringOwner != nil && origin.declaringOwner.Class != nil {
				owner = origin.declaringOwner.Class.OriginalName
			}
			nominal := ""
			if origin.nominalScope != nil && origin.nominalScope.Class != nil {
				nominal = origin.nominalScope.Class.OriginalName
			}
			t.Logf("receiver=%s logical=%s owner=%s external=%s nominal=%s parameter=%v rank=%d arguments=%d", row.receiver, javaType, owner, origin.externalName, nominal, origin.parameter, origin.arrayRank, len(origin.arguments))
			for i, arg := range origin.arguments {
				t.Logf("arg%d: java=%s external=%s nominal=%p parameter=%v rank=%d complete=%v", i, arg.javaType, arg.externalName, arg.nominalScope, arg.parameter, arg.arrayRank, completeJavaTypeOrigin(arg))
			}
			selected, admitted := intrinsicReceiverTypeName(receiver, ctx, current.Source)
			t.Logf("intrinsic=%q admitted=%v", selected, admitted)
			view, erasure, erased := directOwnerFieldAccessView(receiver, ctx, current.Source)
			t.Logf("directOwnerFieldAccessView logical=%s physical=%s proved=%v", view, erasure, erased)
			var b bytes.Buffer
			if err := format.Node(&b, token.NewFileSet(), ParseExpr(receiver, current.Source, ctx)); err != nil {
				t.Fatal(err)
			}
			t.Logf("physical expression=%s", b.String())
		})
	}
}
