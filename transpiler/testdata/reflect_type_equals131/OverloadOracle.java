package eqoverload;
public class OverloadOracle {
 static class Child{}
 interface More {String equals(Child child);}
 static class Both implements java.lang.reflect.Type,More {
  public String equals(Child child){return "child";}
  public boolean equals(java.lang.Object other){return false;}
 }
 static <T extends java.lang.reflect.Type & More> String selected(T value,Child child){return value.equals(child);}
 static <T extends java.lang.reflect.Type & More> boolean object(T value,java.lang.Object other){return value.equals(other);}
 static class Type {public String equals(Child child){return "source";}}
 static <Type extends OverloadOracle.Type> String shadow(Type value,Child child){return value.equals(child);}
 public static void main(String[] args){Both value=new Both();Child child=new Child();System.out.println("selected="+selected(value,child));System.out.println("object="+object(value,child));System.out.println("shadow="+shadow(new Type(),child));}
}
