package probe;
public class Main {
 static String log="";
 static class Item {int key;Item(int k){key=k;}}
 static class Other extends Item {Other(int k){super(k);}}
 static class Base<T extends Item> implements java.lang.Comparable<T> {
  int key;Base(int k){key=k;}
  public int compareTo(T value){log+="B"+key+":"+(value==null?"null":value.key)+";";return value==null?-17:key-value.key;}
 }
 public static void main(String[] args){
  Base<Other> source=new Base<Other>(10);java.lang.Comparable raw=source;
  Item item=new Item(3);Other other=new Other(4);
  System.out.println(raw.compareTo(item));System.out.println(raw.compareTo(other));
  source.key=20;System.out.println(raw.compareTo(item));System.out.println(raw.compareTo(null));
  try{raw.compareTo("wrong");System.out.println("bad");}catch(ClassCastException expected){System.out.println("ClassCast");}
  System.out.println(log);
 }
}
