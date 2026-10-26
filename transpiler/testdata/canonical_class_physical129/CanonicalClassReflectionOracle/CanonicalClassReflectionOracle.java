package classproof;
import java.lang.reflect.*;
public class CanonicalClassReflectionOracle {
 static class Slots<T>{Class<? super T> low;Class<? extends T> high;Class<T> exact;Class<T>[] array;Class[] rawArray;}
 static class Outer<T>{class Capture<U>{Class<T> outer;Class<U> own;}}
 static ParameterizedType field(Class<?> owner,String name)throws Exception{return (ParameterizedType)owner.getDeclaredField(name).getGenericType();}
 public static void main(String[] args)throws Exception{
  ParameterizedType low=field(Slots.class,"low");WildcardType lower=(WildcardType)low.getActualTypeArguments()[0];TypeVariable<?> variable=(TypeVariable<?>)lower.getLowerBounds()[0];
  System.out.println("low="+(low.getRawType()==Class.class)+":"+(lower.getUpperBounds()[0]==Object.class)+":"+(variable.getGenericDeclaration()==Slots.class)+":"+variable.equals(Slots.class.getTypeParameters()[0]));
  WildcardType high=(WildcardType)field(Slots.class,"high").getActualTypeArguments()[0];
  System.out.println("high="+high.getLowerBounds().length+":"+high.getUpperBounds()[0].equals(variable));
  Type exact=field(Slots.class,"exact").getActualTypeArguments()[0];System.out.println("exact="+exact.equals(variable));
  GenericArrayType array=(GenericArrayType)Slots.class.getDeclaredField("array").getGenericType();ParameterizedType component=(ParameterizedType)array.getGenericComponentType();
  System.out.println("array="+(component.getRawType()==Class.class)+":"+component.getActualTypeArguments()[0].equals(variable)+":"+(Slots.class.getDeclaredField("array").getType()==Class[].class));
  System.out.println("raw-array="+(Slots.class.getDeclaredField("rawArray").getGenericType()==Class[].class));
  TypeVariable<?> outer=(TypeVariable<?>)field(Outer.Capture.class,"outer").getActualTypeArguments()[0];TypeVariable<?> own=(TypeVariable<?>)field(Outer.Capture.class,"own").getActualTypeArguments()[0];
  System.out.println("capture="+(outer.getGenericDeclaration()==Outer.class)+":"+(own.getGenericDeclaration()==Outer.Capture.class)+":"+!outer.equals(own));
 }
}
