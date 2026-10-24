public final class VolatileAnonymousFamilyFixture {
 interface Carrier {long read();}
 public static long run() throws Exception {
  Carrier value=new Carrier(){volatile long stamp=7L;public long read(){return stamp;}};
  java.lang.reflect.Field field=value.getClass().getDeclaredField("stamp");field.setAccessible(true);
  field.setLong(value,9L);
  return value.read()==9L && field.getLong(value)==9L ? 9L : 0L;
 }
}
