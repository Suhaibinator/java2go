package records;
public class ExplicitCanonicalProbe {
 public record Value(int number) { public Value(int number){this.number=number+2;} }
 public static void main(String[] args){System.out.println(new Value(4).number());}
}
