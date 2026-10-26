package metadatadefaults;
import java.lang.reflect.*;
public class DefaultWildcards<T> {
 Class<?> any;
 Class<? super T> lower;
 Class<? extends java.lang.Object> explicit;
 static WildcardType wildcard(String name)throws Exception{return (WildcardType)((ParameterizedType)DefaultWildcards.class.getDeclaredField(name).getGenericType()).getActualTypeArguments()[0];}
 public static void main(String[] args)throws Exception {
  WildcardType any=wildcard("any"),lower=wildcard("lower"),explicit=wildcard("explicit");TypeVariable<?> variable=DefaultWildcards.class.getTypeParameters()[0];
  System.out.println("defaults="+(any.getUpperBounds()[0]==Object.class)+":"+any.getLowerBounds().length+":"+(lower.getUpperBounds()[0]==Object.class)+":"+lower.getLowerBounds()[0].equals(variable));
  System.out.println("explicit="+(explicit.getUpperBounds()[0]==java.lang.Object.class)+":"+!explicit.getUpperBounds()[0].equals(Object.class));
 }
}
