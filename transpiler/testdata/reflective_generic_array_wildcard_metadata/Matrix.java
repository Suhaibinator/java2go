package metadatamatrix;
import java.lang.reflect.*;
import foreign.Class;
public class Matrix<T> {
 java.lang.Class<?> any;
 java.lang.Class<? extends Number> number;
 java.lang.Class<? super T> lower;
 java.lang.Class<? extends T[]> boundArray;
 java.lang.Class<T>[][] generic;
 T[][] variable;
 java.lang.Class[] rawClass;
 String[][] rawString;
 int[][] rawPrimitive;
 Class[] rawSource;
 Class<T>[] genericSource;
 java.util.List<?>[] wildcardArray;
 java.lang.Class<InitTarget>[] target;
 static int initialized;
 static class InitTarget {static {initialized++;}}
 static class Binder<Class> {java.lang.Class<? super Class[]> lower;Class[][] array;}
 static class Outer<T> {class Inner<U>{java.lang.Class<? super T[]> outer;java.lang.Class<? extends U[][]> own;}}
 static Type field(java.lang.Class<?> owner,String name)throws Exception{return owner.getDeclaredField(name).getGenericType();}
 static WildcardType wildcard(String name)throws Exception{return (WildcardType)((ParameterizedType)field(Matrix.class,name)).getActualTypeArguments()[0];}
 public static void main(String[] args)throws Exception {
  TypeVariable<?> variable=Matrix.class.getTypeParameters()[0];WildcardType any=wildcard("any"),number=wildcard("number"),lower=wildcard("lower"),bound=wildcard("boundArray");
  System.out.println("wild="+(any.getUpperBounds()[0]==Object.class)+":"+any.getLowerBounds().length+":"+(number.getUpperBounds()[0]==Number.class)+":"+lower.getLowerBounds()[0].equals(variable)+":"+((GenericArrayType)bound.getUpperBounds()[0]).getGenericComponentType().equals(variable));
  GenericArrayType generic=(GenericArrayType)field(Matrix.class,"generic");ParameterizedType component=(ParameterizedType)((GenericArrayType)generic.getGenericComponentType()).getGenericComponentType();GenericArrayType vars=(GenericArrayType)field(Matrix.class,"variable");ParameterizedType list=(ParameterizedType)((GenericArrayType)field(Matrix.class,"wildcardArray")).getGenericComponentType();
  System.out.println("arrays="+(component.getRawType()==java.lang.Class.class)+":"+component.getActualTypeArguments()[0].equals(variable)+":"+((GenericArrayType)vars.getGenericComponentType()).getGenericComponentType().equals(variable)+":"+(field(Matrix.class,"rawClass")==java.lang.Class[].class)+":"+(field(Matrix.class,"rawString")==String[][].class)+":"+(field(Matrix.class,"rawPrimitive")==int[][].class)+":"+(list.getRawType()==java.util.List.class));
  ParameterizedType source=(ParameterizedType)((GenericArrayType)field(Matrix.class,"genericSource")).getGenericComponentType();TypeVariable<?> binder=Binder.class.getTypeParameters()[0];WildcardType binderLower=(WildcardType)((ParameterizedType)field(Binder.class,"lower")).getActualTypeArguments()[0];
  System.out.println("source="+(source.getRawType()==Class.class)+":"+source.getActualTypeArguments()[0].equals(variable)+":"+(field(Matrix.class,"rawSource")==Class[].class)+":"+((GenericArrayType)binderLower.getLowerBounds()[0]).getGenericComponentType().equals(binder)+":"+((GenericArrayType)((GenericArrayType)field(Binder.class,"array")).getGenericComponentType()).getGenericComponentType().equals(binder)+":"+(binder.getGenericDeclaration()==Binder.class));
  WildcardType outer=(WildcardType)((ParameterizedType)field(Outer.Inner.class,"outer")).getActualTypeArguments()[0];WildcardType own=(WildcardType)((ParameterizedType)field(Outer.Inner.class,"own")).getActualTypeArguments()[0];TypeVariable<?> captured=(TypeVariable<?>)((GenericArrayType)outer.getLowerBounds()[0]).getGenericComponentType();TypeVariable<?> declared=(TypeVariable<?>)((GenericArrayType)((GenericArrayType)own.getUpperBounds()[0]).getGenericComponentType()).getGenericComponentType();
  System.out.println("owners="+(captured.getGenericDeclaration()==Outer.class)+":"+(declared.getGenericDeclaration()==Outer.Inner.class)+":"+!captured.equals(declared));
  Type[] argsCopy=component.getActualTypeArguments();argsCopy[0]=null;Type[] upperCopy=lower.getUpperBounds();upperCopy[0]=null;Type[] lowerCopy=lower.getLowerBounds();lowerCopy[0]=null;
  System.out.println("copies="+component.getActualTypeArguments()[0].equals(variable)+":"+(lower.getUpperBounds()[0]==Object.class)+":"+lower.getLowerBounds()[0].equals(variable)+":"+component.equals(((GenericArrayType)((GenericArrayType)field(Matrix.class,"generic")).getGenericComponentType()).getGenericComponentType())+":"+(component.hashCode()==((GenericArrayType)((GenericArrayType)field(Matrix.class,"generic")).getGenericComponentType()).getGenericComponentType().hashCode()));
  ParameterizedType target=(ParameterizedType)((GenericArrayType)field(Matrix.class,"target")).getGenericComponentType();System.out.println("initialization="+initialized+":"+(target.getActualTypeArguments()[0]==InitTarget.class)+":"+initialized);
 }
}
