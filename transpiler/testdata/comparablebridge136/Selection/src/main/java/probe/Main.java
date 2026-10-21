package probe;
public class Main {
 static String log="";static int calls=0;
 static class Item {int key;Item(int k){key=k;}}
 static class Other extends Item {Other(int k){super(k);}}
 static class Base<T extends Item> implements java.lang.Comparable<T> {
  int key;Base(int k){key=k;log+="ctor:"+((java.lang.Comparable)this).compareTo(new Item(1))+";";}
  public int compareTo(T value){log+="B"+key+";";return value==null?-19:key-value.key;}
 }
 static class Child extends Base<Item> {
  int bonus=5;Child(int k){super(k);bonus=9;}
  public int compareTo(Item value){log+="C"+key+":"+bonus+";";return value==null?-23:key+bonus-value.key;}
  public int compareTo(Other value){log+="O;";return 88;}
 }
 static class Inherited extends Base<Item> {Inherited(int k){super(k);}}
 static java.lang.Comparable receiver(Child value){log+="R;";calls++;return value;}
 static Item argument(Item value){log+="A;";calls++;return value;}
 public static void main(String[] args){
  Child one=new Child(10),two=new Child(30);Item item=new Item(2);Other other=new Other(3);
  System.out.println(receiver(one).compareTo(argument(other)));System.out.println(calls);
  java.lang.Comparable raw=two;System.out.println(raw.compareTo(item));one.key=40;raw=one;System.out.println(raw.compareTo(item));
  System.out.println(one.compareTo(other));System.out.println(raw.compareTo(null));
  try{raw.compareTo("wrong");System.out.println("bad");}catch(ClassCastException expected){System.out.println("ClassCast");}
  Inherited inherited=new Inherited(50);System.out.println(((java.lang.Comparable)inherited).compareTo(item));System.out.println(log);
 }
}
