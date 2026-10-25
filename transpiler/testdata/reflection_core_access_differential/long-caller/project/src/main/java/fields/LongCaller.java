package fields;
import java.lang.reflect.Field;
public class LongCaller {
 private long value=7L;
 private static String read(Field f,Object value){try{return ""+f.getLong(value);}catch(IllegalAccessException expected){return "denied";}}
 private static String write(Field f,Object value,long next){try{f.setLong(value,next);return "ok";}catch(IllegalAccessException expected){return "denied";}}
 public static void main(String[] args)throws Exception{
  LongCaller x=new LongCaller();Field f=LongCaller.class.getDeclaredField("value");
  System.out.println("own="+f.canAccess(x)+":"+read(f,x)+":"+write(f,x,9L)+":"+read(f,x)+":"+x.value);
  PackageValue y=new PackageValue();Field g=PackageValue.class.getDeclaredField("value");
  System.out.println("package="+g.canAccess(y)+":"+read(g,y)+":"+write(g,y,13L)+":"+read(g,y)+":"+y.value);
 }
}
