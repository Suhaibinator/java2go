import java.util.function.*;
public class DefaultParentInferenceProbe {
 static class Named implements Function<String,String> {
  public String apply(String value){return value+"#";}
 }
 public static void main(String[] args){
  Function<String,String> function=value->value+"?";
  UnaryOperator<String> unary=value->value+"!";
  System.out.println(function.andThen(unary).apply("a"));
  System.out.println(function.compose(unary).apply("b"));
  Named named=new Named();
  System.out.println(function.andThen(named).apply("c"));
  System.out.println(function.compose(named).apply("d"));
 }
}
