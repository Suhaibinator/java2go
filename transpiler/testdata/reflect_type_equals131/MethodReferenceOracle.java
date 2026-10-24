package eqmethodref;
public class MethodReferenceOracle {
 static class Object{}
 static class Custom implements java.lang.reflect.Type {public boolean equals(java.lang.Object other){return other instanceof Integer;}public int hashCode(){return 31;}}
 public static void main(String[] args){java.lang.reflect.Type value=new Custom();java.util.function.Predicate<Integer> bound=value::equals;java.util.function.BiPredicate<java.lang.reflect.Type,Integer> unbound=java.lang.reflect.Type::equals;System.out.println("refs="+bound.test(9)+":"+unbound.test(value,10));java.lang.reflect.Type missing=null;try{java.util.function.Predicate<Integer> bad=missing::equals;System.out.println("wrong");}catch(NullPointerException expected){System.out.println("bound-null");}try{unbound.test(null,11);System.out.println("wrong");}catch(NullPointerException expected){System.out.println("unbound-null");}}
}
