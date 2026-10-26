package flags;
import java.lang.reflect.Modifier;
import java.util.function.IntPredicate;
import static java.lang.reflect.Modifier.isStatic;
public final class ModifierFlagsProbe {
 public static int staticValue=3;
 public int value=4;
 static int reads;
 static int next(){reads++;return 8;}
 static void show(int f){
  System.out.println("mask="+f+":"+Modifier.isPublic(f)+":"+Modifier.isPrivate(f)+":"+Modifier.isProtected(f)+":"+Modifier.isStatic(f)+":"+Modifier.isFinal(f)+":"+Modifier.isSynchronized(f)+":"+Modifier.isVolatile(f)+":"+Modifier.isTransient(f)+":"+Modifier.isNative(f)+":"+Modifier.isInterface(f)+":"+Modifier.isAbstract(f)+":"+Modifier.isStrict(f));
 }
 public static void main(String[] args)throws Exception {
  int[] flags={0,1,2,4,8,16,32,64,128,256,512,1024,2048,4096,-1,Integer.MIN_VALUE,0x87654321};
  for(int f:flags)show(f);
  System.out.println("fields="+Modifier.isStatic(ModifierFlagsProbe.class.getDeclaredField("staticValue").getModifiers())+":"+Modifier.isStatic(ModifierFlagsProbe.class.getDeclaredField("value").getModifiers()));
  System.out.println("widen="+Modifier.isStatic((byte)8)+":"+Modifier.isStatic((short)8)+":"+Modifier.isStatic((char)8));
  Integer boxed=8;
  System.out.println("boxed="+Modifier.isStatic(boxed));
  try{Modifier.isStatic((Integer)null);System.out.println("null=false");}catch(NullPointerException expected){System.out.println("null=true");}
  IntPredicate reference=Modifier::isStatic;
  System.out.println("reference="+reference.test(next())+":"+reads+":"+isStatic(8));
  System.out.println("constants="+Modifier.PUBLIC+":"+Modifier.PRIVATE+":"+Modifier.PROTECTED+":"+Modifier.STATIC+":"+Modifier.FINAL+":"+Modifier.SYNCHRONIZED+":"+Modifier.VOLATILE+":"+Modifier.TRANSIENT+":"+Modifier.NATIVE+":"+Modifier.INTERFACE+":"+Modifier.ABSTRACT+":"+Modifier.STRICT);
 }
}
