package transpiler

import (
	"bytes"
	"encoding/json"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestErasedFieldArrayOperandWorkflowJVM(t *testing.T) {
	var files map[string]string
	if err := json.Unmarshal([]byte(`{
  "pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>arrayfield</groupId><artifactId>operand</artifactId><version>1</version></project>",
  "src/main/java/app/Main.java": "package app;import model.*;public class Main{\n static class java{static class lang{static class Class{}}}\n static<Leaf>void probe(){\n System.out.println(Provider.refs.value.length);System.out.println(Provider.refs.value[0].key());Provider.refs.value[0]=new model.Leaf(9);for(model.Leaf v:Provider.refs.value)System.out.println(v.key());\n System.out.println(Provider.matrix.value[0][0].key());System.out.println(Provider.ints.value[0]++);System.out.println(Provider.ints.value[0]+=Trace.rhs());System.out.println(Provider.boxed.value[0]++);System.out.println(Provider.boxed.value[0]+=Trace.rhs());System.out.println(Provider.bases.value[0].key());\n Object saved=Provider.refs.value;Provider.put(Provider.refs,new Object[]{new Other()});Object erased=Provider.refs.value;System.out.println(erased!=null);\n Trace.reset();try{System.out.println(Provider.refs.value[Trace.index()].key());}catch(ClassCastException e){System.out.println(\"read:\"+Trace.text);}\n Trace.reset();try{Provider.refs.value[Trace.index()]=Trace.leaf();}catch(ClassCastException e){System.out.println(\"store:\"+Trace.text);}\n Trace.reset();try{System.out.println(Provider.refs.value.length);}catch(ClassCastException e){System.out.println(\"length:\"+Trace.text);}\n Trace.reset();try{for(model.Leaf v:Provider.refs.value){Trace.text+=\"L\";}}catch(ClassCastException e){System.out.println(\"loop:\"+Trace.text);}\n Provider.put(Provider.refs,null);Trace.reset();try{System.out.println(Provider.refs.value[Trace.index()].key());}catch(NullPointerException e){System.out.println(\"nullread:\"+Trace.text);}\n Trace.reset();try{Provider.refs.value[Trace.index()]=Trace.leaf();}catch(NullPointerException e){System.out.println(\"nullstore:\"+Trace.text);}\n Provider.put(Provider.refs,saved);System.out.println(Provider.refs.value==saved);System.out.println(Provider.refs.value[0].key());\n Provider.put(Provider.ints,new long[]{9});Trace.reset();try{System.out.println(Provider.ints.value[Trace.index()]);}catch(ClassCastException e){System.out.println(\"primitive:\"+Trace.text);}\n Provider.put(Provider.matrix,new model.Leaf[]{new model.Leaf(1)});Trace.reset();try{System.out.println(Provider.matrix.value[Trace.index()][0].key());}catch(ClassCastException e){System.out.println(\"rank:\"+Trace.text);}\n }\n public static void main(String[]args){probe();}}\n",
  "src/main/java/model/Base.java": "package model;public class Base{public int n;public Base(int n){this.n=n;}public int key(){return n;}}",
  "src/main/java/model/Box.java": "package model;public class Box<T>{public T value;public Box(T v){value=v;}}",
  "src/main/java/model/Leaf.java": "package model;public class Leaf extends Base{public Leaf(int n){super(n);}}",
  "src/main/java/model/Other.java": "package model;public class Other{}",
  "src/main/java/model/Provider.java": "package model;public class Provider{public static Box<Leaf[]> refs=new Box<Leaf[]>(new Leaf[]{new Leaf(7)});public static Box<int[]> ints=new Box<int[]>(new int[]{3});public static Box<Integer[]> boxed=new Box<Integer[]>(new Integer[]{4});public static Box<Leaf[][]> matrix=new Box<Leaf[][]>(new Leaf[][]{{new Leaf(6)}});public static Box<Base[]> bases=new Box<Base[]>(new Leaf[]{new Leaf(5)});public static void put(Box raw,Object value){raw.value=value;}}",
  "src/main/java/model/Trace.java": "package model;public class Trace{public static String text=\"\";public static int index(){text+=\"I\";return 0;}public static int rhs(){text+=\"R\";return 2;}public static Leaf leaf(){text+=\"R\";return new Leaf(8);}public static void reset(){text=\"\";}}"
}`), &files); err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, "app.Main", []campaignCompilerProjectObservation{
		{name: "repeat1", stdout: "1\n7\n9\n6\n3\n6\n4\n7\n5\ntrue\nread:\nstore:\nlength:\nloop:\nnullread:I\nnullstore:IR\ntrue\n9\nprimitive:\nrank:\n", stderr: ""},
		{name: "repeat2", stdout: "1\n7\n9\n6\n3\n6\n4\n7\n5\ntrue\nread:\nstore:\nlength:\nloop:\nnullread:I\nnullstore:IR\ntrue\n9\nprimitive:\nrank:\n", stderr: ""},
		{name: "repeat3", stdout: "1\n7\n9\n6\n3\n6\n4\n7\n5\ntrue\nread:\nstore:\nlength:\nloop:\nnullread:I\nnullstore:IR\ntrue\n9\nprimitive:\nrank:\n", stderr: ""},
	})
}

func TestErasedFieldArrayOperandUsesActualOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Item{}class Box<T>{T value;}class Provider{static Box<Item[]> refs;static Item[] direct;static Object unknown;}class Main{static<Item>void probe(){Object a=Provider.refs.value;Object b=Provider.direct;Object c=Provider.unknown;}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Main")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	nodes := map[string]*sitter.Node{}
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.Type() == "field_access" {
			nodes[n.Content(h.File.Source)] = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	once := ast.NewIdent("once")
	ctx.expectedType = "Item[][]"
	ctx.expectedTypeRoot = ctx.localScope.DeclarationNode
	projected := projectErasedFieldArrayOperand(once, nodes["Provider.refs.value"], ctx, h.File.Source)
	var output bytes.Buffer
	if err := format.Node(&output, token.NewFileSet(), projected); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "JavaArrayCast[*stdjava.ReferenceArray]") || strings.Count(got, "ArrayTypeID(") != 1 || !strings.Contains(got, `"Item"`) {
		t.Fatalf("caller binder/target changed actual field array: %s", got)
	}
	if ctx.expectedType != "Item[][]" || ctx.expectedTypeRoot != ctx.localScope.DeclarationNode {
		t.Fatal("array proof changed caller target")
	}
	for _, name := range []string{"Provider.direct", "Provider.unknown"} {
		if projected := projectErasedFieldArrayOperand(once, nodes[name], ctx, h.File.Source); projected != once {
			t.Errorf("%s acquired unproved erased-field cast", name)
		}
	}
	if projected := projectErasedFieldArrayOperand(once, nodes["Provider.refs.value"], Ctx{}, h.File.Source); projected != once {
		t.Fatal("default context acquired array proof")
	}
}
