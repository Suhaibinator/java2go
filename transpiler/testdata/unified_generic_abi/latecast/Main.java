package edge;
class java{static class lang{static class Object{int code=9;}}}
class Object{}
class Child extends java.lang.Object{}
class Alien{}
interface Poly{<T>T id(T value);}
class Named implements Poly{public<T>T id(T value){Main.calls++;return value;}}
class Polluted implements Poly{public<T>T id(T value){Main.reads++;T result=(T)new Alien();Main.bodies++;return result;}}
public class Main{static int receivers,numbers,calls,reads,bodies;static Poly receiver(Poly p){receivers++;return p;}static int number(){numbers++;return 7;}
 public static void main(String[]args){Poly p=new Named();Child child=new Child();Child same=p.id(child);System.out.println((same==child)+":"+same.code);Integer boxed=receiver(p).id(number());System.out.println(boxed+":"+receivers+":"+numbers+":"+calls);
 try{Child wrong=new Polluted().id(child);System.out.println("wrong");}catch(ClassCastException expected){System.out.println("cast="+reads+":"+bodies);}finally{System.out.println("finally="+bodies);}
 try{Integer wrong=receiver(null).id(number());System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+receivers+":"+numbers+":"+calls);}
 }
}
