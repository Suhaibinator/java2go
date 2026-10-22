import java.lang.reflect.Array;
import java.util.Arrays;
public class ArrayBounds189 {
 static String trace="";
 static final RuntimeException marker=new IllegalStateException("marker");
 static void run(String name,Runnable action){trace="";try{action.run();System.out.println(name+":"+trace+":OK");}catch(Throwable e){System.out.println(name+":"+trace+":"+(e==marker?"MARKER":e.getClass().getName()+(e instanceof ArrayIndexOutOfBoundsException?":"+e.getMessage():"")));}}
 static int[] p(boolean nil){trace+="A";return nil?null:new int[2];}
 static Object[] r(boolean nil){trace+="A";return nil?null:new String[2];}
 static int i(int value,boolean abrupt){trace+="I";if(abrupt)throw marker;return value;}
 static int v(boolean abrupt){trace+="V";if(abrupt)throw marker;return 7;}
 static Object o(boolean abrupt){trace+="V";if(abrupt)throw marker;return Integer.valueOf(7);}
 static void consume(Object value){}
 public static void main(String[] args){
  run("ref-negative",()->consume((new Object[2])[-1]));
  run("ref-high",()->consume((new Object[2])[2]));
  run("ref-empty",()->consume((new Object[0])[0]));
  run("ref-min",()->consume((new Object[2])[Integer.MIN_VALUE]));
  run("ref-max",()->consume((new Object[2])[Integer.MAX_VALUE]));
  run("boolean-negative",()->consume((new boolean[2])[-1]));run("boolean-high",()->consume((new boolean[2])[2]));
  run("byte-negative",()->consume((new byte[2])[-1]));run("byte-high",()->consume((new byte[2])[2]));
  run("short-negative",()->consume((new short[2])[-1]));run("short-high",()->consume((new short[2])[2]));
  run("char-negative",()->consume((new char[2])[-1]));run("char-high",()->consume((new char[2])[2]));
  run("int-negative",()->consume((new int[2])[-1]));run("int-high",()->consume((new int[2])[2]));
  run("long-negative",()->consume((new long[2])[-1]));run("long-high",()->consume((new long[2])[2]));
  run("float-negative",()->consume((new float[2])[-1]));run("float-high",()->consume((new float[2])[2]));
  run("double-negative",()->consume((new double[2])[-1]));run("double-high",()->consume((new double[2])[2]));
  run("primitive-empty",()->consume((new int[0])[-1]));
  run("byte-index",()->consume((new int[2])[(byte)-128]));run("short-index",()->consume((new int[2])[(short)32767]));run("char-index",()->consume((new int[2])[(char)65535]));
  run("rank-outer",()->consume((new int[2][1])[2][0]));
  run("rank-inner",()->consume((new int[2][1])[0][1]));
  run("primitive-store-negative",()->{p(false)[i(-1,false)]=v(false);});
  run("primitive-store-high",()->{p(false)[i(2,false)]=v(false);});
  run("reference-store-negative",()->{r(false)[i(-1,false)]=o(false);});
  run("reference-store-high",()->{r(false)[i(2,false)]=o(false);});
  run("primitive-null-store",()->{p(true)[i(-1,false)]=v(false);});
  run("reference-null-store",()->{r(true)[i(-1,false)]=o(false);});
  run("abrupt-index",()->{p(false)[i(-1,true)]=v(false);});
  run("abrupt-rhs",()->{p(false)[i(-1,false)]=v(true);});
  run("reference-abrupt-rhs",()->{r(true)[i(-1,false)]=o(true);});
  run("reference-component-check",()->{r(false)[i(0,false)]=o(false);});
  run("compound-bounds-before-rhs",()->{p(false)[i(-1,false)]+=v(false);});
  run("as-list-get-negative",()->consume(Arrays.asList(new String[2]).get(-1)));
  run("as-list-set-high",()->consume(Arrays.asList(new String[2]).set(2,"value")));
  run("reflect-get-negative",()->consume(Array.get(new Object[2],-1)));
  run("reflect-get-int-high",()->consume(Array.getInt(new int[2],2)));
  run("reflect-set-high",()->Array.set(new Object[2],2,"value"));
  run("copy-of-range-negative",()->consume(Arrays.copyOfRange(new byte[2],-1,1)));
  System.out.println("explicit-int-constructor:"+new ArrayIndexOutOfBoundsException(-1).getMessage());
  System.out.println("explicit-string-constructor:"+new ArrayIndexOutOfBoundsException("caller message").getMessage());
 }
}
