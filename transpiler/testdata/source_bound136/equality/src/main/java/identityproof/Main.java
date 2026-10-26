package identityproof;
public class Main {
 static String events="";
 static class Item { int key; Item(int key){this.key=key;} }
 static class Other extends Item { int extra; Other(int key){super(key);extra=key+1;} }
 static class Box<T extends Item> implements java.lang.Comparable<T> {
  T value;
  Box(T value){this.value=value;}
  T get(){events+="G";return value;}
  void put(T value){events+="P";this.value=value;}
  public int compareTo(T other){return other==null?0:value.key-other.key;}
 }
 static Box<Other> receiver(Box<Other> box){events+="R";return box;}
 static Other rhs(Other value){events+="S";return value;}
 static Other fail(){events+="X";throw new IllegalStateException("argument");}
 public static void main(String[] args){
  Other other=new Other(4);Item item=new Item(3);Box<Other> box=new Box<Other>(other);Box raw=box;
  events="";System.out.println((receiver(box).value==rhs(other))+":"+events);
  events="";System.out.println((receiver(box).get()!=rhs(other))+":"+events);
  raw.put(item);events="";System.out.println((receiver(box).value==rhs(other))+":"+events);
  events="";System.out.println((receiver(box).get()!=rhs(other))+":"+events);
  events="";try{System.out.println(((Other)receiver(box).get())==rhs(other));}catch(ClassCastException expected){System.out.println("cast:"+events);}
  events="";try{System.out.println(receiver(box).get()==fail());}catch(IllegalStateException expected){System.out.println("abrupt:"+events);}
  Other typedNull=null;events="";raw.put(typedNull);System.out.println((box.get()==typedNull)+":"+events);
  events="";System.out.println((receiver(box).get()==rhs(typedNull))+":"+events);
  events="";try{System.out.println(receiver(box).get().extra);}catch(NullPointerException expected){System.out.println("null:"+events);}
  events="";try{System.out.println(receiver(null).get()==rhs(other));}catch(NullPointerException expected){System.out.println("receiver-null:"+events);}
  events="";try{System.out.println(((Other)receiver(box).get()).extra+fail().extra);}catch(NullPointerException expected){System.out.println("null-cast:"+events);}
 }
}
