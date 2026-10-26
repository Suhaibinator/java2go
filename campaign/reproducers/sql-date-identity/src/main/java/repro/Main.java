package repro;
import java.sql.Date;
public class Main {
 public static void main(String[] args) {
  Date imported = new Date(0L);
  Object value = imported;
  Object qualified = new java.sql.Date(0L);
  System.out.print(Date.class.getName() + ":" + value.getClass().getName() + ":" +
      (value instanceof java.util.Date) + ":" + qualified.getClass().getName());
 }
}
