package probe;
import java.security.MessageDigest;
import java.nio.ByteBuffer;
import java.util.Arrays;
public class Main {
  static String units(String value) {
    if (value == null) return "null";
    StringBuilder result = new StringBuilder();
    for (int i=0;i<value.length();i++) {
      if (i != 0) result.append(',');
      result.append(Integer.toHexString(value.charAt(i)));
    }
    return result.toString();
  }
  static String failure(Throwable value) {
    return value.getClass().getName()+"|"+units(value.getMessage());
  }
  static void names() throws Exception {
    String[] names = {"MD5","SHA","SHA1","sHa-1","SHA224","sHa-256","SHA384","SHA512","SHA-512/224","SHA-512/256","SHA3-224","SHA3-256","SHA3-384","SHA3-512"};
    for (String literal : names) {
      String name = new String(literal.toCharArray());
      MessageDigest digest = MessageDigest.getInstance(name);
      System.out.println("name|"+units(name)+"|"+(digest.getAlgorithm()==name)+"|"+(digest.getAlgorithm()==digest.getAlgorithm())+"|"+digest.getDigestLength());
    }
    String[] missing = {"", "NoSuchDigest", "SHA-256 ", new String(new char[]{'X',0,0xd800,0xdc00,0xdfff})};
    for (String name : missing) {
      try { MessageDigest.getInstance(name); System.out.println("missing|accepted"); }
      catch (Exception e) {System.out.println("missing|"+failure(e));}
    }
    try { MessageDigest.getInstance((String)null); System.out.println("null-name|accepted"); }
    catch (Exception e) {System.out.println("null-name|"+failure(e));}
  }
  static void state(int seed) throws Exception {
    byte[] input = {(byte)seed,98,99,100,101};
    MessageDigest digest = MessageDigest.getInstance("SHA-256");
    digest.update(input[0]); digest.update(input,1,1);
    ByteBuffer buffer = ByteBuffer.wrap(input,2,1);
    digest.update(buffer);
    byte[] first = digest.digest();
    byte[] empty = digest.digest();
    digest.update(input); digest.reset();
    byte[] again = digest.digest(new byte[]{(byte)seed,98,99});
    System.out.println("state|"+Arrays.toString(first)+"|"+Arrays.toString(empty)+"|"+Arrays.equals(first,again)+"|"+buffer.position()+"|"+buffer.limit()+"|"+Arrays.toString(input));
    byte[] equal1=digest.digest(input), equal2=digest.digest(input);
    System.out.println("allocation|"+(equal1!=equal2)+"|"+Arrays.equals(equal1,equal2));
    int[][] ranges={{-1,1},{-1,0},{3,0},{4,0},{0,-1},{2,2},{Integer.MAX_VALUE,1}};
    for (int[] range:ranges) {
      digest.update((byte)97); String outcome="ok";
      try {digest.update(new byte[]{98,99,100},range[0],range[1]);}
      catch(RuntimeException e){outcome=failure(e);}
      System.out.println("range|"+range[0]+"|"+range[1]+"|"+outcome+"|"+Arrays.toString(digest.digest()));
    }
    digest.update((byte)97);
    try {digest.update((byte[])null);System.out.println("null-array|accepted");}
    catch(RuntimeException e){System.out.println("null-array|"+failure(e));}
    System.out.println("after-null-array|"+Arrays.toString(digest.digest()));
    digest.update((byte)97);
    try {digest.update((byte[])null,0,0);System.out.println("null-range|accepted");}
    catch(RuntimeException e){System.out.println("null-range|"+failure(e));}
    System.out.println("after-null-range|"+Arrays.toString(digest.digest()));
    digest.update((byte)97);
    try {digest.digest((byte[])null);System.out.println("null-digest|accepted");}
    catch(RuntimeException e){System.out.println("null-digest|"+failure(e));}
    System.out.println("after-null-digest|"+Arrays.toString(digest.digest()));
    digest.update((byte)97);
    try {digest.update((ByteBuffer)null);System.out.println("null-buffer|accepted");}
    catch(RuntimeException e){System.out.println("null-buffer|"+failure(e));}
    System.out.println("after-null-buffer|"+Arrays.toString(digest.digest()));
  }
  public static void main(String[] args) throws Exception {
    int seed=Integer.parseInt(args[0]); names();state(seed);
  }
}
