package staticcast;
class Object{}
class Base extends Object{}
class Child extends Base{}
class Alien extends Object{}
public class Main{
 static int calls,after;
 static <T extends Base>T bounded(java.lang.Object value){calls++;T result=(T)value;after++;return result;}
 static <T extends Object>T source(java.lang.Object value){calls++;T result=(T)value;after++;return result;}
 static <T>T universal(java.lang.Object value){calls++;T result=(T)value;after++;return result;}
 public static void main(String[]args){
  Base base=new Base();java.lang.Object a=Main.<Base>bounded(base);System.out.println("base:"+(a==base)+":"+calls+":"+after);
  Child child=new Child();java.lang.Object b=Main.<Child>bounded(child);System.out.println("child:"+(b==child)+":"+calls+":"+after);
  Child nil=Main.<Child>bounded(null);System.out.println("null:"+(nil==null)+":"+calls+":"+after);
  Alien alien=new Alien();try{java.lang.Object bad=Main.<Child>bounded(alien);System.out.println("bad:"+bad);}catch(ClassCastException expected){System.out.println("alien:"+calls+":"+after);}finally{System.out.println("finally:"+calls+":"+after);}
  java.lang.Object c=Main.<Alien>source(alien);System.out.println("source:"+(c==alien)+":"+calls+":"+after);
  java.lang.Object boxed=Main.<String>universal(17);System.out.println("boxed:"+boxed.equals(Integer.valueOf(17))+":"+calls+":"+after);
 }
}
