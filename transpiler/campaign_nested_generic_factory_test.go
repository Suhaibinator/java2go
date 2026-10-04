package transpiler

import (
	"strings"
	"testing"
)

// This source is the independently minimized frozen Gson factory contract.
// Preserve it as the acceptance gate while smaller erased-layout slices land.
func TestCampaignNestedGenericFactoryIdentity(t *testing.T) {
	files := map[string]string{
		"pom.xml": `<project><groupId>example</groupId><artifactId>generic-factory</artifactId><version>1</version></project>`,
		"src/main/java/example/GsonGenericFactoryRepro.java": `package example;
public final class GsonGenericFactoryRepro {
  static final class Token<T> {
    final String label;
    Token(String label) { this.label = label; }
  }
  static final class Adapter<T> {
    private T value;
    Adapter(T value) { this.value = value; }
    T read() { return value; }
    void write(T value) { this.value = value; }
  }
  interface Factory {
    <T> Adapter<T> create(Token<T> type);
  }
  static Factory newFactory() {
    Adapter<Number> shared = new Adapter<Number>(7);
    return new Factory() {
      @SuppressWarnings("unchecked")
      public <T> Adapter<T> create(Token<T> type) {
        return type.label.equals("number") ? (Adapter<T>) shared : null;
      }
    };
  }
  public static void main(String[] args) {
    Factory factory = newFactory();
    Token<Number> number = new Token<Number>("number");
    Token<String> string = new Token<String>("string");
    Adapter<Number> first = factory.create(number);
    Adapter<Number> again = factory.create(number);
    Adapter<String> missing = factory.create(string);
    System.out.println("same=" + (first == again) + ",missing=" + (missing == null) + ",value=" + first.read());
    first.write(42);
    System.out.println("updated=" + again.read() + ",missing2=" + (factory.create(string) == null));
  }
}
`,
	}
	// Keep the original class name as acceptance. Its shortened receiver is
	// currently a separate Go-keyword bug; an alpha-renamed companion exposes
	// the nested generic descriptor failure without changing the original.
	for _, name := range []string{"GsonGenericFactoryRepro", "FactoryContract"} {
		t.Run(name, func(t *testing.T) {
			runCampaignCompilerProjectOracle(t, map[string]string{
				"pom.xml": files["pom.xml"],
				"src/main/java/example/" + name + ".java": strings.ReplaceAll(files["src/main/java/example/GsonGenericFactoryRepro.java"], "GsonGenericFactoryRepro", name),
			}, "example."+name, "same=true,missing=true,value=7\nupdated=42,missing2=true\n")
		})
	}
}

func TestCampaignGenericLeafRawPollutionTiming(t *testing.T) {
	files := map[string]string{
		"pom.xml":                        `<project><groupId>example</groupId><artifactId>generic-leaf</artifactId><version>1</version></project>`,
		"src/main/java/example/Box.java": `package example;public final class Box<T>{public T value;public Box(T value){this.value=value;}public T read(){T result=(T)value;return result;}public void write(T value){T incoming=value;this.value=incoming;}}`,
		"src/main/java/example/Main.java": `package example;public class Main{
 @SuppressWarnings({"rawtypes","unchecked"}) public static void main(String[] args){
  Box<Integer> typed=new Box<Integer>(7);Box raw=typed;
  System.out.println("same="+(typed==raw));
  raw.write("method-pollution");System.out.println("write=ok");
  Object broad=typed.read();System.out.println(broad);
  try{Integer narrow=typed.read();System.out.println("wrong="+narrow);}catch(ClassCastException expected){System.out.println("read=cast");}
  typed.write(42);System.out.println(typed.read());
  raw.value="field-pollution";System.out.println("field-write=ok");
  Object broadField=typed.value;System.out.println(broadField);
  try{Integer narrowField=typed.value;System.out.println("wrong="+narrowField);}catch(ClassCastException expected){System.out.println("field-read=cast");}
  raw.write(null);System.out.println("null="+(typed.read()==null));
 }
}`,
	}
	runCampaignCompilerProjectOracle(t, files, "example.Main", "same=true\nwrite=ok\nmethod-pollution\nread=cast\n42\nfield-write=ok\nfield-pollution\nfield-read=cast\nnull=true\n")
}

func TestCampaignCanonicalFactoryAliasContracts(t *testing.T) {
	files := map[string]string{
		"pom.xml":                            `<project><groupId>example</groupId><artifactId>canonical-factory</artifactId><version>1</version></project>`,
		"src/main/java/example/Box.java":     `package example;public final class Box<T>{public T value;public Box(T value){this.value=value;}public T read(){return value;}public void write(T value){this.value=value;}}`,
		"src/main/java/example/Factory.java": `package example;public interface Factory{<T> Box<T> identity(Box<T> value);}`,
		"src/main/java/example/Named.java":   `package example;public class Named<T> implements Factory{private final T tag;public Named(T tag){this.tag=tag;}public T tag(){return tag;}public <T> Box<T> identity(Box<T> value){return value;}}`,
		"src/main/java/example/Main.java": `package example;
public class Main {
 @SuppressWarnings({"rawtypes","unchecked"}) public static void main(String[] args){
  Named<String> named=new Named<String>("outer");Factory anonymous=new Factory(){public <U> Box<U> identity(Box<U> value){return value;}};
  Factory[] factories=new Factory[]{named,anonymous};
  for(Factory factory:factories){
   Box<String> typed=new Box<String>("value");Box<String> again=factory.identity(typed);Box raw=again;
   Box<Integer> pollutedView=(Box<Integer>)(Box)typed;
   System.out.println("same="+(typed==again)+":"+((Object)typed==(Object)pollutedView));
   System.out.println("null="+(factory.identity((Box<String>)null)==null));
   System.out.println("class="+(typed.getClass()==Box.class)+":"+(pollutedView.getClass()==typed.getClass()));
   Box[] slots=new Box[]{typed};Object[] broad=slots;broad[0]=again;System.out.println("array="+(slots[0]==typed));
   try{broad[0]="wrong";System.out.println("bad-store");}catch(ArrayStoreException expected){System.out.println("array=checked");}
   synchronized(typed){synchronized(again){System.out.println("monitor="+Thread.holdsLock(raw));}}
   raw.write(17);Object observed=typed.read();System.out.println("broad="+observed);
   try{String narrow=typed.read();System.out.println("bad-read="+narrow);}catch(ClassCastException expected){System.out.println("read=checked");}
   typed.write("restored");System.out.println(again.read());
  }
  System.out.println(named.tag());
 }
}`,
	}
	const iteration = "same=true:true\nnull=true\nclass=true:true\narray=true\narray=checked\nmonitor=true\nbroad=17\nread=checked\nrestored\n"
	runCampaignCompilerProjectOracle(t, files, "example.Main", iteration+iteration+"outer\n")
}
