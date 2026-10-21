package boundproof;
public class Main {
 static String events="";
 static class Item {int key; Item(int key){this.key=key;}}
 static class Other extends Item {int extra; Other(int key,int extra){super(key);this.extra=extra;}}
 static class Box<T extends Item> implements java.lang.Comparable<T> {
  T value;
  Box(T value,int marker){events+="C";this.value=value;}
  T get(){events+="G";return value;}
  void put(T value){events+="P";this.value=value;}
  public int compareTo(T other){events+="Q";return other==null?-17:value.key-other.key;}
  <T> T shadow(T value){return value;}
 }
 static class Child<T extends Item> extends Box<T> {
  Child(T value,int marker){super(value,marker);}
  T get(){events+="H";return super.get();}
 }
 static Other receiver(Other value){events+="R";return value;}
 static int effect(){events+="E";return 1;}
 static Child<Other> boxReceiver(Child<Other> value){events+="B";return value;}
 public static void main(String[] args){
  Other other=new Other(4,9);Item item=new Item(3);
  Child<Other> typed=new Child<Other>(receiver(other),effect());Box raw=typed;
  System.out.println(events);events="";
  System.out.println(typed.get().extra);System.out.println(typed.value.extra);
  System.out.println(typed.shadow("shadow"));
  raw.put(item);System.out.println(raw.get()==item);System.out.println(typed==raw);
  typed.get();System.out.println("discard:"+events);events="";
  try{System.out.println(boxReceiver(typed).get().extra);}catch(ClassCastException expected){System.out.println("late:"+events);}
  events="";try{Other result=typed.value;System.out.println(result.extra);}catch(ClassCastException expected){System.out.println("field:"+events);}
  events="";java.util.function.Supplier<Other> reference=typed::get;
  try{Other result=reference.get();System.out.println(result.extra);}catch(ClassCastException expected){System.out.println("ref:"+events);}
  typed.put(null);System.out.println(typed.get()==null);System.out.println(raw.get()==null);
  events="";try{System.out.println(typed.get().extra);}catch(NullPointerException expected){System.out.println("null:"+events);}
  raw.put(other);java.lang.Comparable comparable=typed;
  System.out.println(comparable.compareTo(item));System.out.println(comparable.compareTo(null));
  events="";try{comparable.compareTo("wrong");}catch(ClassCastException expected){System.out.println("bound:"+events);}
  Object alien=new Object();events="";
  try{new Box<Other>((Other)alien,effect());}catch(ClassCastException expected){System.out.println("constructor-cast:"+events);}
  events="";Box<Other> nullable=new Box<Other>(receiver(null),effect());System.out.println((nullable.get()==null)+":"+events);
 }
}
