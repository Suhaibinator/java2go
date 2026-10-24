package probe;
public class Main {
 static String log="";
 static class Item {int key;Item(int k){key=k;}}
 static class Other extends Item {Other(int k){super(k);}}
 static class Value implements java.lang.Comparable<Item> {
  int key;int CompareToJava2goExecution=12;int CompareToJava2goExecution1=13;
  Value(int k){key=k;}
  public int compareTo(Other value){log+="O;";return 71;}
  public int compareTo(Item value){log+="I"+key+";";return value==null?-31:key-value.key;}
  public int compareToJava2goExecution(Object value){log+="D;";return 901;}
  public int compareToJava2goExecution1(Object value){log+="E;";return 902;}
 }
 public static void main(String[] args){Value one=new Value(10),two=new Value(20);Other other=new Other(2);java.lang.Comparable raw=one;
  System.out.println(raw.compareTo(other));raw=two;System.out.println(raw.compareTo(new Item(3)));System.out.println(one.compareTo(other));
  System.out.println(raw.compareTo(null));try{raw.compareTo("wrong");System.out.println("bad");}catch(ClassCastException expected){System.out.println("ClassCast");}
  System.out.println(one.compareToJava2goExecution(other));System.out.println(one.compareToJava2goExecution1(other));System.out.println(one.CompareToJava2goExecution+two.CompareToJava2goExecution1);System.out.println(log);
 }
}
