package app;
import model.Item;import model.Other;
public class Main {
 public static String trace="";static int calls;static RuntimeException abrupt=new RuntimeException("body");
 static ThreadLocal<String> context=new ThreadLocal<String>();
 static class Value implements java.lang.Comparable<Item> {
  int key;Value(int key){this.key=key;}
  public synchronized int compareTo(Item value){trace+="V"+key+":"+context.get()+";";calls++;if(key==99)throw abrupt;return value==null?-17:key-value.key;}
  public int compareTo(Other value){trace+="other;";return 77;}
 }
 static class Generic<T> implements java.lang.Comparable<Item> {int key;Generic(int key){this.key=key;}public int compareTo(Item value){trace+="G"+key+";";return key-value.key;}}
 static class Shadow extends Child {Shadow(int key){super(key);}public int compareToJava2goErasedExecution1(Object value){trace+="S;";return 902;}}
 static Value receiver(Value value){trace+="R;";return value;}static Item argument(int key){trace+="A;";return new Item(key);}
 static Value throwingReceiver(){trace+="X;";throw abrupt;}static Item throwingArgument(){trace+="Y;";throw abrupt;}
 public static void main(String[] args){
  context.set("live");Value a=new Value(10),b=new Value(30);java.lang.Comparable raw=a;
  try{raw.compareTo("wrong");System.out.println("bad");}catch(ClassCastException expected){System.out.println("cold-cast");}
  System.out.println(raw.compareTo(new Other(2)));raw=b;System.out.println(raw.compareTo(new Item(3)));raw=a;System.out.println(raw.compareTo(null));
  try{raw.compareTo(new Item[1]);System.out.println("bad");}catch(ClassCastException expected){System.out.println("array-cast");}
  System.out.println(((java.lang.Comparable)receiver(a)).compareTo(argument(4)));
  try{((java.lang.Comparable)throwingReceiver()).compareTo(argument(5));}catch(RuntimeException expected){System.out.println(expected==abrupt);}
  try{((java.lang.Comparable)receiver(a)).compareTo(throwingArgument());}catch(RuntimeException expected){System.out.println(expected==abrupt);}
  try{((java.lang.Comparable)new Value(99)).compareTo(new Item(1));}catch(RuntimeException expected){System.out.println(expected==abrupt);}
  try{((java.lang.Comparable)null).compareTo(argument(6));}catch(NullPointerException expected){System.out.println("null-receiver");}
  Child child=new Child(20);System.out.println(((java.lang.Comparable)child).compareTo(new Other(7)));System.out.println(child.compareTo(new Other(7)));System.out.println(child.compareToJava2goErasedExecution(new Object()));
  Shadow shadow=new Shadow(40);System.out.println(((java.lang.Comparable)shadow).compareTo(new Item(8)));System.out.println(shadow.compareToJava2goErasedExecution1(new Object()));
  System.out.println(((java.lang.Comparable)new DefaultValue()).compareTo(new Item(9)));System.out.println(((java.lang.Comparable)new DefaultValue()).compareTo(null));
  System.out.println(((java.lang.Comparable)new Generic<String>(50)).compareTo(new Item(10)));System.out.println(((java.lang.Comparable)new Generic<Integer>(70)).compareTo(new Item(11)));
  System.out.println(calls);System.out.println(trace);
 }
}
