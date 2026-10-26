package eqroute;
import java.lang.reflect.*;
public class RoutingOracle {
 static class Box<T>{java.util.List<String> left;java.util.List<String> right;java.util.List<Integer> other;T item;}
 static class Peer<T>{}
 static String events="";
 static final java.lang.Object lock=new java.lang.Object();
 static class Custom implements Type {
  public boolean equals(java.lang.Object other){synchronized(lock){events+="E";return other==null;}}
  public boolean equals(Custom other){events+="O";return false;}
  public int hashCode(){synchronized(lock){events+="H";return 17;}}
 }
 static class Child extends Custom {public boolean equals(java.lang.Object other){synchronized(lock){events+="C";return other==null;}}}
 static Type receiver(Type value){events+="R";return value;}
 static java.lang.Object argument(){events+="A";return null;}
 static java.lang.Object throwing(){events+="X";throw new IllegalStateException("argument");}
 public static void main(String[] args)throws Exception{
  Type left=Box.class.getDeclaredField("left").getGenericType();Type right=Box.class.getDeclaredField("right").getGenericType();Type other=Box.class.getDeclaredField("other").getGenericType();
  TypeVariable<?> variable=Box.class.getTypeParameters()[0];Type item=Box.class.getDeclaredField("item").getGenericType();TypeVariable<?> peer=Peer.class.getTypeParameters()[0];
  System.out.println("metadata="+left.equals(right)+":"+!left.equals(other)+":"+!left.equals(null)+":"+(left.hashCode()==right.hashCode()));
  System.out.println("variables="+variable.equals(item)+":"+!variable.equals(peer)+":"+(variable.hashCode()==item.hashCode())+":"+(variable.getGenericDeclaration()==Box.class));
  Type classType=String.class;System.out.println("class="+classType.equals(String.class)+":"+!classType.equals(Integer.class)+":"+!classType.equals(null));
  Type custom=new Child();events="";synchronized(lock){boolean a=custom.equals(null);boolean self=custom.equals(custom);int hash=custom.hashCode();System.out.println("source="+a+":"+self+":"+hash+":"+events);}
  events="";try{receiver(null).equals(argument());System.out.println("wrong");}catch(NullPointerException expected){System.out.println("null="+events);}
  events="";try{receiver(null).equals(throwing());System.out.println("wrong");}catch(IllegalStateException expected){System.out.println("throw="+events+":"+expected.getMessage());}
  events="";System.out.println("once="+receiver(custom).equals(argument())+":"+events);
 }
}
