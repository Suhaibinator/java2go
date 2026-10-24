package transpiler

import "testing"

// The abstract adapter, generic forwarding superclass, captured inner null-safe
// adapter, factory descriptor and stateful anonymous implementation follow the
// frozen Gson hierarchy's representation requirements without replacing Gson.
func TestCampaignAbstractGenericFactoryFamily(t *testing.T) {
	files := map[string]string{
		"pom.xml":                                `<project><groupId>example</groupId><artifactId>abstract-factory</artifactId><version>1</version></project>`,
		"src/main/java/example/ValueAccess.java": `package example;public interface ValueAccess<T>{T read();void write(T value);default T again(){return read();}}`,
		"src/main/java/example/ValueAccessJava2goErased.java":                `package example;public class ValueAccessJava2goErased{}`,
		"src/main/java/example/ValueAccessJava2goDefaultsJava2goErased.java": `package example;public class ValueAccessJava2goDefaultsJava2goErased{}`,
		"src/main/java/example/Adapter.java": `package example;
public abstract class Adapter<T> implements ValueAccess<T>{
 public abstract T read();public abstract void write(T value);
 public final Adapter<T> nullSafe(){if(this instanceof Adapter.NullSafe){return this;}return new NullSafe();}
 private final class NullSafe extends Adapter<T>{public T read(){return Adapter.this.read();}public void write(T value){if(value!=null){Adapter.this.write(value);}}}
}`,
		"src/main/java/example/Intermediate.java": `package example;public abstract class Intermediate<X> extends Adapter<X>{public final X twice(){read();return read();}}`,
		"src/main/java/example/Memory.java":       `package example;public class Memory<T> extends Intermediate<T>{public T value;public int reads;public int writes;public Memory(T value){this.value=value;}public T read(){reads++;return value;}public void write(T value){writes++;this.value=value;}}`,
		"src/main/java/example/Specialized.java":  `package example;public class Specialized extends Intermediate<String>{private String value="special";public int entered;public String read(){return value;}public void write(String value){entered++;this.value=value;}}`,
		"src/main/java/example/Token.java":        `package example;public final class Token<T>{public final String label;public Token(String label){this.label=label;}}`,
		"src/main/java/example/Factory.java":      `package example;public interface Factory{<T> Adapter<T> create(Token<T> type);}`,
		"src/main/java/example/Selector.java":     `package example;public class Selector<S>{private final Adapter<S> fallback;public Selector(Adapter<S> value){fallback=value;}@SuppressWarnings("unchecked") public <S> Adapter<S> select(Token<S> token,Factory factory){Adapter<S> result=factory.create(token);return result==null?(Adapter<S>)fallback:result;}}`,
		"src/main/java/example/Main.java": `package example;
public class Main{
 @SuppressWarnings({"unchecked","rawtypes"}) public static void main(String[] args){
  Memory<String> memory=new Memory<String>("seed");Adapter<String> wrapped=memory.nullSafe();
  Factory factory=new Factory(){public <T> Adapter<T> create(Token<T> type){return type.label.equals("known")?(Adapter<T>)wrapped:null;}};
  Selector<String> selector=new Selector<String>(wrapped);Adapter<String> first=selector.select(new Token<String>("known"),factory);Adapter<String> again=factory.create(new Token<String>("known"));
  System.out.println("same="+(first==again));System.out.println("wrapper="+(first.nullSafe()==first));
  first.write("updated");ValueAccess<String> access=first;System.out.println(access.again());first.write(null);System.out.println("writes="+memory.writes);
  System.out.println("missing="+(factory.create(new Token<Integer>("missing"))==null));
  ValueAccess raw=memory;raw.write(17);System.out.println("broad="+raw.read());
  try{String narrow=memory.read();System.out.println("wrong="+narrow);}catch(ClassCastException expected){System.out.println("delayed="+memory.reads);}
  Specialized specialized=new Specialized();ValueAccess wrong=specialized;
  try{wrong.write(17);System.out.println("wrong-bridge");}catch(ClassCastException expected){System.out.println("bridge="+specialized.entered);}
  specialized.write("specialized");System.out.println(specialized.twice());
  Adapter<String> anonymous=new Adapter<String>(){String value="anonymous";public String read(){return value;}public void write(String value){this.value=value;}};
  anonymous.write("changed");System.out.println(anonymous.read());
  Adapter<?>[] array=new Adapter<?>[]{memory,anonymous,wrapped};Object[] broad=array;
  try{broad[0]=new Token<String>("wrong");System.out.println("wrong-array");}catch(ArrayStoreException expected){System.out.println("array=checked");}
  System.out.println("class="+(memory.getClass()==Memory.class)+":"+(anonymous.getClass()!=Adapter.class));
  ValueAccess<String> facet=memory;synchronized(memory){System.out.println("monitor="+Thread.holdsLock(facet));}
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "same=true\nwrapper=true\nupdated\nwrites=1\nmissing=true\nbroad=17\ndelayed=3\nbridge=0\nspecialized\nchanged\narray=checked\nclass=true:true\nmonitor=true\n")
}
