import java.util.function.Function;
import java.util.function.IntUnaryOperator;
import java.util.function.ToLongFunction;
public class Main{
 static <T>T identity(T value){return value;}
 static <T>int count(T value){return 7;}
 static <T>Integer boxed(T value){return Integer.valueOf(8);}
 public static void main(String[]args){
  IntUnaryOperator generic=Main::identity;
  ToLongFunction<String> primitive=Main::count;
  Function<String,java.lang.Object> boxed=Main::boxed;
  System.out.println(generic.applyAsInt(9)+":"+primitive.applyAsLong("x")+":"+boxed.apply("x").equals(Integer.valueOf(8)));
 }
}
