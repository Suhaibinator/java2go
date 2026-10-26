package metadatawitness;
import java.lang.reflect.*;
public class Witness<T> {
 Class<? super T> low;
 Class<T>[] generic;
 Class[] raw;
 public static void main(String[] args)throws Exception {
  TypeVariable<?> variable=Witness.class.getTypeParameters()[0];
  ParameterizedType low=(ParameterizedType)Witness.class.getDeclaredField("low").getGenericType();
  WildcardType wildcard=(WildcardType)low.getActualTypeArguments()[0];
  System.out.println("wild="+(low.getRawType()==Class.class)+":"+(wildcard.getUpperBounds()[0]==Object.class)+":"+wildcard.getLowerBounds()[0].equals(variable));
  GenericArrayType array=(GenericArrayType)Witness.class.getDeclaredField("generic").getGenericType();
  ParameterizedType component=(ParameterizedType)array.getGenericComponentType();
  System.out.println("generic="+(component.getRawType()==Class.class)+":"+component.getActualTypeArguments()[0].equals(variable)+":"+(Witness.class.getDeclaredField("generic").getType()==Class[].class));
  System.out.println("raw="+(Witness.class.getDeclaredField("raw").getGenericType()==Class[].class));
 }
}
