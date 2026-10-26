package support;
public class Effects {
 public static int count;
 public static String text(String value) {count++;return value;}
 public static byte[] bytes(byte[] value) {count++;return value;}
 public static java.util.UUID value(java.util.UUID value) {count++;return value;}
 public static java.util.UUID abrupt() {count++;throw new IllegalArgumentException("argument");}
}
