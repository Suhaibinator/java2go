package consumer;
import producer.Box;
import model.Other;
public class Main {
 static class Item {int key=99;}
 static class T {int key=98;}
 static class java {static class lang {interface Comparable<T>{int compareTo(T value);}}}
 static <T> int use(Box<Other> value,T ignored){return value.read().extra;}
 public static void main(String[] args){
  Other value=new Other(7,11);Box<Other> box=new Box<Other>(value);
  System.out.println(use(box,new T()));System.out.println(box.echo(new Item()).key);
  System.out.println(box.value==value);Box raw=box;model.Item base=new model.Item(3);raw.put(base);
  System.out.println(raw.read()==base);try{System.out.println(box.read().extra);}catch(ClassCastException expected){System.out.println("late");}
  box.put(null);System.out.println(box.read()==null);
 }
}
