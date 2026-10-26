package transpiler

import "testing"

func TestCampaignSourceIteratorErasedNextBoundaryJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>iteration</groupId><artifactId>erased</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.Iterator;
class RawCursor implements Iterator<Object> {
 Object[] values;int position;
 RawCursor(Object[] values){this.values=values;}
 public boolean hasNext(){return position<values.length;}
 public Object next(){return values[position++];}
}
class Forwarder<T> implements Iterator<T> {
 Iterator<T> backing;int returned;boolean held;
 Forwarder(Iterator<T> backing){this.backing=backing;}
 public boolean hasNext(){return backing.hasNext();}
 public T next(){T value=backing.next();returned++;held=Thread.holdsLock(this);return value;}
}
public class Main {
 @SuppressWarnings({"rawtypes","unchecked"}) public static void main(String[] args){
  Object token=new Object();RawCursor raw=new RawCursor(new Object[]{token,token,token,null,"tail"});
  Forwarder<String> forward=new Forwarder((Iterator)raw);Iterator<String> view=forward;
  synchronized(forward){
   Object first=view.next();view.next();boolean caught=false;
   try{String bad=view.next();System.out.println("unexpected:"+bad);}catch(ClassCastException expected){caught=true;}
   String absent=view.next();String tail=view.next();Object alias=view;
   System.out.println((first==token)+":"+caught+":"+forward.returned+":"+raw.position+":"+(absent==null)+":"+tail+":"+forward.held+":"+(alias==forward)+":"+view.hasNext());
  }
 }
}`,
	}, "probe.Main", "true:true:5:5:true:tail:true:true:false\n")
}

func TestCampaignSourceIterableInheritedAnonymousAndCovariantJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>iteration</groupId><artifactId>source</artifactId><version>1</version></project>`,
		"src/main/java/probe/Main.java": `package probe;
import java.util.Iterator;
class Cursor implements Iterator<String> {
 String text;int calls;boolean held;
 Cursor(String text){this.text=text;}
 public boolean hasNext(){return calls==0;}
 public String next(){calls++;held=Thread.holdsLock(this);return text;}
}
class Base<T> implements Iterable<T> {
 T value;int factories;boolean held;
 Base(T value){this.value=value;}
 public Iterator<T> iterator(){factories++;held=Thread.holdsLock(this);return new Iterator<T>(){
  boolean used;
  public boolean hasNext(){return !used;}
  public T next(){used=true;return value;}
 };}
}
class Child extends Base<String>{Child(){super("inherited");}}
public class Main {public static void main(String[] args){
 Cursor cursor=new Cursor("typed");Iterator<String> view=cursor;
 synchronized(cursor){Object value=view.next();System.out.println(value+":"+cursor.calls+":"+cursor.held+":"+view.hasNext());}
 Child child=new Child();Iterable<String> source=child;
 synchronized(child){Iterator<String> values=source.iterator();String value=values.next();Object alias=source;System.out.println(value+":"+child.factories+":"+child.held+":"+values.hasNext()+":"+(alias==child));}
 Iterator<String> absent=null;boolean iteratorNull=false;try{absent.next();}catch(NullPointerException expected){iteratorNull=true;}
 Iterable<String> noFactory=()->null;boolean factoryNull=noFactory.iterator()==null;
 System.out.println(iteratorNull+":"+factoryNull);
}}
`,
	}, "probe.Main", "typed:1:true:false\ninherited:1:true:false:true\ntrue:true\n")
}

func TestCampaignSourceIterationCanonicalShadowJVM(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{
		"pom.xml":                            `<project><modelVersion>4.0.0</modelVersion><groupId>iteration</groupId><artifactId>shadows</artifactId><version>1</version></project>`,
		"src/main/java/shadow/Iterator.java": `package shadow;public class Iterator {public String next(){return "source";}public int hasNext(){return 7;}}`,
		"src/main/java/shadow/Iterable.java": `package shadow;public class Iterable {public Iterator iterator(){return new Iterator();}}`,
		"src/main/java/probe/Main.java": `package probe;import shadow.Iterator;import shadow.Iterable;
public class Main {
 static <Iterator> Iterator echo(Iterator value){return value;}
 public static void main(String[] args){Iterable source=new Iterable();Iterator iterator=source.iterator();Object sourceAlias=source;Object iteratorAlias=iterator;System.out.println(iterator.next()+":"+iterator.hasNext()+":"+echo("binder")+":"+(sourceAlias instanceof java.lang.Iterable)+":"+(iteratorAlias instanceof java.util.Iterator));}
}`,
	}, "probe.Main", "source:7:binder:false:false\n")
}
