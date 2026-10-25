package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func nativeListRecordIterationProject() map[string]string {
	return map[string]string{
		"pom.xml": `<project><groupId>probe</groupId><artifactId>native-list-iteration</artifactId><version>1</version></project>`,
		"src/main/java/probe/NativeListMain.java": `package probe;import java.util.*;
public class NativeListMain {
 record Entry(int value) {}
 public static void main(String[] args) {
  List<Entry> entries=new ArrayList<>();
  ArrayList<Entry> array=new ArrayList<>();LinkedList<Entry> linked=new LinkedList<>();
  entries.add(new Entry(7));entries.add(new Entry(11));
  array.add(new Entry(13));linked.add(new Entry(17));
  int sum=0;for(Entry entry:entries)sum+=entry.value();
  for(Entry entry:array)sum+=entry.value();for(Entry entry:linked)sum+=entry.value();
  System.out.println("entries:"+entries.size()+":"+sum);
 }
}`}
}

func TestNativeListNestedRecordElementIterationJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, nativeListRecordIterationProject(), "probe.NativeListMain", "entries:2:48\n")
}

func TestNativeListRecordIterationWithSourceAbstractListJDK21(t *testing.T) {
	files := nativeListRecordIterationProject()
	files["src/main/java/probe/Projection.java"] = `package probe;import java.util.*;
class Projection extends AbstractList<String>{public String get(int index){return "source";}public int size(){return 1;}}
`
	// Allocate and consume the source projection in the same conversion. Its
	// existence must not make native record elements lose their nominal identity.
	files["src/main/java/probe/NativeListMain.java"] = `package probe;import java.util.*;
public class NativeListMain {
 record Entry(int value) {}
 public static void main(String[] args) {
  List<String> source=new Projection();System.out.println(source.get(0));
  List<Entry> entries=new ArrayList<>();entries.add(new Entry(7));entries.add(new Entry(11));
  int sum=0;for(Entry entry:entries)sum+=entry.value();
  System.out.println("entries:"+entries.size()+":"+sum);
 }
}`
	runCampaignCompilerStrictProjectOracle(t, files, "probe.NativeListMain", "source\nentries:2:18\n")
}

func TestSourceAbstractListRecordAccessorCollisionJDK21(t *testing.T) {
	files := nativeListRecordIterationProject()
	files["src/main/java/probe/Projection.java"] = `package probe;import java.util.*;
class Projection extends AbstractList<String>{public String get(int index){return "source";}public int size(){return 1;}}
`
	files["src/main/java/probe/NativeListMain.java"] = `package probe;import java.util.*;
public class NativeListMain {
 record Entry(int JavaDynamicTypeID,int JavaDynamicTypeIDJava2goExecution) {}
 public static void main(String[] args) {
  List<String> source=new Projection();System.out.println(source.get(0));
  List<Entry> entries=new ArrayList<>();entries.add(new Entry(23,31));
  for(Entry entry:entries)System.out.println("accessors:"+entry.JavaDynamicTypeID()+":"+entry.JavaDynamicTypeIDJava2goExecution());
 }
}`
	runCampaignCompilerStrictProjectOracle(t, files, "probe.NativeListMain", "source\naccessors:23:31\n")
}

func TestSourceAbstractListRecordPollutionStillRejectsJDK21(t *testing.T) {
	files := nativeListRecordIterationProject()
	files["src/main/java/probe/Projection.java"] = `package probe;import java.util.*;
class Projection extends AbstractList<String>{public String get(int index){return "source";}public int size(){return 1;}}
`
	files["src/main/java/probe/NativeListMain.java"] = `package probe;import java.util.*;
@SuppressWarnings({"rawtypes","unchecked"}) public class NativeListMain {
 record Entry(int value) {}
 record Other(int value) {}
 public static void main(String[] args) {
  List<String> source=new Projection();System.out.println(source.get(0));
  Entry first=new Entry(7);List<Entry> entries=new ArrayList<>();entries.add(first);
  System.out.println("identity:"+(entries.get(0)==first));
  List raw=entries;raw.add(new Other(11));
  try{for(Entry entry:entries)System.out.println("value:"+entry.value());}
  catch(ClassCastException e){System.out.println("cast:true");}
 }
}`
	runCampaignCompilerStrictProjectOracle(t, files, "probe.NativeListMain", "source\nidentity:true\nvalue:7\ncast:true\n")
}

func TestSourceAbstractListExplicitRecordAccessorCollisionJDK21(t *testing.T) {
	files := nativeListRecordIterationProject()
	files["src/main/java/probe/Projection.java"] = `package probe;import java.util.*;
class Projection extends AbstractList<String>{public String get(int index){return "source";}public int size(){return 1;}}
`
	files["src/main/java/probe/NativeListMain.java"] = `package probe;import java.util.*;
public class NativeListMain {
 record Entry(int JavaDynamicTypeID,int JavaDynamicTypeIDJava2goExecution) {
  public int JavaDynamicTypeID(){return JavaDynamicTypeID+1;}
  public int JavaDynamicTypeIDJava2goExecution(){return JavaDynamicTypeIDJava2goExecution+2;}
 }
 public static void main(String[] args) {
  List<String> source=new Projection();System.out.println(source.get(0));
  List<Entry> entries=new ArrayList<>();entries.add(new Entry(23,31));
  for(Entry entry:entries)System.out.println("accessors:"+entry.JavaDynamicTypeID()+":"+entry.JavaDynamicTypeIDJava2goExecution());
 }
}`
	runCampaignCompilerStrictProjectOracle(t, files, "probe.NativeListMain", "source\naccessors:24:33\n")
}

func TestNativeListArrayIterationKeepsArrayProtocol(t *testing.T) {
	out := renderGoFileFromJava(t, `import java.util.List;class ListArrays{static void run(List<String>[] groups){for(List<String> group:groups){System.out.println(group.size());}}}`)
	assertContains(t, out, "range stdjava.ReferenceArrayIterationElements(groups)")
}

func TestSourceAbstractListRecordSourceInterfaceIdentityJDK21(t *testing.T) {
	files := nativeListRecordIterationProject()
	files["src/main/java/probe/Projection.java"] = `package probe;import java.util.*;
class Projection extends AbstractList<String>{public String get(int index){return "source";}public int size(){return 1;}}
`
	files["src/main/java/probe/NativeListMain.java"] = `package probe;import java.util.*;
interface HasValue{int value();}
public class NativeListMain {
 record Entry(int value) implements HasValue {}
 public static void main(String[] args) {
  List<String> source=new Projection();System.out.println(source.get(0));
  Entry first=new Entry(37);List<Entry> entries=new ArrayList<>();entries.add(first);
  for(Entry entry:entries){Object object=entry;HasValue view=(HasValue)object;
   System.out.println("interface:"+view.value()+":"+((Object)view==object));}
 }
}`
	runCampaignCompilerStrictProjectOracle(t, files, "probe.NativeListMain", "source\ninterface:37:true\n")
}

func TestSourceAbstractListDirectRootKeepsFullProof(t *testing.T) {
	for _, row := range []struct{ name, source string }{
		{"qualified", `class Values extends java.util.AbstractList<String>{public String get(int i){return "value";}public int size(){return 1;}}`},
		{"simple", `import java.util.AbstractList;class Values extends AbstractList<String>{public String get(int i){return "value";}public int size(){return 1;}}`},
		{"commented", `import /*header*/ java.util./*part*/AbstractList;class Values extends AbstractList<String>{public String get(int i){return "value";}public int size(){return 1;}}`},
		{"source-shadow", `class AbstractList<T>{}class Values extends AbstractList<String>{}`},
		{"qualified-source-shadow", `package own;class AbstractList<T>{}class Values extends own.AbstractList<String>{}`},
		{"unrelated", `class Parent<T>{}class Values extends Parent<String>{}`},
		{"unicode-as-parsed", `class Values extends java.util.\u0041bstractList<String>{public String get(int i){return "value";}public int size(){return 1;}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			owner := h.File.Symbols.FindClassScope("Values")
			declaring, owned := sourceAbstractListDeclaration(owner, h.Ctx)
			original := false
			if owned {
				edge := declaredJavaTypeOrigin(symbol.JavaType{Original: owner.Superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(declaring))}, declaring)
				original = !edge.unresolved && !edge.wildcard && edge.nominalScope == nil && edge.parameter == nil && edge.arrayRank == 0 && edge.externalName == "java.util.AbstractList" && len(edge.arguments) <= 1 && sourceArrayListNativeConstructors(owner, declaring)
			}
			if got := sourceAbstractListDirectRoot(owner, h.Ctx); got != original {
				t.Fatalf("direct root=%t original full proof=%t", got, original)
			}
			detached := *owner
			if sourceAbstractListDirectRoot(&detached, h.Ctx) {
				t.Fatal("candidate spelling donated ownership to detached scope")
			}
		})
	}
}
