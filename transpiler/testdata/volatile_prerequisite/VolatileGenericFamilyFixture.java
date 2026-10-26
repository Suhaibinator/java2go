public final class VolatileGenericFamilyFixture {
 static final class Box<T> {volatile T value;}
 public static long run() throws Exception {
  Box<Object> box=new Box<Object>();Object marker=new Object();
  java.lang.reflect.Field field=Box.class.getDeclaredField("value");field.setAccessible(true);
  field.set(box,marker);
  return box.value==marker && field.get(box)==marker ? 9L : 0L;
 }
}
