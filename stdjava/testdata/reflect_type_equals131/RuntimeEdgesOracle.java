package eqruntimeedges;
import java.lang.reflect.*;
import java.util.*;
import sun.reflect.generics.reflectiveObjects.*;
public class RuntimeEdgesOracle {
 static String events="";
 static class Effect implements Type {public boolean equals(Object other){events+="E";if(other==null)throw new IllegalStateException("null");return false;}public int hashCode(){return 23;}}
 static class Array implements GenericArrayType {public Type getGenericComponentType(){events+="G";return null;}}
 static class Outer<T>{class Inner<U>{}}
 public static void main(String[] args){
  GenericArrayType array=GenericArrayTypeImpl.make(new Effect());
  try{array.equals(new Array());System.out.println("wrong");}catch(IllegalStateException expected){System.out.println("null-throw="+events+":"+expected.getMessage());}
  ParameterizedType owner=ParameterizedTypeImpl.make(Outer.class,new Type[]{String.class},null);
  ParameterizedType owner2=ParameterizedTypeImpl.make(Outer.class,new Type[]{String.class},null);
  ParameterizedType otherOwner=ParameterizedTypeImpl.make(Outer.class,new Type[]{Integer.class},null);
  ParameterizedType left=ParameterizedTypeImpl.make(Outer.Inner.class,new Type[]{Integer.class},owner);
  ParameterizedType equal=ParameterizedTypeImpl.make(Outer.Inner.class,new Type[]{Integer.class},owner2);
  ParameterizedType unequal=ParameterizedTypeImpl.make(Outer.Inner.class,new Type[]{Integer.class},otherOwner);
  System.out.println("owner="+left.equals(equal)+":"+!left.equals(unequal)+":"+(left.hashCode()==equal.hashCode()));
 }
}
