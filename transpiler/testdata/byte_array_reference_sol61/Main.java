package probe;
import java.util.Arrays;
public class Main {
  public static void main(String[] args) {
    int seed=Integer.parseInt(args[0]);
    String absent=Arrays.toString((byte[])null), absentAgain=Arrays.toString((byte[])null);
    String empty=Arrays.toString(new byte[0]), emptyAgain=Arrays.toString(new byte[0]);
    byte[] values={(byte)-128,(byte)-1,0,1,127,(byte)seed};
    String first=Arrays.toString(values), second=Arrays.toString(values);
    System.out.println("null|"+absent+"|"+(absent=="null")+"|"+(absent==absentAgain));
    System.out.println("empty|"+empty+"|"+(empty=="[]")+"|"+(empty==emptyAgain));
    System.out.println("values|"+first+"|"+second+"|"+(first==second));
    values[1]=2;
    System.out.println("mutation|"+first+"|"+Arrays.toString(values));
  }
}
