package cachecontrol;
import java.util.*;
public class Bridge {
 static String log="";
 static class Item {int key;Item(int k){key=k;}}
 static class Other extends Item {Other(int k){super(k);}}
 static class Base<T extends Item> implements Comparable<T> {int key;Base(int k){key=k;}public int compareTo(T other){log+="B"+key+":"+other.key+";";return key-other.key;}}
 static class Child extends Base<Item> {Child(int k){super(k);}public int compareTo(Item other){log+="C"+key+":"+other.key+";";return key-other.key;}public int compareTo(Other other){log+="O";return 88;}}
 static class Inherited extends Base<Item> {Inherited(int k){super(k);}}
 public static void main(String[] args){Child a=new Child(10),b=new Child(30);Comparable raw=a;Item first=new Item(2);Other second=new Other(3);
  System.out.println(raw.compareTo(second));System.out.println(raw.compareTo(first));a.key=40;System.out.println(raw.compareTo(second));raw=b;System.out.println(raw.compareTo(first));System.out.println(a.compareTo(second));
  try{raw.compareTo("wrong");System.out.println("bad");}catch(ClassCastException expected){System.out.println("ClassCast");}
  Inherited x=new Inherited(14),y=new Inherited(50);System.out.println(((Comparable)x).compareTo(first));x.key=20;System.out.println(((Comparable)x).compareTo(second));System.out.println(((Comparable)y).compareTo(first));System.out.println(log);
 }
}
