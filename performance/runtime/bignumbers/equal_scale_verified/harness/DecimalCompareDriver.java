public final class DecimalCompareDriver {
 public static void main(String[] args){
  System.out.println(DecimalCompareProbe.audit());
  DecimalCompareProbe.setup(Integer.parseInt(args[0]),Integer.parseInt(args[1]),Integer.parseInt(args[2]));
  System.out.println(DecimalCompareProbe.run(Integer.parseInt(args[3])));
 }
}
