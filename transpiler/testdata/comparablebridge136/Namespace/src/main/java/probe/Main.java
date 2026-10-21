package probe;
import foreign.Value;
import java.lang.Comparable;
public class Main {
 static class Item {int key;Item(int k){key=k;}}
 static class java {static class lang {interface Comparable<T>{int compareTo(T value);}}}
 static class Source implements java.lang.Comparable<Source>{int key;Source(int k){key=k;}public int compareTo(Source value){return key-value.key;}}
 public static void main(String[] args){Value one=new Value(8),two=new Value(3);foreign.Comparable<Value> foreign=one;System.out.println(foreign.compareTo(two));
  Source source=new Source(11);System.out.println(source.compareTo(new Source(4)));
  Canonical<Object> canonical=new Canonical<Object>(14);Comparable raw=canonical;System.out.println(raw.compareTo(new Item(2)));System.out.println(raw.compareTo(null));
  System.out.println((((Object)one) instanceof Comparable)+":"+(((Object)source) instanceof Comparable)+":"+(((Object)canonical) instanceof Comparable));
 }
}
