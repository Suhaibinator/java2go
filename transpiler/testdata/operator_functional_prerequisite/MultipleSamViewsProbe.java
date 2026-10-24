import java.util.function.*;
public class MultipleSamViewsProbe {
 static class Both implements IntUnaryOperator,IntBinaryOperator {
  public int applyAsInt(int a){return a+1;}
  public int applyAsInt(int a,int b){return a+b;}
 }
 static class Distinct implements java.util.function.Function<String,String>,Consumer<String> {
  String seen;
  public String apply(String value){return value+"!";}
  public void accept(String value){seen=value;}
 }
 public static void main(String[] args) {
  Both both=new Both();Object alias=both;
  IntUnaryOperator unary=(IntUnaryOperator)alias;
  IntBinaryOperator binary=(IntBinaryOperator)alias;
  System.out.println((alias==unary)+":"+(alias==binary)+":"+unary.applyAsInt(2)+":"+binary.applyAsInt(3,4));
  System.out.println(unary.andThen(x->x*2).applyAsInt(5));
  Distinct source=new Distinct();Object stored=source;
  java.util.function.Function<String,String> function=(java.util.function.Function<String,String>)stored;
  Consumer<String> consumer=(Consumer<String>)stored;
  consumer.accept("kept");
  System.out.println((stored==function)+":"+(stored==consumer)+":"+function.apply(source.seen));
 }
}
