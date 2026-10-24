import java.util.function.*;
public class OperatorAncestryProbe {
 public static void main(String[] args) {
  UnaryOperator<String> unary=s->s+"!";
  java.util.function.Function<String,String> function=unary;
  Object u=unary;
  java.util.function.Function<String,String> restored=(java.util.function.Function<String,String>)u;
  BinaryOperator<String> binary=(a,b)->a+b;
  BiFunction<String,String,String> bifunction=binary;
  Object b=binary;
  BiFunction<String,String,String> restoredBinary=(BiFunction<String,String,String>)b;
  System.out.println((u==function)+":"+(u==restored)+":"+restored.apply("a"));
  System.out.println((b==bifunction)+":"+(b==restoredBinary)+":"+restoredBinary.apply("b","c"));
  System.out.println(function.andThen(String::length).apply("d"));
  System.out.println(bifunction.andThen(String::length).apply("e","f"));
 }
}
