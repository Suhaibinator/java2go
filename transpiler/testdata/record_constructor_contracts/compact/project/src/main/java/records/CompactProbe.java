package records;
public class CompactProbe {
 public record Value(int number) {
  public Value { if(number<0) throw new IllegalArgumentException("negative"); number=number+1; }
 }
 public static void main(String[] args){System.out.println(new Value(4).number());}
}
