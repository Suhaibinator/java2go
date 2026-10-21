package cachecontrol;
import java.util.*;
public class Execution {
 static ThreadLocal<String> local=new ThreadLocal<>();static String log="";static RuntimeException marker=new RuntimeException("marker");
 static class Value implements Comparable<Value> {int key;boolean fail;Value(int k){key=k;}public int compareTo(Value right){synchronized(this){log+=local.get()+":"+Thread.holdsLock(this)+":"+key+":"+(right==null?-1:right.key)+";";if(fail)throw marker;return right==null?key:key-right.key;}}}
 static Value receiver(Value v){log+="R;";return v;}static Value argument(Value v){log+="A;";return v;}
 public static void main(String[] args){Value a=new Value(9),b=new Value(21),r=new Value(2);local.set("one");System.out.println(((Comparable)receiver(a)).compareTo(argument(r)));r.key=5;local.set("two");System.out.println(((Comparable)b).compareTo(r));a.key=40;System.out.println(((Comparable)a).compareTo(r));System.out.println(((Comparable)a).compareTo(null));a.fail=true;try{((Comparable)a).compareTo(r);}catch(RuntimeException e){System.out.println(e==marker);}a.fail=false;System.out.println(((Comparable)a).compareTo(r));System.out.println(log);}
}
