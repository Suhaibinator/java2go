package enumfrontier;
public class TextControl {
 static String saved=new String("saved");static String superSeen;static String trace="";
 static ThreadLocal<String> token=new ThreadLocal<>();
 enum Default { A;String implicit(){return toString();}String explicit(){return this.toString();} }
 enum Varargs { A;String toString(String... ignored){return "varargs";} }
 enum Style {
  A {public String toString(){superSeen=super.toString();trace+=Thread.holdsLock(this)+":"+token.get();return saved;}},
  B {public String toString(){return null;}}, C;
  public String toString(){return "enum:"+name();}
  String parent(){return super.toString();}
  String toString(int ignored){return "overload";}
 }
 enum ConstantOnly {A {public String toString(){return "constant:"+super.toString();}},B;}
 static int calls;static Default once(){calls++;return Default.A;}
 public static void main(String[] args){
  Default d=Default.A;java.lang.Enum<Default> e=d;java.lang.Enum<?> q=d;java.lang.Object o=d;
  System.out.println("default="+(d.toString()==d.name())+":"+(e.toString()==d.name())+":"+(q.toString()==d.name())+":"+(o.toString()==d.name()));
  System.out.println("implicit="+d.implicit()+":"+d.explicit()+":"+Varargs.A.toString()+":"+Varargs.A.toString("x"));
  token.set("caller");synchronized(Style.A){System.out.println("override="+(Style.A.toString()==saved)+":"+(String.valueOf(Style.A)==saved));}
  System.out.println("callback="+trace+":"+superSeen+":"+Style.A.parent()+":"+Style.C.toString()+":"+Style.A.toString(1));
  java.lang.Enum<?> b=Style.B;java.lang.Object a=Style.A;
  System.out.println("aliases="+(b.toString()==null)+":"+(String.valueOf(b)==null)+":"+(a.toString()==saved));
  System.out.println("constant="+ConstantOnly.A.toString()+":"+ConstantOnly.B.toString());
  System.out.println("once="+once().toString()+":"+calls);
  Default absent=null;try{absent.toString();System.out.println("null=unexpected");}catch(NullPointerException expected){System.out.println("null=NullPointerException:"+String.valueOf(absent));}
 }
}
