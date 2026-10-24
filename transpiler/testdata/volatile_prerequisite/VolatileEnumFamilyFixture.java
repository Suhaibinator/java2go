public final class VolatileEnumFamilyFixture {
 enum Choice { ONE; private volatile long stamp=7L; long read(){return stamp;} }
 public static long run() throws Exception {
  Choice value=Choice.ONE;
  java.lang.reflect.Field field=Choice.class.getDeclaredField("stamp");field.setAccessible(true);
  field.setLong(value,9L);
  return value.read()==9L && field.getLong(value)==9L ? 9L : 0L;
 }
}
