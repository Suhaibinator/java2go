public final class VolatileLocalFamilyFixture {
 public static long run() throws Exception {
  class Local {volatile long stamp=7L;long read(){return stamp;}}
  Local local=new Local();
  java.lang.reflect.Field field=Local.class.getDeclaredField("stamp");field.setAccessible(true);
  field.setLong(local,9L);
  return local.read()==9L && field.getLong(local)==9L ? 9L : 0L;
 }
}
