package records;
public class SameArityProbe {
 public record Value(int number) { public Value(String text){this(text.length());} }
 public static void main(String[] args){System.out.println(new Value(4).number()+":"+new Value("abc").number());}
}
